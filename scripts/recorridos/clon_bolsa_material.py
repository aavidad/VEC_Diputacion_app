#!/usr/bin/env python3
"""Prepare B-BACK/H5 material in the explicitly owned synthetic clone.

No application lifecycle, migrations, grants to people or government publication.
The existing main bootstrap remains the only authority for V3 publication.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import sys
from urllib.parse import quote, urlencode


SOURCE = "7f1ecea2fd9f8912d255a80e74da84c69e46b978"
# Main contracts used by this administrative helper. An unrelated main
# increment may advance the clone's current source; a changed contract needs
# a fresh review rather than silent acceptance.
CONTRACT_HASHES = {
    # Preserved approved baseline. D's changed publisher remains closed until
    # root supplies its final source approval and reviewed replacement digest.
    "internal/app/bootstrap/bolsa_borrador_contexto_postgresql_desarrollo.go": "76613f3fba276b2f20eb0ac9057e6c1d112ed875640816ee73f719a8c4998fab",
    "internal/app/bootstrap/bolsa_borrador_identidad_desarrollo.go": "451d9d56f108480cea5a92164f0f1b8cfc04a168267a36b9e79de16f71e9762e",
    "internal/app/bootstrap/bolsa_borrador_llamamiento_desarrollo.go": "a9a4cc7e268e2d7cca3825102001003c338cd72ad37d21d03580c4ad056bbc05",
    "internal/app/bootstrap/bolsa_ofertas_desarrollo.go": "9727a2d8e4b0f00112e42377b5674e0cf0ca339225d543f35f97b0deeef82bec",
    "internal/app/bootstrap/bolsa_auditoria_frontera_postgresql_desarrollo.go": "3fe40db4d72ac34374b3817078b9dda69b7a8283eb0f4c49cc3f7e228e27182a",
    "internal/app/bootstrap/bolsa_borrador_politica_desarrollo.go": "190089f6b533b9c7e0c1135fcfc6b5d8143656faca82f4db6b02b7f29ab96a13",
    "config/postgresql_borradores.go": "6770f91af7bd67b75b8beb14b13e78290b88364217d5efafe70659c3a2dd8725",
    "internal/app/bootstrap/postgresql_borradores_configuracion.go": "d33403dde4e0f77e4864198f8e758b3959c3a946bf9e76112cce5c7022f19abc",
}
CONTAINER = "vec-codexm-recorridos-20260930"
STATE = Path.home() / ".local/state/vec-recorridos-codexm-20260930"
CALCULATOR = "vec_bolsa_calculador_politica_desarrollo"
CALCULATOR_GROUP = "vec_bolsa_llamamientos_calculador_politica"
BUSINESS = "vec_bolsa_llamamientos_desarrollo"
AUDIT = "vec_b2_auditoria_frontera_desarrollo"
REQUIRED = {
    BUSINESS: "vec_bolsa_llamamientos_ejecutor",
    AUDIT: "vec_bolsa_llamamientos_registrador_frontera",
    CALCULATOR: CALCULATOR_GROUP,
}
OPTIONAL_GROUPS = (
    "vec_bolsa_convocatorias_ejecutor_consulta",
    "vec_bolsa_convocatorias_proyector_gobierno",
    "vec_bolsa_convocatorias_verificador_recibo",
)


class ProvisionError(RuntimeError):
    """Messages deliberately carry no SQL, credentials or identity values."""


def _source_contracts(repo: Path, source_ref: str) -> None:
    if not isinstance(source_ref, str) or not re.fullmatch("[a-f0-9]{40}", source_ref):
        raise ProvisionError("source_commit_not_authorized")
    _check_path(repo, directory=True)
    for name, expected in CONTRACT_HASHES.items():
        # The shared root may contain unrelated WIP or an older checkout.
        # Read the requested committed source, preserving every byte.
        source = subprocess.run(["git", "-C", str(repo), "show", source_ref + ":" + name],
                                capture_output=True, timeout=15, check=False,
                                env={"PATH": os.defpath, "LC_ALL": "C",
                                     "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null",
                                     "GIT_NO_REPLACE_OBJECTS": "1"})
        if source.returncode:
            raise ProvisionError("source_contract_unavailable")
        if hashlib.sha256(source.stdout).hexdigest() != expected:
            raise ProvisionError("source_contract_changed")


@contextmanager
def _sql_lock(state: Path):
    path = state / "sql.lock"
    descriptor = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        _check_path(path, private=True)
        try:
            fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise ProvisionError("clone_sql_lock_busy") from exc
        yield
    finally:
        os.close(descriptor)


def _run(argv: list[str], *, data: str | None = None) -> str:
    result = subprocess.run(argv, input=data, capture_output=True, text=True,
                            timeout=45, check=False)
    if result.returncode:
        raise ProvisionError("local_command_failed")
    return result.stdout.strip()


def _check_path(path: Path, *, private: bool = False, directory: bool = False) -> None:
    for component in (path, *path.parents):
        if component.is_symlink():
            raise ProvisionError("symlink_path_rejected")
    info = path.stat()
    expected = stat.S_ISDIR if directory else stat.S_ISREG
    if not expected(info.st_mode) or info.st_uid != os.getuid():
        raise ProvisionError("unsafe_material_owner_or_type")
    if not directory and info.st_nlink != 1:
        raise ProvisionError("hardlink_material_rejected")
    if private and info.st_mode & 0o077:
        raise ProvisionError("private_permissions_required")


def _read_json(path: Path) -> dict:
    _check_path(path, private=True)
    if path.stat().st_size > 1024 * 1024:
        raise ProvisionError("material_too_large")
    def unique(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ProvisionError("duplicate_json_key")
            result[key] = value
        return result
    value = json.loads(path.read_text(), object_pairs_hook=unique)
    if not isinstance(value, dict):
        raise ProvisionError("invalid_json_shape")
    return value


def _write_new(path: Path, value: dict) -> None:
    if path.exists() or path.is_symlink():
        if _read_json(path) != value:
            raise ProvisionError("existing_material_differs")
        return
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as out:
        json.dump(value, out, ensure_ascii=False, indent=2)
        out.write("\n")
        out.flush()
        os.fsync(out.fileno())


def _literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def _sql_authority():
    path = Path(__file__).with_name("clon_sql.py")
    if not path.exists():
        raise ProvisionError("sql_source_authority_unavailable")
    _check_path(path)
    spec = importlib.util.spec_from_file_location("vec_recorridos_sql_authority", path)
    try:
        module = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = module
        spec.loader.exec_module(module)
        if not all(callable(getattr(module, name, None)) for name in ("approved_source_plan", "validate_receipts")):
            raise ProvisionError("sql_source_authority_unavailable")
    except Exception as exc:
        raise ProvisionError("sql_source_authority_unavailable") from exc
    return module


def _approved_plan(repo: Path, state: Path, ready: dict, source_context: dict | None, authority):
    current = ready.get("current_source_ref", ready.get("commit"))
    if not isinstance(current, str) or not re.fullmatch("[a-f0-9]{40}", current):
        raise ProvisionError("source_commit_not_authorized")
    if source_context is None:
        source_context = {"source_repo": state / ("source-" + current),
                          "git_repo": repo, "source_ref": current}
    if (not isinstance(source_context, dict)
            or set(source_context) != {"source_repo", "git_repo", "source_ref"}
            or source_context["source_ref"] != current):
        raise ProvisionError("source_context_mismatch")
    archive, git_repo = Path(source_context["source_repo"]), Path(source_context["git_repo"])
    if not archive.exists():
        raise ProvisionError("source_archive_missing")
    _check_path(archive, directory=True)
    _check_path(git_repo, directory=True)
    try:
        plan = authority.approved_source_plan(archive, current, git_repo=git_repo)
    except Exception as exc:
        raise ProvisionError("sql_source_needs_review") from exc
    if not isinstance(plan, dict) or plan.get("source_ref") != current:
        raise ProvisionError("sql_source_authority_invalid")
    return plan, git_repo


def _ready_receipts(ready: dict, journal: dict, plan: dict, authority) -> None:
    current = ready.get("current_source_ref", ready.get("commit"))
    expected = plan.get("file_count")
    if (current != plan.get("source_ref") or journal.get("current_source_ref") != current
            or journal.get("verified_source_ref") != current
            or journal.get("approved_sql_ref") != plan.get("approved_sql_ref")
            or journal.get("inventory_sha") != plan.get("inventory_sha")
            or journal.get("current_plan_sha") != plan.get("plan_sha")):
        raise ProvisionError("clone_receipt_source_mismatch")
    installed = ready.get("sql_instaladas")
    receipts = journal.get("installed")
    if (type(expected) is not int or expected <= 0 or type(installed) is not int or installed != expected
            or not isinstance(receipts, list) or len(receipts) != expected):
        raise ProvisionError("clone_receipt_count_mismatch")
    try:
        authority.validate_receipts(receipts, plan, complete=True)
    except Exception as exc:
        raise ProvisionError("clone_receipt_validation_failed") from exc


class Clone:
    def __init__(self, engine: str, container: str, state: Path, pg_port: int,
                 plan: dict | None = None, authority=None):
        self.engine, self.container, self.state, self.pg_port = engine, container, state, pg_port
        self.plan, self.authority = plan, authority

    def owner_guard(self) -> dict:
        ready = _read_json(self.state / "DB_READY.json")
        inventory = _read_json(self.state / "clon.json")
        for document in (ready, inventory):
            if (document.get("propietario") != "Codex-M"
                    or document.get("estado") != str(self.state)
                    or document.get("contenedor") != self.container
                    or document.get("puerto_pg") != self.pg_port):
                raise ProvisionError("clone_inventory_mismatch")
        inspected = json.loads(_run([self.engine, "inspect", self.container]))[0]
        labels = inspected.get("Config", {}).get("Labels", {})
        if (labels.get("vec.recorridos.owner") != "Codex-M"
                or labels.get("vec.recorridos.state") != str(self.state)
                or not inspected.get("State", {}).get("Running")):
            raise ProvisionError("clone_labels_or_liveness_mismatch")
        ports = inspected.get("NetworkSettings", {}).get("Ports", {}).get("5432/tcp", [])
        if ports != [{"HostIp": "127.0.0.1", "HostPort": str(self.pg_port)}]:
            raise ProvisionError("clone_binding_mismatch")
        return ready

    def guard(self) -> dict:
        ready = self.owner_guard()
        if self.plan is None or self.authority is None:
            raise ProvisionError("sql_source_authority_unavailable")
        _ready_receipts(ready, _read_json(self.state / "sql-journal.json"), self.plan, self.authority)
        return ready

    def sql(self, text: str, *, user: str = "postgres", tls_ca: str | None = None) -> str:
        self.guard()
        argv = [self.engine, "exec", "-i"]
        if tls_ca:
            argv += ["-e", "PGSSLMODE=verify-full",
                     "-e", "PGSSLROOTCERT=" + tls_ca]
        argv += [self.container, "psql", "-XAtq", "-v", "ON_ERROR_STOP=1",
                 "-h", "localhost" if tls_ca else "/var/run/postgresql", "-p", "5432",
                 "-U", user, "-d", "postgres"]
        return _run(argv, data=text)

    def snapshot(self) -> dict:
        names = ",".join(_literal(x) for x in (*REQUIRED, *REQUIRED.values(), *OPTIONAL_GROUPS))
        return json.loads(self.sql(f"""
          SELECT json_build_object(
            'roles', COALESCE((SELECT json_agg(row_to_json(r) ORDER BY r.rolname)
              FROM (SELECT rolname,rolcanlogin,rolinherit,rolsuper,rolcreatedb,
                   rolcreaterole,rolreplication,rolbypassrls,rolconnlimit,
                   md5(COALESCE(rolpassword,'')) AS password_digest
                   FROM pg_authid WHERE rolname IN ({names})) r), '[]'::json),
            'memberships', COALESCE((SELECT json_agg(row_to_json(m) ORDER BY m.member,m.role)
              FROM (SELECT member.rolname AS member,g.rolname AS role,
                   a.admin_option,a.inherit_option,a.set_option
                   FROM pg_auth_members a JOIN pg_roles member ON member.oid=a.member
                   JOIN pg_roles g ON g.oid=a.roleid
                   WHERE member.rolname IN ({names}) OR g.rolname IN ({names})) m), '[]'::json));
        """))


def _fingerprint(snapshot: dict) -> str:
    return hashlib.sha256(json.dumps(snapshot, sort_keys=True).encode()).hexdigest()


def _nominal(snapshot: dict, login: str, group: str, *, allow_absent: bool = False) -> bool:
    roles = {r["rolname"]: r for r in snapshot["roles"]}
    target = roles.get(group)
    if not target or target["rolcanlogin"] or not target["rolinherit"]:
        raise ProvisionError("required_group_absent_or_invalid")
    for role in (target, roles.get(login)):
        if role and any(role[k] for k in ("rolsuper", "rolcreatedb", "rolcreaterole", "rolreplication", "rolbypassrls")):
            raise ProvisionError("privileged_runtime_role_rejected")
    if any(m["member"] == group for m in snapshot["memberships"]):
        raise ProvisionError("runtime_group_inherits_other_role")
    actor = roles.get(login)
    if login == CALCULATOR:
        group_members = [m["member"] for m in snapshot["memberships"] if m["role"] == group]
        if group_members != ([CALCULATOR] if actor else []):
            raise ProvisionError("calculator_group_unexpected_member")
    if actor is None:
        if allow_absent:
            return False
        raise ProvisionError("required_login_absent")
    memberships = [m for m in snapshot["memberships"] if m["member"] == login]
    if (not actor["rolcanlogin"] or not actor["rolinherit"] or len(memberships) != 1
            or memberships[0]["role"] != group or memberships[0]["admin_option"]
            or not memberships[0]["inherit_option"]
            or (login == CALCULATOR and (memberships[0]["set_option"] or actor["rolconnlimit"] != 4))):
        raise ProvisionError("runtime_login_membership_mismatch")
    # Historical auditor SET TRUE is accepted by main; never alter its grant.
    return True


def _calculator_sql(snapshot: dict, password: str) -> str:
    # Compare the complete read preimage inside the transaction before the DDL.
    names = ",".join(_literal(x) for x in (*REQUIRED, *REQUIRED.values(), *OPTIONAL_GROUPS))
    expected = _literal(json.dumps(snapshot, sort_keys=True))
    return f"""BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('Codex-M:bolsa-material:v1',0));
