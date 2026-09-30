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
ANCHORS_QUERY = HERE / "000132_anclas_h6.sql"
ANCHORS_EXPECTED = HERE / "000132_anclas_h6.json"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
ROLE = re.compile(r"[a-z][a-z0-9_]{2,95}\Z")
REF = re.compile(r"[A-Za-z0-9._:/-]{8,120}\Z")
RELEASE_PATHS = {
    "sql": "deploy/postgresql/autorizacion_atestada_v3/migraciones/000132_cierre_temp_public_base_vec.up.sql",
    "cli": "deploy/postgresql/autorizacion_atestada_v3/aplicar_000132_temp_public.py",
    "inventory": "deploy/postgresql/autorizacion_atestada_v3/migraciones/000132_inventario_temp_public.sql",
    "doc": "deploy/postgresql/autorizacion_atestada_v3/000132_TEMP_PUBLIC.md",
    "anchors_query": "deploy/postgresql/autorizacion_atestada_v3/000132_anclas_h6.sql",
    "anchors_expected": "deploy/postgresql/autorizacion_atestada_v3/000132_anclas_h6.json",
}
NEW_FUNCTIONAL = [
    "deploy/postgresql/contratacion_temporal/migraciones/000145_enlace_firma_documento_custodiado.up.sql",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000125_consumidor_consulta_firmas_documento_ct.up.sql",
    "deploy/postgresql/contratacion_temporal/migraciones/000152_consulta_firmas_documento_atestada.up.sql",
    "deploy/postgresql/contratacion_temporal/migraciones/000153_perfil_reincorporacion_titular.up.sql",
]
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


class PsqlFailure(Stop):
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
        value = json.loads(read_regular(path), object_pairs_hook=unique_object)
    except (ValueError, UnicodeError) as exc:
        raise Stop(f"JSON inválido: {path}") from exc
    if not isinstance(value, dict):
        raise Stop(f"JSON no es objeto: {path}")
    return value


def canonical_json(data):
    return (json.dumps(data, sort_keys=True, separators=(",", ":"), ensure_ascii=False) + "\n").encode()


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise Stop("JSON con clave duplicada")
        result[key] = value
    return result


def lock_values(data):
    values = {}
    for line in data.decode("ascii", "strict").splitlines():
        parts = line.split()
        if len(parts) != 2 or parts[0] in values:
            raise Stop("release.lock duplicado o inválido")
        values[parts[0]] = parts[1]
    for key in ("COMMIT", "SQL_LIST_SHA256", "SQL_RELEASE_SHA256", "PAQUETE_SHA256"):
        if key not in values:
            raise Stop(f"release.lock sin {key}")
    return values


def package_file(relative):
    if not isinstance(relative, str) or Path(relative).is_absolute() or \
            ".." in Path(relative).parts or not relative or "//" in relative:
        raise Stop("ruta del paquete inválida")
    path = ROOT / relative
    if not path.resolve().is_relative_to(ROOT.resolve()) or \
            any(part.is_symlink() for part in (path, *path.parents)
                if part != ROOT and ROOT in part.parents):
        raise Stop("enlace o ruta fuera del paquete")
    return read_regular(path)


