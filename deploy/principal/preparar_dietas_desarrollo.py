#!/usr/bin/env python3
"""Activa Dietas R1 en el perfil privado de desarrollo.

Un único comando prepara SQL, identidad sintética, política y arranque.
No publica gobierno V3: Dietas firma con el gobierno único que Contratación
publica y renueva al arrancar (claves HMAC derivadas por audiencia). Publicar
otra configuración movería el puntero único y dejaría sin firma a CT y Bolsa.
No contiene valores de cidonia ni imprime DSN, claves o certificados.
"""

from __future__ import annotations

import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shlex
import shutil
import subprocess
import sys
import tempfile
import time
from urllib.parse import quote, urlsplit


ROOT = Path(__file__).resolve().parents[2]
MATERIAL = Path(os.environ.get("VEC_DEVELOPMENT_MATERIAL_DIR", str(Path.home() / ".local/state/vec-diputacion/desarrollo"))).resolve()
STATE = MATERIAL / "identidad/dietas-r1d-estado.json"
MANIFEST = MATERIAL / "identidad/dietas-comisiones.json"
ENGINE = os.environ.get("VEC_DIETAS_CONTAINER_ENGINE", "podman")
PG = os.environ.get("VEC_PRINCIPAL_POSTGRES_CONTAINER", "vec-postgresql-20260906")
APP = os.environ.get("VEC_PRINCIPAL_APP_CONTAINER", "vec-aplicacion-incorporacion-20260910")
DB = os.environ.get("VEC_DIETAS_DATABASE", "postgres")
DBUSER = os.environ.get("VEC_DIETAS_DATABASE_USER", "postgres")
# En la principal, arrancar_app.sh se ejecuta DENTRO del contenedor (bind mount
# /vec-arrancar.sh) y cuenta las conexiones que exporta arrancar-local.sh
# (/vec-conexiones-referencia.sh). La OSRM ya está configurada en él.
START = Path(os.environ.get("VEC_DIETAS_START_SCRIPT", str(Path.home() / ".local/state/vec-desarrollo-20260906/arrancar_app.sh")))
CONEXIONES = Path(os.environ.get("VEC_DIETAS_CONEXIONES_FILE", str(Path.home() / ".local/state/vec-desarrollo-20260906/material/arrancar-local.sh")))
CONTAINER_CA = "/vec-pg-ca.crt"
ENV_DSN = (("dietas", "VEC_DIETAS_BORRADORES_DATABASE_URL"), ("personal", "VEC_DIETAS_PERSONAL_RELACIONES_DATABASE_URL"))
ACTOR = "desarrollo:dietas-r1d:provisional"
DATE = dt.timezone.utc
ROLES = (
    ("registro_identidad", "vec_identidad_sesiones_v1_registrador"),
    ("revalidacion_identidad", "vec_identidad_sesiones_v1_revalidador"),
    ("contexto", "vec_contexto_actor_v1_runtime"),
    ("fuente_autorizacion", "vec_autorizacion_fuente"),
    ("registro_autorizacion", "vec_autorizacion_registro"),
    ("motivos", "vec_autorizacion_motivos_evaluador"),
    ("dietas", "vec_dietas_ejecutor"),
    ("personal", "vec_dietas_ejecutor"),
    ("auditoria_frontera", "vec_dietas_registrador_frontera"),
)


def fail(message: str) -> None:
    raise RuntimeError(message)


def run(args: list[str], data: str | bytes | None = None) -> bytes:
    if isinstance(data, str):
        data = data.encode()
    result = subprocess.run(args, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
    if result.returncode:
        # psql puede incluir valores de sentencias en su diagnóstico; sólo se
        # conserva en material privado y nunca se imprime en la terminal.
        if MATERIAL.is_dir():
            atomic_private(MATERIAL / "dietas-r1d-error.log", result.stderr.decode(errors="replace"))
        fail(f"falló {Path(args[0]).name} (código {result.returncode}); ver dietas-r1d-error.log en material privado")
    return result.stdout


def sql_literal(value: str) -> str:
    if "\x00" in value:
        fail("valor SQL inválido")
    return "'" + value.replace("'", "''") + "'"


def sql_json(value: object) -> str:
    return sql_literal(json.dumps(value, ensure_ascii=False, separators=(",", ":"))) + "::jsonb"


def psql(script: str, *, output: bool = False) -> str:
    result = run([ENGINE, "exec", "-i", PG, "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-U", DBUSER, "-d", DB, "-At"], script)
    return result.decode().strip() if output else ""


def query(sql: str) -> list[str]:
    return psql("SET search_path=pg_catalog;\n" + sql + "\n", output=True).splitlines()


def digest(value: bytes | str) -> str:
    return hashlib.sha256(value.encode() if isinstance(value, str) else value).hexdigest()


def ref(prefix: str, *parts: str) -> str:
    return prefix + digest("vec.dietas.r1d.v1\0" + "\0".join(parts))[:32]


def now_text(value: dt.datetime) -> str:
    value = value.astimezone(DATE)
    fraction = f".{value.microsecond:06d}".rstrip("0") if value.microsecond else ""
    return value.strftime("%Y-%m-%dT%H:%M:%S") + fraction + "Z"


def atomic_private(path: Path, contents: str) -> None:
    if path.is_symlink():
        fail(f"enlace simbólico privado rechazado: {path.name}")
    with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=path.parent, prefix=".dietas-r1d-", delete=False) as tmp:
        os.fchmod(tmp.fileno(), 0o600)
        tmp.write(contents)
        tmp.flush()
        os.fsync(tmp.fileno())
        temporary = Path(tmp.name)
    temporary.replace(path)


