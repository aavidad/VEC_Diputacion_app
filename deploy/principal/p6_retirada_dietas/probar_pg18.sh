#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${VEC_P6_TEST_BASE:?directorio de pruebas privado requerido}"
base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"
test_base="$(realpath -e "$VEC_P6_TEST_BASE")"
case "$test_base/" in "$repo_dir/"*) echo 'base de prueba dentro de Git' >&2; exit 2;; esac
[[ "$(stat -c '%a' "$test_base")" == 700 ]] || { echo 'base debe ser 0700' >&2; exit 2; }
test_dir="$(mktemp -d "$test_base/p6-pg18.XXXXXXXX")"
container="p6-dietas-pg18-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT
docker run --rm --network none -d --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$repo_dir:$repo_dir:ro" -v "$test_dir:$test_dir:rw" \
  postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null
cat > "$test_dir/psql" <<'SH'
#!/usr/bin/env bash
if [[ "${VEC_P6_TEST_HOST_PSQL_FAIL:-}" == 1 ]]; then
  echo 'psql host prohibido en transporte contenedor' >&2
  exit 88
fi
for argumento in "$@"; do
  case "$argumento" in
    postgresql://*|postgres://*|*password=*)
      echo 'credencial o DSN en argv de psql' >&2
      exit 2;;
  esac
done
printf '%s\0' "$@" >> "$VEC_P6_ARGV_LOG"
exec docker exec -i -u root \
  --env "PGSERVICE=$PGSERVICE" --env "PGSERVICEFILE=$PGSERVICEFILE" \
  --env "PGPASSFILE=$PGPASSFILE" "$VEC_P6_CONTAINER" psql "$@"
SH
chmod 700 "$test_dir/psql"
cat > "$test_dir/podman" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == exec && "${2:-}" == -i ]] || { echo 'podman de prueba exige exec -i' >&2; exit 2; }
for argumento in "$@"; do
  case "$argumento" in
    postgresql://*|postgres://*|*password=*)
      echo 'credencial o DSN en argv de podman' >&2
      exit 2;;
  esac
done
printf '%s\0' "$@" >> "$VEC_P6_PODMAN_ARGV_LOG"
if [[ "${VEC_P6_TEST_PODMAN_FAIL:-}" == antes ]]; then exit 77; fi
if [[ "${VEC_P6_TEST_PODMAN_FAIL:-}" == sin_dba ]]; then printf '%s\n' '180004|false|true'; exit 0; fi
if [[ "${VEC_P6_TEST_PODMAN_FAIL:-}" == pg17 ]]; then printf '%s\n' '170000|true|true'; exit 0; fi
if [[ "${VEC_P6_TEST_PODMAN_FAIL:-}" == despues_rollback && " $* " == *p6_finalizar=ROLLBACK* ]]; then
  docker "$@"
  exit 77
fi
exec docker "$@"
SH
chmod 700 "$test_dir/podman"
cat > "$test_dir/pg_service.conf" <<'SERVICE'
[p6fixture]
host=127.0.0.1
port=5432
dbname=postgres
user=postgres
SERVICE
: > "$test_dir/pgpass"
chmod 600 "$test_dir/pg_service.conf" "$test_dir/pgpass"
export VEC_P6_CONTAINER="$container" VEC_P6_EVIDENCIA_DIR="$test_dir"
export VEC_P6_ARGV_LOG="$test_dir/psql-argv.bin"
export VEC_P6_PODMAN_ARGV_LOG="$test_dir/podman-argv.bin"
export PGSERVICE=p6fixture PGSERVICEFILE="$test_dir/pg_service.conf" PGPASSFILE="$test_dir/pgpass"
export PATH="$test_dir:$PATH"
run_p6_contenedor() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE \
    VEC_P6_POSTGRES_CONTAINER="$container" VEC_P6_APLICAR="${VEC_P6_APLICAR:-}" \
    VEC_P6_TEST_HOST_PSQL_FAIL=1 \
    bash "$base_dir/ejecutar.sh" "$@"
}
psql -Xq -v ON_ERROR_STOP=1 -f "$repo_dir/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null
psql -Xq -v ON_ERROR_STOP=1 -f "$repo_dir/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql" >/dev/null
python3 "$base_dir/fixture_pg18.py" > "$test_dir/fixture.sql"
psql -Xq -v ON_ERROR_STOP=1 -f "$test_dir/fixture.sql" >/dev/null

