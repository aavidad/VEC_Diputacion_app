"""Offline boundary tests. Docker and approved helpers are replaced by a runner."""
from contextlib import ExitStack
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("clon_h6_canario", Path(__file__).with_name("clon_h6_canario.py"))
canary = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = canary
SPEC.loader.exec_module(canary)


class Fixture:
    def __init__(self, root):
        self.root = root
        self.state = root / "state"
        self.state.mkdir(mode=0o700)
        self.output = self.state / "canario-salida-fixture"
        self.output.mkdir(mode=0o700)
        self.paths = {}
        for name in canary.FILES:
            self.paths[name] = root / name
            self.paths[name].write_bytes((name + "\n").encode())
            self.paths[name].chmod(0o600)
        self.paths["arr"].write_bytes(b"#!/bin/bash\nexport VEC_PORTAL_PROCESO=interno\nexec /usr/local/bin/vec-server\n")
        self.paths["inspector"].chmod(0o700)
        self.paths["resolver"].chmod(0o700)
        for name in canary.TREES:
            self.paths[name] = root / name
            self.paths[name].mkdir(mode=0o700)
            (self.paths[name] / "fixture.json").write_bytes(b"{}\n")
            (self.paths[name] / "fixture.json").chmod(0o600)
        with tarfile.open(self.paths["package"], "w:gz") as archive:
            for name, member in (("inspector", "./h6-canario-bin.sh"), ("connections", "./h6-conexiones.py"),
                                 ("resolver", "./h6-pg-resolver")):
                data = self.paths[name].read_bytes()
                info = tarfile.TarInfo(member)
                info.size = len(data)
                archive.addfile(info, io.BytesIO(data))
            data = (self.paths["public_data"] / "fixture.json").read_bytes()
            info = tarfile.TarInfo("./data/fixture.json")
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        self.paths["lock"].write_text("COMMIT " + canary.SOURCE + "\nPAQUETE_SHA256 " +
                                      canary.sha(self.paths["package"].read_bytes()) + "\n")
        self.paths["guiones"].write_text("".join(canary.sha(self.paths[key].read_bytes()) + "  " + filename + "\n"
                                               for key, filename in canary.HELPERS.items()))
        self.approved = {key: canary.sha(self.paths[key].read_bytes()) for key in canary.FILES}
        self.config = canary.Config(self.state, self.output, "a" * 64, "sha256:" + "b" * 64,
                                   self.paths, self.approved)
        self.runner = FakeRunner(self)

    def refresh(self, name):
        self.approved[name] = canary.sha(self.paths[name].read_bytes())


