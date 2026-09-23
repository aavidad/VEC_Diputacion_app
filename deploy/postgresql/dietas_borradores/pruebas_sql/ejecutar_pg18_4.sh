#!/usr/bin/env bash
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container=""
cleanup() { [[ -z "$container" ]] || docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
preparar() {
  container="vec-dietas-r12-$RANDOM"
  docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
echo 'PG18: esperando servicio aislado'
ready=false
for _ in $(seq 1 60); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
    sleep 0.2
    if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
      ready=true
      break
    fi
  fi
  sleep 0.5
done
  if [[ "$ready" != true ]]; then docker logs "$container" >&2 || true; return 1; fi
docker cp "$base_dir/preparar_entorno_stub_ad3.sql" "$container:/tmp/preparar_entorno_stub_ad3.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/roles_up.sql" "$container:/tmp/dietas.roles.up.sql"
docker cp "$repo_dir/deploy/postgresql/personal/roles_up.sql" "$container:/tmp/personal.roles.up.sql"
docker cp "$repo_dir/deploy/postgresql/personal/migraciones/000007_relacion_empleado_dietas.up.sql" "$container:/tmp/personal.000007.up.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql" "$container:/tmp/dietas.up.sql"
docker cp "$base_dir/borrador_comision_durable.sql" "$container:/tmp/borrador_comision_durable.sql"
docker cp "$base_dir/casos_funcionales.sql" "$container:/tmp/casos_funcionales.sql"
docker cp "$base_dir/retirar_entorno_stub_ad3.sql" "$container:/tmp/retirar_entorno_stub_ad3.sql"
docker cp "$repo_dir/deploy/postgresql/dietas_borradores/migraciones/000001_borrador_comision_durable.down.sql" "$container:/tmp/dietas.down.sql"
echo 'PG18: preparando dependencias sintéticas y aplicando Dietas'
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.roles.up.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/personal.roles.up.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/preparar_entorno_stub_ad3.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/personal.000007.up.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.up.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/borrador_comision_durable.sql
}
preparar
echo 'PG18: preparando una base sin historia para la carrera del DOWN'
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -v solo_preparar=true -v confirmar=false -U postgres -d postgres -f /tmp/casos_funcionales.sql
echo 'PG18: comprobando DOWN frente a la única alta pendiente en otra conexión'
docker exec "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SET application_name='vec_dietas_alta_pendiente'; BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SET LOCAL SESSION AUTHORIZATION vec_prueba_dietas; DO \$\$ DECLARE mt text; capacidad bytea; decision bytea; contexto bytea; BEGIN SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_down_pendiente','clave_idempotente_0008'); PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); END \$\$; SELECT pg_sleep(4); COMMIT;" &
alta_pid=$!
alta_pendiente=false
for _ in $(seq 1 50); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name='vec_dietas_alta_pendiente' AND state='active' AND wait_event='PgSleep')" | grep -qx t; then
    alta_pendiente=true
    break
  fi
  sleep 0.1
done
if [[ "$alta_pendiente" != true ]]; then
  wait "$alta_pid" || true
  echo 'ERROR: no se acreditó la alta pendiente antes del DOWN' >&2
  exit 1
fi
docker exec -e PGAPPNAME=vec_dietas_down_pendiente -i "$container" psql -X --set=VERBOSITY=verbose -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.down.sql >/tmp/vec-dietas-r13-down-bloqueado.out 2>&1 &
down_pid=$!
down_en_lock_previo=false
for _ in $(seq 1 40); do
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name='vec_dietas_down_pendiente' AND state='active' AND wait_event_type='Lock' AND query LIKE 'LOCK TABLE vec_dietas.borrador_comision,%')" | grep -qx t; then
    down_en_lock_previo=true
    break
  fi
  sleep 0.05
done
if [[ "$down_en_lock_previo" != true ]]; then
  wait "$down_pid" || true
  wait "$alta_pid" || true
  cat /tmp/vec-dietas-r13-down-bloqueado.out >&2
  echo 'ERROR: el DOWN no esperó en el LOCK TABLE previo a la inspección' >&2
  exit 1
fi
if wait "$down_pid"; then
  echo 'ERROR: DOWN atravesó una alta pendiente' >&2
  exit 1
fi
if ! rg -q 'ERROR:  55P03:' /tmp/vec-dietas-r13-down-bloqueado.out; then
  cat /tmp/vec-dietas-r13-down-bloqueado.out >&2
  echo 'ERROR: DOWN pendiente falló por una causa distinta del lock_timeout' >&2
  exit 1
fi
wait "$alta_pid"
if docker exec -i "$container" psql -X --set=VERBOSITY=verbose -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.down.sql >/tmp/vec-dietas-r13-down-historia.out 2>&1; then
  echo 'ERROR: DOWN aceptó historia confirmada' >&2
  exit 1
fi
if ! rg -q 'ERROR:  55000:' /tmp/vec-dietas-r13-down-historia.out; then
  cat /tmp/vec-dietas-r13-down-historia.out >&2
  echo 'ERROR: DOWN confirmado falló por una causa distinta de la guarda histórica' >&2
  exit 1
fi
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT (SELECT count(*) FROM vec_dietas.borrador_comision)=1 AND (SELECT count(*) FROM vec_dietas.outbox_borrador_comision)=1 AND to_regclass('vec_dietas.borrador_comision') IS NOT NULL" | grep -qx t
cleanup
container=""
preparar
echo 'PG18: ejecutando operaciones funcionales y negativos ligados'
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -v solo_preparar=false -v confirmar=true -U postgres -d postgres -f /tmp/casos_funcionales.sql
if docker exec -i "$container" psql -X --set=VERBOSITY=verbose -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.down.sql >/tmp/vec-dietas-r13-down-funcional.out 2>&1; then
  echo 'ERROR: DOWN aceptó la historia funcional confirmada' >&2
  exit 1
fi
rg -q 'ERROR:  55000:' /tmp/vec-dietas-r13-down-funcional.out
docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT (SELECT count(*) FROM vec_dietas.borrador_comision)=3 AND (SELECT count(*) FROM vec_dietas.outbox_borrador_comision)=3 AND to_regclass('vec_dietas.borrador_comision') IS NOT NULL" | grep -qx t
cleanup
container=""
preparar
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -v solo_preparar=false -v confirmar=false -U postgres -d postgres -f /tmp/casos_funcionales.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/retirar_entorno_stub_ad3.sql
docker exec -i "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres -f /tmp/dietas.down.sql
echo 'OK: Dietas R13 PostgreSQL 18.4 aislado; negativos ligados y DOWN 55P03/55000/vacío; stub AD3 retirado antes del DOWN'
