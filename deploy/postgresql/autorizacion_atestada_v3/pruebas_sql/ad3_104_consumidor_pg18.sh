#!/usr/bin/env bash
set -Eeuo pipefail
repo=$(git rev-parse --show-toplevel); imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}; c=vec-ad3-104-$$; d=/dev/shm/$c
clean(){ docker rm -f "$c" >/dev/null 2>&1 || true; if [[ -d $d ]]; then docker run --rm --pull never -v "$d:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; rmdir "$d" 2>/dev/null || true; fi; }; trap clean EXIT
mkdir -p "$d"; docker run -d --rm --pull never --network none --name "$c" -v "$d:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do docker exec "$c" pg_isready -q -U postgres >/dev/null 2>&1 && break; sleep .5; done; sleep 2
p(){ docker exec -i "$c" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }; b=$repo/deploy/postgresql/autorizacion_atestada_v3
p < "$b/pruebas_sql/ad3_101_preimagen_minima.sql" >/dev/null
for n in 000101_consumidor_consulta_reincorporacion_titular_bolsa 000102_consumidor_catalogo_causas_bolsa 000103_consumidor_consulta_catalogo_causas_bolsa; do p < "$b/migraciones/$n.up.sql" >/dev/null; done
m=$b/migraciones/000104_consumidor_propuesta_catalogo_causas_bolsa; f='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'; pre=$(p -Atc "SELECT md5(pg_get_functiondef('$f'::regprocedure))")
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | p >/dev/null
[[ $(p -Atc "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]
[[ $(p -Atc "SELECT md5(pg_get_functiondef('$f'::regprocedure))") == "$pre" ]]
p < "$m.up.sql" >/dev/null; if p < "$m.up.sql" >/dev/null 2>&1; then echo 'AD3-104 doble UP aceptado' >&2; exit 1; fi
p <<'SQL' >/dev/null
DO $x$ BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN RAISE EXCEPTION 'AD3-104 ACL incorrecta'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check' AND pg_get_constraintdef(c.oid) LIKE '%causas_participacion.proponer.v1%') THEN RAISE EXCEPTION 'AD3-104 audiencia ausente'; END IF;
END $x$;
SQL
p < "$m.down.sql" >/dev/null; [[ $(p -Atc "SELECT md5(pg_get_functiondef('$f'::regprocedure))") == "$pre" ]]
echo 'AD3-104 PG18: ROLLBACK, UP, doble UP, ACL, audiencia y DOWN sin historia OK'
