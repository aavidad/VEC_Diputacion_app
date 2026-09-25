#!/usr/bin/env bash
# Ensayo PG18.4 desechable de cronos_v1 000009 con fachadas AD3-57 de
# PRUEBA: instala 000001–000008, ROLLBACK y COMMIT de 000009, ACL y RLS;
# bandejas de jefatura y RRHH filtradas por el circuito publicado;
# resolución J-A en dos pasos y A en uno, con motivo, versión optimista,
# replay, conflicto de clave, separación de funciones, no resolver lo propio
# ni lo ajeno, asignación futura sin efecto; avisos de resolución final,
# archivo con replay y conflicto; conservación tras reiniciar. No acredita
# MAC, COSE ni gobierno V3. El contenedor se borra al salir.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
container="vec-cronos-000009-${RANDOM}${RANDOM}"
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
for n in 000002_registrar_marcaje_propio 000003_resultado_ejecucion_marcaje 000004_programacion_y_libro_saldo 000005_periodos_teletrabajo_y_cierre_remoto 000006_rls_resultado_ejecucion 000007_saldo_y_fichaje_remoto_empleado 000008_movimientos_y_permisos_empleado; do
 run < "$mig/$n.up.sql" >/dev/null
done
m9="$mig/000009_resolucion_permisos_y_avisos.up.sql"
if run < "$m9" >/dev/null 2>&1; then echo 'FALLO 000009 aceptada sin AD3-57' >&2; exit 1; fi
printf 'OK 000009 rechazada sin consumidores AD3-57\n'
run < "$base_dir/cronos_resolucion_000009_stub_ad3.sql" >/dev/null
ruta_previa=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'")
outbox_previo=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'")
politicas_previas=$(scalar "SELECT count(*) FROM pg_policy")
sed '$s/^COMMIT;$/ROLLBACK;/' "$m9" | run >/dev/null
comprobar "SELECT to_regclass('vec_cronos_v1.permiso_resolutor') IS NULL AND to_regprocedure('vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL" t 'ROLLBACK de 000009 sin objetos'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'" "$ruta_previa" 'ROLLBACK conserva las rutas auditadas'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'" "$outbox_previo" 'ROLLBACK conserva los tipos de outbox'
comprobar "SELECT count(*) FROM pg_policy" "$politicas_previas" 'ROLLBACK conserva las políticas'
run < "$m9" >/dev/null
if run < "$m9" >/dev/null 2>&1; then echo 'FALLO segunda aplicación aceptada' >&2; exit 1; fi
printf 'OK segunda aplicación de 000009 rechazada\n'
run < "$base_dir/cronos_empleado_000007_preparar.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_preparar.sql" >/dev/null
run < "$base_dir/cronos_resolucion_000009_preparar.sql" >/dev/null

firma='(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
for f in consultar_bandeja_permisos_v1 resolver_permiso_v1 consultar_avisos_propio_v1 archivar_aviso_propio_v1; do
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('public','vec_cronos_v1.$f$firma','EXECUTE') AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog'] FROM pg_proc WHERE oid='vec_cronos_v1.$f$firma'::regprocedure" t "ACL y search_path de $f"
done
comprobar "SELECT NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.consumir_resolucion_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.resolutor_competente_v1(text,text)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.estado_permiso_visible_v1()','EXECUTE')" t 'auxiliares sin EXECUTE'
for t in permiso_resolutor permiso_resolutor_retirada permiso_resolucion permiso_aviso permiso_aviso_archivo bandeja_acceso resolucion_replay avisos_acceso; do
 comprobar "SELECT relrowsecurity AND relforcerowsecurity AND NOT has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$t','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege('public','vec_cronos_v1.$t','SELECT') FROM pg_class WHERE oid='vec_cronos_v1.$t'::regclass" t "RLS FORCE y sin DML directo en $t"
