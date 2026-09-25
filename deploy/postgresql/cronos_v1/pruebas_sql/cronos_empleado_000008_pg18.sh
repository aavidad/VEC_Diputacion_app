#!/usr/bin/env bash
# Ensayo PG18.4 desechable de cronos_v1 000008 con fachadas AD3-53/70 de
# PRUEBA: instala 000001–000007, ROLLBACK y COMMIT de 000008, ACL, RLS,
# movimientos con calendario, absentismos y correcciones; corrección de un
# olvido con replay y conflicto; permisos del año; solicitud con cómputo de
# laborables, cupos, solapes y replay; conservación tras reiniciar. No
# acredita MAC, COSE ni gobierno V3. El contenedor se borra al salir.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
container="vec-cronos-000008-${RANDOM}${RANDOM}"
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
app() { comprobar "$1" "$2" "$3" login_cronos_prueba; }
[ "$(scalar 'SHOW server_version')" = 18.4 ] || { echo 'Se requiere PostgreSQL 18.4' >&2; exit 2; }

run < "$mig/000001_esquema_marcajes.up.sql" >/dev/null
run < "$base_dir/cronos_empleado_000007_stub_ad3.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_stub_ad3.sql" >/dev/null
for n in 000002_registrar_marcaje_propio 000003_resultado_ejecucion_marcaje 000004_programacion_y_libro_saldo 000005_periodos_teletrabajo_y_cierre_remoto 000006_rls_resultado_ejecucion 000007_saldo_y_fichaje_remoto_empleado; do
 run < "$mig/$n.up.sql" >/dev/null
