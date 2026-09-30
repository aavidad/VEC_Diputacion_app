"""Bounded H6 Docker archive primitives; no host ownership changes or extraction.

An externally approved, immutable inventory supplies every input name and hash.
The integrator owns container/volume creation and removal, Docker timeouts and
the final plan/receipt publication. A stopped staging container mounts its sole
volume at /h6-stage: import creates input/ and output/ there, owned by 10002.
The canary mounts input subpaths read-only and output/ at /output read-write.
Only /output/plan.json is exported; trusted host helpers produce the receipt.

DockerArchive can be implemented with Docker cp stdin/stdout or the archive API.
No Docker SDK is required here. Its inspect returns one decoded container object,
and get_archive returns a context manager over an uncompressed binary stream.
The adapter must impose wall-clock and stream limits and close failed streams.
"""
from __future__ import annotations

from contextlib import contextmanager
from dataclasses import dataclass
import hashlib
import io
import os
from pathlib import Path
import re
import stat
import tarfile
from typing import BinaryIO, ContextManager, Protocol

UID = GID = 10002
OUTPUT_LIMIT = 131072
INPUT_LIMIT = 64 * 1048576
FILE_LIMIT = 32 * 1048576
MEMBER_LIMIT = 1024
STAGE_PATH = "/h6-stage"
OUTPUT_PATH = "/output/plan.json"
OWNER_LABEL = "vec.h6.archive.owner"
HEX = re.compile(r"[0-9a-f]{64}\Z")
IMAGE = re.compile(r"sha256:[0-9a-f]{64}\Z")
COMPONENT = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}\Z")
DIR_FLAGS = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC
FILE_FLAGS = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC


class Refused(RuntimeError):
    pass


def require(condition, code):
    if not condition:
        raise Refused(code)


@dataclass(frozen=True)
class ApprovedInput:
    name: str
    path: Path
    sha256: str
    executable: bool = False
    limit: int = FILE_LIMIT


@dataclass(frozen=True)
class VolumeMount:
    name: str
    destination: str
    writable: bool
    subpath: str = ""


@dataclass(frozen=True)
class OwnedContainer:
    id: str
    image: str
    owner: str
    mounts: tuple[VolumeMount, ...]


class DockerArchive(Protocol):
    def inspect(self, container_id: str) -> dict: ...

    def put_archive(self, container_id: str, path: str, stream: BinaryIO) -> None: ...

    def get_archive(self, container_id: str, path: str) -> ContextManager[BinaryIO]: ...


def closed_name(name):
    require(isinstance(name, str) and 0 < len(name) <= 240, "input_name")
    parts = name.split("/")
    require(len(parts) <= 8 and all(COMPONENT.fullmatch(p) for p in parts), "input_name")
    return parts


@contextmanager
def directory(path):
    path = Path(path)
    require(path.is_absolute() and ".." not in path.parts and path != Path("/"), "host_path")
    fd = os.open("/", DIR_FLAGS)
    try:
        for part in path.parts[1:]:
            child = os.open(part, DIR_FLAGS, dir_fd=fd)
            os.close(fd)
            fd = child
        yield fd
    finally:
        os.close(fd)


def identity(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_gid,
            info.st_nlink, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def approved_bytes(entry):
    require(HEX.fullmatch(entry.sha256) and type(entry.executable) is bool and
            type(entry.limit) is int and 0 < entry.limit <= FILE_LIMIT, "input_approval")
    path = Path(entry.path)
    with directory(path.parent) as parent:
        fd = os.open(path.name, FILE_FLAGS, dir_fd=parent)
        try:
            before = os.fstat(fd)
            require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                    0 < before.st_size <= entry.limit and not before.st_mode & 0o022, "input_file")
            result = bytearray()
            while len(result) < before.st_size:
                chunk = os.read(fd, min(65536, before.st_size - len(result)))
                require(bool(chunk), "input_changed")
                result.extend(chunk)
            require(not os.read(fd, 1) and identity(before) == identity(os.fstat(fd)), "input_changed")
            data = bytes(result)
            require(hashlib.sha256(data).hexdigest() == entry.sha256, "input_hash")
            return data
        finally:
            os.close(fd)


def member(name, *, data=None, executable=False):
    info = tarfile.TarInfo(name)
    info.uid = UID
    info.gid = GID
    info.mtime = 0
    info.mode = 0o700 if data is None or executable else 0o600
    if data is None:
        info.type = tarfile.DIRTYPE
    else:
        info.size = len(data)
    return info


