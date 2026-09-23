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