def private_json(path: Path) -> dict:
    if path.is_symlink() or not path.is_file() or path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o077:
        fail(f"material privado ausente o inseguro: {path.name}")
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        fail(f"material privado inválido: {path.name}")
    return data


def inspect(name: str) -> dict:
    data = json.loads(run([ENGINE, "inspect", name]))
    if not isinstance(data, list) or len(data) != 1:
        fail(f"contenedor no disponible: {name}")
    return data[0]


def dsn_template() -> tuple[str, str]:
    """Devuelve la conexión de CT tal como está escrita (con marcadores) y resuelta
    como la resuelve arrancar_app.sh dentro del contenedor."""
    if CONEXIONES.is_symlink() or not CONEXIONES.is_file() or CONEXIONES.stat().st_uid != os.getuid():
        fail("fichero de conexiones ausente o inseguro; fijar VEC_DIETAS_CONEXIONES_FILE")
    found = re.findall(r'^[ \t]*export[ \t]+VEC_CT_DATABASE_URL="([^"]+)"$', CONEXIONES.read_text(encoding="utf-8"), re.M)
    if len(found) != 1:
        fail("VEC_CT_DATABASE_URL no es inequívoca en el fichero de conexiones")
    raw = found[0]
    resolved = raw
    for marker, value in (("$vec_local_pg_puerto", "5432"), ("${vec_local_pg_puerto}", "5432"),
                          ("$vec_local_ca", CONTAINER_CA), ("${vec_local_ca}", CONTAINER_CA)):
        resolved = resolved.replace(marker, value)
    parsed = urlsplit(resolved)
    if "$" in resolved or parsed.scheme not in ("postgres", "postgresql") or not parsed.hostname or "sslmode=verify-full" not in parsed.query:
        fail("la conexión de referencia no es una URL PostgreSQL TLS verificable")
    return raw, resolved


def dsn_for(template: str, login: str, password: str) -> str:
    # Sustitución textual del usuario: la plantilla puede llevar marcadores
    # ($vec_local_pg_puerto) que urlsplit no admite como puerto.
    match = re.fullmatch(r"(postgres(?:ql)?://)(?:[^@/]*@)?(.+)", template)
    if not match:
        fail("plantilla de conexión inválida")
    return f"{match.group(1)}{quote(login, safe='')}:{quote(password, safe='')}@{match.group(2)}"


def starts() -> list[Path]:
    paths = [START, Path(str(START) + ".con-bback"), Path(str(START) + ".sin-bback")]
    for path in paths:
        if path.is_symlink() or not path.is_file() or path.stat().st_uid != os.getuid():
            fail(f"arranque ausente o inseguro: {path}")
        text = path.read_text(encoding="utf-8")
        if text.count("exec /usr/local/bin/vec-server") != 1 or len(re.findall(r'\[\[ "\$vec_conexiones" == (13|15) \]\]', text)) != 1:
            fail(f"arranque con forma inesperada (exec o recuento de conexiones): {path.name}")
    return paths


def backup_now(path: Path) -> None:
    # Copia del estado previo a ESTA ejecución: restaurar nunca deshace un
    # despliegue posterior a una ejecución anterior del script.
    atomic_private(MATERIAL / (path.name + ".antes-dietas-r1d"), path.read_text(encoding="utf-8"))


def patch_runtime(paths: list[Path], raw_dsns: dict[str, str]) -> list[Path]:
    """Añade las dos conexiones de Dietas y el selector. Devuelve lo cambiado."""
    changed = []
    text = CONEXIONES.read_text(encoding="utf-8")
    missing = [f'export {var}="{raw_dsns[name]}"' for name, var in ENV_DSN if f"export {var}=" not in text]
    if missing:
        backup_now(CONEXIONES)
        atomic_private(CONEXIONES, text.rstrip("\n") + "\n" + "\n".join(missing) + "\n")
        changed.append(CONEXIONES)
    for path in paths:
        text = path.read_text(encoding="utf-8")
        new = text.replace('[[ "$vec_conexiones" == 13 ]]', '[[ "$vec_conexiones" == 15 ]]')
        if "export VEC_DIETAS_BORRADORES_ENABLED=true" not in new:
            new = new.replace("exec /usr/local/bin/vec-server", "export VEC_DIETAS_BORRADORES_ENABLED=true\nexec /usr/local/bin/vec-server")
        if new != text:
            backup_now(path)
            atomic_private(path, new)
            os.chmod(path, 0o700)
            changed.append(path)
    return changed


