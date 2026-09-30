#!/usr/bin/env python3
"""Prepare the owned clone's synthetic STARTTLS sink and Users selectors.

No VEC process, PostgreSQL statement, role or shared manifest is changed.
The material driver consumes env/profiles/blockers and seals private files.
"""
from __future__ import annotations

import argparse
import ipaddress
import json
import os
from pathlib import Path
import smtplib
import resource
import socket
import ssl
import stat
import subprocess
import time

OWNER = "Codex-M"
STATE = Path.home() / ".local/state/vec-recorridos-codexm-20260930"
PG_CONTAINER = "vec-codexm-recorridos-20260930"
MAIL_CONTAINER = "vec-codexm-recorridos-mailpit-20260930"
NETWORK = "vec-codexm-recorridos-mailpit-red-20260930"
IMAGE = "axllent/mailpit:v1.27.8"
SOURCE_REF = "e78687528"
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
    path = Path(path).absolute()
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


def _free_ports() -> None:
    sockets = []
    try:
        for port in (SMTP_PORT, HTTP_PORT):
            listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sockets.append(listener)
            listener.bind(("127.0.0.1", port))
    except OSError:
        raise PreparationError("mailpit_loopback_port_busy") from None
    finally:
        for listener in sockets:
            listener.close()


def _validate_sink(resource: dict, state: Path, image_id: str) -> None:
    _owned(resource, state, "container")
    host = resource.get("HostConfig", {})
    if (resource.get("Image") != image_id or resource.get("Config", {}).get("Cmd") != SMTP_FLAGS
            or not host.get("ReadonlyRootfs") or not host.get("AutoRemove")
            or host.get("NetworkMode") != NETWORK):
        raise PreparationError("mailpit_configuration_mismatch")
    expected = {"1025/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(SMTP_PORT)}],
                "8025/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(HTTP_PORT)}]}
    if host.get("PortBindings") != expected:
        raise PreparationError("mailpit_nonloopback_bind")
    mounts = {entry.get("Destination"): entry for entry in resource.get("Mounts", [])}
    for destination, source, writable in (
            ("/tls", state / "material/comunicaciones", False),
            ("/data", state / "comunicaciones/buzon", True)):
        entry = mounts.get(destination, {})
        if entry.get("Source") != str(source) or entry.get("RW") is not writable:
            raise PreparationError("mailpit_mount_mismatch")
    if not resource.get("State", {}).get("Running"):
        raise PreparationError("mailpit_not_running")


def _source_contract(repo: Path) -> str:
    source = _run(["git", "-C", str(repo), "rev-parse", SOURCE_REF + "^{commit}"]).decode().strip()
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


def _scope(repo, container, state, material, pg_port, engine) -> tuple[Path, Path, Path]:
    repo, state, material = map(_canonical, (repo, state, material))
    if (state != STATE or material != state / "material" or container != PG_CONTAINER
            or pg_port != 55531 or engine != "docker"):
        raise PreparationError("owned_clone_scope_required")
    clone = json.loads(_private(state / "clon.json"))
    if clone.get("propietario") != OWNER or clone.get("contenedor") != container or clone.get("puerto_pg") != pg_port:
        raise PreparationError("clone_owner_mismatch")
    return repo, state, material


def _result(*, source="", image=False, smtp=False, error="") -> dict:
    env = dict(SELECTORS) if image else {}
    if smtp:
        env.update({"VEC_USUARIOS_CORREOS_ENABLED": "true", "VEC_SMTP_HOST": "127.0.0.1",
                    "VEC_SMTP_PORT": str(SMTP_PORT), "VEC_SMTP_FROM": "rrhh@example.test",
                    "VEC_SMTP_CA_FILE": str(STATE / "material/ca/ca.crt"), "VEC_SMTP_MODO_TLS": "starttls"})
    profiles = {"usuarios_comunicaciones": {"source_commit": source, "imagen_selector_prepared": image,
                 "smtp_ready": smtp, "mailpit_container": MAIL_CONTAINER,
                 "mailpit_origin": "http://127.0.0.1:" + str(HTTP_PORT),
                 "smtp_scope": "synthetic_local_sink", "corporate_delivery": False,
                 "application_started": False}}
    blockers = [] if not error else [{"code": "smtp_sintetico_no_preparado", "profile": "usuarios_correos", "detail": error}]
    return {"env": env, "profiles": profiles, "blockers": blockers}


def preflight(repo, container, state, material, pg_port, engine) -> dict:
    """Read contracts and owned resources; never create files or start processes."""
    repo, state, material = _scope(repo, container, state, material, pg_port, engine)
    source = _source_contract(repo)
    proof = json.loads(_private(state / "usuarios-h4-result.json"))
    if (proof.get("roles_version") != 2 or proof.get("assignments_version") != 2
            or proof.get("surfaces") != 2 or proof.get("grants_per_role") != 10):
        return _result(source=source, error="h4_roles_profiles_unverified")
    try:
        _private(material / "ca/ca.crt")
        _private(material / "ca/ca.key")
        image = _inspect(engine, "image", IMAGE)
        if image is None:
            raise PreparationError("mailpit_image_not_installed")
        network = _inspect(engine, "network", NETWORK)
        if network:
            _owned(network, state, "network")
            if network.get("Internal") is not True:
                raise PreparationError("mailpit_network_not_isolated")
        sink = _inspect(engine, "container", MAIL_CONTAINER)
        if sink:
            if not network:
                raise PreparationError("mailpit_network_missing")
            _validate_sink(sink, state, image["Id"])
        else:
            _free_ports()
    except (PreparationError, OSError, ValueError) as error:
        return _result(source=source, image=True, error=str(error) if isinstance(error, PreparationError) else "private_dependency_unavailable")
    return _result(source=source, image=True)


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


def _smtp_probe(ca: Path, *, send=False) -> dict:
    context = ssl.create_default_context(cafile=str(ca))
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    with smtplib.SMTP("127.0.0.1", SMTP_PORT, timeout=5) as connection:
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
    if (port, target) not in ((SMTP_PORT, 1025), (HTTP_PORT, 8025)):
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


def _ensure_proxies(sink: dict, state: Path) -> list[dict]:
    """Docker internal networks have no NAT publications: use owned fixed proxies.

    socat only forwards a loopback listener to the exact internal container IP;
    it has no relay choice, external destination, TLS termination or payload log.
    Existing unrelated listeners/processes are never stopped or reused.
    """
    networks = sink.get("NetworkSettings", {}).get("Networks", {})
    if set(networks) != {NETWORK}:
        raise PreparationError("mailpit_extra_network")
    address = networks[NETWORK].get("IPAddress", "")
    records = []
    for port, target in ((SMTP_PORT, 1025), (HTTP_PORT, 8025)):
        command = _proxy_command(address, port, target)
        record = state / "comunicaciones" / f"proxy-{port}.json"
        if record.exists():
            data = json.loads(_private(record))
            pid = data.get("pid")
            if (data.get("owner") != OWNER or data.get("state") != str(state)
                    or data.get("command") != command or not isinstance(pid, int) or pid < 2):
                raise PreparationError("mailpit_proxy_record_mismatch")
            proc = Path("/proc") / str(pid)
            actual = proc.joinpath("cmdline").read_bytes().rstrip(b"\0").split(b"\0")
            if (actual != [value.encode() for value in command] or proc.stat().st_uid != os.getuid()
                    or proc.joinpath("stat").read_text().split()[21] != data.get("start_ticks")):
                raise PreparationError("mailpit_proxy_process_mismatch")
        else:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
                listener.bind(("127.0.0.1", port))
            log = state / "comunicaciones" / f"proxy-{port}.log"
            fd = os.open(log, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
            with os.fdopen(fd, "wb") as stream:
                process = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                           stderr=stream, cwd=state / "comunicaciones", close_fds=True,
                                           start_new_session=True, preexec_fn=_proxy_limits,
                                           env={"PATH": "/usr/bin:/bin", "LANG": "C"})
            time.sleep(0.1)
            if process.poll() is not None:
                raise PreparationError("mailpit_proxy_start_failed")
            data = {"owner": OWNER, "state": str(state), "pid": process.pid, "command": command,
                    "start_ticks": Path(f"/proc/{process.pid}/stat").read_text().split()[21]}
            _write_new(record, (json.dumps(data, indent=2) + "\n").encode())
        records.append({"port": port, "pid": data["pid"], "record": str(record)})
    return records


def provision(repo, container, state, material, pg_port, engine) -> dict:
    """Prepare SMTP only; return selectors to the sole material driver."""
    result = preflight(repo, container, state, material, pg_port, engine)
    if result["blockers"]:
        return result
    state, material = Path(state), Path(material)
    source = result["profiles"]["usuarios_comunicaciones"]["source_commit"]
    try:
        tls = _certificate(material)
        _directory(state / "comunicaciones/buzon")
        if _inspect(engine, "network", NETWORK) is None:
            _run([engine, "network", "create", "--internal", "--label", LABEL_OWNER + "=" + OWNER,
                  "--label", LABEL_STATE + "=" + str(state), NETWORK])
        if _inspect(engine, "container", MAIL_CONTAINER) is None:
            _run([engine, "run", "--detach", "--rm", "--name", MAIL_CONTAINER,
                  "--label", LABEL_OWNER + "=" + OWNER, "--label", LABEL_STATE + "=" + str(state),
                  "--network", NETWORK, "--read-only", "--cap-drop", "ALL",
                  "--security-opt", "no-new-privileges", "--memory", "128m", "--cpus", "0.5", "--pids-limit", "32",
                  "--user", f"{os.getuid()}:{os.getgid()}", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m",
                  "--mount", f"type=bind,src={tls},dst=/tls,readonly",
                  "--mount", f"type=bind,src={state / 'comunicaciones/buzon'},dst=/data",
                  "--publish", f"127.0.0.1:{SMTP_PORT}:1025", "--publish", f"127.0.0.1:{HTTP_PORT}:8025",
                  IMAGE, *SMTP_FLAGS])
        image = _inspect(engine, "image", IMAGE)
        sink = _inspect(engine, "container", MAIL_CONTAINER)
        _validate_sink(sink, state, image["Id"])
        proxies = _ensure_proxies(sink, state)
        for attempt in range(10):
            try:
                probe = _smtp_probe(material / "ca/ca.crt")
                break
            except (OSError, smtplib.SMTPException):
                if attempt == 9:
                    raise PreparationError("smtp_starttls_not_ready") from None
                time.sleep(0.2)
        result = _result(source=source, image=True, smtp=True)
        result["profiles"]["usuarios_comunicaciones"].update(probe)
        result["profiles"]["usuarios_comunicaciones"]["loopback_proxies"] = proxies
        result["files"] = [str(path) for path in sorted(tls.iterdir()) if path.is_file()]
        return result
    except (PreparationError, OSError, ValueError, smtplib.SMTPException) as error:
        return _result(source=source, image=True,
                       error=str(error) if isinstance(error, PreparationError) else "smtp_local_preparation_failed")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("preflight", "provision", "verify-smtp"))
    parser.add_argument("--repo", type=Path, required=True)
    args = parser.parse_args()
    parameters = dict(repo=args.repo, container=PG_CONTAINER, state=STATE,
                      material=STATE / "material", pg_port=55531, engine="docker")
    result = provision(**parameters) if args.mode != "preflight" else preflight(**parameters)
    if args.mode == "verify-smtp" and not result["blockers"]:
        result["smtp_probe"] = _smtp_probe(STATE / "material/ca/ca.crt", send=True)
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 1 if result["blockers"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
