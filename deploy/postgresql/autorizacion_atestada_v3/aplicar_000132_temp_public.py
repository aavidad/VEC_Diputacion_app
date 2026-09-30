#!/usr/bin/env python3
"""Previsualiza y aplica AD3-132 con aprobación DBA de preimagen exacta.

El fichero de aprobación es privado. Este programa no acepta SQL como argumento
ni construye sentencias con nombres obtenidos del plan o del inventario.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import stat
import subprocess
import sys


HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
INVENTORY = HERE / "migraciones/000132_inventario_temp_public.sql"
MIGRATION = HERE / "migraciones/000132_cierre_temp_public_base_vec.up.sql"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
ROLE = re.compile(r"[a-z][a-z0-9_]{2,95}\Z")
REF = re.compile(r"[A-Za-z0-9._:/-]{8,120}\Z")
ANCHORS = [
    ["vec_autorizacion_atestada_v3", "vec_autorizacion_atestada_v3_propietario"],
    ["vec_contratacion_temporal", "vec_contratacion_temporal_propietario"],
    ["vec_identidad_sesiones_v1", "vec_identidad_sesiones_v1_propietario"],
]
SCHEMAS = [
    "vec_aspirantes|vec_aspirantes_propietario",
    "vec_autorizacion|vec_autorizacion_propietario",
    "vec_autorizacion_atestada_v3|vec_autorizacion_atestada_v3_propietario",
    "vec_bolsa_importacion_convoca|vec_bolsa_importacion_convoca_propietario",
    "vec_bolsa_llamamientos|vec_bolsa_llamamientos_propietario",
    "vec_calendarios|vec_calendarios_propietario",
    "vec_contexto_actor_v1|vec_contexto_actor_v1_propietario",
    "vec_contratacion_temporal|vec_contratacion_temporal_propietario",
    "vec_cronos_v1|vec_cronos_v1_propietario",
    "vec_dietas|vec_dietas_propietario",
    "vec_documentos|vec_documentos_propietario",
    "vec_identidad_externa_v1|vec_identidad_sesiones_v1_propietario",
    "vec_identidad_sesiones_v1|vec_identidad_sesiones_v1_propietario",
    "vec_personal|vec_personal_propietario",
    "vec_usuarios|vec_usuarios_propietario",
    "vec_usuarios_correos_avisos|vec_usuarios_correos_externo_propietario",
    "vec_usuarios_correos_externo|vec_usuarios_correos_externo_propietario",
    "vec_usuarios_correos_interno|vec_usuarios_correos_interno_propietario",
]
PUBLIC_RELATIONS = ["i:vectores_o2_05_pkey:postgres", "r:vectores_o2_05:postgres"]
PUBLIC_FUNCTIONS = [
    "aplicar_bundle_go_o2_05", "durabilizar_decision_o2_05",
    "exportar_entrada_go_o2_05", "invocar_vector_o2_05",
    "mutar_efecto_o2_05", "mutar_tipo_capacidad_o2_05", "preparar_vector_o2_05",
]


class Stop(Exception):
    pass


def sha_bytes(data):
    return hashlib.sha256(data).hexdigest()


def read_regular(path, maximum=4_000_000):
    path = Path(path)
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        before = os.fstat(descriptor)
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or before.st_size > maximum:
            raise Stop(f"fichero no regular o tamaño excesivo: {path}")
        with os.fdopen(os.dup(descriptor), "rb") as source:
            data = source.read(maximum + 1)
        after = os.fstat(descriptor)
        if len(data) != before.st_size or len(data) > maximum or \
                (before.st_dev, before.st_ino, before.st_size) != \
                (after.st_dev, after.st_ino, after.st_size):
            raise Stop(f"fichero cambió durante la lectura: {path}")
        return data
    finally:
        os.close(descriptor)


def json_file(path):
    try:
        value = json.loads(read_regular(path))
    except (ValueError, UnicodeError) as exc:
        raise Stop(f"JSON inválido: {path}") from exc
    if not isinstance(value, dict):
        raise Stop(f"JSON no es objeto: {path}")
    return value


def psql(args, sql_path=None, variables=(), sql_bytes=None):
    if not HEX64.fullmatch(args.container):
        raise Stop("ID de contenedor no canónico")
    engine = (["podman", "--remote=false"] if args.engine == "podman" else
              ["docker", "--host", "unix:///var/run/docker.sock"])
    command = [*engine, "exec", "-i", "--user", "postgres", args.container,
               "psql", "-X", "-w", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1",
               "-U", "postgres", "-d", "postgres"]
    for name, value in variables:
        command.extend(["-v", f"{name}={value}"])
    if (sql_path is None) == (sql_bytes is None):
        raise Stop("entrada SQL interna ambigua")
    statement = read_regular(sql_path) if sql_path is not None else sql_bytes
    local_env = {"PATH": "/usr/local/bin:/usr/bin:/bin",
                 "HOME": pwd.getpwuid(os.getuid()).pw_dir}
    runtime_dir = Path(f"/run/user/{os.getuid()}")
    if args.engine == "podman" and runtime_dir.is_dir():
        local_env["XDG_RUNTIME_DIR"] = str(runtime_dir)
    result = subprocess.run(command, input=statement, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=60, check=False,
                            env=local_env)
    if result.returncode:
        # PostgreSQL puede imprimir detalles del entorno; el operador consulta
        # su acta privada, mientras stdout de esta CLI no expone esos detalles.
        raise Stop(f"psql falló ({result.returncode}); sin cambios confirmados")
    return result.stdout.decode("utf-8", "strict").strip()


def current_inventory(args):
    lines = psql(args, INVENTORY).splitlines()
    if len(lines) != 1:
        raise Stop("inventario SQL inesperado")
    try:
        item = json.loads(lines[0])
    except ValueError as exc:
        raise Stop("inventario SQL inválido") from exc
    if not isinstance(item, dict):
        raise Stop("inventario SQL vacío")
    return item


def plan_logins(path):
    plan = json_file(path)
    connections = plan.get("conexiones") if isinstance(plan, dict) else None
    if not isinstance(connections, list) or not connections or not isinstance(plan.get("huellas"), dict) or \
            "incorporación/servidor.json" not in plan["huellas"]:
        raise Stop("plan H6 de conexiones inválido")
    pairs = set()
    for connection in connections:
        if not isinstance(connection, dict):
            raise Stop("conexión H6 inválida")
        login, group = connection.get("login"), connection.get("rol")
        if not isinstance(login, str) or not ROLE.fullmatch(login) or not isinstance(group, str) or not ROLE.fullmatch(group):
            raise Stop("LOGIN/grupo H6 inválido")
        pairs.add((login, group))
    if any(not isinstance(value, str) or not HEX64.fullmatch(value) for value in plan["huellas"].values()):
        raise Stop("huellas del plan H6 inválidas")
    return pairs


def source_file(relative):
    if not isinstance(relative, str) or not relative.startswith("deploy/postgresql/") or \
            Path(relative).is_absolute() or ".." in Path(relative).parts or \
            not relative.endswith(".sql"):
        raise Stop("ruta de provisión SQL fuera de VEC")
    # El kit se opera sin .git. Sólo acepta artefactos incluidos y fijados por
    # su SHA256; otra provisión se consigna como referencia privada verificable.
    path = ROOT / relative
    if not path.resolve().is_relative_to(ROOT.resolve()) or \
            any(parent.is_symlink() for parent in (path, *path.parents) if parent != ROOT and ROOT in parent.parents):
        raise Stop("provisión SQL con enlace o fuera del paquete")
    return read_regular(path)


def validate(args, inventory, pairs, approval=None):
    if inventory.get("database_name") != "postgres" or inventory.get("allowconn") is not True or \
            inventory.get("owner_name") != "postgres" or \
            inventory.get("vec_anchors") != ANCHORS or \
            inventory.get("non_system_schemas") != SCHEMAS or \
            inventory.get("public_relations") != PUBLIC_RELATIONS or \
            inventory.get("public_nonextension_functions") != PUBLIC_FUNCTIONS or \
            inventory.get("extensions") != [["pgcrypto", "public"], ["plpgsql", "pg_catalog"]] or \
            inventory.get("functions") != {"ad3_core": True, "identity_read": True}:
        raise Stop("base, dueño o anclas VEC incompatibles")
    if inventory.get("public_temp") is not True:
        raise Stop("PUBLIC TEMP ya ausente o preimagen distinta")
    for key in ("acl_sha256", "role_graph_sha256"):
        if not HEX64.fullmatch(str(inventory.get(key, ""))):
            raise Stop("huella de inventario inválida")
    logins = inventory.get("logins")
    if not isinstance(logins, list) or not all(isinstance(x, dict) for x in logins):
        raise Stop("inventario LOGIN inválido")
    indexed = {x.get("role"): x for x in logins}
    if len(indexed) != len(logins) or sorted(indexed) != inventory.get("login_roles"):
        raise Stop("inventario LOGIN incompleto")
    connect = {x["role"] for x in logins if x.get("connect") is True}
    if sorted(connect) != inventory.get("connect_roles"):
        raise Stop("inventario CONNECT incompleto")
    planned = {login for login, _ in pairs}
    if not planned <= connect:
        raise Stop("LOGIN del plan sin CONNECT")
    for login, group in pairs:
        row = indexed[login]
        if row.get("superuser") is True or group not in row.get("memberships", []) or \
                not row.get("vec_schema_usage"):
            raise Stop("LOGIN del plan sin grupo/ACL VEC exactos")
    extras = sorted(connect - planned - {inventory["owner_name"]})
    if approval is None:
        return extras
    if any(x.get("active_sessions") != 0 for x in logins if x["role"] != inventory["owner_name"]):
        raise Stop("hay sesiones LOGIN no DBA activas; cerrar solo las propias")
    if approval.get("approved") is not True or approval.get("approved_by") != inventory["owner_name"] or \
            not REF.fullmatch(str(approval.get("approval_ref", ""))):
        raise Stop("falta aprobación DBA positiva y referenciada")
    if approval.get("inventory") != inventory or approval.get("plan_sha256") != sha_bytes(read_regular(args.plan)) or \
            approval.get("sql_list_sha256") != sha_bytes(read_regular(args.sql_list)) or \
            approval.get("source_commit") != args.source_commit:
        raise Stop("aprobación no coincide con preimagen, plan, lista SQL o fuente")
    services = approval.get("other_vec_services")
    if not isinstance(services, list) or not all(isinstance(x, dict) and isinstance(x.get("role"), str)
                                                  for x in services) or \
            sorted(x.get("role") for x in services) != extras:
        raise Stop("servicios extra CONNECT sin clasificación gobernada exacta")
    for service in services:
        role = service["role"]
        row = indexed[role]
        purpose = service.get("purpose")
        schema_owners = {item.split("|", 1)[1] for item in SCHEMAS}
        can_migrate = purpose == "migrador_vec_inactivo" and any(
            group in schema_owners for group in row.get("memberships", []))
        if row.get("superuser") is True or not row.get("memberships") or \
                (not row.get("vec_schema_usage") and not can_migrate) or \
                purpose not in ("servicio_vec_inactivo", "migrador_vec_inactivo") or \
                service.get("approved_by") != inventory["owner_name"] or \
                service.get("approval_ref") != approval["approval_ref"]:
            raise Stop("servicio extra sin procedencia VEC/aprobación comprobable")
        relative = service.get("provision_sql")
        if relative:
            data = source_file(relative)
            if service.get("provision_sha256") != sha_bytes(data) or \
                    not any(re.search(r"\b" + re.escape(group) + r"\b", data.decode("utf-8", "strict"))
                            for group in row["memberships"]):
                raise Stop("provisión SQL del servicio extra no acredita su grupo")
        elif not REF.fullmatch(str(service.get("private_provision_ref", ""))):
            raise Stop("servicio extra sin referencia privada de provisión")
    if any(x.get("temp") is True for x in logins if x["role"] != inventory["owner_name"] and
           x["role"] not in connect):
        raise Stop("LOGIN sin CONNECT conserva TEMP y requiere investigación")
    return extras


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("preview", "trial", "apply"))
    parser.add_argument("--engine", choices=("docker", "podman"), required=True)
    parser.add_argument("--container", required=True, help="ID completo del contenedor PostgreSQL aislado")
    parser.add_argument("--plan", required=True, type=Path)
    parser.add_argument("--sql-list", required=True, type=Path)
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--approval", type=Path)
    args = parser.parse_args()
    if not HEX40.fullmatch(args.source_commit) or \
            not read_regular(args.sql_list) or \
            (args.mode == "preview" and (not args.output or args.approval)) or \
            (args.mode in ("trial", "apply") and (not args.approval or args.output)):
        raise Stop("argumentos de aprobación incompletos")
    pairs = plan_logins(args.plan)
    inventory = current_inventory(args)
    extras = validate(args, inventory, pairs)
    if args.mode == "preview":
        proposal = {
            "operation": "vec.h6.ad3_132.revoke_public_temp.v1",
            "approved": False,
            "approved_by": inventory["owner_name"],
            "approval_ref": "",
            "source_commit": args.source_commit,
            "sql_list_sha256": sha_bytes(read_regular(args.sql_list)),
            "plan_sha256": sha_bytes(read_regular(args.plan)),
            "inventory": inventory,
            "other_vec_services": [
                {"role": role, "purpose": "", "provision_sql": "",
                 "provision_sha256": "", "private_provision_ref": "",
                 "approved_by": "", "approval_ref": ""} for role in extras
            ],
        }
        args.output.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        descriptor = os.open(args.output, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        with os.fdopen(descriptor, "w", encoding="utf-8") as out:
            json.dump(proposal, out, ensure_ascii=False, indent=2, sort_keys=True)
            out.write("\n")
        print(f"PREVIEW-OK: {len(inventory['login_roles'])} LOGIN, {len(inventory['connect_roles'])} CONNECT, {len(extras)} extras; propuesta privada creada")
        return
    approval = json_file(args.approval)
    if args.approval.stat().st_mode & 0o077:
        raise Stop("aprobación privada legible por otros usuarios")
    validate(args, inventory, pairs, approval)
    selected = {
        "operation": approval["operation"],
        "database_name": inventory["database_name"],
        "database_oid": inventory["database_oid"],
        "owner_name": inventory["owner_name"],
        "owner_oid": inventory["owner_oid"],
        "acl_sha256": inventory["acl_sha256"],
        "role_graph_sha256": inventory["role_graph_sha256"],
        "connect_roles": inventory["connect_roles"],
        "login_roles": inventory["login_roles"],
        "plan_sha256": approval["plan_sha256"],
        "sql_list_sha256": approval["sql_list_sha256"],
        "source_commit": approval["source_commit"],
        "approved_by": approval["approved_by"],
        "approval_ref": approval["approval_ref"],
    }
    if selected["operation"] != "vec.h6.ad3_132.revoke_public_temp.v1":
        raise Stop("operación de aprobación distinta")
    variables = (("h6_approval", json.dumps(selected, separators=(",", ":"), sort_keys=True)),)
    if args.mode == "trial":
        original = read_regular(MIGRATION)
        if not original.endswith(b"\nCOMMIT;\n") or original.count(b"\nCOMMIT;\n") != 1:
            raise Stop("migración sin cierre transaccional canónico")
        trial = original[:-len(b"\nCOMMIT;\n")] + b"\nROLLBACK;\n"
        psql(args, variables=variables, sql_bytes=trial)
        if current_inventory(args) != inventory:
            raise Stop("ensayo ROLLBACK alteró la preimagen")
        print("AD3-132-ROLLBACK-OK: preimagen íntegra")
        return
    psql(args, MIGRATION, variables)
    after = current_inventory(args)
    if after.get("public_temp") is not False or \
            any(x.get("temp") is True for x in after.get("logins", []) if x.get("role") != after.get("owner_name")):
        raise Stop("postimagen TEMP incompatible tras COMMIT; detener arranque")
    print("AD3-132-OK: PUBLIC TEMP retirado; LOGIN sin TEMP efectivo")


if __name__ == "__main__":
    try:
        main()
    except (Stop, OSError, subprocess.TimeoutExpired) as exc:
        print(f"PARO AD3-132: {exc}", file=sys.stderr)
        raise SystemExit(1)
