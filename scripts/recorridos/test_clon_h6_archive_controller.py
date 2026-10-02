"""Temporary synthetic approvals exercise the body; no real clon receipts.

The sole gate replacement is in-process monkeypatch. Docker, trusted helpers
and PG are fake; CLI never accepts fixture authority or an approval bypass.
"""
from contextlib import ExitStack
from dataclasses import replace
import copy
import io
import json
import os
from pathlib import Path
import unittest
from unittest.mock import patch
import tempfile
import tarfile

import clon_h6_archive_controller as controller
import clon_h6_archive as archive
from test_clon_h6_canario import Fixture
from test_clon_h6_docker_archive_driver import FakeCLI, tar_output


class FixtureCLI(FakeCLI):
    def __init__(self, fixture, request):
        super().__init__()
        self.fixture, self.request = fixture, request
        self.output = tar_output(fixture.runner.plan)
        self.after_start = lambda: None
        self.pg_mutate = lambda data: None
        self.pg = {"Id": request.pgid, "Image": request.pg_image,
            "Config": {"Image": request.pg_image, "Labels": {
                "vec.recorridos.owner": "Codex-M", "vec.recorridos.state": str(request.state)}},
            "State": {"Running": True}, "HostConfig": {"NetworkMode": "none"},
            "Mounts": [{"Type": "volume", "Name": "synthetic-pg-volume",
                "Source": "/synthetic-volume/data", "Destination": "/var/lib/postgresql", "RW": True}]}

    def execute(self, args, **kwargs):
        # Even the first read-only Docker call follows durable attempt reservation.
        assert (self.request.state / controller.ATTEMPT).is_file()
        if args[:3] == ["inspect", "--type", "container"] and args[-1] == self.request.pgid:
            self.calls.append((list(args), dict(kwargs)))
            data = copy.deepcopy(self.pg)
            self.pg_mutate(data)
            return json.dumps([data]).encode()
        result = super().execute(args, **kwargs)
        if args[0] == "start":
            self.after_start()
        return result


