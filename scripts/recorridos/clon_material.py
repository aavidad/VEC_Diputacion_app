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


def load_source_validator():
    path = Path(__file__).with_name("clon_sql.py")
    if not path.is_file() or path.is_symlink():
        fail("missing central SQL source validator")
    spec = importlib.util.spec_from_file_location("clon_sql_material_validator", path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    if not callable(getattr(module, "approved_source_plan", None)) or not callable(getattr(module, "validate_receipts", None)):
        fail("central SQL source validator contract unavailable")
    return module


def source_context(args: argparse.Namespace, output: Path, source: str) -> dict:
    archive = canonical(getattr(args, "source_archive", None) or output / ("source-" + source))
    if not archive.is_relative_to(output) or not archive.is_dir() or archive.stat().st_uid != os.getuid() or archive.stat().st_mode & 0o077:
        fail("source archive must be owned private material within clone state")
    return {"source_repo": archive, "git_repo": canonical(args.repo), "source_ref": source}


def approved_material_plan(args: argparse.Namespace, output: Path, source: str) -> dict:
    context = source_context(args, output, source)
    validator = load_source_validator()
    try:
        plan = validator.approved_source_plan(context["source_repo"], source, git_repo=context["git_repo"])
    except (RuntimeError, OSError, ValueError, KeyError) as error:
        log_module_failure(output, "clon_sql_source_approval", error)
        raise ModuleProvisionError("clon_sql_source_approval", error) from None
    if not isinstance(plan, dict) or plan.get("source_ref") != source or type(plan.get("file_count")) is not int or plan["file_count"] <= 0:
        fail("invalid central SQL approved plan")
    args._source_context = context
    return plan


def validate_source_receipts(args: argparse.Namespace, output: Path, source: str) -> tuple[dict, bytes]:
    plan = approved_material_plan(args, output, source)
    ready = json.loads(private_read(output / "DB_READY.json"))
    journal_bytes = private_read(output / "sql-journal.json")
    journal = json.loads(journal_bytes)
    expected_journal = {"current_source_ref": source, "verified_source_ref": source,
                        "approved_sql_ref": plan["approved_sql_ref"], "current_plan_sha": plan["plan_sha"],
                        "inventory_sha": plan["inventory_sha"]}
    if ready.get("commit") != source or ready.get("sql_instaladas") != plan["file_count"] or any(journal.get(k) != v for k, v in expected_journal.items()):
        fail("source readiness and journal do not match the approved SQL plan")
    try:
        load_source_validator().validate_receipts(journal.get("installed", []), plan, complete=True)
    except (RuntimeError, OSError, ValueError, KeyError) as error:
        log_module_failure(output, "clon_sql_receipts", error)
        raise ModuleProvisionError("clon_sql_receipts", error) from None
    return plan, journal_bytes


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


def complete_profiles(args: argparse.Namespace, output: Path, manifest: dict, *, only_bolsa: bool = False) -> dict:
    env = json.loads(private_read(output / "runtime-config.json"))
    profiles = json.loads(private_read(output / "perfiles.json"))
    modules = (
        ("clon_usuarios", set()),
        ("clon_usuarios_h4", {"concesiones_correos_imagen_pendientes"}),
        ("clon_comunicaciones", {"smtp_sintetico_no_preparado"}),
        ("clon_bolsa_material", {"bback_politica_ofertas_pendiente"}),
        ("clon_candidato_material", {"cuenta_contexto_candidato_pendiente", "contexto_externo_provision_autoridad_ausente_main"}),
    )
    if only_bolsa:
        modules = tuple(item for item in modules if item[0] == "clon_bolsa_material")
    # Preflight every dependency before permitting any side effect.
    loaded = [(name, load_profile_module(name), codes) for name, codes in modules]
    blockers = [b for b in manifest["blockers"] if only_bolsa or b["code"] != "lector_adicional_pendiente"]
    for name, module, owned_codes in loaded:
        try:
            options = {}
            if name == "clon_bolsa_material" and getattr(args, "_source_context", None):
                options["source_context"] = args._source_context
            if name == "clon_comunicaciones":
                for key, argument in (("smtp_port", "smtp_port"), ("http_port", "mailpit_http_port")):
                    value = getattr(args, argument, None)
                    if value is not None:
                        options[key] = value
            result = module.provision(repo=args.repo, container=args.container, state=output,
                                      material=output / "material", pg_port=args.pg_port, engine=args.engine, **options)
        except (RuntimeError, OSError, ValueError, KeyError, subprocess.TimeoutExpired) as error:
            log_module_failure(output, name, error)
            raise ModuleProvisionError(name, error) from None
        if not isinstance(result, dict) or not isinstance(result.get("env", {}), dict) or not isinstance(result.get("profiles", {}), dict) or not isinstance(result.get("blockers", []), list):
            fail("invalid profile provisioning result")
        files = result.get("files", [])
        if not isinstance(files, list):
            fail("invalid private module file inventory")
        for declared in files:
            path = canonical(Path(declared))
            if not path.is_relative_to(output / "material"):
                fail("module file outside private material")
            private_read(path)
        env.update(result.get("env", {}))
        profiles["profiles"].update(result.get("profiles", {}))
        # Bolsa owns the "bolsa" profile diagnostics, including transient
        # source gates. Replace its previous result, then retain current errors.
        blockers = [b for b in blockers if b["code"] not in owned_codes
                    and not (name == "clon_bolsa_material" and b.get("profile") == "bolsa")]
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


NOMINAL_CONNECT_GROUPS = {
    "VEC_CT_DATABASE_URL": "vec_contratacion_temporal_ejecutor",
    "VEC_CT_GOBIERNO_DATABASE_URL": "vec_contratacion_temporal_gobernador",
    "VEC_CT_CONFIRMADOR_DATABASE_URL": "vec_contratacion_temporal_confirmador_cobertura",
    "VEC_CT_CONSULTAS_RRHH_DATABASE_URL": "vec_contratacion_temporal_consultor_rrhh",
    "VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL": "vec_contratacion_temporal_registrador_frontera",
    "VEC_CT_MOTIVOS_RRHH_DATABASE_URL": "vec_autorizacion_motivos_rrhh_resolutor",
    "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL": "vec_bolsa_llamamientos_ejecutor",
    "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL": "vec_bolsa_llamamientos_registrador_frontera",
}
NOMINAL_CONNECT_SOURCES = {
    "VEC_CT_DATABASE_URL": ("CT execution", "deploy/postgresql/contratacion_temporal/roles_up.sql"),
    "VEC_CT_GOBIERNO_DATABASE_URL": ("CT inherited government connection", "deploy/postgresql/contratacion_temporal/roles_up.sql"),
    "VEC_CT_CONFIRMADOR_DATABASE_URL": ("CT coverage confirmation", "deploy/postgresql/contratacion_temporal/roles_up.sql"),
    "VEC_CT_CONSULTAS_RRHH_DATABASE_URL": ("CT RRHH query", "deploy/postgresql/contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql"),
    "VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL": ("CT boundary audit", "internal/app/bootstrap/contratacion_temporal_postgresql_desarrollo.go"),
    "VEC_CT_MOTIVOS_RRHH_DATABASE_URL": ("RRHH governed query motives", "internal/modules/contrataciontemporal/adapters/postgres/acreditacion_pool_resolucion_motivos_rrhh.go"),
    "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL": ("Bolsa operations", "deploy/postgresql/bolsa_llamamientos/roles_up.sql"),
    "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL": ("Bolsa boundary audit", "internal/app/bootstrap/bolsa_auditoria_frontera_postgresql_desarrollo.go"),
}


def nominal_source_bindings(repo: Path, source: str) -> list[dict]:
    bindings = []
    for key, group in NOMINAL_CONNECT_GROUPS.items():
        capability, path = NOMINAL_CONNECT_SOURCES[key]
        data = run(["git", "-C", str(repo), "show", source + ":" + path])
        if group.encode() not in data:
            fail("nominal CONNECT role is not in its reviewed source")
        bindings.append({"variable": key, "group": group, "capability": capability,
                         "source_file": path, "source_sha256": hashlib.sha256(data).hexdigest()})
    return bindings


MOTIVES_SOURCE = "internal/modules/contrataciontemporal/adapters/postgres/acreditacion_pool_resolucion_motivos_rrhh.go"
MOTIVES_SOURCE_SHA = "2ce9f494fc3b52393e2f3e0444d9061342fef5552214a651911a1f3f1320e822"


def motives_accreditation_sql(repo: Path, source: str, login: str, *, physical: bool) -> str:
    data = run(["git", "-C", str(repo), "show", source + ":" + MOTIVES_SOURCE])
    if hashlib.sha256(data).hexdigest() != MOTIVES_SOURCE_SHA or not re.fullmatch(r"[a-z0-9_]{3,63}", login):
        fail("motives source contract or LOGIN changed")
    match = re.findall(r"const consultaAcreditacionPoolResolucionMotivosRRHH = `(.*?)`", data.decode(), re.S)
    if len(match) != 1:
        fail("motives accreditation contract missing")
    sql = match[0].strip().rstrip(";").replace("$1", "'" + login + "'").replace("$2", "true" if physical else "false")
    return "SELECT row_to_json(ROW(motives_metadata.*)) FROM (" + sql + ") AS motives_metadata"


def motives_metadata_valid(value: dict, login: str | None = None) -> bool:
    if len(value) != 15 or not all(re.fullmatch(r"[1-9][0-9]{0,9}", str(value.get(k, ""))) for k in ("f3", "f4")) or value["f3"] == value["f4"]:
        return False
    if login is not None and (value.get("f1") != login or value.get("f2") != login):
        return False
    return all(value.get("f" + str(n)) is True for n in range(5, 16))


def pool_login(raw: str, args: argparse.Namespace, output: Path) -> str:
    dsn = urlsplit(raw)
    pairs = parse_qsl(dsn.query, strict_parsing=True)
    if dsn.scheme not in ("postgres", "postgresql") or dsn.hostname != "127.0.0.1" or dsn.port != args.pg_port or dsn.path != "/postgres" or dsn.fragment or not re.fullmatch(r"[a-z0-9_]{3,63}", dsn.username or ""):
        fail("nominal runtime pool left the owned clone")
    if len(pairs) != 2 or dict(pairs).get("sslmode") != "verify-full" or dict(pairs).get("sslrootcert") != str(output / "material/pg/ca.crt"):
        fail("nominal runtime pool has unsupported parameters")
    return dsn.username


def physical_pool_query(args: argparse.Namespace, login: str, ca: str, sql: str) -> dict:
    return json.loads(run([args.engine, "exec", "-i", "-e", "PGSSLMODE=verify-full", "-e", "PGSSLROOTCERT=" + ca,
                          "-e", "PGOPTIONS=", args.container, "psql", "-XqAt", "-h", "localhost", "-p", "5432", "-U", login,
                          "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-f", "-"], "BEGIN READ ONLY;SET LOCAL statement_timeout='8s';SET LOCAL lock_timeout='3s';" + sql + ";ROLLBACK;"))


def nominal_connect_snapshot_sql(logins: list[str]) -> str:
    names = ",".join("'" + login + "'" for login in sorted(logins))
    return """SELECT jsonb_build_object(
 'database_acl',(SELECT coalesce(datacl,acldefault('d',datdba))::text FROM pg_database WHERE datname='postgres'),
 'acl_rows',(SELECT coalesce(jsonb_agg(jsonb_build_object('grantor',a.grantor,'grantee',a.grantee,'privilege',a.privilege_type,'grantable',a.is_grantable) ORDER BY a.grantor,a.grantee,a.privilege_type),'[]'::jsonb) FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname='postgres'),
 'roles',(SELECT jsonb_agg(row_to_json(r) ORDER BY rolname) FROM pg_roles r),
 'memberships',(SELECT jsonb_agg(row_to_json(m) ORDER BY roleid,member) FROM pg_auth_members m),
 'non_database_acl_sha256',(SELECT encode(sha256(convert_to(string_agg(v,E'\n' ORDER BY kind,oid),'UTF8')),'hex') FROM (
   SELECT 'schema' kind,oid,nspacl::text v FROM pg_namespace WHERE nspname NOT LIKE 'pg_%' AND nspname<>'information_schema'
   UNION ALL SELECT 'function',oid,proacl::text FROM pg_proc WHERE pronamespace IN(SELECT oid FROM pg_namespace WHERE nspname LIKE 'vec_%')
   UNION ALL SELECT 'relation',oid,relacl::text FROM pg_class WHERE relnamespace IN(SELECT oid FROM pg_namespace WHERE nspname LIKE 'vec_%')
   UNION ALL SELECT 'type',oid,typacl::text FROM pg_type WHERE typnamespace IN(SELECT oid FROM pg_namespace WHERE nspname LIKE 'vec_%')) acl),
 'pool_connect',(SELECT jsonb_object_agg(rolname,jsonb_build_object('connect',has_database_privilege(oid,'postgres','CONNECT'),'create',has_database_privilege(oid,'postgres','CREATE'),'temp',has_database_privilege(oid,'postgres','TEMP'))) FROM pg_roles WHERE rolname IN(""" + names + ")) )"


IMPORTACION_KEY = "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL"
IMPORTACION_LOGIN = "vec_bolsa_importacion_convoca_desarrollo"
IMPORTACION_GROUP = "vec_bolsa_importacion_convoca_ejecutor"
IMPORTACION_RECUPERADOR = "vec_bolsa_importacion_convoca_recuperador"
IMPORTACION_SOURCE = "internal/app/bootstrap/bolsa_importacion_convoca_pool.go"
IMPORTACION_SOURCE_SHA = "5707fbbe5c78c4b7b48267b8bb39f0095e224eddbd3b16c071d216daad1b72be"


def importacion_snapshot_sql() -> str:
    base = nominal_connect_snapshot_sql([IMPORTACION_LOGIN])
    return "SELECT snapshot.value || jsonb_build_object('login_dependencies',(SELECT count(*) FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=(SELECT oid FROM pg_roles WHERE rolname='" + IMPORTACION_LOGIN + "')),'login_settings',(SELECT count(*) FROM pg_db_role_setting WHERE setrole=(SELECT oid FROM pg_roles WHERE rolname='" + IMPORTACION_LOGIN + "')),'groups_owned_objects',(SELECT count(*) FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND deptype='o' AND refobjid IN(SELECT oid FROM pg_roles WHERE rolname IN('" + IMPORTACION_GROUP + "','" + IMPORTACION_RECUPERADOR + "')))) FROM (" + base + ") snapshot(value)"


def importacion_preimage(image: dict, login: str) -> tuple[int, bool]:
    if login != IMPORTACION_LOGIN:
        fail("importacion LOGIN is not the preserved H1 account")
    roles = {r["rolname"]: r for r in image["roles"]}
    names = (login, IMPORTACION_GROUP, IMPORTACION_RECUPERADOR)
    if any(name not in roles for name in names):
        fail("importacion existing role missing")
    for name in names:
        role = roles[name]
        if role["rolcanlogin"] is not (name == login) or any(role[field] for field in ("rolsuper", "rolcreatedb", "rolcreaterole", "rolreplication", "rolbypassrls")):
            fail("importacion existing role authority changed")
    if roles[login]["rolinherit"] is not True or roles[login].get("rolconfig") is not None or image["login_dependencies"] != 0 or image["login_settings"] != 0 or image["groups_owned_objects"] != 0:
        fail("importacion LOGIN or ownership preimage changed")
    members = [m for m in image["memberships"] if m["member"] == roles[login]["oid"]]
    expected = {roles[name]["oid"] for name in names[1:]}
    if len(members) != 2 or {m["roleid"] for m in members} != expected or any(m["inherit_option"] is not True or m["admin_option"] is not False or m["set_option"] is not True for m in members):
        fail("importacion historical memberships changed")
    oid = roles[IMPORTACION_GROUP]["oid"]
    if not re.fullmatch(r"[1-9][0-9]*", str(oid)):
        fail("importacion invalid group OID")
    public = [a for a in image["acl_rows"] if str(a["grantee"]) == "0"]
    if any(a["privilege"] != "CONNECT" or a["grantable"] is not False for a in public) or len(public) > 1:
        fail("importacion PUBLIC preimage is not CONNECT-only or absent")
    own = [a for a in image["acl_rows"] if a["grantee"] == oid]
    if any(a["privilege"] != "CONNECT" or a["grantable"] is not False for a in own) or len(own) > 1:
        fail("importacion group database authority changed")
    if any(a["grantee"] == roles[login]["oid"] for a in image["acl_rows"]):
        fail("importacion LOGIN has direct database authority")
    effective = image["pool_connect"].get(login)
    if not effective or effective["create"] is not False or effective["temp"] is not False or effective["connect"] is not bool(public or own):
        fail("importacion effective CONNECT preimage changed")
    return oid, bool(own)


def repair_importacion_connect(args: argparse.Namespace, output: Path, source: str) -> None:
    approved_material_plan(args, output, source)
    try:
        with socket.create_connection(("127.0.0.1", args.port), timeout=0.2):
            fail("application must be stopped before importacion CONNECT repair")
    except OSError:
        pass
    info, = json.loads(run([args.engine, "inspect", args.container]))
    validate_container(info, args.pg_port, output)
    manifest = json.loads(private_read(output / "material-manifest.json"))
    if manifest["target"]["container_id"] != info["Id"] or manifest["target"]["source_commit"] != source:
        fail("importacion clone container or source changed")
    data = run(["git", "-C", str(args.repo), "show", source + ":" + IMPORTACION_SOURCE])
    if hashlib.sha256(data).hexdigest() != IMPORTACION_SOURCE_SHA:
        fail("importacion source contract changed")
    env = json.loads(private_read(output / "runtime-config.json"))
    login = pool_login(env[IMPORTACION_KEY], args, output)
    if login != IMPORTACION_LOGIN:
        fail("importacion LOGIN is not the preserved H1 account")
    snapshot = importacion_snapshot_sql()
    fd = os.open(output / "sql.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
    try:
        meta = os.fstat(fd)
        if not stat.S_ISREG(meta.st_mode) or meta.st_nlink != 1 or meta.st_uid != os.getuid() or meta.st_mode & 0o077:
            fail("unsafe importacion CONNECT SQL lock")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        image = json.loads(query(args, snapshot + ";"))
        oid, complete = importacion_preimage(image, login)
        ca = query(args, "SHOW ssl_ca_file;").decode().strip()
        if not re.fullmatch(r"/var/lib/postgresql/(?:data|18/docker)/vec-recorridos-ca.crt", ca):
            fail("importacion CONNECT TLS CA changed")
        probe = "SELECT jsonb_build_object('session',session_user,'current',current_user,'tls',(SELECT ssl AND version='TLSv1.3' FROM pg_stat_ssl WHERE pid=pg_backend_pid()),'member',pg_has_role(session_user,'" + IMPORTACION_GROUP + "','MEMBER'),'database',current_database())"
        def physical():
            if physical_pool_query(args, login, ca, probe) != {"session": login, "current": login, "tls": True, "member": True, "database": "postgres"}:
                fail("importacion LOGIN physical TLS accreditation failed")
        if complete:
            physical()
            return
        path = output / "importacion-connect-receipt.json"
        if path.exists():
            fail("importacion CONNECT existing receipt requires reconciliation")
        after = json.loads(json.dumps(image))
        after["acl_rows"].append({"grantor": "10", "grantee": oid, "privilege": "CONNECT", "grantable": False})
        after["acl_rows"].sort(key=lambda row: (int(row["grantor"]), int(row["grantee"]), row["privilege"]))
        after["pool_connect"][login]["connect"] = True
        before_literal = "'" + json.dumps(image, sort_keys=True).replace("'", "''") + "'::jsonb"
        after_literal = "'" + json.dumps(after, sort_keys=True).replace("'", "''") + "'::jsonb"
        body = "SET LOCAL search_path=pg_catalog;SET LOCAL lock_timeout='3s';SET LOCAL statement_timeout='8s';SELECT pg_advisory_xact_lock(hashtextextended('Codex-M:importacion-connect:v1',0));SELECT datacl FROM pg_database WHERE datname='postgres' FOR UPDATE;"
        body += "DO $importacion$ DECLARE actual jsonb;BEGIN " + snapshot + " INTO actual;IF actual<>" + before_literal + " THEN RAISE EXCEPTION 'importacion CONNECT preimage changed';END IF;END $importacion$;GRANT CONNECT ON DATABASE postgres TO " + IMPORTACION_GROUP + ";"
        body += "DO $importacion$ DECLARE actual jsonb;BEGIN " + snapshot + " INTO actual;IF (actual-'database_acl')<>(" + after_literal + "-'database_acl') THEN RAISE EXCEPTION 'importacion CONNECT changed another authority';END IF;END $importacion$;"
        query(args, "BEGIN;" + body + "ROLLBACK;")
        if json.loads(query(args, snapshot + ";")) != image:
            fail("importacion CONNECT rollback changed the preimage")
        # A denied LOGIN cannot open an independent session before this GRANT
        # commits. Metadata inside the transaction is not a TLS session claim.
        query(args, "BEGIN;" + body + "COMMIT;")
        postimage = json.loads(query(args, snapshot + ";"))
        if not importacion_preimage(postimage, login)[1]:
            fail("importacion CONNECT postimage incomplete")
        physical()
        receipt = {"version": 1, "source_commit": source, "group": IMPORTACION_GROUP, "login": login,
                   "only_change": "existing_group_database_CONNECT", "rollback_preimage_verified": True,
                   "physical_TLS_after_commit": True, "metadata_is_not_a_physical_precommit_LOGIN": True,
                   "preimage_sha256": hashlib.sha256(json.dumps(image, sort_keys=True).encode()).hexdigest(),
                   "postimage_sha256": hashlib.sha256(json.dumps(postimage, sort_keys=True).encode()).hexdigest()}
        json_write(path, receipt)
    finally:
        os.close(fd)


def repair_nominal_connect(args: argparse.Namespace, output: Path, source: str) -> None:
    approved_material_plan(args, output, source)
    try:
        with socket.create_connection(("127.0.0.1", args.port), timeout=0.2):
            fail("application must be stopped before nominal CONNECT migration")
    except OSError:
        pass
    bindings = nominal_source_bindings(args.repo, source)
    env = json.loads(private_read(output / "runtime-config.json"))
    expected = set(DSN_KEYS) | {"VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL", "VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL"}
    if IMPORTACION_KEY in env:
        expected.add(IMPORTACION_KEY)
    if {k for k in env if k.endswith("_DATABASE_URL")} != expected:
        fail("nominal CONNECT runtime inventory changed")
    pools = {key: pool_login(env[key], args, output) for key in expected}
    users = {}
    for surface in ("interna", "externa"):
        config = json.loads(private_read(output / f"material/identidad/usuarios-preferencias-{surface}.json"))
        for key, raw in config.items():
            if key.startswith("dsn_"):
                users[surface + ":" + key] = pool_login(raw, args, output)
    if len(users) != 16 or len(set(users.values())) != 16:
        fail("Users nominal CONNECT inventory is not exactly two eight-pool surfaces")
    all_logins = sorted(set(pools.values()) | set(users.values()))
    snapshot = nominal_connect_snapshot_sql(all_logins)
    motives = motives_accreditation_sql(args.repo, source, pools["VEC_CT_MOTIVOS_RRHH_DATABASE_URL"], physical=False)
    coverage = coverage_accreditation_sql(args.repo, source, pools["VEC_CT_LECTOR_RESULTADO_DATABASE_URL"])
    fd = os.open(output / "sql.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("unsafe nominal CONNECT SQL lock")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        image = json.loads(query(args, snapshot + ";"))
        roles = {r["rolname"]: r for r in image["roles"]}
        groupoids = {g: roles[g]["oid"] for g in NOMINAL_CONNECT_GROUPS.values()}
        # Preserve the effective CONNECT already in use, not new capabilities.
        if any(not image["pool_connect"].get(l, {}).get("connect") or image["pool_connect"][l]["create"] or image["pool_connect"][l]["temp"] for l in all_logins):
            fail("configured pool effective preimage is not CONNECT-only")
        public = [a for a in image["acl_rows"] if str(a["grantee"]) == "0"]
        if public != [{"grantor": "10", "grantee": "0", "privilege": "CONNECT", "grantable": False}]:
            # A successful repeat has no PUBLIC and already exact physical accreditations.
            if not public and motives_metadata_valid(json.loads(query(args, motives + ";"))) and coverage_metadata_valid(json.loads(query(args, coverage + ";"))):
                return
            fail("PUBLIC database ACL is not the reviewed CONNECT-only preimage")
        for key, group in NOMINAL_CONNECT_GROUPS.items():
            r = roles[group]
            if r["rolcanlogin"] or any(r[x] for x in ("rolsuper","rolcreatedb","rolcreaterole","rolreplication","rolbypassrls")) or group.endswith("propietario"):
                fail("nominal group is not a safe non-owner runtime authority")
            owner = query(args,"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datdba="+str(r["oid"])+") OR EXISTS(SELECT 1 FROM pg_namespace WHERE nspowner="+str(r["oid"])+") OR EXISTS(SELECT 1 FROM pg_class WHERE relowner="+str(r["oid"])+") OR EXISTS(SELECT 1 FROM pg_proc WHERE proowner="+str(r["oid"])+");").strip()
            if owner != b"f": fail("nominal group owns a database object")
            login_oid = roles[pools[key]]["oid"]
            if not any(m["member"]==login_oid and m["roleid"]==r["oid"] and m["inherit_option"] and not m["admin_option"] for m in image["memberships"]):
                fail("nominal group is not already inherited by its configured LOGIN")
            if any(a["grantee"]==r["oid"] for a in image["acl_rows"]):
                fail("one of the eight nominal CONNECT grants is already present or broader")
        ca = query(args,"SHOW ssl_ca_file;").decode().strip()
        if not re.fullmatch(r"/var/lib/postgresql/(?:data|18/docker)/vec-recorridos-ca.crt",ca):fail("nominal CONNECT TLS CA changed")
        probe = "SELECT jsonb_build_object('session',session_user,'current',current_user,'tls',(SELECT ssl AND version='TLSv1.3' FROM pg_stat_ssl WHERE pid=pg_backend_pid()),'connect',has_database_privilege(current_user,'postgres','CONNECT'),'database',current_database())"
        def probes():
            for login in all_logins:
                value = physical_pool_query(args,login,ca,probe)
                if value != {"session":login,"current":login,"tls":True,"connect":True,"database":"postgres"}:fail("nominal pool physical TLS continuity failed")
        probes()
        expected_acl = [a for a in image["acl_rows"] if str(a["grantee"])!="0"] + [{"grantor":"10","grantee":oid,"privilege":"CONNECT","grantable":False} for oid in groupoids.values()]
        before_literal = "'" + json.dumps(image,sort_keys=True).replace("'","''") + "'::jsonb"
        acl_literal = "'" + json.dumps(sorted(expected_acl,key=lambda a:(int(a['grantor']),int(a['grantee']),a['privilege'])),sort_keys=True).replace("'","''") + "'::jsonb"
        checks = " AND ".join("(accredited->>'f"+str(n)+"')::boolean" for n in range(6,16))
        coverage_checks = " AND ".join("(accredited->>'f"+str(n)+"')::boolean" for n in range(5,21))
        body = "SET LOCAL search_path=pg_catalog;SET LOCAL lock_timeout='3s';SET LOCAL statement_timeout='8s';SELECT pg_advisory_xact_lock(hashtextextended('Codex-M:nominal-database-connect:v1',0));SELECT datacl FROM pg_database WHERE datname='postgres' FOR UPDATE;"
        body += "DO $nominal$ DECLARE actual jsonb;BEGIN " + snapshot + " INTO actual;IF actual<>"+before_literal+" THEN RAISE EXCEPTION 'nominal CONNECT preimage changed';END IF;END $nominal$;"
        body += "".join("GRANT CONNECT ON DATABASE postgres TO "+g+";" for g in NOMINAL_CONNECT_GROUPS.values())+"REVOKE CONNECT ON DATABASE postgres FROM PUBLIC;"
        body += "DO $nominal$ DECLARE accredited jsonb;actual jsonb; BEGIN "+motives+" INTO accredited;IF ("+checks+") IS NOT TRUE THEN RAISE EXCEPTION 'motives complete accreditation failed';END IF;"+coverage+" INTO accredited;IF ("+coverage_checks+") IS NOT TRUE THEN RAISE EXCEPTION 'coverage complete accreditation failed';END IF;"+snapshot+" INTO actual;IF (actual-'database_acl'-'acl_rows')<>("+before_literal+"-'database_acl'-'acl_rows') OR actual->'acl_rows'<>"+acl_literal+" THEN RAISE EXCEPTION 'nominal CONNECT postimage changed another privilege';END IF;END $nominal$;"
        query(args,"BEGIN;"+body+"ROLLBACK;")
        if json.loads(query(args,snapshot+";"))!=image:fail("nominal CONNECT rollback did not preserve preimage")
        # Keep the transaction open for real LOGIN TLS continuity probes before COMMIT.
        command=[args.engine,"exec","-i",args.container,"psql","-XqAt","-h","/var/run/postgresql","-p","5432","-U","postgres","-d","postgres","-v","ON_ERROR_STOP=1","-f","-"]
        process=subprocess.Popen(command,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        try:
            process.stdin.write(("BEGIN;"+body+"\nSELECT 'NOMINAL_READY_TO_COMMIT';\n").encode());process.stdin.flush()
            while process.stdout.readline().strip()!=b"NOMINAL_READY_TO_COMMIT":
                if process.poll() is not None:fail("nominal CONNECT transaction preparation failed")
            probes() # MVCC preimage visible externally; new CONNECT is separately proved above in the same TX.
            process.stdin.write(b"COMMIT;\n");process.stdin.close()
            if process.wait(timeout=20):fail("nominal CONNECT commit failed")
        except BaseException:
            process.kill();process.wait();raise
        probes()
        physical_motives=motives_accreditation_sql(args.repo,source,pools["VEC_CT_MOTIVOS_RRHH_DATABASE_URL"],physical=True)
        if not motives_metadata_valid(physical_pool_query(args,pools["VEC_CT_MOTIVOS_RRHH_DATABASE_URL"],ca,physical_motives),pools["VEC_CT_MOTIVOS_RRHH_DATABASE_URL"]):fail("physical motives postimage failed")
        if not coverage_metadata_valid(physical_pool_query(args,pools["VEC_CT_LECTOR_RESULTADO_DATABASE_URL"],ca,coverage),physical_login=pools["VEC_CT_LECTOR_RESULTADO_DATABASE_URL"]):fail("physical coverage postimage failed")
        result={"source_commit":source,"only_change":"eight_nominal_group_CONNECT_and_PUBLIC_CONNECT_removed","runtime_pools":len(pools),"users_pools":16,"groups":list(NOMINAL_CONNECT_GROUPS.values()),"source_bindings":bindings,"preserved_effective_privilege":"CONNECT only", "transaction_metadata_is_not_a_physical_LOGIN_TLS_assertion":True,"rollback_preimage_verified":True,"physical_TLS_probes_before_commit_and_after":len(all_logins),"new_connect_proved_in_transaction":True,"motives_full15":True,"coverage_full20":True,"preimage_sha256":hashlib.sha256(json.dumps(image,sort_keys=True).encode()).hexdigest(),"postimage_sha256":hashlib.sha256(query(args,snapshot+";")).hexdigest()}
        path=output/"nominal-connect-receipt.json"
        if not path.exists():json_write(path,result)
        else:
            existing=json.loads(private_read(path))
            if existing.get("source_commit")!=source or existing.get("only_change")!=result["only_change"] or existing.get("preimage_sha256")!=result["preimage_sha256"]:
                fail("nominal CONNECT receipt preimage changed")
            replace_private(path,result)
    finally:os.close(fd)


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
    approved_material_plan(args, output, source)
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
        plan, journal_bytes = validate_source_receipts(args, output, identity["source_commit"])
        probe_pg_tls(args.pg_port, output / "material/pg/ca.crt")
        manifest["source_sql_receipts_sha256"] = hashlib.sha256(journal_bytes).hexdigest()
        manifest["source_sql_approval"] = {k: plan[k] for k in ("approved_sql_ref", "plan_sha", "inventory_sha", "file_count")}
        manifest["source_updated_from"] = old["source_commit"]
        manifest["target"] = identity
        replace_private(output / "material-manifest.json", manifest)
    return manifest


def finish_preparation(args: argparse.Namespace, output: Path, source: str, manifest: dict) -> dict:
    if getattr(args, "complete_profiles", False):
        manifest = complete_profiles(args, output, manifest)
    # A fresh H1 lacks coverage CONNECT: establish it before nominal migration
    # demands the complete coverage postimage and removes PUBLIC CONNECT.
    if getattr(args, "repair_coverage_connect", False):
        repair_coverage_connect(args, output, source)
    if getattr(args, "repair_importacion_connect", False):
        repair_importacion_connect(args, output, source)
        # CONNECT changes no application config or actor; refresh only Bolsa's
        # previously pending pool accreditation after the verified repair.
        manifest = complete_profiles(args, output, manifest, only_bolsa=True)
    if getattr(args, "repair_nominal_connect", False):
        repair_nominal_connect(args, output, source)
    if getattr(args, "complete_profiles", False) or getattr(args, "project_internal", False) or getattr(args, "refresh_internal_proof", False):
        manifest = seal_internal_projection(args, output, source, manifest)
    return manifest


def seal_internal_projection(args: argparse.Namespace, output: Path, source: str, manifest: dict) -> dict:
    # This proof has its own root: profile-module result.files still permits
    # only files under operator material/. Do not expose that root to runtime.
    name = "clon_interno_material"
    try:
        descriptor = load_profile_module(name).provision(repo=args.repo, container=args.container, state=output,
            material=output / "material", pg_port=args.pg_port, engine=args.engine,
            source_context=getattr(args, "_source_context", None),
            refresh_operator_proof=getattr(args, "refresh_internal_proof", False))
        expected = {"mode": "interno", "portal": "interno", "material": "runtime-interno/material",
                    "config": "runtime-interno/runtime-config.json", "manifest": "runtime-interno/material-manifest.json",
                    "source_commit": source}
        if not isinstance(descriptor, dict) or any(descriptor.get(k) != v for k, v in expected.items()):
            fail("invalid internal projection descriptor")
        expected_rw = [{"source": "runtime-interno/rw/" + kind, "target": str(output / "runtime-interno/rw" / kind), "kind": kind}
                       for kind in ("documentos", "imagenes", "data")]
        expected_rw.append({"source": "runtime-interno/rw/comunicaciones", "target": str(output / "runtime-interno/material/comunicaciones"), "kind": "comunicaciones"})
        if descriptor.get("rw") != expected_rw or descriptor.get("manifest_sha256") != hashlib.sha256(private_read(output / expected["manifest"])).hexdigest():
            fail("invalid internal projection proof")
    except (RuntimeError, OSError, ValueError, KeyError, subprocess.TimeoutExpired) as error:
        log_module_failure(output, name, error)
        raise ModuleProvisionError(name, error) from None
    manifest["runtime_interno"] = descriptor
    replace_private(output / "material-manifest.json", manifest)
    return manifest


SYNTHETIC_PROFILE_FIXTURE = Path(__file__).with_name("perfiles_sinteticos.json")


def fresh_profile_names() -> tuple[dict, str]:
    path = canonical(SYNTHETIC_PROFILE_FIXTURE)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as fixture:
        info = os.fstat(fixture.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or not 0 < info.st_size <= 8192:
            fail("invalid synthetic profile fixture file")
        data = fixture.read(8193)
    def unique(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                fail("duplicate synthetic profile fixture key")
            value[key] = item
        return value
    catalog = json.loads(data, object_pairs_hook=unique)
    if not isinstance(catalog, dict) or set(catalog) != {"version", "autoridad", "nombres"} or type(catalog["version"]) is not int or catalog["version"] != 1 or catalog["autoridad"] != "no_autoritativo":
        fail("invalid synthetic profile fixture schema")
    names = catalog["nombres"]
    if not isinstance(names, dict) or set(names) != {"centro_solicitante", "ratificador", "candidato"}:
        fail("synthetic profile fixture must name exactly the three fresh actors")
    for name in names.values():
        if not isinstance(name, str) or not 3 <= len(name) <= 120 or name.strip() != name or not 2 <= len(name.split(" ")) <= 5 or any(not (character.isalpha() or character == " ") for character in name) or "  " in name:
            fail("invalid synthetic profile display name")
    if len(set(names.values())) != 3:
        fail("synthetic profile display names must be distinct")
    return names, hashlib.sha256(data).hexdigest()


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
    plan, journal_bytes = validate_source_receipts(args, output, head)
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
        return finish_preparation(args, output, head, manifest)
    # Fresh-only preflight precedes every copy, key generation and SQL setup.
    names, names_sha256 = fresh_profile_names()
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
                     "subject": subject, "display_name": names[key], "roles": [role]}
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
                "synthetic_profiles_fixture": {"path": "scripts/recorridos/perfiles_sinteticos.json", "sha256": names_sha256},
                "files": files, "blockers": blockers, "application_started": False, "sql_applied": False, "pg_tls_configured": True,
                "source_sql_approval": {k: plan[k] for k in ("approved_sql_ref", "plan_sha", "inventory_sha", "file_count")}}
    json_write(output / "material-manifest.json", manifest)
    return finish_preparation(args, output, head, manifest)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--commit", help="pinned main SHA; defaults to source HEAD")
    parser.add_argument("--source-archive", type=Path, help="sealed private archive of the source commit; defaults to output/source-commit")
    parser.add_argument("--smtp-port", type=int)
    parser.add_argument("--mailpit-http-port", type=int)
    parser.add_argument("--container", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--pg-port", type=int, required=True)
    parser.add_argument("--base-material", type=Path, default=BASE)
    parser.add_argument("--source-env", type=Path)
    parser.add_argument("--repair-nominal-connect", action="store_true", help="migrate only reviewed runtime group CONNECT from PUBLIC on the owned clone")
    parser.add_argument("--repair-coverage-connect", action="store_true", help="accredit and grant only the missing coverage reader group CONNECT on the owned clone")
    parser.add_argument("--repair-importacion-connect", action="store_true", help="restore only CONNECT for the existing H1 importacion group and LOGIN")
    parser.add_argument("--upgrade-source", "--update-source", dest="update_source", action="store_true", help="bind preserved material to a descendant main revision after matching DB_READY")
    parser.add_argument("--complete-profiles", action="store_true", help="run reviewed Users/Bolsa/candidate provisioning modules on existing material")
    parser.add_argument("--project-internal", action="store_true", help="seal only the internal runtime projection; complete-profiles also seals it")
    parser.add_argument("--refresh-internal-proof", action="store_true", help="CAS only a changed operator-manifest reference when all internal runtime bytes remain identical")
    parser.add_argument("--engine", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    if not (1024 <= args.port <= 65535 and 1024 <= args.pg_port <= 65535) or args.port == args.pg_port:
        parser.error("distinct unprivileged ports are required")
    optional_ports = [v for v in (args.smtp_port, args.mailpit_http_port) if v is not None]
    if any(not 1024 <= v <= 65535 for v in optional_ports) or len(set(optional_ports + [args.port, args.pg_port])) != len(optional_ports) + 2:
        parser.error("distinct unprivileged SMTP and Mailpit ports are required")
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
