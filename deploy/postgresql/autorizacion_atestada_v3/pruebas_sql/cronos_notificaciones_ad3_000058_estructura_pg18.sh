#!/usr/bin/env bash
# AD3-58 sobre la PREIMAGEN SINTÉTICA de AD3-51/52 (stub del núcleo) con
# AD3-53, AD3-70 y AD3-57 reales: rechazo sin AD3-57, ROLLBACK, COMMIT,
# anclas, audiencias, ACL de las cuatro fachadas, contratos previos intactos,
# rechazo de una segunda aplicación y conservación tras reiniciar. No
# acredita la cadena AD3 real ni un consumo válido.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-ad3-58-${RANDOM}${RANDOM}"
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
scalar() { docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
comprobar() {
 local obtenido
 obtenido=$(scalar "$1")
 if [ "$obtenido" != "$2" ]; then printf 'FALLO %s: %s (esperado %s)\n' "$3" "$obtenido" "$2" >&2; exit 1; fi
 printf 'OK %s\n' "$3"
}
[ "$(scalar 'SHOW server_version')" = 18.4 ] || { echo 'Se requiere PostgreSQL 18.4' >&2; exit 2; }
mig="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones"
m58="$mig/000058_consumidores_cronos_notificaciones.up.sql"
run < "$base_dir/organizacion_historica_ad3_000051_stub.sql" >/dev/null
run <<'SQL' >/dev/null
CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cronos_prueba_ajeno NOLOGIN;
SQL
run < "$mig/000051_consumidor_organizacion_historica.up.sql" >/dev/null
run < "$mig/000052_consumidor_importacion_organizacion.up.sql" >/dev/null
run < "$mig/000053_consumidores_cronos_empleado.up.sql" >/dev/null
run < "$mig/000070_consumidores_cronos_movimientos_permisos.up.sql" >/dev/null
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
if run < "$m58" >/dev/null 2>&1; then echo 'FALLO AD3-58 aceptada sin AD3-57' >&2; exit 1; fi
printf 'OK AD3-58 rechazada sin AD3-57\n'
run < "$mig/000057_consumidores_cronos_resolucion_avisos.up.sql" >/dev/null
previa=$(scalar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))")
restriccion_previa=$(scalar "SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'")

sed '$s/^COMMIT;$/ROLLBACK;/' "$m58" | run >/dev/null
comprobar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))" "$previa" 'ROLLBACK conserva el núcleo'
comprobar "SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'" "$restriccion_previa" 'ROLLBACK conserva audiencias'
comprobar "SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname LIKE '%cronos%notificacion%'" 0 'ROLLBACK sin fachadas'

run < "$m58" >/dev/null
contar() { printf "SELECT (length(d)-length(replace(d,%s,'')))/length(%s) FROM pg_get_functiondef('%s'::regprocedure) d" "$1" "$1" "$nucleo"; }
for perfil in cronos_marcaje_propio cronos_saldo_propio cronos_movimientos_propio cronos_correccion_solicitar cronos_permisos_propio cronos_permiso_solicitar cronos_permisos_bandeja cronos_permiso_resolver cronos_avisos_propio cronos_aviso_archivar cronos_notificacion_registrar cronos_notificaciones_propio cronos_notificaciones_bandeja cronos_notificacion_atender; do
 comprobar "$(contar "'IS DISTINCT FROM ''$perfil'''")" 1 "exclusión única de $perfil"
 comprobar "$(contar "'p_perfil_mutacion=''$perfil'''")" 1 "contrato único de $perfil"
done
comprobar "$(contar "'''cronos_notificacion_atender'')'")" 2 'guarda de sesión y contrato con la lista ampliada'
comprobar "$(contar "'''cronos_aviso_archivar'')'")" 0 'lista previa sustituida en ambos sitios'
for audiencia in saldo_propio.consultar.v1 permiso.resolver.v1 aviso_propio.archivar.v1 notificacion_propia.registrar.v1 notificaciones_propio.consultar.v1 notificaciones_bandeja.consultar.v1 notificacion.atender.v1; do
 comprobar "SELECT strpos(pg_get_constraintdef(oid),'vec_cronos_v1.$audiencia')>0 FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'" t "audiencia $audiencia"
done
for fachada in registrar_y_consumir_cronos_notificacion_v3_atestada consumir_cronos_notificaciones_propio_v3_atestada consumir_cronos_notificaciones_bandeja_v3_atestada registrar_y_consumir_cronos_atencion_notificacion_v3_atestada registrar_y_consumir_cronos_archivo_aviso_v3_atestada; do
 firma="vec_autorizacion_atestada_v3.$fachada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
 comprobar "SELECT has_function_privilege('vec_cronos_v1_propietario','$firma','EXECUTE')" t "EXECUTE propietario Cronos en $fachada"
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','$firma','EXECUTE') OR has_function_privilege('vec_cronos_prueba_ajeno','$firma','EXECUTE')" f "sin EXECUTE ajeno en $fachada"
 comprobar "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='$firma'::regprocedure AND a.grantee NOT IN (p.proowner,'vec_cronos_v1_propietario'::regrole)" 0 "ACL cerrada en $fachada"
 comprobar "SELECT prosecdef AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole FROM pg_proc WHERE oid='$firma'::regprocedure" t "definidora nominal $fachada"
done
interna="vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(text,text,text,text,text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
comprobar "SELECT has_function_privilege('vec_cronos_v1_propietario','$interna','EXECUTE') OR has_function_privilege('vec_cronos_v1_ejecutor','$interna','EXECUTE')" f 'auxiliar interna sin EXECUTE de Cronos'
if run < "$m58" >/dev/null 2>&1; then
 echo 'FALLO segunda aplicación aceptada' >&2; exit 1
fi
printf 'OK segunda aplicación rechazada por precondición\n'
posterior=$(scalar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))")
inicio=$(scalar "SELECT pg_postmaster_start_time()")
docker restart "$container" >/dev/null
esperar
[ "$(scalar "SELECT pg_postmaster_start_time()")" != "$inicio" ] || { echo 'FALLO PostgreSQL no se reinició' >&2; exit 1; }
comprobar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))" "$posterior" 'núcleo ampliado conservado tras reiniciar'
printf 'PG18.4: AD3-58 ROLLBACK/COMMIT, ACL y reinicio sobre PREIMAGEN SINTÉTICA con AD3-53/70/57; NO acredita la cadena AD3 real.\n'