cat > "$test_dir/refresco-sesiones.sql" <<'SQL'
BEGIN;
SELECT 'antes='||count(*) FROM pg_stat_activity WHERE usename='vec_dietas_r1d_dietas_desarrollo';
SELECT pg_advisory_lock(314159);
SELECT pg_sleep(5);
SELECT 'sin_refresco='||count(*) FROM pg_stat_activity WHERE usename='vec_dietas_r1d_dietas_desarrollo';
SELECT pg_stat_clear_snapshot();
SELECT 'con_refresco='||count(*) FROM pg_stat_activity WHERE usename='vec_dietas_r1d_dietas_desarrollo';
ROLLBACK;
SQL
psql -XAtq -v ON_ERROR_STOP=1 -f "$test_dir/refresco-sesiones.sql" > "$test_dir/refresco-sesiones.out" &
observador=$!
visto=f
for _ in $(seq 1 40); do
  visto="$(psql -XAtq -v ON_ERROR_STOP=1 -c "SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND objid='314159'::oid AND granted)")"
  [[ "$visto" == t ]] && break
  sleep 0.1
done
[[ "$visto" == t ]] || { echo 'observador PG18 no adquirio el bloqueo de sincronizacion' >&2; exit 1; }
docker exec -u root "$container" psql -h 127.0.0.1 -U vec_dietas_r1d_dietas_desarrollo -d postgres \
  -XAtq -v ON_ERROR_STOP=1 -c 'SELECT pg_sleep(8)' > "$test_dir/conexion-entre-lecturas.out" &
conexion=$!
wait "$observador"
wait "$conexion"
python3 - "$test_dir/refresco-sesiones.out" <<'PY'
from pathlib import Path
import sys
lineas = [line.strip() for line in Path(sys.argv[1]).read_text().splitlines() if line.strip()]
if lineas != ["antes=0", "sin_refresco=0", "con_refresco=1"]:
    raise SystemExit(f"snapshot PG18 no demostro refresco: {lineas}")
PY

# El transporte contenedor debe rechazar tanto una avería previa como la
# pérdida de la respuesta después de un ROLLBACK ejecutado.
if VEC_P6_TEST_PODMAN_FAIL=antes run_p6_contenedor --inventario \
    > "$test_dir/transporte-antes.out" 2> "$test_dir/transporte-antes.err"; then
  echo 'fallo de transporte previo fue aceptado' >&2; exit 1
fi
if VEC_P6_TEST_PODMAN_FAIL=sin_dba run_p6_contenedor --inventario \
    > "$test_dir/sin-dba.out" 2> "$test_dir/sin-dba.err"; then
  echo 'sesion sin DBA fue aceptada' >&2; exit 1
fi
if VEC_P6_TEST_PODMAN_FAIL=pg17 run_p6_contenedor --inventario \
    > "$test_dir/pg17.out" 2> "$test_dir/pg17.err"; then
  echo 'PostgreSQL 17 fue aceptado' >&2; exit 1
fi
bash "$base_dir/ejecutar.sh" --inventario >/dev/null
if VEC_P6_TEST_PODMAN_FAIL=despues_rollback run_p6_contenedor --rollback \
    > "$test_dir/transporte-rollback.out" 2> "$test_dir/transporte-rollback.err"; then
  echo 'respuesta perdida tras ROLLBACK fue aceptada' >&2; exit 1
fi
grep -q 'revisar inventario privado' "$test_dir/transporte-rollback.err" || {
  echo 'fallo de transporte no dejo diagnostico recuperable' >&2; exit 1;
}

# Una sesion nominal viva hace abortar la operacion antes de la revocacion.
docker exec -u root "$container" psql -h 127.0.0.1 -U vec_dietas_r1d_dietas_desarrollo -d postgres \
  -XAtq -v ON_ERROR_STOP=1 -c 'SELECT pg_sleep(8)' > "$test_dir/sesion-viva.out" &
sesion_viva=$!
sesion_detectada=f
for _ in $(seq 1 40); do
  sesion_detectada="$(psql -XAtq -v ON_ERROR_STOP=1 -c "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE usename='vec_dietas_r1d_dietas_desarrollo')")"
  [[ "$sesion_detectada" == t ]] && break
  sleep 0.1
