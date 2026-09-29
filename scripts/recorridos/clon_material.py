#!/usr/bin/env python3
"""Prepare private synthetic mTLS material for an explicitly owned local clone.

No application is started. Existing synthetic certificates and encryption keys retain their accounts
and history; new certificate identities never inherit an old account. Missing
candidate/Users account provisioning is recorded as a blocker, not success.
"""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import shlex
import socket
import ssl
import struct
import stat
import subprocess
import sys
import tempfile
import traceback
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

OWNER = "Codex-M"
GUARD = "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"
REVIEWED_SQL_COUNTS = {
    "7f1ecea2fd9f8912d255a80e74da84c69e46b978": 33,
    "ff6493cfccb2da4e83c94fa7c59be24c025cb7c9": 34,
    "e78687528d5725efd74e95c858d389f4437099ca": 36,
}
BASE = Path.home() / ".local/state/vec-clon/material-hito1"
DSN_KEYS = (
    "VEC_CT_DATABASE_URL", "VEC_CT_GOBIERNO_DATABASE_URL",
    "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL", "VEC_CT_CONFIRMADOR_DATABASE_URL",
    "VEC_CT_LECTOR_RESULTADO_DATABASE_URL", "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL",
    "VEC_CT_CONSULTAS_RRHH_DATABASE_URL", "VEC_CT_MOTIVOS_RRHH_DATABASE_URL",
    "VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL", "VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL",
    "VEC_CT_CONTEXTO_ACTOR_DATABASE_URL", "VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL",
    "VEC_AUTORIZACION_FUENTE_DATABASE_URL", "VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL",
)
HISTORY_FILES = (
    "kms/clave-maestra.bin", "kms/atestacion-ed25519.key", "kms/atestacion-ed25519.pub",
    "kms/revalidacion-ed25519.key", "kms/revalidacion-ed25519.pub", "tsa/clave-hmac.bin",
    "idempotencia/configuracion.json", "idempotencia/g1-localizador.bin",
    "idempotencia/g1-huella-solicitud.bin", "idempotencia/g2-localizador.bin",
    "idempotencia/g2-huella-solicitud.bin",
)
ROLES = {
    "rrhh": ("cliente", "identidad", "tecnico_rrhh"),
    "intervencion": ("intervencion", "intervencion", "intervencion"),
    "centro_solicitante": ("solicitante", "solicitante", "solicitante_centro"),
    "ratificador": ("ratificador", "ratificador", "ratificador_centro"),
    "candidato": ("candidato", "candidato", "candidato_bolsa"),
}


class MaterialError(RuntimeError):
    pass


class ModuleProvisionError(MaterialError):
    def __init__(self, module: str, cause: Exception):
        self.module = module
        name = type(cause).__name__
        self.cause_type = name if re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]{0,80}", name) else "module_error"
        super().__init__(module + ":" + self.cause_type)


def fail(reason: str) -> None:
    raise MaterialError(reason)


def run(args: list[str], text: str | None = None) -> bytes:
    # Never echo subprocess diagnostics: SQL and connection failures may contain secrets.
    r = subprocess.run(args, input=None if text is None else text.encode(),
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False, timeout=120)
    if r.returncode:
        fail("local command failed: " + Path(args[0]).name)
    return r.stdout


def canonical(path: Path) -> Path:
    path = path.absolute()
    if path != path.resolve():
        fail("non-canonical or symlinked path")
    return path


