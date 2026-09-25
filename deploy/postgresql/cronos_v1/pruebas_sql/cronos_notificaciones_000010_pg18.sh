#!/usr/bin/env bash
# Ensayo PG18.4 desechable de cronos_v1 000010 con fachadas AD3-57 y AD3-58
# de PRUEBA: instala 000001–000009, rechazo sin AD3-58, ROLLBACK y COMMIT de
# 000010, segunda aplicación rechazada, ACL y RLS.
# Circuito: J-A por defecto aunque el catálogo diga A; A sólo con la marca
# expresa y vigente de la unidad; sin jefatura ni marca, pendiente de
# asignación visible para RRHH y no resoluble; asignación que empieza tarde
# como no competente. consumir_propio_v1 rechaza un contexto de quien
# resuelve ya fijado. Notificaciones: tipo vigente, texto, adjunto por
# referencia y huella, recibo, replay, conflicto de clave, clave de otra
# persona sin oráculo, bandeja de RRHH filtrada por el circuito, atención con
# recibo, replay y conflicto, estado visible para la persona; conservación
# tras reiniciar. Preimagen exacta de lo sustituido (ACL, configuración,
# seguridad y definición retocadas se rechazan). La persona ve en su consulta
# el circuito aplicado y lo pendiente de asignar. Separación de funciones:
# quien dio la conformidad como jefatura no resuelve como RRHH; RRHH con
# asignación sobre sí no ve ni atiende su notificación. Bandeja de más de 500
# rechazada con PC013. No acredita MAC, COSE ni gobierno V3. El contenedor se
# borra al salir.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mig=$(CDPATH= cd -- "$base_dir/../migraciones" && pwd)
datos=$(CDPATH= cd -- "$base_dir/../datos_sinteticos" && pwd)
container="vec-cronos-000010-${RANDOM}${RANDOM}"
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
run < "$base_dir/cronos_resolucion_000009_stub_ad3.sql" >/dev/null
for n in 000002_registrar_marcaje_propio 000003_resultado_ejecucion_marcaje 000004_programacion_y_libro_saldo 000005_periodos_teletrabajo_y_cierre_remoto 000006_rls_resultado_ejecucion 000007_saldo_y_fichaje_remoto_empleado 000008_movimientos_y_permisos_empleado 000009_resolucion_permisos_y_avisos; do
 run < "$mig/$n.up.sql" >/dev/null
