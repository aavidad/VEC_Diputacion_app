"""Pure archive and fake Docker checks; no Docker or database operations."""
from contextlib import contextmanager
import copy
import hashlib
import importlib.util
import io
import os
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("clon_h6_archive", Path(__file__).with_name("clon_h6_archive.py"))
transport = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = transport
SPEC.loader.exec_module(transport)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def archive(data=b'{"conexiones":[],"huellas":{"fixture":"synthetic"}}\n', *,
            name="plan.json", kind=tarfile.REGTYPE, mode=0o600, uid=10002,
            gid=10002, extra=False, pax=False, linkname=""):
    result = io.BytesIO()
    with tarfile.open(fileobj=result, mode="w", format=tarfile.PAX_FORMAT if pax else tarfile.USTAR_FORMAT) as tar:
        info = tarfile.TarInfo(name)
        info.uid, info.gid, info.mode, info.type = uid, gid, mode, kind
        info.linkname = linkname
        if pax:
            info.pax_headers = {"path": name}
        if kind in (tarfile.REGTYPE, tarfile.AREGTYPE):
            info.size = len(data)
        tar.addfile(info, io.BytesIO(data) if info.size else None)
        if extra:
            tar.addfile(tarfile.TarInfo("extra.json"))
    return result.getvalue()


class FakeDocker:
    def __init__(self):
        self.image = "sha256:" + "b" * 64
        self.stage = transport.OwnedContainer("a" * 64, self.image, "fixture-owner", (
            transport.VolumeMount("fixture-volume", "/h6-stage", True),))
        self.canary = transport.OwnedContainer("c" * 64, self.image, "fixture-owner", (
            transport.VolumeMount("fixture-volume", "/input", False, "input"),
            transport.VolumeMount("fixture-volume", "/output", True, "output")))
        self.calls = []
        self.uploaded = []
        self.payload = archive()
        self.stream_closed = False
        self.mutate = lambda owned, data: None

    def inspect(self, cid):
        self.calls.append(("inspect", cid))
        owned = self.stage if cid == self.stage.id else self.canary
        data = {"Id": cid, "Image": self.image,
                "Config": {"Image": self.image, "User": "10002:10002",
                           "Labels": {transport.OWNER_LABEL: owned.owner}},
                "HostConfig": {"NetworkMode": "none", "Privileged": False,
                               "Mounts": [{"Type": "volume", "Source": m.name, "Target": m.destination,
                                           "ReadOnly": not m.writable,
                                           "VolumeOptions": {"Subpath": m.subpath, "NoCopy": True}}
                                          for m in owned.mounts]},
                "Mounts": [{"Type": "volume", "Name": m.name, "Destination": m.destination,
                            "RW": m.writable} for m in owned.mounts],
                "State": {"Running": False, "Status": "created" if cid == self.stage.id else "exited",
                          "ExitCode": 0, "OOMKilled": False}}
        self.mutate(owned, data)
        return data

    def put_archive(self, cid, path, stream):
        self.calls.append(("put", cid, path))
        self.uploaded.append(stream.read())

    @contextmanager
    def get_archive(self, cid, path):
        self.calls.append(("get", cid, path))
        stream = io.BytesIO(self.payload)
        try:
            yield stream
        finally:
            stream.close()
            self.stream_closed = True


class ArchiveTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.scratch = self.root / "scratch"
        self.scratch.mkdir(mode=0o700)
        self.file = self.root / "private.json"
        self.file.write_bytes(b'{"synthetic":"private"}\n')
        self.file.chmod(0o600)
        self.entry = transport.ApprovedInput("material/private.json", self.file, digest(self.file.read_bytes()))
        self.docker = FakeDocker()

    def tearDown(self):
        self.temp.cleanup()

    def export(self):
        return transport.export_output(self.docker, self.docker.canary, self.scratch)

    def assert_export_refused(self):
        with self.assertRaises((transport.Refused, OSError)):
            self.export()
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_host_private_input_packed_for_canary_without_host_changes(self):
        before = transport.identity(self.file.stat())
        result = transport.import_inputs(self.docker, self.docker.stage, [self.entry], directories=["imports"])
        self.assertEqual(result, digest(self.docker.uploaded[0]))
        self.assertEqual(transport.identity(self.file.stat()), before)
        with tarfile.open(fileobj=io.BytesIO(self.docker.uploaded[0]), mode="r:") as tar:
            self.assertEqual([m.name for m in tar], ["input", "input/imports", "input/material", "input/material/private.json"])
            for info in tar:
                self.assertEqual((info.uid, info.gid), (10002, 10002))
                self.assertEqual(info.mode, 0o700 if info.isdir() else 0o600)
                self.assertEqual(info.pax_headers, {})
            self.assertEqual(tar.extractfile("input/material/private.json").read(), self.file.read_bytes())
        with tarfile.open(fileobj=io.BytesIO(self.docker.uploaded[1]), mode="r:") as tar:
            self.assertEqual([(m.name, m.uid, m.gid, m.mode, m.type) for m in tar],
                             [("output", 10002, 10002, 0o700, tarfile.DIRTYPE)])
        self.assertEqual([c for c in self.docker.calls if c[0] == "put"],
                         [("put", self.docker.stage.id, "/h6-stage")] * 2)

    def test_unapproved_bytes_refused_before_any_docker_operation(self):
        self.file.write_bytes(b"changed\n")
        with self.assertRaisesRegex(transport.Refused, "input_hash"):
            transport.import_inputs(self.docker, self.docker.stage, [self.entry])
        self.assertEqual(self.docker.calls, [])

    def test_verified_snapshot_is_not_reopened_after_import_begins(self):
        original = self.file.read_bytes()
        original_put = self.docker.put_archive
        def put(cid, path, stream):
            self.file.write_bytes(b"later change\n")
            original_put(cid, path, stream)
        self.docker.put_archive = put
        transport.import_inputs(self.docker, self.docker.stage, [self.entry])
        with tarfile.open(fileobj=io.BytesIO(self.docker.uploaded[0])) as tar:
            self.assertEqual(tar.extractfile("input/material/private.json").read(), original)

    def test_changed_file_during_descriptor_read_refused(self):
        original_read = os.read
        def read(fd, count):
            result = original_read(fd, count)
            if result:
                with self.file.open("ab") as target:
                    target.write(b"change")
            return result
        with patch.object(transport.os, "read", read):
            with self.assertRaisesRegex(transport.Refused, "input_changed"):
                transport.input_archive([self.entry])

    def test_input_links_and_special_files_refused(self):
        for kind in ("symlink", "hardlink", "fifo"):
            with self.subTest(kind=kind):
                path = self.root / kind
                if kind == "symlink":
                    path.symlink_to(self.file)
                elif kind == "hardlink":
                    os.link(self.file, path)
                else:
                    os.mkfifo(path)
                entry = transport.ApprovedInput("fixture", path, self.entry.sha256)
                with self.assertRaises((transport.Refused, OSError)):
                    transport.input_archive([entry])
                path.unlink()

    def test_symlink_parent_refused(self):
        (self.root / "linked").symlink_to(self.root, target_is_directory=True)
        entry = transport.ApprovedInput("fixture", self.root / "linked/private.json", self.entry.sha256)
        with self.assertRaises(OSError):
            transport.input_archive([entry])

    def test_input_closed_names_collisions_and_limits(self):
        cases = ["../escape", "/absolute", "a/../escape", "a//b", "a\\b", "a\nfile", "a/.", "a/" ]
        for name in cases:
            with self.subTest(name=name):
                with self.assertRaises(transport.Refused):
                    transport.input_archive([transport.ApprovedInput(name, self.file, self.entry.sha256)])
        with self.assertRaisesRegex(transport.Refused, "duplicate_input"):
            transport.input_archive([self.entry, self.entry])
        with self.assertRaisesRegex(transport.Refused, "input_inventory"):
            transport.input_archive([self.entry], directories=[self.entry.name])
        with patch.object(transport, "INPUT_LIMIT", 1):
            with self.assertRaisesRegex(transport.Refused, "input_limit"):
                transport.input_archive([self.entry])

    def test_executable_uses_private_execute_mode(self):
        entry = transport.ApprovedInput("resolver", self.file, self.entry.sha256, executable=True)
        with tarfile.open(fileobj=io.BytesIO(transport.input_archive([entry]))) as tar:
            self.assertEqual(tar.getmember("input/resolver").mode, 0o700)

    def test_export_private_bytes_fsynced_and_no_extraction(self):
        data = transport.plan_bytes(io.BytesIO(self.docker.payload))
        with patch.object(tarfile.TarFile, "extract", side_effect=AssertionError("extract forbidden")), \
             patch.object(tarfile.TarFile, "extractall", side_effect=AssertionError("extractall forbidden")), \
             patch.object(transport.os, "fsync", wraps=os.fsync) as sync:
            result = self.export()
            self.assertEqual(sync.call_count, 2)
        self.assertEqual(result, digest(data))
        self.assertEqual((self.scratch / "plan.json").read_bytes(), data)
        info = (self.scratch / "plan.json").stat()
        self.assertEqual((info.st_uid, info.st_mode & 0o777, info.st_nlink), (os.getuid(), 0o600, 1))
        self.assertEqual([c for c in self.docker.calls if c[0] == "get"],
                         [("get", self.docker.canary.id, "/output/plan.json")])
        self.assertTrue(self.docker.stream_closed)

    def test_exact_limit_allowed_and_over_limit_refused(self):
        self.docker.payload = archive(b"x" * transport.OUTPUT_LIMIT)
        self.export()
        (self.scratch / "plan.json").unlink()
        self.docker.payload = archive(b"x" * (transport.OUTPUT_LIMIT + 1))
        self.assert_export_refused()

    def test_output_names_types_metadata_pax_and_extra_refused(self):
        cases = [{"name": name} for name in ("../plan.json", "/plan.json", "sub/plan.json", "./plan.json", "receipt.json")]
        cases += [{"kind": kind} for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.DIRTYPE,
                                               tarfile.FIFOTYPE, tarfile.CHRTYPE, tarfile.BLKTYPE,
                                               tarfile.GNUTYPE_SPARSE, tarfile.GNUTYPE_LONGNAME)]
        cases += [{"mode": 0o644}, {"uid": 1000}, {"gid": 1000}, {"pax": True},
                  {"extra": True}, {"linkname": "private"}, {"data": b""}]
        for case in cases:
            with self.subTest(case=case):
                self.docker.payload = archive(**case)
                self.assert_export_refused()

    def test_truncation_checksums_padding_and_trailing_archives_refused(self):
        complete = archive(b"{}")
        cases = [complete[:length] for length in (0, 511, 512, 513, 1024, 1535)]
        bad_header = bytearray(complete)
        bad_header[0] ^= 1
        cases.append(bytes(bad_header))
        bad_padding = bytearray(complete)
        bad_padding[514] = 1
        cases.append(bytes(bad_padding))
        cases += [complete + archive(), complete + b"x", complete + bytes(10241)]
        for payload in cases:
            with self.subTest(length=len(payload)):
                self.docker.payload = payload
                self.assert_export_refused()

    def test_chunked_archive_stream_reads_are_supported(self):
        class Chunked(io.BytesIO):
            def read(self, count=-1):
                return super().read(min(count, 3))
        self.assertEqual(transport.plan_bytes(Chunked(archive(b"{}"))), b"{}")

    def test_container_identity_isolation_exit_and_mounts_refused(self):
        mutations = [lambda d: d.update(Id="d" * 64),
                     lambda d: d.update(Image="sha256:" + "e" * 64),
                     lambda d: d["Config"].update(Image="tag:latest"),
                     lambda d: d["Config"].update(Labels={transport.OWNER_LABEL: "another-owner"}),
                     lambda d: d["Config"].update(User="0:0"),
                     lambda d: d["HostConfig"].update(NetworkMode="bridge"),
                     lambda d: d["HostConfig"].update(Privileged=True),
                     lambda d: d["HostConfig"].update(Binds=["/private:/private"]),
                     lambda d: d["HostConfig"].update(VolumesFrom=["another-container"]),
                     lambda d: d["State"].update(Running=True),
                     lambda d: d["State"].update(ExitCode=1),
                     lambda d: d["State"].update(ExitCode=False),
                     lambda d: d["State"].update(OOMKilled=True),
                     lambda d: d["State"].update(Status="created"),
                     lambda d: d["Mounts"].append(copy.deepcopy(d["Mounts"][0])),
                     lambda d: d["Mounts"][0].update(RW=True),
                     lambda d: d["Mounts"][0].update(Type="bind"),
                     lambda d: d["Mounts"][0].update(Name="another-volume"),
                     lambda d: d["HostConfig"]["Mounts"][1]["VolumeOptions"].update(Subpath=""),
                     lambda d: d["HostConfig"]["Mounts"][1]["VolumeOptions"].update(Subpath="input"),
                     lambda d: d["HostConfig"]["Mounts"][0]["VolumeOptions"].update(Subpath="output"),
                     lambda d: d["HostConfig"]["Mounts"][0]["VolumeOptions"].update(NoCopy=False)]
        for mutation in mutations:
            with self.subTest(mutation=mutations.index(mutation)):
                self.docker.calls = []
                self.docker.mutate = lambda owned, data: mutation(data)
                self.assert_export_refused()
                self.assertFalse(any(c[0] == "get" for c in self.docker.calls))

    def test_container_rechecked_after_archive_is_consumed(self):
        self.docker.mutate = lambda owned, data: data["State"].update(Running=True) if self.docker.stream_closed else None
        self.assert_export_refused()

    def test_stage_must_be_owned_created_and_single_writable_volume(self):
        self.docker.mutate = lambda owned, data: data["State"].update(Status="exited")
        with self.assertRaises(transport.Refused):
            transport.import_inputs(self.docker, self.docker.stage, [self.entry])
        self.assertEqual(self.docker.uploaded, [])

    def test_host_scratch_links_permissions_and_existing_outputs_refused(self):
        self.scratch.chmod(0o755)
        self.assert_export_refused()
        self.scratch.chmod(0o700)
        linked = self.root / "linked-scratch"
        linked.symlink_to(self.scratch)
        with self.assertRaises(OSError):
            transport.export_output(self.docker, self.docker.canary, linked)
        (self.scratch / "plan.json").symlink_to(self.file)
        with self.assertRaises(transport.Refused):
            self.export()
        self.assertTrue((self.scratch / "plan.json").is_symlink())
        self.assertEqual(self.file.read_bytes(), b'{"synthetic":"private"}\n')

    def test_fsync_failure_discards_partial_output(self):
        with patch.object(transport.os, "fsync", side_effect=[OSError("fixture fsync failure"), None]):
            with self.assertRaises(OSError):
                self.export()
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_restrictive_umask_refuses_non_0600_destination(self):
        previous = os.umask(0o777)
        try:
            self.assert_export_refused()
        finally:
            os.umask(previous)


if __name__ == "__main__":
    unittest.main()