DO $cas$ DECLARE actual jsonb; BEGIN
 SELECT jsonb_build_object(
 'roles',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY r.rolname) FROM
 (SELECT rolname,rolcanlogin,rolinherit,rolsuper,rolcreatedb,rolcreaterole,
 rolreplication,rolbypassrls,rolconnlimit,md5(COALESCE(rolpassword,'')) AS password_digest
 FROM pg_authid WHERE rolname IN ({names})) r),'[]'::jsonb),
 'memberships',COALESCE((SELECT jsonb_agg(to_jsonb(m) ORDER BY m.member,m.role) FROM
 (SELECT member.rolname AS member,g.rolname AS role,a.admin_option,a.inherit_option,a.set_option
 FROM pg_auth_members a JOIN pg_roles member ON member.oid=a.member JOIN pg_roles g ON g.oid=a.roleid
 WHERE member.rolname IN ({names}) OR g.rolname IN ({names})) m),'[]'::jsonb)) INTO actual;
 IF actual <> {expected}::jsonb THEN RAISE EXCEPTION 'bolsa material preimage changed'; END IF;
 IF to_regrole('{CALCULATOR}') IS NOT NULL THEN RAISE EXCEPTION 'calculator already exists'; END IF;
END $cas$;
CREATE ROLE {CALCULATOR} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE
 NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 4 PASSWORD {_literal(password)};