def restore_runtime(changed: list[Path]) -> None:
    for path in changed:
        backup = MATERIAL / (path.name + ".antes-dietas-r1d")
        atomic_private(path, backup.read_text(encoding="utf-8"))
        if path != CONEXIONES:
            os.chmod(path, 0o700)


def restart_app() -> bool:
    since = dt.datetime.now(DATE).strftime("%Y-%m-%dT%H:%M:%SZ")
    run([ENGINE, "stop", "--time", "30", APP])
    run([ENGINE, "start", APP])
    for _ in range(30):
        time.sleep(2)
        logs = subprocess.run([ENGINE, "logs", "--since", since, APP], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False).stdout.decode(errors="replace")
        if "vec server listening" in logs:
            return True
    return False


def migration_state() -> bool:
    result = query("SELECT (to_regnamespace('vec_dietas') IS NOT NULL)::int || '|' || (to_regprocedure('vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL)::int;")
    if result == ["0|0"]:
        return False
    if result == ["1|1"]:
        # Una instalación anterior con las listas de campos antiguas (sin cálculo
        # en AD3-49 o con cuatro ámbitos en la huella) denegaría toda operación.
        current = query("""SELECT (strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'["comision.calculo",')>0
          AND strpos(pg_get_functiondef('vec_dietas.cotejar_recurso_dietas_borrador_v1(text,bytea,bytea,bytea)'::regprocedure),'"unidad_ref":')=0)::text;""")
        if current != ["true"]:
            fail("Dietas ya está instalado con listas de campos anteriores; requiere migración correctiva revisada")
        return True
    fail("migraciones de Dietas parciales; revisar sin reaplicar ni ejecutar DOWN")


def migrate() -> None:
    if not migration_state():
        assembled = run(["bash", str(ROOT / "deploy/principal/04_dietas_migraciones.sh")]).decode()
        if not assembled.endswith(":finalizar;\n"):
            fail("paquete de migraciones incompatible")
        for action in ("ROLLBACK", "COMMIT"):
            psql(assembled.replace(":finalizar;\n", action + ";\n"))
        if not migration_state():
            fail("migraciones de Dietas no quedaron instaladas")
    # 000001–000004 ya tienen historia en la principal. La nueva bitácora se
    # instala como delta y jamás se vuelven a aplicar las migraciones previas.
    firma = "vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)"
    vigente = query(f"SELECT (to_regprocedure({sql_literal(firma)}) IS NOT NULL)::int;")
    if vigente == ["1"]:
        return
    if vigente != ["0"]:
        fail("estado de auditoría Dietas incompatible")
    ruta = ROOT / "deploy/postgresql/dietas_borradores/migraciones/000005_auditoria_frontera.up.sql"
    texto = ruta.read_text(encoding="utf-8")
    if not texto.endswith("COMMIT;\n") or texto.count("\nCOMMIT;\n") != 1:
        fail("migración de auditoría Dietas con forma inesperada")
    for action in ("ROLLBACK", "COMMIT"):
        psql(texto[: -len("COMMIT;\n")] + action + ";\n")
    if query(f"SELECT (to_regprocedure({sql_literal(firma)}) IS NOT NULL)::int;") != ["1"]:
        fail("auditoría de frontera Dietas no quedó instalada")


def identity() -> dict:
    data = private_json(MATERIAL / "identidad/identidad.json")
    certificate = MATERIAL / "mtls/cliente.crt"
    if certificate.is_symlink() or not certificate.is_file():
        fail("certificado cliente de la demo ausente")
    der = run(["openssl", "x509", "-in", str(certificate), "-outform", "DER"])
    # subject es el principal del certificado (p. ej. desarrollo:…); la persona
    # (persona_ref per_…) es otra referencia y se toma del contexto de B-BACK.
    if data.get("certificate_sha256") != digest(der) or not re.fullmatch(r"[A-Za-z0-9_:.-]{3,256}", data.get("subject", "")):
        fail("la identidad privada no coincide con el certificado cliente")
    if data.get("autoridad") != "no_autoritativo":
        fail("la identidad de desarrollo no está rotulada como sintética")
    bback = private_json(MATERIAL / "identidad/bolsa-bback.json")
    if bback.get("sujeto") != data["subject"] or bback.get("certificado_sha256") != data["certificate_sha256"]:
        fail("la identidad de Bolsa/B-BACK no coincide con el certificado")
    data["perfil_bback"] = bback.get("perfil_ref", "")
    return data