done
m10="$mig/000010_circuito_y_notificaciones_rrhh.up.sql"
if run < "$m10" >/dev/null 2>&1; then echo 'FALLO 000010 aceptada sin AD3-58' >&2; exit 1; fi
printf 'OK 000010 rechazada sin consumidores AD3-58\n'
run < "$base_dir/cronos_notificaciones_000010_stub_ad3.sql" >/dev/null
firma='(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
ruta_previa=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'")
outbox_previo=$(scalar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'")
politicas_previas=$(scalar "SELECT count(*) FROM pg_policy")
resolver_previo=$(scalar "SELECT md5(pg_get_functiondef('vec_cronos_v1.resolver_permiso_v1$firma'::regprocedure))")
propio_previo=$(scalar "SELECT md5(pg_get_functiondef('vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")
permisos_previo=$(scalar "SELECT md5(pg_get_functiondef('vec_cronos_v1.consultar_permisos_propio_v1$firma'::regprocedure))")
sed '$s/^COMMIT;$/ROLLBACK;/' "$m10" | run >/dev/null
comprobar "SELECT to_regclass('vec_cronos_v1.notificacion') IS NULL AND to_regclass('vec_cronos_v1.permiso_circuito_directo') IS NULL AND NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_cronos_v1.permiso_resolucion'::regclass AND attname='circuito')" t 'ROLLBACK de 000010 sin objetos'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='denegacion_frontera_ruta_check'" "$ruta_previa" 'ROLLBACK conserva las rutas auditadas'
comprobar "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='solicitud_outbox_tipo_check'" "$outbox_previo" 'ROLLBACK conserva los tipos de outbox'
comprobar "SELECT count(*) FROM pg_policy" "$politicas_previas" 'ROLLBACK conserva las políticas'
comprobar "SELECT md5(pg_get_functiondef('vec_cronos_v1.resolver_permiso_v1$firma'::regprocedure))" "$resolver_previo" 'ROLLBACK conserva la resolución de 000009'
comprobar "SELECT md5(pg_get_functiondef('vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))" "$propio_previo" 'ROLLBACK conserva el consumo propio de 000008'
comprobar "SELECT md5(pg_get_functiondef('vec_cronos_v1.consultar_permisos_propio_v1$firma'::regprocedure))" "$permisos_previo" 'ROLLBACK conserva la consulta de permisos propios de 000008'
# Preimagen exacta: una ACL, una configuración o una seguridad retocadas
# después de 000008/000009 detienen la instalación sin dejar nada.
perturbar() {
 run -c "$1" >/dev/null
 local salida
 if salida=$(run < "$m10" 2>&1); then echo "FALLO 000010 aceptada con la preimagen alterada: $3" >&2; exit 1; fi
 case "$salida" in *'Cronos 000010: preimagen de '*) ;; *) echo "FALLO rechazo por otro motivo ($3): $salida" >&2; exit 1;; esac
 run -c "$2" >/dev/null
 printf 'OK 000010 rechazada con la preimagen alterada: %s\n' "$3"
}
propia_firma='vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
perturbar "GRANT EXECUTE ON FUNCTION vec_cronos_v1.resolver_permiso_v1$firma TO vec_cronos_v1_auditor" "REVOKE EXECUTE ON FUNCTION vec_cronos_v1.resolver_permiso_v1$firma FROM vec_cronos_v1_auditor" 'ACL de resolver_permiso_v1'
perturbar "ALTER FUNCTION vec_cronos_v1.consultar_bandeja_permisos_v1$firma SET lock_timeout='3s'" "ALTER FUNCTION vec_cronos_v1.consultar_bandeja_permisos_v1$firma SET lock_timeout='2s'" 'configuración de consultar_bandeja_permisos_v1'
perturbar "ALTER FUNCTION $propia_firma SECURITY DEFINER" "ALTER FUNCTION $propia_firma SECURITY INVOKER" 'seguridad de consumir_propio_v1'
perturbar "ALTER FUNCTION vec_cronos_v1.consultar_permisos_propio_v1$firma SET work_mem='8MB'" "ALTER FUNCTION vec_cronos_v1.consultar_permisos_propio_v1$firma RESET work_mem" 'configuración de consultar_permisos_propio_v1'
perturbar "ALTER FUNCTION vec_cronos_v1.resolver_permiso_v1$firma COST 101" "ALTER FUNCTION vec_cronos_v1.resolver_permiso_v1$firma COST 100" 'definición de resolver_permiso_v1'
run < "$m10" >/dev/null
if run < "$m10" >/dev/null 2>&1; then echo 'FALLO segunda aplicación aceptada' >&2; exit 1; fi
printf 'OK segunda aplicación de 000010 rechazada\n'
run < "$base_dir/cronos_empleado_000007_preparar.sql" >/dev/null
run < "$base_dir/cronos_empleado_000008_preparar.sql" >/dev/null
run < "$base_dir/cronos_resolucion_000009_preparar.sql" >/dev/null
run < "$datos/tipos_notificacion_sintetico_duda48.sql" >/dev/null
run < "$datos/tipos_notificacion_sintetico_duda48.sql" >/dev/null
run < "$base_dir/cronos_notificaciones_000010_preparar.sql" >/dev/null
comprobar "SELECT count(*) FROM vec_cronos_v1.notificacion_tipo WHERE sintetico AND fuente_ref='fuente:sintetica:a-confirmar-duda-48'" 3 'tipos sintéticos rotulados, sin duplicar al reejecutar'

