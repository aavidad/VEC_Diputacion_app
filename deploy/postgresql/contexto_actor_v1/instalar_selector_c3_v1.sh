#!/usr/bin/env bash
set -euo pipefail

# Destino: exclusivamente PostgreSQL dentro de un contenedor local explícito.
# Nunca recibe DSN ni contraseñas; --inspect fija la preimagen para --apply.
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
modo='' contenedor='' base='' usuario='' esperado=''
uso() {
  echo 'uso: instalar_selector_c3_v1.sh --inspect|--apply --container NOMBRE --database BASE --admin-user USUARIO [--expected-preimage-sha256 SHA256]' >&2
  exit 2
}
while [[ $# -gt 0 ]]; do
  case $1 in
    --inspect|--apply) [[ -z $modo ]] || uso; modo=${1#--}; shift ;;
    --container|--database|--admin-user|--expected-preimage-sha256)
      [[ $# -ge 2 ]] || uso
      case $1 in
        --container) [[ -z $contenedor ]] || uso; contenedor=$2 ;;
        --database) [[ -z $base ]] || uso; base=$2 ;;
        --admin-user) [[ -z $usuario ]] || uso; usuario=$2 ;;
        --expected-preimage-sha256) [[ -z $esperado ]] || uso; esperado=$2 ;;
      esac
      shift 2 ;;
    *) uso ;;
  esac
done
[[ $modo == inspect || $modo == apply ]] || uso
[[ $contenedor =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ && ${contenedor,,} != *cidonia* ]] || uso
[[ $base =~ ^[A-Za-z_][A-Za-z0-9_]{0,62}$ && $usuario =~ ^[A-Za-z_][A-Za-z0-9_]{0,62}$ ]] || uso
if [[ $modo == apply ]]; then
  [[ $esperado =~ ^[0-9a-f]{64}$ ]] || uso
else
  [[ -z $esperado ]] || uso
fi

contexto=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)
[[ $contexto == unix:///var/run/docker.sock || $contexto =~ ^unix:///run/user/[0-9]+/docker\.sock$ ]] || {
  echo 'C3 ACL: Docker no usa un socket local admitido' >&2; exit 1
}
docker_local() { docker --host "$contexto" "$@"; }
docker_local container inspect "$contenedor" >/dev/null 2>&1 || {
  echo 'C3 ACL: contenedor explícito no disponible' >&2; exit 1
}
psql_destino() {
  docker_local exec -i -e PGAPPNAME=vec-c3-acl-installer "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 \
    -h /var/run/postgresql -U "$usuario" -d "$base"
}
rls_b1_cerrada() {
  consulta_rls_b1 | psql_destino
}
consulta_rls_b1() {
  cat <<'SQL'
WITH nombres(nombre) AS (
 SELECT unnest(ARRAY[
  'bolsa_autoritativa','necesidad_autoritativa','necesidad_actual',
  'politica_autoritativa','instantanea_autoritativa',
  'evaluacion_autoritativa','atestacion_autorizacion_version',
  'atestacion_autorizacion_actual','propuesta','referencia_consumida',
  'uso_decision','auditoria','auditoria_actual','outbox'
 ])
)
SELECT (pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario') IS NOT NULL
        AND count(*)=14 AND coalesce(bool_and(
          c.oid IS NOT NULL AND c.relkind='r'
          AND c.relowner=pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario')::oid
          AND c.relrowsecurity AND c.relforcerowsecurity
          AND NOT EXISTS (
            SELECT 1 FROM pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) AS a
             WHERE a.grantee<>c.relowner
          )
          AND NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_attribute AS x
            CROSS JOIN LATERAL pg_catalog.aclexplode(x.attacl) AS a
             WHERE x.attrelid=c.oid AND a.grantee<>c.relowner
          )
          AND (SELECT count(*) FROM pg_catalog.pg_policy q WHERE q.polrelid=c.oid)=1
          AND p.polname='solo_propietario' AND p.polcmd='*' AND p.polpermissive
          AND p.polroles=ARRAY[pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario')::oid]
          AND p.polqual IS NOT NULL AND p.polwithcheck IS NOT NULL
          AND pg_catalog.pg_get_expr(p.polqual,p.polrelid)
              = '(CURRENT_USER = ''vec_bolsa_llamamientos_propietario''::name)'
          AND pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)
              = '(CURRENT_USER = ''vec_bolsa_llamamientos_propietario''::name)'
        ),false))
