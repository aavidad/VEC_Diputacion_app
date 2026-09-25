#!/usr/bin/env bash
# Ensayo de CT120 (fiscalización de un expediente modificado tras el
# nombramiento) en PostgreSQL 18.4 desechable sobre la estructura real
# restaurada de la principal (volcado con datos sintéticos), como el ensayo
# de CT115/116: instala AD3-82/83, CT113, CT115 y CT116 y ejecuta sus pruebas
# (que dejan el expediente B modificado y en fiscalización); después CT120:
# ROLLBACK, UP/DOWN/UP, doble aplicación, pruebas funcionales y negativas con
# un doble explícito de la fachada AD3-10, reinicio y DOWN protegido.
# Uso: probar_ct120_fiscalizacion_tras_modificacion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct120-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct120-$$"
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
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
echo '== Cadena previa: AD3-82/83, CT113, CT115, CT116 y sus pruebas'
for f in "$ad3/000082_consumidor_cese_cierre_contratacion_temporal.up.sql" "$ad3/000083_consumidor_modificacion_tras_nombramiento_ct.up.sql" \
         "$ct/000113_publicacion_contratos_bolsa.up.sql" "$ct/000115_cese_y_cierre_expediente.up.sql" "$ct/000116_modificacion_tras_nombramiento.up.sql"; do
  run <"$f"
done
prueba 'CT115 OK' "$pruebas/ct115_ct116_fixture_pg18.sql" "$pruebas/ct115_cese_cierre_expediente.sql"
prueba 'CT116 OK' "$pruebas/ct116_modificacion_tras_nombramiento.sql"

m=$ct/000120_fiscalizacion_tras_modificacion
funciones="SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND (proname LIKE '%ct120' OR proname IN ('preparar_fiscalizacion_v2','confirmar_fiscalizacion_v2'))"
echo '== CT120: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run
[[ $(escalar "$funciones") == 0 ]] || { echo 'FALLO: ROLLBACK dejó funciones' >&2; exit 1; }; ok 'ROLLBACK sin rastro'
run <"$m.up.sql"; [[ $(escalar "$funciones") == 6 ]] || { echo 'FALLO: UP incompleto' >&2; exit 1; }; ok 'UP'
falla_con "$m.up.sql" 'CT120 ya instalada'; ok 'doble UP rechazado'
run <"$m.down.sql"; [[ $(escalar "$funciones") == 0 ]] || { echo 'FALLO: DOWN incompleto' >&2; exit 1; }; ok 'DOWN sin historia'
falla_con "$m.down.sql" 'CT120 no instalada'; ok 'doble DOWN rechazado'
run <"$m.up.sql"; ok 'UP de nuevo'
acl=$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.antecedente_fiscalizacion_modificacion_ct120(jsonb)','EXECUTE') OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.preparar_fiscalizacion_v2(jsonb)','EXECUTE') OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.pronamespace='vec_contratacion_temporal'::regnamespace AND p.proname LIKE '%ct120' AND a.grantee=0)")
[[ $acl == f ]] || { echo 'FALLO: ACL de CT120' >&2; exit 1; }; ok 'ACL con roles reales'

echo '== Fiscalización del expediente modificado (doble explícito de la fachada AD3-10)'
prueba 'CT120 OK' "$pruebas/ct120_fiscalizacion_tras_modificacion.sql"

echo '== Reinicio de PostgreSQL'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
b='expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
tras=$(escalar "SELECT v.fase_clave||'/'||v.estado||'/'||(SELECT r.estado FROM vec_contratacion_temporal.reserva_fiscalizacion r WHERE r.reserva_ref='reserva:ct120:fav') FROM vec_contratacion_temporal.expediente_version_integral v JOIN vec_contratacion_temporal.expediente_integral_actual a USING (expediente_ref,version) WHERE v.expediente_ref='$b'")
[[ $tras == 'nombramiento/en_curso/confirmada' ]] || { echo "FALLO: estado tras reinicio: $tras" >&2; exit 1; }; ok 'historia conservada tras reinicio'
falla_con "$m.down.sql" 'reversión denegada'; ok 'DOWN con historia rechazado'
echo 'CT120 verificada'
