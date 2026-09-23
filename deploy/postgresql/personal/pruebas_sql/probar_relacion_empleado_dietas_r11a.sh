#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-r11a-$RANDOM"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT

run_sql() {
  local db=$1
  docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d "$db"
}
run_file() {
  local db=$1 file=$2
  docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d "$db" -f "$file"
}
must_fail_file() {
  local db=$1 file=$2
  if run_file "$db" "$file" >/tmp/vec-personal-r11a-down.out 2>&1; then
    echo "ERROR: la retirada con historia confirmada no fue rechazada" >&2
    return 1
  fi
  rg -q 'protege relación/historia|consumidor Dietas' /tmp/vec-personal-r11a-down.out
}
must_fail_sql() {
  local db=$1 sql=$2
  if printf '%s\n' "$sql" | run_sql "$db" >/tmp/vec-personal-r11a-denegado.out 2>&1; then
    echo "ERROR: una llamada que debía denegarse terminó correctamente" >&2
    return 1
  fi
}

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null

for source in \
  "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.down.sql"; do
  docker cp "$source" "$container:/tmp/$(basename "$source")"
done
docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/personal_roles_up.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql" "$container:/tmp/dietas_roles_up.sql"

echo 'PG18: preparando roles nominales y aplicando Personal 000007'
run_file postgres /tmp/personal_roles_up.sql
run_file postgres /tmp/dietas_roles_up.sql
run_file postgres /tmp/000007_relacion_empleado_dietas.up.sql

echo 'PG18: comprobando ACL efectiva y fronteras de llamada'
run_sql postgres <<'SQL'
DO $acl$
DECLARE resolver regprocedure:='vec_personal.resolver_relacion_dietas_v1(text,text,text,date)'::regprocedure;
        revalida regprocedure:='vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)'::regprocedure;
BEGIN
 IF has_schema_privilege('vec_dietas_ejecutor','vec_personal','USAGE')
    OR NOT has_schema_privilege('vec_dietas_propietario','vec_personal','USAGE')
    OR has_table_privilege('vec_dietas_ejecutor','vec_personal.relacion_empleado_dietas','SELECT')
    OR has_table_privilege('vec_dietas_propietario','vec_personal.relacion_empleado_dietas','SELECT')
    OR has_function_privilege('vec_dietas_ejecutor',resolver,'EXECUTE')
    OR has_function_privilege('vec_dietas_propietario',resolver,'EXECUTE')
    OR has_function_privilege('vec_dietas_ejecutor',revalida,'EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario',revalida,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid IN (resolver,revalida) AND a.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
       WHERE n.oid='vec_personal'::regnamespace AND a.grantee=0)
 THEN RAISE EXCEPTION 'ACL Personal/Dietas R11A incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
BEGIN;
SET ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','emp_abcdefghijklmnopqrstuv','unidad-sintetica','activa',DATE '2026-01-01',NULL,1,'acto-sintetico','fuente-sintetica',1);
RESET ROLE;
COMMIT;
CREATE ROLE vec_personal_r11a_login NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_personal_r11a_login;
CREATE FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(
 p_relacion text,p_persona text,p_empleado text,p_unidad text,p_desde text,p_hasta text,
 p_version bigint,p_acto text,p_fuente text,p_fuente_version bigint,p_fecha date
) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $$
BEGIN
 RETURN vec_personal.revalidar_relacion_dietas_v1(p_relacion,p_persona,p_empleado,p_unidad,p_desde,p_hasta,p_version,p_acto,p_fuente,p_fuente_version,p_fecha);
END $$;
ALTER FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date) TO vec_dietas_ejecutor;
SQL

must_fail_sql postgres "SET SESSION AUTHORIZATION vec_personal_r11a_login; SET ROLE vec_dietas_ejecutor; SELECT vec_personal.resolver_relacion_dietas_v1('per_abcdefghijklmnopqrstuv','emp_abcdefghijklmnopqrstuv','rel_abcdefghijklmnopqrstuv',DATE '2026-02-01');"
must_fail_sql postgres "SET SESSION AUTHORIZATION vec_personal_r11a_login; SET ROLE vec_dietas_ejecutor; SELECT vec_personal.revalidar_relacion_dietas_v1('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','emp_abcdefghijklmnopqrstuv','unidad-sintetica','2026-01-01','',1,'acto-sintetico','fuente-sintetica',1,DATE '2026-02-01');"
run_sql postgres <<'SQL'
SET SESSION AUTHORIZATION vec_personal_r11a_login;
SET ROLE vec_dietas_ejecutor;
SELECT vec_dietas.r11a_revalidar_relacion_anidada('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','emp_abcdefghijklmnopqrstuv','unidad-sintetica','2026-01-01','',1,'acto-sintetico','fuente-sintetica',1,DATE '2026-02-01');
SQL
run_sql postgres <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date) FROM vec_dietas_ejecutor;
DROP FUNCTION vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date) RESTRICT;
DO $$ BEGIN
 IF to_regprocedure('vec_dietas.r11a_revalidar_relacion_anidada(text,text,text,text,text,text,bigint,text,text,bigint,date)') IS NOT NULL THEN
   RAISE EXCEPTION 'wrapper R11A residual' USING ERRCODE='55000';
 END IF;
END $$;
SQL

echo 'PG18: DOWN desde sesión fresca con historia confirmada debe rechazar y preservar objetos'
must_fail_file postgres /tmp/000007_relacion_empleado_dietas.down.sql
run_sql postgres <<'SQL'
DO $$ BEGIN
 IF to_regclass('vec_personal.relacion_empleado_dietas') IS NULL
    OR to_regprocedure('vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)') IS NULL
    OR (SELECT count(*) FROM vec_personal.relacion_empleado_dietas)<>1 THEN
   RAISE EXCEPTION 'DOWN rechazado no preservó historia u objetos' USING ERRCODE='55000';
 END IF;
END $$;
SQL

echo 'PG18: DOWN en base vacía debe retirar únicamente la fachada'
docker exec "$container" createdb -U postgres personal_r11a_vacia
run_sql personal_r11a_vacia <<'SQL'
CREATE SCHEMA vec_personal AUTHORIZATION vec_personal_propietario;
REVOKE ALL ON SCHEMA vec_personal FROM PUBLIC;
SQL
run_file personal_r11a_vacia /tmp/000007_relacion_empleado_dietas.up.sql
run_file personal_r11a_vacia /tmp/000007_relacion_empleado_dietas.down.sql
run_sql personal_r11a_vacia <<'SQL'
DO $$ BEGIN
 IF to_regclass('vec_personal.relacion_empleado_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.resolver_relacion_dietas_v1(text,text,text,date)') IS NOT NULL
    OR to_regprocedure('vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)') IS NOT NULL THEN
   RAISE EXCEPTION 'DOWN vacío dejó objetos de 000007' USING ERRCODE='55000';
 END IF;
END $$;
SQL

echo 'OK: Personal 000007 R11A PostgreSQL 18.4 aislado; ACL y DOWN FORCE RLS verificados'