FROM nombres b
LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='vec_bolsa_llamamientos'
LEFT JOIN pg_catalog.pg_class c ON c.relnamespace=n.oid AND c.relname=b.nombre
LEFT JOIN pg_catalog.pg_policy p ON p.polrelid=c.oid;
SQL
}
preimagen() {
  consulta_preimagen | psql_destino
}
consulta_preimagen() {
  cat <<'SQL'
WITH objetivos(esquema,nombre) AS (
 VALUES
 ('vec_bolsa_llamamientos','bolsa_autoritativa'),
 ('vec_bolsa_llamamientos','necesidad_autoritativa'),
 ('vec_bolsa_llamamientos','necesidad_actual'),
 ('vec_bolsa_llamamientos','politica_autoritativa'),
 ('vec_bolsa_llamamientos','instantanea_autoritativa'),
 ('vec_bolsa_llamamientos','evaluacion_autoritativa'),
 ('vec_bolsa_llamamientos','atestacion_autorizacion_version'),
 ('vec_bolsa_llamamientos','atestacion_autorizacion_actual'),
 ('vec_bolsa_llamamientos','propuesta'),
 ('vec_bolsa_llamamientos','referencia_consumida'),
 ('vec_bolsa_llamamientos','uso_decision'),
 ('vec_bolsa_llamamientos','auditoria'),
 ('vec_bolsa_llamamientos','auditoria_actual'),
 ('vec_bolsa_llamamientos','outbox'),
 ('vec_autorizacion','decision_autorizacion_solicitud_ligada_v2')
), catalogo AS (
 SELECT o.esquema,o.nombre,n.oid AS esquema_oid,n.nspowner,
        t.oid AS tipo_oid,t.typtype,t.typowner,t.typacl::text,
        c.oid AS tabla_oid,c.relowner,c.relacl::text,
        c.relrowsecurity,c.relforcerowsecurity,
        (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
            'attnum',x.attnum,'acl',x.attacl::text) ORDER BY x.attnum)
           FROM pg_catalog.pg_attribute x WHERE x.attrelid=c.oid AND x.attnum>0) AS acl_columnas
   FROM objetivos o
   LEFT JOIN pg_catalog.pg_namespace n ON n.nspname=o.esquema
   LEFT JOIN pg_catalog.pg_type t ON t.typnamespace=n.oid AND t.typname=o.nombre
   LEFT JOIN pg_catalog.pg_class c ON c.oid=t.typrelid
)
SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.jsonb_build_object(
    'base',pg_catalog.current_database(),
    'version',pg_catalog.current_setting('server_version_num'),
    'selector',pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector')::text,
    'politicas_b1',(
      SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'tabla',c.relname,'oid',p.oid,'roles',p.polroles,
        'uso',pg_catalog.pg_get_expr(p.polqual,p.polrelid),
        'check',pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)
      ) ORDER BY c.relname,p.polname)
      FROM pg_catalog.pg_policy p
      JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_bolsa_llamamientos'
    ),
    'objetivos',pg_catalog.jsonb_agg(pg_catalog.to_jsonb(catalogo) ORDER BY esquema,nombre)
  )::text,'UTF8')),'hex')
FROM catalogo;
SQL
}

