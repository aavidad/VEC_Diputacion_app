"""Local Docker CLI transport for H6 archives; volume evidence survives failures.

Only a pinned image and the caller's approved archive inventory are consumed.
The caller owns approval, publication and eventual retirement of the named volume.
No host directory is mounted into a container. No API removes a volume or forces
container removal. Docker diagnostics are discarded to avoid leaking inputs.
"""
from __future__ import annotations

from contextlib import contextmanager
import io
import json
import os
from pathlib import Path
import resource
import secrets
import selectors
import stat
import subprocess
import tarfile
import time

import clon_h6_archive as archive

DOCKER = "/usr/bin/docker"
SOCKET = "unix:///var/run/docker.sock"
STATE_LABEL = "vec.h6.archive.state"
ROLE_LABEL = "vec.h6.archive.role"
OUTPUT_DIR = "/h6-out"
OUTPUT_FILE = OUTPUT_DIR + "/plan.json"
ARCHIVE_LIMIT = archive.INPUT_LIMIT + archive.MEMBER_LIMIT * 1024 + 10240
EXPORT_LIMIT = archive.OUTPUT_LIMIT + 12800
JSON_LIMIT = 1048576
TIMEOUT = 65


def require(condition, code):
    archive.require(condition, code)


def decode(data):
    def unique(pairs):
        result = {}
        for name, value in pairs:
            require(name not in result, "duplicate_json_key")
            result[name] = value
        return result
    try:
        return json.loads(data, object_pairs_hook=unique)
    except (ValueError, UnicodeError) as error:
        raise archive.Refused("docker_json") from error


def cli_limits():
    # The trusted Go CLI reserves virtual address space beyond its resident heap.
    # RLIMIT_AS would prevent startup; bound CPU, files, descriptors and wall time.
    for kind, limit in ((resource.RLIMIT_CPU, 10),
                        (resource.RLIMIT_NOFILE, 128), (resource.RLIMIT_FSIZE, 1048576),
                        (resource.RLIMIT_CORE, 0)):
        resource.setrlimit(kind, (limit, limit))


class LocalDockerCLI:
    """Fixed local socket, private empty config, no shell, TTY or ambient env."""
    def __init__(self, config):
        self.config = Path(config)
        with archive.directory(self.config) as fd:
            info = os.fstat(fd)
            require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
                    and not os.listdir(fd), "docker_config_private_empty")
            self.config_identity = (info.st_dev, info.st_ino)

    def execute(self, arguments, *, input_bytes=None, output_limit=JSON_LIMIT, timeout=TIMEOUT):
        require(type(output_limit) is int and 0 < output_limit <= ARCHIVE_LIMIT and
                type(timeout) is int and 0 < timeout <= TIMEOUT, "docker_limits")
        require(input_bytes is None or isinstance(input_bytes, bytes) and
                len(input_bytes) <= ARCHIVE_LIMIT, "docker_input_limit")
        with archive.directory(self.config) as fd:
            info = os.fstat(fd)
            require((info.st_dev, info.st_ino) == self.config_identity and
                    info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
                    and not os.listdir(fd), "docker_config_changed")
            argv = [DOCKER, "--host", SOCKET, "--config", str(self.config), *arguments]
            process = subprocess.Popen(argv, stdin=subprocess.PIPE if input_bytes is not None else subprocess.DEVNULL,
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, shell=False, close_fds=True,
                env={"PATH": "/usr/bin:/bin", "HOME": str(self.config), "LANG": "C.UTF-8"},
                cwd=str(self.config), start_new_session=True, preexec_fn=cli_limits)
            try:
                return self._collect(process, input_bytes, output_limit, timeout)
            except BaseException:
                # Killing the client never claims that Docker's effect was cancelled.
                if process.poll() is None:
                    process.kill()
                process.wait(timeout=5)
                raise
            finally:
                process.stdout.close()
                if process.stdin is not None:
                    process.stdin.close()

    @staticmethod
    def _collect(process, payload, limit, timeout):
        result, sent = bytearray(), 0
        deadline = time.monotonic() + timeout
        with selectors.DefaultSelector() as selector:
            os.set_blocking(process.stdout.fileno(), False)
            selector.register(process.stdout, selectors.EVENT_READ)
            if payload is not None:
                os.set_blocking(process.stdin.fileno(), False)
                selector.register(process.stdin, selectors.EVENT_WRITE)
            while selector.get_map():
                remaining = deadline - time.monotonic()
                require(remaining > 0, "docker_timeout_effect_unknown")
                for key, _ in selector.select(remaining):
                    if key.fileobj is process.stdout:
                        chunk = os.read(process.stdout.fileno(), min(65536, limit - len(result) + 1))
                        require(len(result) + len(chunk) <= limit, "docker_output_limit_effect_unknown")
                        if chunk:
                            result.extend(chunk)
                        else:
                            selector.unregister(process.stdout)
                    elif sent == len(payload):
                        selector.unregister(process.stdin)
                        process.stdin.close()
                    else:
                        try:
                            sent += os.write(process.stdin.fileno(), payload[sent:sent + 65536])
                        except BrokenPipeError as error:
                            raise archive.Refused("docker_input_effect_unknown") from error
        remaining = deadline - time.monotonic()
        require(remaining > 0, "docker_timeout_effect_unknown")
        try:
            code = process.wait(timeout=remaining)
        except subprocess.TimeoutExpired as error:
            raise archive.Refused("docker_timeout_effect_unknown") from error
        require(code == 0, "docker_failed_effect_unknown")
        return bytes(result)


