"""Synthetic scratch fixtures only; never connect to or mutate the live clone."""
from copy import deepcopy
from datetime import datetime, timedelta, timezone
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from cryptography import x509
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.serialization import Encoding
from cryptography.x509.oid import NameOID, ExtendedKeyUsageOID

import fuente_interna_sintetica as f
from clon_sql import ReadOnlyDB


def certificate(subject, issuer_cert=None, issuer_key=None):
    # Keys exist solely in memory in this fixture and never leave the sandbox.
    key = ec.generate_private_key(ec.SECP256R1())
    name = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Scratch synthetic")])
    now = datetime.now(timezone.utc)
    builder = x509.CertificateBuilder().subject_name(name).issuer_name(
        issuer_cert.subject if issuer_cert else name).public_key(key.public_key()).serial_number(
        x509.random_serial_number()).not_valid_before(now - timedelta(days=1)).not_valid_after(
        now + timedelta(days=1)).add_extension(x509.BasicConstraints(ca=issuer_cert is None, path_length=None), True)
    if issuer_cert:
        builder = builder.add_extension(x509.ExtendedKeyUsage([ExtendedKeyUsageOID.CLIENT_AUTH]), False).add_extension(
            x509.SubjectAlternativeName([x509.UniformResourceIdentifier("urn:vec:" + subject)]), False)
    cert = builder.sign(issuer_key or key, hashes.SHA256())
    return key, cert


class DB:
    def __init__(self, snapshot, evidence):
        self.snapshot, self.evidence = snapshot, evidence
        self.calls = []
        self.snapshot_calls = 0
        self.on_query = None

    def system_identity(self):
        return self.evidence["clone_identity"]

    def database_acl_digest(self):
        return self.evidence["clone_datacl_sha256"]

    def query(self, text):
        self.calls.append(text)
        if not text.startswith("BEGIN READ ONLY;\nSELECT ") or not text.endswith(";\nCOMMIT;"):
            raise AssertionError("RO wrapper missing")
        if self.on_query:
            self.on_query(text)
        if f.SNAPSHOT_SQL in text:
            self.snapshot_calls += 1
            return f.encoded(self.snapshot)
        if f.RO_SQL in text:
            return f.encoded({"transaction_read_only": "on", "database": "postgres"})
        return f.encoded({"derived_account_exists": False, "derived_account_projection_exists": False,
                          "derived_profile_assignment_exists": False, "derived_profile_context_exists": False})


class SourceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.h1 = self.root / "h1"
        self.h1.mkdir(mode=0o700)
        for sub in ("ca", "identidad", "mtls"):
            (self.h1 / sub).mkdir(mode=0o700)
        ca_key, ca = certificate("unused")
        self.evidence = {"installed_count": 62, "clone_identity": {"owned": "scratch"},
            "clone_datacl_sha256": "d" * 64, "results": {"ro": {"database": "postgres", "transaction_read_only": "on"},
            "center_catalog": {"catalog_sha": "c" * 64}, "nominal_assignments": []},
            "h1_public_files": {}, "public_certificates": {}, "dynamic_query_templates": {}}
        self.put("ca/ca.crt", ca.public_bytes(Encoding.PEM))
        self.evidence["public_certificates"]["ca"] = {"pem_file_sha256": f.digest(ca.public_bytes(Encoding.PEM))}
        for key, role, ident, certpath in (("rrhh", "tecnico_rrhh", "identidad/identidad.json", "mtls/cliente.crt"),
                                         ("intervencion", "intervencion", "identidad/intervencion.json", "mtls/intervencion.crt")):
            _, cert = certificate("desarrollo:scratch:" + key, ca, ca_key)
            identity = {"version": 1, "autoridad": "no_autoritativo", "roles": [role],
                "subject": "desarrollo:scratch:" + key, "display_name": "Scratch",
                "certificate_sha256": f.digest(cert.public_bytes(Encoding.DER))}
            self.put(ident, f.encoded(identity))
            self.put(certpath, cert.public_bytes(Encoding.PEM))
            self.evidence["h1_public_files"][ident] = {"sha256": f.digest(f.encoded(identity))}
            self.evidence["public_certificates"][key] = {"pem_file_sha256": f.digest(cert.public_bytes(Encoding.PEM))}
            absence = {"derived_account_exists": False, "derived_account_projection_exists": False,
                       "derived_profile_assignment_exists": False, "derived_profile_context_exists": False}
            self.evidence["results"]["source_h1_" + key] = absence
            refs = f.derived(identity["subject"], identity["certificate_sha256"])
            template = "SELECT jsonb_build_object('account','<CUENTA_DERIVADA_H1>','profile','<PERFIL_DERIVADO_H1>')"
            query = template.replace("<CUENTA_DERIVADA_H1>", refs["cuenta_ref"]).replace("<PERFIL_DERIVADO_H1>", refs["perfil_ref"])
            self.evidence["dynamic_query_templates"][key] = {"query_template": template, "query_sha256": f.digest(query.encode())}
        catalog = {"version": 1, "revision": 4, "sha256": "c" * 64, "document": {"entradas": [
            {"clave": "centro:scratch", "atributos": {"tipo": "centro"}},
            *[{"clave": "puesto:" + x, "atributos": {"tipo": "puesto_responsabilidad", "adscripcion_clave": "centro:scratch"}}
              for x in ("solicitante", "ratificador")]]}}
        self.snapshot = {"ro": "on", "catalog": catalog, "petitions": []}
        for index, function in enumerate(("solicitante", "ratificador"), 1):
            row = {"petition": "peticion:scratch", "version": index,
                "operation": "presentar" if index == 1 else "ratificar",
                "state": "pendiente_ratificacion" if index == 1 else "ratificada",
                "actor": "desarrollo:scratch:" + function, "profile": "perfil:" + function,
                "center": "centro:scratch", "position": "puesto:" + function,
                "material_sha256": "e" * 64, "assignment_ref": "asignacion:" + function,
                "assignment_version": 2, "assignment_sha256": str(index) * 64,
                "assignment": {"principal_id": "desarrollo:scratch:" + function, "perfil_activo_ref": "perfil:" + function,
                    "estado": "activa", "ambitos": [{"clave": "centro_ref", "valores": ["centro:scratch"]}]},
                "role": function + "_centro", "role_version": 2, "role_sha256": "a" * 64,
                "control_revision": 1, "control_sha256": "b" * 64, "control_state": "habilitada",
                "policies_revision": 1, "policies_sha256": "f" * 64}
            self.snapshot["petitions"].append(row)
            self.evidence["results"]["nominal_assignments"].append({
                "profile_ref_sha256": f.digest(row["profile"].encode()), "role": row["role"],
                "role_version": 2, "role_sha": "a" * 64, "assignment_version": 2,
                "assignment_sha": str(index) * 64, "assignment_active": True,
                "control_revision": 1, "control_sha": "b" * 64, "control_state": "habilitada"})
        self.act = self.root / "act.txt"
        self.pin_act()
        self.db = DB(self.snapshot, self.evidence)

    def put(self, relative, data):
        path = self.h1 / relative
        path.write_bytes(data)
        path.chmod(0o600)

    def pin_act(self):
        data = b"Scratch fixture\nEvidencia\n" + f.encoded(self.evidence)
        self.act.write_bytes(data)
        self.act.chmod(0o600)
        self.act_sha = f.digest(data)

    def prepare(self):
        with patch.object(f, "ACT_SHA", self.act_sha):
            return f.prepare(ReadOnlyDB(self.db), self.h1, self.act)

    def test_four_identities_separation_and_pending_certificates(self):
        output = self.prepare()
        source = f.decoded(output[f.SOURCE_NAME])
        self.assertEqual(len(source["personas"]), 4)
        self.assertEqual(source["estado"], "propuesta")
        self.assertIsNone(source["acreditacion"])
        self.assertFalse(source["provision_ejecutada"])
        for p in source["personas"][2:]:
            self.assertIsNone(p["sujeto_certificado"])
            self.assertIsNone(p["referencias_derivadas_certificado"])
            self.assertEqual(p["certificado"]["estado"], "pendiente")
        self.assertEqual(self.prepare(), output)
        self.assertNotIn(b"autoridad_maestra_acreditada", output[f.SOURCE_NAME])
        self.assertNotIn(b"PRIVATE KEY", output[f.SOURCE_NAME])

    def test_same_actor_rejected(self):
        self.snapshot["petitions"][1]["actor"] = self.snapshot["petitions"][0]["actor"]
        with self.assertRaises(f.InternalSourceError): self.prepare()

    def test_same_position_rejected(self):
        self.snapshot["petitions"][1]["position"] = self.snapshot["petitions"][0]["position"]
        with self.assertRaises(f.InternalSourceError): self.prepare()

    def test_wrong_center_position_rejected(self):
        self.snapshot["catalog"]["document"]["entradas"][2]["atributos"]["adscripcion_clave"] = "centro:otro"
        with self.assertRaises(f.InternalSourceError): self.prepare()

    def test_missing_h6_assignment_rejected(self):
        self.snapshot["petitions"][0]["assignment"] = None
        with self.assertRaises((f.InternalSourceError, TypeError)): self.prepare()

    def test_missing_h1_rejected_without_output(self):
        (self.h1 / "mtls/cliente.crt").unlink()
        with self.assertRaises(OSError): self.prepare()
        self.assertFalse((self.root / "proposal").exists())

    def test_fake_certificate_rejected(self):
        with self.assertRaises((f.InternalSourceError, ValueError)):
            f.public_identity((self.h1 / "identidad/identidad.json").read_bytes(), b"fake", (self.h1 / "ca/ca.crt").read_bytes(), "tecnico_rrhh")

    def test_wrong_ca_rejected(self):
        _, fake_ca = certificate("unused")
        with self.assertRaises(f.InternalSourceError):
            f.public_identity((self.h1 / "identidad/identidad.json").read_bytes(), (self.h1 / "mtls/cliente.crt").read_bytes(), fake_ca.public_bytes(Encoding.PEM), "tecnico_rrhh")

    def test_certificate_identity_sha_and_subject_mismatch(self):
        identity = f.decoded((self.h1 / "identidad/identidad.json").read_bytes())
        for key, value in (("certificate_sha256", "0" * 64), ("subject", "desarrollo:scratch:otro")):
            wrong = {**identity, key: value}
            with self.assertRaises(f.InternalSourceError):
                f.public_identity(f.encoded(wrong), (self.h1 / "mtls/cliente.crt").read_bytes(),
                                  (self.h1 / "ca/ca.crt").read_bytes(), "tecnico_rrhh")

    def test_changed_absence_rejected(self):
        query = self.db.query
        def changed(text):
            if "'account'" in text:
                value = f.decoded(query(text)); value["derived_profile_assignment_exists"] = True
                return f.encoded(value)
            return query(text)
        self.db.query = changed
        with self.assertRaisesRegex(f.InternalSourceError, "h1_absence_changed"): self.prepare()

    def test_clone_preimage_and_catalog_changed(self):
        self.db.system_identity = lambda: {"owned": "other"}
        with self.assertRaisesRegex(f.InternalSourceError, "owned_clone_preimage"): self.prepare()
        self.db.system_identity = lambda: self.evidence["clone_identity"]
        self.snapshot["catalog"]["sha256"] = "f" * 64
        with self.assertRaisesRegex(f.InternalSourceError, "catalog_changed"): self.prepare()

    def test_act_changed_and_output_unknown_keys_rejected(self):
        self.act.write_bytes(self.act.read_bytes() + b"changed")
        with self.assertRaisesRegex(f.InternalSourceError, "act_preimage"): self.prepare()
        self.pin_act()
        output = self.prepare()
        value = f.decoded(output[f.SOURCE_NAME]); value["autoridad_maestra_acreditada"] = True
        with patch.object(f, "PRIVATE_ROOT", self.root / "proposal"):
            with self.assertRaisesRegex(f.InternalSourceError, "source_schema"):
                f.write(self.root / "proposal", {f.SOURCE_NAME: f.encoded(value)})
        self.assertFalse((self.root / "proposal").exists())

    def test_output_symlink_root_refused(self):
        output = self.prepare()
        target = self.root / "target"; target.mkdir(mode=0o700)
        dest = self.root / "proposal"; dest.symlink_to(target)
        with patch.object(f, "PRIVATE_ROOT", dest):
            with self.assertRaises(RuntimeError): f.write(dest, output)
        self.assertEqual(list(target.iterdir()), [])

    def test_unknown_identity_fields_and_duplicate_json(self):
        data = f.decoded((self.h1 / "identidad/identidad.json").read_bytes())
        data["permiso"] = True
        with self.assertRaises(f.InternalSourceError):
            f.public_identity(f.encoded(data), (self.h1 / "mtls/cliente.crt").read_bytes(), (self.h1 / "ca/ca.crt").read_bytes(), "tecnico_rrhh")
        with self.assertRaises(f.InternalSourceError): f.decoded(b'{"version":1,"version":2}')

    def test_changed_assignment_from_act_rejected(self):
        self.snapshot["petitions"][0]["assignment_sha256"] = "9" * 64
        with self.assertRaisesRegex(f.InternalSourceError, "assignment_changed"): self.prepare()

    def test_changed_cas_during_read_rejected(self):
        def change(text):
            if f.SNAPSHOT_SQL in text and self.db.snapshot_calls:
                self.db.snapshot = deepcopy(self.snapshot)
                self.db.snapshot["petitions"][0]["policies_revision"] = 2
        self.db.on_query = change
        with self.assertRaisesRegex(f.InternalSourceError, "h6_preimage_changed"): self.prepare()

    def test_readonly_and_clone_identity_rejected(self):
        self.snapshot["ro"] = "off"
        with self.assertRaises(f.InternalSourceError): self.prepare()
        with self.assertRaises(f.InternalSourceError): f.prepare(self.db, self.h1, self.act)

    def test_symlink_and_hardlink_inputs_rejected(self):
        cert = self.h1 / "mtls/cliente.crt"
        other = self.h1 / "mtls/other.crt"
        cert.rename(other)
        cert.symlink_to(other.name)
        with self.assertRaises(OSError): self.prepare()
        cert.unlink()
        os.link(other, cert)
        with self.assertRaises(f.InternalSourceError): self.prepare()

    def test_toctou_source_change_rejected(self):
        def change(text):
            if f.SNAPSHOT_SQL in text and self.db.snapshot_calls:
                self.put("identidad/identidad.json", b"changed")
        self.db.on_query = change
        with self.assertRaisesRegex(f.InternalSourceError, "source_changed"): self.prepare()

    def test_atomic_private_replay_and_preimage_rejection(self):
        output = self.prepare()
        dest = self.root / "proposal"
        with patch.object(f, "PRIVATE_ROOT", dest):
            result = f.write(dest, output)
            before = (dest / f.SOURCE_NAME).stat()
            self.assertEqual(f.write(dest, output), result)
            after = (dest / f.SOURCE_NAME).stat()
            self.assertEqual((before.st_ino, before.st_mtime_ns), (after.st_ino, after.st_mtime_ns))
            self.assertEqual(after.st_mode & 0o777, 0o600)
            self.assertEqual(dest.stat().st_mode & 0o777, 0o700)
            changed = deepcopy(output)
            value = f.decoded(changed[f.SOURCE_NAME]); value["personas"][0]["nombre_visible_sintetico"] = "Otro"
            changed[f.SOURCE_NAME] = f.encoded(value)
            with self.assertRaises(RuntimeError): f.write(dest, changed)
            self.assertEqual((dest / f.SOURCE_NAME).read_bytes(), output[f.SOURCE_NAME])
        with self.assertRaises(f.InternalSourceError): f.write(self.root / "other", output)

    def test_prepare_has_no_filesystem_effects_or_non_ro_sql(self):
        before = {p.relative_to(self.root): p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.prepare()
        after = {p.relative_to(self.root): p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)
        self.assertTrue(self.db.calls)
        self.assertTrue(all(x.startswith("BEGIN READ ONLY;\nSELECT ") for x in self.db.calls))


if __name__ == "__main__":
    unittest.main()