def source_context(profile_bback: str) -> dict:
    rows = query(f"""SELECT jsonb_build_object('persona',v.persona_ref,'cuenta',v.cuenta_ref,'procedencia',v.procedencia_ref,
        'procedencia_version',v.procedencia_version,'procedencia_huella',v.procedencia_huella_sha256,
        'procedencia_autoridad',v.procedencia_autoridad,'vigente_hasta',v.vigente_hasta)::text
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
      WHERE v.perfil_ref={sql_literal(profile_bback)}
        AND v.estado='activo' AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta;""")
    if len(rows) != 1:
        fail("la identidad Bolsa/B-BACK no tiene un contexto de cuenta inequívoco")
    context = json.loads(rows[0])
    if not re.fullmatch(r"per_[A-Za-z0-9_-]{22,128}", context.get("persona", "")):
        fail("la persona del contexto B-BACK no tiene forma per_…")
    return context


def employee(person: str) -> str | None:
    # El empleado canónico procede de la proyección gobernada de Personal
    # (000016); ContextoActor 000007 cerró los punteros de empleado del núcleo.
    rows = query(f"""SELECT resultado || '|' || coalesce(empleado_ref,'')
      FROM vec_personal.resolver_empleado_canonico_persona_v1({sql_literal(person)},clock_timestamp());""")
    if len(rows) != 1:
        fail("Personal no devolvió una clase de proyección persona-empleado")
    resultado, emp = rows[0].split("|", 1)
    if resultado == "ambiguo":
        fail("la identidad tiene varios empleados canónicos en Personal; requiere resolución RRHH")
    return emp if resultado == "empleado" else None


def existing_relation(person: str, emp: str) -> dict | None:
    rows = query(f"""SELECT jsonb_build_object('relacion_ref',relacion_ref,'unidad_ref',unidad_ref,
      'desde',desde,'hasta',hasta)::text FROM vec_personal.relacion_empleado_dietas
      WHERE persona_ref={sql_literal(person)} AND empleado_ref={sql_literal(emp)} AND estado='activa'
        AND desde<=current_date AND (hasta IS NULL OR current_date<hasta);""")
    selected = os.environ.get("VEC_DIETAS_RELACION_REF", "")
    if selected:
        rows = [row for row in rows if json.loads(row)["relacion_ref"] == selected]
    if len(rows) > 1:
        fail("varias relaciones de Personal vigentes; fijar VEC_DIETAS_RELACION_REF")
    if selected and not rows:
        fail("VEC_DIETAS_RELACION_REF no corresponde a una relación vigente")
    return json.loads(rows[0]) if rows else None


def motive() -> dict:
    rows = query("""SELECT jsonb_build_object('catalogo_id',c.catalogo_id,'catalogo_version',c.catalogo_version,
      'catalogo_huella_sha256',c.catalogo_huella_publicada_sha256,'entrada_clave',e.entrada_clave)::text
      FROM vec_autorizacion.motivo_v2_catalogo_publicado c JOIN vec_autorizacion.motivo_v2_entrada e
      USING(catalogo_id,catalogo_version)
      WHERE e.vigente_desde<=clock_timestamp() AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta)
        AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
          WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version)
      ORDER BY c.catalogo_id,c.catalogo_version DESC,e.entrada_clave LIMIT 1;""")
    if len(rows) != 1:
        fail("no existe una entrada de motivo V3 publicada y vigente")
    return json.loads(rows[0])


def new_state(identity_data: dict, context: dict, emp: str | None, relation: dict | None, reason: dict) -> dict:
    person = context["persona"]
    certificate = identity_data["certificate_sha256"]
    return {"version": 2, "autoridad": "no_autoritativo", "person": person, "subject": identity_data["subject"], "certificate_sha256": certificate,
            "account": context["cuenta"], "employee": emp or ref("emp_", person, certificate),
            "employee_link_new": emp is None,
            "profile": ref("prf_", person, certificate, "perfil-dietas"),
            "context_link": ref("vca_", person, certificate, "contexto-dietas"),
            "employee_link": ref("vin_", person, certificate, "empleado-dietas"),
            "relation": relation["relacion_ref"] if relation else ref("rel_", person, certificate),
            "relation_new": relation is None,
            "unit": relation["unidad_ref"] if relation else "unidad:granada:dietas-provisional",
            "assignment": ref("ads_", person, certificate),
            "center": "centro:granada:dietas-provisional", "group": 2,
            "source_context": context, "motive": reason,
            "policy_issued": now_text(dt.datetime.now(DATE).replace(microsecond=0)),
            "policy_expires": now_text(dt.datetime.now(DATE).replace(microsecond=0) + dt.timedelta(days=180)),
            "login_passwords": {name: secrets.token_urlsafe(36) for name, _ in ROLES}}


