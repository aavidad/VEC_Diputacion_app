#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-d7-auditoria-$$-$RANDOM"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
psql_file() {
  local usuario=$1 archivo=$2
  docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U "$usuario" -d postgres -f "$archivo"
}
psql_dba() { docker exec -i "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }

docker run -d --rm --network none --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -q -h 127.0.0.1 -U postgres -d postgres; then break; fi
  sleep 0.5
done
psql_dba -c 'SELECT 1' >/dev/null

for source in \
  "$repo_dir/deploy/postgresql/personal/roles_up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000012_auditoria_frontera_asignacion_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000012_auditoria_frontera_asignacion_dietas.down.sql" \
  "$base_dir/auditoria_frontera_asignacion_dietas_000012.sql" \
  "$base_dir/auditoria_frontera_asignacion_dietas_000012_login.sql" \
  "$base_dir/auditoria_frontera_asignacion_dietas_000012_extra.sql"; do
  docker cp "$source" "$container:/tmp/$(basename "$source")"
done

psql_file postgres /tmp/roles_up.sql
if psql_file postgres /tmp/000012_auditoria_frontera_asignacion_dietas.up.sql >/dev/null 2>&1; then
  echo 'Personal 000012 aceptó ausencia de 000011' >&2; exit 1
fi
[[ "$(psql_dba -c "SELECT NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_registrador_frontera')")" == t ]]
psql_dba <<'SQL' >/dev/null
SET ROLE vec_personal_propietario;
-- Fachada 000011 mínima para este ensayo focal de la migración 000012.
CREATE FUNCTION vec_personal.consultar_asignacion_dietas_v1(
  text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS boolean LANGUAGE sql AS 'SELECT true';
SQL
docker exec "$container" sed 's/^COMMIT;$/ROLLBACK;/' \
  /tmp/000012_auditoria_frontera_asignacion_dietas.up.sql \
  | psql_dba >/dev/null
[[ "$(psql_dba -c "SELECT to_regclass('vec_personal.auditoria_frontera_asignacion_dietas') IS NULL AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_registrador_frontera')")" == t ]]

psql_file postgres /tmp/000012_auditoria_frontera_asignacion_dietas.up.sql
psql_file postgres /tmp/000012_auditoria_frontera_asignacion_dietas.down.sql
[[ "$(psql_dba -c "SELECT to_regclass('vec_personal.auditoria_frontera_asignacion_dietas') IS NULL AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_registrador_frontera')")" == t ]]
psql_file postgres /tmp/000012_auditoria_frontera_asignacion_dietas.up.sql

psql_file postgres /tmp/auditoria_frontera_asignacion_dietas_000012.sql
psql_dba <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
DO $prueba$
BEGIN
  BEGIN
    INSERT INTO vec_personal.auditoria_frontera_asignacion_dietas (
      correlacion_ref,motivo,superficie,ruta,accion,actor_ref,
      recurso_ref,estado_http,registrada_en)
    VALUES ('corr_no_disponible','acceso_denegado',
      'api.personal.asignaciones_dietas',
      '/api/vec/personal/asignaciones-dietas/detalle','consultar',NULL,
      'rel_AbCdef0123456789_-QRST',403,clock_timestamp());
    RAISE EXCEPTION 'propietario sin registrador aceptado';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END
$prueba$;
ROLLBACK;
SQL
psql_file vec_personal_auditoria_prueba /tmp/auditoria_frontera_asignacion_dietas_000012_login.sql
psql_file vec_personal_auditoria_extra /tmp/auditoria_frontera_asignacion_dietas_000012_extra.sql
psql_dba <<'SQL' >/dev/null
DO $prueba$
BEGIN
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible', 'acceso_denegado', 'api.personal.asignaciones_dietas',
      '/api/vec/personal/asignaciones-dietas/detalle', 'consultar', NULL,
      'rel_AbCdef0123456789_-QRST', 403::smallint);
    RAISE EXCEPTION 'DBA aceptado como registrador';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END
$prueba$;
SQL
[[ "$(psql_dba -c "SELECT count(*) FROM vec_personal.auditoria_frontera_asignacion_dietas")" == 12 ]]
[[ "$(psql_dba -c "SELECT string_agg(estado_http::text,',' ORDER BY estado_http) FROM vec_personal.auditoria_frontera_asignacion_dietas")" == '400,401,403,403,404,404,405,405,406,409,503,503' ]]
[[ "$(psql_dba -c "SELECT count(*) FROM vec_personal.auditoria_frontera_asignacion_dietas WHERE actor_ref='per_sintetico' AND recurso_ref IS NULL AND estado_http IN (404,405,503)")" == 3 ]]

psql_dba <<'SQL' >/dev/null
DO $prueba$
BEGIN
  BEGIN
    UPDATE vec_personal.auditoria_frontera_asignacion_dietas SET motivo='acceso_denegado';
    RAISE EXCEPTION 'se modifico historia';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
  BEGIN
    TRUNCATE vec_personal.auditoria_frontera_asignacion_dietas;
    RAISE EXCEPTION 'se trunco historia';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END
$prueba$;
SQL
psql_dba -c 'REVOKE vec_personal_registrador_frontera FROM vec_personal_auditoria_prueba,vec_personal_auditoria_extra' >/dev/null
if psql_file postgres /tmp/000012_auditoria_frontera_asignacion_dietas.down.sql >/dev/null 2>&1; then
  echo 'Personal 000012 DOWN eliminó historia' >&2; exit 1
fi
[[ "$(psql_dba -c "SELECT count(*) FROM vec_personal.auditoria_frontera_asignacion_dietas")" == 12 ]]
echo 'OK: Personal 000012, PG18 desechable; rollback/UP/DOWN vacío, rol y ACL, 12 eventos HTTP, recurso opcional preselector/metodo, guardas, inmutabilidad y DOWN con historia.'
