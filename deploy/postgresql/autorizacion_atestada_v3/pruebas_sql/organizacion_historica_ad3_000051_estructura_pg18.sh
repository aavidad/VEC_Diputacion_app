#!/usr/bin/env bash
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-ad3-b3-${RANDOM}"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
  sleep 0.3
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
 fi
 sleep 0.3
done
docker cp "$base_dir/organizacion_historica_ad3_000051_stub.sql" "$container:/tmp/stub.sql"
docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000051_consumidor_organizacion_historica.up.sql" "$container:/tmp/000051.sql"
docker cp "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000052_consumidor_importacion_organizacion.up.sql" "$container:/tmp/000052.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/stub.sql
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000051.sql >/tmp/000051.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000051.rollback.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000051.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" = t
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000052.sql >/tmp/000052.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000052.rollback.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000052.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" = t
printf 'PG18.4: AD3-51/52 ROLLBACK/COMMIT y ACL sobre PREIMAGEN SINTÉTICA compatibles; NO acredita cadena AD3 real.\n'