[[ $(rls_b1_cerrada) == t ]] || { echo 'C3 ACL: falta el delta RLS B1 cerrado y revisado' >&2; exit 1; }
actual=$(preimagen) || { echo 'C3 ACL: no se pudo leer la preimagen' >&2; exit 1; }
[[ $actual =~ ^[0-9a-f]{64}$ ]] || { echo 'C3 ACL: preimagen inválida' >&2; exit 1; }
if [[ $modo == inspect ]]; then
  printf '%s\n' "$actual"
  exit 0
fi
[[ $actual == "$esperado" ]] || { echo 'C3 ACL: preimagen distinta de la revisada' >&2; exit 1; }

extraer_cuerpo() {
  python3 - "$raiz/$1" <<'PY'
from pathlib import Path
import re
import sys

lineas = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
inicio = [i for i, linea in enumerate(lineas) if linea == "BEGIN;"]
fin = [i for i, linea in enumerate(lineas) if linea == "COMMIT;"]
if len(inicio) != 1 or len(fin) != 1 or inicio[0] >= fin[0]:
    raise SystemExit("C3 ACL: delimitadores de migración inesperados")
previo = lineas[:inicio[0]]
posterior = lineas[fin[0] + 1:]
if any(linea.strip() and not linea.lstrip().startswith("--")
       and linea != r"\set ON_ERROR_STOP on" for linea in previo):
    raise SystemExit("C3 ACL: prefijo de migración inesperado")
if any(linea.strip() and not linea.lstrip().startswith("--") for linea in posterior):
    raise SystemExit("C3 ACL: sufijo de migración inesperado")
if any(re.fullmatch(r"\s*(?:BEGIN|COMMIT|ROLLBACK)\s*;", linea)
       for linea in lineas[inicio[0] + 1:fin[0]]):
    raise SystemExit("C3 ACL: transacción interna inesperada")
sys.stdout.write("\n".join(lineas[inicio[0] + 1:fin[0]]) + "\n")
PY
}
puerta() {
  python3 - "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/acl_tipos_preselector_c3_v1.sql" <<'PY'
from pathlib import Path
import re
import sys

lineas = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
if not lineas or lineas[0] != r"\set ON_ERROR_STOP on" or any(
        re.fullmatch(r"\s*(?:BEGIN|COMMIT|ROLLBACK)\s*;", linea)
        for linea in lineas[1:]):
    raise SystemExit("C3 ACL: puerta de tipos incompatible")
sys.stdout.write("\n".join(lineas[1:]) + "\n")
PY
}
comprobar_acl_final() {
  cat <<'SQL'
DO $acl_final$
DECLARE total integer;
BEGIN
  WITH objetivos(esquema,nombre) AS (
    VALUES
      ('vec_bolsa_llamamientos','bolsa_autoritativa'),
      ('vec_bolsa_llamamientos','necesidad_autoritativa'),
      ('vec_bolsa_llamamientos','necesidad_actual'),
      ('vec_bolsa_llamamientos','politica_autoritativa'),
      ('vec_bolsa_llamamientos','instantanea_autoritativa'),
      ('vec_bolsa_llamamientos','evaluacion_autoritativa'),
      ('vec_bolsa_llamamientos','atestacion_autorizacion_version'),
      ('vec_bolsa_llamamientos','atestacion_autorizacion_actual'),
      ('vec_bolsa_llamamientos','propuesta'),
      ('vec_bolsa_llamamientos','referencia_consumida'),
      ('vec_bolsa_llamamientos','uso_decision'),
      ('vec_bolsa_llamamientos','auditoria'),
      ('vec_bolsa_llamamientos','auditoria_actual'),
      ('vec_bolsa_llamamientos','outbox'),
      ('vec_autorizacion','decision_autorizacion_solicitud_ligada_v2')
  )
  SELECT count(*) INTO total
    FROM objetivos o JOIN pg_catalog.pg_namespace n ON n.nspname=o.esquema
    JOIN pg_catalog.pg_type t ON t.typnamespace=n.oid AND t.typname=o.nombre
   WHERE t.typtype='c' AND t.typrelid<>0 AND t.typacl IS NOT NULL
     AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a
        WHERE a.grantee<>t.typowner OR a.privilege_type<>'USAGE'
           OR a.grantor<>t.typowner OR a.is_grantable
     )
     AND EXISTS (
       SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a
        WHERE a.grantee=t.typowner AND a.privilege_type='USAGE'
     );
  IF total<>15 THEN
    RAISE EXCEPTION 'C3 ACL: postimagen de tipos no es solo propietario' USING ERRCODE='55000';
  END IF;
