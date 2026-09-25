#!/usr/bin/env bash
# Ensayo de Calendarios sobre PostgreSQL 18 desechable: UP, retirada vacía,
# ROLLBACK de la carga, COMMIT, casos, retiradas protegidas y reinicio. El
# contenedor se borra siempre al terminar.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig_dir=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
roles_dir=$(CDPATH= cd -- "$base_dir/.." && pwd)
imagen=${VEC_PG_IMAGEN:-postgres:18.4-alpine}
ensayo="calendarios-$RANDOM"
container="vec-calendarios-pg18-$ensayo"
# Datos en memoria y montados: sin volúmenes anónimos; se borran al terminar.
datos="/dev/shm/vec-pg-$ensayo"
mkdir -p "$datos"
salida=$(mktemp -d)
cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  docker run --rm -v "$datos:/borrar" --entrypoint sh "$imagen" -c 'rm -rf /borrar/* /borrar/.[!.]* 2>/dev/null; true' >/dev/null 2>&1 || true
  rmdir "$datos" 2>/dev/null || true
  rm -rf "$salida"
}
trap cleanup EXIT

psql_c() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
aplicar() { psql_c -f "/tmp/$1" >/dev/null; }
esperar() {
  for _ in $(seq 1 80); do
    if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then return 0; fi
    sleep 0.25
  done
  docker logs "$container" >&2 || true
  return 1
}
debe_fallar() { # fichero codigo
  if psql_c --set=VERBOSITY=verbose -f "/tmp/$1" >"$salida/fallo.out" 2>&1; then
    echo "ERROR: $1 debía fallar" >&2; exit 1
  fi
  if ! grep -q "ERROR:  $2:" "$salida/fallo.out"; then
    cat "$salida/fallo.out" >&2; echo "ERROR: $1 falló por otra causa" >&2; exit 1
  fi
}
huella() {
  psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT md5(string_agg(v::text, '|' ORDER BY v.id)) FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['nacional','autonomico','local','local','local','centro','centro','centro','centro'], ARRAY['es','es-an','municipio:ine:18087','municipio:ine:18098','municipio:sintetico:a','centro-530','centro-520','centro-102','centro-752'], now()) v"
}