def load_state(identity_data: dict, context: dict, emp: str | None, relation: dict | None, reason: dict) -> dict:
    if STATE.exists():
        state = private_json(STATE)
        if state.get("version") != 2 or state.get("person") != context["persona"] or state.get("subject") != identity_data["subject"] or state.get("certificate_sha256") != identity_data["certificate_sha256"] or state.get("account") != context["cuenta"] or state.get("employee") != (emp or state.get("employee")):
            fail("estado privado previo no corresponde a la identidad y cuenta actuales")
        passwords = state.get("login_passwords")
        if not isinstance(passwords, dict) or any(not isinstance(passwords.get(name), str) or not passwords[name] for name, _ in ROLES if name != "auditoria_frontera"):
            fail("LOGIN existentes de Dietas incompletos")
        if "auditoria_frontera" not in passwords:
            passwords["auditoria_frontera"] = secrets.token_urlsafe(36)
            atomic_private(STATE, json.dumps(state, ensure_ascii=False, separators=(",", ":")) + "\n")
        return state
    state = new_state(identity_data, context, emp, relation, reason)
    atomic_private(STATE, json.dumps(state, ensure_ascii=False, separators=(",", ":")) + "\n")
    return state


def logins(state: dict, template: str) -> dict[str, str]:
    names = {}
    # roles_up de Dietas no concedía CONNECT en las instalaciones anteriores:
    # sin él los LOGIN heredados de vec_dietas_ejecutor no pueden conectar.
    statements = ["BEGIN;", "SET LOCAL search_path=pg_catalog;", "SET LOCAL lock_timeout='5s';",
                  "DO $conectar$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_dietas_ejecutor', current_database()); END $conectar$;"]
    for name, role in ROLES:
        login = "vec_dietas_r1d_" + name + "_desarrollo"
        names[name] = login
        password = state["login_passwords"][name]
        statements.append(f"""DO $login$ BEGIN
          IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname={sql_literal(login)}) THEN
            EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS', {sql_literal(login)}, {sql_literal(password)});
            EXECUTE format('GRANT %I TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE', {sql_literal(role)}, {sql_literal(login)});
          END IF;
          IF NOT EXISTS (SELECT 1 FROM pg_roles r JOIN pg_auth_members m ON m.member=r.oid
            JOIN pg_roles parent ON parent.oid=m.roleid
            WHERE r.rolname={sql_literal(login)} AND r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
              AND NOT r.rolreplication AND NOT r.rolbypassrls
              AND parent.rolname={sql_literal(role)} AND NOT m.admin_option
              AND m.inherit_option AND NOT m.set_option
              AND (SELECT count(*) FROM pg_auth_members WHERE member=r.oid)=1)
          THEN RAISE EXCEPTION 'LOGIN Dietas ajeno o con privilegios adicionales' USING ERRCODE='42501'; END IF;
          EXECUTE format('ALTER ROLE %I PASSWORD %L', {sql_literal(login)}, {sql_literal(password)});
        END $login$;""")
    statements.append("FINALIZAR;")
    script = "\n".join(statements)
    for action in ("ROLLBACK", "COMMIT"):
        psql(script.replace("FINALIZAR;", action + ";"))
    return {name: dsn_for(template, login, state["login_passwords"][name]) for name, login in names.items()}


def logins_raw(state: dict, raw_template: str) -> dict[str, str]:
    return {name: dsn_for(raw_template, "vec_dietas_r1d_" + name + "_desarrollo", state["login_passwords"][name]) for name, _ in ROLES}


