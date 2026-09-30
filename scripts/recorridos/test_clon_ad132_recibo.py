"""Pruebas puras del adaptador; ningún doble accede a Docker, SQL o red."""

import copy
from dataclasses import replace
import io
import inspect
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import types
import unittest
from unittest.mock import Mock, patch

import clon_ad132_recibo as adapter


def canonical(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=False,
                       separators=(",", ":")) + "\n").encode()


class AdapterTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.folder = Path(self.temp.name)
        self.root = self.folder / "package"
        self.root.mkdir(mode=0o755)
        self.root.chmod(0o755)
        self.inventory = {"public_temp": False, "acl_sha256": "b" * 64,
                          "logins": [{"role": "vec_test", "temp": False,
                                      "active_sessions": 0}]}
        self.anchors = {"ct145_table": "c" * 64}
        self.approval = {"approved": True, "inventory": {
            "public_temp": True, "acl_sha256": "a" * 64}}
        self.approval_path = self.folder / "approval.json"
        self.approval_path.write_bytes(canonical(self.approval))
        self.approval_path.chmod(0o600)
        self.source = "2" * 40
        self.material = {adapter.HELPER_REL: b"# helper double fixture\n",
                         adapter.CLI_REL: b"# CLI double fixture\n",
                         "h6-sql-release.json": canonical({"version": 1})}
        archive = io.BytesIO()
        with tarfile.open(fileobj=archive, mode="w:gz") as tar:
            directories = {".", *(str(p) for name in self.material
                                   for p in Path(name).parents if str(p) != ".")}
            for name in sorted(directories):
                entry = tarfile.TarInfo(name)
                entry.type, entry.mode = tarfile.DIRTYPE, 0o755
                tar.addfile(entry)
            for name, data in self.material.items():
                entry = tarfile.TarInfo("./" + name)
                entry.size, entry.mode = len(data), 0o644
                tar.addfile(entry, io.BytesIO(data))
                file = self.root / name
                file.parent.mkdir(parents=True, exist_ok=True)
                for parent in file.parents:
                    if parent == self.root:
                        break
                    parent.chmod(0o755)
                file.write_bytes(data)
                file.chmod(0o644)
        package = self.folder / "package.tar.gz"
        package.write_bytes(archive.getvalue())
        self.lock_path = self.folder / "release.lock"
        self.lock = {"PAQUETE_SHA256": adapter.digest(archive.getvalue()),
                     "COMMIT": self.source,
                     "SQL_RELEASE_SHA256": adapter.digest(self.material["h6-sql-release.json"]),
                     "AD132_RECIBO_SHA256": adapter.digest(self.material[adapter.HELPER_REL]),
                     "AD132_CLI_SHA256": adapter.digest(self.material[adapter.CLI_REL])}
        self.lock_path.write_text("".join(f"{k} {v}\n" for k, v in self.lock.items()))
        self.lock_path.chmod(0o600)
        self.request = adapter.Request(
            package_tar=package, package_root=self.root,
            approved_package_sha256=self.lock["PAQUETE_SHA256"],
            release_lock=self.lock_path, approved_lock_sha256=adapter.digest(self.lock_path.read_bytes()),
            source_commit=self.source, approved_release_sha256=self.lock["SQL_RELEASE_SHA256"],
            plan=self.folder / "plan-conexiones.json",
            plan_receipt=self.folder / "plan-canonico-clon.json", approved_plan_sha256="5" * 64,
            approval=self.approval_path, approval_sha256=adapter.digest(self.approval_path.read_bytes()),
            container="a" * 64, receipt=self.folder / "ad132.json",
            apply_output=self.folder / "apply.stdout", pending_path=self.folder / "ad132.pending.json")
        self.common = {"version": 1, "kind": "ad132_apply_confirmed",
                       "package_sha256": self.request.approved_package_sha256,
                       "source_commit": self.source,
                       "release_sha256": self.request.approved_release_sha256,
                       "lock_sha256": self.request.approved_lock_sha256,
                       "plan_sha256": self.request.approved_plan_sha256,
                       "plan_receipt_sha256": "6" * 64,
                       "approval_sha256": self.request.approval_sha256,
                       "pg_container_id": self.request.container,
                       "cli_sha256": self.lock["AD132_CLI_SHA256"],
                       "apply_stdout_sha256": adapter.digest(adapter.SUCCESS)}
        self.mod = types.SimpleNamespace(
            current_inventory=Mock(side_effect=lambda *_: copy.deepcopy(self.inventory)),
            current_anchors=Mock(side_effect=lambda *_: copy.deepcopy(self.anchors)),
            postimage_compatible=lambda before, after: (
                before.get("public_temp") is True and after.get("public_temp") is False
                and before.get("acl_sha256") != after.get("acl_sha256")))
        self.helper = types.ModuleType("synthetic_helper")
        self.helper.SUCCESS = adapter.SUCCESS
        self.helper.FIELDS = set(self.common) | {"postimage", "installed_anchors"}
        self.helper.context = Mock(side_effect=self.context)
        self.helper.canonical = canonical
        self.helper.json_object = json.loads
        self.helper.read_regular = lambda path, *_args, **_kw: path.read_bytes()
        self.helper.stable_inventory = lambda inventory: {
            **inventory, "logins": [{k: v for k, v in row.items() if k != "active_sessions"}
                                    for row in inventory["logins"]]}
        self.helper.atomic_receipt = self.atomic_receipt
        # La carga del adaptador sí verifica bytes/modos/tar/lock; únicamente las
        # funciones D y las consultas fijas son dobles de contrato sin secretos.
        self.module_patch = patch.object(adapter.types, "ModuleType", return_value=self.helper)
        self.module_patch.start()
        self.addCleanup(self.module_patch.stop)
        self.calls = []
        self.runner = Mock(side_effect=self.run_cli)
        self.runner_patch = patch.object(adapter.subprocess, "run", self.runner)
        self.runner_patch.start()
        self.addCleanup(self.runner_patch.stop)

    def context(self, args):
        if adapter.digest(args.approval.read_bytes()) != args.approval_sha256:
            raise adapter.Refused("approval changed")
        self.assertEqual(args.engine, "docker")
        return (self.mod, {"anchors_expected": {"ct145_table": "c" * 64}},
                self.approval, copy.deepcopy(self.common))

    def atomic_receipt(self, path, data):
        with path.open("xb") as output:
            path.chmod(0o600)
            output.write(data)
            output.flush()
            os.fsync(output.fileno())
        directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
        if path == self.request.pending_path:
            self.calls.append("pending")

    def run_cli(self, command, **kwargs):
        self.assertTrue(self.request.pending_path.exists())
        self.assertEqual(json.loads(self.request.pending_path.read_bytes())["context"], self.common)
        self.calls.append("cli")
        self.assertEqual(command[:4], ["/usr/bin/python3", "-I", "-B", "-S"])
        self.assertEqual(command[5:8], ["apply", "--engine", "docker"])
        self.assertNotIn("DOCKER_HOST", kwargs["env"])
        self.assertEqual(kwargs["env"]["PYTHONDONTWRITEBYTECODE"], "1")
        return subprocess.CompletedProcess(command, 0, adapter.SUCCESS, b"")

    def apply(self):
        return adapter.apply_and_confirm(self.request)

    def complete_canary_pair(self):
        for path in (self.request.plan, self.request.plan_receipt):
            path.write_bytes(b"{}\n")
            path.chmod(0o600)

    def test_complete_canary_pair_with_pending_blocks_all_entrypoints(self):
        self.complete_canary_pair()
        pending = self.folder / adapter.CANARY_PENDING
        pending.write_bytes(b"{}\n")
        pending.chmod(0o600)
        for operation in (adapter.load_adapter, adapter.apply_and_confirm, adapter.revalidar):
            with self.subTest(operation=operation.__name__), self.assertRaises(adapter.Refused):
                operation(self.request)
        self.assertEqual(pending.read_bytes(), b"{}\n")
        self.assertTrue(self.request.plan.is_file())
        self.assertTrue(self.request.plan_receipt.is_file())
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()
        self.mod.current_inventory.assert_not_called()
        self.mod.current_anchors.assert_not_called()
        self.assertFalse(self.request.pending_path.exists())

    def test_canary_pending_link_directory_or_fifo_also_blocks(self):
        self.complete_canary_pair()
        pending = self.folder / adapter.CANARY_PENDING
        for kind in ("broken_link", "directory", "fifo"):
            with self.subTest(kind=kind):
                if kind == "broken_link":
                    pending.symlink_to(self.folder / "missing")
                elif kind == "directory":
                    pending.mkdir(mode=0o700)
                else:
                    os.mkfifo(pending, 0o600)
                for operation in (adapter.apply_and_confirm, adapter.revalidar):
                    with self.assertRaises(adapter.Refused):
                        operation(self.request)
                pending.lstat()  # El rechazo conserva también tipos inválidos.
                if kind == "directory":
                    pending.rmdir()
                else:
                    pending.unlink()
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()

    def test_canary_pending_created_after_ad132_pending_prevents_cli(self):
        self.complete_canary_pair()
        original = self.helper.atomic_receipt
        pending = self.folder / adapter.CANARY_PENDING
        def write(path, data):
            original(path, data)
            if path == self.request.pending_path:
                pending.write_bytes(b"{}\n")
                pending.chmod(0o600)
        self.helper.atomic_receipt = write
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertTrue(pending.is_file())
        self.assertTrue(self.request.pending_path.is_file())
        self.assertFalse(self.request.receipt.exists())
        self.runner.assert_not_called()

    def test_completed_ad132_receipt_with_canary_pending_cannot_revalidate(self):
        self.complete_canary_pair()
        self.apply()
        pending = self.folder / adapter.CANARY_PENDING
        pending.write_bytes(b"{}\n")
        pending.chmod(0o600)
        self.helper.context.reset_mock()
        self.mod.current_inventory.reset_mock()
        self.mod.current_anchors.reset_mock()
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.assertTrue(pending.is_file())
        self.assertTrue(self.request.pending_path.is_file())
        self.helper.context.assert_not_called()
        self.mod.current_inventory.assert_not_called()
        self.mod.current_anchors.assert_not_called()
        self.assertEqual(self.runner.call_count, 1)

    def test_unrelated_canary_pending_does_not_block_committed_directory(self):
        self.complete_canary_pair()
        unrelated = self.folder / "unrelated"
        unrelated.mkdir(mode=0o700)
        pending = unrelated / adapter.CANARY_PENDING
        pending.write_bytes(b"{}\n")
        pending.chmod(0o600)
        receipt = self.apply()
        self.assertEqual(adapter.revalidar(self.request), receipt)
        self.assertTrue(pending.is_file())
        self.assertTrue(self.request.pending_path.is_file())
        self.assertEqual(self.runner.call_count, 1)

    def test_noncanonical_or_separate_plan_paths_cannot_bypass_canary_pending(self):
        self.complete_canary_pair()
        pending = self.folder / adapter.CANARY_PENDING
        pending.write_bytes(b"{}\n")
        pending.chmod(0o600)
        unrelated = self.folder / "unrelated"
        unrelated.mkdir(mode=0o700)
        cases = ({"plan": self.folder / "other-plan.json"},
                 {"plan_receipt": self.folder / "other-receipt.json"},
                 {"plan": unrelated / self.request.plan.name},
                 {"plan_receipt": unrelated / self.request.plan_receipt.name})
        for fields in cases:
            request = replace(self.request, **fields)
            for operation in (adapter.apply_and_confirm, adapter.revalidar):
                with self.subTest(fields=fields, operation=operation.__name__):
                    with self.assertRaises(adapter.Refused):
                        operation(request)
        self.assertTrue(pending.is_file())
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()

    def test_success_is_d_format_and_revalidation_is_read_only(self):
        receipt = self.apply()
        self.assertEqual(self.calls, ["pending", "cli"])
        self.assertEqual(set(receipt), self.helper.FIELDS)
        self.assertEqual(self.request.receipt.read_bytes(), canonical(receipt))
        self.assertEqual(self.request.apply_output.read_bytes(), adapter.SUCCESS)
        self.assertEqual(self.request.receipt.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.request.pending_path.stat().st_mode & 0o777, 0o600)
        self.assertNotIn("active_sessions", receipt["postimage"]["logins"][0])
        self.inventory["logins"][0]["active_sessions"] = 3
        self.assertEqual(adapter.revalidar(self.request), receipt)
        self.assertEqual(self.runner.call_count, 1)
        self.assertFalse(list(self.root.rglob("__pycache__")))

    def test_api_cannot_supply_fake_success_or_pending_callback(self):
        self.assertEqual(list(inspect.signature(adapter.apply_and_confirm).parameters), ["request"])
        with self.assertRaises(TypeError):
            adapter.apply_and_confirm(self.request, lambda _: True)
        with self.assertRaises(TypeError):
            adapter.apply_and_confirm(self.request, runner=lambda *_: subprocess.CompletedProcess(
                [], 0, adapter.SUCCESS, b""))
        self.runner.assert_not_called()
        self.assertFalse(self.request.pending_path.exists())
        self.assertFalse(self.request.receipt.exists())

    def test_pending_write_failure_prevents_cli_and_receipt(self):
        self.helper.atomic_receipt = Mock(side_effect=OSError("fixture write failure"))
        with self.assertRaises(OSError):
            self.apply()
        self.runner.assert_not_called()
        self.assertFalse(self.request.receipt.exists())

    def test_pending_fsync_error_stops_before_cli_and_blocks_retry(self):
        original = self.helper.atomic_receipt
        def fail(path, data):
            original(path, data)
            raise OSError("fixture fsync failure")
        self.helper.atomic_receipt = fail
        with self.assertRaises(OSError):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.runner.assert_not_called()
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertFalse(self.request.receipt.exists())

    def test_existing_pending_without_receipt_blocks_fake_success(self):
        self.request.pending_path.write_bytes(b"{}\n")
        self.request.pending_path.chmod(0o600)
        self.runner.return_value = subprocess.CompletedProcess([], 0, adapter.SUCCESS, b"")
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.runner.assert_not_called()
        self.assertFalse(self.request.receipt.exists())

    def test_changed_pending_before_cli_prevents_apply(self):
        original = self.helper.atomic_receipt
        def write(path, data):
            original(path, b"{}\n" if path == self.request.pending_path else data)
        self.helper.atomic_receipt = write
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.runner.assert_not_called()
        self.assertFalse(self.request.receipt.exists())

    def test_same_named_helper_altered_stops_before_import(self):
        (self.root / adapter.HELPER_REL).write_bytes(b"raise RuntimeError('must not execute')\n")
        with self.assertRaises(adapter.clon_sql.Refused):
            self.apply()
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()

    def test_extra_bytecode_cli_or_mode_change_rejected(self):
        for change in ("bytecode", "cli", "mode"):
            with self.subTest(change=change):
                cli = self.root / adapter.CLI_REL
                extra = self.root / "injected.pyc"
                if change == "bytecode":
                    extra.write_bytes(b"injected")
                elif change == "cli":
                    cli.write_bytes(b"altered")
                else:
                    cli.chmod(0o755)
                with self.assertRaises((adapter.Refused, adapter.clon_sql.Refused)):
                    self.apply()
                extra.unlink(missing_ok=True)
                cli.write_bytes(self.material[adapter.CLI_REL])
                cli.chmod(0o644)
        self.runner.assert_not_called()

    def test_group_writable_or_tar_mode_different_directories_stop_before_import(self):
        for directory in (self.root, self.root / "deploy",
                          self.root / "deploy/postgresql/autorizacion_atestada_v3"):
            for mode in (0o775, 0o700):
                with self.subTest(directory=directory, mode=mode):
                    directory.chmod(mode)
                    with self.assertRaises(adapter.Refused):
                        self.apply()
                    directory.chmod(0o755)
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()
        self.assertFalse(self.request.pending_path.exists())

    def test_group_writable_ancestor_stops_before_import(self):
        self.folder.chmod(0o770)
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.folder.chmod(0o700)
        self.runner.assert_not_called()
        self.helper.context.assert_not_called()

    def test_foreign_owned_directory_and_file_stop_before_import(self):
        original = Path.lstat
        for foreign in (self.root, self.root / "deploy",
                        self.root / adapter.CLI_REL):
            def fake_owner(path, *args, **kwargs):
                status = original(path, *args, **kwargs)
                if path == foreign:
                    values = list(status)
                    values[4] = os.getuid() + 1
                    return os.stat_result(values)
                return status
            with self.subTest(foreign=foreign), patch.object(Path, "lstat", fake_owner):
                with self.assertRaises(adapter.Refused):
                    self.apply()
        self.helper.context.assert_not_called()
        self.runner.assert_not_called()

    def test_package_altered_after_preflight_prevents_cli(self):
        original = self.helper.context.side_effect
        def change(args):
            result = original(args)
            (self.root / adapter.CLI_REL).write_bytes(b"altered after check\n")
            return result
        self.helper.context.side_effect = change
        with self.assertRaises(adapter.clon_sql.Refused):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.runner.assert_not_called()
        self.assertFalse(self.request.receipt.exists())

    def test_package_altered_during_cli_prevents_receipt(self):
        def change(command, **kwargs):
            result = self.run_cli(command, **kwargs)
            (self.root / adapter.CLI_REL).write_bytes(b"altered during CLI\n")
            return result
        self.runner.side_effect = change
        with self.assertRaises(adapter.clon_sql.Refused):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.assertFalse(self.request.receipt.exists())

    def test_externally_wrong_tar_lock_source_release_and_plan_rejected(self):
        for field in ("approved_package_sha256", "approved_lock_sha256", "source_commit",
                      "approved_release_sha256", "approved_plan_sha256"):
            request = replace(self.request, **{field: "f" * (40 if field == "source_commit" else 64)})
            with self.subTest(field=field), self.assertRaises((adapter.Refused, adapter.clon_sql.Refused)):
                adapter.apply_and_confirm(request)
        self.runner.assert_not_called()

    def test_bad_stdout_and_nonzero_never_confirm(self):
        for code, stdout in ((0, adapter.SUCCESS.rstrip()), (0, adapter.SUCCESS + b"extra\n"),
                             (1, adapter.SUCCESS), (0, b"postimage compatible\n")):
            self.runner.side_effect = None
            self.runner.return_value = subprocess.CompletedProcess([], code, stdout, b"")
            with self.subTest(code=code, stdout=stdout), self.assertRaises(adapter.Refused):
                self.apply()
            self.assertTrue(self.request.pending_path.exists())
            self.assertFalse(self.request.receipt.exists())
            self.assertFalse(self.request.apply_output.exists())
            self.request.pending_path.unlink()  # Escenarios sintéticos independientes.

    def test_timeout_never_confirms(self):
        self.runner.side_effect = subprocess.TimeoutExpired("fixture", 1)
        with self.assertRaises(subprocess.TimeoutExpired):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.assertFalse(self.request.receipt.exists())

    def test_approval_changed_during_pending_prevents_apply(self):
        original = self.helper.atomic_receipt
        def pending(path, data):
            original(path, data)
            if path == self.request.pending_path:
                self.approval_path.write_bytes(b"{}\n")
        self.helper.atomic_receipt = pending
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.runner.assert_not_called()

    def test_approval_changed_during_apply_prevents_confirmation(self):
        def run(command, **kwargs):
            self.approval_path.write_bytes(b"{}\n")
            return self.run_cli(command, **kwargs)
        self.runner.side_effect = run
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.assertFalse(self.request.receipt.exists())

    def test_evidence_created_during_pending_prevents_apply(self):
        original = self.helper.atomic_receipt
        def pending(path, data):
            original(path, data)
            if path == self.request.pending_path:
                self.request.apply_output.write_bytes(adapter.SUCCESS)
        self.helper.atomic_receipt = pending
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.runner.assert_not_called()

    def test_postimage_and_anchors_divergent_never_confirm(self):
        self.inventory["public_temp"] = True
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.request.pending_path.unlink()  # Otro escenario sintético.
        self.inventory["public_temp"] = False
        self.anchors["ct145_table"] = "d" * 64
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertFalse(self.request.receipt.exists())

    def test_receipt_foreign_drift_and_output_change_rejected(self):
        receipt = self.apply()
        other = {**receipt, "pg_container_id": "f" * 64}
        self.request.receipt.write_bytes(canonical(other))
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.request.receipt.write_bytes(canonical(receipt))
        self.inventory["acl_sha256"] = "d" * 64
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.inventory["acl_sha256"] = "b" * 64
        self.anchors["ct145_table"] = "d" * 64
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.anchors["ct145_table"] = "c" * 64
        self.request.apply_output.write_bytes(b"other")
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.assertEqual(self.runner.call_count, 1)

    def test_revalidation_requires_same_pending_evidence(self):
        self.apply()
        pending = self.request.pending_path.read_bytes()
        self.request.pending_path.unlink()
        with self.assertRaises(FileNotFoundError):
            adapter.revalidar(self.request)
        self.request.pending_path.write_bytes(pending.replace(b'"pg_container_id":"a',
                                                             b'"pg_container_id":"f'))
        self.request.pending_path.chmod(0o600)
        with self.assertRaises(adapter.Refused):
            adapter.revalidar(self.request)
        self.assertEqual(self.runner.call_count, 1)

    def test_compatible_postimage_without_receipt_is_not_apply(self):
        with self.assertRaises(FileNotFoundError):
            adapter.revalidar(self.request)
        self.runner.assert_not_called()
        self.mod.current_inventory.assert_not_called()
        self.assertFalse(self.request.pending_path.exists())

    def test_existing_evidence_blocks_reapply(self):
        self.apply()
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertEqual(self.runner.call_count, 1)

    def test_receipt_write_failure_leaves_pending_and_never_reapplies(self):
        original = self.helper.atomic_receipt
        def write(path, data):
            if path == self.request.receipt:
                raise OSError("fixture disk failure")
            return original(path, data)
        self.helper.atomic_receipt = write
        with self.assertRaises(OSError):
            self.apply()
        self.assertTrue(self.request.pending_path.exists())
        self.assertFalse(self.request.receipt.exists())
        self.assertTrue(self.request.apply_output.exists())
        with self.assertRaises(adapter.Refused):
            self.apply()
        self.assertEqual(self.runner.call_count, 1)

    def test_symlink_ancestor_denied(self):
        linked = self.folder / "linked"
        linked.symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(adapter.clon_sql.Refused):
            adapter.apply_and_confirm(replace(self.request, package_root=linked))
        self.runner.assert_not_called()

    def test_cli_does_not_expose_confirmation_or_engine_choice(self):
        with patch("sys.stderr", io.StringIO()), self.assertRaises(SystemExit):
            adapter.main(["confirmar"])
        with patch("sys.stderr", io.StringIO()), self.assertRaises(SystemExit):
            adapter.main(["apply", "--engine", "podman"])


if __name__ == "__main__":
    unittest.main()
