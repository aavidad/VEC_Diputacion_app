#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
container="vec-b2-personal-17-$BASHPID"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in {1..40}; do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done

docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/roles.sql"
docker cp "$repo_dir/deploy/postgresql/personal/pruebas_sql/organizacion_historica_000010_stub.sql" "$container:/tmp/stub.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000010_organizacion_historica.up.sql" "$container:/tmp/010.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000016_proyeccion_empleado_persona.up.sql" "$container:/tmp/016.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000017_registro_empleado_bitemporal.up.sql" "$container:/tmp/017.sql"

for fichero in roles stub 010 016; do
  docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f "/tmp/$fichero.sql" >/dev/null
done
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/017.sql >/tmp/017.rollback.sql"
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/017.rollback.sql >/dev/null
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regclass('vec_personal.relacion_servicio_historia') IS NULL")" = t
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/017.sql >/dev/null

test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*)=5 FROM pg_class WHERE oid IN ('vec_personal.relacion_servicio_historia'::regclass,'vec_personal.ocupacion_empleado_historia'::regclass,'vec_personal.servicio_reconocido_historia'::regclass,'vec_personal.situacion_empleado_historia'::regclass,'vec_personal.cobertura_ocupaciones_historia'::regclass) AND relrowsecurity AND relforcerowsecurity")" = t
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT bool_and(NOT has_table_privilege('vec_personal_ejecutor',format('vec_personal.%I',t),'SELECT')) FROM unnest(ARRAY['relacion_servicio_historia','ocupacion_empleado_historia','servicio_reconocido_historia','situacion_empleado_historia','cobertura_ocupaciones_historia']) t")" = t

docker restart "$container" >/dev/null
for _ in {1..40}; do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done
test "$(docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regclass('vec_personal.cobertura_ocupaciones_historia') IS NOT NULL")" = t
printf 'Personal 000017 PG18: ROLLBACK, COMMIT, RLS/ACL, reinicio correctos (sin consumo V3)\n'
