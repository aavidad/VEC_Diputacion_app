#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${VEC_D7_TEST_BASE:?directorio privado 0700 fuera de Git requerido}"
base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_dir=$(cd "$base_dir/../../.." && pwd -P)
test_base=$(realpath -e "$VEC_D7_TEST_BASE")
[[ $(stat -c %a "$test_base") == 700 && $(stat -c %u "$test_base") == "$(id -u)" ]] || exit 2
case "$test_base/" in "$repo_dir/"*) exit 2;; esac
test_dir=$(mktemp -d "$test_base/d7-pg18.XXXXXXXX")
contenedor="d7-personal-pg18-$$"
trap 'docker rm -f "$contenedor" >/dev/null 2>&1 || true' EXIT
docker run --rm --network none -d --name "$contenedor" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres; then break; fi
  sleep 1
done
psql_admin() { docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -h /var/run/postgresql -U postgres -d postgres "$@"; }
run_d7() {
  env -u PGSERVICE -u PGSERVICEFILE -u PGPASSFILE -u PGHOST -u PGHOSTADDR \
    -u PGPORT -u PGDATABASE -u PGUSER -u PGPASSWORD -u PGOPTIONS \
    VEC_D7_POSTGRES_CONTAINER="$contenedor" VEC_D7_EVIDENCIA_DIR="$test_dir" \
    VEC_D7_APLICAR=SI-D7-REVISADO bash "$base_dir/ejecutar.sh" "$@"
}
psql_admin < "$base_dir/fixture_pg18.sql" >/dev/null
run_d7 --preparar-rollback > "$test_dir/preparar-rollback.out"
[[ $(psql_admin -c "SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera')") == 0 ]]

# Negativos: preimagen F4, numeración y grants exactos.
psql_admin -c "UPDATE vec_autorizacion.asignacion_perfil SET version=4" >/dev/null
if run_d7 --preparar-commit > "$test_dir/f4.out" 2> "$test_dir/f4.err"; then exit 1; fi
psql_admin -c "UPDATE vec_autorizacion.asignacion_perfil SET version=3" >/dev/null
psql_admin -c 'CREATE ROLE vec_dietas_r1d_auditoria_frontera_desarrollo NOLOGIN' >/dev/null
if run_d7 --preparar-commit > "$test_dir/novena.out" 2> "$test_dir/novena.err"; then exit 1; fi
psql_admin -c 'DROP ROLE vec_dietas_r1d_auditoria_frontera_desarrollo' >/dev/null
psql_admin -c 'REVOKE EXECUTE ON FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_personal_d7_ejecutor' >/dev/null
if run_d7 --preparar-commit > "$test_dir/grant.out" 2> "$test_dir/grant.err"; then exit 1; fi
psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_d7_ejecutor' >/dev/null
psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint) TO PUBLIC' >/dev/null
if run_d7 --preparar-commit > "$test_dir/public.out" 2> "$test_dir/public.err"; then exit 1; fi
psql_admin -c 'REVOKE EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint) FROM PUBLIC' >/dev/null
psql_admin -c 'GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_personal_d7_ejecutor' >/dev/null
psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.efecto_sensible() TO vec_personal_d7_ejecutor' >/dev/null
if run_d7 --preparar-commit > "$test_dir/grant-ct.out" 2> "$test_dir/grant-ct.err"; then exit 1; fi
psql_admin -c 'REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.efecto_sensible() FROM vec_personal_d7_ejecutor' >/dev/null
psql_admin -c 'REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_personal_d7_ejecutor' >/dev/null
psql_admin -c "CREATE FUNCTION vec_personal.efecto_ajeno() RETURNS boolean LANGUAGE sql SECURITY DEFINER AS \$\$SELECT true\$\$" >/dev/null
psql_admin -c 'REVOKE ALL ON FUNCTION vec_personal.efecto_ajeno() FROM PUBLIC' >/dev/null
psql_admin -c 'GRANT EXECUTE ON FUNCTION vec_personal.efecto_ajeno() TO vec_personal_d7_ejecutor' >/dev/null
if run_d7 --preparar-commit > "$test_dir/funcion-ajena.out" 2> "$test_dir/funcion-ajena.err"; then exit 1; fi
psql_admin -c 'DROP FUNCTION vec_personal.efecto_ajeno()' >/dev/null
psql_admin -c 'ALTER FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SECURITY INVOKER' >/dev/null
if run_d7 --preparar-commit > "$test_dir/invocador.out" 2> "$test_dir/invocador.err"; then exit 1; fi
psql_admin -c 'ALTER FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SECURITY DEFINER' >/dev/null
psql_admin -c 'CREATE SCHEMA vec_d7_propiedad_ajena AUTHORIZATION vec_personal_d7_ejecutor' >/dev/null
if run_d7 --preparar-commit > "$test_dir/propiedad.out" 2> "$test_dir/propiedad.err"; then exit 1; fi
psql_admin -c 'DROP SCHEMA vec_d7_propiedad_ajena' >/dev/null
[[ $(psql_admin -c "SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera')") == 0 ]]