def input_archive(entries, *, directories=()):
    """Verify host bytes once and pack that verified snapshot, never reread paths."""
    entries = tuple(entries)
    directories = tuple(directories)
    require(0 < len(entries) <= MEMBER_LIMIT and len(directories) <= MEMBER_LIMIT, "input_count")
    files, dirs, total = {}, {"input"}, 0
    for name in directories:
        parts = closed_name(name)
        dirs.update("input/" + "/".join(parts[:i]) for i in range(1, len(parts) + 1))
    for entry in entries:
        parts = closed_name(entry.name)
        key = "input/" + entry.name
        require(key not in files, "duplicate_input")
        dirs.update("input/" + "/".join(parts[:i]) for i in range(1, len(parts)))
        data = approved_bytes(entry)
        total += len(data)
        require(total <= INPUT_LIMIT, "input_limit")
        files[key] = (entry, data)
    require(not dirs.intersection(files) and len(files) + len(dirs) <= MEMBER_LIMIT, "input_inventory")
    result = io.BytesIO()
    with tarfile.open(fileobj=result, mode="w", format=tarfile.USTAR_FORMAT) as archive:
        for name in sorted(dirs):
            archive.addfile(member(name))
        for name, (entry, data) in sorted(files.items()):
            archive.addfile(member(name, data=data, executable=entry.executable), io.BytesIO(data))
    return result.getvalue()


def output_archive():
    """Create an empty UID10002 private output directory inside the owned volume."""
    result = io.BytesIO()
    with tarfile.open(fileobj=result, mode="w", format=tarfile.USTAR_FORMAT) as archive:
        archive.addfile(member("output"))
    return result.getvalue()


def validate_container(docker, owned, *, canary):
    require(HEX.fullmatch(owned.id) and IMAGE.fullmatch(owned.image) and
            COMPONENT.fullmatch(owned.owner), "container_approval")
    expected = tuple((m.name, m.destination, m.writable) for m in owned.mounts)
    require(expected and len(set(expected)) == len(expected) and
            len({m.destination for m in owned.mounts}) == len(expected), "mount_approval")
    for mount in owned.mounts:
        require(COMPONENT.fullmatch(mount.name) and type(mount.writable) is bool and
                mount.destination.startswith("/") and
                closed_name(mount.destination[1:]), "mount_approval")
        if mount.subpath:
            closed_name(mount.subpath)
    data = docker.inspect(owned.id)
    require(isinstance(data, dict), "container_inspect")
    config, host, state = (data.get(k, {}) for k in ("Config", "HostConfig", "State"))
    require(data.get("Id") == owned.id and data.get("Image") == owned.image and
            config.get("Image") == owned.image and
            (config.get("Labels") or {}).get(OWNER_LABEL) == owned.owner, "container_identity")
    require(host.get("NetworkMode") == "none" and host.get("Privileged") is False and
            not host.get("Binds") and not host.get("VolumesFrom") and
            not host.get("Devices") and not host.get("DeviceRequests") and
            not host.get("PortBindings"), "container_isolation")
    actual = data.get("Mounts")
    require(isinstance(actual, list) and len(actual) == len(expected) and
            all(m.get("Type") == "volume" and type(m.get("RW")) is bool for m in actual) and
            sorted((m.get("Name"), m.get("Destination"), m.get("RW")) for m in actual) == sorted(expected),
            "container_mounts")
    # Docker's resolved Mounts omits volume subpaths. Inspect the original mount
    # configuration too: a writable mount of the volume root would expose input/.
    configured = host.get("Mounts")
    require(isinstance(configured, list) and len(configured) == len(expected), "container_subpaths")
    for mount in owned.mounts:
        matches = [m for m in configured if m.get("Target") == mount.destination]
        require(len(matches) == 1, "container_subpaths")
        actual = matches[0]
        options = actual.get("VolumeOptions") or {}
        require(actual.get("Type") == "volume" and actual.get("Source") == mount.name and
                actual.get("ReadOnly") is (not mount.writable) and
                options.get("Subpath", "") == mount.subpath and options.get("NoCopy") is True,
                "container_subpaths")
    require(state.get("Running") is False and state.get("Paused") is not True and
            state.get("Restarting") is not True, "container_running")
    if canary:
        require(config.get("User") == "10002:10002" and
                type(state.get("ExitCode")) is int and state["ExitCode"] == 0 and
                state.get("Status") == "exited" and state.get("OOMKilled") is False, "canary_exit")
        require(sum(m.writable for m in owned.mounts) == 1 and
                any(m.destination == "/output" and m.writable and m.subpath == "output" for m in owned.mounts) and
                all(m.subpath == "input" or m.subpath.startswith("input/") for m in owned.mounts if not m.writable),
                "canary_mount_policy")
    else:
        require(state.get("Status") == "created" and
                expected == ((owned.mounts[0].name, STAGE_PATH, True),) and
                owned.mounts[0].subpath == "", "staging_mount_policy")
    return data


