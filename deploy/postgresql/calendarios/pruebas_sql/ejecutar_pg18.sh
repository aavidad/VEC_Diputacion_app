#!/usr/bin/env bash
# Ensayo de Calendarios sobre PostgreSQL 18 desechable: UP, retirada vacía,
# ROLLBACK de la carga, COMMIT, casos, retiradas protegidas y reinicio. El
# contenedor se borra siempre al terminar.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig_dir=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
roles_dir=$(CDPATH= cd -- "$base_dir/.." && pwd)
imagen=${VEC_PG_IMAGEN:-postgres:18.4-alpine}
container="vec-calendarios-pg18-$RANDOM"
salida=$(mktemp -d)
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf "$salida"; }
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
  psql_c -At -c "SET ROLE vec_prueba_calendarios_lector; SELECT md5(string_agg(v::text, '|' ORDER BY v.id)) FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['nacional','autonomico','local','local','centro','centro','centro','centro'], ARRAY['es','es-an','municipio:ine:18087','municipio:sintetico:a','centro-530','centro-520','centro-102','centro-752'], now()) v"
}

docker run -d --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
for f in "$roles_dir"/roles_up.sql "$roles_dir"/roles_down.sql "$mig_dir"/*.sql "$base_dir"/casos.sql; do
  docker cp "$f" "$container:/tmp/$(basename "$f")"
done
sed 's/^COMMIT;$/ROLLBACK;/' "$mig_dir/000002_calendarios_2026.up.sql" >"$salida/000002_rollback.sql"
docker cp "$salida/000002_rollback.sql" "$container:/tmp/000002_rollback.sql"

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

echo 'PG18: retiradas protegidas con historia'
debe_fallar 000002_calendarios_2026.down.sql 55000
debe_fallar 000001_historia_calendarios.down.sql 55000
debe_fallar roles_down.sql 55000
psql_c -At -c "SELECT count(*) FROM vec_calendarios.version_calendario" | grep -qx 9

echo 'PG18: reinicio y misma lectura'
antes=$(huella)
docker restart "$container" >/dev/null
esperar
despues=$(huella)
if [[ -z "$antes" || "$antes" != "$despues" ]]; then
  echo "ERROR: la lectura cambia tras reiniciar ($antes / $despues)" >&2; exit 1
fi
echo "OK: Calendarios en PostgreSQL ${imagen#postgres:} desechable; huella $antes estable tras reinicio"
