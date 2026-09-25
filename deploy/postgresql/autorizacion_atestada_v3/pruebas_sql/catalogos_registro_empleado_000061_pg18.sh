#!/usr/bin/env bash
# Ensayo estructural AD3-61; preimagen 051/052 sintética, 059/060 reales.
# No sustituye la cadena V3 completa ni una decisión firmada.
set -euo pipefail
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-b2-ad3-61-${BASHPID}"
base=postgres
ad3="$repo_dir/deploy/postgresql/autorizacion_atestada_v3"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
psql_admin() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" -o /dev/null; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "$1"; }
archivo() { psql_admin < "$1"; }
fallo() { printf 'AD3-61: %s\n' "$*" >&2; exit 1; }
if [[ -n ${AD3_59_SQL:-} ]]; then
  [[ -f $AD3_59_SQL ]] || fallo 'AD3_59_SQL no existe'
  fuente59=$AD3_59_SQL
else
  fuente59=$(mktemp)
  trap 'limpiar; rm -f "$fuente59"' EXIT
  git -C "$repo_dir" show trabajo/dietas-montaje-20260925:deploy/postgresql/autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql > "$fuente59" || fallo 'falta AD3-59 real de Dietas'
fi
"$motor" run -d --rm --name "$contenedor" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in {1..80}; do
  if valor 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'se requiere PostgreSQL 18.4'
archivo "$ad3/pruebas_sql/organizacion_historica_ad3_000051_stub.sql"
archivo "$ad3/migraciones/000051_consumidor_organizacion_historica.up.sql"
archivo "$ad3/migraciones/000052_consumidor_importacion_organizacion.up.sql"
psql_admin <<'SQL'
CREATE ROLE vec_dietas_propietario NOLOGIN;
CREATE ROLE vec_dietas_ejecutor NOLOGIN NOBYPASSRLS;
SQL
archivo "$fuente59"
archivo "$ad3/migraciones/000060_consumidor_registro_empleado_b2.up.sql"
sed '$s/^COMMIT;/ROLLBACK;/' "$ad3/migraciones/000061_consumidor_catalogos_registro_empleado.up.sql" | psql_admin
firma='vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
[[ $(valor "SELECT to_regprocedure('$firma') IS NULL") == t ]] || fallo 'ROLLBACK dejó consumidor'
archivo "$ad3/migraciones/000061_consumidor_catalogos_registro_empleado.up.sql"
psql_admin <<'SQL'
CREATE ROLE vec_catalogo_ajeno NOLOGIN NOBYPASSRLS;
SQL
[[ $(valor "SELECT has_function_privilege('vec_personal_propietario','$firma','EXECUTE'),has_function_privilege('vec_personal_ejecutor','$firma','EXECUTE'),has_function_privilege('vec_catalogo_ajeno','$firma','EXECUTE')") == 't|f|f' ]] || fallo 'ACL consumidor divergente'
psql_admin <<'SQL'
DO $neg$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
   convert_to('{"operacion":"personal.registro_empleado.catalogo.borrar"}','UTF8'),
   convert_to('{"modulo_id":"personal"}','UTF8'),'x'::bytea,'x'::bytea,1,1,
   'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'operación no nominal aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $neg$;
SQL
"$motor" restart "$contenedor" >/dev/null
for _ in {1..80}; do
  if valor 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 0.25
done
[[ $(valor "SELECT to_regprocedure('$firma') IS NOT NULL") == t ]] || fallo 'consumidor perdido tras reinicio'
printf 'AD3-61 PG18: ROLLBACK, COMMIT, ACL y reinicio correctos (preimagen 051/052 sintética, 059/060 reales).\n'