for f in consultar_bandeja_permisos_v1 resolver_permiso_v1 registrar_notificacion_propia_v1 consultar_notificaciones_propio_v1 consultar_bandeja_notificaciones_v1 atender_notificacion_v1; do
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_auditor','vec_cronos_v1.$f$firma','EXECUTE') AND NOT has_function_privilege('public','vec_cronos_v1.$f$firma','EXECUTE') AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog'] FROM pg_proc WHERE oid='vec_cronos_v1.$f$firma'::regprocedure" t "ACL y search_path de $f"
done
comprobar "SELECT NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.consumir_notificacion_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.circuito_directo_vigente_v1(text,timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.jefatura_asignada_v1(text,timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.notificacion_tipos_vigentes_v1(timestamptz)','EXECUTE') AND NOT has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')" t 'auxiliares sin EXECUTE'
for t in permiso_circuito_directo permiso_circuito_directo_retirada notificacion_tipo notificacion notificacion_atencion notificaciones_acceso notificaciones_bandeja_acceso atencion_replay; do
 comprobar "SELECT relrowsecurity AND relforcerowsecurity AND NOT has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.$t','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege('public','vec_cronos_v1.$t','SELECT') FROM pg_class WHERE oid='vec_cronos_v1.$t'::regclass" t "RLS FORCE y sin DML directo en $t"
done
comprobar "SELECT prueba.espera_error(\$\$DELETE FROM vec_cronos_v1.permiso_circuito_directo\$\$,'55000')" 'OK 55000' 'marca de circuito inmutable'
comprobar "SELECT prueba.espera_error(\$\$UPDATE vec_cronos_v1.notificacion_tipo SET nombre='x'\$\$,'55000')" 'OK 55000' 'catálogo de tipos inmutable'
comprobar "SELECT vec_cronos_v1.registrar_denegacion_frontera_v1('corr_no_disponible','acceso_denegado','/api/interna/cronos/notificaciones/atenciones','POST','') LIKE 'denegacion:cronos:%'" t 'denegación en ruta nueva auditada' login_cronos_auditor_prueba

A=emp_AAAAAAAAAAAAAAAAAAAAAA
B=emp_BBBBBBBBBBBBBBBBBBBBBB
C=emp_CCCCCCCCCCCCCCCCCCCCCC
D=emp_DDDDDDDDDDDDDDDDDDDDDD

