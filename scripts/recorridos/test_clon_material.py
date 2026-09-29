"""Behavior checks for private clone material; fixtures contain dummy secrets."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("clon_material", Path(__file__).with_name("clon_material.py"))
material = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(material)


class MaterialTests(unittest.TestCase):
    def test_dsn_retarget_preserves_login_and_credential_with_verified_local_tls(self):
        dsn = "postgresql://nominal:dummy%40credential@127.0.0.1:55441/postgres?sslmode=verify-full&sslrootcert=%2Fold%2Fca.crt"
        result = material.retarget_dsn(dsn, 55531, Path("/private/pg/ca.crt"))
        self.assertEqual(result, "postgresql://nominal:dummy%40credential@127.0.0.1:55531/postgres?sslmode=verify-full&sslrootcert=%2Fprivate%2Fpg%2Fca.crt")
        for invalid in (dsn.replace("127.0.0.1", "cidonia.cloud"), dsn.replace("55441", "5432"),
                        dsn.replace("verify-full", "disable"), dsn + "&sslmode=disable", dsn.replace("/postgres?", "/other?")):
            with self.subTest(invalid=invalid), self.assertRaises(material.MaterialError):
                material.retarget_dsn(invalid, 55531, Path("/private/ca.crt"))

    def test_container_must_be_owned_running_and_only_loopback_bound(self):
        data = {"Config": {"Labels": {"vec.recorridos.owner": "Codex-M"}}, "State": {"Running": True},
                "NetworkSettings": {"Ports": {"5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": "55531"}]}}}
        material.validate_container(data, 55531)
        for mutate in (lambda d: d["Config"]["Labels"].clear(),
                       lambda d: d["State"].update(Running=False),
                       lambda d: d["NetworkSettings"]["Ports"]["5432/tcp"][0].update(HostIp="0.0.0.0")):
            invalid = json.loads(json.dumps(data))
            mutate(invalid)
            with self.assertRaises(material.MaterialError):
                material.validate_container(invalid, 55531)

    def test_private_file_reader_rejects_symlinks_hardlinks_and_public_modes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root / "secret"
            material.private_write(path, b"dummy")
            self.assertEqual(material.private_read(path), b"dummy")
            link = root / "link"
            link.symlink_to(path)
            with self.assertRaises(material.MaterialError):
                material.private_read(link)
            hardlink = root / "hardlink"
            os.link(path, hardlink)
            with self.assertRaises(material.MaterialError):
                material.private_read(path)
            hardlink.unlink()
            path.chmod(0o644)
            with self.assertRaises(material.MaterialError):
                material.private_read(path)

    def test_env_is_parsed_without_executing_shell_and_rejects_duplicates(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "runtime.env"
            material.private_write(path, "VEC_HTTP_ADDR='127.0.0.1:18531'\n")
            self.assertEqual(material.load_env(path), {"VEC_HTTP_ADDR": "127.0.0.1:18531"})
            for invalid in ("VEC_A=$(touch injected)\n", "VEC_A='`touch injected`'\n", "VEC_A=1\nVEC_A=2\n"):
                path.write_text(invalid)
                with self.assertRaises(material.MaterialError):
                    material.load_env(path)
            self.assertFalse((Path(directory) / "injected").exists())

    def test_replay_checks_source_container_and_all_bytes_without_rotating(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            identity = {"source_commit": "f" * 40, "container_id": "clone", "pg_port": 55531, "app_port": 18531}
            material.private_write(root / "runtime.env", "VEC_AUTH_MODE=desarrollo\n")
            expected = hashlib.sha256((root / "runtime.env").read_bytes()).hexdigest()
            manifest = {"target": identity, "files": {"runtime.env": expected}, "blockers": []}
            material.json_write(root / "material-manifest.json", manifest)
            self.assertEqual(material.verify_existing(root, identity), manifest)
            with self.assertRaises(material.MaterialError):
                material.verify_existing(root, dict(identity, container_id="another"))
            (root / "runtime.env").write_text("changed\n")
            with self.assertRaises(material.MaterialError):
                material.verify_existing(root, identity)

    def test_source_upgrade_preserves_material_and_rejects_destination_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            old = {"source_commit": "a" * 40, "container_id": "clone", "pg_port": 55531, "app_port": 18531}
            material.private_write(root / "material/cert", "preserved certificate")
            sha = hashlib.sha256((root / "material/cert").read_bytes()).hexdigest()
            material.json_write(root / "material-manifest.json", {"target": old, "files": {"material/cert": sha}, "blockers": []})
            args = SimpleNamespace(repo=root, pg_port=55531)
            changed = dict(old, source_commit="b" * 40)
            material.json_write(root / "DB_READY.json", {"commit": "b" * 40, "sql_instaladas": 34})
            material.json_write(root / "sql-journal.json", {"source_ref": "b" * 40, "installed": [
                {"position": n, "path": "deploy/postgresql/fixture/" + str(n) + ".sql", "sha256": hashlib.sha256(b"").hexdigest()}
                for n in range(1, 35)]})
            with patch.object(material, "run", return_value=b""), patch.object(material, "probe_pg_tls"), patch.object(material.socket, "create_connection", side_effect=OSError):
                with self.assertRaises(material.MaterialError):
                    material.update_source(args, root, dict(changed, container_id="different"))
                upgraded = material.update_source(args, root, changed)
            self.assertEqual(upgraded["source_updated_from"], old["source_commit"])
            self.assertEqual(material.verify_existing(root, changed)["files"], {"material/cert": sha})
            self.assertEqual((root / "material/cert").read_text(), "preserved certificate")

    def test_completion_missing_module_has_no_side_effect_and_seals_each_completed_dependency(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            args = SimpleNamespace(repo=root, container="vec-fixture", pg_port=55531, engine="docker")
            material.private_write(root / "material/identity", "preserved")
            material.json_write(root / "runtime-config.json", {"VEC_AUTH_MODE": "desarrollo"})
            material.private_write(root / "runtime.env", "VEC_AUTH_MODE=desarrollo\n")
            material.json_write(root / "perfiles.json", {"profiles": {}, "blockers": []})
            blockers = [{"code": code} for code in ("concesiones_correos_imagen_pendientes", "bback_politica_ofertas_pendiente", "cuenta_contexto_candidato_pendiente")]
            manifest = {"target": {}, "files": {}, "blockers": blockers}
            material.json_write(root / "material-manifest.json", manifest)
            called = []
            def first(**kwargs):
                called.append("users")
                return {"profiles": {"users": {"status": "prepared"}}}
            with patch.object(material, "load_profile_module", side_effect=[SimpleNamespace(provision=first), material.MaterialError("missing")]):
                with self.assertRaises(material.MaterialError):
                    material.complete_profiles(args, root, manifest)
            self.assertEqual(called, [])
            def later(**kwargs):
                raise material.MaterialError("later dependency blocked")
            with patch.object(material, "load_profile_module", side_effect=[SimpleNamespace(provision=first), SimpleNamespace(provision=later), SimpleNamespace(provision=later)]):
                with self.assertRaises(material.MaterialError):
                    material.complete_profiles(args, root, manifest)
            current = material.verify_existing(root, {})
            self.assertEqual(current["status"], "partial_blocked")
            self.assertFalse(current["profiles_provisioned"])
            self.assertTrue(current["profiles_provisioning_attempted"])
            self.assertNotIn("concesiones_correos_imagen_pendientes", [b["code"] for b in current["blockers"]])
            self.assertEqual((root / "material/identity").read_text(), "preserved")

    def test_full_preparation_preserves_existing_actors_and_never_claims_candidate_account(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            base, repo, output = root / "base", root / "repo", root / "output"
            base.mkdir(mode=0o700)
            repo.mkdir()
            # Use the repository's real generator solely for fresh dummy fixtures.
            source = Path(__file__).resolve().parents[1] / "generar_credenciales_desarrollo.sh"
            material.run(["bash", str(source), str(base / "material")])
            base = base / "material"
            for surface, actor_name in (("interna", "identidad"), ("externa", "intervencion")):
                actor = json.loads((base / f"identidad/{actor_name}.json").read_text())
                data = {"version": 1, "autoridad": "no_autoritativo", "cuentas": [{"sujeto": actor["subject"],
                        "certificado_sha256": actor["certificate_sha256"], "cuenta_ref": "cta_dummy_" + surface,
                        "perfil_ref": "prf_dummy_" + surface}],
                        "dsn_usuarios": "postgresql://dummy@127.0.0.1:55441/postgres?sslmode=verify-full&sslrootcert=%2Fdummy"}
                material.json_write(base / f"identidad/usuarios-preferencias-{surface}.json", data)
            output.mkdir(mode=0o700)
            material.json_write(output / "DB_READY.json", {"commit": "a" * 40, "contenedor": "vec-owned",
                "propietario": "Codex-M", "puerto_pg": 55531, "puerto_web": 18531})
            env = root / "source.env"
            material.private_write(env, "".join(k + "='postgresql://dummy@127.0.0.1:55441/postgres?sslmode=verify-full&sslrootcert=%2Fdummy'\n" for k in material.DSN_KEYS))
            args = argparse.Namespace(repo=repo, base_material=base, output=output, source_env=env,
                    container="vec-owned", engine="docker", pg_port=55531, port=18531)
            real_run = material.run
            def mocked_run(argv, text=None):
                if argv[0] == "git":
                    return ("a" * 40).encode()
                if argv[:2] == ["docker", "inspect"]:
                    return json.dumps([{"Id": "dummy-clone", "Config": {"Labels": {"vec.recorridos.owner": "Codex-M", "vec.recorridos.state": str(output)}},
                        "State": {"Running": True}, "NetworkSettings": {"Ports": {"5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": "55531"}]}}}]).encode()
                if argv[:2] == ["docker", "exec"]:
                    if text == "SHOW data_directory;\n":
                        return b"/var/lib/postgresql/18/docker\n"
                    if not text or "WITH latest" not in text:
                        return b""
                    return json.dumps({"version": 1, "revision": 2, "centers": ["centro-fixture"],
                        "positions": [{"ref": "puesto-fixture-solicitante", "center": "centro-fixture"},
                                      {"ref": "puesto-fixture-ratificador", "center": "centro-fixture"}]}).encode()
                return real_run(argv, text)
            with patch.object(material, "run", side_effect=mocked_run), patch.object(material, "probe_pg_tls") as probe:
                manifest = material.prepare(args)
                second = material.prepare(args)
            self.assertEqual(manifest, second)
            self.assertEqual(probe.call_count, 2)
            self.assertFalse(manifest["sql_applied"])
            self.assertEqual(manifest["status"], "partial_blocked")
            for relative in (*material.HISTORY_FILES, "identidad/identidad.json", "mtls/cliente.crt", "ca/ca.crt"):
                self.assertEqual((base / relative).read_bytes(), (output / "material" / relative).read_bytes())
            profiles = json.loads((output / "perfiles.json").read_text())["profiles"]
            self.assertEqual(profiles["area_personal"]["subject"], profiles["intervencion"]["subject"])
            self.assertNotEqual(profiles["candidato"]["subject"], profiles["area_personal"]["subject"])
            self.assertEqual(profiles["candidato"]["status"], "certificate_only")
            self.assertEqual(profiles["centro_solicitante"]["status"], "bound_catalog")
            self.assertEqual(profiles["centro_solicitante"]["centro_ref"], profiles["ratificador"]["centro_ref"])
            self.assertNotEqual(profiles["centro_solicitante"]["subject"], profiles["ratificador"]["subject"])
            self.assertFalse((output / "material/identidad/bolsa-candidato.json").exists())
            self.assertNotIn("VEC_CT_PROVISION_PERFILES_RRHH", (output / "runtime.env").read_text())


if __name__ == "__main__":
    unittest.main()
