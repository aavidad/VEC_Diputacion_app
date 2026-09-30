#!/usr/bin/env python3
"""Run the approved H6 ARR offline; retain only the local clone plan and receipt.

The caller prepares a fresh private output directory owned by 10002:10002.
This program never provisions accounts, changes ownership, or connects to PG.
All approval hashes come from the caller, outside the package being checked.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
from dataclasses import dataclass
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import resource
import stat
import subprocess
import sys
import tarfile
import tempfile

SOURCE = "73e56c106d12fdda0bd16d6fe573503c42c5495f"
HEX = re.compile(r"[0-9a-f]{64}\Z")
IMAGE = re.compile(r"sha256:[0-9a-f]{64}\Z")
UID = GID = 10002
FILES = ("arr", "reference", "ca", "inspector", "connections", "resolver",
         "promoter", "receipt_helper", "guiones", "lock", "package")
TREES = ("material", "incorporacion", "public_data", "communications", "imports")
HELPERS = {"inspector": "h6_canario_bin.sh", "connections": "h6_conexiones.py",
           "promoter": "h6_promover_plan.py", "receipt_helper": "h6_recibo_plan.py"}
FLAGS_DIR = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC
FLAGS_FILE = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC


class Refused(RuntimeError):
    pass


def require(condition, code):
    if not condition:
        raise Refused(code)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def unique(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate_json_key")
        result[key] = value
    return result


def decode(data):
    return json.loads(data, object_pairs_hook=unique)


def canonical(data):
    return (json.dumps(data, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n").encode()


@contextmanager
def directory(path):
    path = Path(path)
    require(path.is_absolute() and ".." not in path.parts and path != Path("/"), "unsafe_path")
    fd = os.open("/", FLAGS_DIR)
    try:
        for part in path.parts[1:]:
            child = os.open(part, FLAGS_DIR, dir_fd=fd)
            os.close(fd)
            fd = child
        yield fd
    finally:
        os.close(fd)


def identity(info):
    return (info.st_dev, info.st_ino, info.st_uid, info.st_gid,
            info.st_mode, info.st_nlink, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def read_at(fd, name, limit=1048576, *, owner=None, modes=None):
    leaf = os.open(name, FLAGS_FILE, dir_fd=fd)
    try:
        before = os.fstat(leaf)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                0 < before.st_size <= limit, "unsafe_file")
        if owner is not None:
            require(before.st_uid == owner, "file_owner")
        if modes is not None:
            require(stat.S_IMODE(before.st_mode) in modes, "file_mode")
        digest = hashlib.sha256()
        data = bytearray()
        count = 0
        while count < before.st_size:
            chunk = os.read(leaf, min(65536, before.st_size - count))
            require(bool(chunk), "file_changed")
            count += len(chunk)
            digest.update(chunk)
            if limit <= 1048576:
                data.extend(chunk)
        require(not os.read(leaf, 1) and identity(before) == identity(os.fstat(leaf)), "file_changed")
        return bytes(data), digest.hexdigest(), identity(before)
    finally:
        os.close(leaf)


def read(path, limit=1048576, **kwargs):
    path = Path(path)
    with directory(path.parent) as fd:
        return read_at(fd, path.name, limit, **kwargs)


def tree(path):
    result = {}
    total = 0
    def walk(fd, prefix="", depth=0):
        nonlocal total
        require(depth <= 8, "tree_depth")
        for name in sorted(os.listdir(fd)):
            require(re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", name) is not None and
                    name not in (".", ".."), "tree_name")
            info = os.stat(name, dir_fd=fd, follow_symlinks=False)
            key = prefix + name
            require(not info.st_mode & 0o022, "writable_input")
            if stat.S_ISDIR(info.st_mode):
                child = os.open(name, FLAGS_DIR, dir_fd=fd)
                try:
                    require(identity(info) == identity(os.fstat(child)), "tree_changed")
                    result[key + "/"] = identity(info)
                    walk(child, key + "/", depth + 1)
                finally:
                    os.close(child)
            else:
                _, digest, metadata = read_at(fd, name)
                total += metadata[6]
                result[key] = (metadata, digest)
            require(len(result) <= 1024 and total <= 32 * 1048576, "tree_limit")
    with directory(path) as fd:
        require(not os.fstat(fd).st_mode & 0o022, "writable_input")
        result["/"] = identity(os.fstat(fd))
        walk(fd)
    return result


@dataclass(frozen=True)
class Config:
    state: Path
    output: Path
    pgid: str
    image: str
    paths: dict[str, Path]
    approved: dict[str, str]


def snapshot(config):
    require(set(config.paths) == set(FILES + TREES) and set(config.approved) == set(FILES), "input_set")
    require(HEX.fullmatch(config.pgid) and IMAGE.fullmatch(config.image), "identity_format")
    result, contents = {}, {}
    for name in FILES:
        require(HEX.fullmatch(config.approved[name]), "approval_format")
        limit = {"package": 512 * 1048576, "resolver": 32 * 1048576}.get(name, 1048576)
        data, digest, metadata = read(config.paths[name], limit)
        require(digest == config.approved[name], "approval_hash")
        result[name] = (metadata, digest)
        contents[name] = data
    for name in TREES:
        result[name] = tree(config.paths[name])
    release = {}
    for line in contents["lock"].decode("ascii").splitlines():
        pair = line.split()
        require(len(pair) == 2 and pair[0] not in release, "release_lock")
        release[pair[0]] = pair[1]
    require(release.get("COMMIT") == SOURCE and release.get("PAQUETE_SHA256") == config.approved["package"], "release_mismatch")
    require(result["lock"][0][2] == os.getuid() and stat.S_IMODE(result["lock"][0][4]) == 0o600, "release_private")
    inventory = {}
    for line in contents["guiones"].decode("ascii").splitlines():
        pair = line.split()
        require(len(pair) == 2 and HEX.fullmatch(pair[0]) and pair[1] not in inventory and
                re.fullmatch(r"[A-Za-z0-9_.-]+", pair[1]), "guiones_inventory")
        inventory[pair[1]] = pair[0]
    require(all(inventory.get(filename) == config.approved[key] for key, filename in HELPERS.items()), "helper_inventory")
    effective = [line.strip() for line in contents["arr"].decode().splitlines()
                 if line.strip() and not line.lstrip().startswith("#")]
    require(effective and effective[-1] == "exec /usr/local/bin/vec-server" and
            sum("vec-server" in line for line in effective) == 1, "arr_final_exec")
    # No extraction: bind sources are tied to the approved archive members.
    members = {"./h6-canario-bin.sh": config.approved["inspector"], "./h6-conexiones.py": config.approved["connections"],
               "./h6-pg-resolver": config.approved["resolver"]}
    members.update({"./data/" + key: value[1] for key, value in result["public_data"].items() if not key.endswith("/")})
    with directory(config.paths["package"].parent) as parent:
        fd = os.open(config.paths["package"].name, FLAGS_FILE, dir_fd=parent)
        with os.fdopen(fd, "rb") as archive, tarfile.open(fileobj=archive, mode="r:gz") as tar:
            found = {}
            for member in tar:
                require(not member.issym() and not member.islnk() and
                        (member.isfile() or member.isdir()), "package_member_type")
                if member.name in members:
                    limit = 32 * 1048576 if member.name == "./h6-pg-resolver" else 1048576
                    require(member.name not in found and member.isfile() and 0 < member.size <= limit, "package_member")
                    with tar.extractfile(member) as stream:
                        found[member.name] = sha(stream.read(member.size + 1))
            require(found == members, "package_inspector")
    return result, contents


def runtime_access(metadata, *, executable=False):
    uid, gid, mode = metadata[2], metadata[3], metadata[4]
    permissions = (mode >> 6) & 7 if uid == UID else (mode >> 3) & 7 if gid == GID else mode & 7
    required = 5 if executable else 4
    require(permissions & required == required, "runtime_input_access")


def validate_runtime_inputs(before):
    for name in ("inspector", "connections", "resolver", "arr", "reference", "ca"):
        runtime_access(before[name][0], executable=name in ("inspector", "resolver"))
    for name in TREES:
        for relative, value in before[name].items():
            runtime_access(value if relative.endswith("/") else value[0], executable=relative.endswith("/"))


def output_identity(config, *, empty):
    require(config.output.parent == config.state and config.output.name.startswith("canario-salida-"), "output_path")
    with directory(config.output) as fd:
        info = os.fstat(fd)
        require(info.st_uid == UID and info.st_gid == GID and stat.S_IMODE(info.st_mode) == 0o700, "output_owner")
        require(os.listdir(fd) == ([] if empty else ["plan.json"]), "output_entries")
        return (info.st_dev, info.st_ino, info.st_uid, info.st_gid, stat.S_IMODE(info.st_mode))


def limits():
    for what, value in ((resource.RLIMIT_CPU, 10), (resource.RLIMIT_AS, 512 * 1048576),
                        (resource.RLIMIT_FSIZE, 262144),
                        (resource.RLIMIT_NOFILE, 128)):
        resource.setrlimit(what, (value, value))


def run(argv):
    # Raw diagnostics can contain secrets; they never leave the subprocess.
    result = subprocess.run(argv, env={"PATH": "/usr/bin:/bin", "HOME": "/tmp", "LANG": "C.UTF-8"},
                            stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL if argv[:2] == ["docker", "start"] else subprocess.PIPE,
                            stderr=subprocess.DEVNULL, timeout=65, check=False, preexec_fn=limits)
    require(result.returncode == 0, "command_failed")
    require(result.stdout is None or len(result.stdout) <= 1048576, "command_output_limit")
    return result.stdout or b""


def pg_identity(config, runner):
    data = decode(runner(["docker", "inspect", "--type", "container", config.pgid]))
    require(isinstance(data, list) and len(data) == 1, "pg_inspect")
    data = data[0]
    labels = data.get("Config", {}).get("Labels") or {}
    require(data.get("Id") == config.pgid and data.get("State", {}).get("Running") is True and
            labels.get("vec.recorridos.owner") == "Codex-M" and
            labels.get("vec.recorridos.state") == str(config.state) and
            data.get("HostConfig", {}).get("NetworkMode") == "none", "pg_identity")
    return data["Id"], data.get("Image"), labels, data["HostConfig"]["NetworkMode"]


def image_identity(config, runner):
    data = decode(runner(["docker", "image", "inspect", config.image]))
    require(isinstance(data, list) and len(data) == 1 and data[0].get("Id") == config.image, "image_identity")
    return data[0]["Id"]


def mounts(config):
    p = config.paths
    return [(p["inspector"], "/usr/local/bin/vec-server", False),
            (p["connections"], "/h6-conexiones.py", False), (p["resolver"], "/h6-pg-resolver", False),
            (p["arr"], "/vec-arrancar.sh", False), (p["reference"], "/vec-conexiones-referencia.sh", False),
            (p["ca"], "/vec-pg-ca.crt", False), (p["material"], "/vec-material", False),
            (p["incorporacion"], "/vec-incorporacion", False), (p["public_data"], "/app/h6-public-data", False),
            (p["communications"], "/vec-material/comunicaciones", False), (p["imports"], "/vec-importaciones", False),
            (config.output, "/h6-out", True)]


def create_command(config):
    command = ["docker", "create", "--pull=never", "--network=none", "--user=10002:10002",
               "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64",
               "--memory=512m", "--cpus=1", "--ulimit=fsize=131072:131072", "--log-driver=none",
               "--tmpfs=/tmp:rw,nosuid,nodev,noexec,size=8m", "--tmpfs=/dev/shm:rw,nosuid,nodev,noexec,size=8m",
               "--env=VEC_CT_INCORPORACION_V2_FILE=/vec-incorporacion/servidor.json", "--entrypoint=/bin/bash"]
    for source, target, writable in mounts(config):
        require(not any(char in str(source) for char in ",\n\r"), "mount_path")
        command.extend(["--mount", f"type=bind,src={source},dst={target}" + ("" if writable else ",readonly")])
    return command + [config.image, "/vec-arrancar.sh"]


def validate_canary(config, data, cid, *, stopped=False):
    require(isinstance(data, list) and len(data) == 1, "canary_inspect")
    data = data[0]
    host, process = data.get("HostConfig", {}), data.get("Config", {})
    require(data.get("Id") == cid and data.get("Image") == config.image and process.get("Image") == config.image,
            "canary_identity")
    require(host.get("NetworkMode") == "none" and host.get("ReadonlyRootfs") is True and
            host.get("Privileged") is False and host.get("CapDrop") == ["ALL"] and
            not host.get("CapAdd") and not host.get("DeviceRequests") and not host.get("PortBindings") and
            host.get("PidMode", "") == "" and host.get("IpcMode", "private") == "private" and
            "no-new-privileges" in host.get("SecurityOpt", []) and
            host.get("PidsLimit") == 64 and host.get("Memory") == 512 * 1048576 and
            host.get("NanoCpus") == 1000000000 and host.get("LogConfig", {}).get("Type") == "none" and
            not host.get("Devices") and not host.get("VolumesFrom") and not host.get("Binds"), "canary_isolation")
    require(process.get("User") == "10002:10002" and process.get("Entrypoint") == ["/bin/bash"] and
            process.get("Cmd") == ["/vec-arrancar.sh"], "canary_process")
    require(host.get("Tmpfs") == {"/tmp": "rw,nosuid,nodev,noexec,size=8m", "/dev/shm": "rw,nosuid,nodev,noexec,size=8m"} and
            {"Name": "fsize", "Soft": 131072, "Hard": 131072} in host.get("Ulimits", []), "canary_limits")
    actual = [(m.get("Source"), m.get("Destination"), m.get("RW")) for m in data.get("Mounts", []) if m.get("Type") == "bind"]
    require(len(actual) == len(mounts(config)) and set(actual) == {(str(p), target, rw) for p, target, rw in mounts(config)} and
            all(m.get("Type") in ("bind", "tmpfs") for m in data.get("Mounts", [])), "canary_mounts")
    if stopped:
        state = data.get("State", {})
        require(state.get("Running") is False and state.get("ExitCode") == 0 and state.get("OOMKilled") is False,
                "canary_exit")


def helper_command(scripts, source, destination, lock, helper, arguments):
    # Fixed helper snapshots run without network, host home, sockets or inherited env.
    command = ["bwrap", "--unshare-all", "--die-with-parent", "--new-session", "--cap-drop", "ALL",
               "--ro-bind", "/usr", "/usr", "--symlink", "usr/bin", "/bin",
               "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib64", "/lib64",
               "--proc", "/proc", "--dev", "/dev", "--size", "8388608", "--tmpfs", "/tmp",
               "--ro-bind", str(scripts), "/helpers", "--bind", str(source), "/source",
               "--bind", str(destination), "/destination", "--ro-bind", str(lock), "/release.lock",
               "--clearenv", "--setenv", "PATH", "/usr/bin:/bin", "--setenv", "HOME", "/tmp",
               "--setenv", "LANG", "C.UTF-8", "--chdir", "/tmp", "--",
               "/usr/bin/prlimit", "--nproc=64", "--", "/usr/bin/python3", "-B", "/helpers/" + helper]
    return command + arguments


def write(path, data):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def canary(config, runner=run):
    with directory(config.state) as state_fd:
        info = os.fstat(state_fd)
        require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700, "state_private")
        require(not any((Path(parent) / ".git").exists() for parent in (config.state, *config.state.parents)), "state_in_git")
        fcntl.flock(state_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        require(not {"plan-conexiones.json", "plan-canonico-clon.json"} & set(os.listdir(state_fd)), "existing_plan")
        before, contents = snapshot(config)
        validate_runtime_inputs(before)
        output_before = output_identity(config, empty=True)
        pg_before = pg_identity(config, runner)
        image_identity(config, runner)
        cid = None
        try:
            cid = runner(create_command(config)).decode("ascii").strip()
            require(HEX.fullmatch(cid) is not None and cid != config.pgid, "canary_id")
            validate_canary(config, decode(runner(["docker", "inspect", "--type", "container", cid])), cid)
            runner(["docker", "start", "--attach", cid])
            validate_canary(config, decode(runner(["docker", "inspect", "--type", "container", cid])), cid, stopped=True)
        finally:
            if cid and HEX.fullmatch(cid) and cid != config.pgid:
                runner(["docker", "rm", "--force", cid])
        require(snapshot(config)[0] == before, "inputs_changed")
        require(pg_identity(config, runner) == pg_before, "pg_changed")
        image_identity(config, runner)
        require(output_identity(config, empty=False) == output_before, "output_changed")
        with directory(config.output) as output_fd:
            plan, output_sha, metadata = read_at(output_fd, "plan.json", 131072, owner=UID, modes=(0o600,))
        require(metadata[3] == GID, "output_group")
        parsed = decode(plan)
        require(isinstance(parsed, dict) and set(parsed) == {"conexiones", "huellas"} and
                isinstance(parsed["conexiones"], list) and isinstance(parsed["huellas"], dict) and parsed["huellas"], "output_json")
        # Both helpers write into fresh private scratch. No partial canonical receipt.
        with tempfile.TemporaryDirectory(prefix=".canario-trusted-", dir=config.state) as scratch:
            root = Path(scratch)
            scripts, source, destination = root / "helpers", root / "source", root / "destination"
            for path in (scripts, source, destination):
                path.mkdir(mode=0o700)
            for key in ("connections", "promoter", "receipt_helper"):
                write(scripts / HELPERS[key], contents[key])
            write(root / "lock", contents["lock"])
            write(source / "plan.json", plan)
            runner(helper_command(scripts, source, destination, root / "lock", HELPERS["promoter"], ["/source", "/destination"]))
            require(os.listdir(source) == [] and os.listdir(destination) == ["plan-conexiones.json"], "promotion_entries")
            promoted, digest, _ = read(destination / "plan-conexiones.json", 131072, owner=os.getuid(), modes=(0o600,))
            require(digest == output_sha and promoted == plan, "promotion_changed")
            runner(helper_command(scripts, source, destination, root / "lock", HELPERS["receipt_helper"],
                ["clon", "/destination/plan-conexiones.json", output_sha, config.pgid, config.image,
                 config.approved["arr"], "/release.lock", "/destination/plan-canonico-clon.json"]))
            require(sorted(os.listdir(destination)) == ["plan-canonico-clon.json", "plan-conexiones.json"], "receipt_entries")
            receipt, receipt_sha, _ = read(destination / "plan-canonico-clon.json", 8192, owner=os.getuid(), modes=(0o600,))
            expected = {"version": 1, "kind": "clon", "package_sha256": config.approved["package"], "source_commit": SOURCE,
                        "pg_container_id": config.pgid, "plan_sha256": output_sha,
                        "material_inventory_sha256": sha(canonical(decode(plan)["huellas"])),
                        "canary_image_id": config.image, "arranque_sha256": config.approved["arr"], "canary_output_sha256": output_sha}
            require(receipt == canonical(expected), "receipt_binding")
            require(snapshot(config)[0] == before and pg_identity(config, runner) == pg_before, "identity_changed")
            image_identity(config, runner)
            with directory(config.output) as output_fd:
                require(output_identity(config, empty=False) == output_before, "output_changed")
                final_plan, final_sha, final_metadata = read_at(output_fd, "plan.json", 131072, owner=UID, modes=(0o600,))
                require(final_plan == plan and final_sha == output_sha and final_metadata == metadata, "output_changed")
                os.unlink("plan.json", dir_fd=output_fd)
            os.rmdir(config.output.name, dir_fd=state_fd)
            created = []
            try:
                for name, data in (("plan-conexiones.json", plan), ("plan-canonico-clon.json", receipt)):
                    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=state_fd)
                    created.append(name)
                    with os.fdopen(fd, "wb") as stream:
                        stream.write(data)
                        stream.flush()
                        os.fsync(stream.fileno())
                os.fsync(state_fd)
            except Exception:
                for name in created:
                    os.unlink(name, dir_fd=state_fd)
                raise
        return {"kind": "clon", "canary_exit_code": 0, "plan_sha256": output_sha, "receipt_sha256": receipt_sha,
                "pg_container_id": config.pgid, "canary_image_id": config.image}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("state", "output", "pgid", "image"):
        parser.add_argument("--" + name, required=True)
    for name in FILES + TREES:
        parser.add_argument("--" + name.replace("_", "-"), required=True, type=Path)
    for name in FILES:
        parser.add_argument("--" + name.replace("_", "-") + "-sha256", required=True)
    args = parser.parse_args()
    config = Config(Path(args.state), Path(args.output), args.pgid, args.image,
                    {name: getattr(args, name) for name in FILES + TREES},
                    {name: getattr(args, name + "_sha256") for name in FILES})
    try:
        print(json.dumps(canary(config), sort_keys=True))
        return 0
    except (Refused, OSError, ValueError, KeyError, TypeError, tarfile.TarError, subprocess.SubprocessError):
        print(json.dumps({"status": "refused", "code": "h6_offline_canary_unverified"}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
