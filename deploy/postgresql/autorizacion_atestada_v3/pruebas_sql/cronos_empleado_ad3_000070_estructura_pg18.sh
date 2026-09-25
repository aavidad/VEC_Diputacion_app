#!/usr/bin/env bash
# AD3-70 sobre la PREIMAGEN SINTÉTICA de AD3-51/52 (stub del núcleo) con
# AD3-53 real: ROLLBACK, COMMIT, anclas, audiencias, ACL de las cuatro
# fachadas, rechazo sin AD3-53 y de una segunda aplicación. No acredita la
# cadena AD3 real ni un consumo válido.
set -euo pipefail
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../../.." && pwd)
container="vec-ad3-70-${RANDOM}${RANDOM}"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --rm --network none --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
  sleep 0.3
  if docker exec "$container" psql -X -qAt -U postgres -d postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
 fi
 sleep 0.3
done
run() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
comprobar() {
 local obtenido
 obtenido=$(scalar "$1")
 if [ "$obtenido" != "$2" ]; then printf 'FALLO %s: %s (esperado %s)\n' "$3" "$obtenido" "$2" >&2; exit 1; fi
 printf 'OK %s\n' "$3"
}
mig="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones"
m70="$mig/000070_consumidores_cronos_movimientos_permisos.up.sql"
run < "$base_dir/organizacion_historica_ad3_000051_stub.sql" >/dev/null
run <<'SQL' >/dev/null
CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cronos_prueba_ajeno NOLOGIN;
SQL
run < "$mig/000051_consumidor_organizacion_historica.up.sql" >/dev/null
run < "$mig/000052_consumidor_importacion_organizacion.up.sql" >/dev/null
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
if run < "$m70" >/dev/null 2>&1; then echo 'FALLO AD3-70 aceptada sin AD3-53' >&2; exit 1; fi
printf 'OK AD3-70 rechazada sin AD3-53\n'
run < "$mig/000053_consumidores_cronos_empleado.up.sql" >/dev/null
previa=$(scalar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))")
restriccion_previa=$(scalar "SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'")

sed '$s/^COMMIT;$/ROLLBACK;/' "$m70" | run >/dev/null
comprobar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))" "$previa" 'ROLLBACK conserva el núcleo'
comprobar "SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'" "$restriccion_previa" 'ROLLBACK conserva audiencias'
comprobar "SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND (p.proname LIKE '%cronos_movimientos%' OR p.proname LIKE '%cronos_permiso%' OR p.proname LIKE '%cronos_correccion%' OR p.proname LIKE '%cronos_nominal%')" 0 'ROLLBACK sin fachadas'

run < "$m70" >/dev/null
for perfil in cronos_marcaje_propio cronos_saldo_propio cronos_movimientos_propio cronos_correccion_solicitar cronos_permisos_propio cronos_permiso_solicitar; do
 comprobar "SELECT (length(pg_get_functiondef('$nucleo'::regprocedure))-length(replace(pg_get_functiondef('$nucleo'::regprocedure),'IS DISTINCT FROM ''$perfil''','')))/length('IS DISTINCT FROM ''$perfil''')" 1 "exclusión única de $perfil"
 comprobar "SELECT (length(pg_get_functiondef('$nucleo'::regprocedure))-length(replace(pg_get_functiondef('$nucleo'::regprocedure),'p_perfil_mutacion=''$perfil''','')))/length('p_perfil_mutacion=''$perfil''')" 1 "contrato único de $perfil"
done
comprobar "SELECT (length(d)-length(replace(d,'''cronos_permiso_solicitar'')','')))/length('''cronos_permiso_solicitar'')') FROM pg_get_functiondef('$nucleo'::regprocedure) d" 2 'guarda de sesión y contrato con la lista ampliada'
for audiencia in saldo_propio.consultar.v1 movimientos_propio.consultar.v1 correccion_propia.solicitar.v1 permisos_propio.consultar.v1 permiso_propio.solicitar.v1; do
 comprobar "SELECT strpos(pg_get_constraintdef(oid),'vec_cronos_v1.$audiencia')>0 FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'" t "audiencia $audiencia"
done
for fachada in consumir_cronos_movimientos_propio_v3_atestada registrar_y_consumir_cronos_correccion_v3_atestada consumir_cronos_permisos_propio_v3_atestada registrar_y_consumir_cronos_permiso_v3_atestada; do
 firma="vec_autorizacion_atestada_v3.$fachada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
 comprobar "SELECT has_function_privilege('vec_cronos_v1_propietario','$firma','EXECUTE')" t "EXECUTE propietario Cronos en $fachada"
 comprobar "SELECT has_function_privilege('vec_cronos_v1_ejecutor','$firma','EXECUTE') OR has_function_privilege('vec_cronos_prueba_ajeno','$firma','EXECUTE')" f "sin EXECUTE ajeno en $fachada"
 comprobar "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='$firma'::regprocedure AND a.grantee NOT IN (p.proowner,'vec_cronos_v1_propietario'::regrole)" 0 "ACL cerrada en $fachada"
 comprobar "SELECT prosecdef AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole FROM pg_proc WHERE oid='$firma'::regprocedure" t "definidora nominal $fachada"
done
interna="vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(text,text,text,text,text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
comprobar "SELECT has_function_privilege('vec_cronos_v1_propietario','$interna','EXECUTE') OR has_function_privilege('vec_cronos_v1_ejecutor','$interna','EXECUTE')" f 'auxiliar interna sin EXECUTE de Cronos'
if run < "$m70" >/dev/null 2>&1; then
 echo 'FALLO segunda aplicación aceptada' >&2; exit 1
fi
printf 'OK segunda aplicación rechazada por precondición\n'
printf 'PG18.4: AD3-70 ROLLBACK/COMMIT y ACL sobre PREIMAGEN SINTÉTICA con AD3-53; NO acredita la cadena AD3 real.\n'