class DockerArchiveDriver:
    """Transport restricted to container IDs created by this driver instance."""
    def __init__(self, cli):
        self.cli = cli
        self.owned = {}
        self.roles = {}
        self.states = {}

    def inspect(self, container_id):
        require(container_id in self.owned, "foreign_container")
        return self._inspect(container_id)

    def _inspect(self, container_id):
        require(archive.HEX.fullmatch(container_id), "container_id")
        data = decode(self.cli.execute(["inspect", "--type", "container", container_id]))
        require(isinstance(data, list) and len(data) == 1 and isinstance(data[0], dict), "container_inspect")
        return data[0]

    def put_archive(self, container_id, path, stream):
        require(self.roles.get(container_id) == "staging" and path == archive.STAGE_PATH, "archive_target")
        archive.validate_container(self, self.owned[container_id], canary=False)
        data = stream.read(ARCHIVE_LIMIT + 1)
        require(isinstance(data, bytes) and 0 < len(data) <= ARCHIVE_LIMIT and not stream.read(1), "archive_input_limit")
        self.cli.execute(["cp", "-a", "-", container_id + ":" + path], input_bytes=data)

    @contextmanager
    def get_archive(self, container_id, path):
        require(self.roles.get(container_id) == "canary" and path == OUTPUT_FILE, "archive_source")
        self.validate_canary(self.owned[container_id], exited=True)
        data = self.cli.execute(["cp", "-a", container_id + ":" + path, "-"], output_limit=EXPORT_LIMIT)
        with io.BytesIO(data) as stream:
            yield stream

    def register(self, owned, role, state):
        require(owned.id not in self.owned and role in ("staging", "canary"), "container_duplicate")
        self.owned[owned.id], self.roles[owned.id], self.states[owned.id] = owned, role, state

    def validate_canary(self, owned, *, exited=False):
        data = self.inspect(owned.id)
        validate_common(data, owned, self.states[owned.id], "canary")
        config, host, state = (data.get(key, {}) for key in ("Config", "HostConfig", "State"))
        require(config.get("User") == "10002:10002" and config.get("Entrypoint") == ["/bin/bash"] and
                config.get("Cmd") == ["/vec-arrancar.sh"] and config.get("Tty") is False and
                config.get("OpenStdin") is False, "canary_process")
        require(host.get("Tmpfs") == {"/tmp": "rw,nosuid,nodev,noexec,size=8m",
                "/dev/shm": "rw,nosuid,nodev,noexec,size=8m"} and
                host.get("Ulimits") == [{"Name": "fsize", "Soft": 131072, "Hard": 131072}], "canary_limits")
        require(state.get("Status") == ("exited" if exited else "created"), "canary_state")
        if exited:
            require(type(state.get("ExitCode")) is int and state["ExitCode"] == 0 and
                    state.get("OOMKilled") is False, "canary_exit")
        return data