run_d7 --preparar-commit > "$test_dir/preparar-commit.out"
if run_d7 --activar-commit > "$test_dir/sin-credencial.out" 2> "$test_dir/sin-credencial.err"; then exit 1; fi
# Contraseñas sintéticas generadas solo en memoria para esta base efímera.
clave_a=$(openssl rand -hex 32)
clave_b=$(openssl rand -hex 32)
printf "ALTER ROLE vec_personal_d7_asignacion PASSWORD '%s';\n" "$clave_a" | psql_admin >/dev/null
psql_admin <<'SQL' >/dev/null
DO $x$ DECLARE secreto text; BEGIN
 SELECT rolpassword INTO secreto FROM pg_authid WHERE rolname='vec_personal_d7_asignacion';
 EXECUTE format('ALTER ROLE vec_personal_d7_auditoria_frontera PASSWORD %L',secreto);
END $x$;
SQL
if run_d7 --activar-commit > "$test_dir/credencial-compartida.out" 2> "$test_dir/credencial-compartida.err"; then exit 1; fi
printf "ALTER ROLE vec_personal_d7_auditoria_frontera PASSWORD '%s';\n" "$clave_b" | psql_admin >/dev/null
unset clave_a clave_b
run_d7 --activar-rollback > "$test_dir/activar-rollback.out"
[[ $(psql_admin -c "SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera') AND rolcanlogin") == 0 ]]
if docker exec "$contenedor" psql -XAtq -h /var/run/postgresql -U vec_personal_d7_asignacion -d postgres -c 'SELECT 1' > "$test_dir/login-previo.out" 2> "$test_dir/login-previo.err"; then exit 1; fi
psql_admin -c 'GRANT vec_personal_registrador_frontera TO vec_personal_d7_asignacion WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' >/dev/null
if run_d7 --activar-commit > "$test_dir/cruce.out" 2> "$test_dir/cruce.err"; then exit 1; fi
psql_admin -c 'REVOKE vec_personal_registrador_frontera FROM vec_personal_d7_asignacion' >/dev/null
psql_admin -c 'GRANT SELECT ON vec_autorizacion.version_rol TO vec_personal_d7_asignacion' >/dev/null
if run_d7 --activar-commit > "$test_dir/directo.out" 2> "$test_dir/directo.err"; then exit 1; fi
psql_admin -c 'REVOKE SELECT ON vec_autorizacion.version_rol FROM vec_personal_d7_asignacion' >/dev/null
run_d7 --activar-commit > "$test_dir/activar-commit.out"
[[ $(psql_admin -c "SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera') AND rolcanlogin") == 2 ]]
[[ $(docker exec "$contenedor" psql -XAtq -h /var/run/postgresql -U vec_personal_d7_asignacion -d postgres -c 'SELECT current_user') == vec_personal_d7_asignacion ]]
[[ $(docker exec "$contenedor" psql -XAtq -h /var/run/postgresql -U vec_personal_d7_auditoria_frontera -d postgres -c 'SELECT current_user') == vec_personal_d7_auditoria_frontera ]]
[[ $(docker exec "$contenedor" psql -XAtq -h /var/run/postgresql -U vec_personal_d7_asignacion -d postgres -c "SELECT has_function_privilege(current_user,'vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')||'|'||has_function_privilege(current_user,'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)','EXECUTE')") == true\|false ]]
[[ $(docker exec "$contenedor" psql -XAtq -h /var/run/postgresql -U vec_personal_d7_auditoria_frontera -d postgres -c "SELECT has_function_privilege(current_user,'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)','EXECUTE')||'|'||has_function_privilege(current_user,'vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == true\|false ]]
[[ $(psql_admin -c "SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin") == 8 ]]
if run_d7 --activar-commit > "$test_dir/repetir.out" 2> "$test_dir/repetir.err"; then exit 1; fi
echo 'D7 PostgreSQL 18: rollback, commit, F4, ACL, PUBLIC y segregacion verificados'
