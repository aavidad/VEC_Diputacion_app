#!/usr/bin/env bash
# AD3-101: prueba estructural en PostgreSQL 18 sintético, sin material criptográfico.
set -Eeuo pipefail
repo=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor=vec-ad3-101-$$
datos=/dev/shm/$contenedor
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm --pull never -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
    rmdir "$datos" 2>/dev/null || true
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$contenedor" \
  -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break; sleep 0.5; done
sleep 2
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
base=$repo/deploy/postgresql/autorizacion_atestada_v3
psql_pg < "$base/pruebas_sql/ad3_101_preimagen_minima.sql" >/dev/null
m=$base/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa
pre=$(psql_pg -Atc "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -Atc "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]
[[ $(psql_pg -Atc "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))") == "$pre" ]]
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'AD3-101 doble UP aceptado' >&2; exit 1; fi
psql_pg <<'SQL' >/dev/null
DO $acl$ BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'AD3-101 ACL incorrecta'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'
  AND pg_get_constraintdef(c.oid) LIKE '%reincorporacion_titular.consultar.v1%') THEN
  RAISE EXCEPTION 'AD3-101 audiencia ausente'; END IF;
END $acl$;
SQL
psql_pg < "$m.down.sql" >/dev/null
[[ $(psql_pg -Atc "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))") == "$pre" ]]
echo 'AD3-101 PG18: ROLLBACK, UP, doble UP, ACL, audiencia y DOWN sin historia OK'
