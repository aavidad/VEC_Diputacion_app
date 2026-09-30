import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from types import SimpleNamespace

SPEC = importlib.util.spec_from_file_location("internal_projection", Path(__file__).with_name("clon_interno_material.py"))
projection = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(projection)


class InternalProjectionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.state = Path(self.temp.name) / "state"
        self.state.mkdir(mode=0o700)
        self.material = self.state / "material"
        self.material.mkdir(mode=0o700)
        self.repo = Path(self.temp.name) / "repo"
        self.repo.mkdir()
        self.source = "a" * 40
        self.target = {"source_commit": self.source, "pg_port": 55531, "app_port": 18531, "container_id": "owned-container"}
        for name in projection.REQUIRED:
            if name.endswith(".json"):
                value = {"version": 1, "autoridad": "no_autoritativo"}
                if name == "identidad/usuarios-preferencias-interna.json":
                    value.update(superficie="interna_corporativa", cuentas=[{"cuenta_ref": "cta_original", "perfil_ref": "prf_original"}],
                                 dsn_usuarios=self.dsn())
                data = projection.json_bytes(value)
            else:
                data = ("preserved synthetic " + name).encode()
            self.put(self.material / name, data)
        for name in ("ca/ca.key", "ca/serie", "mtls/cliente.key", "mtls/cliente.p12", "mtls/cliente.p12.password",
                     "mtls/candidato.crt", "mtls/candidato.key", "identidad/candidato.json",
                     "identidad/usuarios-preferencias-externa.json", "identidad/.usuarios-preferencias-externa.backup.json",
                     "scripts/provision-external.sh", "pg/servidor.key", "comunicaciones/servidor.key"):
            self.put(self.material / name, b"outside-runtime original synthetic data")
        self.put(self.material / "mtls/solicitante.crt", b"requester certificate")
        self.put(self.material / "mtls/ratificador.crt", b"ratifier certificate")
        self.put(self.material / "identidad/centros.json", projection.json_bytes({"certificate": "mtls/solicitante.crt", "identity": "identidad/identidad.json"}))
        env = {"VEC_EXECUTION_PROFILE": "desarrollo", "VEC_AUTH_MODE": "desarrollo", "VEC_DEVELOPMENT_GUARD": projection.GUARD,
               "VEC_HTTP_ADDR": "127.0.0.1:18531",
               "VEC_DEVELOPMENT_MATERIAL_DIR": str(self.material), "VEC_CT_DATABASE_URL": self.dsn(),
               "VEC_EXTERNO_REGISTRO_DATABASE_URL": self.dsn(), "VEC_BOLSA_PORTAL_CANDIDATO_ENABLED": "true",
               "VEC_TLS_KEY_FILE": str(self.material / "tls/servidor.key"), "VEC_USUARIOS_PREFERENCIAS_ENABLED": "true",
               "VEC_SMTP_CA_FILE": str(self.material / "ca/ca.crt"), "PGPASSWORD": "dummy outside runtime"}
        self.put(self.state / "runtime-config.json", projection.json_bytes(env))
        metadata = {"propietario": "Codex-M", "estado": str(self.state), "contenedor": "vec-owned", "puerto_pg": 55531, "commit": self.source}
        for name in ("clon.json", "DB_READY.json"):
            self.put(self.state / name, projection.json_bytes(metadata))
        self.seal()

    def tearDown(self):
        self.temp.cleanup()

    def put(self, path, data):
        path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        path.write_bytes(data)
        path.chmod(0o600)

    def dsn(self):
        from urllib.parse import urlencode
        return "postgresql://nominal:dummy%40password@127.0.0.1:55531/postgres?" + urlencode({"sslmode": "verify-full", "sslrootcert": str(self.material / "pg/ca.crt")})

    def seal(self):
        files = {str(p.relative_to(self.state)): hashlib.sha256(p.read_bytes()).hexdigest() for p in self.material.rglob("*") if p.is_file()}
        files["runtime-config.json"] = hashlib.sha256((self.state / "runtime-config.json").read_bytes()).hexdigest()
        self.put(self.state / "material-manifest.json", projection.json_bytes({"owner": "Codex-M", "target": self.target, "files": files}))

    def provision(self, refresh=False):
        with patch.object(projection, "source_contracts", return_value={"source.go": "f" * 64}):
            return projection.provision(self.repo, "vec-owned", self.state, self.material, 55531,
                                        source_context={"source_ref": self.source}, refresh_operator_proof=refresh)

    def test_positive_copy_excludes_every_external_backup_and_client_key_preserves_history(self):
        original = {str(p.relative_to(self.state)): p.read_bytes() for p in self.state.rglob("*") if p.is_file()}
        descriptor = self.provision()
        root = self.state / "runtime-interno"
        copied = {str(p.relative_to(root / "material")) for p in (root / "material").rglob("*") if p.is_file()}
        self.assertEqual(copied, set(projection.REQUIRED) | {"mtls/solicitante.crt", "mtls/ratificador.crt", "identidad/centros.json", "portal-proceso.json", "desarrollo.env"})
        for name in ("ca/ca.crt", "mtls/cliente.crt", "mtls/intervencion.crt", "tls/servidor.key", "kms/clave-maestra.bin", "idempotencia/g1-localizador.bin"):
            self.assertEqual((self.material / name).read_bytes(), (root / "material" / name).read_bytes())
        for name, before in original.items():
            self.assertEqual((self.state / name).read_bytes(), before)
        config = json.loads((root / "runtime-config.json").read_text())
        self.assertEqual(config["VEC_PORTAL_PROCESO"], "interno")
        self.assertNotIn("PGPASSWORD", config)
        self.assertFalse(any(k.startswith("VEC_EXTERNO_") for k in config))
        self.assertNotIn("VEC_BOLSA_PORTAL_CANDIDATO_ENABLED", config)
        self.assertIn("nominal:dummy%40password@127.0.0.1:55531", config["VEC_CT_DATABASE_URL"])
        self.assertNotIn(str(self.material), config["VEC_CT_DATABASE_URL"])
        users = json.loads((root / "material/identidad/usuarios-preferencias-interna.json").read_text())
        self.assertEqual(users["cuentas"], [{"cuenta_ref": "cta_original", "perfil_ref": "prf_original"}])
        self.assertEqual(json.loads((root / "material/portal-proceso.json").read_text()), {"version": 1, "portal": "interno"})
        self.assertEqual(descriptor["material"], "runtime-interno/material")
        self.assertEqual(len(descriptor["rw"]), 4)
        self.assertEqual(descriptor["rw"][-1]["target"], str(root / "material/comunicaciones"))
        self.assertEqual(list((root / "material/comunicaciones").iterdir()), [])
        manifest = json.loads((root / "material-manifest.json").read_text())
        for name, sha in manifest["files"].items():
            self.assertEqual(hashlib.sha256((root / name).read_bytes()).hexdigest(), sha)
        sealed = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()}
        parent = json.loads((self.state / "material-manifest.json").read_text())
        parent["runtime_interno"] = descriptor
        self.put(self.state / "material-manifest.json", projection.json_bytes(parent))
        self.assertEqual(self.provision(), descriptor)
        self.assertEqual(sealed, {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()})

    def test_invalid_external_reference_or_source_seal_fails_before_writes(self):
        for failure in ("reference", "dsn", "surface", "seal"):
            with self.subTest(failure=failure):
                user = {"version": 1, "autoridad": "no_autoritativo", "superficie": "interna_corporativa", "cuentas": ["fixture"], "dsn_usuarios": self.dsn()}
                if failure == "reference": user["certificate"] = "mtls/candidato.crt"
                if failure == "dsn": user["dsn_usuarios"] = self.dsn().replace("127.0.0.1", "remote.invalid")
                if failure == "surface": user["superficie"] = "externa_personal"
                self.put(self.material / "identidad/usuarios-preferencias-interna.json", projection.json_bytes(user))
                self.seal()
                if failure == "seal": self.put(self.material / "tls/servidor.crt", b"changed after seal")
                with self.assertRaises(projection.ProjectionError): self.provision()
                self.assertFalse((self.state / "runtime-interno").exists())
                self.assertEqual(list(self.state.glob(".runtime-interno-*")), [])

    def test_replay_rejects_changed_projection_or_target_instead_of_renaming(self):
        self.provision()
        projected = self.state / "runtime-interno/material/mtls/cliente.crt"
        self.put(projected, b"unapproved changed certificate")
        with self.assertRaises(projection.ProjectionError): self.provision()
        self.assertEqual(projected.read_bytes(), b"unapproved changed certificate")

    def test_replay_rejects_extra_secret_backup_and_symlink_before_any_write(self):
        self.provision()
        root = self.state / "runtime-interno"
        for relative in ("material/ca/ca.key", "material/identidad/usuarios-preferencias-externa.json", "material/mtls/cliente.p12", "material/unknown-backup.txt"):
            with self.subTest(relative=relative):
                injected = root / relative
                self.put(injected, b"unapproved private fixture")
                with self.assertRaises(projection.ProjectionError): self.provision()
                self.assertEqual(injected.read_bytes(), b"unapproved private fixture")
                injected.unlink()
        linked = root / "material/operator-link"
        linked.symlink_to(self.material)
        with self.assertRaises(projection.ProjectionError): self.provision()
        self.assertTrue(linked.is_symlink())

    def test_owned_rw_data_does_not_enter_immutable_inventory_or_original(self):
        descriptor = self.provision()
        rw = self.state / "runtime-interno/rw"
        self.put(rw / "comunicaciones/notice.json", b"runtime synthetic notice")
        self.put(rw / "data/bolsa/state.json", b"runtime synthetic data")
        self.assertEqual(self.provision(), descriptor)
        self.assertFalse((self.material / "comunicaciones/notice.json").exists())
        config = json.loads((self.state / descriptor["config"]).read_text())
        for key in projection.RW_ENV:
            self.assertTrue(Path(config[key]).is_relative_to(rw))

    def test_unsealed_selected_file_and_changed_go_contract_fail_before_projection(self):
        self.put(self.material / "identidad/solicitante.json", projection.json_bytes({"version": 1}))
        with self.assertRaises(projection.ProjectionError): self.provision()
        self.assertFalse((self.state / "runtime-interno").exists())
        self.seal()
        with patch.object(projection, "source_contracts", side_effect=projection.ProjectionError("projection_source_contract_review_required")):
            with self.assertRaises(projection.ProjectionError):
                projection.provision(self.repo, "vec-owned", self.state, self.material, 55531)
        self.assertFalse((self.state / "runtime-interno").exists())

    def test_driver_seals_descriptor_separately_and_replays_identical_operator_inventory(self):
        spec = importlib.util.spec_from_file_location("projection_driver", Path(__file__).with_name("clon_material.py"))
        driver = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(driver)
        manifest = json.loads((self.state / "material-manifest.json").read_text())
        original = json.loads(json.dumps(manifest))
        original_files = {name: (self.state / name).read_bytes() for name in manifest["files"]}
        args = SimpleNamespace(repo=self.repo, container="vec-owned", pg_port=55531, engine="docker",
                               _source_context={"source_ref": self.source})
        with patch.object(driver, "load_profile_module", return_value=projection), patch.object(projection, "source_contracts", return_value={"source.go": "f" * 64}):
            result = driver.seal_internal_projection(args, self.state, self.source, manifest)
            sealed = (self.state / "material-manifest.json").read_bytes()
            self.assertEqual(driver.seal_internal_projection(args, self.state, self.source, result), result)
            self.assertEqual((self.state / "material-manifest.json").read_bytes(), sealed)
        self.assertEqual({k: v for k, v in result.items() if k != "runtime_interno"}, original)
        for name, before in original_files.items():
            self.assertEqual((self.state / name).read_bytes(), before)

    def proof_refresh_parent(self, descriptor):
        parent = json.loads((self.state / "material-manifest.json").read_text())
        parent.update(runtime_interno=descriptor, status="prepared")
        self.put(self.state / "material-manifest.json", projection.json_bytes(parent))

    def test_explicit_operator_reference_refresh_is_cas_only_and_records_preimage(self):
        descriptor = self.provision()
        root = self.state / "runtime-interno"
        before = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()}
        self.proof_refresh_parent(descriptor)
        with self.assertRaises(projection.ProjectionError): self.provision()
        self.assertEqual((root / "material-manifest.json").read_bytes(), before["material-manifest.json"])
        new_descriptor = self.provision(refresh=True)
        for name, data in before.items():
            if name != "material-manifest.json": self.assertEqual((root / name).read_bytes(), data)
        old, new = map(json.loads, (before["material-manifest.json"], (root / "material-manifest.json").read_bytes()))
        old["source_proof"]["operator_manifest_sha256"] = new["source_proof"]["operator_manifest_sha256"]
        self.assertEqual(old, new)
        receipt_path, = self.state.glob("runtime-interno-proof-refresh-*.json")
        receipt = json.loads(receipt_path.read_bytes())
        self.assertEqual(receipt["preimage_manifest_sha256"], descriptor["manifest_sha256"])
        self.assertEqual(receipt["postimage_manifest_sha256"], new_descriptor["manifest_sha256"])
        self.assertEqual(receipt["effect"], "operator_manifest_reference_only")
        self.proof_refresh_parent(new_descriptor)
        self.assertEqual(self.provision(refresh=True), new_descriptor)
        self.assertEqual(list(self.state.glob("runtime-interno-proof-refresh-*.json")), [receipt_path])

    def test_refresh_rejects_parent_cas_or_runtime_configuration_changes(self):
        descriptor = self.provision()
        root = self.state / "runtime-interno"
        before = (root / "material-manifest.json").read_bytes()
        self.proof_refresh_parent(dict(descriptor, manifest_sha256="e" * 64))
        with self.assertRaises(projection.ProjectionError): self.provision(refresh=True)
        self.assertEqual((root / "material-manifest.json").read_bytes(), before)
        self.proof_refresh_parent(descriptor)
        env = json.loads((self.state / "runtime-config.json").read_bytes())
        env["VEC_USUARIOS_CORREOS_ENABLED"] = "true"
        self.put(self.state / "runtime-config.json", projection.json_bytes(env))
        parent = json.loads((self.state / "material-manifest.json").read_bytes())
        parent["files"]["runtime-config.json"] = projection.digest((self.state / "runtime-config.json").read_bytes())
        self.put(self.state / "material-manifest.json", projection.json_bytes(parent))
        with self.assertRaises(projection.ProjectionError): self.provision(refresh=True)
        self.assertEqual((root / "material-manifest.json").read_bytes(), before)
        self.assertEqual(list(self.state.glob("runtime-interno-proof-refresh-*.json")), [])