def private_read(path: Path) -> bytes:
    canonical(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as f:
        info = os.fstat(f.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077 or not 0 < info.st_size <= 262144:
            fail("unsafe private file")
        return f.read(262145)


def private_write(path: Path, data: bytes | str) -> None:
    canonical(path.parent)
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    if path.exists() or path.is_symlink():
        fail("refusing to replace existing private file")
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as f:
        f.write(data.encode() if isinstance(data, str) else data)
        f.flush()
        os.fsync(f.fileno())


def json_write(path: Path, value: object) -> None:
    private_write(path, json.dumps(value, ensure_ascii=False, indent=2) + "\n")


def replace_private(path: Path, value: object, *, plain: bool = False) -> None:
    # Controlled refresh after the previous manifest has been verified.
    private_read(path)
    fd, temp = tempfile.mkstemp(prefix=".material-refresh-", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            os.fchmod(stream.fileno(), 0o600)
            stream.write(value if plain else json.dumps(value, ensure_ascii=False, indent=2) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def load_profile_module(name: str):
    path = Path(__file__).with_name(name + ".py")
    if not path.is_file() or path.is_symlink():
        fail("missing profile provisioning module: " + name)
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    if not callable(getattr(module, "provision", None)):
        fail("invalid profile provisioning module: " + name)
    return module


def seal_state(output: Path, manifest: dict, env: dict, profiles: dict, blockers: list) -> dict:
    env.pop("VEC_HTTP_ALLOWED_CIDRS", None)  # runtime enforces exact loopback
    profiles["blockers"] = blockers
    replace_private(output / "perfiles.json", profiles)
    replace_private(output / "runtime-config.json", env)
    replace_private(output / "runtime.env", "".join(k + "=" + shlex.quote(v) + "\n" for k, v in sorted(env.items())), plain=True)
    files = {}
    for path in sorted((output / "material").rglob("*")):
        if path.is_file():
            files[str(path.relative_to(output))] = hashlib.sha256(private_read(path)).hexdigest()
    for name in ("perfiles.json", "runtime.env", "runtime-config.json"):
        files[name] = hashlib.sha256(private_read(output / name)).hexdigest()
    manifest.update(files=files, blockers=blockers, status="partial_blocked" if blockers else "prepared",
                    profiles_provisioned=not blockers, profiles_provisioning_attempted=True)
    replace_private(output / "material-manifest.json", manifest)
    return manifest


def log_module_failure(output: Path, name: str, error: Exception) -> None:
    path = output / "material-provision-private.log"
    canonical(path)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as log:
        info = os.fstat(log.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("unsafe private diagnostic file")
        log.write("\nmodule=" + name + "\n")
        log.write("".join(traceback.format_exception(error))[:65536])
        log.flush()
        os.fsync(log.fileno())


def current_users(material: Path) -> dict:
    users = {}
    for surface in ("interna", "externa"):
        config = json.loads(private_read(material / "identidad" / f"usuarios-preferencias-{surface}.json"))
        users[surface] = [{field: account[field] for field in ("sujeto", "certificado_sha256", "cuenta_ref", "perfil_ref")}
                          for account in config["cuentas"]]
    return users


def complete_profiles(args: argparse.Namespace, output: Path, manifest: dict) -> dict:
    env = json.loads(private_read(output / "runtime-config.json"))
    profiles = json.loads(private_read(output / "perfiles.json"))
    modules = (
        ("clon_usuarios", set()),
        ("clon_usuarios_h4", {"concesiones_correos_imagen_pendientes"}),
        ("clon_bolsa_material", {"bback_politica_ofertas_pendiente"}),
        ("clon_candidato_material", {"cuenta_contexto_candidato_pendiente", "contexto_externo_provision_autoridad_ausente_main"}),
    )
    # Preflight every dependency before permitting any side effect.
    loaded = [(name, load_profile_module(name), codes) for name, codes in modules]
    blockers = [b for b in manifest["blockers"] if b["code"] != "lector_adicional_pendiente"]
    for name, module, owned_codes in loaded:
        try:
            result = module.provision(repo=args.repo, container=args.container, state=output,
                                      material=output / "material", pg_port=args.pg_port, engine=args.engine)
        except (RuntimeError, OSError, ValueError, KeyError, subprocess.TimeoutExpired) as error:
            log_module_failure(output, name, error)
            raise ModuleProvisionError(name, error) from None
        if not isinstance(result, dict) or not isinstance(result.get("env", {}), dict) or not isinstance(result.get("profiles", {}), dict) or not isinstance(result.get("blockers", []), list):
            fail("invalid profile provisioning result")
        env.update(result.get("env", {}))
        profiles["profiles"].update(result.get("profiles", {}))
        blockers = [b for b in blockers if b["code"] not in owned_codes]
        blockers.extend(result.get("blockers", []))
        unique = {}
        for blocker in blockers:
            if not isinstance(blocker, dict) or not isinstance(blocker.get("code"), str):
                fail("invalid profile provisioning blocker")
            unique[(blocker.get("profile", ""), blocker["code"])] = blocker
        blockers = list(unique.values())
        if name in ("clon_usuarios", "clon_usuarios_h4"):
            # The JSON files are the actual CAS postimage, not the H1 source account.
            profiles["users"] = current_users(output / "material")
        # Seal every completed module before moving to the next dependency.
        # A later blocker preserves all already completed material and receipts.
        manifest = seal_state(output, manifest, env, profiles, blockers)
    return manifest


def load_env(path: Path) -> dict[str, str]:
    result: dict[str, str] = {}
    for raw in private_read(path).decode().splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        match = re.fullmatch(r"(?:export\s+)?(VEC_[A-Z0-9_]+)=(.*)", line)
        if not match or match[1] in result:
            fail("invalid private env")
        words = shlex.split(match[2])
        if len(words) != 1 or any(x in words[0] for x in ("\x00", "\n", "\r", "$", "`")):
            fail("invalid private env value")
        result[match[1]] = words[0]
    return result


def reference(prefix: str, value: str) -> str:
    return prefix + hashlib.sha256(("vec.ct.alta.desarrollo.v1\0" + value).encode()).hexdigest()[:32]


def cert_hash(path: Path) -> str:
    return hashlib.sha256(run(["openssl", "x509", "-in", str(path), "-outform", "DER"])).hexdigest()


def retarget_dsn(raw: str, port: int, ca: Path) -> str:
    p = urlsplit(raw)
    pairs = parse_qsl(p.query, strict_parsing=True)
    if p.scheme not in ("postgres", "postgresql") or p.hostname != "127.0.0.1" or p.port != 55441 or p.path != "/postgres" or not p.username or p.fragment:
        fail("source DSN is not the declared H1 local clone")
    if len(pairs) != 2 or dict(pairs).get("sslmode") != "verify-full" or set(dict(pairs)) != {"sslmode", "sslrootcert"}:
        fail("source DSN lacks verified TLS")
    # Preserve each nominal LOGIN and its private credential; no ACL expansion.
    auth = p.netloc.rsplit("@", 1)[0]
    return urlunsplit((p.scheme, auth + f"@127.0.0.1:{port}", "/postgres",
                       urlencode({"sslmode": "verify-full", "sslrootcert": str(ca)}), ""))


def validate_container(info: dict, port: int, state: Path | None = None) -> None:
    labels = info.get("Config", {}).get("Labels") or {}
    ports = info.get("NetworkSettings", {}).get("Ports")
    if state is not None and labels.get("vec.recorridos.state") != str(state):
        fail("clone state label does not match private output")
    if labels.get("vec.recorridos.owner") != OWNER or info.get("State", {}).get("Running") is not True:
        fail("container is not the running owned clone")
    if ports != {"5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(port)}]}:
        fail("clone PostgreSQL port is not exclusively loopback")


def certificate(material: Path, name: str, subject: str) -> None:
    mtls = material / "mtls"
    key, csr, ext, crt = (mtls / (name + x) for x in (".key", ".csr", ".ext", ".crt"))
    run(["openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", str(key)])
    run(["openssl", "req", "-new", "-sha256", "-key", str(key), "-subj",
         f"/CN={name}-clon-local/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", str(csr)])
    private_write(ext, "basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\n"
                  + f"subjectAltName=URI:urn:vec:{subject}\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n")
    run(["openssl", "x509", "-req", "-sha256", "-days", "397", "-in", str(csr),
         "-CA", str(material / "ca/ca.crt"), "-CAkey", str(material / "ca/ca.key"),
         "-CAserial", str(material / "ca/serie"), "-extfile", str(ext), "-out", str(crt)])
    password = mtls / (name + ".p12.password")
    private_write(password, secrets.token_hex(32) + "\n")
    run(["openssl", "pkcs12", "-export", "-out", str(mtls / (name + ".p12")), "-inkey", str(key),
         "-in", str(crt), "-certfile", str(material / "ca/ca.crt"), "-name", name,
         "-passout", "file:" + str(password)])
    csr.unlink()
    ext.unlink()
    for path in mtls.glob(name + ".*"):
        path.chmod(0o600)


def verify_existing(output: Path, identity: dict) -> dict:
    manifest = json.loads(private_read(output / "material-manifest.json"))
    if manifest.get("target") != identity:
        fail("existing material belongs to another source or clone")
    for relative, expected in manifest["files"].items():
        p = Path(relative)
        if p.is_absolute() or ".." in p.parts:
            fail("invalid material manifest path")
        if hashlib.sha256(private_read(output / p)).hexdigest() != expected:
            fail("existing private material changed; no automatic repair")
    return manifest


def query(args: argparse.Namespace, sql: str) -> bytes:
    return run([args.engine, "exec", "-i", args.container, "psql", "-X", "-qAt", "-v", "ON_ERROR_STOP=1",
                "-h", "/var/run/postgresql", "-p", "5432", "-U", "postgres", "-d", "postgres", "-f", "-"], sql)


def probe_pg_tls(port: int, ca: Path) -> None:
    context = ssl.create_default_context(cafile=str(ca))
    context.minimum_version = ssl.TLSVersion.TLSv1_3
    with socket.create_connection(("127.0.0.1", port), timeout=5) as connection:
        connection.sendall(struct.pack("!II", 8, 80877103))
        if connection.recv(1) != b"S":
            fail("PostgreSQL does not negotiate TLS")
        with context.wrap_socket(connection, server_hostname="127.0.0.1") as secured:
            if secured.version() not in ("TLSv1.2", "TLSv1.3"):
                fail("PostgreSQL TLS version insufficient")


def configure_pg_tls(args: argparse.Namespace, material: Path) -> None:
    # Only before application startup; no database or container restart is needed.
    try:
        with socket.create_connection(("127.0.0.1", args.port), timeout=0.2):
            fail("application port is active; PostgreSQL TLS preparation refused")
    except OSError:
        pass
    pgdir = query(args, "SHOW data_directory;\n").decode().strip()
    if not re.fullmatch(r"/var/lib/postgresql/(?:data|18/docker)", pgdir):
        fail("unexpected PostgreSQL 18 data directory")
    pg = material / "pg"
    key, csr, ext, crt = (pg / ("servidor" + x) for x in (".key", ".csr", ".ext", ".crt"))
    run(["openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", str(key)])
    run(["openssl", "req", "-new", "-sha256", "-key", str(key), "-subj", "/CN=localhost/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", str(csr)])
    private_write(ext, "basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=serverAuth\n"
                      "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n")
    run(["openssl", "x509", "-req", "-sha256", "-days", "397", "-in", str(csr), "-CA", str(material / "ca/ca.crt"),
         "-CAkey", str(material / "ca/ca.key"), "-CAserial", str(material / "ca/serie"), "-extfile", str(ext), "-out", str(crt)])
    csr.unlink()
    ext.unlink()
    key.chmod(0o600)
    crt.chmod(0o600)
    for source, suffix in ((key, "server.key"), (crt, "server.crt"), (pg / "ca.crt", "ca.crt")):
        # Fixed, validated PGDATA; private PEM travels only on stdin, never argv.
        target = pgdir + "/vec-recorridos-" + suffix
        run([args.engine, "exec", "-i", "-u", "postgres", args.container, "sh", "-c",
             "umask 077; cat > " + shlex.quote(target)], private_read(source).decode())
    query(args, "ALTER SYSTEM SET ssl='on';\n"
          + "ALTER SYSTEM SET ssl_cert_file='" + pgdir + "/vec-recorridos-server.crt';\n"
          + "ALTER SYSTEM SET ssl_key_file='" + pgdir + "/vec-recorridos-server.key';\n"
          + "ALTER SYSTEM SET ssl_ca_file='" + pgdir + "/vec-recorridos-ca.crt';\nALTER SYSTEM SET ssl_min_protocol_version='TLSv1.3';\nSELECT pg_reload_conf();\n")
    # Reload signal is asynchronous: bounded connection probes, never false success.
    import time
    for attempt in range(20):
        try:
            probe_pg_tls(args.pg_port, pg / "ca.crt")
            return
        except (OSError, ssl.SSLError, MaterialError):
            if attempt == 19:
                fail("PostgreSQL verified TLS probe failed")
            time.sleep(0.1)


def read_catalog(args: argparse.Namespace) -> dict:
    sql = """BEGIN READ ONLY;
SET LOCAL statement_timeout='5s';
WITH latest AS (
 SELECT version,revision,catalogo::jsonb AS doc
 FROM vec_contratacion_temporal.organizacion_catalogo_revision
 ORDER BY version DESC,revision DESC LIMIT 1
), entries AS (
 SELECT e FROM latest,LATERAL jsonb_array_elements(doc->'entradas') e
 WHERE (e->>'vigente_desde')::timestamptz<=clock_timestamp()
   AND (e->>'vigente_hasta'='0001-01-01T00:00:00Z'
        OR clock_timestamp()<(e->>'vigente_hasta')::timestamptz)
)
SELECT jsonb_build_object(
 'version',(SELECT version FROM latest),'revision',(SELECT revision FROM latest),
 'centers',coalesce((SELECT jsonb_agg(e->>'clave' ORDER BY e->>'clave') FROM entries
    WHERE e->'atributos'->>'tipo'='centro'),'[]'::jsonb),
 'positions',coalesce((SELECT jsonb_agg(jsonb_build_object('ref',e->>'clave',
    'center',e->'atributos'->>'adscripcion_clave') ORDER BY (e->>'orden')::integer)
    FROM entries WHERE e->'atributos'->>'tipo'='puesto_responsabilidad'),'[]'::jsonb)
)::text;
ROLLBACK;"""
    return json.loads(query(args, sql))


def center_bindings(catalog: dict) -> tuple[str, str, str] | None:
    groups: dict[str, list[str]] = {}
    for position in catalog.get("positions", []):
        center, ref = position.get("center"), position.get("ref")
        if center in catalog.get("centers", []) and isinstance(ref, str) and re.fullmatch(r"[A-Za-z0-9_:.-]{3,256}", ref):
            groups.setdefault(center, []).append(ref)
    for center, positions in groups.items():
        if re.fullmatch(r"[A-Za-z0-9_:.-]{3,256}", center) and len(set(positions)) >= 2:
            distinct = list(dict.fromkeys(positions))
            return center, distinct[0], distinct[1]
    return None


COVERAGE_GROUP = "vec_contratacion_temporal_lector_resultado_cobertura"
COVERAGE_SOURCE = "internal/modules/contrataciontemporal/adapters/postgres/acreditacion_pool_recuperacion_cobertura_o4_05.go"
COVERAGE_SOURCE_SHA = "1c07d19cec869c80cbef20ab160e4ad9507c11c1fc35c5bdb7489abf1abf5d58"


def coverage_accreditation_sql(repo: Path, source: str, login: str, *, missing_connect: bool = False) -> str:
    data = run(["git", "-C", str(repo), "show", source + ":" + COVERAGE_SOURCE])
    if hashlib.sha256(data).hexdigest() != COVERAGE_SOURCE_SHA:
        fail("coverage reader source contract changed")
    match = re.findall(r"const consultaAcreditacionPoolRecuperacionCoberturaO405 = `(.*?)`", data.decode(), re.S)
    if len(match) != 1 or not re.fullmatch(r"[a-z0-9_]{3,63}", login):
        fail("invalid coverage reader contract or LOGIN")
    sql = match[0].strip().rstrip(";")
    values = {
        1: "'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)'",
        2: "'" + COVERAGE_GROUP + "'", 3: "'vec_contratacion_temporal'",
        4: "'recuperar_resultado_propio_decision_cobertura_o405_v1'", 5: "'vec_contratacion_temporal_propietario'",
        6: "ARRAY['TimeZone=UTC','lock_timeout=2s','row_security=on','search_path=pg_catalog']",
        7: "'p_consulta jsonb'", 8: "'TABLE(resultado_json jsonb)'", 9: "'plpgsql'",
        10: "'379f7913cb05cf1a393d2104a3ad588f6e5945b5bbcb15e2358c739967020c7a'",
        11: "'a721175e1548fc1c98bc3fffaadb69f1e0657bd3e2f53169594121c3f64a6e53'",
    }
    sql = re.sub(r"\$(\d+)", lambda item: values[int(item[1])], sql)
    # Metadata as postgres explicitly selects the nominal LOGIN. Native session/TLS
    # columns remain native and are never represented as an actual LOGIN TLS probe.
    sql = sql.replace("WHERE login.rolname=session_user", "WHERE login.rolname='" + login + "'")
    if missing_connect:
        if sql.count("SELECT count(*)=3") != 1:
            fail("coverage dependency preimage marker changed")
        sql = sql.replace("SELECT count(*)=3", "SELECT count(*)=2")
        sql, count = re.subn(r"SELECT count\(\*\)=1\s+AND bool_and\(acl.privilege_type='CONNECT'\)\s+AND bool_and\(NOT acl.is_grantable\)",
                            "SELECT count(*)=0", sql)
        if count != 1:
            fail("coverage CONNECT preimage marker changed")
    return "SELECT row_to_json(ROW(coverage_metadata.*)) FROM (" + sql + ") AS coverage_metadata"


def coverage_metadata_valid(value: dict, *, physical_login: str | None = None, missing_connect: bool = False) -> bool:
    oid = str(value.get("f1", ""))
    if len(value) != 20 or not re.fullmatch(r"[1-9][0-9]{0,9}", oid) or int(oid) > 4294967295:
        return False
    if physical_login is not None and (value.get("f2") != physical_login or value.get("f3") != physical_login or value.get("f4") is not True):
        return False
    return all(value.get("f" + str(index)) is (False if missing_connect and index == 11 else True) for index in range(5, 21))


def coverage_snapshot_sql(login: str) -> str:
    return """SELECT jsonb_build_object(
 'database_acl',(SELECT datacl::text FROM pg_catalog.pg_database WHERE datname='postgres'),
 'schema_acl',(SELECT nspacl::text FROM pg_catalog.pg_namespace WHERE nspname='vec_contratacion_temporal'),
 'function_acl',(SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid='vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)'::regprocedure),
 'login',(SELECT row_to_json(r) FROM pg_catalog.pg_roles r WHERE rolname='""" + login + """'),
 'group',(SELECT row_to_json(r) FROM pg_catalog.pg_roles r WHERE rolname='""" + COVERAGE_GROUP + """'),
 'memberships',(SELECT jsonb_agg(row_to_json(m) ORDER BY roleid,member) FROM pg_catalog.pg_auth_members m WHERE member IN(SELECT oid FROM pg_catalog.pg_roles WHERE rolname IN('""" + login + "','" + COVERAGE_GROUP + "'))))"


def repair_coverage_connect(args: argparse.Namespace, output: Path, source: str) -> None:
    if source not in REVIEWED_SQL_COUNTS:
        fail("coverage repair source is not reviewed")
    try:
        with socket.create_connection(("127.0.0.1", args.port), timeout=0.2):
            fail("application must be stopped before coverage repair")
    except OSError:
        pass
    env = json.loads(private_read(output / "runtime-config.json"))
    dsn = urlsplit(env["VEC_CT_LECTOR_RESULTADO_DATABASE_URL"])
    if dsn.hostname != "127.0.0.1" or dsn.port != args.pg_port or dsn.path != "/postgres" or dsn.username != "vec_ct_o207_lector":
        fail("coverage reader LOGIN is outside the declared clone")
    login = dsn.username
    original = coverage_accreditation_sql(args.repo, source, login)
    missing = coverage_accreditation_sql(args.repo, source, login, missing_connect=True)
    snapshot = coverage_snapshot_sql(login)
    lock_path = output / "sql.lock"
    fd = os.open(lock_path, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("unsafe coverage SQL lock")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        # Source accreditation via the existing real LOGIN over verified TLS.
        ca = query(args, "SHOW ssl_ca_file;\n").decode().strip()
        if not re.fullmatch(r"/var/lib/postgresql/(?:data|18/docker)/vec-recorridos-ca.crt", ca):
            fail("coverage TLS CA path changed")
        def physical() -> dict:
            return json.loads(run([args.engine, "exec", "-i", "-e", "PGSSLMODE=verify-full", "-e", "PGSSLROOTCERT=" + ca,
                                  args.container, "psql", "-XqAt", "-h", "localhost", "-p", "5432", "-U", login,
                                  "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-f", "-"], "BEGIN READ ONLY;\n" + original + ";\nROLLBACK;\n"))
        before = physical()
        if coverage_metadata_valid(before, physical_login=login):
            return
        if not coverage_metadata_valid(before, physical_login=login, missing_connect=True):
            fail("coverage reader has another failed precondition")
        image = json.loads(query(args, snapshot + ";\n"))
        image_literal = "'" + json.dumps(image, sort_keys=True).replace("'", "''") + "'::jsonb"
        fields = " AND ".join("(accredited->>'f" + str(index) + "')::boolean" for index in range(5, 21))
        statement = "BEGIN; SET LOCAL search_path=pg_catalog; SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='8s';\n"
        statement += "SELECT pg_advisory_xact_lock(hashtextextended('Codex-M:coverage-reader-connect:v1',0));\nSELECT datacl FROM pg_catalog.pg_database WHERE datname='postgres' FOR UPDATE;\n"
        statement += "DO $coverage$ DECLARE actual jsonb; accredited jsonb; BEGIN " + snapshot + " INTO actual; IF actual<>" + image_literal + " THEN RAISE EXCEPTION 'coverage reader preimage changed'; END IF; "
        statement += missing + " INTO accredited; IF (" + fields + ") IS NOT TRUE THEN RAISE EXCEPTION 'coverage missing CONNECT precondition failed'; END IF; END $coverage$;\n"
        statement += "GRANT CONNECT ON DATABASE postgres TO " + COVERAGE_GROUP + ";\n"
        statement += "DO $coverage$ DECLARE accredited jsonb; after_image jsonb; BEGIN " + original + " INTO accredited; IF (" + fields + ") IS NOT TRUE THEN RAISE EXCEPTION 'coverage reader complete accreditation failed'; END IF; "
        statement += snapshot + " INTO after_image; IF (after_image-'database_acl')<>(" + image_literal + "-'database_acl') THEN RAISE EXCEPTION 'coverage reader immutable privilege changed'; END IF; END $coverage$;\n"
        query(args, statement + "ROLLBACK;\n")
        if json.loads(query(args, snapshot + ";\n")) != image:
            fail("coverage CONNECT rollback changed preimage")
        query(args, statement + "COMMIT;\n")
        if not coverage_metadata_valid(physical(), physical_login=login):
            fail("coverage reader physical postimage failed")
        receipt = {"version": 1, "source_commit": source, "group": COVERAGE_GROUP, "login": login,
                   "only_change": "database_CONNECT", "rollback_preimage_verified": True, "physical_TLS_before_after": True,
                   "preimage_sha256": hashlib.sha256(json.dumps(image, sort_keys=True).encode()).hexdigest()}
        path = output / "coverage-connect-receipt.json"
        if not path.exists():
            json_write(path, receipt)
    finally:
        os.close(fd)


def update_source(args: argparse.Namespace, output: Path, identity: dict) -> dict:
    previous = json.loads(private_read(output / "material-manifest.json"))
    old = previous.get("target", {})
    if {k: v for k, v in old.items() if k != "source_commit"} != {k: v for k, v in identity.items() if k != "source_commit"}:
        fail("source update cannot move material to another clone or port")
    manifest = verify_existing(output, old)
    if old.get("source_commit") != identity["source_commit"]:
        run(["git", "-C", str(args.repo), "merge-base", "--is-ancestor", old["source_commit"], identity["source_commit"]])
        try:
            with socket.create_connection(("127.0.0.1", identity["app_port"]), timeout=0.2):
                fail("application must be stopped before source upgrade")
        except OSError:
            pass
        ready = json.loads(private_read(output / "DB_READY.json"))
        journal_bytes = private_read(output / "sql-journal.json")
        journal = json.loads(journal_bytes)
        installed = journal.get("installed", [])
        count = REVIEWED_SQL_COUNTS.get(identity["source_commit"])
        if count is None or ready.get("commit") != identity["source_commit"] or ready.get("sql_instaladas") != count or journal.get("current_source_ref", journal.get("source_ref")) != identity["source_commit"] or len(installed) != count:
            fail("source upgrade requires the exact reviewed SQL receipt plan")
        seen = set()
        for position, receipt in enumerate(installed, 1):
            path = receipt.get("path", "")
            if receipt.get("position") != position or not isinstance(path, str) or not path.startswith("deploy/postgresql/") or not path.endswith(".sql") or ".." in Path(path).parts:
                fail("invalid source SQL receipt")
            if path in seen:
                fail("repeated source SQL receipt")
            seen.add(path)
            sql = run(["git", "-C", str(args.repo), "show", identity["source_commit"] + ":" + path])
            if hashlib.sha256(sql).hexdigest() != receipt.get("sha256"):
                fail("source SQL receipt does not match the reviewed revision")
        probe_pg_tls(args.pg_port, output / "material/pg/ca.crt")
        manifest["source_sql_receipts_sha256"] = hashlib.sha256(journal_bytes).hexdigest()
        manifest["source_updated_from"] = old["source_commit"]
        manifest["target"] = identity
        replace_private(output / "material-manifest.json", manifest)
    return manifest


def prepare(args: argparse.Namespace) -> dict:
    os.umask(0o077)
    repo, output, base = map(canonical, (args.repo, args.output, args.base_material))
    for directory in (repo, base):
        if not directory.is_dir():
            fail("required source directory missing")
    if output == repo or repo in output.parents or base == output or base in output.parents:
        fail("private output must be separate from source and base material")
    declared = json.loads(private_read(base / "manifiesto.json"))
    if declared.get("autoridad") != "no_autoritativo" or declared.get("perfil") != "desarrollo" or declared.get("migrable_a_produccion") is not False:
        fail("source material is not explicitly synthetic")
    # Reject every Git worktree ancestor, not just the supplied source tree.
    for ancestor in (output, *output.parents):
        if (ancestor / ".git").exists():
            fail("private output is inside Git")
    requested = getattr(args, "commit", None) or "HEAD"
    if requested != "HEAD" and not re.fullmatch(r"[0-9a-f]{40}", requested):
        fail("source commit must be a full SHA")
    head = run(["git", "-C", str(repo), "rev-parse", requested + "^{commit}"]).decode().strip()
    run(["git", "-C", str(repo), "merge-base", "--is-ancestor", head, "origin/main"])
    ready = json.loads(private_read(output / "DB_READY.json"))
    expected = {"commit": head, "contenedor": args.container, "propietario": OWNER,
                "puerto_pg": args.pg_port, "puerto_web": args.port}
    if any(ready.get(key) != value for key, value in expected.items()):
        fail("database readiness marker belongs to another source or target")
    if not re.fullmatch(r"vec-[a-z0-9_-]+", args.container):
        fail("invalid clone container name")
    info = json.loads(run([args.engine, "inspect", args.container]))[0]
    validate_container(info, args.pg_port, output)
    identity = {"source_commit": head, "container_id": info["Id"], "pg_port": args.pg_port, "app_port": args.port}
    if (output / "material-manifest.json").exists():
        manifest = update_source(args, output, identity) if getattr(args, "update_source", False) else verify_existing(output, identity)
        probe_pg_tls(args.pg_port, output / "material/pg/ca.crt")
        if getattr(args, "repair_coverage_connect", False):
            repair_coverage_connect(args, output, head)
        if getattr(args, "complete_profiles", False):
            return complete_profiles(args, output, manifest)
        return manifest
    output.mkdir(mode=0o700, parents=True, exist_ok=True)
    if output.stat().st_uid != os.getuid() or output.stat().st_mode & 0o077:
        fail("output must be owned and private")
    material = output / "material"
    if material.exists():
        fail("incomplete material already exists; preserve and inspect it")
    material.mkdir(mode=0o700)
    mandatory = HISTORY_FILES + (
        "ca/ca.crt", "ca/ca.key", "ca/serie", "tls/servidor.crt", "tls/servidor.key",
        "mtls/cliente.crt", "mtls/cliente.key", "mtls/cliente.p12", "mtls/cliente.p12.password",
        "mtls/intervencion.crt", "mtls/intervencion.key", "mtls/intervencion.p12", "mtls/intervencion.p12.password",
        "identidad/identidad.json", "identidad/intervencion.json", "manifiesto.json",
    )
    # Named inventory only: do not recursively copy scripts, logs or backups.
    for relative in mandatory:
        private_write(material / relative, private_read(base / relative))
    private_write(material / "pg/ca.crt", private_read(base / "ca/ca.crt"))
    configure_pg_tls(args, material)
    env = load_env(args.source_env or base.parent / "runtime-hito1.env")
    env = {key: retarget_dsn(env[key], args.pg_port, material / "pg/ca.crt") for key in DSN_KEYS}
    env.update({"VEC_EXECUTION_PROFILE": "desarrollo", "VEC_AUTH_MODE": "desarrollo",
                "VEC_DEVELOPMENT_GUARD": GUARD, "VEC_DEVELOPMENT_MATERIAL_DIR": str(material),
                "VEC_TLS_CERT_FILE": str(material / "tls/servidor.crt"),
                "VEC_TLS_KEY_FILE": str(material / "tls/servidor.key"),
                "VEC_HTTP_ADDR": f"127.0.0.1:{args.port}",
                "VEC_USUARIOS_PREFERENCIAS_ENABLED": "true"})
    profiles = {}
    for key, (name, file_id, role) in ROLES.items():
        subject = f"desarrollo:clon-recorridos:{key}"
        if key in ("rrhh", "intervencion"):
            actor = json.loads(private_read(material / f"identidad/{file_id}.json"))
            subject = actor["subject"]
            if actor["roles"] != [role] or actor["certificate_sha256"] != cert_hash(material / f"mtls/{name}.crt"):
                fail("source synthetic identity does not match its certificate")
        else:
            certificate(material, name, subject)
            actor = {"version": 1, "autoridad": "no_autoritativo", "certificate_sha256": cert_hash(material / f"mtls/{name}.crt"),
                     "subject": subject, "display_name": "Perfil sintético " + key.replace("_", " "), "roles": [role]}
            json_write(material / f"identidad/{file_id}.json", actor)
        profiles[key] = {"subject": subject, "role": role, "cert": f"material/mtls/{name}.crt",
                         "key": f"material/mtls/{name}.key", "pkcs12": f"material/mtls/{name}.p12",
                         "pkcs12_password_file": f"material/mtls/{name}.p12.password",
                         "certificate_sha256": actor["certificate_sha256"],
                         "status": "preserved" if key in ("rrhh", "intervencion") else "certificate_only"}
    catalog = read_catalog(args)
    binding = center_bindings(catalog)
    if binding:
        center, requester_position, ratifier_position = binding
        entries = []
        for profile_key, position in (("centro_solicitante", requester_position), ("ratificador", ratifier_position)):
            name, file_id, role = ROLES[profile_key]
            entry = {"certificate": f"mtls/{name}.crt", "identity": f"identidad/{file_id}.json", "role": role,
                     "centro_ref": center, "puesto_ref": position}
            if profile_key == "centro_solicitante":
                entry["ratificador_subject"] = profiles["ratificador"]["subject"]
            entries.append(entry)
            profiles[profile_key].update(status="bound_catalog", centro_ref=center, puesto_ref=position)
        json_write(material / "identidad/centros.json", {"version": 1, "autoridad": "no_autoritativo", "entradas": entries})
        env.update(VEC_PERSONAL_ORGANIZACION_POSTGRESQL="true", VEC_PERSONAL_ORGANIZACION_VERSION=str(catalog["version"]))
    rrhh = profiles["rrhh"]
    json_write(material / "identidad/consultas-rrhh.json", {"version": 1, "autoridad": "no_autoritativo", "entradas": [{
        "certificate": "mtls/cliente.crt", "identity": "identidad/identidad.json", "subject": rrhh["subject"],
        "perfil_ref": reference("prf_", rrhh["subject"] + "\0" + rrhh["certificate_sha256"] + "\0perfil"),
        "organizacion_ref": "organizacion:desarrollo:dipgra", "clase_ambito": "organizacion", "ambito_ref": "organizacion:desarrollo:dipgra"}]})
    users = {}
    for surface in ("interna", "externa"):
        relative = f"identidad/usuarios-preferencias-{surface}.json"
        data = json.loads(private_read(base / relative))
        for key, value in data.items():
            if key.startswith("dsn_"):
                data[key] = retarget_dsn(value, args.pg_port, material / "pg/ca.crt")
        json_write(material / relative, data)
        users[surface] = [{k: c[k] for k in ("sujeto", "certificado_sha256", "cuenta_ref", "perfil_ref")} for c in data["cuentas"]]
    # The existing external Users actor is retained, never relabelled as candidate.
    external_subject = users["externa"][0]["sujeto"]
    external = next((p for p in profiles.values() if p["subject"] == external_subject), None)
    if external is None:
        fail("external Users account has no matching trusted identity")
    profiles["area_personal"] = dict(external, users_profile=users["externa"][0]["perfil_ref"], status="preserved_users")
    # Center/catalog references must come from the installed catalog, not guessed constants.
    blockers = [
        {"profile": "centro_solicitante,ratificador", "code": "catalogo_centro_puesto_pendiente",
         "detail": "Falta seleccionar centro y puestos de responsabilidad del catálogo instalado para centros.json."},
        {"profile": "candidato", "code": "cuenta_contexto_candidato_pendiente",
         "detail": "El certificado nuevo requiere cuenta propia, perfil y vínculo candidato durables; no se reutiliza el actor externo de Usuarios."},
        {"profile": "usuarios", "code": "concesiones_correos_imagen_pendientes",
         "detail": "H4 requiere roles del hito H3; H1 usa otras cuentas y perfiles. Falta provisión focal por huella sobre la asignación real."},
        {"profile": "bolsa", "code": "bback_politica_ofertas_pendiente",
         "detail": "Faltan bolsa-bback.json nominal, LOGIN calculador y comprobación de la política de ofertas instalada."},
    ]
    if binding:
        blockers = [b for b in blockers if b["code"] != "catalogo_centro_puesto_pendiente"]
    blockers.append({"profile": "lector_rrhh", "code": "lector_adicional_pendiente",
                     "detail": "La lectura de RRHH usa el técnico conservado; los lectores de centro o unidad adicionales requieren alta nominal."})
    json_write(output / "perfiles.json", {"version": 1, "profiles": profiles, "users": users, "blockers": blockers})
    json_write(output / "runtime-config.json", env)
    private_write(output / "runtime.env", "".join(key + "=" + shlex.quote(value) + "\n" for key, value in sorted(env.items())))
    private_write(material / "desarrollo.env", "".join(key + "=" + shlex.quote(env[key]) + "\n" for key in (
        "VEC_EXECUTION_PROFILE", "VEC_AUTH_MODE", "VEC_DEVELOPMENT_GUARD", "VEC_DEVELOPMENT_MATERIAL_DIR", "VEC_TLS_CERT_FILE", "VEC_TLS_KEY_FILE")))
    files = {}
    for path in sorted(material.rglob("*")):
        if path.is_file():
            files[str(path.relative_to(output))] = hashlib.sha256(private_read(path)).hexdigest()
    for name in ("perfiles.json", "runtime.env", "runtime-config.json"):
        files[name] = hashlib.sha256(private_read(output / name)).hexdigest()
    manifest = {"version": 1, "owner": OWNER, "target": identity, "status": "partial_blocked",
                "files": files, "blockers": blockers, "application_started": False, "sql_applied": False, "pg_tls_configured": True}
    json_write(output / "material-manifest.json", manifest)
    if getattr(args, "repair_coverage_connect", False):
        repair_coverage_connect(args, output, head)
    if getattr(args, "complete_profiles", False):
        return complete_profiles(args, output, manifest)
    return manifest


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--commit", help="pinned main SHA; defaults to source HEAD")
    parser.add_argument("--container", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--pg-port", type=int, required=True)
    parser.add_argument("--base-material", type=Path, default=BASE)
    parser.add_argument("--source-env", type=Path)
    parser.add_argument("--repair-coverage-connect", action="store_true", help="accredit and grant only the missing coverage reader group CONNECT on the owned clone")
    parser.add_argument("--upgrade-source", "--update-source", dest="update_source", action="store_true", help="bind preserved material to a descendant main revision after matching DB_READY")
    parser.add_argument("--complete-profiles", action="store_true", help="run reviewed Users/Bolsa/candidate provisioning modules on existing material")
    parser.add_argument("--engine", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    if not (1024 <= args.port <= 65535 and 1024 <= args.pg_port <= 65535) or args.port == args.pg_port:
        parser.error("distinct unprivileged ports are required")
    try:
        manifest = prepare(args)
        print(json.dumps({"status": manifest["status"], "blockers": [b["code"] for b in manifest["blockers"]]}))
        return 3 if manifest["blockers"] else 0
    except ModuleProvisionError as error:
        print(json.dumps({"status": "failed", "module": error.module, "code": error.cause_type,
                          "detail_log": "material-provision-private.log"}), file=sys.stderr)
        return 1
    except (MaterialError, OSError, ValueError, KeyError, subprocess.TimeoutExpired):
        print("Private material preparation failed; no automatic repair.", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