docker run -d --rm --name "$container" -p 127.0.0.1::5432 -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
for f in "$roles_dir"/roles_up.sql "$roles_dir"/roles_down.sql "$mig_dir"/*.sql "$base_dir"/casos.sql; do
  docker cp "$f" "$container:/tmp/$(basename "$f")"
done
sed 's/^COMMIT;$/ROLLBACK;/' "$mig_dir/000002_calendarios_2026.up.sql" >"$salida/000002_rollback.sql"
docker cp "$salida/000002_rollback.sql" "$container:/tmp/000002_rollback.sql"
sed 's/^COMMIT;$/ROLLBACK;/' "$mig_dir/000003_centros_sin_truncar.up.sql" >"$salida/000003_rollback.sql"
docker cp "$salida/000003_rollback.sql" "$container:/tmp/000003_rollback.sql"
sed 's/^COMMIT;$/ROLLBACK;/' "$mig_dir/000004_calendario_ejemplo_2026.up.sql" >"$salida/000004_rollback.sql"
docker cp "$salida/000004_rollback.sql" "$container:/tmp/000004_rollback.sql"

echo 'PG18: roles y estructura; retirada de una historia vacía y reinstalación'
aplicar roles_up.sql
aplicar 000001_historia_calendarios.up.sql
aplicar 000001_historia_calendarios.down.sql
psql_c -At -c "SELECT to_regclass('vec_calendarios.version_calendario') IS NULL" | grep -qx t
aplicar 000001_historia_calendarios.up.sql

echo 'PG18: la carga en ROLLBACK no deja rastro'
aplicar 000002_rollback.sql
psql_c -At -c "SELECT count(*) FROM vec_calendarios.version_calendario" | grep -qx 0

echo 'PG18: carga 2026 confirmada y casos'
aplicar 000002_calendarios_2026.up.sql
psql_c -At -c "SELECT count(*)||'/'||(SELECT count(*) FROM vec_calendarios.dia_calendario) FROM vec_calendarios.version_calendario" | grep -qx '8/22'
psql_c -f /tmp/casos.sql | grep -q 'OK casos Calendarios'

echo 'PG18: 000003 sustituye la lista de centros sin truncar'
aplicar 000003_rollback.sql
psql_c -At -c "SELECT position('LIMIT 1000' in pg_get_functiondef('vec_calendarios.centros_con_calendario_v1(integer,timestamptz)'::regprocedure))>0" | grep -qx t
aplicar 000003_centros_sin_truncar.up.sql
psql_c -At -c "SELECT position('54000' in pg_get_functiondef('vec_calendarios.centros_con_calendario_v1(integer,timestamptz)'::regprocedure))>0" | grep -qx t
psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT count(*) FROM vec_calendarios.centros_con_calendario_v1(2026, now())" | grep -qx 4
debe_fallar 000003_centros_sin_truncar.up.sql 55000
debe_fallar 000003_centros_sin_truncar.down.sql 55000

echo 'PG18: 000004 calendario de ejemplo; retirada y reaplicación sin borrar historia'
ultima() { # tipo ref: id y días de la última versión, leídos por el lector
  psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT v.id||'|'||coalesce(v.municipio_ref,'-')||'|'||coalesce((SELECT string_agg(d->>'fecha', ',' ORDER BY d->>'fecha') FROM jsonb_array_elements(v.dias) d),'') FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['$1'], ARRAY['$2'], now()) v"
}
debe_fallar 000004_calendario_ejemplo_2026.down.sql 55000
aplicar 000004_rollback.sql
psql_c -At -c "SELECT count(*) FROM vec_calendarios.version_calendario" | grep -qx 9
if docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U vec_prueba_calendarios_lector -d postgres -f /tmp/000004_calendario_ejemplo_2026.up.sql >"$salida/lector.out" 2>&1; then
  echo "ERROR: el lector no puede aplicar 000004" >&2; exit 1
fi
grep -q 'permission denied' "$salida/lector.out"
aplicar 000004_calendario_ejemplo_2026.up.sql
ultima local municipio:ine:18087 | grep -qx 'calendario:local:municipio:ine:18087:2026:v3|-|2026-01-02,2026-06-04'
ultima local municipio:ine:18098 | grep -qx 'calendario:local:municipio:ine:18098:2026:v1|-|2026-05-25,2026-10-22'
ultima centro centro-752 | grep -qx 'calendario:centro:centro-752:2026:v2|municipio:ine:18098|2026-12-24,2026-12-31'
psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT count(*) FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['local','local','centro'], ARRAY['municipio:ine:18087','municipio:ine:18098','centro-752'], now()) WHERE sintetica AND procedencia_referencia LIKE 'paquete:ejemplo:vec:v1%'" | grep -qx 3
debe_fallar 000004_calendario_ejemplo_2026.up.sql 55000
aplicar 000004_calendario_ejemplo_2026.down.sql
ultima local municipio:ine:18087 | grep -qx 'calendario:local:municipio:ine:18087:2026:v4|-|2026-03-16,2026-06-15'
ultima local municipio:ine:18098 | grep -qx 'calendario:local:municipio:ine:18098:2026:v2|-|'
ultima centro centro-752 | grep -qx 'calendario:centro:centro-752:2026:v3|municipio:sintetico:a|'
debe_fallar 000004_calendario_ejemplo_2026.down.sql 55000
aplicar 000004_calendario_ejemplo_2026.up.sql
ultima local municipio:ine:18087 | grep -qx 'calendario:local:municipio:ine:18087:2026:v5|-|2026-01-02,2026-06-04'
ultima centro centro-752 | grep -qx 'calendario:centro:centro-752:2026:v4|municipio:ine:18098|2026-12-24,2026-12-31'
psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT count(*) FROM vec_calendarios.centros_con_calendario_v1(2026, now())" | grep -qx 4

echo 'PG18: retiradas protegidas con historia'
debe_fallar 000002_calendarios_2026.down.sql 55000
debe_fallar 000001_historia_calendarios.down.sql 55000
debe_fallar roles_down.sql 55000
psql_c -At -c "SELECT count(*) FROM vec_calendarios.version_calendario" | grep -qx 18

echo 'PG18: reinicio y misma lectura'
antes=$(huella)
docker restart "$container" >/dev/null
esperar
despues=$(huella)
if [[ -z "$antes" || "$antes" != "$despues" ]]; then
  echo "ERROR: la lectura cambia tras reiniciar ($antes / $despues)" >&2; exit 1
fi
if [[ "${VEC_CALENDARIOS_PRUEBA_GO:-}" == 1 ]]; then
  echo 'PG18: adaptador Go con el login lector tras el reinicio'
  puerto=$(docker port "$container" 5432/tcp | head -n1 | sed 's/.*://')
  repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
  (cd "$repo_dir" && VEC_CALENDARIOS_TEST_PG_URL="postgres://vec_prueba_calendarios_lector@127.0.0.1:$puerto/postgres?sslmode=disable" \
    go test -count=1 -run TestRepositorioPostgreSQLReal -v ./internal/modules/calendarios/adapters/postgres/ | grep -E '^(--- |ok|FAIL)')
fi
echo "OK: Calendarios en PostgreSQL ${imagen#postgres:} desechable; huella $antes estable tras reinicio"