END $acl_final$;
SQL
}

# La misma sesión conserva los locks hasta COMMIT. Se extraen solo los cuerpos
# de las dos migraciones versionadas, verificando que no hay COMMIT interno.
{
  cat <<'SQL'
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE vec_c3_preimagen (sha text) ON COMMIT DROP;
CREATE TEMP TABLE vec_c3_rls (cerrada boolean) ON COMMIT DROP;
LOCK TABLE
  vec_bolsa_llamamientos.atestacion_autorizacion_actual,
  vec_bolsa_llamamientos.atestacion_autorizacion_version,
  vec_bolsa_llamamientos.auditoria,
  vec_bolsa_llamamientos.auditoria_actual,
  vec_bolsa_llamamientos.bolsa_autoritativa,
  vec_bolsa_llamamientos.evaluacion_autoritativa,
  vec_bolsa_llamamientos.instantanea_autoritativa,
  vec_bolsa_llamamientos.necesidad_actual,
  vec_bolsa_llamamientos.necesidad_autoritativa,
  vec_bolsa_llamamientos.outbox,
  vec_bolsa_llamamientos.politica_autoritativa,
  vec_bolsa_llamamientos.propuesta,
  vec_bolsa_llamamientos.referencia_consumida,
  vec_bolsa_llamamientos.uso_decision,
  vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
  IN ACCESS EXCLUSIVE MODE;
-- GRANT/REVOKE ON TYPE actualizan pg_type, no bloquean necesariamente la tabla
-- asociada. Esta barrera excluye un grant nominal concurrente hasta COMMIT.
LOCK TABLE pg_catalog.pg_type IN SHARE ROW EXCLUSIVE MODE;
INSERT INTO vec_c3_preimagen(sha)
SQL
  consulta_preimagen
  printf "DO \$preimagen\$ BEGIN IF (SELECT sha FROM vec_c3_preimagen) IS DISTINCT FROM '%s' THEN RAISE EXCEPTION 'C3 ACL: preimagen cambió bajo lock' USING ERRCODE='55000'; END IF; END \$preimagen\$;\n" "$esperado"
  cat <<'SQL'
INSERT INTO vec_c3_rls(cerrada)
SQL
  consulta_rls_b1
  cat <<'SQL'
DO $rls$ BEGIN IF (SELECT cerrada FROM vec_c3_rls) IS DISTINCT FROM true
  THEN RAISE EXCEPTION 'C3 ACL: RLS B1 cambió bajo lock' USING ERRCODE='55000'; END IF; END $rls$;
SQL
  extraer_cuerpo deploy/postgresql/contexto_actor_v1/acl_tipos_preselector_c3_v1.up.sql
  puerta
  comprobar_acl_final
  extraer_cuerpo deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql
  puerta
  comprobar_acl_final
  cat <<'SQL'
DELETE FROM vec_c3_rls;
INSERT INTO vec_c3_rls(cerrada)
SQL
  consulta_rls_b1
  cat <<'SQL'
DO $rls$ BEGIN IF (SELECT cerrada FROM vec_c3_rls) IS DISTINCT FROM true
  THEN RAISE EXCEPTION 'C3 ACL: RLS B1 cambió antes del COMMIT' USING ERRCODE='55000'; END IF; END $rls$;
COMMIT;
SQL
} | psql_destino
echo 'C3 ACL: paquete y selector confirmados en una transacción local; puerta de tipos y RLS verdes'
