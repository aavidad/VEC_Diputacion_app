#!/usr/bin/env bash
# Ensayo de AD3-88 y CT124 (incorporación acreditada) en PostgreSQL 18.4
# desechable sobre la estructura real restaurada de la principal (volcado con
# datos sintéticos). Instala la cadena previa (AD3-82/83, CT113/115/116) si el
# volcado no la trae; comprueba precondiciones, ROLLBACK, UP, doble UP, DOWN
# exacto (núcleo AD3, audiencias, origen de versión y las dos funciones del
# cierre de CT115) y UP otra vez; recorre con dobles explícitos de las fachadas
# AD3 el cese, el cierre rechazado sin GINPIX, la confirmación de GINPIX, el
# cierre con su número y la confirmación del centro; reinicia PostgreSQL,
# repite los materiales (mismo recibo, ninguna fila nueva) y comprueba que los
# DOWN se niegan con historia.
# Uso: probar_ct124_incorporacion_acreditada_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct124-<pid> y se borran al terminar.
# VEC_CT124_CONSERVAR=1 deja el contenedor para depurar (borrarlo a mano).
# Con VEC_CT124_GO=1 publica el puerto solo en 127.0.0.1 y, en lugar de las
# pruebas SQL de transacción, ejecuta la prueba de contrato Go↔SQL.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct124-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
[[ -n ${VEC_CT124_CONSERVAR:-} ]] || trap limpiar EXIT
mkdir -p "$datos"
red=(--network none)
[[ ${VEC_CT124_GO:-} == 1 ]] && red=(-p 127.0.0.1::5432)
docker run -d --rm "${red[@]}" --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3" >&2; exit 1; }; ok "$3"; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() { # $1 marca final, $2... ficheros en una sola sesión
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -8 >&2; exit 1; }
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
echo '== Cadena previa: AD3-82/83, CT113/115/116 (si faltan)'
if [[ $(escalar "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]; then
  run <"$ad3/000082_consumidor_cese_cierre_contratacion_temporal.up.sql"
  run <"$ad3/000083_consumidor_modificacion_tras_nombramiento_ct.up.sql"
fi
if [[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL") == t ]]; then
  run <"$ct/000113_publicacion_contratos_bolsa.up.sql"
  run <"$ct/000115_cese_y_cierre_expediente.up.sql"
  run <"$ct/000116_modificacion_tras_nombramiento.up.sql"
fi

nucleo="SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))||md5(pg_get_constraintdef(c.oid)) FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
cierre="SELECT md5(pg_get_functiondef('vec_contratacion_temporal.preparar_cierre_expediente_v1(jsonb)'::regprocedure))||md5(pg_get_functiondef('vec_contratacion_temporal.confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))||(SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='expediente_version_integral_origen_version_check')||coalesce((SELECT proacl::text FROM pg_proc WHERE oid='vec_contratacion_temporal.confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'')"
inicial_nucleo=$(escalar "$nucleo"); inicial_cierre=$(escalar "$cierre")

echo '== CT124 sin AD3-88 se niega'
falla_con "$ct/000124_incorporacion_acreditada.up.sql" 'AD3-88 requerida'
igual "$(escalar "$cierre")" "$inicial_cierre" 'el rechazo no toca el cierre'
echo '== AD3-88: ROLLBACK, UP, doble UP, DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql" | run
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'ROLLBACK AD3-88 conserva el núcleo'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql"; ok 'UP AD3-88'
con_ad388=$(escalar "$nucleo")
falla_con "$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql" 'preimagen incompatible'; ok 'doble UP AD3-88 rechazado'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.down.sql"
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'DOWN AD3-88 restaura exactamente núcleo y audiencias'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql"
igual "$(escalar "$nucleo")" "$con_ad388" 'UP tras DOWN reproduce el núcleo'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" f 'solo el propietario de CT invoca las fachadas AD3-88'

echo '== CT124: ROLLBACK, UP, doble UP, DOWN exacto, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ct/000124_incorporacion_acreditada.up.sql" | run
igual "$(escalar "$cierre")" "$inicial_cierre" 'ROLLBACK CT124 conserva cierre y origen'
run <"$ct/000124_incorporacion_acreditada.up.sql"; ok 'UP CT124'
con_ct124=$(escalar "$cierre")
[[ $con_ct124 != "$inicial_cierre" ]] || { echo 'FALLO: CT124 no amplió el cierre' >&2; exit 1; }
falla_con "$ct/000124_incorporacion_acreditada.up.sql" 'CT124 ya instalada'; ok 'doble UP CT124 rechazado'
falla_con "$ad3/000088_consumidor_incorporacion_acreditada_ct.down.sql" 'DOWN no admitido'; ok 'DOWN AD3-88 rechazado con CT124 instalada'
run <"$ct/000124_incorporacion_acreditada.down.sql"
igual "$(escalar "$cierre")" "$inicial_cierre" 'DOWN CT124 restaura exactamente las funciones del cierre, su ACL y el origen'
run <"$ct/000124_incorporacion_acreditada.up.sql"
igual "$(escalar "$cierre")" "$con_ct124" 'UP tras DOWN reproduce CT124'
igual "$(escalar "SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal'
  AND p.proname IN ('preparar_confirmacion_ginpix_v1','confirmar_confirmacion_ginpix_v1','consultar_incorporaciones_centro_v1','confirmar_incorporacion_centro_v1','consultar_incorporacion_acreditada_v1')
  AND p.prosecdef AND has_function_privilege('vec_contratacion_temporal_ejecutor',p.oid,'EXECUTE') AND NOT has_function_privilege('public',p.oid,'EXECUTE')")" 5 'cinco fachadas definidoras solo para el ejecutor'

echo '== Transacciones (dobles explícitos de las fachadas AD3)'
prueba 'fixture CT124 OK' "$pruebas/ct115_ct116_fixture_pg18.sql" "$pruebas/ct124_fixture_pg18.sql"
if [[ ${VEC_CT124_GO:-} == 1 ]]; then
  echo '== Contrato Go↔SQL (dobles explícitos de las fachadas AD3)'
  puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
  a='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
  b='expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
  org=$(escalar "SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='$a' AND version=7")
  actor=$(escalar "SELECT (peticion->'configuracion'->'ratificador')::text FROM vec_contratacion_temporal.peticion_centro_revision WHERE version=2 AND centro_ref='centro-520'")
  peticion=$(escalar "SELECT peticion_ref FROM vec_contratacion_temporal.peticion_centro_revision WHERE version=2 AND centro_ref='centro-520'")
  (cd "$repo" && VEC_CT124_PG_DSN="postgres://vec_ct115_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT124_ORG="$org" \
    VEC_CT124_ACTOR="$actor" VEC_CT124_PETICION="$peticion" \
    VEC_CT124_EXP_A="$a" VEC_CT124_EXP_B="$b" TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestIncorporacionAcreditadaPostgreSQLContratoGoSQL -v \
    ./internal/modules/contrataciontemporal/adapters/postgres/ 2>&1 | tail -20)
  echo 'ENSAYO GO COMPLETO'
  exit 0
fi
prueba 'CT124 OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_incorporacion_acreditada.sql"
antes=$(escalar "SELECT (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_ginpix_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.incorporacion_centro_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral)||'/'||(SELECT max(version) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001')")

echo '== Reinicio de PostgreSQL y repetición de los mismos materiales'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
prueba 'CT124 reinicio OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_reinicio.sql"
igual "$(escalar "SELECT (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_ginpix_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.incorporacion_centro_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral)||'/'||(SELECT max(version) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001')")" "$antes" 'tras el reinicio ninguna fila nueva'
falla_con "$ct/000124_incorporacion_acreditada.down.sql" 'no admitido con historia'; ok 'DOWN CT124 rechazado con historia'
echo 'CT124 verificado'
