"""Fake CLI protocol checks. No Docker, network, PG or host UID changes."""
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch

import clon_h6_archive as archive
import clon_h6_docker_archive_driver as driver


def tar_output(data=b'{"conexiones":[],"huellas":{"fixture":"synthetic"}}\n', *, uid=10002):
    result = io.BytesIO()
    with tarfile.open(fileobj=result, mode="w", format=tarfile.USTAR_FORMAT) as output:
        item = archive.member("plan.json", data=data)
        item.uid = uid
        output.addfile(item, io.BytesIO(data))
    return result.getvalue()


class FakeCLI:
    def __init__(self):
        self.calls = []
        self.containers = {}
        self.volumes = {}
        self.archives = []
        self.output = tar_output()
        self.directory_uid = 10002
        self.mutate = lambda data: None
        self.fail = None
        self.image = "sha256:" + "b" * 64

    @staticmethod
    def labels(args):
        result = {}
        for i, arg in enumerate(args):
            if arg == "--label":
                key, value = args[i + 1].split("=", 1)
                result[key] = value
        return result

    def execute(self, args, **kwargs):
        self.calls.append((list(args), dict(kwargs)))
        if self.fail is not None and args[0] == self.fail:
            raise archive.Refused("fixture_effect_unknown")
        if args[:2] == ["image", "inspect"]:
            return json.dumps([{"Id": self.image}]).encode()
        if args[:2] == ["volume", "create"]:
            self.volumes[args[-1]] = {"Name": args[-1], "Driver": "local", "Scope": "local", "Options": {},
                                      "Labels": self.labels(args)}
            return (args[-1] + "\n").encode()
        if args[:2] == ["volume", "inspect"]:
            return json.dumps([self.volumes[args[-1]]]).encode()
        if args[0] == "create":
            cid = ("a" if not self.containers else "c") * 64
            mount_options, mounts = [], []
            for i, arg in enumerate(args):
                if arg != "--mount":
                    continue
                parts = dict(part.split("=", 1) if "=" in part else (part, True) for part in args[i + 1].split(","))
                writable = not parts.get("readonly", False)
                mounts.append({"Type": "volume", "Name": parts["src"], "Destination": parts["dst"], "RW": writable})
                mount_options.append({"Type": "volume", "Source": parts["src"], "Target": parts["dst"], "ReadOnly": not writable,
                    "VolumeOptions": {"Subpath": parts.get("volume-subpath", ""), "NoCopy": bool(parts.get("volume-nocopy"))}})
            canary = "--entrypoint=/bin/bash" in args
            self.containers[cid] = {"Id": cid, "Image": self.image,
                "Config": {"Image": self.image, "Labels": self.labels(args), "User": "10002:10002", "Tty": False,
                    "OpenStdin": False, "Entrypoint": ["/bin/bash" if canary else "/bin/false"],
                    "Cmd": ["/vec-arrancar.sh"] if canary else []},
                "HostConfig": {"NetworkMode": "none", "ReadonlyRootfs": True, "Privileged": False, "CapDrop": ["ALL"],
                    "SecurityOpt": ["no-new-privileges"], "IpcMode": "private", "PidMode": "", "PidsLimit": 64,
                    "Memory": 512 * 1048576, "NanoCpus": 1000000000, "LogConfig": {"Type": "none"},
                    "RestartPolicy": {"Name": "no", "MaximumRetryCount": 0}, "AutoRemove": False, "Mounts": mount_options,
                    "Tmpfs": {"/tmp": "rw,nosuid,nodev,noexec,size=8m", "/dev/shm": "rw,nosuid,nodev,noexec,size=8m"} if canary else {},
                    "Ulimits": [{"Name": "fsize", "Soft": 131072, "Hard": 131072}] if canary else []},
                "Mounts": mounts, "State": {"Status": "created", "Running": False, "ExitCode": 0, "OOMKilled": False}}
            return (cid + "\n").encode()
        if args[0] == "inspect":
            data = copy.deepcopy(self.containers[args[-1]])
            self.mutate(data)
            return json.dumps([data]).encode()
        if args[0] == "cp":
            if args[2] == "-":
                self.archives.append(kwargs["input_bytes"])
                return b""
            if args[2].endswith(":/h6-stage/output"):
                result = io.BytesIO()
                with tarfile.open(fileobj=result, mode="w", format=tarfile.USTAR_FORMAT) as output:
                    item = archive.member("output")
                    item.uid = self.directory_uid
                    output.addfile(item)
                return result.getvalue()
            return self.output
        if args[0] == "start":
            self.containers[args[-1]]["State"]["Status"] = "exited"
            return b""
        if args[0] == "rm":
            del self.containers[args[-1]]
            return b""
        raise AssertionError(args)


class DriverTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.state = Path(self.temp.name)
        self.file = self.state / "fixture"
        self.file.write_bytes(b"synthetic\n")
        self.file.chmod(0o600)
        self.entry = archive.ApprovedInput("fixture", self.file, hashlib.sha256(self.file.read_bytes()).hexdigest())
        self.cli = FakeCLI()
        self.session = driver.ArchiveSession(self.state, self.cli.image, cli=self.cli)
        self.scratch = self.state / "scratch"
        self.scratch.mkdir(mode=0o700)

    def tearDown(self):
        self.temp.cleanup()

    def prepare(self):
        self.session.create_staging()
        self.session.stage([self.entry])
        return self.session.create_canary((
            archive.VolumeMount(self.session.volume, "/vec-arrancar.sh", False, "input/fixture"),
            archive.VolumeMount(self.session.volume, "/h6-out", True, "output")))

    def test_roundtrip_keeps_source_uid_and_uses_only_own_volume(self):
        before = archive.identity(self.file.stat())
        canary = self.prepare()
        self.session.start_canary()
        result = self.session.export_output(self.scratch)
        self.assertEqual(result, hashlib.sha256((self.scratch / "plan.json").read_bytes()).hexdigest())
        self.assertEqual(archive.identity(self.file.stat()), before)
        self.assertEqual(len(self.cli.archives), 2)
        for packed in self.cli.archives:
            with tarfile.open(fileobj=io.BytesIO(packed), mode="r:") as stream:
                self.assertTrue(all((m.uid, m.gid) == (10002, 10002) for m in stream))
        copies = [args for args, _ in self.cli.calls if args[0] == "cp"]
        self.assertEqual(copies, [["cp", "-a", "-", self.session.staging.id + ":/h6-stage"]] * 2 +
            [["cp", "-a", self.session.staging.id + ":/h6-stage/output", "-"],
             ["cp", "-a", canary.id + ":/h6-out/plan.json", "-"]])
        self.session.cleanup_stopped()
        self.assertEqual(self.cli.containers, {})
        self.assertIn(self.session.volume, self.cli.volumes)
        self.assertFalse(any("--force" in args or args[:2] == ["volume", "rm"] for args, _ in self.cli.calls))
        self.assertTrue(list(self.session.private.glob("evidence-*.json")))

    def test_all_created_containers_have_pinned_image_and_closed_isolation(self):
        self.prepare()
        creates = [args for args, _ in self.cli.calls if args[0] == "create"]
        for args in creates:
            for flag in ("--pull=never", "--network=none", "--user=10002:10002", "--read-only", "--cap-drop=ALL"):
                self.assertIn(flag, args)
            self.assertIn(self.cli.image, args)
            self.assertFalse(any("type=bind" in arg for arg in args))
            self.assertIn(driver.STATE_LABEL + "=" + str(self.state), args)
            self.assertIn(archive.OWNER_LABEL + "=" + self.session.owner, args)

    def test_mutated_hostconfig_mount_or_identity_prevents_start_and_preserves_evidence(self):
        mutations = [lambda d: d["HostConfig"].update(NetworkMode="bridge"),
            lambda d: d["HostConfig"].update(Binds=["/private:/private"]),
            lambda d: d["HostConfig"].update(CapAdd=["SYS_ADMIN"]),
            lambda d: d["HostConfig"].update(AutoRemove=True),
            lambda d: d["HostConfig"].update(SecurityOpt=["no-new-privileges", "seccomp=unconfined"]),
            lambda d: d["Mounts"][1].update(RW=False),
            lambda d: d["HostConfig"]["Mounts"][1]["VolumeOptions"].update(Subpath=""),
            lambda d: d["Mounts"].append({"Type": "bind", "Source": "/host"}),
            lambda d: d["State"].update(Running=True),
            lambda d: d["Config"].update(User="0:0"),
            lambda d: d["Config"]["Labels"].update({driver.STATE_LABEL: "foreign"}),
            lambda d: d["Config"].update(Entrypoint=["/bin/sh"])]
        self.prepare()
        for mutate in mutations:
            with self.subTest(mutation=mutations.index(mutate)):
                self.session.failed = False
                self.cli.mutate = mutate
                with self.assertRaises(archive.Refused):
                    self.session.start_canary()
                self.assertFalse(any(args[0] in ("start", "rm") for args, _ in self.cli.calls))
                self.assertIn(self.session.volume, self.cli.volumes)
        self.assertIn("failed_effect_unknown", self.session.evidence["phase"])

    def test_fsync_failure_at_export_completion_refuses_cleanup_and_retry(self):
        self.prepare()
        self.session.start_canary()
        original_record = self.session._record
        def record(phase):
            if phase == "export_complete":
                with patch.object(driver.os, "fsync", side_effect=OSError("fixture fsync failure")):
                    return original_record(phase)
            return original_record(phase)
        with patch.object(self.session, "_record", side_effect=record):
            with self.assertRaisesRegex(OSError, "fixture fsync failure"):
                self.session.export_output(self.scratch)
        self.assertTrue(self.session.failed)
        self.assertEqual(self.session.evidence["phase"], "export_failed_effect_unknown")
        evidence = [json.loads(p.read_bytes()) for p in self.session.private.glob("evidence-*.json")]
        self.assertTrue(any(e["phase"] == "export_failed_effect_unknown" for e in evidence))
        for operation in (self.session.cleanup_stopped, lambda: self.session.export_output(self.scratch)):
            with self.assertRaises(archive.Refused):
                operation()
        self.assertEqual(len(self.cli.containers), 2)
        self.assertIn(self.session.volume, self.cli.volumes)
        self.assertFalse(any(args[0] == "rm" for args, _ in self.cli.calls))

    def test_failed_failure_marker_cannot_enable_cleanup(self):
        self.prepare()
        self.session.start_canary()
        original_record = self.session._record
        def record(phase):
            if phase in ("export_complete", "export_failed_effect_unknown"):
                with patch.object(driver.os, "fsync", side_effect=OSError("fixture broken writer")):
                    return original_record(phase)
            return original_record(phase)
        with patch.object(self.session, "_record", side_effect=record):
            with self.assertRaisesRegex(OSError, "fixture broken writer"):
                self.session.export_output(self.scratch)
        self.assertTrue(self.session.failed)
        self.assertEqual(self.session.evidence["phase"], "export_failed_effect_unknown")
        with self.assertRaises(archive.Refused):
            self.session.cleanup_stopped()
        self.assertEqual(len(self.cli.containers), 2)
        self.assertIn(self.session.volume, self.cli.volumes)
        self.assertFalse(any(args[0] == "rm" for args, _ in self.cli.calls))

    def test_parent_fsync_failure_aborts_before_docker_and_preserves_session_directory(self):
        owner = "h6-" + "d" * 32
        with patch.object(driver.secrets, "token_hex", return_value="d" * 32), \
             patch.object(driver.os, "fsync", side_effect=OSError("fixture parent fsync failure")), \
             patch.object(driver, "LocalDockerCLI", side_effect=AssertionError("Docker must not be reached")):
            with self.assertRaisesRegex(OSError, "fixture parent fsync failure"):
                driver.ArchiveSession(self.state, self.cli.image, cli=self.cli)
        preserved = self.state / owner
        self.assertTrue(preserved.is_dir())
        self.assertEqual(list(preserved.iterdir()), [])
        self.assertEqual(self.cli.calls, [])
        # Reusing the interrupted name never opens or overwrites that directory.
        with patch.object(driver.secrets, "token_hex", return_value="d" * 32):
            with self.assertRaises(FileExistsError):
                driver.ArchiveSession(self.state, self.cli.image, cli=self.cli)
        self.assertEqual(list(preserved.iterdir()), [])
        self.assertEqual(self.cli.calls, [])

    def test_parent_directory_is_synced_before_session_evidence(self):
        original_sync = driver.os.fsync
        locations = []
        def sync(fd):
            locations.append(os.readlink("/proc/self/fd/" + str(fd)))
            return original_sync(fd)
        with patch.object(driver.os, "fsync", side_effect=sync):
            session = driver.ArchiveSession(self.state, self.cli.image, cli=self.cli)
        self.assertEqual(locations[0], str(self.state))
        self.assertTrue(any(path.startswith(str(session.private)) for path in locations[1:]))
        self.assertEqual(self.cli.calls, [])

    def test_intent_record_failure_blocks_effect_and_retry(self):
        with patch.object(driver.os, "fsync", side_effect=OSError("fixture broken intent")):
            with self.assertRaisesRegex(OSError, "fixture broken intent"):
                self.session.create_staging()
        self.assertTrue(self.session.failed)
        self.assertEqual(self.cli.calls, [])
        with self.assertRaises(archive.Refused):
            self.session.create_staging()

    def test_cli_resource_limits_do_not_restrict_go_virtual_reservation(self):
        with patch.object(driver.resource, "setrlimit") as limits:
            driver.cli_limits()
        actual = dict(call.args for call in limits.call_args_list)
        self.assertNotIn(driver.resource.RLIMIT_AS, actual)
        self.assertEqual(actual[driver.resource.RLIMIT_CPU], (10, 10))
        self.assertEqual(actual[driver.resource.RLIMIT_FSIZE], (1048576, 1048576))

    def test_ambiguous_start_failure_cannot_retry_or_cleanup(self):
        self.prepare()
        self.cli.fail = "start"
        with self.assertRaises(archive.Refused):
            self.session.start_canary()
        for operation in (self.session.start_canary, self.session.cleanup_stopped):
            with self.assertRaises(archive.Refused):
                operation()
        self.assertEqual(len([args for args, _ in self.cli.calls if args[0] == "start"]), 1)
        self.assertFalse(any(args[0] == "rm" for args, _ in self.cli.calls))
        self.assertIn(self.session.volume, self.cli.volumes)

    def test_create_failure_retains_intent_with_resource_names(self):
        self.cli.fail = "create"
        with self.assertRaises(archive.Refused):
            self.session.create_staging()
        evidence = [json.loads(p.read_bytes()) for p in self.session.private.glob("evidence-*.json")]
        self.assertTrue(any(e["phase"] == "create_staging_intent" and e["stage_name"] == self.session.stage_name for e in evidence))
        self.assertIn(self.session.volume, self.cli.volumes)
        with self.assertRaises(archive.Refused):
            self.session.create_staging()

    def test_uid_preservation_must_be_observed_before_canary_creation(self):
        self.session.create_staging()
        self.cli.directory_uid = 0
        with self.assertRaisesRegex(archive.Refused, "docker_uid10002_no_go"):
            self.session.stage([self.entry])
        self.assertFalse(self.session.imported)
        self.assertTrue(self.session.failed)
        self.assertEqual(len([args for args, _ in self.cli.calls if args[0] == "create"]), 1)
        self.assertIn(self.session.volume, self.cli.volumes)

    def test_ownership_loss_is_no_go_without_host_repair(self):
        self.prepare()
        self.session.start_canary()
        self.cli.output = tar_output(uid=0)
        with patch.object(os, "chown", side_effect=AssertionError("host chown forbidden")), \
             patch.object(os, "chmod", side_effect=AssertionError("host chmod forbidden")):
            with self.assertRaisesRegex(archive.Refused, "archive_member"):
                self.session.export_output(self.scratch)
        self.assertEqual(list(self.scratch.iterdir()), [])
        self.assertTrue(self.session.failed)
        self.assertIn(self.session.volume, self.cli.volumes)
        with self.assertRaises(archive.Refused):
            self.session.cleanup_stopped()

    def test_foreign_ids_archive_paths_mount_root_or_output_changes_are_refused(self):
        self.prepare()
        for path in ("/output/plan.json", "/h6-out/../private", "/h6-out/receipt.json"):
            with self.assertRaises(archive.Refused):
                with self.session.driver.get_archive(self.session.canary.id, path):
                    pass
        with self.assertRaises(archive.Refused):
            self.session.driver.inspect("f" * 64)
        with self.assertRaises(archive.Refused):
            self.session.driver.put_archive("f" * 64, "/h6-stage", io.BytesIO(b"x"))
        for mounts in ((archive.VolumeMount(self.session.volume, "/h6-out", True, ""),),
                       (archive.VolumeMount("foreign", "/h6-out", True, "output"),),
                       (archive.VolumeMount(self.session.volume, "/h6-out", True, "output"),
                        archive.VolumeMount(self.session.volume, "/input", True, "input"))):
            self.session.canary = None
            with self.assertRaises(archive.Refused):
                self.session.create_canary(mounts)

    def test_cleanup_rechecks_ownership_and_never_forces_running_container(self):
        self.prepare()
        self.session.start_canary()
        self.session.export_output(self.scratch)
        self.cli.mutate = lambda d: d["State"].update(Running=True)
        with self.assertRaises(archive.Refused):
            self.session.cleanup_stopped()
        self.assertFalse(any(args[0] == "rm" for args, _ in self.cli.calls))

    def test_private_state_and_config_reject_link_writable_or_inherited_context(self):
        config = self.state / "config"
        config.mkdir(mode=0o700)
        (config / "config.json").write_bytes(b"{}")
        with self.assertRaises(archive.Refused):
            driver.LocalDockerCLI(config)
        (config / "config.json").unlink()
        local = driver.LocalDockerCLI(config)
        (config / "contexts").mkdir()
        with patch.object(subprocess, "Popen", side_effect=AssertionError("no process")):
            with self.assertRaises(archive.Refused):
                local.execute(["image", "inspect", self.cli.image])
        linked = self.state / "linked"
        linked.symlink_to(config)
        with self.assertRaises(OSError):
            driver.LocalDockerCLI(linked)
        self.state.chmod(0o755)
        with self.assertRaises(archive.Refused):
            driver.ArchiveSession(self.state, self.cli.image, cli=self.cli)

    def test_fixed_socket_cli_env_and_argv_without_shell_or_tty(self):
        config = self.session.private / "docker-config"
        local = driver.LocalDockerCLI(config)
        process = unittest.mock.Mock()
        process.stdout = io.BytesIO()
        process.stdin = None
        with patch.object(subprocess, "Popen", return_value=process) as popen, \
             patch.object(driver.LocalDockerCLI, "_collect", return_value=b"[]"):
            self.assertEqual(local.execute(["inspect", "--type", "container", "a" * 64]), b"[]")
        args, kwargs = popen.call_args
        self.assertEqual(args[0], ["/usr/bin/docker", "--host", "unix:///var/run/docker.sock", "--config", str(config),
                                 "inspect", "--type", "container", "a" * 64])
        self.assertIs(kwargs["shell"], False)
        self.assertEqual(kwargs["env"], {"PATH": "/usr/bin:/bin", "HOME": str(config), "LANG": "C.UTF-8"})
        self.assertEqual(kwargs["stdin"], subprocess.DEVNULL)
        self.assertEqual(kwargs["stderr"], subprocess.DEVNULL)
        self.assertTrue(kwargs["close_fds"])

    def test_collector_stream_bound_and_timeout(self):
        # Fixture child has no Docker capability; tests run in a networkless sandbox.
        for payload, limit, expected in ((b"abcd", 3, "docker_output_limit"), (b"", 3, None)):
            process = subprocess.Popen(["/usr/bin/python3", "-c", "import os;os.write(1," + repr(payload) + ")"],
                stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env={})
            try:
                if expected:
                    with self.assertRaisesRegex(archive.Refused, expected):
                        driver.LocalDockerCLI._collect(process, None, limit, 3)
                else:
                    self.assertEqual(driver.LocalDockerCLI._collect(process, None, limit, 3), b"")
            finally:
                if process.poll() is None:
                    process.kill()
                process.wait(timeout=3)
                process.stdout.close()
        process = subprocess.Popen(["/usr/bin/python3", "-c", "import time;time.sleep(3)"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env={})
        try:
            with self.assertRaisesRegex(archive.Refused, "docker_timeout"):
                driver.LocalDockerCLI._collect(process, None, 3, 0.01)
        finally:
            process.kill()
            process.wait(timeout=3)
            process.stdout.close()


if __name__ == "__main__":
    unittest.main()
