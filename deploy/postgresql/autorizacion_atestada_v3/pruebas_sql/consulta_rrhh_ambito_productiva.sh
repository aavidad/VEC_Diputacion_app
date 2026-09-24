#!/usr/bin/env bash
# Ejecutar solo sobre un contenedor PostgreSQL 18 efímero con --network none.
set -euo pipefail

contenedor=${1:?indicar contenedor PostgreSQL 18 efimero}
if [[ ${VEC_AD351_EFIMERA:-} != si ]]; then
  echo 'AD3-51: se exige VEC_AD351_EFIMERA=si' >&2
  exit 2
fi
imagen=$(docker inspect --format '{{.Config.Image}}' "$contenedor")
red=$(docker inspect --format '{{.HostConfig.NetworkMode}}' "$contenedor")
if [[ $imagen != postgres:18* || $red != none ]]; then
  echo 'AD3-51: el contenedor debe ser PostgreSQL 18 sin red' >&2
  exit 2
fi

raiz=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$raiz/consulta_rrhh_ambito_productiva.sql"

# Estas concesiones existen únicamente en la base efímera; permiten invocar
# las dos funciones internas y distinguir su guarda nominal del permiso SQL.
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
CREATE ROLE vec_ad351_nominal LOGIN INHERIT;
CREATE ROLE vec_ad351_legacy LOGIN INHERIT;
CREATE ROLE vec_ad351_doble LOGIN INHERIT;
CREATE ROLE vec_ad351_sin_rol LOGIN INHERIT;
GRANT vec_contratacion_temporal_consultor_rrhh_ambito TO vec_ad351_nominal
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_ad351_legacy
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh_ambito TO vec_ad351_doble
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_ad351_doble
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO
  vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO
  vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
GRANT EXECUTE ON FUNCTION
  vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
  TO vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
GRANT EXECUTE ON FUNCTION
  vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
  TO vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
SQL

limpiar() {
  docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
DROP OWNED BY vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
DROP ROLE vec_ad351_nominal,vec_ad351_legacy,vec_ad351_doble,vec_ad351_sin_rol;
SQL
}
trap limpiar EXIT

for caso in vec_ad351_nominal:22023 vec_ad351_legacy:22023 \
            vec_ad351_doble:42501 vec_ad351_sin_rol:42501; do
  usuario=${caso%%:*}
  esperado=${caso##*:}
  docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U "$usuario" \
    -d postgres -v esperado="$esperado" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='10s';
SET LOCAL idle_in_transaction_session_timeout='10s';
SELECT set_config('app.expected', :'esperado', true);
DO $prueba$
DECLARE codigo text; esperado text := current_setting('app.expected');
BEGIN
  BEGIN
    PERFORM * FROM vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(
      'cuadro',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
    RAISE EXCEPTION 'consumo aceptó entrada nula';
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS codigo = RETURNED_SQLSTATE; END;
  IF codigo <> esperado THEN RAISE EXCEPTION 'consumo %, esperado %',codigo,esperado; END IF;
  BEGIN
    PERFORM * FROM vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
      'cuadro',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
    RAISE EXCEPTION 'revalidación aceptó entrada nula';
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS codigo = RETURNED_SQLSTATE; END;
  IF codigo <> esperado THEN RAISE EXCEPTION 'revalidación %, esperado %',codigo,esperado; END IF;
END $prueba$;
ROLLBACK;
SQL
done
echo 'AD3-51: cuatro identidades y las dos funciones verificadas en PG18 efímero'
limpiar
trap - EXIT

# El dump schema-only omite filas de control. Se restituye únicamente la
# génesis vacía para probar el DOWN protegido, nunca un consumo de negocio.
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.control_cadena_auditoria
  (control_id,secuencia,cabeza_sha256,actualizada_en)
SELECT true,0,repeat('0',64),clock_timestamp()
WHERE NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria);
DO $vacia$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.control_cadena_auditoria) <> 1
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
                WHERE secuencia<>0 OR cabeza_sha256<>repeat('0',64))
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
 THEN RAISE EXCEPTION 'AD3-51: la base de prueba no está vacía'; END IF;
END $vacia$;
COMMIT;
SQL

migraciones="$raiz/../migraciones"
down="$migraciones/000051_consulta_rrhh_ambito_productiva.down.sql"
up="$migraciones/000051_consulta_rrhh_ambito_productiva.up.sql"

rechazar_down() {
  local salida estado
  salida=$(mktemp)
  set +e
  docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
    < "$down" > "$salida" 2>&1
  estado=$?
  set -e
  if [[ $estado -ne 3 ]] || ! rg -q 'DOWN rechazado por historia V3' "$salida"; then
    cat "$salida" >&2
    rm -f "$salida"
    echo 'AD3-51: DOWN no rechazó la historia como se esperaba' >&2
    exit 1
  fi
  rm -f "$salida"
  docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
    < "$raiz/consulta_rrhh_ambito_productiva.sql"
}