done
comprobar "SELECT prueba.espera_error(\$\$DELETE FROM vec_cronos_v1.permiso_resolutor\$\$,'55000')" 'OK 55000' 'circuito publicado inmutable'
comprobar "SELECT vec_cronos_v1.registrar_denegacion_frontera_v1('corr_no_disponible','acceso_denegado','/api/interna/cronos/permisos/resoluciones','POST','') LIKE 'denegacion:cronos:%'" t 'denegación en ruta nueva auditada' login_cronos_auditor_prueba

A=emp_AAAAAAAAAAAAAAAAAAAAAA
B=emp_BBBBBBBBBBBBBBBBBBBBBB
J=emp_JJJJJJJJJJJJJJJJJJJJJJ
# Solicitudes de partida: vacaciones y horas de médico (J-A), asuntos propios
# (A), una de B sin circuito y una de la propia jefatura.
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-va-00001','vacaciones',prueba.semana()+7,prueba.semana()+9,'','','n-p-1') d" 'solicitado' 'A pide vacaciones'
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+1,'','','n-p-2') d" 'solicitado' 'A pide asuntos propios'
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-hm-00001','horas-medico',prueba.semana()+14,prueba.semana()+14,'09:00','10:00','n-p-3') d" 'solicitado' 'A pide horas de médico'
app "SELECT d->>'estado' FROM prueba.pedir('$B','perm-bt-00001','traslado',prueba.semana(),prueba.semana(),'','','n-p-4') d" 'solicitado' 'B pide traslado'
app "SELECT d->>'estado' FROM prueba.pedir('$J','perm-jt-00001','traslado',prueba.semana(),prueba.semana(),'','','n-p-5') d" 'solicitado' 'la jefatura pide traslado'

# Bandejas: cada paso muestra sólo lo que le toca a quien resuelve.
app "SELECT string_agg(e->>'permiso_ref'||'/'||(e->>'empleado_etiqueta'),',' ORDER BY e->>'permiso_ref') FROM prueba.bandeja('J','responsable','n-b-1') d, jsonb_array_elements(d->'pendientes') e" 'permiso:cronos:horas-medico/Persona sintética A,permiso:cronos:vacaciones/Persona sintética A' 'jefatura: J-A de A, sin lo propio ni lo de B'
app "SELECT string_agg(e->>'permiso_ref',',' ORDER BY e->>'permiso_ref') FROM prueba.bandeja('R','administracion','n-b-2') d, jsonb_array_elements(d->'pendientes') e" 'permiso:cronos:asuntos-propios' 'RRHH: sólo el circuito A pendiente'
app "SELECT jsonb_array_length(d->'pendientes') FROM prueba.bandeja('R','responsable','n-b-3') d" '0' 'asignación futura sin efecto'
app "SELECT prueba.espera_error(\$\$SELECT prueba.bandeja('J','responsable','n-b-1')\$\$,'PC003')" 'OK PC003' 'decisión reutilizada rechazada'
app "SELECT prueba.espera_error(\$\$SELECT prueba.bandeja('J','jefatura','n-b-4')\$\$,'PC001')" 'OK PC001' 'paso desconocido rechazado'

