#!/usr/bin/env bash
# Ensayo de AD3-87 y CT122 (cancelación del expediente antes de la
# fiscalización) en PostgreSQL 18.4 desechable sobre la estructura real
# restaurada de la principal (volcado con datos sintéticos).
# Uso: probar_ct122_cancelacion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# Antes instala, en orden, las migraciones de la principal que el volcado aún
# no lleva y que admiten esta estructura (AD3-82…86 y CT110…121 salvo CT117,
# que exige una preimagen que el volcado no tiene), para que AD3-87 y CT122
# se prueben sobre el núcleo y la lista de orígenes vigentes.
# Con VEC_CT122_GO=1 publica el puerto solo en 127.0.0.1 y ejecuta además la
# prueba de contrato Go↔SQL. El contenedor usa --rm, sin red ni volúmenes
# anónimos; sus datos viven en /dev/shm/vec-pg-ct122-<pid> y se borran al
# terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct122-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
red=(--network none)
[[ ${VEC_CT122_GO:-} == 1 ]] && red=(-p 127.0.0.1::5432)
docker run -d --rm "${red[@]}" --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
# La imagen arranca un servidor temporal para inicializar y después el
# definitivo: se espera al segundo antes de conectar.
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
nucleo="SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))||md5(pg_get_constraintdef(c.oid)) FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
origen="SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='expediente_version_integral_origen_version_check'"
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3" >&2; exit 1; }; ok "$3"; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}

echo '== Migraciones previas de la principal'
echo '== AD3-87 sin AD3-82 se niega'
falla_con "$ad3/000087_consumidor_cancelacion_expediente_ct.up.sql" 'AD3-82 requerida'
for m in 000082 000083 000084 000085 000086; do run <"$(ls "$ad3"/${m}_*.up.sql)"; done; ok 'AD3-82…86'
for m in 000110 000111 000113 000115 000116 000118 000119 000120 000121; do run <"$(ls "$ct"/${m}_*.up.sql)"; done; ok 'CT110…121 (sin CT117)'

