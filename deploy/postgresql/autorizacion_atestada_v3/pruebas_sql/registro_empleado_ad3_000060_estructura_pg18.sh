#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
sql_059="${AD3_59_SQL:-$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql}"
if [[ ! -f "$sql_059" ]]; then
  printf 'Falta AD3-59: integrar primero la rama Dietas o indicar AD3_59_SQL.\n' >&2
  exit 2
fi
container="vec-b2-ad3-60-$BASHPID"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in {1..40}; do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done

docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/organizacion_historica_ad3_000051_stub.sql" "$container:/tmp/stub.sql"
docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000051_consumidor_organizacion_historica.up.sql" "$container:/tmp/051.sql"
docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consumidor_importacion_organizacion.up.sql" "$container:/tmp/052.sql"
docker cp "$sql_059" "$container:/tmp/059.sql"
docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000060_consumidor_registro_empleado_b2.up.sql" "$container:/tmp/060.sql"

for fichero in stub 051 052; do
  docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f "/tmp/$fichero.sql" >/dev/null
done
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -c \
  'CREATE ROLE vec_dietas_propietario NOLOGIN; CREATE ROLE vec_dietas_ejecutor NOLOGIN NOBYPASSRLS;' >/dev/null
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/059.sql >/dev/null
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/060.sql >/tmp/060.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/060.rollback.sql >/dev/null
firma='vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regprocedure('$firma') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/060.sql >/dev/null
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT has_function_privilege('vec_personal_propietario','$firma','EXECUTE'),has_function_privilege('vec_personal_ejecutor','$firma','EXECUTE')")" = 't|f'
docker restart "$container" >/dev/null
for _ in {1..40}; do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regprocedure('$firma') IS NOT NULL")" = t
printf 'AD3-60 PG18: ROLLBACK, COMMIT, ACL, reinicio sobre preimagen sintética 051/052 + SQL real 059; NO acredita cadena AD3 íntegra.\n'