def provision_context_personal(state: dict) -> None:
    person, account, profile, employee = (state[key] for key in ("person", "account", "profile", "employee"))
    context = state["source_context"]
    source = (context["procedencia"], str(context["procedencia_version"]),
              context["procedencia_huella"], context["procedencia_autoridad"])
    source_sql = ",".join(sql_literal(value) if i != 1 else value for i, value in enumerate(source))
    expiry = context["vigente_hasta"]
    first = f"""BEGIN;
      SET LOCAL ROLE vec_contexto_actor_v1_propietario;
      SET LOCAL search_path=pg_catalog;
      SET LOCAL lock_timeout='5s';
      SELECT pg_advisory_xact_lock(hashtextextended({sql_literal('dietas-r1d:' + person)},0));
      INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
        ({sql_literal(profile)},1,{sql_literal(person)},{source_sql},'activo',clock_timestamp()-interval '1 hour',{sql_literal(expiry)}::timestamptz)
        ON CONFLICT DO NOTHING;
      INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ({sql_literal(profile)},1) ON CONFLICT DO NOTHING;
      INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
        ({sql_literal(state['context_link'])},1,{sql_literal(account)},{sql_literal(profile)},{sql_literal(person)},
         {source_sql},'activo',clock_timestamp()-interval '1 hour',{sql_literal(expiry)}::timestamptz)
        ON CONFLICT DO NOTHING;
      INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ({sql_literal(state['context_link'])},1)
        ON CONFLICT DO NOTHING;
    """
    first += f"""DO $check$ BEGIN
      IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
        WHERE a.perfil_ref={sql_literal(profile)} AND v.persona_ref={sql_literal(person)} AND v.estado='activo')
      OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
        WHERE a.vinculo_ref={sql_literal(state['context_link'])} AND v.cuenta_ref={sql_literal(account)} AND v.perfil_ref={sql_literal(profile)}
          AND v.persona_ref={sql_literal(person)} AND v.estado='activo')
      THEN RAISE EXCEPTION 'proyección de perfil Dietas incompatible' USING ERRCODE='55000'; END IF;
    END $check$;
    FINALIZAR;"""
    for action in ("ROLLBACK", "COMMIT"):
        psql(first.replace("FINALIZAR;", action + ";"))

    since = dt.datetime.now(DATE).date().isoformat()
    relation = state["relation"]
    unit = state["unit"]
    second = f"""BEGIN;
      SET LOCAL ROLE vec_personal_propietario;
      SET LOCAL search_path=pg_catalog;
      SET LOCAL lock_timeout='5s';
      SELECT set_config('vec.dietas.persona_ref',{sql_literal(person)},true);
    """
    if state["employee_link_new"]:
        # Proyección sintética persona→empleado publicada en Personal (nunca en
        # el núcleo). Idempotente: una segunda ejecución no la republica.
        projection = ref("pep_", person, state["certificate_sha256"], "empleado-dietas")
        second += f"""DO $proyeccion$ BEGIN
          IF NOT EXISTS(SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia h
            WHERE h.proyeccion_ref={sql_literal(projection)}) THEN
            PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1({sql_literal(projection)},1,
              {sql_literal(person)},{sql_literal(employee)},'activa',clock_timestamp()-interval '1 hour',
              {sql_literal(expiry)}::timestamptz,NULL,{sql_literal(context['procedencia'])},
              {int(context['procedencia_version'])},{sql_literal(context['procedencia_huella'])});
          END IF;
        END $proyeccion$;"""
    if state["relation_new"]:
        second += f"""INSERT INTO vec_personal.relacion_empleado_dietas
          (relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
          VALUES ({sql_literal(relation)},{sql_literal(person)},{sql_literal(employee)},{sql_literal(unit)},'activa',
            {sql_literal(since)}::date,NULL,1,'acto:dietas-r1d:relacion-provisional',
            'fuente:dietas-r1d:sintetica',1) ON CONFLICT DO NOTHING;"""
    second += f"""INSERT INTO vec_personal.asignacion_dietas
      (asignacion_ref,relacion_ref,persona_ref,unidad_ref,version,centro_ref,administrativo_persona_ref,
       responsable_persona_ref,grupo_dieta,vigente_desde,motivo_revision,procedencia_acto_ref,
       registrada_por_ref,registrada_en)
      VALUES ({sql_literal(state['assignment'])},{sql_literal(relation)},{sql_literal(person)},
       {sql_literal(unit)},1,{sql_literal(state['center'])},
       {sql_literal(ref('per_', person, 'administrativo-sintetico'))},
       {sql_literal(ref('per_', person, 'responsable-sintetico'))},
       {int(state['group'])},{sql_literal(since)}::date,'Asignación provisional sintética para demo Dietas',
       'acto:dietas-r1d:asignacion-provisional',{sql_literal(ACTOR)},clock_timestamp())
      ON CONFLICT DO NOTHING;
      DO $check$ BEGIN
        IF NOT EXISTS(SELECT 1 FROM vec_personal.relacion_empleado_dietas r
          WHERE r.relacion_ref={sql_literal(relation)} AND r.persona_ref={sql_literal(person)}
            AND r.empleado_ref={sql_literal(employee)} AND r.unidad_ref={sql_literal(unit)} AND r.estado='activa')
        OR NOT EXISTS(SELECT 1 FROM vec_personal.asignacion_dietas a
          WHERE a.asignacion_ref={sql_literal(state['assignment'])} AND a.relacion_ref={sql_literal(relation)}
            AND a.centro_ref={sql_literal(state['center'])} AND a.grupo_dieta={int(state['group'])})
        THEN RAISE EXCEPTION 'relación o asignación provisional incompatible' USING ERRCODE='55000'; END IF;
      END $check$;
      FINALIZAR;"""
    for action in ("ROLLBACK", "COMMIT"):
        psql(second.replace("FINALIZAR;", action + ";"))