def load_release(args):
    if args.release_manifest.resolve() != (ROOT / "h6-sql-release.json").resolve():
        raise Stop("manifiesto fuera de la raíz del paquete")
    lock = lock_values(read_regular(args.release_lock, 64_000))
    if any(not HEX64.fullmatch(str(lock[key])) for key in
           ("SQL_LIST_SHA256", "SQL_RELEASE_SHA256", "PAQUETE_SHA256")):
        raise Stop("huellas de release.lock inválidas")
    manifest_bytes = read_regular(args.release_manifest, 1_000_000)
    if sha_bytes(manifest_bytes) != lock["SQL_RELEASE_SHA256"]:
        raise Stop("manifiesto SQL distinto de release.lock")
    try:
        manifest = json.loads(manifest_bytes, object_pairs_hook=unique_object)
    except ValueError as exc:
        raise Stop("manifiesto SQL inválido") from exc
    if not isinstance(manifest, dict) or manifest_bytes != canonical_json(manifest) or \
            manifest.get("version") != 1 or \
            not HEX40.fullmatch(str(manifest.get("source_commit", ""))) or \
            manifest["source_commit"] != lock["COMMIT"] or \
            not HEX64.fullmatch(str(manifest.get("sql_list_sha256", ""))) or \
            manifest["sql_list_sha256"] != lock["SQL_LIST_SHA256"]:
        raise Stop("manifiesto SQL sin identidad de fuente canónica")
    if read_regular(ROOT / "COMMIT", 100).decode().strip() != manifest["source_commit"]:
        raise Stop("COMMIT del paquete distinto del release")
    list_bytes = package_file("lista_sql_h6.txt")
    if sha_bytes(list_bytes) != manifest["sql_list_sha256"]:
        raise Stop("lista SQL distinta del release")
    try:
        lines = list_bytes.decode("ascii").splitlines()
    except UnicodeError as exc:
        raise Stop("lista SQL no canónica") from exc
    functions = manifest.get("functional_sql")
    if not list_bytes.endswith(b"\n") or b"\r" in list_bytes or b"\n\n" in list_bytes or \
            len(lines) != 45 or not isinstance(functions, list) or len(functions) != 45 or \
            len(set(lines)) != 45 or any(not isinstance(x, dict) for x in functions) or \
            [x.get("path") for x in functions] != lines or \
            any(not isinstance(x.get("sha256"), str) or not HEX64.fullmatch(x["sha256"])
                for x in functions):
        raise Stop("lista funcional H6 no es causal45 exacta")
    positions = [lines.index(name) if name in lines else -1 for name in NEW_FUNCTIONAL]
    if positions != sorted(positions) or any(x < 0 for x in positions):
        raise Stop("faltan CT145/AD125/CT152/CT153 en orden causal")
    for item in functions:
        path = item["path"]
        if not re.fullmatch(r"deploy/postgresql/[A-Za-z0-9_/-]+(?:\.up\.sql|_up\.sql)", path) or \
                sha_bytes(package_file(path)) != item["sha256"]:
            raise Stop("SQL funcional distinta de fuente fijada")
    ad132 = manifest.get("ad132")
    if not isinstance(ad132, dict) or set(ad132) != set(RELEASE_PATHS):
        raise Stop("artefactos AD3-132 incompletos")
    artifacts = {}
    for key, path in RELEASE_PATHS.items():
        item = ad132[key]
        if not isinstance(item, dict) or item.get("path") != path or \
                not isinstance(item.get("sha256"), str) or not HEX64.fullmatch(item["sha256"]):
            raise Stop(f"artefacto AD3-132 {key} sin pin")
        data = package_file(path)
        if sha_bytes(data) != item["sha256"]:
            raise Stop(f"artefacto AD3-132 {key} cambió")
        artifacts[key] = data
    for key, lock_key in (("sql", "AD132_SQL_SHA256"), ("cli", "AD132_CLI_SHA256"),
                          ("inventory", "AD132_INVENTARIO_SHA256"),
                          ("doc", "AD132_DOC_SHA256"),
                          ("anchors_query", "AD132_ANCLAS_SQL_SHA256"),
                          ("anchors_expected", "AD132_ANCLAS_JSON_SHA256")):
        if lock_key in lock and lock[lock_key] != ad132[key]["sha256"]:
            raise Stop(f"pin de {key} distinto en release.lock")
    try:
        anchors_expected = json.loads(artifacts["anchors_expected"], object_pairs_hook=unique_object)
    except ValueError as exc:
        raise Stop("anclas esperadas inválidas") from exc
    if not isinstance(anchors_expected, dict) or \
            artifacts["anchors_expected"] != canonical_json(anchors_expected) or \
            manifest.get("installed_anchors") != anchors_expected or \
            set(anchors_expected) != {"functions", "ct145_table"} or \
            not HEX64.fullmatch(str(anchors_expected["ct145_table"])) or \
            not isinstance(anchors_expected["functions"], dict) or \
            set(anchors_expected["functions"]) != {"ad125_core", "ad125_consumer",
                                                     "ct145_register", "ct145_read",
                                                     "ct152_attested_read",
                                                     "ct153_reincorporation"} or \
            any(not HEX64.fullmatch(str(value)) for value in anchors_expected["functions"].values()):
        raise Stop("anclas H6 sin postimagen exacta")
    return {"sha256": sha_bytes(manifest_bytes), "source_commit": manifest["source_commit"],
            "sql_list_sha256": manifest["sql_list_sha256"], "artifacts": artifacts,
            "package_sha256": lock["PAQUETE_SHA256"],
            "anchors_expected": anchors_expected,
            "artifact_sha256": {key: ad132[key]["sha256"] for key in RELEASE_PATHS}}