# ---- Circuito: J-A por defecto, A sólo con marca, pendiente de asignación ----
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-ap-00001','asuntos-propios',prueba.semana(),prueba.semana()+1,'','','c-p-1') d" 'solicitado' 'A pide asuntos propios (A en el catálogo)'
app "SELECT d->>'estado' FROM prueba.pedir('$C','perm-ct-00001','traslado',prueba.semana(),prueba.semana(),'','','c-p-2') d" 'solicitado' 'C (sin jefatura) pide traslado'
app "SELECT d->>'estado' FROM prueba.pedir('$D','perm-dt-00001','traslado',prueba.semana(),prueba.semana(),'','','c-p-3') d" 'solicitado' 'D (marca directa) pide traslado'
app "SELECT string_agg((e->>'permiso_ref')||'/'||(e->>'circuito')||'/'||(e->>'pendiente_asignacion'),',' ORDER BY e->>'permiso_ref') FROM prueba.bandeja('J','responsable','c-b-1') d, jsonb_array_elements(d->'pendientes') e" 'permiso:cronos:asuntos-propios/J-A/false' 'jefatura: el permiso A del catálogo le llega; D no (marca directa)'
app "SELECT string_agg((e->>'empleado_etiqueta')||'/'||(e->>'circuito')||'/'||(e->>'pendiente_asignacion'),',' ORDER BY e->>'empleado_etiqueta') FROM prueba.bandeja('R','administracion','c-b-2') d, jsonb_array_elements(d->'pendientes') e" 'Persona sintética C/J-A/true,Persona sintética D/A/false' 'RRHH: D directo y C pendiente de asignación; A aún no'
# La persona ve lo mismo en su consulta de permisos, sin saber quién resuelve.
propia() { printf "SELECT string_agg(coalesce(e->>'circuito','-')||'/'||(e->>'estado')||'/'||(e->>'pendiente_asignacion'),',' ORDER BY e->>'solicitud_ref') FROM prueba.permisos('%s','%s') d, jsonb_array_elements(d->'solicitudes') e WHERE e->>'solicitud_ref'='permiso:cronos:solicitud:%s'" "$1" "$2" "$3"; }
app "$(propia "$A" c-pp-1 perm-ap-00001)" 'J-A/solicitado/false' 'A ve su solicitud pendiente de jefatura aunque el catálogo diga A'
app "$(propia "$C" c-pp-2 perm-ct-00001)" 'J-A/solicitado/true' 'C ve su solicitud pendiente de asignar jefatura'
app "$(propia "$D" c-pp-3 perm-dt-00001)" 'A/solicitado/false' 'D ve su solicitud directa a RRHH'
app "SELECT count(*) FROM prueba.permisos('$A','c-pp-4') d, jsonb_array_elements(d->'solicitudes') e WHERE e ?| ARRAY['resolutor_ref','asignacion_ref','empleado_etiqueta','circuito_directo_ref']" 0 'la consulta propia no revela quién resuelve'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('R','res-ap-00001','perm-ap-00001','administracion','aprobar','',1,'c-r-1')\$\$,'PC011')" 'OK PC011' 'el circuito A del catálogo ya no va directo a RRHH'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('R','res-ct-00001','perm-ct-00001','administracion','aprobar','',1,'c-r-2')\$\$,'PC014')" 'OK PC014' 'sin jefatura ni marca: pendiente de asignación, no se resuelve'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-dt-00001','perm-dt-00001','responsable','aprobar','',1,'c-r-3')\$\$,'PC011')" 'OK PC011' 'con marca directa la jefatura no interviene'
app "SELECT (d->>'estado')||'|'||(d->>'version') FROM prueba.resolver('J','res-ap-00002','perm-ap-00001','responsable','aprobar','',1,'c-r-4') d" 'pendiente_administracion|2' 'jefatura da conformidad a asuntos propios'
app "$(propia "$A" c-pp-5 perm-ap-00001)" 'J-A/pendiente_administracion/false' 'A ve su solicitud pendiente de RRHH'
app "SELECT (d->>'estado')||'|'||(d->>'version') FROM prueba.resolver('R','res-ap-00003','perm-ap-00001','administracion','aprobar','',2,'c-r-5') d" 'concedido|3' 'RRHH concede por último'
app "SELECT (d->>'estado')||'|'||(d->>'version') FROM prueba.resolver('R','res-dt-00002','perm-dt-00001','administracion','aprobar','',1,'c-r-6') d" 'concedido|2' 'RRHH concede directo a D por la marca'
comprobar "SELECT string_agg(clave_operacion||'='||coalesce(circuito,'-')||'/'||coalesce(circuito_directo_ref,'-'),',' ORDER BY clave_operacion) FROM vec_cronos_v1.permiso_resolucion" 'res-ap-00002=J-A/-,res-ap-00003=J-A/-,res-dt-00002=A/circuito:cronos:unidad-sin-jefatura:d' 'circuito y marca aplicados constan en la resolución'
app "SELECT jsonb_array_length(d->'avisos') FROM prueba.avisos('$D','c-a-1') d" '1' 'D recibe el aviso de la concesión directa'
app "$(propia "$A" c-pp-6 perm-ap-00001)" 'J-A/concedido/false' 'A ve concedido con el circuito aplicado'
app "$(propia "$D" c-pp-7 perm-dt-00001)" 'A/concedido/false' 'D ve concedido por el circuito directo'
# La competencia se evalúa en el instante del consumo: sin INTO STRICT, una
# asignación que empieza entre ambos da no competente, nunca error interno.
comprobar "SELECT strpos(pg_get_functiondef('vec_cronos_v1.resolver_permiso_v1$firma'::regprocedure),'INTO STRICT asignacion')=0 AND strpos(pg_get_functiondef('vec_cronos_v1.atender_notificacion_v1$firma'::regprocedure),'INTO STRICT asignacion')=0" t 'asignación sin INTO STRICT'
# Una jefatura publicada después saca a C de lo pendiente de asignación.
run <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.permiso_resolutor VALUES ('resolutor:cronos:c-responsable-j','emp_CCCCCCCCCCCCCCCCCCCCCC','Persona sintética C','responsable','per_JJJJJJJJJJJJJJJJJJJJJJ',clock_timestamp()+interval '1 second',NULL,true,'fuente:sintetica:duda-47',clock_timestamp());
COMMIT;
SQL
sleep 1.5
app "SELECT jsonb_array_length(d->'pendientes') FROM prueba.bandeja('R','administracion','c-b-3') d" '0' 'C con jefatura ya no está pendiente de asignación en RRHH'
app "SELECT (d->>'estado') FROM prueba.resolver('J','res-ct-00002','perm-ct-00001','responsable','aprobar','',1,'c-r-7') d" 'pendiente_administracion' 'la jefatura nueva de C da conformidad'
app "$(propia "$C" c-pp-8 perm-ct-00001)" 'J-A/pendiente_administracion/false' 'C con jefatura ya no ve pendiente de asignar'
# ---- consumir_propio_v1 endurecido ----
app "SELECT prueba.espera_error(\$\$WITH b AS MATERIALIZED (SELECT prueba.bandeja('R','administracion','c-h-1') x) SELECT prueba.permisos('$A','c-h-2') FROM b\$\$,'PC003')" 'OK PC003' 'consumo propio rechaza el contexto de quien resuelve ya fijado'
app "SELECT prueba.espera_error(\$\$WITH b AS MATERIALIZED (SELECT prueba.notificaciones('$A','c-h-3') x) SELECT prueba.bandeja_notificaciones('R','c-h-4') FROM b\$\$,'PC003')" 'OK PC003' 'consumo de RRHH rechaza el contexto propio ya fijado'

