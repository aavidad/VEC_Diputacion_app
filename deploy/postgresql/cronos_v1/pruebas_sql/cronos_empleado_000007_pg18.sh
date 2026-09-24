#!/usr/bin/env bash
# Ensayo PG18.4 desechable de cronos_v1 000007 con fachadas AD3-53 de PRUEBA:
# instala 000001–000006, ROLLBACK y COMMIT de 000007, ACL, recorrido de saldo,
# disponibilidad, fichaje remoto con replay y recuperación, denegaciones y
# conservación tras reiniciar PostgreSQL. No acredita MAC, COSE ni gobierno V3.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
container="vec-cronos-000007-${RANDOM}${RANDOM}"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --network none --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
esperar() {
 for _ in $(seq 1 80); do
  if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
   sleep 0.3
   if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then return 0; fi
  fi
  sleep 0.3
 done
 echo 'PostgreSQL no disponible' >&2; exit 2
}
esperar
run() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U "${2:-postgres}" -d postgres -c "$1"; }
comprobar() {
 local obtenido
 obtenido=$(scalar "$1" "${4:-postgres}")
 if [ "$obtenido" != "$2" ]; then printf 'FALLO %s: %s (esperado %s)\n' "$3" "$obtenido" "$2" >&2; exit 1; fi
 printf 'OK %s\n' "$3"
}
como_app() { scalar "$1" login_cronos_prueba; }
[ "$(scalar 'SHOW server_version')" = 18.4 ] || { echo 'Se requiere PostgreSQL 18.4' >&2; exit 2; }

run < "$mig/000001_esquema_marcajes.up.sql" >/dev/null
run < "$base_dir/cronos_empleado_000007_stub_ad3.sql" >/dev/null
for n in 000002_registrar_marcaje_propio 000003_resultado_ejecucion_marcaje 000004_programacion_y_libro_saldo 000005_periodos_teletrabajo_y_cierre_remoto 000006_rls_resultado_ejecucion; do
 run < "$mig/$n.up.sql" >/dev/null
done
sed '$s/^COMMIT;$/ROLLBACK;/' "$mig/000007_saldo_y_fichaje_remoto_empleado.up.sql" | run >/dev/null
comprobar "SELECT to_regclass('vec_cronos_v1.marcaje_remoto_autorizado') IS NULL AND to_regprocedure('vec_cronos_v1.registrar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL" t 'ROLLBACK de 000007 sin objetos'
comprobar "SELECT strpos(prosrc,'marcaje_remoto_autorizado')=0 FROM pg_proc WHERE oid='vec_cronos_v1.bloquear_marcaje_remoto_sin_consumidor_v1()'::regprocedure" t 'ROLLBACK conserva el bloqueo de 000005'
run < "$mig/000007_saldo_y_fichaje_remoto_empleado.up.sql" >/dev/null
if run < "$mig/000007_saldo_y_fichaje_remoto_empleado.up.sql" >/dev/null 2>&1; then echo 'FALLO segunda aplicación aceptada' >&2; exit 1; fi
printf 'OK segunda aplicación de 000007 rechazada\n'
run < "$base_dir/cronos_empleado_000007_preparar.sql" >/dev/null