class FakeRunner:
    def __init__(self, fixture):
        self.f = fixture
        self.calls = []
        self.cid = "c" * 64
        self.started = False
        self.after_start = lambda: None
        self.pg_mutate = lambda data: None
        self.canary_mutate = lambda data: None
        self.image = fixture.config.image
        self.plan = canary.canonical({"conexiones": [{"fuente": "fixture", "login": "synthetic_login", "rol": "synthetic_role"}],
                                      "huellas": {"fixture": "d" * 64}})

    def canary_inspect(self):
        config = self.f.config
        data = {"Id": self.cid, "Image": config.image,
                "Config": {"Image": config.image, "User": "10002:10002", "Entrypoint": ["/bin/bash"], "Cmd": ["/vec-arrancar.sh"]},
                "HostConfig": {"NetworkMode": "none", "ReadonlyRootfs": True, "Privileged": False,
                               "CapDrop": ["ALL"], "SecurityOpt": ["no-new-privileges"], "PidsLimit": 64,
                               "Memory": 512 * 1048576, "NanoCpus": 1000000000, "LogConfig": {"Type": "none"},
                               "Tmpfs": {"/tmp": "rw,nosuid,nodev,noexec,size=8m", "/dev/shm": "rw,nosuid,nodev,noexec,size=8m"},
                               "Ulimits": [{"Name": "fsize", "Soft": 131072, "Hard": 131072}]},
                "Mounts": [{"Type": "bind", "Source": str(p), "Destination": target, "RW": rw}
                           for p, target, rw in canary.mounts(config)],
                "State": {"Running": not self.started, "ExitCode": 0, "OOMKilled": False}}
        self.canary_mutate(data)
        return data

    def __call__(self, argv):
        self.calls.append(argv)
        if argv[:3] == ["docker", "image", "inspect"]:
            return json.dumps([{"Id": self.image}]).encode()
        if argv[:4] == ["docker", "inspect", "--type", "container"]:
            if argv[-1] == self.f.config.pgid:
                data = {"Id": self.f.config.pgid, "Image": "sha256:" + "e" * 64, "State": {"Running": True},
                        "Config": {"Labels": {"vec.recorridos.owner": "Codex-M", "vec.recorridos.state": str(self.f.state)}},
                        "HostConfig": {"NetworkMode": "none"}}
                self.pg_mutate(data)
                return json.dumps([data]).encode()
            return json.dumps([self.canary_inspect()]).encode()
        if argv[:2] == ["docker", "create"]:
            return self.cid.encode() + b"\n"
        if argv[:2] == ["docker", "start"]:
            self.started = True
            canary.write(self.f.output / "plan.json", self.plan)
            self.after_start()
            return b""
        if argv[:2] == ["docker", "rm"]:
            return self.cid.encode()
        if argv[0] == "bwrap":
            def bind_source(target):
                return Path(argv[argv.index(target) - 1])
            destination = bind_source("/destination")
            if "/helpers/h6_promover_plan.py" in argv:
                source = bind_source("/source")
                shutil.copyfile(source / "plan.json", destination / "plan-conexiones.json")
                (destination / "plan-conexiones.json").chmod(0o600)
                (source / "plan.json").unlink()
                return b"PLAN-CONEXIONES-CERRADO\n"
            plan = (destination / "plan-conexiones.json").read_bytes()
            config = self.f.config
            receipt = {"version": 1, "kind": "clon", "package_sha256": config.approved["package"],
                       "source_commit": canary.SOURCE, "pg_container_id": config.pgid, "plan_sha256": canary.sha(plan),
                       "material_inventory_sha256": canary.sha(canary.canonical(canary.decode(plan)["huellas"])),
                       "canary_image_id": config.image, "arranque_sha256": config.approved["arr"], "canary_output_sha256": canary.sha(plan)}
            canary.write(destination / "plan-canonico-clon.json", canary.canonical(receipt))
            return b"RECIBO-PLAN-CLON-OK\n"
        raise AssertionError("Unexpected command: " + argv[0])