class ControllerTests(unittest.TestCase):
    def setUp(self):
        self.stack = ExitStack()
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        self.fixture = Fixture(self.root)
        self.fixture.config = replace(self.fixture.config, pgid="e" * 64)
        config = self.fixture.config
        self.request = controller.Request(config.state, config.pgid, "sha256:" + "f" * 64,
            config.image, tuple(config.paths.items()), tuple(config.approved.items()))
        self.cli = FixtureCLI(self.fixture, self.request)
        self.authority = controller.Authority(controller.validation.sha(controller.request_bytes(self.request)),
            tuple(controller.tree_digests(controller.validation.snapshot(controller.config_for(self.request))[0]).items()),
            controller.validation.sha(controller.validation.canonical(self.cli.pg["Mounts"])))
        self.stack.enter_context(patch.object(controller.transport, "LocalDockerCLI", return_value=self.cli))
        self.stack.enter_context(patch.object(controller.validation, "run", side_effect=self.fixture.runner))

    def tearDown(self):
        self.stack.close()

    def run_fixture(self):
        with patch.object(controller, "DEFINITIVE_AUTHORITY", self.authority):
            return controller.canario_archivo(self.request)

    def assert_no_pair(self):
        self.assertFalse((self.request.state / "plan-conexiones.json").exists())
        self.assertFalse((self.request.state / "plan-canonico-clon.json").exists())

    def test_missing_authority_denies_before_state_lock_snapshot_or_docker(self):
        missing = self.root / "state-absent"
        with patch.object(controller.validation, "directory", side_effect=AssertionError("no state")), \
             patch.object(controller, "freeze_inputs", side_effect=AssertionError("no snapshot")), \
             patch.object(controller.fcntl, "flock", side_effect=AssertionError("no lock")), \
             patch.object(controller.transport, "ArchiveSession", side_effect=AssertionError("no session")):
            with self.assertRaisesRegex(archive.Refused, "fresh_definitive_material_authority_missing"):
                controller.canario_archivo(replace(self.request, state=missing))
        self.assertFalse(missing.exists())
        self.assertEqual(self.cli.calls, [])

    def test_success_fixed_mounts_isolation_pair_and_retained_resources(self):
        before = {name: archive.identity(path.stat()) for name, path in self.fixture.paths.items()}
        result = self.run_fixture()
        self.assertEqual(result["canary_exit_code"], 0)
        self.assertEqual(result["pg_container_id"], self.request.pgid)
        self.assertFalse((self.request.state / controller.validation.PENDING).exists())
        receipt = json.loads((self.request.state / "plan-canonico-clon.json").read_bytes())
        self.assertEqual(receipt["plan_sha256"], result["plan_sha256"])
        self.assertEqual(len(self.cli.containers), 2)
        self.assertEqual(len(self.cli.volumes), 1)
        self.assertEqual(before, {name: archive.identity(path.stat()) for name, path in self.fixture.paths.items()})
        for args, _ in self.cli.calls:
            self.assertNotIn("--force", args)
            self.assertNotIn("rm", args)
            self.assertFalse(any("type=bind" in arg for arg in args))
            if args[0] == "create":
                self.assertIn("--network=none", args)
                self.assertIn("--pull=never", args)
                self.assertIn(self.request.image, args)
        canary = next(c for c in self.cli.containers.values() if c["Config"]["Entrypoint"] == ["/bin/bash"])
        self.assertEqual([m["Destination"] for m in canary["Mounts"] if m["RW"]], ["/h6-out"])
        for mount in canary["HostConfig"]["Mounts"]:
            self.assertEqual(mount["VolumeOptions"]["Subpath"], "output" if not mount["ReadOnly"] else (
                "input/files/inspector" if mount["Target"] == "/usr/local/bin/vec-server" else
                mount["VolumeOptions"]["Subpath"]))
            self.assertTrue(mount["ReadOnly"] or mount["Target"] == "/h6-out")

    def test_attempt_fsynced_before_cli_or_session_and_failed_fsync_prevents_retry(self):
        real_sync = controller.os.fsync
        synced = []
        def sync(fd):
            synced.append(os.readlink("/proc/self/fd/" + str(fd)))
            real_sync(fd)
        with patch.object(controller.os, "fsync", side_effect=sync):
            self.run_fixture()
        self.assertEqual(synced[:2], [str(self.request.state / controller.ATTEMPT), str(self.request.state)])
        first = (self.request.state / controller.ATTEMPT).read_bytes()
        calls = len(self.cli.calls)
        with self.assertRaisesRegex(archive.Refused, "new_state_required"):
            self.run_fixture()
        self.assertEqual(len(self.cli.calls), calls)
        self.assertEqual((self.request.state / controller.ATTEMPT).read_bytes(), first)

    def test_session_failure_after_reservation_is_never_replayed(self):
        with patch.object(controller.transport, "ArchiveSession", side_effect=OSError("fixture session crash")):
            with self.assertRaisesRegex(OSError, "session crash"):
                self.run_fixture()
        self.assertTrue((self.request.state / controller.ATTEMPT).exists())
        self.assertEqual(self.cli.containers, {})
        before = len(self.cli.calls)
        with self.assertRaisesRegex(archive.Refused, "new_state_required"):
            self.run_fixture()
        self.assertEqual(len(self.cli.calls), before)

    def test_partial_attempt_fsync_denies_session_and_replay(self):
        with patch.object(controller.os, "fsync", side_effect=OSError("fixture fsync")):
            with self.assertRaisesRegex(OSError, "fixture fsync"):
                self.run_fixture()
        self.assertTrue((self.request.state / controller.ATTEMPT).exists())
        self.assertEqual(self.cli.calls, [])
        with self.assertRaisesRegex(archive.Refused, "new_state_required"):
            self.run_fixture()
        self.assertEqual(self.cli.calls, [])

    def test_changed_runtime_bytes_or_tree_inventory_denies_before_reservation(self):
        for name in ("arr", "material"):
            path = self.fixture.paths[name] if name == "arr" else self.fixture.paths[name] / "fixture.json"
            original = path.read_bytes()
            path.write_bytes(b"changed\n")
            with self.assertRaises((archive.Refused, controller.validation.Refused)):
                self.run_fixture()
            self.assertFalse((self.request.state / controller.ATTEMPT).exists())
            self.assertEqual(self.cli.calls, [])
            path.write_bytes(original)

    def test_self_consistent_cli_pins_cannot_replace_authority(self):
        self.fixture.paths["ca"].write_bytes(b"different approved claim\n")
        pins = dict(self.request.approved)
        pins["ca"] = controller.validation.sha(self.fixture.paths["ca"].read_bytes())
        with patch.object(controller, "DEFINITIVE_AUTHORITY", self.authority):
            with self.assertRaisesRegex(archive.Refused, "authority_binding"):
                controller.canario_archivo(replace(self.request, approved=tuple(pins.items())))
        self.assertEqual(self.cli.calls, [])

    def test_mutated_inputs_after_start_preserve_resources_without_publication(self):
        self.cli.after_start = lambda: self.fixture.paths["reference"].write_bytes(b"changed\n")
        with self.assertRaises((archive.Refused, controller.validation.Refused)):
            self.run_fixture()
        self.assert_no_pair()
        self.assertEqual(len(self.cli.containers), 2)
        self.assertEqual(len(self.cli.volumes), 1)
        self.assertTrue((self.request.state / controller.WORK / "export" / "plan.json").exists())

    def test_staging_uses_frozen_bytes_even_if_original_changes_after_reservation(self):
        original = self.fixture.paths["arr"].read_bytes()
        real_create = controller.transport.ArchiveSession.create_staging
        def create(session):
            result = real_create(session)
            self.fixture.paths["arr"].write_bytes(b"changed original after reservation\n")
            return result
        with patch.object(controller.transport.ArchiveSession, "create_staging", create):
            with self.assertRaises((archive.Refused, controller.validation.Refused)):
                self.run_fixture()
        with tarfile.open(fileobj=io.BytesIO(self.cli.archives[0]), mode="r:") as packed:
            self.assertEqual(packed.extractfile("input/files/arr").read(), original)
        self.assert_no_pair()

    def test_changed_canary_subpath_identity_or_volume_stops_before_start(self):
        mutations = (
            lambda c: c.update(Id=c["Id"][:12]),
            lambda c: c["Mounts"][0].update(Name="foreign-volume"),
            lambda c: c["HostConfig"]["Mounts"][-1]["VolumeOptions"].update(Subpath=""))
        for mutate in mutations:
            with self.subTest(mutation=mutate):
                self.cli.mutate = lambda c: mutate(c) if c["Config"]["Entrypoint"] == ["/bin/bash"] else None
                with self.assertRaises(archive.Refused):
                    self.run_fixture()
                self.assertFalse(any(args[0] in ("start", "rm") for args, _ in self.cli.calls))
                self.assertTrue(self.cli.volumes)
                self.assert_no_pair()
                self.tearDown()
                self.setUp()

    def test_pg_mount_change_after_start_prevents_publication(self):
        self.cli.after_start = lambda: self.cli.pg["Mounts"][0].update(Source="/synthetic-volume/replaced")
        with self.assertRaisesRegex(archive.Refused, "pg_physical_identity"):
            self.run_fixture()
        self.assert_no_pair()
        self.assertEqual(len(self.cli.containers), 2)

    def test_pg_physical_image_volume_or_full_id_changes_before_session_are_refused(self):
        for mutate in (lambda p: p.update(Id="e" * 12), lambda p: p.update(Image=self.request.image),
                       lambda p: p["Mounts"][0].update(Source="/synthetic-volume/foreign")):
            with self.subTest(mutation=mutate):
                self.cli.pg_mutate = mutate
                with self.assertRaises((archive.Refused, controller.validation.Refused)):
                    self.run_fixture()
                self.assertEqual(self.cli.containers, {})
                self.assertEqual(self.cli.volumes, {})
                # Each synthetic scenario gets a fresh state, never an operational replay.
                self.tearDown()
                self.setUp()

    def test_timeout_or_create_error_preserves_intent_and_blocks_second_attempt(self):
        for failure in ("create", "start"):
            with self.subTest(failure=failure):
                self.cli.fail = failure
                with self.assertRaisesRegex(archive.Refused, "effect_unknown"):
                    self.run_fixture()
                self.assertTrue(self.cli.volumes)
                self.assert_no_pair()
                before = len(self.cli.calls)
                with self.assertRaisesRegex(archive.Refused, "new_state_required"):
                    self.run_fixture()
                self.assertEqual(len(self.cli.calls), before)
                self.assertFalse(any(args[0] == "rm" for args, _ in self.cli.calls))
                self.tearDown()
                self.setUp()

    def test_partial_publication_keeps_pending_plan_and_blocks_replay(self):
        real_open = controller.os.open
        def fail_receipt(path, *args, **kwargs):
            if path == "plan-canonico-clon.json" and args[0] & os.O_CREAT:
                raise OSError("fixture publication crash")
            return real_open(path, *args, **kwargs)
        with patch.object(controller.os, "open", side_effect=fail_receipt):
            with self.assertRaisesRegex(OSError, "publication crash"):
                self.run_fixture()
        self.assertTrue((self.request.state / controller.validation.PENDING).exists())
        self.assertTrue((self.request.state / "plan-conexiones.json").exists())
        self.assertFalse((self.request.state / "plan-canonico-clon.json").exists())
        before = len(self.cli.calls)
        with self.assertRaisesRegex(archive.Refused, "new_state_required"):
            self.run_fixture()
        self.assertEqual(len(self.cli.calls), before)
        self.assertEqual(len(self.cli.containers), 2)

    def test_cli_rejects_duplicate_abbreviated_or_bypass_options(self):
        for args in (("--state", "/first", "--state", "/second"), ("--sta", "/absent"),
                     ("--approved", "true"), ("--fixture-authority", "/absent")):
            with self.assertRaises(archive.Refused):
                controller.arguments(args)


if __name__ == "__main__":
    unittest.main()