# encoding/json no omite un time.Time cero aunque lleve omitempty, y sí omite
# listas vacías: los documentos reproducen exactamente json.Marshal de Go.
GO_ZERO_TIME = "0001-01-01T00:00:00Z"


def go_hash(document: dict) -> str:
    # El orden de inserción reproduce los campos de los tipos Go de autorización.
    return digest(json.dumps(document, ensure_ascii=False, separators=(",", ":")))


def policy(state: dict) -> None:
    person, employee, profile = (state[key] for key in ("person", "employee", "profile"))
    # Fechas fijadas en el estado privado: repetir el script reproduce los mismos
    # documentos y huellas, y ON CONFLICT DO NOTHING no choca con la comprobación.
    issued, expires = state["policy_issued"], state["policy_expires"]
    personal_fields = ["desde", "empleado_ref", "estado", "fuente_ref", "fuente_version", "hasta",
                       "persona_ref", "procedencia_acto_ref", "relacion_ref", "unidad_ref", "version"]
    commission_fields = ["comision.calculo", "comision.codigos_ruta", "comision.estado", "comision.fecha_fin",
                         "comision.fecha_inicio", "comision.motivo", "comision.referencia", "comision.relacion_ref",
                         "recibo.registrado_en", "recibo.referencia", "recibo.repeticion", "recibo.version"]
    commission_fields = sorted(commission_fields)
    grants = []
    for action, module, resource, purpose, fields in (
        ("personal.relacion.propia.consultar_dietas", "personal", "relacion_empleado_dietas", "preparar_borrador_dietas", personal_fields),
        ("dietas.borrador.propio.crear", "dietas", "comision_borrador", "crear_borrador_propio", commission_fields),
        # Una sola concesión sirve al listado y al detalle (el PDP no distingue
        # recursos en una acción); AD3-49 y Dietas 000004 exigen esta unión.
        ("dietas.borrador.propio.consultar", "dietas", "comision_borrador", "consultar_borrador_propio",
         sorted(commission_fields + ["items." + campo for campo in commission_fields] + ["siguiente_cursor"])),
    ):
        grants.append(dict(accion=action, modulo_id=module, tipo_recurso=resource,
                           finalidades=[purpose], garantia_minima="alto", campos_permitidos=fields))
    role_id = "dietas_r1d_provisional"
    role_ref = f"rol:{role_id}:v1"
    role_doc = dict(rol_id=role_id, version=1, nombre="Dietas R1 sintético provisional", estado="publicada",
                    concesiones=grants, publicada_por=ACTOR, publicada_en=issued,
                    retirada_en=GO_ZERO_TIME)
    control_doc = dict(version_rol_ref=role_ref, revision=1, estado="habilitada",
                       actualizado_por=ACTOR, actualizado_en=issued)
    assignment_id = "dietas_r1d_" + digest(profile)[:16]
    assignment_ref = f"asignacion:{assignment_id}:v1"
    assignment_doc = dict(asignacion_id=assignment_id, version=1, perfil_activo_ref=profile,
                          principal_id=person, version_rol_ref=role_ref, estado="activa",
                          ambitos=[{"clave": "empleado_ref", "valores": [employee]}, {"clave": "persona_ref", "valores": [person]}],
                          vigente_desde=issued, vigente_hasta=expires, emitida_por=ACTOR, emitida_en=issued,
                          revocada_en=GO_ZERO_TIME)
    sql = f"""BEGIN;
      SET LOCAL ROLE vec_autorizacion_propietario;
      SET LOCAL search_path=pg_catalog;
      SET LOCAL lock_timeout='5s';
      SELECT pg_advisory_xact_lock(hashtextextended('dietas-r1d:rbac',0));
      INSERT INTO vec_autorizacion.version_rol
        (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
      VALUES ({sql_literal(role_ref)},{sql_literal(role_id)},1,{sql_literal(go_hash(role_doc))},
        {sql_literal(issued)}::timestamptz,{sql_json(role_doc)}) ON CONFLICT DO NOTHING;
      INSERT INTO vec_autorizacion.control_vigencia_version_rol
        (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
      VALUES ({sql_literal(role_ref)},1,'habilitada',{sql_literal(go_hash(control_doc))},
        {sql_literal(issued)}::timestamptz,{sql_json(control_doc)}) ON CONFLICT DO NOTHING;
      INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
        (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
      VALUES ({sql_literal(role_ref)},1,{sql_literal(issued)}::timestamptz,
        {sql_literal(ACTOR)},'acto:dietas-r1d:control') ON CONFLICT DO NOTHING;
      INSERT INTO vec_autorizacion.asignacion_perfil
        (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
      VALUES ({sql_literal(assignment_ref)},{sql_literal(assignment_id)},1,
        {sql_literal(profile)},{sql_literal(person)},{sql_literal(role_ref)},
        {sql_literal(go_hash(assignment_doc))},{sql_literal(issued)}::timestamptz,
        {sql_json(assignment_doc)}) ON CONFLICT DO NOTHING;
      INSERT INTO vec_autorizacion.asignacion_perfil_actual
        (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
      VALUES ({sql_literal(profile)},{sql_literal(assignment_ref)},
        {sql_literal(issued)}::timestamptz,{sql_literal(ACTOR)},'acto:dietas-r1d:asignacion')
        ON CONFLICT DO NOTHING;
      DO $check$ BEGIN
        IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol
          WHERE version_rol_ref={sql_literal(role_ref)} AND huella_sha256={sql_literal(go_hash(role_doc))})
        OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual
          WHERE version_rol_ref={sql_literal(role_ref)} AND revision=1)
        OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
          WHERE perfil_activo_ref={sql_literal(profile)} AND asignacion_ref={sql_literal(assignment_ref)})
        OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil
          WHERE asignacion_ref={sql_literal(assignment_ref)} AND huella_sha256={sql_literal(go_hash(assignment_doc))})
        THEN RAISE EXCEPTION 'política V3 Dietas incompatible' USING ERRCODE='55000'; END IF;
      END $check$;
      FINALIZAR;"""
    for action in ("ROLLBACK", "COMMIT"):
        psql(sql.replace("FINALIZAR;", action + ";"))


