#!/usr/bin/env bash
# Ensayo de CT113 (publicación a Bolsa del histórico de contratos) y Bolsa
# 000024 (inbox) en PostgreSQL 18.4 desechable sobre la estructura real
# restaurada (volcado con datos sintéticos, con al menos dos propuestas CT61).
# Uso: probar_publicacion_contratos_bolsa_pg18.sh GLOBALES_SQL VOLCADO_PG_DUMP
# Comprueba ROLLBACK, UP, doble UP, DOWN, doble DOWN y UP de CT113; la prueba
# b13 (proyección, inbox idempotente, negativos y ACL) y la marca de agua con
# dos transacciones concurrentes: una incorporación que confirma tarde con un
# instante anterior no queda nunca por detrás del cursor. El contenedor usa
# --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct113-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
nombre="vec-pg-ct113-$$"
datos="/dev/shm/$nombre"
trabajo=$(mktemp -d /dev/shm/vec-ct113-XXXXXX)
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then docker run --rm -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true; fi
  rmdir "$datos" 2>/dev/null || true
  rm -rf "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" -v "$repo:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3: obtenido «$1», esperado «$2»" >&2; exit 1; }; ok "$3"; }
rechaza() { if run -f "/repo/$1" >/dev/null 2>&1; then echo "FALLO: se aceptó $1" >&2; exit 1; fi; ok "rechazo de $(basename "$1")"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT count(*) >= 2 FROM vec_contratacion_temporal.propuesta_formalizacion") == t ]] || { echo 'Restauración sin dos propuestas CT61' >&2; exit 2; }

ct=deploy/postgresql/contratacion_temporal/migraciones/000113_publicacion_contratos_bolsa
lectura="vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)"
echo '== CT113: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$ct.up.sql" | run
igual "$(escalar "SELECT to_regprocedure('$lectura') IS NULL AND NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.incorporacion_outbox_v2'::regclass AND attname='transaccion_publicacion' AND NOT attisdropped)")" t 'ROLLBACK no deja objetos'
run -f "/repo/$ct.up.sql"; ok 'UP CT113'
rechaza "$ct.up.sql"
run -f "/repo/$ct.down.sql"; ok 'DOWN CT113'
rechaza "$ct.down.sql"
run -f "/repo/$ct.up.sql"; ok 'UP CT113 otra vez'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','$lectura','EXECUTE') AND NOT has_function_privilege('public','$lectura','EXECUTE') AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.posicion_contrato_bolsa_v1(xid8)','EXECUTE')")" t 'ACL de la lectura'

if [[ $(escalar "SELECT to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL") == t ]]; then
  run -f /repo/deploy/postgresql/bolsa_llamamientos/migraciones/000024_historico_contratos_participacion.up.sql; ok 'Bolsa 000024 instalada'
fi
salida=$(docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f /repo/deploy/postgresql/bolsa_llamamientos/pruebas_sql/b13_historico_contratos.sql 2>&1) || { echo "$salida" >&2; exit 1; }
grep -q 'B13: OK' <<<"$salida" || { echo "$salida" >&2; exit 1; }
ok 'b13: proyección, inbox idempotente, negativos y ACL'

echo '== Marca de agua con dos transacciones concurrentes'
fixture=/repo/deploy/postgresql/contratacion_temporal/pruebas_sql/ct113_incorporacion_sintetica.sql
ref_a=$(escalar "SELECT 'ref:outbox:' || encode(sha256(convert_to('ct113:' || expediente_ref, 'UTF8')), 'hex') FROM vec_contratacion_temporal.propuesta_formalizacion ORDER BY expediente_ref LIMIT 1")
ref_b=$(escalar "SELECT 'ref:outbox:' || encode(sha256(convert_to('ct113:' || expediente_ref, 'UTF8')), 'hex') FROM vec_contratacion_temporal.propuesta_formalizacion ORDER BY expediente_ref OFFSET 1 LIMIT 1")
contar() { # cursor posición/ref o NULL
  docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres \
    -c "SET ROLE vec_contratacion_temporal_ejecutor" \
    -c "SELECT coalesce(string_agg(origen_ref, ',' ORDER BY origen_posicion, origen_ref), '') FROM vec_contratacion_temporal.leer_contratos_bolsa_v1($1,$2,100) WHERE origen_ref IN ('$ref_a','$ref_b')" 2>&1 | tail -1
}
# A escribe primero (instante anterior) y confirma tarde; B escribe después y
# confirma antes.
{ echo 'BEGIN;'; echo '\set n 1'; echo "\\i $fixture"; echo "SELECT pg_sleep(6);"; echo 'COMMIT;'; } \
  | docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres >"$trabajo/a.log" 2>&1 &
pa=$!
sleep 1.5
{ echo 'BEGIN;'; echo '\set n 2'; echo "\\i $fixture"; echo 'COMMIT;'; } | run
igual "$(escalar "SELECT (SELECT creada_en FROM vec_contratacion_temporal.incorporacion_outbox_v2 WHERE outbox_ref='$ref_b') IS NOT NULL")" t 'B confirmada mientras A sigue abierta'
# Con la lectura anterior (por creada_en) B saldría ya y el cursor quedaría
# por delante de A. Con la marca de agua B espera a que A termine.
igual "$(contar NULL NULL)" '' 'B no se publica mientras A (anterior) sigue abierta'
wait "$pa" || { cat "$trabajo/a.log" >&2; exit 1; }
igual "$(contar NULL NULL)" "$ref_a,$ref_b" 'terminada A, se publican A y B en orden de posición'
pos_a=$(escalar "SET ROLE vec_contratacion_temporal_ejecutor; SELECT origen_posicion FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100) WHERE origen_ref='$ref_a'" | tail -1)
pos_b=$(escalar "SET ROLE vec_contratacion_temporal_ejecutor; SELECT origen_posicion FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100) WHERE origen_ref='$ref_b'" | tail -1)
igual "$(escalar "SELECT $pos_a < $pos_b AND (SELECT creada_en FROM vec_contratacion_temporal.incorporacion_outbox_v2 WHERE outbox_ref='$ref_a') < (SELECT creada_en FROM vec_contratacion_temporal.incorporacion_outbox_v2 WHERE outbox_ref='$ref_b')")" t 'A tiene instante y posición anteriores a B'
igual "$(contar "$pos_a" "'$ref_a'")" "$ref_b" 'desde el cursor de A solo queda B'
igual "$(contar "$pos_b" "'$ref_b'")" '' 'desde el cursor de B no queda nada'
echo 'OK CT113: marca de agua, cursor por posición y b13'