inicial_nucleo=$(escalar "$nucleo"); inicial_origen=$(escalar "$origen")
echo '== CT122 sin AD3-87 se niega'
falla_con "$ct/000122_cancelacion_expediente.up.sql" 'AD3-87 requerida'
echo '== AD3-87: ROLLBACK, UP, doble UP, DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ad3/000087_consumidor_cancelacion_expediente_ct.up.sql" | run
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'ROLLBACK AD3-87 conserva el núcleo'
run <"$ad3/000087_consumidor_cancelacion_expediente_ct.up.sql"; ok 'UP AD3-87'
con_87=$(escalar "$nucleo")
falla_con "$ad3/000087_consumidor_cancelacion_expediente_ct.up.sql" 'preimagen incompatible'; ok 'doble UP rechazado AD3-87'
run <"$ad3/000087_consumidor_cancelacion_expediente_ct.down.sql"; ok 'DOWN AD3-87'
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'DOWN restaura exactamente el núcleo y las audiencias'
falla_con "$ad3/000087_consumidor_cancelacion_expediente_ct.down.sql" 'AD3-87 no instalada'; ok 'doble DOWN rechazado AD3-87'
run <"$ad3/000087_consumidor_cancelacion_expediente_ct.up.sql"
igual "$(escalar "$nucleo")" "$con_87" 'UP tras DOWN reproduce el núcleo'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR has_function_privilege('public','vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" f \
  'la fachada AD3-87 solo la invoca el propietario de CT'

echo '== CT122: ROLLBACK, UP, doble UP, DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ct/000122_cancelacion_expediente.up.sql" | run
igual "$(escalar "$origen")" "$inicial_origen" 'ROLLBACK CT122 conserva el origen'
igual "$(escalar "SELECT to_regclass('vec_contratacion_temporal.cancelacion_expediente_v1') IS NULL")" t 'ROLLBACK CT122 no deja la tabla'
run <"$ct/000122_cancelacion_expediente.up.sql"; ok 'UP CT122'
despues=$(escalar "$origen")
falla_con "$ct/000122_cancelacion_expediente.up.sql" 'CT122'; ok 'doble UP rechazado CT122'
run <"$ct/000122_cancelacion_expediente.down.sql"; igual "$(escalar "$origen")" "$inicial_origen" 'DOWN CT122 restaura el origen'
igual "$(escalar "SELECT count(*) FROM pg_proc WHERE proname LIKE '%ct122%' OR proname LIKE '%cancelacion_expediente_v1'")" 0 'DOWN CT122 no deja funciones'
falla_con "$ct/000122_cancelacion_expediente.down.sql" 'CT122 no instalada'; ok 'doble DOWN rechazado CT122'
run <"$ct/000122_cancelacion_expediente.up.sql"; igual "$(escalar "$origen")" "$despues" 'UP CT122 reproduce'
echo '== AD3-87 DOWN se niega con CT122 instalada'
falla_con "$ad3/000087_consumidor_cancelacion_expediente_ct.down.sql" 'CT 000122 sigue instalada'

# Expedientes sintéticos del volcado en las fases que se prueban.
exp_de() { escalar "SELECT a.expediente_ref FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version) WHERE a.version=$1 AND v.agregado_json->>'fase_actual'='$2' ORDER BY a.expediente_ref OFFSET ${3:-0} LIMIT 1"; }
exp_asignacion=$(exp_de 3 asignacion_unidad); exp_solicitud=$(exp_de 1 solicitud); exp_otra=$(exp_de 1 solicitud 1); exp_fiscalizado=$(exp_de 7 nombramiento)
[[ -n $exp_asignacion && -n $exp_solicitud && -n $exp_otra && -n $exp_fiscalizado ]] || { echo 'Faltan expedientes sintéticos' >&2; exit 2; }

pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
run <"$pruebas/ct122_fixture_pg18.sql"
if [[ ${VEC_CT122_GO:-} == 1 ]]; then
  echo '== Contrato Go↔SQL (doble explícito de la fachada AD3-87)'
  puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
  org=$(escalar "SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='$exp_asignacion' AND version=3")
  (cd "$repo" && VEC_CT122_PG_DSN="postgres://vec_ct122_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT122_ORG="$org" \
    VEC_CT122_EXP_ASIGNACION="$exp_asignacion" VEC_CT122_EXP_SOLICITUD="$exp_solicitud" VEC_CT122_EXP_OTRA="$exp_otra" \
    VEC_CT122_EXP_FISCALIZADO="$exp_fiscalizado" TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestCancelacionPostgreSQLContratoGoSQL -v \
    ./internal/modules/contrataciontemporal/adapters/postgres/ 2>&1 | tail -20) | tee /dev/stderr | grep -q '^ok' || { echo 'FALLO contrato Go↔SQL' >&2; exit 1; }
  echo 'ENSAYO GO COMPLETO'
  exit 0
fi
echo '== Transacciones de cancelación (doble explícito de la fachada AD3-87)'
salida=$(cat "$pruebas/ct122_cancelacion_expediente.sql" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres \
  -v exp_asignacion="$exp_asignacion" -v exp_solicitud="$exp_solicitud" -v exp_fiscalizado="$exp_fiscalizado" 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
grep -q 'CT122 OK' <<<"$salida" || { echo 'FALLO: falta CT122 OK' >&2; exit 1; }
ok "$(grep -c '^ok$' <<<"$salida") comprobaciones de la transacción"
echo '== CT122 DOWN y AD3-87 DOWN se niegan con historia'
falla_con "$ct/000122_cancelacion_expediente.down.sql" 'no admitido con historia'
falla_con "$ad3/000087_consumidor_cancelacion_expediente_ct.down.sql" 'CT 000122 sigue instalada'
echo 'ENSAYO COMPLETO'
