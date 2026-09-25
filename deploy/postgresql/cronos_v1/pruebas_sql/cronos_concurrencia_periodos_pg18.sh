#!/usr/bin/env bash
set -euo pipefail
# Base PG18 desechable con UP 000001/000005. Uso: script NOMBRE_CONTENEDOR
target_container=${1:?falta contenedor PostgreSQL 18 desechable}
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT

docker exec -i "$target_container" psql -U postgres -v ON_ERROR_STOP=1 -v VERBOSITY=verbose >"$test_dir/primera.log" 2>&1 <<'SQL' &
BEGIN;
SET LOCAL application_name='vec_cronos_sql_conc_A';
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_cccccccccccccccccccccc',true);
INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
(autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
VALUES ('teletrabajo:cronos:concurrente1','emp_cccccccccccccccccccccc',
 tstzrange('2026-09-24T06:00:00Z','2026-09-24T15:00:00Z','[)'),
 'resolucion:sintetica:conc1','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
 'auditoria:sintetica:conc1',clock_timestamp());
SELECT pg_sleep(12);
COMMIT;
SQL
first_pid=$!
for attempt in $(seq 1 100); do
 if rg -q '^INSERT 0 1$' "$test_dir/primera.log"; then break; fi
 if ! kill -0 "$first_pid" 2>/dev/null; then cat "$test_dir/primera.log"; exit 1; fi
 sleep 0.05
done
if ! rg -q '^INSERT 0 1$' "$test_dir/primera.log"; then cat "$test_dir/primera.log"; exit 1; fi

docker exec -i "$target_container" psql -U postgres -v ON_ERROR_STOP=1 -v VERBOSITY=verbose >"$test_dir/segunda.log" 2>&1 <<'SQL' &
BEGIN;
SET LOCAL application_name='vec_cronos_sql_conc_B';
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_cccccccccccccccccccccc',true);
INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
(autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
VALUES ('teletrabajo:cronos:concurrente2','emp_cccccccccccccccccccccc',
 tstzrange('2026-09-24T10:00:00Z','2026-09-24T17:00:00Z','[)'),
 'resolucion:sintetica:conc2','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
 'auditoria:sintetica:conc2',clock_timestamp());
COMMIT;
SQL
second_pid=$!
observed_wait=false
for attempt in $(seq 1 100); do
 lock_wait=$(docker exec "$target_container" psql -U postgres -At -v ON_ERROR_STOP=1 -c   "SELECT EXISTS (SELECT 1 FROM pg_stat_activity a JOIN pg_stat_activity b ON a.pid<>b.pid JOIN pg_locks l ON l.pid=b.pid WHERE a.application_name='vec_cronos_sql_conc_A' AND a.xact_start IS NOT NULL AND b.application_name='vec_cronos_sql_conc_B' AND l.locktype='advisory' AND NOT l.granted AND b.wait_event_type='Lock');")
 if [ "$lock_wait" = t ]; then observed_wait=true; break; fi
 if ! kill -0 "$first_pid" 2>/dev/null || ! kill -0 "$second_pid" 2>/dev/null; then break; fi
 sleep 0.05
done
set +e
wait "$first_pid"; first_rc=$?
wait "$second_pid"; second_rc=$?
set -e
if [ "$observed_wait" != true ] || [ "$first_rc" -ne 0 ] || [ "$second_rc" -eq 0 ]  || ! rg -q 'PC002: periodo teletrabajo solapado' "$test_dir/segunda.log"; then
 cat "$test_dir/primera.log" "$test_dir/segunda.log"
 printf '%s\n' "espera advisory observada=$observed_wait, primera=$first_rc, segunda=$second_rc"
 exit 1
fi
rows=$(docker exec -i "$target_container" psql -U postgres -At -v ON_ERROR_STOP=1 <<'SQL'
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SELECT set_config('vec.cronos.empleado_ref','emp_cccccccccccccccccccccc',true);
SELECT count(*) FROM vec_cronos_v1.teletrabajo_autorizacion WHERE empleado_ref='emp_cccccccccccccccccccccc';
ROLLBACK;
SQL
)
if ! printf '%s\n' "$rows" | rg -q '^1$'; then printf '%s\n' "$rows"; exit 1; fi
printf '%s\n' 'PG18 READ COMMITTED: espera advisory observada, un COMMIT, segundo PC002, una autorización.'
# Dos sesiones REPEATABLE READ concurrentes fallan antes del efecto.
intentar_rr() {
 docker exec -i "$target_container" psql -U postgres -v ON_ERROR_STOP=1 -v VERBOSITY=verbose <<'SQL'
BEGIN ISOLATION LEVEL REPEATABLE READ;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_cccccccccccccccccccccc',true);
INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
(autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
VALUES ('teletrabajo:cronos:rr_concurrente','emp_cccccccccccccccccccccc',
 tstzrange('2026-09-25T06:00:00Z','2026-09-25T15:00:00Z','[)'),
 'resolucion:sintetica:rr_conc','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
 'auditoria:sintetica:rr_conc',clock_timestamp());
COMMIT;
SQL
}
set +e
intentar_rr >"$test_dir/rr1.log" 2>&1 & rr1_pid=$!
intentar_rr >"$test_dir/rr2.log" 2>&1 & rr2_pid=$!
wait "$rr1_pid"; rr1_rc=$?
wait "$rr2_pid"; rr2_rc=$?
set -e
if [ "$rr1_rc" -eq 0 ] || [ "$rr2_rc" -eq 0 ] \
 || ! rg -q 'PC003: aislamiento Cronos no admitido' "$test_dir/rr1.log" \
 || ! rg -q 'PC003: aislamiento Cronos no admitido' "$test_dir/rr2.log"; then
 cat "$test_dir/rr1.log" "$test_dir/rr2.log"
 exit 1
fi
printf '%s\n' 'PG18 REPEATABLE READ concurrente: dos PC003, sin nuevos efectos.'
