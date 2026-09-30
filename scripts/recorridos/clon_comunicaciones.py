#!/usr/bin/env python3
"""Prepare the owned clone's synthetic STARTTLS sink and Users selectors.

No VEC process, PostgreSQL statement, role or shared manifest is changed.
The material driver consumes env/profiles/blockers and seals private files.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import errno
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import smtplib
import resource
import re
import select
import signal
import socket
import ssl
import stat
import subprocess
import tempfile
import time

OWNER = "Codex-M"
STATE = Path.home() / ".local/state/vec-recorridos-codexm-20260930"
CLI_STATE = Path.home() / ".local/state/vec-recorridos"
PG_CONTAINER = "vec-codexm-recorridos-20260930"
MAIL_CONTAINER = "vec-codexm-recorridos-mailpit-20260930"
NETWORK = "vec-codexm-recorridos-mailpit-red-20260930"
IMAGE = "axllent/mailpit:v1.27.8"
SMTP_PORT, HTTP_PORT = 11025, 18532
LABEL_OWNER = "vec.recorridos.owner"
LABEL_STATE = "vec.recorridos.state"
SELECTORS = {"VEC_USUARIOS_PREFERENCIAS_ENABLED": "true",
             "VEC_USUARIOS_IMAGEN_ENABLED": "true"}
SMTP_FLAGS = ["--disable-version-check", "--block-remote-css-and-fonts",
              "--smtp-disable-rdns", "--smtp-require-starttls",
              "--smtp-tls-cert", "/tls/servidor.crt", "--smtp-tls-key", "/tls/servidor.key",
              "--smtp-allowed-recipients", r"^[A-Za-z0-9._+\-]+@example\.test$",
              "--smtp", "0.0.0.0:1025", "--listen", "0.0.0.0:8025",
              "--database", "/data/mailpit.db", "--max", "100", "--quiet"]


class PreparationError(RuntimeError):
    pass


def _run(args: list[str], *, missing_ok=False) -> bytes | None:
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            timeout=30, check=False,
                            env={"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "LANG": "C"})
    if result.returncode:
        if missing_ok and (b"No such" in result.stderr or b"not found" in result.stderr) and result.stdout.strip() == b"[]":
            return None
        # Never return engine, certificate or SMTP diagnostics with private data.
        raise PreparationError("local_command_failed:" + Path(args[0]).name)
    return result.stdout


def _canonical(path: Path) -> Path:
    path = Path(path).expanduser().absolute()
    if path != path.resolve():
        raise PreparationError("symlink_or_noncanonical_path")
    return path


def _private(path: Path, limit=262144) -> bytes:
    _canonical(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1
                or info.st_uid != os.getuid() or info.st_mode & 0o077
                or not 0 < info.st_size <= limit):
            raise PreparationError("unsafe_private_file:" + path.name)
        return stream.read(limit + 1)


def _directory(path: Path) -> None:
    _canonical(path)
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    if not path.is_dir() or path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o077:
        raise PreparationError("unsafe_private_directory")


def _write_new(path: Path, data: bytes) -> None:
    _canonical(path.parent)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def _inspect(engine: str, kind: str, name: str) -> dict | None:
    raw = _run([engine, kind, "inspect", name], missing_ok=True)
    if raw is None:
        return None
    result = json.loads(raw)
    if not isinstance(result, list) or len(result) != 1 or not isinstance(result[0], dict):
        raise PreparationError("invalid_engine_response")
    return result[0]


def _labels(state: Path) -> dict:
    return {LABEL_OWNER: OWNER, LABEL_STATE: str(state)}


def _owned(resource: dict, state: Path, kind: str) -> None:
    labels = resource.get("Labels", {}) if kind == "network" else resource.get("Config", {}).get("Labels", {})
    if any(labels.get(key) != value for key, value in _labels(state).items()):
        raise PreparationError("foreign_" + kind)


def _free_ports(ports=None) -> None:
    sockets = []
    try:
        for port in ports or (SMTP_PORT, HTTP_PORT):
            listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sockets.append(listener)
            listener.bind(("127.0.0.1", port))
    except OSError:
        raise PreparationError("mailpit_loopback_port_busy") from None
    finally:
        for listener in sockets:
            listener.close()


def _validate_sink(resource: dict, state: Path, image_id: str, *, allow_stopped=False, target=None) -> None:
    target = target or _default_target()
    _owned(resource, state, "container")
    host = resource.get("HostConfig", {})
    if (resource.get("Image") != image_id or resource.get("Config", {}).get("Cmd") != SMTP_FLAGS
            or not host.get("ReadonlyRootfs") or not host.get("AutoRemove")
            or host.get("NetworkMode") != target["network"]):
        raise PreparationError("mailpit_configuration_mismatch")
    expected = {"1025/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(target["smtp_port"])}],
                "8025/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(target["http_port"])}]}
    if host.get("PortBindings") != expected:
        raise PreparationError("mailpit_nonloopback_bind")
    mounts = {entry.get("Destination"): entry for entry in resource.get("Mounts", [])}
    for destination, source, writable in (
            ("/tls", state / "material/comunicaciones", False),
            ("/data", state / "comunicaciones/buzon", True)):
        entry = mounts.get(destination, {})
        if entry.get("Source") != str(source) or entry.get("RW") is not writable:
            raise PreparationError("mailpit_mount_mismatch")
    if not allow_stopped and not resource.get("State", {}).get("Running"):
        raise PreparationError("mailpit_not_running")


def _source_contract(repo: Path, source_ref: str) -> str:
    if not isinstance(source_ref, str) or not re.fullmatch(r"[0-9a-f]{40}", source_ref):
        raise PreparationError("clone_source_commit_invalid")
    source = _run(["git", "-C", str(repo), "rev-parse", source_ref + "^{commit}"]).decode().strip()
    if source != source_ref:
        raise PreparationError("clone_source_commit_mismatch")
    contracts = (
        ("config/config.go", ["VEC_SMTP_HOST", "VEC_SMTP_PORT", "VEC_SMTP_FROM", "VEC_SMTP_CA_FILE", "VEC_SMTP_MODO_TLS"]),
        ("internal/app/bootstrap/usuarios_correos_material.go", ["VEC_USUARIOS_CORREOS_ENABLED"]),
        ("internal/app/bootstrap/usuarios_imagen_material.go", ["VEC_USUARIOS_IMAGEN_ENABLED"]),
        ("internal/app/bootstrap/correo_llamamiento_smtp_desarrollo.go", ["smtp.STARTTLSObligatorio", "cfg.SMTPCAFile"]),
    )
    for filename, required in contracts:
        content = _run(["git", "-C", str(repo), "show", source + ":" + filename]).decode()
        if any(value not in content for value in required):
            raise PreparationError("source_contract_mismatch")
    return source


def _clone_source(state: Path) -> str:
    clone = json.loads(_private(state / "clon.json"))
    source = clone.get("commit")
    if not isinstance(source, str) or not re.fullmatch(r"[0-9a-f]{40}", source):
        raise PreparationError("clone_source_commit_invalid")
    return source


def _scope(repo, container, state, material, pg_port, engine, *, require_postgres=True) -> tuple[Path, Path, Path]:
    repo, state, material = map(_canonical, (repo, state, material))
    if (material != state / "material" or not isinstance(container, str)
            or not re.fullmatch(r"vec-[a-z0-9-]{1,100}", container)
            or type(pg_port) is not int or not 1024 < pg_port < 65536 or engine != "docker"
            or state == Path.home() or any(character in str(state) for character in (",", "\n", "\r", "\0"))
            or any((p / ".git").exists() for p in (state, *state.parents))):
        raise PreparationError("owned_clone_scope_required")
    info = state.stat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise PreparationError("unsafe_private_directory")
    clone = json.loads(_private(state / "clon.json"))
    if clone.get("propietario") != OWNER or clone.get("contenedor") != container or clone.get("puerto_pg") != pg_port:
        raise PreparationError("clone_owner_mismatch")
    postgres = _inspect(engine, "container", container)
    if postgres is None and require_postgres:
        raise PreparationError("owned_postgres_container_missing")
    if postgres is not None:
        _owned(postgres, state, "container")
    return repo, state, material


def _default_target() -> dict:
    return _build_target(STATE, PG_CONTAINER, 55531, SMTP_PORT, HTTP_PORT)


def _build_target(state: Path, container: str, pg_port: int, smtp_port: int, http_port: int) -> dict:
    if (not isinstance(container, str) or not re.fullmatch(r"vec-[a-z0-9-]{1,100}", container)
            or type(pg_port) is not int or not 1024 < pg_port < 65536
            or type(smtp_port) is not int or type(http_port) is not int
            or not 1024 < smtp_port < 65536 or not 1024 < http_port < 65536
            or len({smtp_port, http_port, pg_port}) != 3):
        raise PreparationError("mailpit_ports_invalid")
    legacy = state == STATE and container == PG_CONTAINER and pg_port == 55531
    suffix = hashlib.sha256((str(state) + "\0" + container + "\0" + str(pg_port)).encode()).hexdigest()[:24]
    return {"version": 1, "owner": OWNER, "state": str(state), "pg_container": container, "pg_port": pg_port,
            "container": MAIL_CONTAINER if legacy else "vec-codexm-mailpit-" + suffix,
            "network": NETWORK if legacy else "vec-codexm-mailpit-red-" + suffix,
            "smtp_host": "127.0.0.1", "smtp_port": smtp_port, "http_port": http_port}


def _target(state: Path, container: str, pg_port: int, smtp_port=None, http_port=None) -> dict:
    path = state / "comunicaciones/target.json"
    if path.exists() or path.is_symlink():
        stored = json.loads(_private(path))
        if not isinstance(stored, dict):
            raise PreparationError("mailpit_target_invalid")
        expected = _build_target(state, container, pg_port, stored.get("smtp_port"), stored.get("http_port"))
        if stored != expected:
            raise PreparationError("mailpit_target_invalid")
        if ((smtp_port is not None and smtp_port != stored["smtp_port"])
                or (http_port is not None and http_port != stored["http_port"])):
            raise PreparationError("mailpit_target_changed")
        target = stored
    else:
        target = _build_target(state, container, pg_port, SMTP_PORT if smtp_port is None else smtp_port,
                               HTTP_PORT if http_port is None else http_port)
    marker = state / "clon.json"
    if marker.exists() or marker.is_symlink():
        clone = json.loads(_private(marker))
        if clone.get("puerto_web") in (target["smtp_port"], target["http_port"]):
            raise PreparationError("mailpit_port_conflicts_with_application")
    return target


def configure(repo, container, state, material, pg_port, engine, *, smtp_port=None, http_port=None) -> dict:
    """Persist ports/resource coordinates after cloning, before H4 material exists."""
    repo, state, material = _scope(repo, container, state, material, pg_port, engine)
    with _proxy_lock(state):
        target = _target(state, container, pg_port, smtp_port, http_port)
        network = _inspect(engine, "network", target["network"])
        if network:
            _owned(network, state, "network")
            if network.get("Internal") is not True:
                raise PreparationError("mailpit_network_not_isolated")
        sink = _inspect(engine, "container", target["container"])
        if sink:
            image = _inspect(engine, "image", IMAGE)
            if image is None or not network:
                raise PreparationError("mailpit_target_dependencies_missing")
            _validate_sink(sink, state, image["Id"], target=target)
        else:
            _free_ports((target["smtp_port"], target["http_port"]))
        path = state / "comunicaciones/target.json"
        if not path.exists():
            _write_new(path, (json.dumps(target, indent=2) + "\n").encode())
        result = _result(target=target)
        result["profiles"]["usuarios_comunicaciones"]["target_configured"] = True
        result["files"] = [str(path)]
        return result


def _result(*, source="", image=False, smtp=False, error="", target=None) -> dict:
    target = target or _default_target()
    env = dict(SELECTORS) if image else {}
    if smtp:
        env.update({"VEC_USUARIOS_CORREOS_ENABLED": "true", "VEC_SMTP_HOST": "127.0.0.1",
                    "VEC_SMTP_PORT": str(target["smtp_port"]), "VEC_SMTP_FROM": "rrhh@example.test",
                    "VEC_SMTP_CA_FILE": str(Path(target["state"]) / "material/ca/ca.crt"), "VEC_SMTP_MODO_TLS": "starttls"})
    profiles = {"usuarios_comunicaciones": {"source_commit": source, "imagen_selector_prepared": image,
                 "smtp_ready": smtp, **target, "mailpit_container": target["container"],
                 "mailpit_origin": "http://127.0.0.1:" + str(target["http_port"]),
                 "smtp_scope": "synthetic_local_sink", "corporate_delivery": False,
                 "application_started": False}}
    blockers = [] if not error else [{"code": "smtp_sintetico_no_preparado", "profile": "usuarios_correos", "detail": error}]
    return {"env": env, "profiles": profiles, "blockers": blockers}


def preflight(repo, container, state, material, pg_port, engine, *, smtp_port=None, http_port=None) -> dict:
    """Read contracts and owned resources; never create files or start processes."""
    repo, state, material = _scope(repo, container, state, material, pg_port, engine)
    target = _target(state, container, pg_port, smtp_port, http_port)
    source = _source_contract(repo, _clone_source(state))
    proof = json.loads(_private(state / "usuarios-h4-result.json"))
    if (proof.get("roles_version") != 2 or proof.get("assignments_version") != 2
            or proof.get("surfaces") != 2 or proof.get("grants_per_role") != 10):
        return _result(source=source, error="h4_roles_profiles_unverified", target=target)
    try:
        if not hasattr(os, "pidfd_open") or not hasattr(signal, "pidfd_send_signal"):
            raise PreparationError("mailpit_pidfd_unavailable")
        own_descriptor = os.pidfd_open(os.getpid())
        os.close(own_descriptor)
        _private(material / "ca/ca.crt")
        _private(material / "ca/ca.key")
        image = _inspect(engine, "image", IMAGE)
        if image is None:
            raise PreparationError("mailpit_image_not_installed")
        network = _inspect(engine, "network", target["network"])
        if network:
            _owned(network, state, "network")
            if network.get("Internal") is not True:
                raise PreparationError("mailpit_network_not_isolated")
        sink = _inspect(engine, "container", target["container"])
        if sink:
            if not network:
                raise PreparationError("mailpit_network_missing")
            _validate_sink(sink, state, image["Id"], target=target)
        else:
            _free_ports((target["smtp_port"], target["http_port"]))
    except (PreparationError, OSError, ValueError) as error:
        return _result(source=source, image=True, target=target, error=str(error) if isinstance(error, PreparationError) else "private_dependency_unavailable")
    return _result(source=source, image=True, target=target)


def _certificate(material: Path) -> Path:
    tls = material / "comunicaciones"
    _directory(tls)
    cert, key = tls / "servidor.crt", tls / "servidor.key"
    if cert.exists() or key.exists():
        _private(cert)
        _private(key)
    else:
        generated = _run(["openssl", "genpkey", "-algorithm", "RSA", "-pkeyopt", "rsa_keygen_bits:2048"])
        _write_new(key, generated)
        request = tls / "servidor.csr"
        _run(["openssl", "req", "-new", "-key", str(key), "-out", str(request),
              "-subj", "/CN=127.0.0.1"])
        os.chmod(request, 0o600)
        extensions = tls / "servidor.ext"
        _write_new(extensions, b"basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=IP:127.0.0.1,DNS:localhost\n")
        # Explicit random serial: never mutate the H1 CA serial/history.
        _run(["openssl", "x509", "-req", "-in", str(request), "-CA", str(material / "ca/ca.crt"),
              "-CAkey", str(material / "ca/ca.key"), "-set_serial", "0x" + os.urandom(16).hex(),
              "-days", "30", "-sha256", "-extfile", str(extensions), "-out", str(cert)])
        os.chmod(cert, 0o600)
    _run(["openssl", "verify", "-CAfile", str(material / "ca/ca.crt"), "-verify_ip", "127.0.0.1",
          "-purpose", "sslserver", str(cert)])
    cert_public = _run(["openssl", "x509", "-in", str(cert), "-pubkey", "-noout"])
    key_public = _run(["openssl", "pkey", "-in", str(key), "-pubout"])
    if cert_public != key_public:
        raise PreparationError("smtp_certificate_key_mismatch")
    return tls


def _smtp_probe(ca: Path, *, send=False, smtp_port=SMTP_PORT) -> dict:
    context = ssl.create_default_context(cafile=str(ca))
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    with smtplib.SMTP("127.0.0.1", smtp_port, timeout=5) as connection:
        connection.ehlo("localhost")
        if not connection.has_extn("starttls"):
            raise PreparationError("smtp_starttls_missing")
        connection.starttls(context=context)
        connection.ehlo("localhost")
        if not connection.sock or not connection.sock.cipher():
            raise PreparationError("smtp_tls_unverified")
        if send:
            connection.sendmail("rrhh@example.test", ["ensayo@example.test"],
                                "From: rrhh@example.test\r\nTo: ensayo@example.test\r\n"
                                "Subject: Ensayo SMTP local\r\nMessage-ID: <ensayo-codexm@example.test>\r\n"
                                "\r\nEnsayo sintetico del relay local.\r\n")
        # Do not transmit DATA for a disallowed address; RCPT alone proves the guard.
        connection.mail("rrhh@example.test")
        code, _ = connection.rcpt("ensayo@external.invalid")
        connection.rset()
        if code < 500:
            raise PreparationError("smtp_external_recipient_not_rejected")
    return {"starttls_verified": True, "external_recipient_rejected": True, "synthetic_probe_sent": send}


def _proxy_command(address: str, port: int, target: int) -> list[str]:
    ip = ipaddress.IPv4Address(address)
    if not ip.is_private or ip.is_loopback or ip.is_unspecified or ip.is_multicast:
        raise PreparationError("mailpit_network_address_invalid")
    if type(port) is not int or not 1024 < port < 65536 or target not in (1025, 8025):
        raise PreparationError("mailpit_proxy_port_invalid")
    return ["/usr/bin/socat", "-T", "15",
            f"TCP4-LISTEN:{port},bind=127.0.0.1,reuseaddr,fork,max-children=8",
            f"TCP4:{ip}:{target},connect-timeout=5"]


def _proxy_limits() -> None:
    resource.setrlimit(resource.RLIMIT_NOFILE, (64, 64))
    # RLIMIT_NPROC counts every process of the shared UID, including other
    # agents. socat's max-children=8 bounds this proxy without affecting them.
    resource.setrlimit(resource.RLIMIT_FSIZE, (1048576, 1048576))
    resource.setrlimit(resource.RLIMIT_AS, (128 * 1048576, 128 * 1048576))
    resource.setrlimit(resource.RLIMIT_CORE, (0, 0))


def _private_append(path: Path):
    _canonical(path.parent)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    info = os.fstat(fd)
    if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid()
            or stat.S_IMODE(info.st_mode) != 0o600):
        os.close(fd)
        raise PreparationError("unsafe_proxy_log")
    return os.fdopen(fd, "ab")


@contextmanager
def _proxy_lock(state: Path):
    _directory(state / "comunicaciones")
    with _private_append(state / "comunicaciones/lifecycle.lock") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise PreparationError("mailpit_lifecycle_busy") from None
        try:
            yield
        finally:
            fcntl.flock(lock, fcntl.LOCK_UN)


def _process_snapshot(pid: int) -> dict:
    proc = Path("/proc") / str(pid)
    details = proc.joinpath("stat").read_text().rsplit(")", 1)[1].split()
    return {"uid": proc.stat().st_uid, "start_ticks": details[19],
            "command": proc.joinpath("cmdline").read_bytes().rstrip(b"\0").split(b"\0")}


def _pidfd_dead(fd: int, timeout=0) -> bool:
    poller = select.poll()
    poller.register(fd, select.POLLIN)
    return bool(poller.poll(timeout))


def _proxy_guard(data: dict, command: list[str], state: Path) -> tuple[str, int | None]:
    """Return an identity-pinned descriptor; never signal by numeric PID."""
    if not isinstance(data, dict):
        raise PreparationError("mailpit_proxy_record_mismatch")
    pid = data.get("pid")
    if (data.get("owner") != OWNER or data.get("state") != str(state) or data.get("command") != command
            or type(pid) is not int or pid < 2 or not isinstance(data.get("start_ticks"), str)
            or not data["start_ticks"].isdigit()):
        raise PreparationError("mailpit_proxy_record_mismatch")
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return "dead", None
    except OSError as error:
        if error.errno == errno.ESRCH:
            return "dead", None
        raise PreparationError("mailpit_pidfd_unavailable") from None
    try:
        if _pidfd_dead(fd):
            os.close(fd)
            return "dead", None
        try:
            actual = _process_snapshot(pid)
        except FileNotFoundError:
            if _pidfd_dead(fd):
                os.close(fd)
                return "dead", None
            raise PreparationError("mailpit_proxy_process_unverifiable") from None
        if (actual["command"] != [value.encode() for value in command] or actual["uid"] != os.getuid()
                or actual["start_ticks"] != data["start_ticks"]):
            raise PreparationError("mailpit_proxy_process_mismatch")
        if _pidfd_dead(fd):
            os.close(fd)
            return "dead", None
        return "alive", fd
    except BaseException:
        os.close(fd)
        raise


def _publish_record(path: Path, data: dict, previous: bytes | None) -> None:
    """Replace only the previously validated private record, under the lifecycle lock."""
    content = (json.dumps(data, indent=2) + "\n").encode()
    if previous is None:
        _write_new(path, content)
        return
    if _private(path) != previous:
        raise PreparationError("mailpit_proxy_record_changed")
    fd, name = tempfile.mkstemp(prefix=".proxy-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            os.fchmod(stream.fileno(), 0o600)
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        if _private(path) != previous:
            raise PreparationError("mailpit_proxy_record_changed")
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def _terminate_child(fd: int) -> None:
    if not _pidfd_dead(fd):
        try:
            signal.pidfd_send_signal(fd, signal.SIGTERM)
        except ProcessLookupError:
            pass
        if not _pidfd_dead(fd, timeout=5000):
            raise PreparationError("mailpit_proxy_stop_pending")


def _ensure_proxies(sink: dict, state: Path, *, target=None) -> list[dict]:
    """Docker internal networks have no NAT publications: use owned fixed proxies.

    socat only forwards a loopback listener to the exact internal container IP;
    it has no relay choice, external destination, TLS termination or payload log.
    Existing unrelated listeners/processes are never stopped or reused.
    """
    target = target or _default_target()
    networks = sink.get("NetworkSettings", {}).get("Networks", {})
    if set(networks) != {target["network"]}:
        raise PreparationError("mailpit_extra_network")
    address = networks[target["network"]].get("IPAddress", "")
    records = []
    for port, destination in ((target["smtp_port"], 1025), (target["http_port"], 8025)):
        command = _proxy_command(address, port, destination)
        record = state / "comunicaciones" / f"proxy-{port}.json"
        previous = _private(record) if record.exists() or record.is_symlink() else None
        data = json.loads(previous) if previous is not None else None
        kind, descriptor = _proxy_guard(data, command, state) if previous is not None else ("dead", None)
        if descriptor is not None:
            os.close(descriptor)
        if kind == "dead":
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
                listener.bind(("127.0.0.1", port))
            log = state / "comunicaciones" / f"proxy-{port}.log"
            with _private_append(log) as stream:
                process = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                           stderr=stream, cwd=state / "comunicaciones", close_fds=True,
                                           start_new_session=True, preexec_fn=_proxy_limits,
                                           env={"PATH": "/usr/bin:/bin", "LANG": "C"})
            child = os.pidfd_open(process.pid)
            try:
                time.sleep(0.1)
                if process.poll() is not None:
                    raise PreparationError("mailpit_proxy_start_failed")
                data = {"owner": OWNER, "state": str(state), "pid": process.pid, "command": command,
                        "start_ticks": _process_snapshot(process.pid)["start_ticks"]}
                _publish_record(record, data, previous)
            except BaseException:
                _terminate_child(child)
                raise
            finally:
                os.close(child)
        records.append({"port": port, "pid": data["pid"], "record": str(record)})
    return records


def provision(repo, container, state, material, pg_port, engine, *, smtp_port=None, http_port=None) -> dict:
    """Prepare SMTP only; return selectors to the sole material driver."""
    result = preflight(repo, container, state, material, pg_port, engine, smtp_port=smtp_port, http_port=http_port)
    if result["blockers"]:
        return result
    state = Path(result["profiles"]["usuarios_comunicaciones"]["state"])
    with _proxy_lock(state):
        return _provision_prepared(result, state, state / "material", engine)


def _provision_prepared(result: dict, state, material, engine) -> dict:
    state, material = Path(state), Path(material)
    source = result["profiles"]["usuarios_comunicaciones"]["source_commit"]
    profile = result["profiles"]["usuarios_comunicaciones"]
    target = {key: profile[key] for key in _default_target()}
    try:
        path = state / "comunicaciones/target.json"
        if path.exists() or path.is_symlink():
            if json.loads(_private(path)) != target:
                raise PreparationError("mailpit_target_changed")
        else:
            _write_new(path, (json.dumps(target, indent=2) + "\n").encode())
        tls = _certificate(material)
        _directory(state / "comunicaciones/buzon")
        if _inspect(engine, "network", target["network"]) is None:
            _run([engine, "network", "create", "--internal", "--label", LABEL_OWNER + "=" + OWNER,
                  "--label", LABEL_STATE + "=" + str(state), target["network"]])
        if _inspect(engine, "container", target["container"]) is None:
            _run([engine, "run", "--detach", "--rm", "--name", target["container"],
                  "--label", LABEL_OWNER + "=" + OWNER, "--label", LABEL_STATE + "=" + str(state),
                  "--network", target["network"], "--read-only", "--cap-drop", "ALL",
                  "--security-opt", "no-new-privileges", "--memory", "128m", "--cpus", "0.5", "--pids-limit", "32",
                  "--user", f"{os.getuid()}:{os.getgid()}", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m",
                  "--mount", f"type=bind,src={tls},dst=/tls,readonly",
                  "--mount", f"type=bind,src={state / 'comunicaciones/buzon'},dst=/data",
                  "--publish", f"127.0.0.1:{target['smtp_port']}:1025", "--publish", f"127.0.0.1:{target['http_port']}:8025",
                  IMAGE, *SMTP_FLAGS])
        image = _inspect(engine, "image", IMAGE)
        sink = _inspect(engine, "container", target["container"])
        _validate_sink(sink, state, image["Id"], target=target)
        proxies = _ensure_proxies(sink, state, target=target)
        for attempt in range(10):
            try:
                probe = _smtp_probe(material / "ca/ca.crt", smtp_port=target["smtp_port"])
                break
            except (OSError, smtplib.SMTPException):
                if attempt == 9:
                    raise PreparationError("smtp_starttls_not_ready") from None
                time.sleep(0.2)
        result = _result(source=source, image=True, smtp=True, target=target)
        result["profiles"]["usuarios_comunicaciones"].update(probe)
        result["profiles"]["usuarios_comunicaciones"]["loopback_proxies"] = proxies
        result["profiles"]["usuarios_comunicaciones"]["proxy_records"] = proxies
        result["profiles"]["usuarios_comunicaciones"]["image_id"] = image["Id"]
        result["profiles"]["usuarios_comunicaciones"]["target_sha256"] = hashlib.sha256(_private(state / "comunicaciones/target.json")).hexdigest()
        result["files"] = [str(state / "comunicaciones/target.json")] + [str(path) for path in sorted(tls.iterdir()) if path.is_file()]
        return result
    except (PreparationError, OSError, ValueError, smtplib.SMTPException) as error:
        return _result(source=source, image=True, target=target,
                       error=str(error) if isinstance(error, PreparationError) else "smtp_local_preparation_failed")


def _record_command(data: dict, port: int, *, target=None) -> list[str]:
    target = target or _default_target()
    command = data.get("command") if isinstance(data, dict) else None
    if not isinstance(command, list) or len(command) != 5 or not isinstance(command[-1], str):
        raise PreparationError("mailpit_proxy_record_mismatch")
    parts = command[-1].split(":")
    if len(parts) != 3 or parts[0] != "TCP4":
        raise PreparationError("mailpit_proxy_record_mismatch")
    destination = 1025 if port == target["smtp_port"] else 8025 if port == target["http_port"] else None
    expected = _proxy_command(parts[1], port, destination)
    if command != expected:
        raise PreparationError("mailpit_proxy_record_mismatch")
    return expected


def _clear_record(path: Path, previous: bytes) -> None:
    if _private(path) != previous:
        raise PreparationError("mailpit_proxy_record_changed")
    path.unlink()


def _lifecycle(repo, container, state, material, pg_port, engine, *, stop=False, smtp_port=None, http_port=None) -> dict:
    _scope(repo, container, state, material, pg_port, engine, require_postgres=False)
    state = _canonical(state)
    target = _target(state, container, pg_port, smtp_port, http_port)
    pinned = []
    try:
        with _proxy_lock(state):
            sink = _inspect(engine, "container", target["container"])
            network = _inspect(engine, "network", target["network"])
            if sink:
                image = _inspect(engine, "image", IMAGE)
                if image is None:
                    raise PreparationError("mailpit_image_not_installed")
                _validate_sink(sink, state, image["Id"], allow_stopped=True, target=target)
            if network:
                _owned(network, state, "network")
                if network.get("Internal") is not True:
                    raise PreparationError("mailpit_network_not_isolated")
                allowed = {sink["Id"]} if sink else set()
                if set(network.get("Containers", {})) - allowed:
                    raise PreparationError("mailpit_network_foreign_attachment")
            # Pin and validate every process before any stop or resource removal.
            proxies = []
            for port in (target["smtp_port"], target["http_port"]):
                path = state / "comunicaciones" / f"proxy-{port}.json"
                if not path.exists() and not path.is_symlink():
                    proxies.append({"port": port, "status": "absent"})
                    continue
                previous = _private(path)
                data = json.loads(previous)
                command = _record_command(data, port, target=target)
                kind, fd = _proxy_guard(data, command, state)
                pinned.append((path, previous, fd))
                proxies.append({"port": port, "pid": data["pid"], "status": kind})
            if stop:
                for path, previous, fd in pinned:
                    if fd is not None:
                        _terminate_child(fd)
                    _clear_record(path, previous)
                if sink:
                    # Immutable resource ID prevents name reuse between inspect and stop.
                    _run([engine, "stop", "--time", "5", sink["Id"]])
                if network:
                    _run([engine, "network", "rm", network["Id"]])
            return {"env": {}, "profiles": {}, "blockers": [],
                    "lifecycle": {"owner": OWNER, "target": target, "mode": "stop" if stop else "status",
                                  "container": "stopped" if stop else "running" if sink and sink.get("State", {}).get("Running") else "absent",
                                  "network": "removed" if stop else "internal" if network else "absent",
                                  "proxies": [{**item, "status": "stopped"} for item in proxies] if stop else proxies}}
    except (PreparationError, OSError, ValueError) as error:
        return {"env": {}, "profiles": {}, "blockers": [{"profile": "usuarios_comunicaciones",
                "code": "mailpit_lifecycle_denied", "detail": str(error) if isinstance(error, PreparationError) else "mailpit_lifecycle_unverifiable"}]}
    finally:
        for _, _, fd in pinned:
            if fd is not None:
                os.close(fd)


def status(repo, container, state, material, pg_port, engine, *, smtp_port=None, http_port=None) -> dict:
    return _lifecycle(repo, container, state, material, pg_port, engine, smtp_port=smtp_port, http_port=http_port)


def stop(repo, container, state, material, pg_port, engine, *, smtp_port=None, http_port=None) -> dict:
    return _lifecycle(repo, container, state, material, pg_port, engine, stop=True, smtp_port=smtp_port, http_port=http_port)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("configure", "preflight", "provision", "verify-smtp", "status", "stop"))
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--state", type=Path, default=CLI_STATE)
    parser.add_argument("--container")
    parser.add_argument("--pg-port", type=int)
    parser.add_argument("--smtp-port", type=int)
    parser.add_argument("--mailpit-http-port", "--http-port", dest="http_port", type=int)
    parser.add_argument("--engine", choices=("docker",), default="docker")
    args = parser.parse_args()
    try:
        state = _canonical(args.state)
        clone = json.loads(_private(state / "clon.json"))
        parameters = dict(repo=args.repo, container=clone["contenedor"] if args.container is None else args.container, state=state,
                          material=state / "material", pg_port=clone["puerto_pg"] if args.pg_port is None else args.pg_port, engine=args.engine,
                          smtp_port=args.smtp_port, http_port=args.http_port)
        operation = {"configure": configure, "preflight": preflight, "status": status, "stop": stop}.get(args.mode, provision)
        result = operation(**parameters)
        if args.mode == "verify-smtp" and not result["blockers"]:
            target = result["profiles"]["usuarios_comunicaciones"]
            result["smtp_probe"] = _smtp_probe(state / "material/ca/ca.crt", smtp_port=target["smtp_port"], send=True)
    except (PreparationError, OSError, ValueError, KeyError) as error:
        result = {"env": {}, "profiles": {}, "blockers": [{"profile": "usuarios_comunicaciones", "code": "mailpit_target_denied",
                  "detail": str(error) if isinstance(error, PreparationError) else "mailpit_target_unavailable"}]}
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 1 if result["blockers"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
