#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-personal-d7c-000014-$RANDOM"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
run_sql() { docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f "$1"; }

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
ready=false
for _ in $(seq 1 60); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then ready=true; break; fi
  sleep 0.5
done
if [[ "$ready" != true ]]; then docker logs "$container" >&2 || true; exit 1; fi

for source in \
 "$repo_dir/deploy/postgresql/personal/roles_up.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.up.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000009_asignacion_dietas.up.sql" \
 "$base_dir/preparar_asignacion_dietas_000011.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000011_asignacion_dietas.up.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000012_auditoria_frontera_asignacion_dietas.up.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000013_competencias_asignacion_dietas.up.sql" \
 "$base_dir/rectificacion_dietas_000014_preparar.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000014_solicitud_rectificacion_dietas.up.sql" \
 "$repo_dir/deploy/postgresql/personal/migraciones/000014_solicitud_rectificacion_dietas.down.sql" \
 "$base_dir/rectificacion_dietas_000014.sql" \
 "$base_dir/rectificacion_dietas_000014_operaciones.sql"; do
  docker cp "$source" "$container:/tmp/$(basename "$source")"
done

run_sql /tmp/roles_up.sql
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql" "$container:/tmp/dietas_roles_up.sql"
run_sql /tmp/dietas_roles_up.sql
run_sql /tmp/000007_relacion_empleado_dietas.up.sql
run_sql /tmp/000009_asignacion_dietas.up.sql
run_sql /tmp/preparar_asignacion_dietas_000011.sql
run_sql /tmp/000011_asignacion_dietas.up.sql
run_sql /tmp/000012_auditoria_frontera_asignacion_dietas.up.sql
run_sql /tmp/rectificacion_dietas_000014_preparar.sql
run_sql /tmp/000013_competencias_asignacion_dietas.up.sql
run_sql /tmp/000014_solicitud_rectificacion_dietas.up.sql
run_sql /tmp/rectificacion_dietas_000014.sql
run_sql /tmp/rectificacion_dietas_000014_operaciones.sql
if down_output=$(run_sql /tmp/000014_solicitud_rectificacion_dietas.down.sql 2>&1); then
  echo 'ERROR: DOWN 000014 borró historia sintética' >&2
  exit 1
fi
if [[ "$down_output" != *'Personal 000014 DOWN protege historia'* ]]; then
  echo "$down_output" >&2
  exit 1
fi
echo 'OK: Personal 000014 PG18 aislado; estructura, ACL, solicitud/consulta/rechazo/confirmación/replay con stub AD3.'