done
[[ "$sesion_detectada" == t ]] || { echo 'sesion nominal de prueba no aparecio' >&2; exit 1; }
if run_p6_contenedor --rollback > "$test_dir/sesion-viva-p6.out" 2> "$test_dir/sesion-viva-p6.err"; then
  echo 'operacion acepto sesion nominal viva' >&2; exit 1
fi
wait "$sesion_viva"

psql -Xq -v ON_ERROR_STOP=1 -c 'CREATE ROLE vec_dietas_r1d_auditoria_frontera_desarrollo LOGIN;' >/dev/null
if run_p6_contenedor --rollback > "$test_dir/noveno.out" 2> "$test_dir/noveno.err"; then
  echo 'noveno LOGIN no fue rechazado' >&2; exit 1
fi
psql -Xq -v ON_ERROR_STOP=1 -c 'DROP ROLE vec_dietas_r1d_auditoria_frontera_desarrollo;' >/dev/null

psql -Xq -v ON_ERROR_STOP=1 -c 'CREATE ROLE p6_otro_login LOGIN; GRANT vec_dietas_ejecutor TO p6_otro_login WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;' >/dev/null
if run_p6_contenedor --rollback > "$test_dir/ruta-dietas.out" 2> "$test_dir/ruta-dietas.err"; then
  echo 'LOGIN ajeno con grupo Dietas no fue rechazado' >&2; exit 1
fi
psql -Xq -v ON_ERROR_STOP=1 -c 'REVOKE vec_dietas_ejecutor FROM p6_otro_login; DROP ROLE p6_otro_login;' >/dev/null

run_p6_contenedor --rollback
python3 - "$test_dir" <<'PY'
import glob
import json
import sys

inventarios = glob.glob(sys.argv[1] + "/p6-inventario-*.json")
if not inventarios:
    raise SystemExit("falta inventario P6")
with open(inventarios[-1], encoding="utf-8") as handle:
    inventario = json.load(handle)
rutas = inventario["rutas_grupos"]
if not any(r["camino"] == ["vec_autorizacion_fuente", "p6_ruta_intermedia", "p6_ct_shared_login"]
           and r["uso_efectivo"] and r["set_efectivo"] for r in rutas):
    raise SystemExit("inventario no muestra herencia/SET de login ajeno sobre grupo compartido")
if not any(r["grupo"] == "vec_autorizacion_fuente" and r["login"] == "p6_ct_shared_login"
           and r["uso"] and r["set"] for r in inventario["login_con_grupo"]):
    raise SystemExit("inventario no muestra LOGIN con acceso efectivo a grupo compartido")
PY
run_p6_contenedor --commit 2> "$test_dir/commit-sin-autorizacion.err" && {
  echo 'COMMIT sin bandera fue aceptado' >&2; exit 1;
}
VEC_P6_APLICAR=SI-P6-REVISADO run_p6_contenedor --commit
if docker exec -u root "$container" psql -h 127.0.0.1 -U vec_dietas_r1d_dietas_desarrollo -d postgres \
    -c 'SELECT 1' > "$test_dir/conexion-rechazada.out" 2> "$test_dir/conexion-rechazada.err"; then
  echo 'conexion Dietas posterior a COMMIT fue aceptada' >&2; exit 1
fi
grep -q 'not permitted to log in' "$test_dir/conexion-rechazada.err" || {
  echo 'conexion rechazada por causa distinta de NOLOGIN' >&2; exit 1;
}
if VEC_P6_APLICAR=SI-P6-REVISADO run_p6_contenedor --commit \
    > "$test_dir/reentrada.out" 2> "$test_dir/reentrada.err"; then
  echo 'reentrada COMMIT no fue rechazada' >&2; exit 1
fi

resultado="$(psql -XAtq -v ON_ERROR_STOP=1 -c "
 SELECT (SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id='dietas_r1d_0123456789abcdef'),
        (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin),
        (SELECT a.documento->>'estado' FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref),
        (SELECT n FROM vec_ct_sentinel.control),(SELECT n FROM vec_bolsa_sentinel.control)
 ")"
[[ "$resultado" == '2|0|revocada|7|11' ]] || { echo "postcondicion PG18 inesperada: $resultado" >&2; exit 1; }
echo "PG18 jerarquia y snapshot fresco, noveno rechazado, ROLLBACK/COMMIT/reentrada y conexion denegada OK; dos versiones, ocho NOLOGIN, CT/Bolsa testigos 7/11. Evidencia privada: $test_dir"
