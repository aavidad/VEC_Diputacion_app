#!/usr/bin/env bash
# AD3-95 sobre una copia sintética con AD3-94 ya instalada. Nunca toca la base
# conservada. Uso: probar_consumidor_politica_cese_95_pg18.sh GLOBALES_SQL DUMP_FC
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta volcado sintético de roles}
volcado=${2:?falta volcado sintético pg_dump -Fc con AD3-94}
[[ -s $globales && -s $volcado ]] || { echo 'faltan los volcados sintéticos' >&2; exit 2; }
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
nombre=vec-pg-ad3-95-$$
datos=/dev/shm/$nombre
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm --pull never -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true
    rmdir "$datos" 2>/dev/null || true
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$nombre" -v "$datos:/var/lib/postgresql" -v "$repo:/repo:ro" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 120); do
  docker exec "$nombre" pg_isready -q -U postgres 2>/dev/null && break
  sleep 0.5
done
psql_pg() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
valor() { docker exec "$nombre" psql -XAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
docker exec -i "$nombre" psql -X -q -U postgres -d postgres < "$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres < "$volcado" >/dev/null 2>&1 || true
f=vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada
firma="$f(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
[[ $(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL") == t ]]
[[ $(valor "SELECT to_regprocedure('$firma') IS NULL") == t ]]
m=deploy/postgresql/autorizacion_atestada_v3/migraciones/000095_consumidor_politica_cese_bolsa
pre=$(valor "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m.up.sql" | psql_pg >/dev/null
[[ $(valor "SELECT to_regprocedure('$firma') IS NULL") == t ]]
[[ $(valor "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))") == "$pre" ]]
psql_pg -f "/repo/$m.up.sql" >/dev/null
if psql_pg -f "/repo/$m.up.sql" >/dev/null 2>&1; then echo 'AD3-95: doble UP aceptado' >&2; exit 1; fi
[[ $(valor "SELECT to_regprocedure('$firma') IS NOT NULL") == t ]]
[[ $(valor "SELECT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$firma','EXECUTE') AND NOT has_function_privilege('public','$firma','EXECUTE')") == t ]]
[[ $(valor "SELECT strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'consulta_politica_cese_bolsa')>0") == t ]]
psql_pg >/dev/null <<'SQL'
CREATE ROLE vec_ad3_95_ajeno LOGIN INHERIT;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_ad3_95_ajeno;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_ad3_95_ajeno;
SQL
codigo=$(docker exec -i "$nombre" psql -XAt -U vec_ad3_95_ajeno -d postgres 2>&1 <<'SQL' | tail -n 1
DO $p$ BEGIN
 PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada(
  convert_to('{"audiencia_consumo":"vec_bolsa_llamamientos.politica_cese.consultar.v1","operacion":"bolsa.politica_cese.consultar","efecto_ref":"politica:bolsa:cese:vigente","huella_efecto_sha256":"'||repeat('a',64)||'"}','UTF8'),
  convert_to('{"accion":"bolsa.politica_cese.consultar","modulo_id":"bolsa","tipo_recurso":"politica_cese_bolsa","finalidad":"consulta_politica_cese_rrhh","recurso_ref":"politica:bolsa:cese:vigente","contexto_recurso_huella_sha256":"'||repeat('a',64)||'","campos_permitidos":["politica_cese"],"obligaciones":[]}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE NOTICE 'aceptada';
EXCEPTION WHEN OTHERS THEN RAISE NOTICE 'codigo:%',SQLSTATE; END $p$;
SQL
)
[[ $codigo == *'codigo:42501'* ]] || { echo "AD3-95: LOGIN ajeno no denegado con 42501: $codigo" >&2; exit 1; }
echo 'AD3-95: PG18 ROLLBACK, UP, doble UP, ACL y LOGIN ajeno 42501 OK'
