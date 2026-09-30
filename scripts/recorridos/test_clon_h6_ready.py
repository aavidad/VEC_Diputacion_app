"""Pruebas documentales puras: sin Docker, SQL, credenciales ni red."""
import copy
from dataclasses import asdict
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import uuid

import clon_h6_ready as ready


class ReadyTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="h6-prueba-")
        self.addCleanup(self.temp.cleanup)
        self.state = Path(self.temp.name) / "clon-ámbito"
        self.state.mkdir(mode=0o700)
        source = "1" * 40
        target = {"source_commit": source, "pg_port": 55441, "app_port": 8445}
        config = {"VEC_PORTAL_MODE": "interno"}
        config_bytes = ready.canonical(config)
        projected = {"owner": "Codex-M", "target": target, "status": "pending_ad132",
            "files": {"runtime-config.json": ready.sha(config_bytes)},
            "database_ready": False, "database_identity_verified": False}
        self.write("runtime-interno/runtime-config.json", config_bytes)
        projected_bytes = ready.canonical(projected)
        self.write("runtime-interno/material-manifest.json", projected_bytes)
        principal = {"owner": "Codex-M", "target": target, "status": "prepared", "blockers": [],
            "files": {"runtime-config.json": ready.sha(config_bytes)}, "runtime_interno": {
                "status": "pending_ad132", "manifest": "runtime-interno/material-manifest.json",
                "config": "runtime-interno/runtime-config.json", "source_commit": source,
                "manifest_sha256": ready.sha(projected_bytes)}}
        self.write("runtime-config.json", config_bytes)
        principal_bytes = ready.canonical(principal)
        self.write("material-manifest.json", principal_bytes)
        self.write("material/pg-ca.crt", b"CA PUBLICA SINTETICA\n")
        self.write("bin/vec-server", b"BINARIO SINTETICO SIN EJECUTAR\n")
        plan = {"conexiones": [{"fuente": "entorno:VEC_PRUEBA_" + str(i),
            "login": "login_prueba", "rol": "vec_rol_prueba"} for i in range(24)],
            "huellas": {"portal-proceso.json": "f" * 64}}
        plan_bytes = ready.canonical(plan)
        self.write("plan-conexiones.json", plan_bytes)
        self.approval = {"repo": self.state / "repo-aprobado", "source_commit": source,
            "container": "vec-prueba", "pg_container_id": "a" * 64,
            "pg_image_id": "sha256:" + "b" * 64,
            "pg_volume": {"source": "/dev/shm/vec-recorridos-prueba", "dev": 1, "ino": 2},
            "pg_port": 55441, "app_port": 8445, "identidad_clon": "c" * 64,
            "h6_package": self.state / "package.tar.gz", "h6_lock": self.state / "release.lock",
            "approved_package_sha256": "d" * 64, "approved_lock_sha256": "e" * 64,
            "h1_state_file": self.state / "h1.json", "estado_h1_sha": "f" * 64,
            "canary_plan": self.state / "plan-conexiones.json",
            "canary_plan_receipt": self.state / "plan-canonico-clon.json",
            "approved_canary_plan_sha256": ready.sha(plan_bytes),
            "canary_image_id": "sha256:" + "2" * 64, "arranque_sha256": "3" * 64,
            "ca_path": self.state / "material/pg-ca.crt",
            "ca_sha256": ready.sha(b"CA PUBLICA SINTETICA\n"),
            "binary_path": self.state / "bin/vec-server",
            "binary_sha256": ready.sha(b"BINARIO SINTETICO SIN EJECUTAR\n"),
            "material_manifest_sha256": ready.sha(principal_bytes),
            "runtime_config_sha256": ready.sha(config_bytes),
            "projection_manifest_sha256": ready.sha(projected_bytes),
            "projection_config_sha256": ready.sha(config_bytes)}
        canary_receipt = {"version": 1, "kind": "clon", "package_sha256": "d" * 64,
            "source_commit": source, "pg_container_id": "a" * 64,
            "plan_sha256": ready.sha(plan_bytes), "material_inventory_sha256": ready.sha(ready.canonical(plan["huellas"])),
            "canary_image_id": self.approval["canary_image_id"], "arranque_sha256": "3" * 64,
            "canary_output_sha256": ready.sha(plan_bytes)}
        self.write("plan-canonico-clon.json", ready.canonical(canary_receipt))
        self.approval["approved_canary_receipt_sha256"] = ready.sha(ready.canonical(canary_receipt))
        self.request = ready.clon_ad132_recibo.Request(
            package_tar=self.approval["h6_package"], package_root=self.state / "package",
            approved_package_sha256="d" * 64, release_lock=self.approval["h6_lock"],
            approved_lock_sha256="e" * 64, source_commit=source,
            approved_release_sha256="4" * 64, plan=self.approval["canary_plan"],
            plan_receipt=self.approval["canary_plan_receipt"],
            approved_plan_sha256=self.approval["approved_canary_plan_sha256"],
            approval=self.state / "ad132-approval.json", approval_sha256="5" * 64,
            container="a" * 64, receipt=self.state / "ad132-receipt.json",
            apply_output=self.state / "ad132.stdout", pending_path=self.state / "ad132.pending.json")
        self.approval["ad132_request"] = self.request
        self.ad132 = {"version": 1, "kind": "ad132_apply_confirmed",
            "package_sha256": "d" * 64, "source_commit": source, "lock_sha256": "e" * 64,
            "plan_sha256": self.approval["approved_canary_plan_sha256"],
            "release_sha256": "4" * 64, "pg_container_id": "a" * 64,
            "approval_sha256": "5" * 64, "postimage": {"public_temp": False},
            "installed_anchors": {"ancla_sintetica": "6" * 64}}
        self.write("ad132-receipt.json", ready.canonical(self.ad132))
        self.write("ad132.stdout", ready.clon_ad132_recibo.SUCCESS)
        self.write("ad132.pending.json", ready.canonical({"kind": "ad132_apply_pending"}))
        clone = {"propietario": "Codex-M", "estado": str(self.state), "contenedor": "vec-prueba",
            "puerto_pg": 55441, "puerto_web": 8445, "commit": source,
            "pgdata": self.approval["pg_volume"]["source"]}
        self.write("clon.json", ready.canonical(clone))
        entries = [{"path": "sql/" + str(i) + ".sql", "sha256": "7" * 64} for i in range(62)]
        self.plan = {"source_ref": source, "approved_sql_ref": source,
            "plan_family": ready.clon_sql.H6_PACKAGE_FAMILY, "file_count": 62,
            "plan_sha": "8" * 64, "inventory_sha": "9" * 64, "package_sha": "d" * 64,
            "lock_sha": "e" * 64, "list_sha": "0" * 64, "release_sha": "4" * 64,
            "estado_h1_sha": "f" * 64, "entries": entries}
        self.journal = {"version": 2, "run_id": str(uuid.uuid4()),
            "identidad_clon": "c" * 64, "revisions": [], "pending": None,
            "source_commit": source, **{k: v for k, v in self.plan.items()},
            "phase": "awaiting_ad132", "installed": [{"position": i + 1, **entry,
                "confirmation": "psql_exit0_observed", "confirmed_at": "2026-09-30T12:00:00+00:00"}
                for i, entry in enumerate(entries)]}
        self.save_journal()
        self.preflight = self.patch(ready.clon_sql, "preflight_h6_package", Mock(return_value=(self.plan, [])))
        self.live_original = ready.live_container
        self.live = self.patch(ready, "live_container", Mock())
        self.revalidate = self.patch(ready.clon_ad132_recibo, "revalidar", Mock(return_value=self.ad132))
        self.subprocess = self.patch(ready.subprocess, "run", Mock(side_effect=AssertionError("Docker/red prohibidos")))

    def patch(self, owner, name, value):
        patcher = patch.object(owner, name, value)
        self.addCleanup(patcher.stop)
        return patcher.start()

    def write(self, relative, data):
        path = self.state / relative
        path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        path.write_bytes(data)
        path.chmod(0o600)
        return path

    def save_journal(self):
        self.journal["journal_sha"] = ready.clon_sql.record_hash(self.journal)
        self.write("sql-journal.json", ready.canonical(self.journal))

    def approval_json(self):
        value = copy.deepcopy(self.approval)
        value["ad132_request"] = asdict(self.request)
        for key in ready.PATHS:
            value[key] = str(value[key])
        for key in ready.REQUEST_PATHS:
            value["ad132_request"][key] = str(value["ad132_request"][key])
        return value

    def test_pre_ad132_is_read_only_and_does_not_require_canary_or_ad132_receipts(self):
        self.request.receipt.unlink()
        self.approval["canary_plan"].unlink()
        before = (self.state / "sql-journal.json").read_bytes()
        value = ready.validate_pre_ad132(self.state, self.approval, live=False)
        self.assertEqual(value["file_count"], 62)
        self.assertEqual(before, (self.state / "sql-journal.json").read_bytes())
        self.assertFalse((self.state / "DB_READY.json").exists())
        self.revalidate.assert_not_called()
        self.live.assert_not_called()

    def test_complete_and_recover_same_seal_preserve_ad132_pending(self):
        pending = self.request.pending_path.read_bytes()
        value = ready.complete_h6(self.state, self.approval)
        data = (self.state / "DB_READY.json").read_bytes()
        self.assertEqual(value["version"], 2)
        self.assertEqual(value["sql_instaladas"], 62)
        self.assertEqual(value["kind"], "h6_db_ready")
        self.assertEqual(value["sql_journal_sha256"], ready.sha((self.state / "sql-journal.json").read_bytes()))
        self.assertEqual(value, ready.complete_h6(self.state, self.approval))
        self.assertEqual(data, ready.canonical(value))
        self.assertEqual(pending, self.request.pending_path.read_bytes())
        self.assertFalse((self.state / "READY.json").exists())
        self.assertFalse((self.state / "runtime-process.json").exists())
        self.assertGreater(self.revalidate.call_count, 0)
        self.subprocess.assert_not_called()

    def test_documentary_validation_never_queries_live(self):
        ready.complete_h6(self.state, self.approval)
        self.live.reset_mock()
        self.revalidate.reset_mock()
        self.assertEqual(ready.validate_h6_ready(self.state, self.approval, live=False)["file_count"], 62)
        self.live.assert_not_called()
        self.revalidate.assert_not_called()

    def test_crash_after_journal_confirmation_recovers_without_reapplying(self):
        with patch.object(ready, "publish", side_effect=OSError("fallo sintético")):
            with self.assertRaises(OSError):
                ready.complete_h6(self.state, self.approval)
        self.assertEqual(json.loads((self.state / "sql-journal.json").read_bytes())["phase"], "ad132_confirmed")
        self.assertFalse((self.state / "DB_READY.json").exists())
        journal = (self.state / "sql-journal.json").read_bytes()
        ready.complete_h6(self.state, self.approval)
        self.assertEqual(journal, (self.state / "sql-journal.json").read_bytes())
        self.subprocess.assert_not_called()

    def test_sql_pending_or_uncertain_confirmation_always_blocks(self):
        self.journal["pending"] = {"position": 63}
        self.save_journal()
        with self.assertRaises(ready.clon_sql.Refused):
            ready.complete_h6(self.state, self.approval)
        self.journal["pending"] = None
        self.save_journal()
        self.write(".sql-confirming", b"incertidumbre\n")
        with self.assertRaisesRegex(ready.Refused, "confirmation_pending"):
            ready.complete_h6(self.state, self.approval)
        self.assertFalse((self.state / "DB_READY.json").exists())

    def test_missing_ad132_receipt_with_pending_fails_closed(self):
        self.request.receipt.unlink()
        with self.assertRaises(FileNotFoundError):
            ready.complete_h6(self.state, self.approval)
        self.assertFalse((self.state / "DB_READY.json").exists())
        self.assertEqual(json.loads((self.state / "sql-journal.json").read_bytes())["phase"], "awaiting_ad132")

    def test_live_ad132_divergence_prevents_journal_confirmation(self):
        self.revalidate.side_effect = ready.clon_ad132_recibo.Refused("postimagen sintética divergente")
        with self.assertRaises(ready.clon_ad132_recibo.Refused):
            ready.complete_h6(self.state, self.approval)
        self.assertFalse((self.state / "DB_READY.json").exists())
        self.assertEqual(json.loads((self.state / "sql-journal.json").read_bytes())["phase"], "awaiting_ad132")

    def test_legacy_or_forged_ready_never_overwritten(self):
        data = ready.canonical({"version": 1, "sql_instaladas": 62})
        self.write("DB_READY.json", data)
        with self.assertRaisesRegex(ready.Refused, "ready_before_confirmation"):
            ready.complete_h6(self.state, self.approval)
        self.assertEqual(data, (self.state / "DB_READY.json").read_bytes())

    def test_changed_binary_ca_or_config_is_rejected(self):
        ready.complete_h6(self.state, self.approval)
        for relative in ("bin/vec-server", "material/pg-ca.crt", "runtime-interno/runtime-config.json"):
            with self.subTest(relative=relative):
                path = self.state / relative
                original = path.read_bytes()
                path.write_bytes(b"CAMBIO SINTETICO\n")
                with self.assertRaisesRegex(ready.Refused, "approval_hash_mismatch"):
                    ready.validate_h6_ready(self.state, self.approval, live=False)
                path.write_bytes(original)

    def test_run_id_or_clone_target_change_invalidates_ready(self):
        ready.complete_h6(self.state, self.approval)
        self.journal = json.loads((self.state / "sql-journal.json").read_bytes())
        self.journal["run_id"] = str(uuid.uuid4())
        self.save_journal()
        with self.assertRaisesRegex(ready.Refused, "db_ready_stale"):
            ready.validate_h6_ready(self.state, self.approval, live=False)

    def test_load_approval_converts_paths_and_request_without_embedded_secrets(self):
        value = self.approval_json()
        path = self.write("external-approval.json", ready.canonical(value))
        loaded = ready.load_approval(path)
        self.assertEqual(loaded, self.approval)
        value["PGPASSWORD"] = "SINTETICO_NO_ACEPTADO"
        path.write_bytes(ready.canonical(value))
        with self.assertRaisesRegex(ready.Refused, "unknown"):
            ready.load_approval(path)

    def test_approval_duplicate_json_or_noncanonical_or_public_denied(self):
        path = self.write("external-approval.json", b'{"source_commit":"x","source_commit":"y"}\n')
        with self.assertRaisesRegex(ready.Refused, "duplicate_json_key"):
            ready.load_approval(path)
        path.write_bytes(json.dumps(self.approval_json(), indent=2).encode())
        with self.assertRaisesRegex(ready.Refused, "not_canonical"):
            ready.load_approval(path)
        path.write_bytes(ready.canonical(self.approval_json()))
        path.chmod(0o644)
        with self.assertRaisesRegex(ready.Refused, "unsafe_file"):
            ready.load_approval(path)

    def test_symlinks_and_hardlinks_in_evidence_rejected(self):
        path = self.state / "sql-journal.json"
        saved = path.read_bytes()
        target = self.write("other.json", saved)
        path.unlink()
        path.symlink_to(target)
        with self.assertRaises(OSError):
            ready.validate_pre_ad132(self.state, self.approval, live=False)
        path.unlink()
        os.link(target, path)
        with self.assertRaisesRegex(ready.Refused, "unsafe_file"):
            ready.validate_pre_ad132(self.state, self.approval, live=False)

    def test_no_old_confirmation_or_incomplete62_adopted(self):
        self.journal["installed"].pop()
        self.save_journal()
        with self.assertRaises(ready.clon_sql.Refused):
            ready.validate_pre_ad132(self.state, self.approval, live=False)
        self.journal["installed"].append({"position": 62, **self.plan["entries"][-1],
            "confirmation": "commit_returned_and_postcheck", "confirmed_at": "2026-09-30T12:00:00+00:00"})
        self.save_journal()
        with self.assertRaises(ready.clon_sql.Refused):
            ready.validate_pre_ad132(self.state, self.approval, live=False)

    def test_postimage_without_stdout_confirmation_denied(self):
        self.request.apply_output.write_bytes(b"postimagen compatible sin exit0\n")
        with self.assertRaisesRegex(ready.Refused, "exit0_evidence_missing"):
            ready.complete_h6(self.state, self.approval)
        self.revalidate.assert_not_called()

    def test_canary_receipt_other_pg_denied_even_with_new_hash(self):
        path = self.approval["canary_plan_receipt"]
        receipt = json.loads(path.read_bytes())
        receipt["pg_container_id"] = "7" * 64
        data = ready.canonical(receipt)
        path.write_bytes(data)
        self.approval["approved_canary_receipt_sha256"] = ready.sha(data)
        with self.assertRaisesRegex(ready.Refused, "canary_receipt_mismatch"):
            ready.complete_h6(self.state, self.approval)

    def test_publication_is_private_synced_and_exclusive(self):
        with ready.directory(self.state) as descriptor:
            original_fsync = ready.os.fsync
            with patch.object(ready.os, "fsync", wraps=original_fsync) as sync:
                ready.publish(descriptor, b"SELLO SINTETICO\n")
                self.assertEqual(sync.call_count, 2)
            with self.assertRaises(FileExistsError):
                ready.publish(descriptor, b"OTRO SELLO\n")
        path = self.state / "DB_READY.json"
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        self.assertEqual(path.stat().st_nlink, 1)
        self.assertEqual(path.read_bytes(), b"SELLO SINTETICO\n")
        self.assertFalse(list(self.state.glob(".h6-ready-*")))

    def test_no_live_false_option_for_complete(self):
        with self.assertRaises(TypeError):
            ready.complete_h6(self.state, self.approval, live=False)

    def container_fixture(self):
        volume = self.state.stat()
        self.approval["pg_volume"].update(dev=volume.st_dev, ino=volume.st_ino)
        return {"Id": "a" * 64, "Image": self.approval["pg_image_id"],
            "Name": "/vec-prueba", "Config": {"Image": "postgres:18.4", "Labels": {
                "vec.recorridos.owner": "Codex-M", "vec.recorridos.state": str(self.state)}},
            "State": {"Running": True}, "NetworkSettings": {"Ports": {"5432/tcp": [
                {"HostIp": "127.0.0.1", "HostPort": "55441"}]}}, "Mounts": [{"Type": "bind",
                "Destination": "/var/lib/postgresql", "Source": self.approval["pg_volume"]["source"],
                "RW": True}]}

    def test_live_container_uses_exact_id_image_owner_state_and_mount(self):
        fixture = self.container_fixture()
        # Sólo inspección de una fixture local; no invocar Docker ni /dev/shm.
        live_impl = self.live_original
        original_directory = ready.directory
        with patch.object(ready, "docker_record", return_value=fixture), patch.object(
                ready, "directory", side_effect=lambda _: original_directory(self.state)):
            live_impl(self.state, self.approval)
            for field in ("Id", "Image", "Name"):
                original = fixture[field]
                fixture[field] = "AJENO_SINTETICO"
                with self.subTest(field=field), self.assertRaisesRegex(ready.Refused, "container_mismatch"):
                    live_impl(self.state, self.approval)
                fixture[field] = original
            fixture["Mounts"][0]["Source"] = "/dev/shm/vec-recorridos-ajeno"
            with self.assertRaisesRegex(ready.Refused, "volume_mismatch"):
                live_impl(self.state, self.approval)

    def test_recreated_volume_and_non_loopback_port_fail_closed(self):
        fixture = self.container_fixture()
        original_directory = ready.directory
        with patch.object(ready, "docker_record", return_value=fixture), patch.object(
                ready, "directory", side_effect=lambda _: original_directory(self.state)):
            self.approval["pg_volume"]["ino"] += 1
            with self.assertRaisesRegex(ready.Refused, "volume_replaced"):
                self.live_original(self.state, self.approval)
            fixture["NetworkSettings"]["Ports"]["5432/tcp"][0]["HostIp"] = "0.0.0.0"
            with self.assertRaisesRegex(ready.Refused, "ports_mismatch"):
                self.live_original(self.state, self.approval)

    def test_preparation_cannot_adopt_old_application_marker(self):
        self.write("READY.json", ready.canonical({"legacy": True}))
        with self.assertRaisesRegex(ready.Refused, "application_or_ready_exists"):
            ready.validate_pre_ad132(self.state, self.approval, live=False)

    def test_external_approval_missing_binding_rejected(self):
        value = self.approval_json()
        for key in ("approved_package_sha256", "binary_sha256", "projection_manifest_sha256"):
            with self.subTest(key=key):
                approval = dict(value)
                approval.pop(key)
                with self.assertRaisesRegex(ready.Refused, "external_approval_missing"):
                    ready.validate_approval(approval)

    def test_pre_approval_does_not_invent_future_canary_pins(self):
        pre = {key: value for key, value in self.approval.items() if key in ready.PRE_APPROVAL_FIELDS}
        self.request.receipt.unlink()
        self.approval["canary_plan"].unlink()
        self.approval["canary_plan_receipt"].unlink()
        context = ready.validate_pre_ad132(self.state, pre, live=False)
        self.assertEqual(context["plan_sha"], self.plan["plan_sha"])
        with self.assertRaisesRegex(ready.Refused, "external_approval_missing"):
            ready.complete_h6(self.state, pre)


if __name__ == "__main__":
    unittest.main()
