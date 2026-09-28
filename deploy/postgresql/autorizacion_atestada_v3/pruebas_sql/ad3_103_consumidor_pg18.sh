#!/usr/bin/env bash
set -Eeuo pipefail
repo=$(git rev-parse --show-toplevel); imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}; contenedor=vec-ad3-103-$$; datos=/dev/shm/$contenedor
limpiar(){ docker rm -f "$contenedor" >/dev/null 2>&1 || true; if [[ -d $datos ]]; then docker run --rm --pull never -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; rmdir "$datos" 2>/dev/null || true; fi; }; trap limpiar EXIT
mkdir -p "$datos"; docker run -d --rm --pull never --network none --name "$contenedor" -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break; sleep .5; done; sleep 2
psql_pg(){ docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }; base=$repo/deploy/postgresql/autorizacion_atestada_v3
psql_pg < "$base/pruebas_sql/ad3_101_preimagen_minima.sql" >/dev/null
psql_pg < "$base/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql" >/dev/null
psql_pg < "$base/migraciones/000102_consumidor_catalogo_causas_bolsa.up.sql" >/dev/null
m=$base/migraciones/000103_consumidor_consulta_catalogo_causas_bolsa; nucleo='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
pre=$(psql_pg -Atc "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))")
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -Atc "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]
[[ $(psql_pg -Atc "SELECT to_regprocedure('vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()') IS NULL") == t ]]
[[ $(psql_pg -Atc "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))") == "$pre" ]]
psql_pg < "$m.up.sql" >/dev/null; if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'AD3-103 doble UP aceptado' >&2; exit 1; fi
psql_pg <<'SQL' >/dev/null
DO $x$ BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN RAISE EXCEPTION 'AD3-103 ACL incorrecta'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()','EXECUTE')
    OR vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()
 THEN RAISE EXCEPTION 'AD3-103 guardia historia incorrecta'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check' AND pg_get_constraintdef(c.oid) LIKE '%causas_participacion.consultar.v1%') THEN RAISE EXCEPTION 'AD3-103 audiencia ausente'; END IF;
END $x$;
SQL
psql_pg < "$m.down.sql" >/dev/null
[[ $(psql_pg -Atc "SELECT to_regprocedure('vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()') IS NULL") == t ]]
[[ $(psql_pg -Atc "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))") == "$pre" ]]
echo 'AD3-103 PG18: ROLLBACK, UP, doble UP, ACL, audiencia y DOWN sin historia OK'