def psql(args, sql_bytes, variables=()):
    if not HEX64.fullmatch(args.container):
        raise Stop("ID de contenedor no canónico")
    engine = (["podman", "--remote=false"] if args.engine == "podman" else
              ["docker", "--host", "unix:///var/run/docker.sock"])
    command = [*engine, "exec", "-i", "--user", "postgres", args.container,
               "psql", "-X", "-w", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1",
               "-U", "postgres", "-d", "postgres"]
    for name, value in variables:
        command.extend(["-v", f"{name}={value}"])
    if not isinstance(sql_bytes, bytes):
        raise Stop("SQL interna no cargada")
    local_env = {"PATH": "/usr/local/bin:/usr/bin:/bin",
                 "HOME": pwd.getpwuid(os.getuid()).pw_dir}
    runtime_dir = Path(f"/run/user/{os.getuid()}")
    if args.engine == "podman" and runtime_dir.is_dir():
        local_env["XDG_RUNTIME_DIR"] = str(runtime_dir)
    result = subprocess.run(command, input=sql_bytes, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=60, check=False,
                            env=local_env)
    if result.returncode:
        raise PsqlFailure(f"psql terminó con código {result.returncode}")
    return result.stdout.decode("utf-8", "strict").strip()


def current_inventory(args, release):
    lines = psql(args, release["artifacts"]["inventory"]).splitlines()
    if len(lines) != 1:
        raise Stop("inventario SQL inesperado")
    try:
        item = json.loads(lines[0])
    except ValueError as exc:
        raise Stop("inventario SQL inválido") from exc
    if not isinstance(item, dict):
        raise Stop("inventario SQL vacío")
    return item


def current_anchors(args, release):
    lines = psql(args, release["artifacts"]["anchors_query"]).splitlines()
    if len(lines) != 1:
        raise Stop("anclas SQL inesperadas")
    try:
        item = json.loads(lines[0])
    except ValueError as exc:
        raise Stop("anclas SQL inválidas") from exc
    if item != release["anchors_expected"]:
        raise Stop("CT145/AD125/CT152 no instaladas con postimagen fijada")
    return item


def plan_logins(path):
    data = read_regular(path)
    try:
        plan = json.loads(data, object_pairs_hook=unique_object)
    except ValueError as exc:
        raise Stop("plan H6 de conexiones inválido") from exc
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
    return pairs, sha_bytes(data), plan


