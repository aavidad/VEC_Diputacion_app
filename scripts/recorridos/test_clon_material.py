"""Behavior checks for private clone material; fixtures contain dummy secrets."""
import argparse
import hashlib
import importlib.util
import contextlib
import io
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

    def test_module_failure_cli_reports_class_and_private_log_without_trace_or_credentials(self):
        class UsersError(RuntimeError):
            pass
        error = material.ModuleProvisionError("clon_usuarios", UsersError("postgresql://dummy:private-value@localhost/db"))
        stderr = io.StringIO()
        argv = ["clon_material.py", "--repo", "/fixture", "--container", "vec-owned",
                "--output", "/fixture-state", "--port", "18531", "--pg-port", "55531"]
        with patch.object(material.sys, "argv", argv), patch.object(material, "prepare", side_effect=error), contextlib.redirect_stderr(stderr):
            self.assertEqual(material.main(), 1)
        diagnostic = json.loads(stderr.getvalue())
        self.assertEqual(diagnostic["module"], "clon_usuarios")
        self.assertEqual(diagnostic["code"], "UsersError")
        self.assertEqual(diagnostic["detail_log"], "material-provision-private.log")
        self.assertNotIn("private-value", stderr.getvalue())
        self.assertNotIn("Traceback", stderr.getvalue())

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

    def test_material_delegates_closed_receipts_to_central_approval_without_reading_wip(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            git = root / "git"
            git.mkdir()
            state = root / "state"
            state.mkdir(mode=0o700)
            source = "b" * 40
            archive = state / ("source-" + source)
            archive.mkdir(mode=0o700)
            args = SimpleNamespace(repo=git)
            material.json_write(state / "DB_READY.json", {"commit": source, "sql_instaladas": 38})
            installed = [{"position": n} for n in range(1, 39)]
            material.json_write(state / "sql-journal.json", {"current_source_ref": source, "verified_source_ref": source,
                "approved_sql_ref": "a" * 40, "current_plan_sha": "c" * 64, "inventory_sha": "d" * 64, "installed": installed})
            plan = {"source_ref": source, "approved_sql_ref": "a" * 40, "file_count": 38,
                    "plan_sha": "c" * 64, "inventory_sha": "d" * 64, "entries": []}
            central = SimpleNamespace(approved_source_plan=unittest.mock.Mock(return_value=plan), validate_receipts=unittest.mock.Mock())
            with patch.object(material, "load_source_validator", return_value=central):
                validated, _ = material.validate_source_receipts(args, state, source)
            self.assertEqual(validated, plan)
            central.approved_source_plan.assert_called_once_with(archive, source, git_repo=git)
            central.validate_receipts.assert_called_once_with(installed, plan, complete=True)
            self.assertEqual(args._source_context, {"source_repo": archive, "git_repo": git, "source_ref": source})
            saved = json.loads(material.private_read(state / "sql-journal.json"))
            material.replace_private(state / "sql-journal.json", dict(saved, verified_source_ref="e" * 40))
            with patch.object(material, "load_source_validator", return_value=central), self.assertRaises(material.MaterialError):
                material.validate_source_receipts(args, state, source)
            material.replace_private(state / "sql-journal.json", saved)
            central.validate_receipts.side_effect = RuntimeError("unreviewed SQL inventory")
            with patch.object(material, "load_source_validator", return_value=central), self.assertRaises(material.ModuleProvisionError):
                material.validate_source_receipts(args, state, source)
            material.replace_private(state / "DB_READY.json", {"commit": source, "sql_instaladas": 39})
            with patch.object(material, "load_source_validator", return_value=central), self.assertRaises(material.MaterialError):
                material.validate_source_receipts(args, state, source)

    def test_fresh_and_repeat_flags_order_complete_coverage_nominal_projection_before_app(self):
        args = SimpleNamespace(complete_profiles=True, repair_coverage_connect=True, repair_nominal_connect=True)
        events = []
        original, completed = {"status": "base"}, {"status": "completed"}
        with patch.object(material, "complete_profiles", side_effect=lambda *a: events.append("complete") or completed), \
             patch.object(material, "repair_coverage_connect", side_effect=lambda *a: events.append("coverage")), \
             patch.object(material, "repair_nominal_connect", side_effect=lambda *a: events.append("nominal")), \
             patch.object(material, "seal_internal_projection", side_effect=lambda *a: events.append("projection") or completed):
            for branch in ("fresh", "repeat"):
                events.clear()
                self.assertEqual(material.finish_preparation(args, Path("/private"), "b" * 40, original), completed)
                self.assertEqual(events, ["complete", "coverage", "nominal", "projection"])

    def test_projection_failure_preserves_last_operator_manifest_and_logs_private_reason(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = {"status": "partial_blocked", "files": {"material/internal": "c" * 64}}
            material.json_write(root / "material-manifest.json", manifest)
            before = (root / "material-manifest.json").read_bytes()
            args = SimpleNamespace(repo=root, container="vec-owned", pg_port=55531, engine="docker")
            module = SimpleNamespace(provision=unittest.mock.Mock(side_effect=RuntimeError("private fixture failure")))
            with patch.object(material, "load_profile_module", return_value=module), self.assertRaises(material.ModuleProvisionError):
                material.seal_internal_projection(args, root, "b" * 40, manifest)
            self.assertEqual((root / "material-manifest.json").read_bytes(), before)
            self.assertIn("private fixture failure", (root / "material-provision-private.log").read_text())

    def test_nominal_connect_matrix_is_closed_and_motives_postimage_is_strict(self):
        self.assertEqual(len(material.NOMINAL_CONNECT_GROUPS), 8)
        self.assertEqual(set(material.NOMINAL_CONNECT_GROUPS), set(material.NOMINAL_CONNECT_SOURCES))
        self.assertFalse(any("propietario" in role or "migrador" in role for role in material.NOMINAL_CONNECT_GROUPS.values()))
        value = {"f" + str(n): True for n in range(5, 16)}
        value.update(f1="nominal_login", f2="nominal_login", f3="16786", f4="16790")
        self.assertTrue(material.motives_metadata_valid(value, "nominal_login"))
        for key in ("f5", "f6", "f9", "f10", "f11", "f12", "f13", "f14", "f15"):
            self.assertFalse(material.motives_metadata_valid(dict(value, **{key: False}), "nominal_login"))
        self.assertFalse(material.motives_metadata_valid(dict(value, f4=value["f3"]), "nominal_login"))
        self.assertFalse(material.motives_metadata_valid(dict(value, f1="postgres"), "nominal_login"))

    def test_coverage_accreditation_denies_any_extra_failure_and_does_not_infer_socket_tls(self):
        value = {"f" + str(n): True for n in range(4, 21)}
        value.update(f1="17852", f2="vec_ct_o207_lector", f3="vec_ct_o207_lector")
        self.assertTrue(material.coverage_metadata_valid(value, physical_login="vec_ct_o207_lector"))
        missing = dict(value, f11=False)
        self.assertTrue(material.coverage_metadata_valid(missing, physical_login="vec_ct_o207_lector", missing_connect=True))
        for key in ("f4", "f5", "f10", "f12", "f19", "f20"):
            self.assertFalse(material.coverage_metadata_valid(dict(missing, **{key: False}), physical_login="vec_ct_o207_lector", missing_connect=True))
        socket_metadata = dict(value, f2="postgres", f3="postgres", f4=False)
        self.assertTrue(material.coverage_metadata_valid(socket_metadata))
        self.assertFalse(material.coverage_metadata_valid(socket_metadata, physical_login="vec_ct_o207_lector"))
        self.assertFalse(material.coverage_metadata_valid(dict(value, f1="0")))

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
            with patch.object(material, "run", return_value=b""), patch.object(material, "probe_pg_tls"), patch.object(material.socket, "create_connection", side_effect=OSError), patch.object(material, "validate_source_receipts", return_value=({"approved_sql_ref": "b" * 40, "plan_sha": "c" * 64, "inventory_sha": "d" * 64, "file_count": 38}, b"receipt")):
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
            for surface in ("interna", "externa"):
                material.json_write(root / f"material/identidad/usuarios-preferencias-{surface}.json", {"cuentas": [{
                    "sujeto": "synthetic-" + surface, "certificado_sha256": "c" * 64,
                    "cuenta_ref": "cta_old_" + surface, "perfil_ref": "prf_preserved_" + surface}]})
            material.json_write(root / "runtime-config.json", {"VEC_AUTH_MODE": "desarrollo"})
            material.private_write(root / "runtime.env", "VEC_AUTH_MODE=desarrollo\n")
            material.json_write(root / "perfiles.json", {"profiles": {}, "blockers": []})
            blockers = [{"code": code} for code in ("concesiones_correos_imagen_pendientes", "bback_politica_ofertas_pendiente", "cuenta_contexto_candidato_pendiente")]
            manifest = {"target": {}, "files": {}, "blockers": blockers}
            material.json_write(root / "material-manifest.json", manifest)
            called = []
            def first(**kwargs):
                called.append("users")
                for surface in ("interna", "externa"):
                    path = root / f"material/identidad/usuarios-preferencias-{surface}.json"
                    config = json.loads(material.private_read(path))
                    config["cuentas"][0]["cuenta_ref"] = "cta_new_" + surface
                    material.replace_private(path, config)
                return {"profiles": {"users": {"status": "prepared"}}, "blockers": [{"code": "concesiones_correos_imagen_pendientes"}]}
            with patch.object(material, "load_profile_module", side_effect=[SimpleNamespace(provision=first), material.MaterialError("missing")]):
                with self.assertRaises(material.MaterialError):
                    material.complete_profiles(args, root, manifest)
            self.assertEqual(called, [])
            def later(**kwargs):
                raise material.MaterialError("later dependency blocked")
            with patch.object(material, "load_profile_module", side_effect=[SimpleNamespace(provision=first), SimpleNamespace(provision=later), SimpleNamespace(provision=later), SimpleNamespace(provision=later), SimpleNamespace(provision=later)]):
                with self.assertRaises(material.MaterialError):
                    material.complete_profiles(args, root, manifest)
            current = material.verify_existing(root, {})
            self.assertEqual(current["status"], "partial_blocked")
            self.assertFalse(current["profiles_provisioned"])
            self.assertTrue(current["profiles_provisioning_attempted"])
            self.assertIn("concesiones_correos_imagen_pendientes", [b["code"] for b in current["blockers"]])
            self.assertEqual([b["code"] for b in current["blockers"]].count("concesiones_correos_imagen_pendientes"), 1)
            current_profiles = json.loads(material.private_read(root / "perfiles.json"))
            self.assertEqual(current_profiles["users"]["interna"][0]["cuenta_ref"], "cta_new_interna")
            self.assertEqual(current_profiles["users"]["externa"][0]["cuenta_ref"], "cta_new_externa")
            self.assertEqual((root / "material-provision-private.log").stat().st_mode & 0o777, 0o600)
            self.assertIn("later dependency blocked", (root / "material-provision-private.log").read_text())
            self.assertEqual((root / "material/identity").read_text(), "preserved")
            def h4(**kwargs):
                sealed = material.verify_existing(root, {})
                self.assertIn("concesiones_correos_imagen_pendientes", [b["code"] for b in sealed["blockers"]])
                self.assertEqual(json.loads(material.private_read(root / "perfiles.json"))["users"]["interna"][0]["cuenta_ref"], "cta_new_interna")
                called.append("h4")
                return {"env": {"VEC_USUARIOS_PREFERENCIAS_ENABLED": "true"}, "blockers": []}
            args.smtp_port = 11125
            args.mailpit_http_port = 18542
            args._source_context = {"source_repo": root / "source", "git_repo": root, "source_ref": "b" * 40}
            def comunicaciones(**kwargs):
                self.assertEqual(kwargs["smtp_port"], args.smtp_port)
                self.assertEqual(kwargs["http_port"], args.mailpit_http_port)
                called.append("comunicaciones")
                certificate = root / "material/comunicaciones/servidor.crt"
                material.private_write(certificate, "synthetic certificate")
                return {"env": {"VEC_USUARIOS_CORREOS_ENABLED": "true", "VEC_USUARIOS_IMAGEN_ENABLED": "true"},
                        "files": [str(certificate)], "blockers": []}
            def bolsa(**kwargs):
                self.assertEqual(kwargs["source_context"], args._source_context)
                called.append("bolsa")
                return {"blockers": []}
            def candidato(**kwargs):
                called.append("candidato")
                return {"blockers": [{"profile": "candidato", "code": "contexto_externo_provision_autoridad_ausente_main"}]}
            called.clear()
            with patch.object(material, "load_profile_module", side_effect=[SimpleNamespace(provision=first), SimpleNamespace(provision=h4), SimpleNamespace(provision=comunicaciones), SimpleNamespace(provision=bolsa), SimpleNamespace(provision=candidato)]):
                final = material.complete_profiles(args, root, current)
            self.assertEqual(called, ["users", "h4", "comunicaciones", "bolsa", "candidato"])
            self.assertNotIn("concesiones_correos_imagen_pendientes", [b["code"] for b in final["blockers"]])
            self.assertEqual(json.loads(material.private_read(root / "runtime-config.json"))["VEC_USUARIOS_PREFERENCIAS_ENABLED"], "true")
            runtime = json.loads(material.private_read(root / "runtime-config.json"))
            self.assertEqual(runtime["VEC_USUARIOS_CORREOS_ENABLED"], "true")
            self.assertEqual(runtime["VEC_USUARIOS_IMAGEN_ENABLED"], "true")
            self.assertIn("material/comunicaciones/servidor.crt", final["files"])
            material.verify_existing(root, {})

    def test_names_catalog_schema_rejects_extra_missing_duplicate_and_non_name_data(self):
        catalog = json.loads(material.SYNTHETIC_PROFILE_FIXTURE.read_text())
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "catalog.json"
            for case in ("extra", "missing", "version", "role", "blank", "duplicate", "duplicate_json"):
                invalid = json.loads(json.dumps(catalog))
                if case == "extra": invalid["nombres"]["rrhh"] = "Nombre Ajeno"
                if case == "missing": del invalid["nombres"]["candidato"]
                if case == "version": invalid["version"] = True
                if case == "role": invalid["roles"] = ["admin"]
                if case == "blank": invalid["nombres"]["ratificador"] = " "
                if case == "duplicate": invalid["nombres"]["ratificador"] = invalid["nombres"]["candidato"]
                path.write_text(json.dumps(invalid) if case != "duplicate_json" else '{"version":1,"version":1}')
                with self.subTest(case=case), patch.object(material, "SYNTHETIC_PROFILE_FIXTURE", path), self.assertRaises(material.MaterialError):
                    material.fresh_profile_names()

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
            with patch.object(material, "run", side_effect=mocked_run), patch.object(material, "probe_pg_tls") as probe, patch.object(material, "validate_source_receipts", return_value=({"approved_sql_ref": "a" * 40, "plan_sha": "c" * 64, "inventory_sha": "d" * 64, "file_count": 38}, b"receipt")):
                invalid = root / "invalid-catalog.json"
                invalid.write_text('{"version":1,"autoridad":"no_autoritativo","nombres":{}}')
                with patch.object(material, "SYNTHETIC_PROFILE_FIXTURE", invalid), patch.object(material, "certificate") as generate, patch.object(material, "configure_pg_tls") as tls, self.assertRaises(material.MaterialError):
                    material.prepare(args)
                generate.assert_not_called()
                tls.assert_not_called()
                self.assertFalse((output / "material").exists())
                manifest = material.prepare(args)
                preserved = {key: (output / "material/identidad" / (role[1] + ".json")).read_bytes() for key, role in material.ROLES.items()}
                with patch.object(material, "fresh_profile_names", side_effect=AssertionError("sealed clone must never rename actors")):
                    second = material.prepare(args)
                self.assertEqual(preserved, {key: (output / "material/identidad" / (role[1] + ".json")).read_bytes() for key, role in material.ROLES.items()})
            self.assertEqual(manifest, second)
            self.assertEqual(probe.call_count, 2)
            self.assertFalse(manifest["sql_applied"])
            self.assertEqual(manifest["status"], "partial_blocked")
            for relative in (*material.HISTORY_FILES, "identidad/identidad.json", "mtls/cliente.crt", "ca/ca.crt"):
                self.assertEqual((base / relative).read_bytes(), (output / "material" / relative).read_bytes())
            profiles = json.loads((output / "perfiles.json").read_text())["profiles"]
            fixture_bytes = material.SYNTHETIC_PROFILE_FIXTURE.read_bytes()
            names = json.loads(fixture_bytes)["nombres"]
            self.assertEqual(manifest["synthetic_profiles_fixture"]["sha256"], hashlib.sha256(fixture_bytes).hexdigest())
            for key, display_name in names.items():
                actor = json.loads((output / "material/identidad" / (material.ROLES[key][1] + ".json")).read_text())
                self.assertEqual(actor["display_name"], display_name)
                self.assertEqual(actor["subject"], "desarrollo:clon-recorridos:" + key)
                self.assertEqual(actor["roles"], [material.ROLES[key][2]])
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
