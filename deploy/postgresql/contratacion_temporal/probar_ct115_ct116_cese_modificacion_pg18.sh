#!/usr/bin/env bash
# Ensayo de AD3-82/83 y CT115/116 en PostgreSQL 18.4 desechable sobre la
# estructura real restaurada de la principal (volcado con datos sintéticos).
# Uso: probar_ct115_ct116_cese_modificacion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# Con VEC_CT115_GO=1 publica el puerto solo en 127.0.0.1 y, en lugar de las
# pruebas SQL de transacción, ejecuta la prueba de contrato Go↔SQL.
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct115-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct115-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
red=(--network none)
[[ ${VEC_CT115_GO:-} == 1 ]] && red=(-p 127.0.0.1::5432)
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

# CT113 (publicación a Bolsa del histórico de contratos) es dependencia de
# CT115; mientras no esté en esta rama se indica con CT113_UP.
ct113=${CT113_UP:-$ct/000113_publicacion_contratos_bolsa.up.sql}
[[ -s $ct113 ]] || { echo 'Falta CT113 (CT113_UP)' >&2; exit 2; }
inicial_nucleo=$(escalar "$nucleo"); inicial_origen=$(escalar "$origen")
echo '== CT115 sin AD3-82 se niega'
falla_con "$ct/000115_cese_y_cierre_expediente.up.sql" 'AD3-82 requerida'
echo '== AD3-82 y AD3-83: ROLLBACK, UP, doble UP, DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ad3/000082_consumidor_cese_cierre_contratacion_temporal.up.sql" | run
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'ROLLBACK AD3-82 conserva el núcleo'
for m in 000082_consumidor_cese_cierre_contratacion_temporal 000083_consumidor_modificacion_tras_nombramiento_ct; do
  run <"$ad3/$m.up.sql"; ok "UP $m"
  falla_con "$ad3/$m.up.sql" 'preimagen incompatible'; ok "doble UP rechazado $m"
done
con_ambos=$(escalar "$nucleo")
run <"$ad3/000082_consumidor_cese_cierre_contratacion_temporal.down.sql"; ok 'DOWN AD3-82 fuera de orden'
run <"$ad3/000083_consumidor_modificacion_tras_nombramiento_ct.down.sql"; ok 'DOWN AD3-83'
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'DOWN restaura exactamente el núcleo y las audiencias'
run <"$ad3/000082_consumidor_cese_cierre_contratacion_temporal.up.sql"
run <"$ad3/000083_consumidor_modificacion_tras_nombramiento_ct.up.sql"
igual "$(escalar "$nucleo")" "$con_ambos" 'UP tras DOWN reproduce el núcleo'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" f 'el ejecutor CT no invoca las fachadas AD3'

echo '== CT115 sin CT113 se niega'
falla_con "$ct/000115_cese_y_cierre_expediente.up.sql" 'CT113 exacta requerida'
run <"$ct113"; ok 'CT113 instalada'
echo '== CT115 y CT116: ROLLBACK, UP, doble UP, DOWN, UP'
for m in 000115_cese_y_cierre_expediente 000116_modificacion_tras_nombramiento; do
  [[ -f $ct/$m.up.sql ]] || continue
  antes=$(escalar "$origen"); antes_bolsa=$(escalar "SELECT md5(prosrc) FROM pg_proc WHERE proname='leer_contratos_bolsa_v1'")
  sed 's/^COMMIT;$/ROLLBACK;/' "$ct/$m.up.sql" | run
  igual "$(escalar "$origen")" "$antes" "ROLLBACK $m"
  run <"$ct/$m.up.sql"; ok "UP $m"
  despues=$(escalar "$origen")
  falla_con "$ct/$m.up.sql" 'CT11[56]'; ok "doble UP rechazado $m"
  run <"$ct/$m.down.sql"; igual "$(escalar "$origen")" "$antes" "DOWN $m restaura el origen"
  igual "$(escalar "SELECT md5(prosrc) FROM pg_proc WHERE proname='leer_contratos_bolsa_v1'")" "$antes_bolsa" "DOWN $m restaura la lectura de Bolsa"
  run <"$ct/$m.up.sql"; igual "$(escalar "$origen")" "$despues" "UP $m reproduce"
done
echo '== AD3-82 DOWN se niega con CT115 instalada'
falla_con "$ad3/000082_consumidor_cese_cierre_contratacion_temporal.down.sql" 'DOWN no admitido'

pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
if [[ ${VEC_CT115_GO:-} == 1 ]]; then
  echo '== Contrato Go↔SQL (doble explícito de las fachadas AD3)'
  run <"$pruebas/ct115_ct116_fixture_pg18.sql"
  puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
  a='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
  b='expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
  org=$(escalar "SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='$a' AND version=7")
  (cd "$repo" && VEC_CT115_PG_DSN="postgres://vec_ct115_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT115_ORG="$org" \
    VEC_CT115_EXP_A="$a" VEC_CT115_EXP_B="$b" TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestSeguimientoPostgreSQLContratoGoSQL -v \
    ./internal/modules/contrataciontemporal/adapters/postgres/ 2>&1 | tail -20)
  echo 'ENSAYO GO COMPLETO'
  exit 0
fi
echo '== Transacciones de cese y cierre (doble explícito de la fachada AD3)'
prueba() { # $1 marca final, $2... ficheros en una sola sesión
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}
prueba 'CT115 OK' "$pruebas/ct115_ct116_fixture_pg18.sql" "$pruebas/ct115_cese_cierre_expediente.sql"
echo '== Transacción de modificación (doble explícito de la fachada AD3)'
prueba 'CT116 OK' "$pruebas/ct116_modificacion_tras_nombramiento.sql"
# Inbox del histórico de contratos de Bolsa (B13, Bolsa 000024): el cese
# publicado por CT115 se registra igual que una incorporación, y su reentrega
# se reconoce. Mientras B13 no esté en esta rama se indica con BOLSA24_UP.
b24=${BOLSA24_UP:-$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000024_historico_contratos_participacion.up.sql}
if [[ -s $b24 ]]; then
  echo '== Bolsa recibe el cese en su inbox del histórico de contratos'
  run <"$b24"
  salida=$(docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1 <<'SQL'
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT evento::text AS ev, huella_sha256 AS hu, origen_creada_en AS cr FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100) WHERE evento->>'tipo'='cese' \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SET ROLE vec_bolsa_llamamientos_propietario;
SELECT 'primera:'||reutilizado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(:'ev'::jsonb,:'hu',:'cr');
SELECT 'segunda:'||reutilizado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(:'ev'::jsonb,:'hu',:'cr');
SELECT 'fila:'||tipo||':'||causa_clave||':'||to_char(fin_previsto AT TIME ZONE 'UTC','YYYY-MM-DD') FROM vec_bolsa_llamamientos.contrato_participacion WHERE tipo='cese';
SQL
) || { printf '%s\n' "$salida" >&2; exit 1; }
  grep -q '^primera:false$' <<<"$salida" && grep -q '^segunda:true$' <<<"$salida" && grep -q '^fila:cese:fin_sustitucion:2027-02-15$' <<<"$salida" \
    || { printf 'FALLO inbox de Bolsa:\n%s\n' "$salida" >&2; exit 1; }
  ok 'Bolsa registra el cese una vez y reconoce la reentrega'
fi
echo '== CT115 DOWN se niega con historia'
falla_con "$ct/000115_cese_y_cierre_expediente.down.sql" 'no admitido con historia'
echo 'ENSAYO COMPLETO'