def load_plan_receipt(args, release, plan_sha, plan):
    receipt_path = args.plan_receipt
    if args.plan.name != "plan-conexiones.json" or \
            receipt_path.parent.resolve() != args.plan.parent.resolve() or \
            receipt_path.name not in ("plan-canonico-clon.json", "plan-canonico-principal.json"):
        raise Stop("recibo de canario fuera de ruta fijada")
    if receipt_path.stat().st_mode & 0o077 or \
            receipt_path.stat().st_uid != os.getuid() or \
            args.plan.stat().st_uid != os.getuid() or \
            args.plan.stat().st_mode & 0o022:
        raise Stop("plan o recibo sin dueño y modo privado esperados")
    receipt_bytes = read_regular(receipt_path, 100_000)
    try:
        receipt = json.loads(receipt_bytes, object_pairs_hook=unique_object)
    except ValueError as exc:
        raise Stop("recibo de canario inválido") from exc
    if not isinstance(receipt, dict) or receipt_bytes != canonical_json(receipt):
        raise Stop("recibo de canario no canónico")
    kind = receipt.get("kind")
    common = {"version", "kind", "package_sha256", "source_commit",
              "pg_container_id", "plan_sha256", "material_inventory_sha256",
              "canary_image_id", "arranque_sha256", "canary_output_sha256"}
    if kind not in ("clon", "principal") or \
            set(receipt) != (common | ({"origin_receipt_sha256"} if kind == "principal" else set())) or \
            receipt_path.name != f"plan-canonico-{kind}.json" or \
            type(receipt["version"]) is not int or receipt["version"] != 1 or \
            receipt["package_sha256"] != release["package_sha256"] or \
            receipt["source_commit"] != release["source_commit"] or \
            receipt["pg_container_id"] != args.container or \
            receipt["plan_sha256"] != plan_sha or \
            receipt["material_inventory_sha256"] != sha_bytes(canonical_json(plan["huellas"])) or \
            receipt["canary_output_sha256"] != plan_sha or \
            not HEX64.fullmatch(str(receipt["arranque_sha256"])):
        raise Stop("recibo de canario no liga plan, material, paquete y base")
    image_id = receipt["canary_image_id"]
    if not isinstance(image_id, str) or not image_id.startswith("sha256:") or \
            not HEX64.fullmatch(image_id[7:]):
        raise Stop("imagen de canario no fijada")
    if kind == "principal":
        origin_path = receipt_path.with_name("plan-canonico-clon.json")
        if origin_path.stat().st_mode & 0o077 or origin_path.stat().st_uid != os.getuid():
            raise Stop("recibo clon sin dueño y modo privado esperados")
        origin_bytes = read_regular(origin_path, 100_000)
        if sha_bytes(origin_bytes) != receipt["origin_receipt_sha256"]:
            raise Stop("recibo principal no deriva del clon fijado")
        try:
            origin = json.loads(origin_bytes, object_pairs_hook=unique_object)
        except ValueError as exc:
            raise Stop("recibo clon inválido") from exc
        if not isinstance(origin, dict) or origin_bytes != canonical_json(origin) or \
                set(origin) != common or origin.get("version") != 1 or \
                origin.get("kind") != "clon" or \
                not HEX64.fullmatch(str(origin.get("pg_container_id"))) or \
                any(origin.get(field) != receipt.get(field) for field in
                    ("package_sha256", "source_commit", "plan_sha256",
                     "material_inventory_sha256", "canary_image_id",
                     "arranque_sha256", "canary_output_sha256")) or \
                origin.get("pg_container_id") == args.container:
            raise Stop("recibo principal sin origen clon coherente")
    return sha_bytes(receipt_bytes)


def validate(inventory, pairs, plan_sha, receipt_sha, release, anchors, approval=None):
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
    if approval.get("inventory") != inventory or approval.get("plan_sha256") != plan_sha or \
            approval.get("plan_receipt_sha256") != receipt_sha or \
            approval.get("release_sha256") != release["sha256"] or \
            approval.get("sql_list_sha256") != release["sql_list_sha256"] or \
            approval.get("source_commit") != release["source_commit"] or \
            approval.get("artifact_sha256") != release["artifact_sha256"] or \
            approval.get("installed_anchors") != anchors or \
            approval.get("unexplained_connect_logins") != extras:
        raise Stop("aprobación no coincide con código, SQL, anclas o preimagen")
    if extras:
        raise Stop("LOGIN CONNECT fuera del plan H6 sin procedencia acreditada")
    if any(x.get("temp") is True for x in logins if x["role"] != inventory["owner_name"] and
           x["role"] not in connect):
        raise Stop("LOGIN sin CONNECT conserva TEMP y requiere investigación")
    return extras