done
m8="$mig/000008_movimientos_y_permisos_empleado.up.sql"
ruta_previa=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'")
sed '$s/^COMMIT;$/ROLLBACK;/' "$m8" | run >/dev/null
comprobar "SELECT to_regclass('vec_cronos_v1.permiso_catalogo') IS NULL AND to_regprocedure('vec_cronos_v1.solicitar_permiso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL" t 'ROLLBACK de 000008 sin objetos'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'" "$ruta_previa" 'ROLLBACK conserva las rutas auditadas'
run < "$m8" >/dev/null
if run < "$m8" >/dev/null 2>&1; then echo 'FALLO segunda aplicación aceptada' >&2; exit 1; fi
printf 'OK segunda aplicación de 000008 rechazada\n'
# El catálogo sintético de desarrollo se instala y es idempotente; se revierte
# para que el arnés use el suyo.
semilla="$base_dir/../datos_sinteticos/catalogo_permisos_sintetico_duda41.sql"
obtenido=$({ sed '/^COMMIT;$/d' "$semilla"; sed -n '/^INSERT/,/DO NOTHING;/p' "$semilla"; echo "SELECT count(*)||'|'||bool_and(sintetico)||'|'||count(*) FILTER (WHERE solicitable) FROM vec_cronos_v1.permiso_catalogo; ROLLBACK;"; } | docker exec -i "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres | tail -1)
[ "$obtenido" = '25|true|15' ] || { echo "FALLO catálogo sintético: $obtenido" >&2; exit 1; }
printf 'OK catálogo sintético instalable dos veces sin duplicar\n'
run < "$base_dir/cronos_empleado_000007_preparar.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_preparar.sql" >/dev/null

firma='(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
for f in consultar_movimientos_propio_v1 solicitar_correccion_propia_v1 consultar_permisos_propio_v1 solicitar_permiso_propio_v1; do
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('public','vec_cronos_v1.$f$firma','EXECUTE') AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog'] FROM pg_proc WHERE oid='vec_cronos_v1.$f$firma'::regprocedure" t "ACL y search_path de $f"
done
comprobar "SELECT NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.material_propio_v1(text,text[])','EXECUTE')" t 'auxiliares sin EXECUTE'
for t in calendario_dia calendario_asignacion permiso_catalogo correccion_solicitud correccion_actuacion permiso_solicitud permiso_estado solicitud_outbox movimientos_acceso permisos_acceso solicitud_replay; do
 comprobar "SELECT relrowsecurity AND relforcerowsecurity AND NOT has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$t','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege('public','vec_cronos_v1.$t','SELECT') FROM pg_class WHERE oid='vec_cronos_v1.$t'::regclass" t "RLS FORCE y sin DML directo en $t"
done
comprobar "SELECT prueba.espera_error(\$\$DELETE FROM vec_cronos_v1.permiso_estado\$\$,'55000')" 'OK 55000' 'historia de permisos inmutable'
comprobar "SELECT vec_cronos_v1.registrar_denegacion_frontera_v1('corr_no_disponible','acceso_denegado','/api/interna/cronos/permisos/solicitudes','POST','') LIKE 'denegacion:cronos:%'" t 'denegación en ruta nueva auditada' login_cronos_auditor_prueba

A=emp_AAAAAAAAAAAAAAAAAAAAAA
B=emp_BBBBBBBBBBBBBBBBBBBBBB
# Movimientos: calendario, absentismos y marcajes por día.
app "SELECT (d->'calendario'->>'disponible')||'|'||jsonb_array_length(d->'calendario'->'dias')||'|'||jsonb_array_length(d->'absentismos')||'|'||(d->'absentismos'->0->>'pendiente_justificar') FROM prueba.movimientos('$A',make_date(extract(year FROM current_date)::int,1,1),make_date(extract(year FROM current_date)::int,12,31),'n-mov-1') d" 'true|2|1|true' 'movimientos del año con calendario y absentismo'
app "SELECT (d->'calendario')::text||'|'||jsonb_array_length(d->'absentismos') FROM prueba.movimientos('$B',current_date,current_date,'n-mov-2') d" '{"dias": [], "disponible": false}|0' 'sin calendario lo dice y no ve lo ajeno'
app "SELECT prueba.espera_error(\$\$SELECT prueba.movimientos('$A',current_date,current_date,'n-mov-1')\$\$,'PC003')" 'OK PC003' 'decisión reutilizada rechazada'
app "SELECT prueba.espera_error(\$\$SELECT prueba.movimientos('$A',current_date-400,current_date,'n-mov-3')\$\$,'PC001')" 'OK PC001' 'periodo de más de un año rechazado'

# Corrección de un olvido: nunca toca el marcaje.
docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U login_cronos_prueba -d postgres >/dev/null <<SQL
BEGIN;
SELECT prueba.corregir('$A','corr-rollback-01',current_date-1,'08:00','n-cor-rb');
ROLLBACK;
SQL
comprobar "SELECT count(*) FROM vec_cronos_v1.correccion_solicitud" 0 'ROLLBACK de la corrección sin hechos'
recibo=$(scalar "SELECT prueba.corregir('$A','corr-olvido-0001',current_date-1,'08:00','n-cor-1')::text" login_cronos_prueba)
case "$recibo" in *'"replay": false'*'"estado": "pendiente_responsable"'*|*'"estado": "pendiente_responsable"'*'"replay": false'*) printf 'OK corrección registrada pendiente de responsable\n';; *) echo "FALLO corrección: $recibo" >&2; exit 1;; esac
rc=$(printf '%s' "$recibo" | sed -E 's/.*"recibo_ref": "([^"]+)".*/\1/')
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.corregir('$A','corr-olvido-0001',current_date-1,'08:00','n-cor-2') d" "$rc|true" 'replay de la corrección con nueva decisión'
app "SELECT prueba.espera_error(\$\$SELECT prueba.corregir('$A','corr-olvido-0001',current_date-1,'09:00','n-cor-3')\$\$,'PC002')" 'OK PC002' 'misma clave con otra hora en conflicto'
app "SELECT prueba.espera_error(\$\$SELECT prueba.corregir('$B','corr-olvido-0001',current_date-1,'08:00','n-cor-4')\$\$,'PC002')" 'OK PC002' 'clave ajena en conflicto sin revelar datos'
app "SELECT prueba.espera_error(\$\$SELECT prueba.corregir('$A','corr-futuro-0001',current_date+2,'08:00','n-cor-5')\$\$,'PC001')" 'OK PC001' 'olvido a futuro rechazado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.corregir('$A','corr-ajeno-0001',current_date-1,'08:00','n-cor-6','marcaje:cronos:inexistente-01')\$\$,'PC001')" 'OK PC001' 'marcaje original inexistente rechazado'
comprobar "SELECT count(*) FROM vec_cronos_v1.marcaje_original" 0 'la corrección no crea ni edita marcajes'
app "SELECT (d->'correcciones'->0->>'estado')||'|'||jsonb_array_length(d->'correcciones') FROM prueba.movimientos('$A',current_date-7,current_date,'n-mov-4') d" 'pendiente_responsable|1' 'la corrección aparece en movimientos'

