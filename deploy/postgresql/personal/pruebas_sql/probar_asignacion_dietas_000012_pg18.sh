#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-d7-000012-$RANDOM"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
psql_file() { docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f "$1"; }

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
ready=false
for _ in $(seq 1 60); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1 \
    && { sleep 0.2; docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; }; then
    ready=true
    break
  fi
  sleep 0.5
done
if [[ "$ready" != true ]]; then docker logs "$container" >&2 || true; exit 1; fi

docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/personal_roles_up.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql" "$container:/tmp/dietas_roles_up.sql"
for source in \
  "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000009_asignacion_dietas.up.sql" \
  "$base_dir/preparar_asignacion_dietas_000012.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000012_asignacion_dietas.up.sql" \
  "$repo_dir/deploy/postgresql/personal/migraciones/000012_asignacion_dietas.down.sql" \
  "$base_dir/asignacion_dietas_000012.sql"; do
  docker cp "$source" "$container:/tmp/$(basename "$source")"
done

echo 'PG18 aislado: roles, 000007 y 000009'
psql_file /tmp/personal_roles_up.sql
psql_file /tmp/dietas_roles_up.sql
psql_file /tmp/000007_relacion_empleado_dietas.up.sql
psql_file /tmp/000009_asignacion_dietas.up.sql
echo 'PG18 aislado: preimagen TEST-ONLY con fachadas AD3 nominales TEST-ONLY'
psql_file /tmp/preparar_asignacion_dietas_000012.sql
psql_file /tmp/000012_asignacion_dietas.up.sql
echo 'PG18 aislado: DOWN vacío de 000012 conserva 000009 y retira sólo 000012'
psql_file /tmp/000012_asignacion_dietas.down.sql
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regclass('vec_personal.asignacion_dietas') IS NOT NULL AND to_regclass('vec_personal.recibo_asignacion_dietas') IS NULL AND to_regclass('vec_personal.evidencia_asignacion_dietas') IS NULL AND to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NULL AND to_regrole('vec_personal_d7_ejecutor') IS NULL" | grep -qx t
psql_file /tmp/000012_asignacion_dietas.up.sql
psql_file /tmp/asignacion_dietas_000012.sql
echo 'PG18 aislado: DOWN con alta confirmada debe rechazar y preservar historia'
if psql_file /tmp/000012_asignacion_dietas.down.sql >/tmp/vec-personal-d7-000012-down-historia.out 2>&1; then
  echo 'ERROR: DOWN 000012 aceptó una alta confirmada' >&2
  exit 1
fi
rg -q 'Personal 000012 DOWN protege historia o consumidor' /tmp/vec-personal-d7-000012-down-historia.out
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT (SELECT count(*) FROM vec_personal.asignacion_dietas)=3 AND (SELECT count(*) FROM vec_personal.recibo_asignacion_dietas)=4 AND (SELECT count(*) FROM vec_personal.evidencia_asignacion_dietas)=4 AND to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NOT NULL AND to_regrole('vec_personal_d7_ejecutor') IS NOT NULL" | grep -qx t
echo 'OK: Personal D7 000012 PostgreSQL 18 aislado; ACL nominal, consulta, correcciones separadas, replay/conflicto, historia y sello.'
