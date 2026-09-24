#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-d7b-000013-$RANDOM"
fixture_pre=$(mktemp)
fixture_post=$(mktemp)
cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -f "$fixture_pre" "$fixture_post"
}
trap cleanup EXIT
psql_file() { docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f "$1"; }

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
ready=false
for _ in $(seq 1 60); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 0.5
done
if [[ "$ready" != true ]]; then docker logs "$container" >&2 || true; exit 1; fi

docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/personal_roles_up.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql" "$container:/tmp/dietas_roles_up.sql"
sed '/^CREATE SCHEMA vec_prueba_d7;/,$d' "$base_dir/preparar_asignacion_dietas_000011.sql" > "$fixture_pre"
sed -n '/^CREATE SCHEMA vec_prueba_d7;/,$p' "$base_dir/preparar_asignacion_dietas_000011.sql" > "$fixture_post"
docker cp "$fixture_pre" "$container:/tmp/preparar_000011_pre.sql"
docker cp "$fixture_post" "$container:/tmp/preparar_000011_post.sql"
for source in \
  "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000009_asignacion_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000011_asignacion_dietas.up.sql" \
  "$base_dir/preparar_competencias_asignacion_dietas_000013.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000013_competencias_asignacion_dietas.up.sql" \
  "$base_dir/competencias_asignacion_dietas_000013.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000013_competencias_asignacion_dietas.down.sql"; do
  docker cp "$source" "$container:/tmp/$(basename "$source")"
done

psql_file /tmp/personal_roles_up.sql
psql_file /tmp/dietas_roles_up.sql
psql_file /tmp/000007_relacion_empleado_dietas.up.sql
psql_file /tmp/000009_asignacion_dietas.up.sql
psql_file /tmp/preparar_000011_pre.sql
psql_file /tmp/000011_asignacion_dietas.up.sql
psql_file /tmp/preparar_000011_post.sql
psql_file /tmp/preparar_competencias_asignacion_dietas_000013.sql
psql_file /tmp/000013_competencias_asignacion_dietas.up.sql
psql_file /tmp/000013_competencias_asignacion_dietas.down.sql
psql_file /tmp/000013_competencias_asignacion_dietas.up.sql
psql_file /tmp/competencias_asignacion_dietas_000013.sql

# DOWN con historia debe fallar; PostgreSQL deja intacta la migración.
if psql_file /tmp/000013_competencias_asignacion_dietas.down.sql >/dev/null 2>&1; then
  echo 'ERROR: DOWN de 000013 destruyó historia' >&2
  exit 1
fi
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres \
  -c "SELECT CASE WHEN to_regprocedure('vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND (SELECT count(*) FROM vec_personal.recibo_competencias_asignacion_dietas)=3 THEN 'ok' ELSE 'no' END" | grep -qx ok
echo 'OK: Personal 000013 PG18 efímero, AD3 stub TEST-ONLY, competencia actual/obsoleta, ACL, recibos y DOWN protegido.'