# Permisos: catálogo, cómputo y cupos.
app "SELECT jsonb_array_length(d->'catalogo')||'|'||(d->'catalogo'->0->>'sintetico')||'|'||(d->'catalogo'->4->>'solicitable')||'|'||jsonb_array_length(d->'solicitudes') FROM prueba.permisos('$A','n-per-1') d" '5|true|false|1' 'catálogo del año y concedido previo'
lunes=$(scalar "SELECT prueba.semana()")
app "SELECT (d->>'cantidad')||'|'||(d->>'estado')||'|'||(d->>'replay') FROM prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+4,'','','n-pd-1') d" '4|solicitado|false' 'asuntos propios: 4 laborables con festivo'
app "SELECT (d->>'cantidad')||'|'||(d->>'replay') FROM prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+4,'','','n-pd-2') d" '4|true' 'replay de la solicitud'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+3,'','','n-pd-3')\$\$,'PC002')" 'OK PC002' 'misma clave con otras fechas en conflicto'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$A','perm-va-00001','vacaciones',prueba.semana()+3,prueba.semana()+8,'','','n-pd-4')\$\$,'PC010')" 'OK PC010' 'permiso en días solapado rechazado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$A','perm-ap-00002','asuntos-propios',prueba.semana()+7,prueba.semana()+9,'','','n-pd-5')\$\$,'PC007')" 'OK PC007' 'cupo anual superado'
app "SELECT (d->>'cantidad')||'|'||(d->>'unidad') FROM prueba.pedir('$A','perm-hm-00001','horas-medico',prueba.semana()+14,prueba.semana()+14,'09:00','11:30','n-pd-6') d" '150|hora' 'horas de médico en minutos'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$A','perm-hm-00002','horas-medico',prueba.semana()+15,prueba.semana()+15,'09:00','10:00','n-pd-7')\$\$,'PC007')" 'OK PC007' 'cupo horario superado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$A','perm-ma-00001','maternidad',prueba.semana()+20,prueba.semana()+21,'','','n-pd-8')\$\$,'PC009')" 'OK PC009' 'permiso no solicitable desde el portal'
app "SELECT prueba.espera_error(\$\$SELECT prueba.pedir('$B','perm-bb-00001','asuntos-propios',prueba.semana(),prueba.semana(),'','','n-pd-9')\$\$,'PC008')" 'OK PC008' 'laborables sin calendario rechazado'
app "SELECT (d->>'cantidad') FROM prueba.pedir('$B','perm-bb-00002','traslado',prueba.semana(),prueba.semana(),'','','n-pd-10') d" '1' 'naturales sin calendario'
app "SELECT jsonb_array_length(d->'solicitudes') FROM prueba.permisos('$B','n-per-2') d" '1' 'B sólo ve lo suyo'

conteo="SELECT (SELECT count(*) FROM vec_cronos_v1.correccion_solicitud)||'/'||(SELECT count(*) FROM vec_cronos_v1.correccion_actuacion)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_solicitud)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_estado)||'/'||(SELECT count(*) FROM vec_cronos_v1.solicitud_outbox)||'/'||(SELECT count(*) FROM vec_cronos_v1.solicitud_replay)||'/'||(SELECT count(*) FROM vec_cronos_v1.movimientos_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.permisos_acceso)"
antes=$(scalar "$conteo")
comprobar "SELECT '$antes'" '1/1/4/5/4/2/3/2' 'filas únicas: solicitudes, estados, outbox, replays y accesos'
inicio=$(scalar "SELECT pg_postmaster_start_time()")
docker restart "$container" >/dev/null
esperar
[ "$(scalar "SELECT pg_postmaster_start_time()")" != "$inicio" ] || { echo 'FALLO PostgreSQL no se reinició' >&2; exit 1; }
printf 'OK PostgreSQL reiniciado\n'
comprobar "$conteo" "$antes" 'mismas filas tras reiniciar'
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.corregir('$A','corr-olvido-0001',current_date-1,'08:00','n-cor-9') d" "$rc|true" 'mismo recibo de corrección tras reiniciar'
app "SELECT (d->>'cantidad')||'|'||(d->>'replay') FROM prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+4,'','','n-pd-11') d" '4|true' 'replay de permiso tras reiniciar sin duplicar'
comprobar "SELECT count(*) FROM vec_cronos_v1.permiso_solicitud WHERE solicitud_ref='permiso:cronos:solicitud:perm-ap-00001'" 1 'una única solicitud tras reiniciar'
printf 'PG18.4: cronos_v1 000008 verificado con fachadas AD3-70 DE PRUEBA; no acredita la cadena V3 real.\n'