def material_json(state: dict, dsns: dict[str, str]) -> str:
    result = {"version": 1, "autoridad": "no_autoritativo", "cuentas": [{"certificado_sha256": state["certificate_sha256"],
               "sujeto": state["subject"], "cuenta_ref": state["account"], "perfil_ref": state["profile"]}],
              "dsn_registro_identidad": dsns["registro_identidad"], "dsn_revalidacion_identidad": dsns["revalidacion_identidad"],
              "dsn_contexto": dsns["contexto"], "dsn_fuente_autorizacion": dsns["fuente_autorizacion"],
              "dsn_registro_autorizacion": dsns["registro_autorizacion"], "dsn_motivos": dsns["motivos"],
              "dsn_auditoria_frontera": dsns["auditoria_frontera"],
              "motivo_personal": state["motive"], "motivo_crear": state["motive"],
              "motivo_consultar": state["motive"]}
    return json.dumps(result, ensure_ascii=False, separators=(",", ":")) + "\n"


def write_runtime(state: dict, dsns: dict[str, str]) -> None:
    atomic_private(MANIFEST, material_json(state, dsns))


def main() -> None:
    if ENGINE not in ("podman", "docker") or not re.fullmatch(r"[A-Za-z0-9_.-]+", PG) or not re.fullmatch(r"[A-Za-z0-9_.-]+", APP):
        fail("motor o nombre de contenedor inválido")
    if not MATERIAL.is_dir() or MATERIAL.is_symlink() or MATERIAL.stat().st_uid != os.getuid() or MATERIAL.stat().st_mode & 0o077:
        fail("VEC_DEVELOPMENT_MATERIAL_DIR debe ser privado, propio y modo 0700")
    if ROOT == MATERIAL or ROOT in MATERIAL.parents:
        fail("el material privado no puede estar dentro de Git")
    for command in (ENGINE, "openssl", "bash"):
        if shutil.which(command) is None:
            fail(f"falta {command}")
    lock = MATERIAL / ".preparar-dietas-r1d.lock"
    with open(lock, "a+b") as handle:
        os.chmod(lock, 0o600)
        fcntl.flock(handle, fcntl.LOCK_EX)
        inspect(PG)
        inspect(APP)
        paths = starts()
        raw_template, template = dsn_template()
        person = identity()
        context = source_context(person["perfil_bback"])
        current_emp = employee(context["persona"])
        current_reason = motive()
        migrate()
        current_relation = existing_relation(context["persona"], current_emp) if current_emp else None
        state = load_state(person, context, current_emp, current_relation, current_reason)
        dsn = logins(state, template)
        raw = logins_raw(state, raw_template)
        provision_context_personal(state)
        policy(state)
        write_runtime(state, dsn)
        changed = patch_runtime(paths, raw)
        if not restart_app():
            restore_runtime(changed)
            restart_app()
            fail("la aplicación no arrancó con Dietas; se restauró el arranque anterior (ver podman logs)")
        print("Dietas R1D activada. Comprobación final (misma clave tras reiniciar = mismo recibo):")
        print("scripts/ensayar_dietas_r1.py con una clave de ensayo estable en VEC_DIETAS_TEST_KEY")
        print("Relación y grupo de la demo: provisionales, sintéticos; sin envío ni liquidación.")

if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, json.JSONDecodeError, RuntimeError) as error:
        print(f"DIETAS-R1D: {error}", file=sys.stderr)
        sys.exit(1)