# ACL: el ejecutor sólo invoca las cuatro funciones; ni tablas ni auxiliares.
for f in consultar_saldo_propio_v1 consultar_estado_remoto_propio_v1 registrar_marcaje_remoto_v1 recuperar_marcaje_remoto_v1; do
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$f(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.$f(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND prosecdef FROM pg_proc WHERE oid='vec_cronos_v1.$f(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure" t "ACL de $f"
done
comprobar "SELECT NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.teletrabajo_en_v1(text,timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.acreditar_empleado_contexto_v1(jsonb,text,text,text,timestamptz)','EXECUTE')" t 'auxiliares sin EXECUTE'
comprobar "SELECT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')" t 'denegaciones sólo por el auditor'
for t in saldo_acceso remoto_consulta_acceso marcaje_remoto_autorizado marcaje_remoto_recuperacion denegacion_frontera; do
 comprobar "SELECT relrowsecurity AND relforcerowsecurity AND NOT has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$t','SELECT,INSERT,UPDATE,DELETE') FROM pg_class WHERE oid='vec_cronos_v1.$t'::regclass" t "RLS FORCE y sin DML directo en $t"
done
comprobar "SELECT prueba.espera_error(\$\$INSERT INTO vec_cronos_v1.marcaje_remoto_autorizado VALUES ('marcaje:cronos:x','emp_AAAAAAAAAAAAAAAAAAAAAA','xxxxxxxx','teletrabajo:cronos:sintetico-a','d',now())\$\$,'42501')" 'OK 42501' 'la aplicación no escribe autorizaciones remotas' login_cronos_prueba

A=emp_AAAAAAAAAAAAAAAAAAAAAA
B=emp_BBBBBBBBBBBBBBBBBBBBBB
# Disponibilidad: teletrabajo vigente y sin él.
comprobar "SELECT d->>'autorizado'||'|'||(d->>'continuidad_confirmada')||'|'||(d->'movimientos_permitidos')::text FROM prueba.disponibilidad('$A','','n-disp-1') d" 'true|true|["entrada"]' 'disponibilidad con teletrabajo' login_cronos_prueba
comprobar "SELECT d::text FROM prueba.disponibilidad('$B','','n-disp-2') d" '{"autorizado": false, "continuidad_confirmada": false, "movimientos_permitidos": []}' 'disponibilidad sin teletrabajo' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.disponibilidad('$A','','n-disp-1')\$\$,'PC003')" 'OK PC003' 'decisión reutilizada rechazada' login_cronos_prueba

# ROLLBACK de un fichaje no deja rastro.
docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U login_cronos_prueba -d postgres >/dev/null <<SQL
BEGIN;
SELECT prueba.fichar('$A','clave-rollback-01','entrada','n-rb-1');
ROLLBACK;
SQL
comprobar "SELECT count(*) FROM vec_cronos_v1.marcaje_original" 0 'ROLLBACK del fichaje sin hechos'

recibo=$(como_app "SELECT prueba.fichar('$A','clave-remota-0001','entrada','n-fichar-1')::text")
ref=$(printf '%s' "$recibo" | sed -E 's/.*"referencia": "([^"]+)".*/\1/')
case "$recibo" in *'"replay": false'*) printf 'OK fichaje remoto registrado %s\n' "$ref";; *) echo "FALLO fichaje: $recibo" >&2; exit 1;; esac
comprobar "SELECT (d->>'referencia')||'|'||(d->>'replay') FROM prueba.fichar('$A','clave-remota-0001','entrada','n-fichar-2') d" "$ref|true" 'replay idempotente con nueva decisión' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$A','clave-remota-0001','salida','n-fichar-3')\$\$,'PC002')" 'OK PC002' 'misma clave con otro movimiento en conflicto' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$A','clave-remota-0002','entrada','n-fichar-4')\$\$,'PC005')" 'OK PC005' 'secuencia no permitida' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$B','clave-remota-b001','entrada','n-fichar-5')\$\$,'PC004')" 'OK PC004' 'sin teletrabajo no ficha' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$A','clave-remota-0003','salida','n-fichar-6','[]'::jsonb)\$\$,'PC003')" 'OK PC003' 'contexto sin empleado deniega' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$A','clave-remota-0003','salida','n-fichar-7',jsonb_build_array(jsonb_build_object('tipo','empleado','estado','activo','referencia','$A','vigente_desde',now()-interval '1 day','vigente_hasta',now()+interval '1 day'),jsonb_build_object('tipo','empleado','estado','activo','referencia','emp_CCCCCCCCCCCCCCCCCCCCCC','vigente_desde',now()-interval '1 day','vigente_hasta',now()+interval '1 day')))\$\$,'PC003')" 'OK PC003' 'contexto con dos empleados deniega' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar_por_terminal('$A','clave-terminal-01','n-fichar-8')\$\$,'PC003')" 'OK PC003' 'la vía de terminal no admite canal remoto' login_cronos_prueba
comprobar "SELECT d->>'continuidad_confirmada'||'|'||(d->'movimientos_permitidos')::text FROM prueba.disponibilidad('$A','','n-disp-3') d" 'true|["salida", "inicio_pausa"]' 'disponibilidad tras la entrada' login_cronos_prueba
comprobar "SELECT (d->'movimientos_permitidos')::text FROM prueba.disponibilidad('$A','clave-remota-0001','n-disp-4') d" '["entrada"]' 'disponibilidad concilia la clave ya registrada' login_cronos_prueba

# Recuperación y saldo.
comprobar "SELECT (d->>'referencia')||'|'||(d->>'replay') FROM prueba.recuperar('$A','clave-remota-0001','entrada','n-rec-1') d" "$ref|true" 'recuperación del recibo' login_cronos_prueba
comprobar "SELECT d::text FROM prueba.recuperar('$A','clave-remota-0999','entrada','n-rec-2') d" '{"ausente": true}' 'ausencia confirmada y auditada' login_cronos_prueba
comprobar "SELECT d::text FROM prueba.recuperar('$B','clave-remota-0001','entrada','n-rec-3') d" '{"ausente": true}' 'clave ajena se ve ausente' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.recuperar('$A','clave-remota-0001','salida','n-rec-4')\$\$,'PC002')" 'OK PC002' 'recuperación con otro movimiento en conflicto' login_cronos_prueba
comprobar "SELECT (s->>'empleado_ref')||'|'||jsonb_array_length(s->'marcajes')||'|'||(s->'marcajes'->0->>'tipo_origen')||'|'||jsonb_array_length(s->'jornadas') FROM prueba.saldo('$A','n-saldo-1') s" "$A|1|remoto|1" 'saldo propio con el fichaje remoto' login_cronos_prueba
comprobar "SELECT prueba.espera_error(\$\$SELECT vec_cronos_v1.consultar_libro_saldo_interno_v1('$A',current_date,current_date,'Europe/Madrid')\$\$,'42501')" 'OK 42501' 'lectura interna sin EXECUTE' login_cronos_prueba
comprobar "SELECT vec_cronos_v1.registrar_denegacion_frontera_v1('corr_no_disponible','sin_empleado','/api/interna/cronos/saldos/propio','GET','') LIKE 'denegacion:cronos:%'" t 'denegación de frontera auditada' login_cronos_auditor_prueba