# ---- Notificaciones ----
TIPO=notificacion:cronos:tipo:incidencia-marcaje:sintetico-1
H=$(printf 'a%.0s' $(seq 1 64))
recibo=$(scalar "SELECT prueba.notificar('$A','not-a-00001','$TIPO',prueba.semana(),E'No pude fichar la salida.\nLo comunico.','','','n-1')::text" login_cronos_prueba)
case "$recibo" in *'"replay": false'*) printf 'OK A registra una notificación sin adjunto\n';; *) echo "FALLO registro: $recibo" >&2; exit 1;; esac
rn=$(printf '%s' "$recibo" | sed -E 's/.*"recibo_ref": "([^"]+)".*/\1/')
na=$(printf '%s' "$recibo" | sed -E 's/.*"notificacion_ref": "([^"]+)".*/\1/')
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay')||'|'||(d->>'notificacion_ref') FROM prueba.notificar('$A','not-a-00001','$TIPO',prueba.semana(),E'No pude fichar la salida.\nLo comunico.','','','n-2') d" "$rn|true|$na" 'replay con nueva decisión: mismo recibo'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00001','$TIPO',prueba.semana(),'Otro texto','','','n-3')\$\$,'PC002')" 'OK PC002' 'misma clave con otro material en conflicto'
app "SELECT (d->>'replay') FROM prueba.notificar('$B','not-a-00001','$TIPO',prueba.semana(),'Clave repetida por otra persona','','','n-4') d" 'false' 'la clave de otra persona no choca ni se revela'
app "SELECT (d->>'replay') FROM prueba.notificar('$C','not-c-00001','notificacion:cronos:tipo:otra-comunicacion:sintetico-1',prueba.semana()+3,'Comunicación con documento','registro:sintetico:0001','$H','n-5') d" 'false' 'C registra con adjunto por referencia y huella'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00002','notificacion:cronos:tipo:caducado:v1',prueba.semana(),'Tipo caducado','','','n-6')\$\$,'PC011')" 'OK PC011' 'tipo no vigente rechazado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00003','$TIPO',prueba.semana(),repeat('x',513),'','','n-7')\$\$,'PC001')" 'OK PC001' 'texto de más de 512 caracteres rechazado'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00004','$TIPO',prueba.semana(),'Con control '||chr(7),'','','n-8')\$\$,'PC001')" 'OK PC001' 'caracteres de control rechazados'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00005','$TIPO',prueba.semana(),'Adjunto a medias','registro:sintetico:0002','','n-9')\$\$,'PC001')" 'OK PC001' 'referencia sin huella rechazada'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00006','$TIPO',prueba.semana(),'Huella nula','registro:sintetico:0003','$(printf '0%.0s' $(seq 1 64))','n-10')\$\$,'PC001')" 'OK PC001' 'huella nula rechazada'
app "SELECT prueba.espera_error(\$\$SELECT prueba.notificar('$A','not-a-00007','$TIPO',prueba.semana(),'   ','','','n-11')\$\$,'PC001')" 'OK PC001' 'texto en blanco rechazado'
app "SELECT (d->>'replay') FROM prueba.notificar('$A','not-a-00008','$TIPO',prueba.semana()+1,repeat('ñ',512),'','','n-12') d" 'false' '512 caracteres multibyte admitidos'

