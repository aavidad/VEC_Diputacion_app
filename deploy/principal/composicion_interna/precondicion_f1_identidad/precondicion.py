#!/usr/bin/env python3
"""Precondición transaccional de Contexto/F1 e Identidad 000004.

El modo por defecto solo inventaría. No contiene credenciales ni abre una red.
El operador proporciona una conexión psql ya autorizada o un contenedor local.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[4]
ROL = ROOT / "deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
IDENTIDAD = ROOT / "deploy/postgresql/identidad_sesiones_v1/migraciones/000004_revalidacion_contexto_corporativo_rrhh_v1.up.sql"
def migracion_sin_transaccion(path: Path) -> tuple[str, str]:
    data = path.read_bytes()
    inicio = re.search(rb"(?m)^BEGIN;\s*\n", data)
    if (inicio is None or not data.rstrip().endswith(b"COMMIT;")
            or any(not line.lstrip().startswith(b"--") and line.strip()
                   for line in data[:inicio.start()].splitlines())):
        raise ValueError(f"delimitadores transaccionales inesperados: {path.name}")
    body = re.sub(rb"\nCOMMIT;\s*\Z", b"\n", data[inicio.end() :])
    if re.search(rb"(?m)^\s*(BEGIN|COMMIT|ROLLBACK)\s*;", body):
        raise ValueError(f"transacción anidada inesperada: {path.name}")
    return body.decode("utf-8"), hashlib.sha256(data).hexdigest()


INVENTARIO = r"""
WITH roles AS (
 SELECT oid FROM pg_catalog.pg_roles
 WHERE rolname IN ('vec_contexto_actor_v1_propietario',
                   'vec_contexto_actor_v1_migrador',
                   'vec_contexto_actor_v1_runtime')
), membresias AS (
 SELECT r.rolname AS rol, m.rolname AS miembro, g.rolname AS otorgante,
        a.admin_option, a.inherit_option, a.set_option
 FROM pg_catalog.pg_auth_members a
 JOIN pg_catalog.pg_roles r ON r.oid=a.roleid
 JOIN pg_catalog.pg_roles m ON m.oid=a.member
 JOIN pg_catalog.pg_roles g ON g.oid=a.grantor
 WHERE a.roleid IN (SELECT oid FROM roles)
    OR a.member IN (SELECT oid FROM roles)
    OR a.grantor IN (SELECT oid FROM roles)
), tipos AS (
 SELECT n.nspname AS esquema, t.typname AS tipo, t.typtype,
        pg_catalog.pg_get_userbyid(t.typowner) AS propietario,
        true AS public_efectivo
 FROM pg_catalog.pg_type t
 JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
   AND EXISTS (
     SELECT 1 FROM pg_catalog.aclexplode(
       coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
     WHERE a.grantee=0)
), public_acl AS (
 SELECT 'base' AS clase, d.datname AS objeto
 FROM pg_catalog.pg_database d,
 LATERAL pg_catalog.aclexplode(coalesce(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
 WHERE NOT d.datistemplate AND a.grantee=0
 UNION ALL
 SELECT 'esquema', n.nspname FROM pg_catalog.pg_namespace n,
 LATERAL pg_catalog.aclexplode(coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' AND a.grantee=0
 UNION ALL
 SELECT 'relacion', n.nspname||'.'||c.relname
 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace,
 LATERAL pg_catalog.aclexplode(c.relacl) a
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' AND a.grantee=0
 UNION ALL
 SELECT 'columna', n.nspname||'.'||c.relname||'.'||att.attname
 FROM pg_catalog.pg_attribute att
 JOIN pg_catalog.pg_class c ON c.oid=att.attrelid
 JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace,
 LATERAL pg_catalog.aclexplode(att.attacl) a
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' AND a.grantee=0
 UNION ALL
 SELECT 'funcion', n.nspname||'.'||p.proname
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace,
 LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' AND a.grantee=0
 UNION ALL
 SELECT 'tipo', esquema||'.'||tipo FROM tipos
 UNION ALL
 SELECT 'acl_por_defecto', pg_catalog.pg_get_userbyid(d.defaclrole)||':'||
   coalesce(n.nspname,'*')||':'||d.defaclobjtype::text
 FROM pg_catalog.pg_default_acl d
 LEFT JOIN pg_catalog.pg_namespace n ON n.oid=d.defaclnamespace,
 LATERAL pg_catalog.aclexplode(d.defaclacl) a
 WHERE a.grantee=0
 UNION ALL
 SELECT 'politica_public', n.nspname||'.'||c.relname||'.'||p.polname
 FROM pg_catalog.pg_policy p
 JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
 JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE 0::oid=ANY(p.polroles)
 UNION ALL
 SELECT 'objeto_grande', lo.oid::text
 FROM pg_catalog.pg_largeobject_metadata lo,
 LATERAL pg_catalog.aclexplode(lo.lomacl) a WHERE a.grantee=0
 UNION ALL
 SELECT 'servidor_foraneo', s.srvname
 FROM pg_catalog.pg_foreign_server s,
 LATERAL pg_catalog.aclexplode(s.srvacl) a WHERE a.grantee=0
 UNION ALL
 SELECT 'envoltorio_foraneo', f.fdwname
 FROM pg_catalog.pg_foreign_data_wrapper f,
 LATERAL pg_catalog.aclexplode(f.fdwacl) a WHERE a.grantee=0
 UNION ALL
 SELECT 'tablespace', s.spcname
 FROM pg_catalog.pg_tablespace s,
 LATERAL pg_catalog.aclexplode(s.spcacl) a WHERE a.grantee=0
 UNION ALL
 SELECT 'parametro', p.parname
 FROM pg_catalog.pg_parameter_acl p,
 LATERAL pg_catalog.aclexplode(p.paracl) a WHERE a.grantee=0
)
SELECT pg_catalog.jsonb_build_object(
 'base',current_database(), 'version',current_setting('server_version_num'),
 'superusuario',(SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=current_user),
 'roles_contexto',(SELECT coalesce(jsonb_agg(rolname ORDER BY rolname),'[]'::jsonb)
                  FROM pg_catalog.pg_roles WHERE oid IN (SELECT oid FROM roles)),
 'membresias',(SELECT coalesce(jsonb_agg(to_jsonb(m) ORDER BY rol,miembro,otorgante),'[]'::jsonb)
                FROM membresias m),
 'tipos_public',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY esquema,tipo),'[]'::jsonb)
                FROM tipos t),
 'acl_public',(SELECT coalesce(jsonb_agg(to_jsonb(p) ORDER BY clase,objeto),'[]'::jsonb)
              FROM public_acl p),
 'selector_existente',pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NOT NULL,
 'fachada_existente',pg_catalog.to_regprocedure(
  'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)') IS NOT NULL
)::text;
"""


PREPARAR = r"""
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='5min';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended('vec:precondicion:f1-identidad:000004',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR NOT (SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=current_user)
    OR pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NOT NULL
    OR pg_catalog.to_regprocedure(
      'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'preimagen incompatible: versión, autoridad o 000004 ya instalada'; END IF;
 IF (SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN
    ('vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_migrador',
     'vec_contexto_actor_v1_runtime')) <> 3
 THEN RAISE EXCEPTION 'faltan roles de contexto'; END IF;
END $pre$;
CREATE TEMP TABLE vec_pre_membresias ON COMMIT DROP AS
 SELECT a.roleid,a.member,a.grantor,a.admin_option,a.inherit_option,a.set_option
 FROM pg_catalog.pg_auth_members a
 WHERE a.roleid IN ('vec_contexto_actor_v1_propietario'::regrole,
                    'vec_contexto_actor_v1_migrador'::regrole,
                    'vec_contexto_actor_v1_runtime'::regrole)
    OR a.member IN ('vec_contexto_actor_v1_propietario'::regrole,
                    'vec_contexto_actor_v1_migrador'::regrole,
                    'vec_contexto_actor_v1_runtime'::regrole)
    OR a.grantor IN ('vec_contexto_actor_v1_propietario'::regrole,
                    'vec_contexto_actor_v1_migrador'::regrole,
                    'vec_contexto_actor_v1_runtime'::regrole);
CREATE TEMP TABLE vec_pre_tipos ON COMMIT DROP AS
 SELECT t.oid,t.typacl
 FROM pg_catalog.pg_type t
 JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
 WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
   AND t.typtype='c' AND t.typrelid<>0
   AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(
     coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) a WHERE a.grantee=0);
DO $guardar$
DECLARE x record;
BEGIN
 IF (SELECT count(*) FROM vec_pre_tipos) <> 187
 THEN RAISE EXCEPTION 'tipos fila PUBLIC: cardinalidad distinta de 187'; END IF;
 IF (SELECT count(*) FROM vec_pre_membresias) <> 5
 OR NOT EXISTS (SELECT 1 FROM vec_pre_membresias
    WHERE roleid='vec_contexto_actor_v1_propietario'::regrole
      AND member='vec_contexto_actor_v1_migrador'::regrole
      AND grantor=10 AND NOT admin_option AND NOT inherit_option AND set_option)
 THEN RAISE EXCEPTION 'grafo de contexto distinto de 1+4 membresías'; END IF;
 IF EXISTS (SELECT 1 FROM vec_pre_membresias
     WHERE NOT (roleid='vec_contexto_actor_v1_propietario'::regrole
       AND member='vec_contexto_actor_v1_migrador'::regrole)
       AND (grantor<>10 OR roleid NOT IN
          ('vec_contexto_actor_v1_propietario'::regrole,
           'vec_contexto_actor_v1_migrador'::regrole,
           'vec_contexto_actor_v1_runtime'::regrole)
         OR NOT (SELECT rolcanlogin FROM pg_catalog.pg_roles WHERE oid=member)
         OR admin_option OR NOT inherit_option OR set_option))
 THEN RAISE EXCEPTION 'membresía extraña; revisar manifest antes de actuar'; END IF;
 IF EXISTS (
  SELECT 1 FROM pg_catalog.pg_type t
  JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
  CROSS JOIN LATERAL pg_catalog.aclexplode(
    coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
  WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
    AND a.grantee=0 AND NOT (t.typtype='c' AND t.typrelid<>0
      AND t.oid IN (SELECT oid FROM vec_pre_tipos)))
 THEN RAISE EXCEPTION 'PUBLIC en otro tipo: intervención ampliada prohibida'; END IF;
 FOR x IN SELECT roleid,member FROM vec_pre_membresias
          WHERE NOT (roleid='vec_contexto_actor_v1_propietario'::regrole
            AND member='vec_contexto_actor_v1_migrador'::regrole)
 LOOP
   EXECUTE pg_catalog.format('REVOKE %I FROM %I',
     (SELECT rolname FROM pg_catalog.pg_roles WHERE oid=x.roleid),
     (SELECT rolname FROM pg_catalog.pg_roles WHERE oid=x.member));
 END LOOP;
 FOR x IN SELECT t.oid,n.nspname,t.typname FROM vec_pre_tipos b
          JOIN pg_catalog.pg_type t ON t.oid=b.oid
          JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
 LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON TYPE %I.%I FROM PUBLIC',x.nspname,x.typname);
 END LOOP;
END $guardar$;
"""


FINALIZAR = r"""
DO $restaurar$
DECLARE x record;
BEGIN
 FOR x IN SELECT roleid,member,admin_option,inherit_option,set_option
          FROM vec_pre_membresias
          WHERE NOT (roleid='vec_contexto_actor_v1_propietario'::regrole
            AND member='vec_contexto_actor_v1_migrador'::regrole)
 LOOP
   EXECUTE pg_catalog.format('GRANT %I TO %I WITH ADMIN %s, INHERIT %s, SET %s',
     (SELECT rolname FROM pg_catalog.pg_roles WHERE oid=x.roleid),
     (SELECT rolname FROM pg_catalog.pg_roles WHERE oid=x.member),
     x.admin_option,x.inherit_option,x.set_option);
 END LOOP;
 IF EXISTS ((SELECT * FROM vec_pre_membresias EXCEPT ALL
             SELECT a.roleid,a.member,a.grantor,a.admin_option,a.inherit_option,a.set_option
             FROM pg_catalog.pg_auth_members a
             WHERE a.roleid IN ('vec_contexto_actor_v1_propietario'::regrole,
                                'vec_contexto_actor_v1_migrador'::regrole,
                                'vec_contexto_actor_v1_runtime'::regrole)
                OR a.member IN ('vec_contexto_actor_v1_propietario'::regrole,
                                'vec_contexto_actor_v1_migrador'::regrole,
                                'vec_contexto_actor_v1_runtime'::regrole)
                OR a.grantor IN ('vec_contexto_actor_v1_propietario'::regrole,
                                 'vec_contexto_actor_v1_migrador'::regrole,
                                 'vec_contexto_actor_v1_runtime'::regrole))
          UNION ALL
           (SELECT a.roleid,a.member,a.grantor,a.admin_option,a.inherit_option,a.set_option
             FROM pg_catalog.pg_auth_members a
             WHERE a.roleid IN ('vec_contexto_actor_v1_propietario'::regrole,
                                'vec_contexto_actor_v1_migrador'::regrole,
                                'vec_contexto_actor_v1_runtime'::regrole)
                OR a.member IN ('vec_contexto_actor_v1_propietario'::regrole,
                                'vec_contexto_actor_v1_migrador'::regrole,
                                'vec_contexto_actor_v1_runtime'::regrole)
                OR a.grantor IN ('vec_contexto_actor_v1_propietario'::regrole,
                                 'vec_contexto_actor_v1_migrador'::regrole,
                                 'vec_contexto_actor_v1_runtime'::regrole)
            EXCEPT ALL SELECT * FROM vec_pre_membresias))
 THEN RAISE EXCEPTION 'membresías no restauradas exactamente'; END IF;
 IF EXISTS (SELECT 1 FROM vec_pre_tipos b JOIN pg_catalog.pg_type t ON t.oid=b.oid
            CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
            WHERE a.grantee=0)
 THEN RAISE EXCEPTION 'PUBLIC conserva USAGE en tipo fila'; END IF;
 IF pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL
    OR pg_catalog.to_regprocedure(
      'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'faltan selector o fachada tras migraciones'; END IF;
 IF pg_catalog.has_schema_privilege('vec_contexto_actor_corporativo_rrhh_selector',
       'vec_identidad_sesiones_v1','USAGE')
    OR pg_catalog.has_schema_privilege('vec_contexto_actor_corporativo_rrhh_selector',
       'vec_identidad_sesiones_v1','CREATE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_corporativo_rrhh_selector',
       'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)',
       'EXECUTE')
    OR NOT pg_catalog.has_schema_privilege('vec_contexto_actor_v1_propietario',
       'vec_identidad_sesiones_v1','USAGE')
    OR NOT pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario',
       'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)',
       'EXECUTE')
 THEN RAISE EXCEPTION 'privilegios efectivos inesperados después de restaurar grafo'; END IF;
END $restaurar$;
SELECT 'VERIFICACION_POSTERIOR_OK';
"""


def ejecutar(args: argparse.Namespace, sql: str) -> str:
    comando = ["psql", "-XAtq", "--set=ON_ERROR_STOP=1", "--set=VERBOSITY=terse",
               "-d", args.database]
    if args.pg_container:
        comando = [args.container_engine, "exec", "-i", args.pg_container] + comando
    if args.admin_user:
        comando += ["-U", args.admin_user]
    result = subprocess.run(comando, input=sql, text=True, capture_output=True, check=False)
    if result.returncode:
        raise RuntimeError(f"psql falló ({result.returncode}): {result.stderr.strip()[-1500:]}")
    return result.stdout.strip()


def escribir_informe(path: Path, report: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC | getattr(os, "O_NOFOLLOW", 0)
    fd = os.open(path, flags, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as target:
        os.fchmod(target.fileno(), 0o600)
        json.dump(report, target, indent=2, ensure_ascii=False, sort_keys=True)
        target.write("\n")


def exigir_ruta_privada(path: Path) -> None:
    """Evita material sensible en Git y redirecciones por enlaces simbólicos."""
    for component in (path.absolute(), *path.absolute().parents):
        git = component / ".git"
        es_git = ((git.is_file() and git.read_bytes()[:7] == b"gitdir:")
                  or (git.is_dir() and (git / "HEAD").exists()
                      and ((git / "objects").exists() or (git / "commondir").exists())))
        if component.is_symlink() or git.is_symlink() or es_git:
            raise ValueError("las actas privadas deben estar fuera de Git y sin enlaces simbólicos")


def diagnosticar_check(args: argparse.Namespace) -> list[dict]:
    """Evalúa los CHECK vigentes sin leer ni emitir los recibos."""
    metadata = json.loads(ejecutar(args, """
SELECT coalesce(jsonb_agg(jsonb_build_object(
 'nombre',c.conname,'expresion',pg_catalog.pg_get_expr(c.conbin,c.conrelid),
 'validado',c.convalidated) ORDER BY c.conname),'[]'::jsonb)::text
FROM pg_catalog.pg_constraint c
WHERE c.conrelid=pg_catalog.to_regclass(
 'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2') AND c.contype='c';
"""))
    if not metadata:
        raise ValueError("no se encontraron CHECK de prueba_resultado_recibo_rrhh_v2")
    result = []
    for check in metadata:
        expression = check["expresion"]
        # La expresión sale del catálogo PostgreSQL, no de argumentos externos.
        sql = f"""BEGIN TRANSACTION READ ONLY;
WITH invalidas AS (
 SELECT acceso_ref FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
 WHERE NOT ({expression})
)
SELECT pg_catalog.jsonb_build_object(
 'filas_invalidas',(SELECT pg_catalog.count(*) FROM invalidas),
 'referencias_sha256',(SELECT coalesce(pg_catalog.jsonb_agg(
  pg_catalog.encode(pg_catalog.sha256(
   pg_catalog.convert_to(acceso_ref,'UTF8')),'hex')),'[]'::jsonb)
  FROM (SELECT acceso_ref FROM invalidas ORDER BY acceso_ref LIMIT 20) muestra))::text;
ROLLBACK;"""
        lines = ejecutar(args, sql).splitlines()
        row = json.loads(next(line for line in lines if line.startswith("{")))
        result.append({"nombre": check["nombre"], "validado": check["validado"], **row})
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("inspect", "rollback", "commit", "diagnose-check"), default="inspect")
    parser.add_argument("--database", required=True)
    parser.add_argument("--admin-user", default="postgres")
    parser.add_argument("--pg-container")
    parser.add_argument("--container-engine", choices=("docker", "podman"), default="docker")
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--expected-inventory-sha256", help="obligatorio para ensayo y COMMIT")
    parser.add_argument("--restoration-evidence", type=Path,
                        help="JSON privado de restauración integral comprobada, obligatorio para COMMIT")
    parser.add_argument("--rollback-report", type=Path,
                        help="informe del ensayo ROLLBACK sobre esta preimagen; obligatorio para COMMIT")
    args = parser.parse_args()
    for path in (args.report, args.restoration_evidence, args.rollback_report):
        if path is not None:
            exigir_ruta_privada(path)
    if args.mode in ("rollback", "commit") and not args.expected_inventory_sha256:
        parser.error("rollback/commit exige --expected-inventory-sha256")
    if args.mode == "commit" and (not args.restoration_evidence or not args.rollback_report):
        parser.error("COMMIT exige --restoration-evidence y --rollback-report")
    inventory = json.loads(ejecutar(args, INVENTARIO))
    canon = json.dumps(inventory, sort_keys=True, separators=(",", ":")).encode()
    inventory_hash = hashlib.sha256(canon).hexdigest()
    report = {"inventario": inventory, "inventario_sha256": inventory_hash,
              "modo": args.mode, "resultado": "preimagen capturada"}
    escribir_informe(args.report, report)
    if args.mode == "diagnose-check":
        report["checks"] = diagnosticar_check(args)
        report["resultado"] = "diagnóstico de solo lectura"
    if args.mode in ("rollback", "commit"):
        if inventory_hash != args.expected_inventory_sha256:
            raise ValueError("la preimagen ya no coincide con el inventario revisado")
        if args.mode == "commit":
            proof = json.loads(args.restoration_evidence.read_text(encoding="utf-8"))
            rollback = json.loads(args.rollback_report.read_text(encoding="utf-8"))
            if not (proof.get("verified") is True and proof.get("source_database") == args.database
                    and proof.get("source_inventory_sha256") == inventory_hash
                    and proof.get("restore_exit_zero") is True
                    and proof.get("check_validated") is True
                    and all(isinstance(proof.get(k), str)
                            and re.fullmatch(r"[0-9a-f]{64}", proof[k])
                            for k in ("backup_sha256", "restore_manifest_sha256"))):
                raise ValueError("falta restauración íntegra comprobada de esta preimagen")
            if (rollback.get("resultado") != "ensayo revertido"
                    or rollback.get("inventario_sha256") != inventory_hash):
                raise ValueError("falta ensayo ROLLBACK sobre esta misma preimagen")
        rol_sql, rol_hash = migracion_sin_transaccion(ROL)
        identidad_sql, identidad_hash = migracion_sin_transaccion(IDENTIDAD)
        report["migraciones_sha256"] = {ROL.name: rol_hash, IDENTIDAD.name: identidad_hash}
        if args.mode == "commit" and rollback.get("migraciones_sha256") != report["migraciones_sha256"]:
            raise ValueError("el ensayo usó otros bytes de las migraciones")
        tail = "COMMIT;" if args.mode == "commit" else "ROLLBACK;"
        script = "BEGIN;\n" + PREPARAR + "\n" + rol_sql + "\n" + identidad_sql
        script += "\n" + FINALIZAR + "\n" + tail + "\n"
        out = ejecutar(args, script)
        if "VERIFICACION_POSTERIOR_OK" not in out:
            raise RuntimeError("falta marcador de verificación anterior al cierre")
        posterior = json.loads(ejecutar(args, INVENTARIO))
        if args.mode == "rollback" and posterior != inventory:
            raise RuntimeError("ROLLBACK no restituyó el inventario exacto")
        if args.mode == "commit" and (
                posterior["membresias"] != inventory["membresias"]
                or posterior["tipos_public"]
                or not posterior["selector_existente"]
                or not posterior["fachada_existente"]):
            raise RuntimeError("postimagen confirmada distinta de la esperada")
        report["postinventario_sha256"] = hashlib.sha256(
            json.dumps(posterior, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        report["resultado"] = "confirmado" if args.mode == "commit" else "ensayo revertido"
    escribir_informe(args.report, report)
    print(json.dumps({"modo": args.mode, "resultado": report["resultado"],
                      "inventario_sha256": inventory_hash}))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, RuntimeError, json.JSONDecodeError) as exc:
        print(f"precondición rechazada: {exc}", file=sys.stderr)
        sys.exit(1)