conteo="SELECT (SELECT count(*) FROM vec_cronos_v1.marcaje_original)||'/'||(SELECT count(*) FROM vec_cronos_v1.marcaje_remoto_autorizado)||'/'||(SELECT count(*) FROM vec_cronos_v1.marcaje_historia)||'/'||(SELECT count(*) FROM vec_cronos_v1.marcaje_outbox)||'/'||(SELECT count(*) FROM vec_cronos_v1.marcaje_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.marcaje_remoto_recuperacion)||'/'||(SELECT count(*) FROM vec_cronos_v1.saldo_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.remoto_consulta_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.denegacion_frontera)"
antes=$(scalar "$conteo")
comprobar "SELECT '$antes'" '1/1/1/1/1/3/1/4/1' 'filas únicas: hecho, autorización, historia, outbox y accesos'

inicio=$(scalar "SELECT pg_postmaster_start_time()")
docker restart "$container" >/dev/null
esperar
[ "$(scalar "SELECT pg_postmaster_start_time()")" != "$inicio" ] || { echo 'FALLO PostgreSQL no se reinició' >&2; exit 1; }
printf 'OK PostgreSQL reiniciado\n'
comprobar "$conteo" "$antes" 'mismas filas tras reiniciar'
comprobar "SELECT (d->>'referencia')||'|'||(d->>'replay') FROM prueba.recuperar('$A','clave-remota-0001','entrada','n-rec-5') d" "$ref|true" 'mismo recibo tras reiniciar' login_cronos_prueba
comprobar "SELECT (d->>'referencia')||'|'||(d->>'replay') FROM prueba.fichar('$A','clave-remota-0001','entrada','n-fichar-9') d" "$ref|true" 'replay tras reiniciar sin duplicar' login_cronos_prueba
comprobar "SELECT count(*) FROM vec_cronos_v1.marcaje_original" 1 'un único hecho tras reiniciar'

# Semántica real del bloqueo de teletrabajo en SERIALIZABLE: un fichaje cuya
# instantánea precede a la confirmación de una revocación se registra y SSI lo
# ordena como «fichaje antes que revocación»; uno iniciado después se rechaza.
run -c 'CREATE EXTENSION IF NOT EXISTS dblink' >/dev/null
revocar="BEGIN; SET LOCAL ROLE vec_cronos_v1_propietario; SELECT set_config('vec.cronos.empleado_ref','$A',true); INSERT INTO vec_cronos_v1.teletrabajo_revocacion VALUES ('revocacion:cronos:sintetica-a','teletrabajo:cronos:sintetico-a','$A',clock_timestamp()-interval '1 minute','resolucion:sintetica:rev-a','per_RRRRRRRRRRRRRRRRRRRRRR','auditoria:sintetica:rev-a',clock_timestamp()); COMMIT;"
docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres >/dev/null <<SQL
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT count(*) FROM vec_cronos_v1.teletrabajo_revocacion;
SELECT dblink_exec('dbname=postgres user=postgres', \$r\$$revocar\$r\$);
SET LOCAL ROLE login_cronos_prueba;
SELECT prueba.fichar('$A','clave-remota-0004','salida','n-fichar-10');
COMMIT;
SQL
comprobar "SELECT count(*)||'|'||(SELECT count(*) FROM vec_cronos_v1.teletrabajo_revocacion) FROM vec_cronos_v1.marcaje_original WHERE movimiento='salida'" '1|1' 'fichaje con instantánea previa a la revocación: orden fichaje antes que revocación'
comprobar "SELECT prueba.espera_error(\$\$SELECT prueba.fichar('$A','clave-remota-0005','entrada','n-fichar-11')\$\$,'PC004')" 'OK PC004' 'fichaje iniciado tras la revocación rechazado' login_cronos_prueba
printf 'PG18.4: cronos_v1 000007 verificado con fachadas AD3-53 DE PRUEBA; no acredita la cadena V3 real.\n'
