#!/usr/bin/env bash
# Ensayo de CT123 (informe jurídico nuevo tras subsanar un reparo) en
# PostgreSQL 18.4 desechable sobre la estructura real restaurada de la
# principal (volcado con datos sintéticos, que ya trae CT51, CT92 y CT93):
# ROLLBACK, UP/DOWN/UP, doble aplicación, ACL con roles reales, recorrido
# subsanación → informe nuevo → nueva fiscalización favorable con negativos e
# idempotencia (dobles explícitos de las fachadas AD3), política publicada
# que la base aplica a la nueva fiscalización (sin política, la conducta de
# siempre), DOWN que devuelve exacta la confirmación de CT93, reinicio y DOWN
# protegido por la historia.
# Uso: probar_ct123_informe_nuevo_tras_subsanacion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct123-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct123-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
esperar() {
  for _ in $(seq 1 240); do
    if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then return 0; fi
    sleep 0.5
  done
  return 1
}
esperar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
falla_con() {
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() {
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regprocedure('vec_contratacion_temporal.antecedente_refiscalizacion_v1(jsonb)') IS NOT NULL") == t ]] || { echo 'Restauración incompleta (falta CT93)' >&2; exit 2; }

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
m=$ct/000123_informe_nuevo_tras_subsanacion
funciones="SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND proname IN ('informe_nuevo_admisible_ct123','preparar_informe_juridico_tras_subsanacion_v1','confirmar_informe_juridico_tras_subsanacion_v1','inicio_ronda_informe_nuevo_v1')"
restriccion="SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='reserva_informe_juridico_version_expediente_check'"
politica="SELECT (to_regclass('vec_contratacion_temporal.politica_informe_tras_subsanacion') IS NOT NULL)::int
  + (to_regprocedure('vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)') IS NOT NULL)::int
  + (to_regprocedure('vec_contratacion_temporal.informe_nuevo_exigido_ct123()') IS NOT NULL)::int"
confirmacion="SELECT md5(pg_get_functiondef(p.oid))||'/'||coalesce(p.proacl::text,'')||'/'||p.proowner::regrole||'/'||coalesce(p.proconfig::text,'')
  FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure"
con_comprobacion="SELECT strpos(prosrc,'informe_nuevo_exigido_ct123') > 0 FROM pg_proc WHERE oid='vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure"
confirmacion_ct93=$(escalar "$confirmacion")
echo '== CT123: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run
[[ $(escalar "$funciones") == 0 && $(escalar "$restriccion") == 'CHECK ((version_expediente = (4)::numeric))' && $(escalar "$politica") == 0
   && $(escalar "$confirmacion") == "$confirmacion_ct93" ]] || { echo 'FALLO: ROLLBACK dejó rastro' >&2; exit 1; }; ok 'ROLLBACK sin rastro'
run <"$m.up.sql"; [[ $(escalar "$funciones") == 4 && $(escalar "$politica") == 3 && $(escalar "$con_comprobacion") == t ]] || { echo 'FALLO: UP incompleto' >&2; exit 1; }; ok 'UP (con la comprobación en la confirmación de CT93)'
falla_con "$m.up.sql" 'CT123 ya instalada'; ok 'doble UP rechazado'
run <"$m.down.sql"; [[ $(escalar "$funciones") == 0 && $(escalar "$restriccion") == 'CHECK ((version_expediente = (4)::numeric))' && $(escalar "$politica") == 0 ]] || { echo 'FALLO: DOWN incompleto' >&2; exit 1; }; ok 'DOWN sin historia'
[[ $(escalar "$confirmacion") == "$confirmacion_ct93" ]] || { echo 'FALLO: DOWN no devolvió exacta la confirmación de CT93' >&2; exit 1; }; ok 'DOWN devuelve exacta la confirmación de CT93 (cuerpo, ACL, propietario y configuración)'
falla_con "$m.down.sql" 'CT123 no instalada'; ok 'doble DOWN rechazado'
run <"$m.up.sql"; ok 'UP de nuevo'
acl=$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.informe_nuevo_admisible_ct123(jsonb)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text)','EXECUTE')
  OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.pronamespace='vec_contratacion_temporal'::regnamespace
     AND p.proname IN ('informe_nuevo_admisible_ct123','preparar_informe_juridico_tras_subsanacion_v1','confirmar_informe_juridico_tras_subsanacion_v1','inicio_ronda_informe_nuevo_v1') AND a.grantee=0)")
[[ $acl == f ]] || { echo 'FALLO: ACL de CT123' >&2; exit 1; }
[[ $(escalar "SELECT has_function_privilege('vec_contratacion_temporal_gobernador','vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)','EXECUTE')
  AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)','EXECUTE')
  AND NOT has_function_privilege('public','vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)','EXECUTE')
  AND NOT has_function_privilege('public','vec_contratacion_temporal.informe_nuevo_exigido_ct123()','EXECUTE')
  AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.informe_nuevo_exigido_ct123()','EXECUTE')
  AND NOT has_table_privilege('vec_contratacion_temporal_gobernador','vec_contratacion_temporal.politica_informe_tras_subsanacion','SELECT')
  AND NOT has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.politica_informe_tras_subsanacion','SELECT')") == t ]] \
  || { echo 'FALLO: ACL de la política de CT123' >&2; exit 1; }
ok 'ACL con roles reales'

echo '== Recorrido: subsanación, informe nuevo y nueva fiscalización (dobles explícitos AD3)'
prueba 'CT123 OK' "$pruebas/ct123_informe_nuevo_tras_subsanacion.sql"

echo '== Reinicio de PostgreSQL'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
c='expediente:ct:9511d16dce57e0ebe1baa849a749f3eed2a29ff7722dc1499132d1d536d66253'
tras=$(escalar "SELECT v.version||'/'||v.fase_clave||'/'||(v.agregado_json#>>'{fiscalizacion,informe_juridico_ref}')||'/'||(SELECT r.estado FROM vec_contratacion_temporal.reserva_informe_juridico r WHERE r.reserva_ref='reserva:ct123:nuevo') FROM vec_contratacion_temporal.expediente_version_integral v JOIN vec_contratacion_temporal.expediente_integral_actual a USING (expediente_ref,version) WHERE v.expediente_ref='$c'")
[[ $tras == '9/fiscalizacion/informe:ct123:nuevo/confirmada' ]] || { echo "FALLO: estado tras reinicio: $tras" >&2; exit 1; }
[[ $(escalar "SELECT string_agg(version||':'||exige_informe_nuevo,',' ORDER BY version) FROM vec_contratacion_temporal.politica_informe_tras_subsanacion") == '1:true,2:false' ]] \
  || { echo 'FALLO: historia de la política tras reinicio' >&2; exit 1; }
ok 'historia conservada tras reinicio'
falla_con "$m.down.sql" 'reversión denegada, hay política de informe nuevo publicada'; ok 'DOWN con historia rechazado'
echo 'CT123 verificada'