def validate_common(data, owned, state_path, role):
    config, host, state = (data.get(key, {}) for key in ("Config", "HostConfig", "State"))
    labels = config.get("Labels") or {}
    require(data.get("Id") == owned.id and data.get("Image") == owned.image and
            config.get("Image") == owned.image and labels.get(archive.OWNER_LABEL) == owned.owner and
            labels.get(STATE_LABEL) == state_path and labels.get(ROLE_LABEL) == role, "container_identity")
    require(host.get("NetworkMode") == "none" and host.get("ReadonlyRootfs") is True and
            host.get("Privileged") is False and host.get("CapDrop") == ["ALL"] and not host.get("CapAdd") and
            host.get("SecurityOpt") == ["no-new-privileges"] and host.get("PidMode", "") == "" and
            host.get("IpcMode") == "private" and host.get("PidsLimit") == 64 and
            host.get("Memory") == 512 * 1048576 and host.get("NanoCpus") == 1000000000 and
            host.get("LogConfig", {}).get("Type") == "none" and not host.get("Binds") and
            not host.get("VolumesFrom") and not host.get("Devices") and not host.get("DeviceRequests") and
            not host.get("PortBindings") and host.get("RestartPolicy") == {"Name": "no", "MaximumRetryCount": 0} and
            host.get("AutoRemove") is False, "container_isolation")
    require(state.get("Running") is False and state.get("Paused") is not True and
            state.get("Restarting") is not True, "container_running")
    expected = sorted((m.name, m.destination, m.writable) for m in owned.mounts)
    actual = data.get("Mounts")
    require(isinstance(actual, list) and len(actual) == len(expected) and
            all(m.get("Type") == "volume" and type(m.get("RW")) is bool for m in actual) and
            sorted((m.get("Name"), m.get("Destination"), m.get("RW")) for m in actual) == expected, "container_mounts")
    configured = host.get("Mounts")
    require(isinstance(configured, list) and len(configured) == len(owned.mounts), "container_subpaths")
    for mount in owned.mounts:
        matching = [m for m in configured if m.get("Target") == mount.destination]
        require(len(matching) == 1, "container_subpaths")
        item = matching[0]
        require(item.get("Type") == "volume" and item.get("Source") == mount.name and
                item.get("ReadOnly") is (not mount.writable) and
                (item.get("VolumeOptions") or {}).get("NoCopy") is True and
                (item.get("VolumeOptions") or {}).get("Subpath", "") == mount.subpath, "container_subpaths")


