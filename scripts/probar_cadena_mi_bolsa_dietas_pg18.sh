#!/usr/bin/env bash
# Ensaya deltas pendientes sobre un pg_dump --schema-only de cidonia en PG18 desechable.
# Uso: scripts/probar_cadena_mi_bolsa_dietas_pg18.sh DUMP_SQL ROLES_TXT
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
dump=${1:?falta pg_dump --schema-only}
roles=${2:?falta lista de roles vec_}
[[ -s $dump && -s $roles ]] || { echo 'Falta estructura o lista de roles' >&2; exit 2; }
[[ $(head -n 9 "$dump") == *'Dumped from database version 18.4'* ]] || {
  echo 'Se requiere estructura real de PostgreSQL 18.4' >&2; exit 2;
}
if rg -q '^(COPY |INSERT INTO |\\copy )' "$dump"; then
  echo 'El volcado contiene datos; se exige --schema-only' >&2; exit 2
fi
if rg -qv '^vec_[a-z0-9_]+$' "$roles"; then
  echo 'Lista de roles inválida' >&2; exit 2
fi

container="vec-p2-pg18-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT
docker run -d --rm --network none --name "$container" --env POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
run() { docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { run -At -c "$1"; }
[[ $(scalar 'SHOW server_version') == 18.4 ]] || { echo 'Contenedor PG18.4 no disponible' >&2; exit 2; }

while IFS= read -r role; do
  printf 'CREATE ROLE %s NOLOGIN;\n' "$role"
done < "$roles" | run >/dev/null
echo 'Restaurando estructura real sin filas ni secretos de conexión'
run < "$dump" >/dev/null

assert_eq() {
  local actual=$1 esperado=$2 etiqueta=$3
  [[ $actual == "$esperado" ]] || {
    printf 'FALLO %s: obtenido %s, esperado %s\n' "$etiqueta" "$actual" "$esperado" >&2
    exit 1
  }
  printf 'OK %s\n' "$etiqueta"
}
bolsa=vec_bolsa_llamamientos
dietas=vec_dietas
consulta_sig="$bolsa.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
relleno_sig="$bolsa.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz)"
auditoria_sig="$dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)"

# Las funciones previas, RLS y permisos provienen del dump, no de dobles SQL.
assert_eq "$(scalar "SELECT to_regprocedure('$consulta_sig') IS NOT NULL")" t 'consulta Mi bolsa previa'
assert_eq "$(scalar "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL")" t 'consumidor V3 previo'
assert_eq "$(scalar "SELECT count(*)=14 AND bool_and(c.relrowsecurity AND c.relforcerowsecurity) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind IN ('r','p') AND (n.nspname='$dietas' OR (n.nspname='$bolsa' AND c.relname IN ('constitucion','constitucion_entrada','vinculo_candidato','situacion_participacion','llamamiento_emitido','contacto_participacion')))")" t 'RLS FORCE en tablas afectadas'
assert_eq "$(scalar "SELECT to_regprocedure('$relleno_sig') IS NULL AND to_regclass('$dietas.auditoria_frontera_comision') IS NULL AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_registrador_frontera')")" t 'preimagen de B21 y D5'
assert_eq "$(scalar "SELECT pg_get_functiondef('$consulta_sig'::regprocedure) NOT LIKE '%ultimo_llamamiento%' AND pg_get_functiondef('$consulta_sig'::regprocedure) NOT LIKE '%situacion_actual%'")" t 'preimagen de B22 y B23'

consulta_previa=$(scalar "SELECT md5(pg_get_functiondef('$consulta_sig'::regprocedure))")
consulta_acl_previa=$(scalar "SELECT coalesce(proacl::text,'') FROM pg_proc WHERE oid='$consulta_sig'::regprocedure")
v3_previa=$(scalar "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")
acl_previa=$(scalar "SELECT md5(coalesce(string_agg(n.nspname||'.'||c.relname||':'||coalesce(c.relacl::text,'')||':'||c.relrowsecurity||':'||c.relforcerowsecurity,'|' ORDER BY n.nspname,c.relname),'')) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('$bolsa','$dietas') AND c.relkind IN ('r','p')")

echo 'Ensayo ROLLBACK B21'
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato.up.sql" | run >/dev/null
assert_eq "$(scalar "SELECT to_regprocedure('$relleno_sig') IS NULL")" t 'rollback B21 conserva ausencia de función'
echo 'Ensayo COMMIT B21'
run < "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato.up.sql" >/dev/null
assert_eq "$(scalar "SELECT to_regprocedure('$relleno_sig') IS NOT NULL")" t 'función mantenimiento B21'
assert_eq "$(scalar "SELECT has_function_privilege('vec_bolsa_llamamientos_ejecutor','$relleno_sig','EXECUTE')")" f 'B21 sin EXECUTE de aplicación'
assert_eq "$(scalar "SELECT md5(pg_get_functiondef('$consulta_sig'::regprocedure))")" "$consulta_previa" 'B21 conserva consulta'

echo 'Ensayo ROLLBACK B22'
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000022_consulta_mi_bolsa_situacion.up.sql" | run >/dev/null
assert_eq "$(scalar "SELECT md5(pg_get_functiondef('$consulta_sig'::regprocedure))")" "$consulta_previa" 'rollback B22 conserva definición'
echo 'Ensayo COMMIT B22'
run < "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000022_consulta_mi_bolsa_situacion.up.sql" >/dev/null
assert_eq "$(scalar "SELECT pg_get_functiondef('$consulta_sig'::regprocedure) LIKE '%situacion_actual%' AND pg_get_functiondef('$consulta_sig'::regprocedure) NOT LIKE '%ultimo_llamamiento%'")" t 'proyección B22'
consulta_b22=$(scalar "SELECT md5(pg_get_functiondef('$consulta_sig'::regprocedure))")

echo 'Ensayo ROLLBACK B23'
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000023_consulta_mi_bolsa_ultimo_llamamiento.up.sql" | run >/dev/null
assert_eq "$(scalar "SELECT md5(pg_get_functiondef('$consulta_sig'::regprocedure))")" "$consulta_b22" 'rollback B23 conserva B22'
echo 'Ensayo COMMIT B23'
run < "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000023_consulta_mi_bolsa_ultimo_llamamiento.up.sql" >/dev/null
assert_eq "$(scalar "SELECT pg_get_functiondef('$consulta_sig'::regprocedure) LIKE '%situacion_actual%' AND pg_get_functiondef('$consulta_sig'::regprocedure) LIKE '%ultimo_llamamiento%'")" t 'proyección B23'
assert_eq "$(scalar "SELECT coalesce(proacl::text,'') FROM pg_proc WHERE oid='$consulta_sig'::regprocedure")" "$consulta_acl_previa" 'ACL de consulta conservada'

echo 'Ensayo ROLLBACK D5'
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/deploy/postgresql/dietas_borradores/migraciones/000005_auditoria_frontera.up.sql" | run >/dev/null
assert_eq "$(scalar "SELECT to_regclass('$dietas.auditoria_frontera_comision') IS NULL AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_registrador_frontera')")" t 'rollback D5 elimina tabla y rol nuevos'
echo 'Ensayo COMMIT D5'
run < "$repo/deploy/postgresql/dietas_borradores/migraciones/000005_auditoria_frontera.up.sql" >/dev/null
assert_eq "$(scalar "SELECT to_regprocedure('$auditoria_sig') IS NOT NULL AND to_regclass('$dietas.auditoria_frontera_comision') IS NOT NULL")" t 'objetos D5'
assert_eq "$(scalar "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='$dietas.auditoria_frontera_comision'::regclass")" t 'D5 RLS FORCE'
assert_eq "$(scalar "SELECT NOT rolcanlogin FROM pg_roles WHERE rolname='vec_dietas_registrador_frontera'")" t 'D5 registrador NOLOGIN'
assert_eq "$(scalar "SELECT has_function_privilege('vec_dietas_registrador_frontera','$auditoria_sig','EXECUTE') AND NOT has_function_privilege('vec_dietas_ejecutor','$auditoria_sig','EXECUTE')")" t 'D5 ACL de función'
assert_eq "$(scalar "SELECT NOT has_table_privilege('vec_dietas_registrador_frontera','$dietas.auditoria_frontera_comision','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege('vec_dietas_ejecutor','$dietas.auditoria_frontera_comision','SELECT,INSERT,UPDATE,DELETE')")" t 'D5 sin acceso directo a tabla'
assert_eq "$(scalar "SELECT EXISTS (SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) acl JOIN pg_roles r ON r.oid=acl.grantee WHERE d.datname=current_database() AND r.rolname='vec_dietas_registrador_frontera' AND acl.privilege_type='CONNECT')")" t 'D5 CONNECT nominal explícito'
assert_eq "$(scalar "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")" "$v3_previa" 'historia funcional V3 conservada'
assert_eq "$(scalar "SELECT md5(coalesce(string_agg(n.nspname||'.'||c.relname||':'||coalesce(c.relacl::text,'')||':'||c.relrowsecurity||':'||c.relforcerowsecurity,'|' ORDER BY n.nspname,c.relname),'')) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('$bolsa','$dietas') AND c.relkind IN ('r','p') AND c.oid<>'$dietas.auditoria_frontera_comision'::regclass")" "$acl_previa" 'ACL/RLS de tablas anteriores conservadas'
echo 'OK cadena incremental en estructura real; no se han restaurado ni rellenado filas'