def import_inputs(docker, staging, entries, *, directories=()):
    """Copy a verified USTAR snapshot to a never-started owned staging container."""
    archive = input_archive(entries, directories=directories)
    validate_container(docker, staging, canary=False)
    docker.put_archive(staging.id, STAGE_PATH, io.BytesIO(archive))
    validate_container(docker, staging, canary=False)
    docker.put_archive(staging.id, STAGE_PATH, io.BytesIO(output_archive()))
    validate_container(docker, staging, canary=False)
    return hashlib.sha256(archive).hexdigest()


def read_exact(stream, size):
    result = bytearray()
    while len(result) < size:
        chunk = stream.read(min(65536, size - len(result)))
        require(isinstance(chunk, bytes) and 0 < len(chunk) <= size - len(result), "archive_truncated")
        result.extend(chunk)
    return bytes(result)


def plan_bytes(stream):
    """Parse the raw header so tarfile cannot silently consume PAX/GNU headers."""
    header = read_exact(stream, 512)
    try:
        info = tarfile.TarInfo.frombuf(header, "ascii", "strict")
    except (tarfile.TarError, UnicodeError, ValueError) as error:
        raise Refused("archive_header") from error
    require(info.name == "plan.json" and info.type in (tarfile.REGTYPE, tarfile.AREGTYPE) and
            not info.linkname and info.uid == UID and info.gid == GID and info.mode == 0o600 and
            0 < info.size <= OUTPUT_LIMIT, "archive_member")
    data = read_exact(stream, info.size)
    padding = read_exact(stream, (-info.size) % 512)
    require(not any(padding), "archive_padding")
    # Require two end blocks; bound and inspect every trailing byte, including
    # concatenated archives and members after an early end marker.
    require(read_exact(stream, 1024) == bytes(1024), "archive_extra_member")
    remaining = 10240
    while True:
        chunk = stream.read(min(512, remaining + 1))
        require(isinstance(chunk, bytes) and len(chunk) <= min(512, remaining + 1), "archive_stream")
        if not chunk:
            break
        remaining -= len(chunk)
        require(remaining >= 0 and not any(chunk), "archive_trailer")
    return data


def export_output(docker, canary, scratch):
    """Export only plan.json into a private host directory, atomically and fsynced.

    The caller must supply a fresh, host-owned 0700 scratch directory. Nothing
    in it is mounted into the canary. On failure the created file is discarded.
    The returned digest binds the bytes consumed by the trusted receipt helper.
    """
    with directory(scratch) as fd:
        before = os.fstat(fd)
        require(before.st_uid == os.getuid() and stat.S_IMODE(before.st_mode) == 0o700 and
                os.listdir(fd) == [], "scratch_private")
        validate_container(docker, canary, canary=True)
        with docker.get_archive(canary.id, OUTPUT_PATH) as stream:
            data = plan_bytes(stream)
        validate_container(docker, canary, canary=True)
        require(identity(before) == identity(os.fstat(fd)) and os.listdir(fd) == [], "scratch_changed")
        leaf = os.open("plan.json", os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                       0o600, dir_fd=fd)
        try:
            with os.fdopen(leaf, "wb") as output:
                info = os.fstat(output.fileno())
                require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and
                        info.st_nlink == 1 and stat.S_IMODE(info.st_mode) == 0o600, "scratch_file")
                output.write(data)
                output.flush()
                os.fsync(output.fileno())
            os.fsync(fd)
        except BaseException:
            os.unlink("plan.json", dir_fd=fd)
            os.fsync(fd)
            raise
    return hashlib.sha256(data).hexdigest()