GRANT {CALCULATOR_GROUP} TO {CALCULATOR} WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
"""


def _provision_calculator(clone: Clone, private: Path) -> bool:
    before = clone.snapshot()
    for login, group in REQUIRED.items():
        _nominal(before, login, group, allow_absent=(login == CALCULATOR))
    exists = _nominal(before, CALCULATOR, CALCULATOR_GROUP, allow_absent=True)
    intent = private / "calculator-intent.json"
    if exists:
        # Existing LOGIN without this provisioner's private intent is accepted
        # without inventing or rotating its password (the historical kit uses trust).
        if intent.exists():
            _read_json(intent)
            receipt = private / "calculator-receipt.json"
            if receipt.exists():
                recorded = _read_json(receipt)
                calculator = next(r for r in before["roles"] if r["rolname"] == CALCULATOR)
                if recorded.get("calculator_password_digest") != calculator["password_digest"]:
                    raise ProvisionError("calculator_credential_preimage_changed")
        return False
    if intent.exists():
        stored = _read_json(intent)
        if stored.get("preimage_sha256") != _fingerprint(before):
            raise ProvisionError("calculator_intent_preimage_changed")
    else:
        stored = {"version": 1, "login": CALCULATOR,
                  "preimage_sha256": _fingerprint(before), "password": secrets.token_urlsafe(36)}
        _write_new(intent, stored)
    statement = _calculator_sql(before, stored["password"])
    clone.sql(statement + "ROLLBACK;\n")
    if clone.snapshot() != before:
        raise ProvisionError("calculator_rollback_changed_preimage")
    clone.sql(statement + "COMMIT;\n")
    after = clone.snapshot()
    _nominal(after, CALCULATOR, CALCULATOR_GROUP)
    _write_new(private / "calculator-receipt.json", {
        "version": 1, "preimage_sha256": _fingerprint(before),
        "postimage_sha256": _fingerprint(after), "rollback_verified": True,
        "login": CALCULATOR, "group": CALCULATOR_GROUP,
        "calculator_password_digest": next(r["password_digest"] for r in after["roles"] if r["rolname"] == CALCULATOR),
    })
    return True


def _historical_manifest(clone: Clone, material: Path) -> dict:
    identity = _read_json(material / "identidad/identidad.json")
    subject, certificate = identity.get("subject"), identity.get("certificate_sha256")
    if (identity.get("version") != 1 or identity.get("autoridad") != "no_autoritativo"
            or not isinstance(subject, str) or not subject or len(subject) > 128
            or not isinstance(certificate, str) or not re.fullmatch("[a-f0-9]{64}", certificate)):
        raise ProvisionError("rrhh_identity_manifest_invalid")
    cert = material / "mtls/cliente.crt"
    _check_path(cert, private=True)
    actual = subprocess.run(["openssl", "x509", "-in", str(cert), "-outform", "DER"],
                            capture_output=True, timeout=10, check=False)
    if actual.returncode or hashlib.sha256(actual.stdout).hexdigest() != certificate:
        raise ProvisionError("rrhh_certificate_fingerprint_mismatch")
    base = "vec.ct.alta.desarrollo.v1\0" + subject + "\0" + certificate + "\0perfil-bolsa-bback-v1"
    profile = "prf_" + hashlib.sha256(base.encode()).hexdigest()[:32]
    rows = json.loads(clone.sql("""SELECT COALESCE(json_agg(a.documento),'[]'::json)
      FROM vec_autorizacion.asignacion_perfil_actual v
      JOIN vec_autorizacion.asignacion_perfil a ON a.perfil_activo_ref=v.perfil_activo_ref
      AND a.asignacion_ref=v.asignacion_ref
      WHERE v.acto_ref='acto:bolsa:bback:asignacion:v1';"""))
    matching = [a for a in rows if a.get("perfil_activo_ref") == profile]
    if len(matching) > 1 or (matching and matching[0].get("estado") != "activa"):
        raise ProvisionError("rrhh_bback_historical_profile_mismatch")
    # A new local H1 profile is derived from the preserved local certificate.
    # The imported principal's other profile remains untouched. Main performs
    # the legitimate first publication; this helper only prepares its manifest.
    if not matching:
        if len(rows) != 1 or rows[0].get("estado") != "activa":
            raise ProvisionError("rrhh_bback_scope_source_ambiguous")
        matching = rows
    assignment = matching[0]
    scopes = {a["clave"]: a["valores"] for a in assignment.get("ambitos", [])}
    if any(len(scopes.get(k, [])) != 1 for k in ("unidad_ref", "ambito_ref")):
        raise ProvisionError("rrhh_bback_historical_scope_mismatch")
    references = json.loads(clone.sql("""SELECT COALESCE(json_agg(bolsa_ref ORDER BY bolsa_ref),'[]'::json)
      FROM vec_bolsa_llamamientos.bolsa_constituida WHERE estado='vigente';"""))
    if not references or len(references) > 256 or len(references) != len(set(references)):
        raise ProvisionError("bback_current_bolsas_missing")
    return {"version": 2, "autoridad": "no_autoritativo", "sujeto": subject,
            "certificado_sha256": certificate, "perfil_ref": profile,
            "unidad_ref": scopes["unidad_ref"][0], "ambito_ref": scopes["ambito_ref"][0],
            "bolsas_ref": references}


def _dsn(login: str, pg_port: int, ca: Path, password: str = "") -> str:
    authority = quote(login, safe="")
    if password:
        authority += ":" + quote(password, safe="")
    # pgx does not recognize libpq's ssl_min_protocol_version URL setting;
    # the clone server enforces TLS 1.3 and the live probes attest negotiation.
    return "postgresql://" + authority + f"@127.0.0.1:{pg_port}/postgres?" + urlencode({
        "sslmode": "verify-full", "sslrootcert": str(ca)})


def provision(repo: Path, container: str, state: Path, material: Path,
              pg_port: int, engine: str = "docker", *, source_context: dict | None = None) -> dict:
    """Return sensitive URLs to the caller only; CLI persists them privately."""
    result = {"env": {}, "profiles": {}, "blockers": []}
    try:
        if (engine not in ("docker", "podman")
                or not re.fullmatch(r"vec-[a-z0-9][a-z0-9-]{0,126}", container)
                or not state.is_absolute() or material != state / "material"
                or type(pg_port) is not int or not 1024 < pg_port <= 65535):
            raise ProvisionError("target_outside_authorized_clone")
        _check_path(state, private=True, directory=True)
        _check_path(material, private=True, directory=True)
        private = state / "bolsa-material"
        if not private.exists():
            private.mkdir(mode=0o700)
        _check_path(private, private=True, directory=True)
        lock_fd = os.open(private / "provision.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        with os.fdopen(lock_fd, "a") as lock, _sql_lock(state):
            _check_path(private / "provision.lock", private=True)
            fcntl.flock(lock, fcntl.LOCK_EX)
            clone = Clone(engine, container, state, pg_port)
            ready = clone.owner_guard()
            clone.authority = _sql_authority()
            clone.plan, git_repo = _approved_plan(repo, state, ready, source_context, clone.authority)
            _source_contracts(git_repo, clone.plan["source_ref"])
            clone.guard()
            # Actor/history is checked before any SQL mutation.
            manifest = _historical_manifest(clone, material)
            path = material / "identidad/bolsa-bback.json"
            if path.exists() and _read_json(path) != manifest:
                raise ProvisionError("bback_manifest_preimage_differs")
            ca = material / "pg/ca.crt"
            _check_path(ca, private=True)
            configured_ca = clone.sql("SHOW ssl_ca_file;")
            if (not configured_ca.startswith("/var/lib/postgresql/")
                    or "\n" in configured_ca or ".." in Path(configured_ca).parts):
                raise ProvisionError("clone_tls_ca_invalid")
            ca_bytes = subprocess.run([engine, "exec", container, "cat", configured_ca],
                                      capture_output=True, timeout=10, check=False)
            if ca_bytes.returncode or hashlib.sha256(ca_bytes.stdout).digest() != hashlib.sha256(ca.read_bytes()).digest():
                raise ProvisionError("clone_tls_ca_mismatch")
            _provision_calculator(clone, private)
            for login in REQUIRED:
                ssl = clone.sql("SELECT ssl::text||'|'||version FROM pg_stat_ssl WHERE pid=pg_backend_pid();",
                                user=login, tls_ca=configured_ca)
                if ssl != "true|TLSv1.3":
                    raise ProvisionError("runtime_tls_probe_failed")
            intent_path = private / "calculator-intent.json"
            password = _read_json(intent_path)["password"] if intent_path.exists() else ""
            _write_new(path, manifest)
            result["env"] = {
                "VEC_BOLSA_BORRADORES_ENABLED": "true",
                "VEC_BOLSA_POLITICA_OFERTAS_ENABLED": "true",
                "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL": _dsn(AUDIT, pg_port, ca),
                "VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL": _dsn(CALCULATOR, pg_port, ca, password),
            }
            result["profiles"] = {"bolsa_bback": {
                "profile_ref": manifest["perfil_ref"], "manifest": "material/identidad/bolsa-bback.json",
                "status": "prepared_for_bootstrap", "bolsas_count": len(manifest["bolsas_ref"]),
                "calculator_role_verified": True, "tls_probes": len(REQUIRED),
                "government_publication": "existing_main_bootstrap_required",
            }}
            absent = [g for g in OPTIONAL_GROUPS if g not in {r["rolname"] for r in clone.snapshot()["roles"]}]
            # These pools are a different, optional capability. Never synthesize their groups.
            if absent:
                result["profiles"]["bolsa_convocatorias"] = {
                    "status": "not_installed", "missing_groups": absent,
                    "blocks_bback_h5": False,
                }
            initial = private / "result-initial.json"
            if initial.exists():
                _read_json(initial)  # Preserve the first private preparation record.
            else:
                _write_new(initial, result)
    except (ProvisionError, OSError, ValueError, KeyError, TypeError, AttributeError, subprocess.TimeoutExpired) as exc:
        code = str(exc) if isinstance(exc, ProvisionError) else "bolsa_material_invalid_or_unavailable"
        result = {"env": {}, "profiles": {}, "blockers": [{"profile": "bolsa", "code": code}]}
    return result


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--container", required=True)
    parser.add_argument("--output", required=True, type=Path, help="Private state root")
    parser.add_argument("--pg-port", required=True, type=int)
    parser.add_argument("--engine", choices=("docker", "podman"), default="docker")
    parser.add_argument("--json-result-file", required=True, type=Path)
    args = parser.parse_args()
    # Validate output containment before calling provision (which can write SQL).
    if not args.output.is_absolute() or args.json_result_file.parent != args.output / "bolsa-material":
        parser.error("result file must be directly inside private bolsa-material")
    result = provision(args.repo, args.container, args.output, args.output / "material", args.pg_port, args.engine)
    try:
        _check_path(args.json_result_file.parent, private=True, directory=True)
        _write_new(args.json_result_file, result)
    except (ProvisionError, OSError):
        print("bolsa_material_result_not_saved")
        return 1
    print("bolsa_material_prepared" if not result["blockers"] else "bolsa_material_blocked")
    return bool(result["blockers"])


if __name__ == "__main__":
    raise SystemExit(main())