# Paso de jefatura (J-A): aprobar deja pendiente de RRHH, sin aviso.
recibo=$(scalar "SELECT prueba.resolver('J','res-va-00001','perm-va-00001','responsable','aprobar','',1,'n-r-1')::text" login_cronos_prueba)
case "$recibo" in *'"estado": "pendiente_administracion"'*) printf 'OK jefatura aprueba: pendiente de RRHH\n';; *) echo "FALLO resolución: $recibo" >&2; exit 1;; esac
rv=$(printf '%s' "$recibo" | sed -E 's/.*"recibo_ref": "([^"]+)".*/\1/')
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay')||'|'||(d->>'version') FROM prueba.resolver('J','res-va-00001','perm-va-00001','responsable','aprobar','',1,'n-r-2') d" "$rv|true|2" 'replay de la resolución con nueva decisión'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-va-00001','perm-va-00001','responsable','aprobar','Otro motivo',1,'n-r-3')\$\$,'PC002')" 'OK PC002' 'misma clave con otro material en conflicto'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-va-00002','perm-va-00001','responsable','denegar','Tarde',1,'n-r-4')\$\$,'PC011')" 'OK PC011' 'versión obsoleta rechazada'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-va-00003','perm-va-00001','administracion','aprobar','',2,'n-r-5')\$\$,'PC012')" 'OK PC012' 'quien resolvió como jefatura no resuelve como RRHH'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-bt-00001','perm-bt-00001','responsable','aprobar','',1,'n-r-6')\$\$,'PC012')" 'OK PC012' 'solicitud sin asignación: no competente'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-jt-00001','perm-jt-00001','responsable','aprobar','',1,'n-r-7')\$\$,'PC012')" 'OK PC012' 'nadie resuelve lo propio'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-no-00001','perm-no-existe','responsable','aprobar','',1,'n-r-8')\$\$,'PC012')" 'OK PC012' 'inexistente responde igual que ajena'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('R','res-ap-00001','perm-ap-00001','administracion','denegar','',1,'n-r-9')\$\$,'PC001')" 'OK PC001' 'denegar sin motivo rechazado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-ap-00002','perm-ap-00001','responsable','aprobar','',1,'n-r-10')\$\$,'PC011')" 'OK PC011' 'circuito A no pasa por jefatura'

# Paso de RRHH.
app "SELECT (d->>'estado')||'|'||(d->>'version') FROM prueba.resolver('R','res-ap-00003','perm-ap-00001','administracion','denegar','Coincide con el cierre del servicio',1,'n-r-11') d" 'denegado|2' 'RRHH deniega asuntos propios con motivo'
app "SELECT (d->>'estado')||'|'||(d->>'version') FROM prueba.resolver('R','res-va-00004','perm-va-00001','administracion','aprobar','',2,'n-r-12') d" 'concedido|3' 'RRHH concede vacaciones tras la jefatura'
app "SELECT (d->>'estado') FROM prueba.resolver('J','res-hm-00001','perm-hm-00001','responsable','aprobar','',1,'n-r-13') d" 'pendiente_administracion' 'jefatura aprueba horas de médico'
app "SELECT (d->>'estado') FROM prueba.resolver('R','res-hm-00002','perm-hm-00001','administracion','aprobar','Justificante en diez días',2,'n-r-14') d" 'concedido' 'RRHH concede horas de médico'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('R','res-hm-00003','perm-hm-00001','administracion','denegar','Otra vez',3,'n-r-15')\$\$,'PC011')" 'OK PC011' 'solicitud ya resuelta no admite otra resolución'
app "SELECT jsonb_array_length(d->'pendientes') FROM prueba.bandeja('R','administracion','n-b-5') d" '0' 'bandeja de RRHH vacía tras resolver'
app "SELECT string_agg((e->>'permiso_ref')||'='||(e->>'estado')||'/'||(e->>'pendiente_justificar'),',' ORDER BY e->>'permiso_ref') FROM prueba.permisos('$A','n-per-1') d, jsonb_array_elements(d->'solicitudes') e" 'permiso:cronos:asuntos-propios=denegado/false,permiso:cronos:horas-medico=concedido/true,permiso:cronos:traslado=concedido/true,permiso:cronos:vacaciones=concedido/false' 'la persona ve el estado resuelto y lo pendiente de justificar'