app "SELECT jsonb_array_length(d->'tipos')||'|'||jsonb_array_length(d->'notificaciones')||'|'||(SELECT string_agg(e->>'estado',',') FROM jsonb_array_elements(d->'notificaciones') e) FROM prueba.notificaciones('$A','n-13') d" '3|2|registrada,registrada' 'A ve tres tipos vigentes y sus dos notificaciones registradas'
app "SELECT jsonb_array_length(d->'notificaciones') FROM prueba.notificaciones('$D','n-14') d" '0' 'D no ve notificaciones ajenas'
app "SELECT string_agg((e->>'empleado_etiqueta')||'/'||(e->>'atendida')||'/'||coalesce(e->>'adjunto_ref','-'),',' ORDER BY e->>'empleado_etiqueta',e->>'registrada_en') FROM prueba.bandeja_notificaciones('R','n-15') d, jsonb_array_elements(d->'notificaciones') e" 'Persona sintética A/false/-,Persona sintética A/false/-,Persona sintética C/false/registro:sintetico:0001' 'RRHH ve lo de A y C; no B (sin asignación)'
app "SELECT jsonb_array_length(d->'notificaciones') FROM prueba.bandeja_notificaciones('J','n-16') d" '2' 'J (RRHH sólo de A) ve sólo lo de A'
app "SELECT prueba.espera_error(\$\$SELECT prueba.bandeja_notificaciones('R','n-15')\$\$,'PC003')" 'OK PC003' 'decisión reutilizada rechazada'