class ArchiveSession:
    """Single attempt, private state evidence, unique named volume never deleted."""
    def __init__(self, state, image, *, cli=None):
        self.state, self.image = Path(state), image
        require(archive.IMAGE.fullmatch(image), "image_pin")
        self.owner = "h6-" + secrets.token_hex(16)
        self.volume = self.owner + "-archive"
        self.stage_name, self.canary_name = self.owner + "-stage", self.owner + "-canary"
        self.staging = self.canary = None
        self.failed = False
        self.imported = False
        self.started = False
        self.evidence = {"version": 1, "owner": self.owner, "image": image, "volume": self.volume,
                         "stage_name": self.stage_name, "canary_name": self.canary_name,
                         "phase": "prepared", "requires_new_session": True}
        with archive.directory(self.state) as fd:
            info = os.fstat(fd)
            require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700, "state_private")
            os.mkdir(self.owner, 0o700, dir_fd=fd)
        self.private = self.state / self.owner
        with archive.directory(self.private) as fd:
            os.mkdir("docker-config", 0o700, dir_fd=fd)
        self._record("prepared")
        self.driver = DockerArchiveDriver(cli if cli is not None else LocalDockerCLI(self.private / "docker-config"))

    def _record(self, phase):
        data = (json.dumps({**self.evidence, "phase": phase}, sort_keys=True) + "\n").encode()
        with archive.directory(self.private) as fd:
            name = "evidence-" + secrets.token_hex(8) + ".json"
            leaf = os.open(name, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=fd)
            with os.fdopen(leaf, "wb") as output:
                output.write(data)
                output.flush()
                os.fsync(output.fileno())
            os.fsync(fd)
        # Completion becomes observable only after both fsync calls succeed.
        self.evidence["phase"] = phase

    @contextmanager
    def _operation(self, phase):
        require(not self.failed, "session_failed_new_state_required")
        try:
            self._record(phase + "_intent")
            yield
            self._record(phase + "_complete")
        except BaseException:
            self.failed = True
            failure = phase + "_failed_effect_unknown"
            self.evidence["phase"] = failure
            try:
                self._record(failure)
            except BaseException:
                # A broken evidence writer never enables cleanup or a retry.
                # The prior intent and named resources remain for reconciliation.
                pass
            raise

    def _labels(self, role):
        return ["--label", archive.OWNER_LABEL + "=" + self.owner, "--label", STATE_LABEL + "=" + str(self.state),
                "--label", ROLE_LABEL + "=" + role]

    def _base(self, name, role):
        return ["create", "--name", name, "--pull=never", "--network=none", "--user=10002:10002",
                "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64",
                "--memory=512m", "--cpus=1", "--log-driver=none", "--ipc=private", "--restart=no", *self._labels(role)]

    def create_staging(self):
        require(self.staging is None, "staging_already_created")
        with self._operation("create_staging"):
            image = decode(self.driver.cli.execute(["image", "inspect", self.image]))
            require(isinstance(image, list) and len(image) == 1 and image[0].get("Id") == self.image, "image_identity")
            result = self.driver.cli.execute(["volume", "create", "--driver", "local", *self._labels("archive"), self.volume])
            require(result.decode("ascii").strip() == self.volume, "volume_identity")
            volumes = decode(self.driver.cli.execute(["volume", "inspect", self.volume]))
            require(isinstance(volumes, list) and len(volumes) == 1 and volumes[0].get("Name") == self.volume and
                    volumes[0].get("Driver") == "local" and volumes[0].get("Scope") == "local" and
                    not volumes[0].get("Options") and volumes[0].get("Labels") == {
                        archive.OWNER_LABEL: self.owner, STATE_LABEL: str(self.state), ROLE_LABEL: "archive"}, "volume_inspect")
            mount = archive.VolumeMount(self.volume, archive.STAGE_PATH, True)
            cid = self._create(self._base(self.stage_name, "staging") + ["--entrypoint=/bin/false", "--mount",
                self._mount(mount), self.image], (mount,), "staging")
            self.staging = self.driver.owned[cid]
            archive.validate_container(self.driver, self.staging, canary=False)
            validate_common(self.driver.inspect(cid), self.staging, str(self.state), "staging")
        return self.staging

    def _create(self, command, mounts, role):
        cid = self.driver.cli.execute(command).decode("ascii").strip()
        require(archive.HEX.fullmatch(cid) and cid not in self.driver.owned, "container_create_id")
        self.evidence[role + "_id"] = cid
        self._record(role + "_created")
        self.driver.register(archive.OwnedContainer(cid, self.image, self.owner, mounts), role, str(self.state))
        return cid

    @staticmethod
    def _mount(mount):
        return "type=volume,src=" + mount.name + ",dst=" + mount.destination + ",volume-nocopy" + (
            ",volume-subpath=" + mount.subpath if mount.subpath else "") + ("" if mount.writable else ",readonly")

    def stage(self, entries, *, directories=()):
        require(self.staging is not None and not self.imported, "staging_required")
        with self._operation("import"):
            result = archive.import_inputs(self.driver, self.staging, entries, directories=directories)
            self.evidence["input_archive_sha256"] = result
            # A root-owned extraction must fail before the UID10002 canary starts.
            # Inspect the directory's archive metadata, without host extraction.
            try:
                payload = self.driver.cli.execute(["cp", "-a", self.staging.id + ":/h6-stage/output", "-"],
                    output_limit=12800)
                stream = io.BytesIO(payload)
                header = archive.read_exact(stream, 512)
                info = tarfile.TarInfo.frombuf(header, "ascii", "strict")
                require(info.name in ("output", "output/") and info.type == tarfile.DIRTYPE and
                        not info.linkname and info.size == 0 and info.uid == archive.UID and
                        info.gid == archive.GID and info.mode == 0o700, "docker_uid10002_no_go")
                require(archive.read_exact(stream, 1024) == bytes(1024) and not any(stream.read()),
                        "docker_uid10002_no_go")
            except (archive.Refused, tarfile.TarError, UnicodeError, ValueError, OSError) as error:
                raise archive.Refused("docker_uid10002_no_go") from error
            self.imported = True
        return result

    def create_canary(self, mounts):
        mounts = tuple(mounts)
        require(self.imported and self.canary is None, "import_required")
        require(0 < len(mounts) <= 32 and len({m.destination for m in mounts}) == len(mounts), "mount_inventory")
        for mount in mounts:
            require(mount.name == self.volume and type(mount.writable) is bool and
                    mount.destination.startswith("/") and archive.closed_name(mount.destination[1:]) and
                    archive.closed_name(mount.subpath), "mount_inventory")
            require((mount.destination == OUTPUT_DIR and mount.subpath == "output" and mount.writable) or
                    (not mount.writable and (mount.subpath == "input" or mount.subpath.startswith("input/"))), "mount_policy")
        require(sum(m.writable for m in mounts) == 1, "mount_output")
        with self._operation("create_canary"):
            command = self._base(self.canary_name, "canary") + ["--ulimit=fsize=131072:131072",
                "--tmpfs=/tmp:rw,nosuid,nodev,noexec,size=8m", "--tmpfs=/dev/shm:rw,nosuid,nodev,noexec,size=8m",
                "--env=VEC_CT_INCORPORACION_V2_FILE=/vec-incorporacion/servidor.json", "--entrypoint=/bin/bash"]
            for mount in mounts:
                command += ["--mount", self._mount(mount)]
            cid = self._create(command + [self.image, "/vec-arrancar.sh"], mounts, "canary")
            self.canary = self.driver.owned[cid]
            self.driver.validate_canary(self.canary)
        return self.canary

    def start_canary(self):
        require(self.canary is not None and not self.started, "canary_already_started_or_missing")
        with self._operation("run_canary"):
            self.driver.validate_canary(self.canary)
            self.started = True
            self.driver.cli.execute(["start", "--attach", self.canary.id], output_limit=1)
            self.driver.validate_canary(self.canary, exited=True)

    def export_output(self, scratch):
        require(self.started and self.canary is not None, "canary_run_required")
        with self._operation("export"):
            self.driver.validate_canary(self.canary, exited=True)
            result = archive.export_output(self.driver, self.canary, scratch, output_mount=OUTPUT_DIR)
            self.evidence["plan_sha256"] = result
        return result

    def cleanup_stopped(self):
        """Explicit cleanup only after successful completion; preserve all failures."""
        require(not self.failed and self.evidence["phase"] == "export_complete", "cleanup_requires_export")
        with self._operation("cleanup"):
            for owned in (self.canary, self.staging):
                if owned is None:
                    continue
                data = self.driver.inspect(owned.id)
                validate_common(data, owned, str(self.state), self.driver.roles[owned.id])
                require(data.get("State", {}).get("Status") in ("created", "exited"), "cleanup_stopped")
                self.driver.cli.execute(["rm", owned.id])
        # Named volume and private evidence are retained even on success.