# Avisos de resolución final y su archivo.
app "SELECT jsonb_array_length(d->'avisos')||'|'||(SELECT string_agg((e->>'estado')||':'||coalesce(e->>'motivo','-'),',' ORDER BY e->>'permiso_ref') FROM jsonb_array_elements(d->'avisos') e) FROM prueba.avisos('$A','n-a-1') d" '3|denegado:Coincide con el cierre del servicio,concedido:Justificante en diez días,concedido:-' 'tres avisos finales con su motivo'
app "SELECT jsonb_array_length(d->'avisos') FROM prueba.avisos('$B','n-a-2') d" '0' 'B no ve avisos ajenos'
aviso=$(scalar "SELECT e->>'aviso_ref' FROM prueba.avisos('$A','n-a-3') d, jsonb_array_elements(d->'avisos') e WHERE e->>'permiso_ref'='permiso:cronos:vacaciones'" login_cronos_prueba)
archivo=$(scalar "SELECT prueba.archivar('$A','arch-va-00001','$aviso','n-x-1')::text" login_cronos_prueba)
case "$archivo" in *'"replay": false'*) printf 'OK aviso archivado\n';; *) echo "FALLO archivo: $archivo" >&2; exit 1;; esac
ra=$(printf '%s' "$archivo" | sed -E 's/.*"recibo_ref": "([^"]+)".*/\1/')
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.archivar('$A','arch-va-00001','$aviso','n-x-2') d" "$ra|true" 'replay del archivo'
app "SELECT prueba.espera_error(\$\$SELECT prueba.archivar('$A','arch-va-00002','$aviso','n-x-3')\$\$,'PC011')" 'OK PC011' 'aviso ya archivado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.archivar('$B','arch-bb-00001','$aviso','n-x-4')\$\$,'PC001')" 'OK PC001' 'aviso ajeno no se archiva'
app "SELECT (e->>'archivado')||'|'||(e->>'archivado_en' IS NOT NULL) FROM prueba.avisos('$A','n-a-4') d, jsonb_array_elements(d->'avisos') e WHERE e->>'aviso_ref'='$aviso'" 'true|true' 'el aviso consta archivado'

# Retirada de la asignación de RRHH: deja de ver y de resolver.
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-va-00002','vacaciones',prueba.semana()+21,prueba.semana()+22,'','','n-p-6') d" 'solicitado' 'A pide más vacaciones'
app "SELECT d->>'estado' FROM prueba.resolver('J','res-va-00010','perm-va-00002','responsable','aprobar','',1,'n-r-16') d" 'pendiente_administracion' 'jefatura aprueba'
run <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.permiso_resolutor_retirada VALUES ('resolutor:cronos:a-administracion-r',clock_timestamp(),'fuente:sintetica:duda-47',clock_timestamp());
COMMIT;
SQL
app "SELECT jsonb_array_length(d->'pendientes') FROM prueba.bandeja('R','administracion','n-b-6') d" '0' 'asignación retirada: bandeja vacía'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('R','res-va-00011','perm-va-00002','administracion','aprobar','',2,'n-r-17')\$\$,'PC012')" 'OK PC012' 'asignación retirada: no resuelve'

conteo="SELECT (SELECT count(*) FROM vec_cronos_v1.permiso_resolucion)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_estado)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_aviso)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_aviso_archivo)||'/'||(SELECT count(*) FROM vec_cronos_v1.solicitud_outbox)||'/'||(SELECT count(*) FROM vec_cronos_v1.resolucion_replay)||'/'||(SELECT count(*) FROM vec_cronos_v1.bandeja_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.avisos_acceso)"
antes=$(scalar "$conteo")
comprobar "SELECT '$antes'" '6/14/3/1/13/1/5/4' 'filas únicas: resoluciones, estados, avisos, archivo, outbox, replays y accesos'
inicio=$(scalar "SELECT pg_postmaster_start_time()")
docker restart "$container" >/dev/null
esperar
[ "$(scalar "SELECT pg_postmaster_start_time()")" != "$inicio" ] || { echo 'FALLO PostgreSQL no se reinició' >&2; exit 1; }
printf 'OK PostgreSQL reiniciado\n'
comprobar "$conteo" "$antes" 'mismas filas tras reiniciar'
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.resolver('J','res-va-00001','perm-va-00001','responsable','aprobar','',1,'n-r-18') d" "$rv|true" 'mismo recibo de resolución tras reiniciar'
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.archivar('$A','arch-va-00001','$aviso','n-x-5') d" "$ra|true" 'mismo recibo de archivo tras reiniciar'
comprobar "SELECT count(*) FROM vec_cronos_v1.permiso_resolucion WHERE resolucion_ref='permiso:cronos:resolucion:res-va-00001'" 1 'una única resolución tras reiniciar'
printf 'PG18.4: cronos_v1 000009 verificado con fachadas AD3-57 DE PRUEBA; no acredita la cadena V3 real.\n'