# Una fila de auditoría sintética basta para impedir DOWN aunque la cadena
# permanezca en génesis. Solo este contenedor efímero usa replica para omitir
# el FK de la fila deliberadamente huérfana; no se instala ni conserva.
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3
  (auditoria_ref,secuencia,decision_ref,efecto_ref,huella_efecto_sha256,
   anterior_sha256,huella_sha256,registrada_en)
VALUES ('aud_ad351_sintetica',1,'decision_ad351_sintetica','efecto_ad351_sintetico',
        repeat('a',64),repeat('0',64),repeat('b',64),clock_timestamp());
COMMIT;
SQL
rechazar_down
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL session_replication_role=replica;
DELETE FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE auditoria_ref='aud_ad351_sintetica';
COMMIT;
SQL

# El marcador de cadena también impide revertir una historia cuyos registros
# hubieran sido retirados fuera del circuito de solo adición.
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
   SET secuencia=1,cabeza_sha256=repeat('b',64) WHERE control_id;
COMMIT;
SQL
rechazar_down
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
   SET secuencia=0,cabeza_sha256=repeat('0',64) WHERE control_id;
COMMIT;
SQL

# Regresión de snapshot heredada: S1 conserva el lock de checkpoint y una
# cadena todavía no confirmada. S2 arranca DOWN con default REPEATABLE READ,
# obtiene su advisory lock y espera el lock de tabla. Al confirmar S1, S2
# debe leer la historia reciente y denegar la reversión sin cambiar ACL/cuerpo.
fifo="/tmp/vec_ad351_fifo_$$"
espera=$(mktemp)
resultado=$(mktemp)
mkfifo "$fifo"
(
  cat <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno IN ROW EXCLUSIVE MODE;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
   SET secuencia=1,cabeza_sha256=repeat('c',64) WHERE control_id;
SELECT 'AD351_LOCK_HELD';
SQL
  read -r _ < "$fifo"
  printf 'COMMIT;\n'
) | docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  > "$espera" 2>&1 &
pid_espera=$!
listo=0
for _ in {1..50}; do
  if rg -q 'AD351_LOCK_HELD' "$espera"; then listo=1; break; fi
  sleep 0.1
done
if [[ $listo -ne 1 ]]; then
  cat "$espera" >&2
  printf 'salir\n' > "$fifo"
  wait "$pid_espera" || true
  rm -f "$fifo" "$espera" "$resultado"
  echo 'AD3-51: no se obtuvo el lock concurrente' >&2
  exit 1
fi
(
  printf "SET default_transaction_isolation='repeatable read';\n"
  cat "$down"
) | docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  > "$resultado" 2>&1 &
pid_down=$!
bloqueado=0
for _ in {1..50}; do
  en_espera=$(docker exec "$contenedor" psql -At -U postgres -d postgres -c \
    "SELECT count(*) FROM pg_stat_activity WHERE pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE 'LOCK TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno,%'")
  if [[ $en_espera == 1 ]]; then bloqueado=1; break; fi
  sleep 0.1
done
printf 'continuar\n' > "$fifo"
wait "$pid_espera"
set +e
wait "$pid_down"
estado_down=$?
set -e
if [[ $bloqueado -ne 1 || $estado_down -ne 3 ]] ||
   ! rg -q 'DOWN rechazado por historia V3' "$resultado"; then
  cat "$espera" "$resultado" >&2
  rm -f "$fifo" "$espera" "$resultado"
  echo 'AD3-51: carrera RR no quedó cerrada' >&2
  exit 1
fi
rm -f "$fifo" "$espera" "$resultado"
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$raiz/consulta_rrhh_ambito_productiva.sql"
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
   SET secuencia=0,cabeza_sha256=repeat('0',64) WHERE control_id;
COMMIT;
SQL
echo 'AD3-51: RR concurrente confirmó historia mientras DOWN esperaba; reversión denegada'

docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$down"
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
DO $reversion$
BEGIN
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
      WHERE oid='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      <> '1cec6ba3faa9d25607273638e458d76dd5f7e1eca0373754d1a2e3f28c6fa137'
    OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
      WHERE oid='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      <> '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff'
 THEN RAISE EXCEPTION 'AD3-51: DOWN no restauró cuerpos previos'; END IF;
END $reversion$;
SQL
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$up"
docker exec -i "$contenedor" psql -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$raiz/consulta_rrhh_ambito_productiva.sql"
echo 'AD3-51: historia sintética rechazada y DOWN/UP vacío verificados'
