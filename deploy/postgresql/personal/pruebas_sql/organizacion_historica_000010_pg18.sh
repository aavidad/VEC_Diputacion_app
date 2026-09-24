#!/usr/bin/env bash
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-b3-${RANDOM}"
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
docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/roles.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000010_organizacion_historica.up.sql" "$container:/tmp/000010.sql"
docker cp "$base_dir/organizacion_historica_000010_stub.sql" "$container:/tmp/stub.sql"
docker cp "$base_dir/organizacion_historica_000010_casos.sql" "$container:/tmp/casos.sql"
docker cp "$base_dir/organizacion_historica_000010_completo.sql" "$container:/tmp/completo.sql"
docker cp "$base_dir/organizacion_historica_000011_casos.sql" "$container:/tmp/importacion_casos.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000011_importacion_organizacion_historica.up.sql" "$container:/tmp/000011.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/roles.sql
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/stub.sql
# Ensayo de reversión sin DOWN: sólo cambia el COMMIT final del archivo copiado.
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000010.sql >/tmp/000010.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000010.rollback.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regclass('vec_personal.org_nodo_historia') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000010.sql
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/casos.sql
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/completo.sql
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000011.sql >/tmp/000011.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000011.rollback.sql
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regclass('vec_personal.importacion_organizacion_revision') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/000011.sql
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/importacion_casos.sql
docker restart "$container" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
  sleep 0.3
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
 fi
 sleep 0.3
done
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_personal.recibo_consulta_organizacion")" = 3
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_personal.importacion_organizacion_recibo")" = 2
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_personal.importacion_organizacion_revision")" = 2
printf 'PG18.4: 000010/000011 ROLLBACK limpio; COMMIT, ACL/RLS, consulta e importación sintéticas y recibos tras reinicio OK (stub AD3, no valida autorización real).\n'