class CanaryTests(unittest.TestCase):
    def setUp(self):
        self.stack = ExitStack()
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        # The double writes as the test user; no real ownership changes are made.
        self.stack.enter_context(patch.object(canary, "UID", os.getuid()))
        self.stack.enter_context(patch.object(canary, "GID", os.getgid()))
        self.f = Fixture(self.root)

    def tearDown(self):
        self.stack.close()

    def run_canary(self):
        return canary.canary(self.f.config, self.f.runner)

    def assert_rejected(self, *, no_helpers=True):
        with self.assertRaises((canary.Refused, OSError)):
            self.run_canary()
        self.assertFalse((self.f.state / "plan-conexiones.json").exists())
        self.assertFalse((self.f.state / "plan-canonico-clon.json").exists())
        if no_helpers:
            self.assertFalse(any(call[0] == "bwrap" for call in self.f.runner.calls))

    def test_success_has_fixed_offline_flags_paths_and_canonical_local_receipt(self):
        result = self.run_canary()
        self.assertEqual(result["canary_exit_code"], 0)
        create = next(c for c in self.f.runner.calls if c[:2] == ["docker", "create"])
        for flag in ("--pull=never", "--network=none", "--user=10002:10002", "--read-only", "--cap-drop=ALL",
                     "--security-opt=no-new-privileges", "--pids-limit=64", "--memory=512m", "--cpus=1", "--log-driver=none"):
            self.assertIn(flag, create)
        self.assertEqual(create[-2:], [self.f.config.image, "/vec-arrancar.sh"])
        binds = [create[i + 1] for i, arg in enumerate(create) if arg == "--mount"]
        self.assertEqual(sum("readonly" not in value for value in binds), 1)
        for destination in ("/usr/local/bin/vec-server", "/vec-material", "/vec-incorporacion", "/app/h6-public-data", "/h6-out"):
            self.assertTrue(any("dst=" + destination in value for value in binds))
        self.assertFalse(any("docker.sock" in value or "network=container" in value for value in create))
        helpers = [c for c in self.f.runner.calls if c[0] == "bwrap"]
        self.assertEqual(len(helpers), 2)
        self.assertIn("--unshare-all", helpers[0])
        self.assertIn("--clearenv", helpers[0])
        self.assertIn("clon", helpers[1])
        self.assertLess(self.f.runner.calls.index(["docker", "rm", "--force", self.f.runner.cid]), self.f.runner.calls.index(helpers[0]))
        receipt = canary.decode((self.f.state / "plan-canonico-clon.json").read_bytes())
        self.assertEqual(receipt["kind"], "clon")
        self.assertEqual(receipt["pg_container_id"], self.f.config.pgid)
        self.assertEqual(receipt["canary_image_id"], self.f.config.image)
        self.assertEqual(receipt["arranque_sha256"], self.f.approved["arr"])
        self.assertEqual(receipt["package_sha256"], self.f.approved["package"])
        self.assertFalse(self.f.output.exists())

    def test_altered_helper_refused_before_any_docker_command(self):
        self.f.paths["promoter"].write_bytes(b"changed\n")
        self.assert_rejected()
        self.assertEqual(self.f.runner.calls, [])

    def test_self_consistent_helper_is_not_an_external_approval(self):
        self.f.paths["promoter"].write_bytes(b"changed\n")
        self.f.refresh("promoter")
        self.assert_rejected()

    def test_tar_or_lock_mismatch_refused_without_commands(self):
        self.f.approved["package"] = "9" * 64
        self.assert_rejected()
        self.assertEqual(self.f.runner.calls, [])

    def test_image_id_divergence_refused_before_create(self):
        self.f.runner.image = "sha256:" + "9" * 64
        self.assert_rejected()
        self.assertFalse(any(c[:2] == ["docker", "create"] for c in self.f.runner.calls))

    def test_pg_identity_network_or_owner_divergence(self):
        for change in (lambda d: d.update(Id="9" * 64), lambda d: d["HostConfig"].update(NetworkMode="bridge"),
                       lambda d: d["Config"]["Labels"].update({"vec.recorridos.owner": "other"})):
            self.f.runner.pg_mutate = change
            self.assert_rejected()

    def test_canary_container_network_or_mount_drift_cleans_up_without_start(self):
        for change in (lambda d: d["HostConfig"].update(NetworkMode="bridge"),
                       lambda d: d["Mounts"].append({"Type": "bind", "Source": "/var/run/docker.sock", "Destination": "/docker.sock", "RW": True}),
                       lambda d: d["Config"].update(User="root")):
            self.f.runner.canary_mutate = change
            self.assert_rejected()
        self.assertFalse(any(c[:2] == ["docker", "start"] for c in self.f.runner.calls))
        self.assertIn(["docker", "rm", "--force", self.f.runner.cid], self.f.runner.calls)

    def test_arr_changed_during_execution_rejects_before_promotion(self):
        self.f.runner.after_start = lambda: self.f.paths["arr"].write_bytes(b"changed\n")
        self.assert_rejected()

    def test_pg_or_image_changed_after_execution_rejects_before_promotion(self):
        self.f.runner.after_start = lambda: setattr(self.f.runner, "image", "sha256:" + "9" * 64)
        self.assert_rejected()

    def test_pg_replaced_after_execution_rejects_before_helpers(self):
        def change():
            self.f.runner.pg_mutate = lambda d: d.update(Image="sha256:" + "9" * 64)
        self.f.runner.after_start = change
        self.assert_rejected()

    def test_final_runtime_uid_cannot_read_host_private_input(self):
        metadata = (1, 2, 1000, 1000, 0o100600, 1, 10, 1, 1)
        with patch.object(canary, "UID", 10002), patch.object(canary, "GID", 10002), self.assertRaises(canary.Refused):
            canary.runtime_access(metadata)

    def test_wrong_package_catalog_is_refused_before_docker(self):
        (self.f.paths["public_data"] / "fixture.json").write_bytes(b'{"other":true}\n')
        self.assert_rejected()
        self.assertEqual(self.f.runner.calls, [])

    def test_extra_missing_or_symlink_output_rejected_before_helpers(self):
        for variant in ("extra", "missing", "symlink"):
            with self.subTest(variant=variant):
                for path in self.f.output.iterdir():
                    path.unlink()
                def change():
                    plan = self.f.output / "plan.json"
                    if variant == "extra":
                        canary.write(self.f.output / "extra", b"ignored")
                    elif variant == "missing":
                        plan.unlink()
                    else:
                        plan.unlink()
                        plan.symlink_to(self.f.paths["arr"])
                self.f.runner.after_start = change
                self.assert_rejected()

    def test_output_path_owner_or_mode_is_fail_closed(self):
        self.f.output.chmod(0o755)
        self.assert_rejected()
        self.f.output.chmod(0o700)
        with patch.object(canary, "UID", os.getuid() + 1):
            self.assert_rejected()
        actual = self.f.output
        alternate = self.f.state / "canario-salida-link"
        alternate.symlink_to(actual, target_is_directory=True)
        self.f.config = canary.Config(self.f.state, alternate, self.f.config.pgid, self.f.config.image,
                                      self.f.paths, self.f.approved)
        self.assert_rejected()

    def test_input_symlink_hardlink_and_special_entry_refused(self):
        original = self.f.paths["reference"]
        original.unlink()
        original.symlink_to(self.f.paths["arr"])
        self.assert_rejected()
        original.unlink()
        os.link(self.f.paths["arr"], original)
        self.assert_rejected()

    def test_nonzero_or_oom_canary_never_promotes(self):
        self.f.runner.canary_mutate = lambda d: d["State"].update(ExitCode=1) if self.f.runner.started else None
        self.assert_rejected()

    def test_existing_canonical_receipt_never_overwrites_or_runs(self):
        receipt = self.f.state / "plan-canonico-clon.json"
        canary.write(receipt, b"historical\n")
        with self.assertRaises(canary.Refused):
            self.run_canary()
        self.assertEqual(receipt.read_bytes(), b"historical\n")
        self.assertEqual(self.f.runner.calls, [])

    def test_helper_failure_leaves_no_partial_canonical_plan(self):
        original = self.f.runner
        def failing(argv):
            if argv[0] == "bwrap" and "/helpers/h6_recibo_plan.py" in argv:
                raise canary.Refused("synthetic_failure")
            return original(argv)
        self.f.runner = failing
        with self.assertRaises(canary.Refused):
            self.run_canary()
        self.assertFalse((self.f.state / "plan-conexiones.json").exists())
        self.assertFalse((self.f.state / "plan-canonico-clon.json").exists())

    def test_divergent_helper_receipt_never_publishes_canonical_files(self):
        original = self.f.runner
        def altered(argv):
            value = original(argv)
            if argv[0] == "bwrap" and "/helpers/h6_recibo_plan.py" in argv:
                destination = Path(argv[argv.index("/destination") - 1])
                path = destination / "plan-canonico-clon.json"
                receipt = canary.decode(path.read_bytes())
                receipt["pg_container_id"] = "9" * 64
                path.write_bytes(canary.canonical(receipt))
            return value
        self.f.runner = altered
        with self.assertRaises(canary.Refused):
            self.run_canary()
        self.assertFalse((self.f.state / "plan-conexiones.json").exists())
        self.assertFalse((self.f.state / "plan-canonico-clon.json").exists())

    def test_docker_client_ignores_ambient_context_and_uses_private_config(self):
        commands = []
        def subprocess_double(argv, **kwargs):
            commands.append(argv)
            self.assertEqual(argv[2], "--host=unix:///var/run/docker.sock")
            config = Path(argv[1].removeprefix("--config="))
            self.assertEqual(config.stat().st_mode & 0o777, 0o700)
            self.assertEqual(list(config.iterdir()), [])
            self.assertEqual(kwargs["env"], {"PATH": "/usr/bin:/bin", "HOME": "/tmp", "LANG": "C.UTF-8"})
            return SimpleNamespace(returncode=0, stdout=b"ok")
        with patch.dict(os.environ, {"DOCKER_CONTEXT": "foreign", "DOCKER_HOST": "tcp://foreign.invalid:2376",
                                    "DOCKER_CONFIG": str(self.root / "foreign"), "HOME": str(self.root / "foreign")}), \
             patch.object(canary.subprocess, "run", side_effect=subprocess_double):
            self.assertEqual(canary.run(["docker", "inspect", self.f.config.pgid]), b"ok")
            self.assertEqual(canary.run(["docker", "image", "inspect", self.f.config.image]), b"ok")
        self.assertNotEqual(commands[0][1], commands[1][1])
        self.assertFalse(Path(commands[0][1].removeprefix("--config=")).exists())

    def test_ancestor_group_write_and_foreign_file_owner_are_refused(self):
        self.root.chmod(0o770)
        self.assert_rejected()
        self.root.chmod(0o700)
        self.f.paths["reference"].chmod(0o620)
        self.assert_rejected()
        self.f.paths["reference"].chmod(0o600)
        info = self.f.paths["reference"].stat()
        foreign = SimpleNamespace(st_mode=info.st_mode, st_uid=999999, st_nlink=1, st_size=info.st_size)
        with canary.directory(self.root) as fd, patch.object(canary.os, "fstat", return_value=foreign), self.assertRaises(canary.Refused):
            canary.read_at(fd, "reference")
        foreign_directory = SimpleNamespace(st_mode=0o040700, st_uid=999999)
        with self.assertRaises(canary.Refused):
            canary.trusted_directory(foreign_directory, self.root)

    def inspector_ownership(self, paths):
        original = canary.os.fstat
        def owned(fd):
            info = original(fd)
            for path in paths:
                if path.exists() and (info.st_dev, info.st_ino) == (path.stat().st_dev, path.stat().st_ino):
                    fields = ("st_dev", "st_ino", "st_uid", "st_gid", "st_mode", "st_nlink",
                              "st_size", "st_mtime_ns", "st_ctime_ns")
                    result = {key: getattr(info, key) for key in fields}
                    result.update(st_uid=10002, st_gid=10002)
                    return SimpleNamespace(**result)
            return info
        return patch.object(canary.os, "fstat", side_effect=owned)

    def test_inspector_owned_input_ancestor_is_refused_before_docker(self):
        parent = self.root / "inspector-parent"
        parent.mkdir(mode=0o755)
        reference = parent / "reference"
        reference.write_bytes(self.f.paths["reference"].read_bytes())
        reference.chmod(0o600)
        self.f.paths["reference"] = reference
        with patch.object(canary, "UID", 10002), self.inspector_ownership([parent]):
            self.assert_rejected()
        self.assertEqual(self.f.runner.calls, [])

    def test_inspector_owned_input_file_and_tree_are_refused(self):
        for path in (self.f.paths["reference"], self.f.paths["material"]):
            with self.subTest(path=path.name), patch.object(canary, "UID", 10002), self.inspector_ownership([path]):
                self.assert_rejected()
        self.assertEqual(self.f.runner.calls, [])

    def test_inspector_output_and_leaf_are_accepted_only_with_output_role(self):
        plan = self.f.output / "plan.json"
        canary.write(plan, self.f.runner.plan)
        with patch.object(canary, "UID", 10002), patch.object(canary, "GID", 10002), \
             self.inspector_ownership([self.f.output, plan]):
            self.assertEqual(canary.output_identity(self.f.config, empty=False)[2:4], (10002, 10002))
            with self.assertRaisesRegex(canary.Refused, "directory_owner"), canary.directory(self.f.output):
                pass
            with canary.directory(self.f.output, path_role="output") as fd:
                with self.assertRaisesRegex(canary.Refused, "untrusted_file_writer"):
                    canary.read_at(fd, "plan.json")
                data, _, metadata = canary.read_at(fd, "plan.json", path_role="output", owner=10002, modes=(0o600,))
                self.assertEqual(data, self.f.runner.plan)
                self.assertEqual(metadata[2:4], (10002, 10002))

    def test_output_role_does_not_trust_inspector_owned_ancestor(self):
        with patch.object(canary, "UID", 10002), self.inspector_ownership([self.f.state]), \
             self.assertRaisesRegex(canary.Refused, "directory_owner"):
            canary.output_identity(self.f.config, empty=True)

    def test_output_role_requires_private_operator_parent(self):
        self.f.state.chmod(0o755)
        with self.assertRaisesRegex(canary.Refused, "output_parent_private"):
            canary.output_identity(self.f.config, empty=True)

    def test_full_canary_accepts_inspector_output_with_operator_owned_inputs(self):
        # Inputs are made readable to the inspector; ownership stays with the
        # operator. This checks output roles without emulating the archive bridge.
        for name in ("inspector", "connections", "resolver", "arr", "reference", "ca"):
            self.f.paths[name].chmod(0o755 if name in ("inspector", "resolver") else 0o644)
        for name in canary.TREES:
            self.f.paths[name].chmod(0o755)
            (self.f.paths[name] / "fixture.json").chmod(0o644)
        with patch.object(canary, "UID", 10002), patch.object(canary, "GID", 10002), \
             self.inspector_ownership([self.f.output, self.f.output / "plan.json"]):
            self.assertEqual(self.run_canary()["canary_exit_code"], 0)
        self.assertTrue((self.f.state / "plan-canonico-clon.json").exists())

    def test_same_file_bytes_under_replaced_ancestor_do_not_promote(self):
        source_parent = self.root / "source-parent"
        source_parent.mkdir(mode=0o700)
        copied = source_parent / "arr"
        copied.write_bytes(self.f.paths["arr"].read_bytes())
        copied.chmod(0o600)
        self.f.paths["arr"] = copied
        def change():
            source_parent.rename(self.root / "source-parent-old")
            shutil.copytree(self.root / "source-parent-old", source_parent)
        self.f.runner.after_start = change
        self.assert_rejected()

    def test_publication_success_removes_pending_after_both_files(self):
        plan, receipt = b"plan\n", canary.canonical({"kind": "clon", "fixture": True})
        with canary.directory(self.f.state) as fd:
            canary.publish_pair(fd, plan, receipt)
        self.assertEqual((self.f.state / "plan-conexiones.json").read_bytes(), plan)
        self.assertEqual((self.f.state / "plan-canonico-clon.json").read_bytes(), receipt)
        self.assertFalse((self.f.state / canary.PENDING).exists())

    def test_publication_write_failure_preserves_pending_and_forces_new_state(self):
        plan, receipt = b"plan\n", canary.canonical({"kind": "clon", "fixture": True})
        original_open = canary.os.open
        def fault(name, flags, *args, **kwargs):
            if name == "plan-canonico-clon.json" and flags & os.O_CREAT:
                raise OSError("synthetic_write_failure")
            return original_open(name, flags, *args, **kwargs)
        with canary.directory(self.f.state) as fd, patch.object(canary.os, "open", side_effect=fault), self.assertRaises(OSError):
            canary.publish_pair(fd, plan, receipt)
        pending = canary.decode((self.f.state / canary.PENDING).read_bytes())
        self.assertEqual(pending["receipt"], canary.decode(receipt))
        self.assertEqual(pending["receipt_sha256"], canary.sha(receipt))
        self.assertTrue(pending["requires_new_state"])
        self.assertEqual((self.f.state / "plan-conexiones.json").read_bytes(), plan)
        with self.assertRaisesRegex(canary.Refused, "new_state_required"):
            self.run_canary()
        self.assertEqual(self.f.runner.calls, [])

    def test_publication_interruption_after_plan_preserves_full_pending_receipt(self):
        original_open = canary.os.open
        def interrupt(name, flags, *args, **kwargs):
            if name == "plan-canonico-clon.json" and flags & os.O_CREAT:
                raise KeyboardInterrupt()
            return original_open(name, flags, *args, **kwargs)
        receipt = canary.canonical({"kind": "clon", "fixture": True})
        with canary.directory(self.f.state) as fd, patch.object(canary.os, "open", side_effect=interrupt), self.assertRaises(KeyboardInterrupt):
            canary.publish_pair(fd, b"plan\n", receipt)
        self.assertEqual(canary.decode((self.f.state / canary.PENDING).read_bytes())["receipt"], canary.decode(receipt))
        with self.assertRaisesRegex(canary.Refused, "new_state_required"):
            self.run_canary()
        self.assertEqual(self.f.runner.calls, [])

    def test_pending_directory_fsync_failure_prevents_any_flat_publication(self):
        receipt = canary.canonical({"kind": "clon", "fixture": True})
        original_sync = canary.os.fsync
        with canary.directory(self.f.state) as state_fd:
            def fault(fd):
                if fd == state_fd:
                    raise OSError("synthetic_directory_fsync_failure")
                original_sync(fd)
            with patch.object(canary.os, "fsync", side_effect=fault), self.assertRaises(OSError):
                canary.publish_pair(state_fd, b"plan\n", receipt)
        self.assertEqual(canary.decode((self.f.state / canary.PENDING).read_bytes())["receipt"], canary.decode(receipt))
        self.assertFalse((self.f.state / "plan-conexiones.json").exists())
        self.assertFalse((self.f.state / "plan-canonico-clon.json").exists())
        with self.assertRaisesRegex(canary.Refused, "new_state_required"):
            self.run_canary()
        self.assertEqual(self.f.runner.calls, [])


if __name__ == "__main__":
    unittest.main()