# Atención por RRHH.
atencion=$(scalar "SELECT prueba.atender('R','ate-r-00001','$na','n-20')::text" login_cronos_prueba)
case "$atencion" in *'"replay": false'*) printf 'OK RRHH atiende la notificación de A\n';; *) echo "FALLO atención: $atencion" >&2; exit 1;; esac
rt=$(printf '%s' "$atencion" | sed -E 's/.*"recibo_ref": "([^"]+)".*/\1/')
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.atender('R','ate-r-00001','$na','n-21') d" "$rt|true" 'replay de la atención: mismo recibo'
nc=$(scalar "SELECT e->>'notificacion_ref' FROM prueba.notificaciones('$C','n-22') d, jsonb_array_elements(d->'notificaciones') e" login_cronos_prueba)
app "SELECT prueba.espera_error(\$\$SELECT prueba.atender('R','ate-r-00001','$nc','n-23')\$\$,'PC002')" 'OK PC002' 'misma clave para otra notificación en conflicto'
app "SELECT prueba.espera_error(\$\$SELECT prueba.atender('J','ate-j-00001','$na','n-24')\$\$,'PC011')" 'OK PC011' 'ya atendida por otra persona de RRHH'
app "SELECT prueba.espera_error(\$\$SELECT prueba.atender('J','ate-j-00002','$nc','n-25')\$\$,'PC012')" 'OK PC012' 'sin asignación de RRHH sobre C: no competente'
app "SELECT prueba.espera_error(\$\$SELECT prueba.atender('R','ate-r-00002','notificacion:cronos:00000000-0000-4000-8000-000000000000','n-26')\$\$,'PC012')" 'OK PC012' 'inexistente responde igual que ajena'
app "SELECT (e->>'estado')||'|'||(e->>'atendida_en' IS NOT NULL) FROM prueba.notificaciones('$A','n-27') d, jsonb_array_elements(d->'notificaciones') e WHERE e->>'notificacion_ref'='$na'" 'atendida|true' 'A ve su notificación atendida'
app "SELECT string_agg(e->>'atendida',',' ORDER BY e->>'registrada_en') FROM prueba.bandeja_notificaciones('R','n-28') d, jsonb_array_elements(d->'notificaciones') e WHERE e->>'empleado_ref'='$A'" 'true,false' 'la bandeja distingue atendida y pendiente'

conteo="SELECT (SELECT count(*) FROM vec_cronos_v1.notificacion)||'/'||(SELECT count(*) FROM vec_cronos_v1.notificacion_atencion)||'/'||(SELECT count(*) FROM vec_cronos_v1.solicitud_outbox WHERE tipo LIKE 'cronos.notificacion.%')||'/'||(SELECT count(*) FROM vec_cronos_v1.atencion_replay)||'/'||(SELECT count(*) FROM vec_cronos_v1.notificaciones_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.notificaciones_bandeja_acceso)||'/'||(SELECT count(*) FROM vec_cronos_v1.permiso_resolucion)"
antes=$(scalar "$conteo")
comprobar "SELECT '$antes'" '4/1/5/1/4/3/4' 'filas únicas: notificaciones, atención, outbox, replay, accesos y resoluciones'
comprobar "SELECT count(*) FROM vec_cronos_v1.solicitud_outbox WHERE tipo LIKE 'cronos.notificacion.%' AND (carga_json ? 'texto' OR carga_json ? 'adjunto_ref')" 0 'el outbox no lleva texto ni adjunto'
inicio=$(scalar "SELECT pg_postmaster_start_time()")
docker restart "$container" >/dev/null
esperar
[ "$(scalar "SELECT pg_postmaster_start_time()")" != "$inicio" ] || { echo 'FALLO PostgreSQL no se reinició' >&2; exit 1; }
printf 'OK PostgreSQL reiniciado\n'
comprobar "$conteo" "$antes" 'mismas filas tras reiniciar'
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.notificar('$A','not-a-00001','$TIPO',prueba.semana(),E'No pude fichar la salida.\nLo comunico.','','','n-30') d" "$rn|true" 'mismo recibo de notificación tras reiniciar'
app "SELECT (d->>'recibo_ref')||'|'||(d->>'replay') FROM prueba.atender('R','ate-r-00001','$na','n-31') d" "$rt|true" 'mismo recibo de atención tras reiniciar'
comprobar "SELECT count(*) FROM vec_cronos_v1.notificacion WHERE empleado_ref='$A' AND clave_operacion IN ('not-a-00001')" 1 'una única notificación tras reiniciar'

