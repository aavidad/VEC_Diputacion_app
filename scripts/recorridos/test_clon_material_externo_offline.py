"""Producer boundary tests with invented scratch fixtures, never live material.

The accreditation authority is a test double only. Its own source/message
validation belongs to test_clon_fuente_acreditada; production cannot inject it.
Run with VEC_TEST_SCRATCH set to an owned private directory outside Git.
"""
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from cryptography import x509
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec

SPEC = importlib.util.spec_from_file_location("external_offline", Path(__file__).with_name("clon_material_externo_offline.py"))
offline = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(offline)
NOW = datetime(2026, 10, 1, 1, tzinfo=timezone.utc)


class OfflineExternalMaterialTests(unittest.TestCase):
    def setUp(self):
        scratch = os.environ.get("VEC_TEST_SCRATCH")
        if not scratch:
            self.fail("VEC_TEST_SCRATCH is required: private scratch outside Git")
        self.temp = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.source_dir = self.base / "source"
        self.source_dir.mkdir(mode=0o700)
        self.output = self.base / "output"
        self.source = self.source_dir / "source.json"
        self.ack = self.source_dir / "ack.json"
        prefixes = {"cuenta": "cta_", "persona": "per_", "perfil": "prf_", "contexto": "vca_",
                    "vinculo": "vin_", "candidato": "can_", "procedencia": "prc_"}
        self.refs = {k: p + hashlib.sha256(("invented-test:" + k).encode()).hexdigest()[:32] for k, p in prefixes.items()}
        self.master = {"datos_sinteticos": True,
            "personas": [{"persona_ref": self.refs["persona"], "nombre_visible": "Persona de prueba"}],
            "candidatoBolsa": {k + "_ref": self.refs[k] for k in ("cuenta", "persona", "perfil", "candidato")}}
        data = offline.encoded(self.master)
        self.source_sha = offline.digest(data)
        self.metadata = {"version": 1, "source_sha256": self.source_sha,
            "autoridad_maestra": "test-only:authority", "responsable": "test-only:owner",
            "acreditado_en": "2026-09-30T23:53:00Z", "refs": self.refs, "sujeto": offline.SUBJECT,
            "vigente_desde": "2026-10-01T00:00:00.000000Z", "vigente_hasta": "2026-10-15T00:00:00.000000Z"}
        self.write(self.source, data)
        self.write(self.ack, offline.encoded({"test-only-ack": self.source_sha}))
        self.addCleanup(patch.stopall)
        patch.object(offline, "SOURCE_SHA", self.source_sha).start()
        self.authority = patch.object(offline, "accreditation", return_value=self.metadata).start()

    @staticmethod
    def write(path, data):
        path.write_bytes(data)
        path.chmod(0o600)

    def produce(self, **kwargs):
        return offline.produce(self.source, self.ack, self.output, now=kwargs.get("now", NOW))

    def bound(self):
        return offline.binding(self.source.read_bytes(), self.ack.read_bytes(), NOW)

    def material(self):
        return {name: (self.output / name).read_bytes() for name in offline.FILES}

    def test_complete_material_matches_contract_and_replay_preserves_every_inode(self):
        receipt = self.produce()
        self.assertEqual(receipt["refs"], self.refs)
        self.assertEqual(receipt["fuente_sha256"], self.source_sha)
        self.assertFalse(receipt["exportador_ejecutado"])
        self.assertFalse(receipt["provision_ejecutada"])
        self.assertFalse(receipt["runtime_activado"])
        self.assertNotIn("nombre_visible", receipt)
        before = {str(p.relative_to(self.output)): (p.stat().st_ino, p.stat().st_mtime_ns)
                  for p in self.output.rglob("*")}
        with patch.object(offline, "generate", side_effect=AssertionError("must not generate on replay")):
            self.assertEqual(receipt, self.produce())
        self.assertEqual(before, {str(p.relative_to(self.output)): (p.stat().st_ino, p.stat().st_mtime_ns)
                                 for p in self.output.rglob("*")})
        self.assertEqual(set(receipt["archivos"]), offline.FILES)
        for path in (self.output, *self.output.rglob("*")):
            self.assertEqual(path.stat().st_mode & 0o777, 0o700 if path.is_dir() else 0o600)
        runtime = self.output / "runtime-externo"
        self.assertEqual({str(p.relative_to(runtime)) for p in runtime.rglob("*") if p.is_file()}, offline.RUNTIME_FILES)
        self.assertFalse(any(p.name == "ca.key" or p.name.startswith("candidato.key") or
                             p.suffix in (".p12", ".password") or p.name in ("kms", "tsa") for p in runtime.rglob("*")))
        bolsa = json.loads((runtime / "identidad/bolsa-candidato.json").read_bytes())
        self.assertEqual({k + "_ref": self.refs[k] for k in ("cuenta", "persona", "perfil", "candidato")},
                         {k: v for k, v in bolsa.items() if k.endswith("_ref")})
        for name, data in self.material().items():
            self.assertEqual(receipt["archivos"][name], {"sha256": offline.digest(data), "bytes": len(data)})

    def test_absent_ack_wrong_sha_and_denied_authority_have_no_effects(self):
        self.ack.unlink()
        with self.assertRaises(FileNotFoundError):
            self.produce()
        self.assertFalse(self.output.exists())
        self.write(self.ack, offline.encoded({"test-only-ack": self.source_sha}))
        self.write(self.source, self.source.read_bytes() + b" ")
        with self.assertRaisesRegex(offline.OfflineMaterialError, "source_sha_mismatch"):
            self.produce()
        self.assertFalse(self.output.exists())
        self.write(self.source, offline.encoded(self.master))
        self.authority.side_effect = offline.OfflineMaterialError("ack_mismatch")
        with self.assertRaisesRegex(offline.OfflineMaterialError, "ack_mismatch"):
            self.produce()
        self.assertFalse(self.output.exists())

    def test_missing_production_authority_is_fail_closed(self):
        # Restoring the production entrypoint checks the actual absent sibling;
        # the test double never enters a production argument or output.
        with patch.object(offline, "accreditation", wraps=ORIGINAL_AUTHORITY):
            with patch.object(offline.Path, "is_file", return_value=False):
                with self.assertRaisesRegex(offline.OfflineMaterialError, "authority_absent"):
                    self.produce()
        self.assertFalse(self.output.exists())

    def test_pending_reservation_precedes_generation_and_blocks_interrupted_replay(self):
        def interrupted(bound, now):
            self.assertEqual((self.output / offline.PENDING).read_bytes(), offline.encoded(bound))
            self.assertEqual(set(self.output.iterdir()), {self.output / offline.PENDING, self.output / offline.LOCK})
            raise offline.OfflineMaterialError("injected_interruption")
        with patch.object(offline, "generate", side_effect=interrupted):
            with self.assertRaisesRegex(offline.OfflineMaterialError, "injected_interruption"):
                self.produce()
        with patch.object(offline, "generate", side_effect=AssertionError("must not regenerate")):
            with self.assertRaisesRegex(offline.OfflineMaterialError, "pending_or_unexpected"):
                self.produce()

    def test_changed_ack_and_private_keys_cannot_be_replayed_or_repaired(self):
        self.produce()
        self.write(self.ack, offline.encoded({"test-only-ack": self.source_sha, "altered": True}))
        with self.assertRaisesRegex(offline.OfflineMaterialError, "binding_changed"):
            self.produce()
        self.write(self.ack, offline.encoded({"test-only-ack": self.source_sha}))
        key_path = self.output / "runtime-externo/idempotencia/g2-localizador.bin"
        self.write(key_path, b"x" * 32)
        with patch.object(offline, "generate", side_effect=AssertionError("must not regenerate")):
            with self.assertRaisesRegex(offline.OfflineMaterialError, "preimage_changed"):
                self.produce()
        self.assertEqual(key_path.read_bytes(), b"x" * 32)

    def test_deleted_output_or_unknown_empty_directory_is_rejected(self):
        self.produce()
        missing = self.output / "runtime-externo/identidad/candidato.json"
        missing.unlink()
        with self.assertRaisesRegex(offline.OfflineMaterialError, "pending_or_unexpected"):
            self.produce()
        self.assertFalse(missing.exists())
        self.write(missing, b"{}")
        (self.output / "runtime-externo/unknown").mkdir(mode=0o700)
        with self.assertRaisesRegex(offline.OfflineMaterialError, "directory_unexpected"):
            self.produce()

    def test_malformed_identity_fingerprint_refs_and_hmac_are_rejected(self):
        self.produce()
        cases = [
            ("identidad/bolsa-candidato.json", b"{}"),
            ("identidad/candidato.json", b"{}"),
            ("manifiesto.json", b"{}"),
            ("idempotencia/configuracion.json", b"{}"),
            ("idempotencia/g1-localizador.bin", bytes(32)),
        ]
        for name, data in cases:
            with self.subTest(name=name):
                outputs = self.material()
                outputs[offline.RUNTIME + name] = data
                with self.assertRaises(offline.OfflineMaterialError):
                    offline.verify_material(outputs, self.bound(), NOW)

    def test_wrong_key_certificate_chain_expiry_and_pkcs12_are_rejected(self):
        self.produce()
        outputs = self.material()
        wrong_key = ec.generate_private_key(ec.SECP256R1()).private_bytes(serialization.Encoding.PEM,
            serialization.PrivateFormat.PKCS8, serialization.NoEncryption())
        for name, replacement in [(offline.RUNTIME + "tls/servidor.key", wrong_key),
                                  (offline.CUSTODY + "mtls/candidato.key", wrong_key),
                                  (offline.CUSTODY + "mtls/candidato.p12", b"invalid")]:
            with self.subTest(name=name):
                modified = outputs | {name: replacement}
                with self.assertRaises((offline.OfflineMaterialError, ValueError)):
                    offline.verify_material(modified, self.bound(), NOW)
        with self.assertRaisesRegex(offline.OfflineMaterialError, "validity_invalid"):
            offline.verify_material(outputs, self.bound(), NOW + timedelta(days=15))
        other = offline.generate(self.bound(), NOW)
        with self.assertRaises((ValueError, InvalidSignature)):
            offline.verify_material(outputs | {offline.RUNTIME + "ca/ca.crt": other[offline.RUNTIME + "ca/ca.crt"]}, self.bound(), NOW)

    def test_signed_client_wrong_san_and_subject_are_rejected(self):
        self.produce()
        outputs = self.material()
        original = x509.load_pem_x509_certificate(outputs[offline.RUNTIME + "mtls/candidato.crt"])
        ca_key = serialization.load_pem_private_key(outputs[offline.CUSTODY + "ca/ca.key"], None)
        for wrong_subject in (False, True):
            builder = (x509.CertificateBuilder().subject_name(offline.subject_name("other") if wrong_subject else original.subject)
                .issuer_name(original.issuer).public_key(original.public_key()).serial_number(original.serial_number)
                .not_valid_before(original.not_valid_before_utc).not_valid_after(original.not_valid_after_utc))
            for extension in original.extensions:
                value = extension.value
                if extension.oid == x509.ExtensionOID.SUBJECT_ALTERNATIVE_NAME and not wrong_subject:
                    value = x509.SubjectAlternativeName([x509.UniformResourceIdentifier("urn:vec:other")])
                builder = builder.add_extension(value, extension.critical)
            cert = builder.sign(ca_key, hashes.SHA256()).public_bytes(serialization.Encoding.PEM)
            with self.assertRaisesRegex(offline.OfflineMaterialError, "subject_invalid|san_invalid"):
                offline.verify_material(outputs | {offline.RUNTIME + "mtls/candidato.crt": cert}, self.bound(), NOW)

    def test_source_expiry_metadata_and_refs_are_rejected_before_output(self):
        for modified in (self.metadata | {"refs": self.refs | {"persona": "per_bad"}},
                         self.metadata | {"sujeto": "desarrollo:otro"},
                         self.metadata | {"vigente_hasta": "2026-10-01T00:00:00Z"}):
            with self.subTest(modified=modified.keys()):
                self.authority.return_value = modified
                with self.assertRaises(offline.OfflineMaterialError):
                    self.produce()
                self.assertFalse(self.output.exists())

    def test_linked_input_and_runtime_files_and_unsafe_modes_are_rejected(self):
        original = self.source_dir / "preserved-source"
        self.source.rename(original)
        self.source.symlink_to(original)
        with self.assertRaises(OSError):
            self.produce()
        self.source.unlink()
        os.link(original, self.source)
        with self.assertRaisesRegex(offline.OfflineMaterialError, "file_invalid"):
            self.produce()
        self.source.unlink()
        self.source.write_bytes(original.read_bytes())
        self.source.chmod(0o644)
        with self.assertRaisesRegex(offline.OfflineMaterialError, "file_invalid"):
            self.produce()
        self.source.chmod(0o600)
        self.produce()
        target = self.output / "runtime-externo/tls/servidor.key"
        preserved = self.base / "preserved-key"
        target.rename(preserved)
        target.symlink_to(preserved)
        with self.assertRaisesRegex(offline.OfflineMaterialError, "entry_invalid"):
            self.produce()
        self.assertTrue(preserved.exists())

    def test_symlinked_root_and_git_ancestor_are_rejected(self):
        target = self.base / "other"
        target.mkdir(mode=0o700)
        self.output.symlink_to(target, target_is_directory=True)
        with self.assertRaises(OSError):
            self.produce()
        self.assertEqual(list(target.iterdir()), [])
        self.output.unlink()
        (self.base / ".git").write_text("gitdir: unrelated")
        with self.assertRaisesRegex(offline.OfflineMaterialError, "inside_git"):
            self.produce()
        self.assertFalse(self.output.exists())

    def test_ancestor_swap_after_open_never_writes_symlink_destination(self):
        ancestor = self.base / "ancestor"
        ancestor.mkdir(mode=0o700)
        self.output = ancestor / "output"
        destination = self.base / "destination"
        destination.mkdir(mode=0o700)
        opened = offline.open_root
        swapped = False
        def race(path, *, create=False):
            nonlocal swapped
            result = opened(path, create=create)
            if path == self.output and create and not swapped:
                ancestor.rename(self.base / "held")
                ancestor.symlink_to(destination, target_is_directory=True)
                swapped = True
            return result
        with patch.object(offline, "open_root", side_effect=race):
            with self.assertRaises(OSError):
                self.produce()
        self.assertEqual(list(destination.iterdir()), [])
        self.assertEqual(list((self.base / "held/output").iterdir()), [])

    def test_generation_toctou_and_file_mutation_cannot_seal_success(self):
        generate = offline.generate
        def race(bound, now):
            result = generate(bound, now)
            moved = self.base / "moved"
            self.output.rename(moved)
            self.output.mkdir(mode=0o700)
            return result
        with patch.object(offline, "generate", side_effect=race):
            with self.assertRaisesRegex(offline.OfflineMaterialError, "directory_changed"):
                self.produce()
        self.assertEqual(list(self.output.iterdir()), [])
        self.assertFalse((self.base / "moved" / offline.RECEIPT).exists())

    def test_leaf_inserted_symlink_cannot_replace_or_receive_a_key(self):
        target = self.base / "untouched"
        self.write(target, b"preserve-exactly")
        write = offline.write_at
        def race(fd, relative, data):
            if relative == offline.CUSTODY + "ca/ca.key":
                parent = self.output / "custodia/ca"
                parent.mkdir(mode=0o700, parents=True, exist_ok=True)
                (parent / "ca.key").symlink_to(target)
            return write(fd, relative, data)
        with patch.object(offline, "write_at", side_effect=race):
            with self.assertRaises(FileExistsError):
                self.produce()
        self.assertEqual(target.read_bytes(), b"preserve-exactly")
        self.assertFalse((self.output / offline.RECEIPT).exists())

    def test_input_modified_during_descriptor_read_is_rejected(self):
        read = os.read
        modified = False
        def race(fd, size):
            nonlocal modified
            data = read(fd, size)
            if not modified and data:
                self.write(self.source, self.source.read_bytes() + b" ")
                modified = True
            return data
        with patch.object(offline.os, "read", side_effect=race):
            with self.assertRaisesRegex(offline.OfflineMaterialError, "file_changed"):
                self.produce()
        self.assertFalse(self.output.exists())


ORIGINAL_AUTHORITY = offline.accreditation

if __name__ == "__main__":
    unittest.main()