def postimage_compatible(before, after):
    preserved = ("database_name", "database_oid", "owner_name", "owner_oid",
                 "allowconn", "role_graph_sha256", "connect_roles", "login_roles",
                 "vec_anchors", "non_system_schemas", "public_relations",
                 "public_nonextension_functions", "extensions", "functions",
                 "public_connect")
    def stable_logins(snapshot):
        return [{key: value for key, value in row.items()
                 if key not in ("temp", "active_sessions")}
                for row in snapshot.get("logins", [])]
    return (all(before.get(key) == after.get(key) for key in preserved) and
            stable_logins(before) == stable_logins(after) and
            before.get("public_temp") is True and after.get("public_temp") is False and
            before.get("acl_sha256") != after.get("acl_sha256") and
            all(x.get("temp") is False for x in after.get("logins", [])
                if x.get("role") != after.get("owner_name")))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("preview", "trial", "apply"))
    parser.add_argument("--engine", choices=("docker", "podman"), required=True)
    parser.add_argument("--container", required=True, help="ID completo del contenedor PostgreSQL aislado")
    parser.add_argument("--plan", required=True, type=Path)
    parser.add_argument("--plan-receipt", required=True, type=Path)
    parser.add_argument("--release-manifest", required=True, type=Path)
    parser.add_argument("--release-lock", required=True, type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--approval", type=Path)
    args = parser.parse_args()
    if (args.mode == "preview" and (not args.output or args.approval)) or \
            (args.mode in ("trial", "apply") and (not args.approval or args.output)):
        raise Stop("argumentos de aprobación incompletos")
    release = load_release(args)
    pairs, plan_sha, plan = plan_logins(args.plan)
    receipt_sha = load_plan_receipt(args, release, plan_sha, plan)
    inventory = current_inventory(args, release)
    anchors = current_anchors(args, release)
    extras = validate(inventory, pairs, plan_sha, receipt_sha, release, anchors)
    if args.mode == "preview":
        proposal = {
            "operation": "vec.h6.ad3_132.revoke_public_temp.v1",
            "approved": False,
            "approved_by": inventory["owner_name"],
            "approval_ref": "",
            "source_commit": release["source_commit"],
            "sql_list_sha256": release["sql_list_sha256"],
            "release_sha256": release["sha256"],
            "artifact_sha256": release["artifact_sha256"],
            "plan_sha256": plan_sha,
            "plan_receipt_sha256": receipt_sha,
            "inventory": inventory,
            "installed_anchors": anchors,
            "unexplained_connect_logins": extras,
        }
        args.output.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        descriptor = os.open(args.output, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        with os.fdopen(descriptor, "w", encoding="utf-8") as out:
            json.dump(proposal, out, ensure_ascii=False, indent=2, sort_keys=True)
            out.write("\n")
        print(f"PREVIEW: {len(inventory['login_roles'])} LOGIN, {len(inventory['connect_roles'])} CONNECT, {len(extras)} sin procedencia; propuesta privada creada")
        return
    approval = json_file(args.approval)
    if args.approval.stat().st_mode & 0o077:
        raise Stop("aprobación privada legible por otros usuarios")
    validate(inventory, pairs, plan_sha, receipt_sha, release, anchors, approval)
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
        "plan_sha256": plan_sha,
        "sql_list_sha256": release["sql_list_sha256"],
        "source_commit": release["source_commit"],
        "approved_by": approval["approved_by"],
        "approval_ref": approval["approval_ref"],
    }
    if selected["operation"] != "vec.h6.ad3_132.revoke_public_temp.v1":
        raise Stop("operación de aprobación distinta")
    variables = (("h6_approval", json.dumps(selected, separators=(",", ":"), sort_keys=True)),)
    if args.mode == "trial":
        original = release["artifacts"]["sql"]
        if not original.endswith(b"\nCOMMIT;\n") or original.count(b"\nCOMMIT;\n") != 1:
            raise Stop("migración sin cierre transaccional canónico")
        trial = original[:-len(b"\nCOMMIT;\n")] + b"\nROLLBACK;\n"
        psql(args, variables=variables, sql_bytes=trial)
        if current_inventory(args, release) != inventory or current_anchors(args, release) != anchors:
            raise Stop("ensayo ROLLBACK alteró la preimagen")
        print("AD3-132-ROLLBACK-OK: preimagen íntegra")
        return
    try:
        psql(args, release["artifacts"]["sql"], variables)
        after = current_inventory(args, release)
        if not postimage_compatible(inventory, after) or \
                current_anchors(args, release) != anchors:
            raise Stop("postimagen TEMP incompatible")
    except (Stop, OSError, subprocess.TimeoutExpired) as exc:
        observed = "no consultable"
        try:
            snapshot = current_inventory(args, release)
            if snapshot == inventory:
                observed = "preimagen observada"
            elif postimage_compatible(inventory, snapshot) and current_anchors(args, release) == anchors:
                observed = "postimagen compatible observada"
            else:
                observed = "postimagen divergente"
        except (Stop, OSError, subprocess.TimeoutExpired):
            pass
        raise Stop(f"resultado de aplicación incierto ({observed}): conciliar ACL, roles y anclas antes de reintentar o arrancar") from exc
    print("AD3-132-OK: PUBLIC TEMP retirado; LOGIN sin TEMP efectivo")


if __name__ == "__main__":
    try:
        main()
    except (Stop, OSError, subprocess.TimeoutExpired) as exc:
        print(f"PARO AD3-132: {exc}", file=sys.stderr)
        raise SystemExit(1)