# ---- Separación de funciones y nada propio ----
# (a) Quien dio la conformidad como jefatura no resuelve después como RRHH.
app "SELECT d->>'estado' FROM prueba.pedir('$A','perm-va-00009','vacaciones',prueba.semana()+14,prueba.semana()+15,'','','s-p-1') d" 'solicitado' 'A pide vacaciones'
app "SELECT d->>'estado' FROM prueba.resolver('J','res-va-00009','perm-va-00009','responsable','aprobar','',1,'s-r-1') d" 'pendiente_administracion' 'J da la conformidad como jefatura'
app "SELECT prueba.espera_error(\$\$SELECT prueba.resolver('J','res-va-00010','perm-va-00009','administracion','aprobar','',2,'s-r-2')\$\$,'PC012')" 'OK PC012' 'J (también RRHH de A) no resuelve como RRHH lo que conformó'
app "SELECT d->>'estado' FROM prueba.resolver('R','res-va-00011','perm-va-00009','administracion','aprobar','',2,'s-r-3') d" 'concedido' 'otra persona de RRHH concede'
# (b) y (c) R tiene asignación de administración sobre sí mismo: su
# notificación no llega a su bandeja y no puede atenderla.
R=emp_RRRRRRRRRRRRRRRRRRRRRR
app "SELECT d->>'replay' FROM prueba.notificar('$R','not-r-00001','$TIPO',prueba.semana(),'Comunicación propia de RRHH','','','s-n-1') d" 'false' 'R registra una notificación propia'
nr=$(scalar "SELECT e->>'notificacion_ref' FROM prueba.notificaciones('$R','s-n-2') d, jsonb_array_elements(d->'notificaciones') e" login_cronos_prueba)
app "SELECT count(*) FILTER (WHERE e->>'empleado_ref'='$R')||'/'||count(*) FILTER (WHERE e->>'empleado_ref'='$A') FROM prueba.bandeja_notificaciones('R','s-n-3') d, jsonb_array_elements(d->'notificaciones') e" '0/2' 'R no ve la suya en su bandeja aunque es competente sobre sí'
app "SELECT prueba.espera_error(\$\$SELECT prueba.atender('R','ate-r-00009','$nr','s-n-4')\$\$,'PC012')" 'OK PC012' 'R no atiende su propia notificación'
comprobar "SELECT count(*) FROM vec_cronos_v1.notificacion_atencion WHERE notificacion_ref='$nr'" 0 'la notificación propia sigue sin atender'

# ---- Bandeja de más de 500: rechazo explícito, no recorte silencioso ----
masivas() {
 run <<SQL >/dev/null
INSERT INTO vec_cronos_v1.notificacion(notificacion_ref,clave_operacion,empleado_ref,actor_ref,perfil_ref,tipo_version_ref,fecha_referida,texto,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,registrada_en)
SELECT 'notificacion:cronos:'||gen_random_uuid(),'masiva-'||lpad(i::text,5,'0'),'$D','per_DDDDDDDDDDDDDDDDDDDDDD','prf_PPPPPPPPPPPPPPPPPPPPPP','$TIPO',
  prueba.semana(),'Carga sintética '||i,repeat('b',64),'decision:masiva:'||i,'auditoria:masiva:'||i,md5(i::text)||md5(i::text),
  'recibo:cronos:'||gen_random_uuid(),clock_timestamp() FROM generate_series($1,$2) i;
SQL
}
pendientes_r=$(scalar "SELECT jsonb_array_length(d->'notificaciones') FROM prueba.bandeja_notificaciones('R','s-g-0') d" login_cronos_prueba)
masivas 1 $((500 - pendientes_r))
app "SELECT jsonb_array_length(d->'notificaciones') FROM prueba.bandeja_notificaciones('R','s-g-1') d" '500' 'con 500 la bandeja se muestra entera'
masivas 9001 9001
app "SELECT prueba.espera_error(\$\$SELECT prueba.bandeja_notificaciones('R','s-g-2')\$\$,'PC013')" 'OK PC013' 'con 501 la bandeja se rechaza con PC013'
app "SELECT jsonb_array_length(d->'notificaciones') FROM prueba.bandeja_notificaciones('J','s-g-3') d" '2' 'la bandeja de J (sólo A) no se ve afectada'
printf 'PG18.4: cronos_v1 000010 verificado con fachadas AD3-57/58 DE PRUEBA; no acredita la cadena V3 real.\n'
