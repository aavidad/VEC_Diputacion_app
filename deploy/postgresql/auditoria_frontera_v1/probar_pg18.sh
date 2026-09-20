#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
image=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
container="vec-auditoria-frontera-pg18-$$"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker run --rm -d --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust -v "$root:/repo:ro" "$image" >/dev/null
estables=0
for _ in $(seq 1 150); do
  if docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c 'SHOW server_version_num' 2>/dev/null | grep -qx 180004; then
    estables=$((estables + 1))
    if [[ "$estables" -ge 3 ]]; then break; fi
  else
    estables=0
  fi
  sleep .1
done
if [[ "$estables" -lt 3 ]]; then echo 'PG18 no alcanzo estado estable' >&2; exit 1; fi
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SHOW server_version_num" | grep -qx 180004
sql() { docker exec -i "$container" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres; }
file() { docker exec "$container" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres -f "$1"; }
file_vacia() { docker exec "$container" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d auditoria_vacia -f "$1"; }
as_role() { docker exec -i "$container" psql -Xq -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
must_fail() { if printf '%s\n' "$2" | as_role "$1" >/dev/null 2>&1; then echo "debio denegarse: $2" >&2; exit 1; fi; }

file /repo/deploy/postgresql/auditoria_frontera_v1/roles_up.sql
printf '%s\n' "CREATE ROLE frontera_acl_hostil NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; ALTER DEFAULT PRIVILEGES FOR ROLE vec_auditoria_frontera_v1_propietario GRANT EXECUTE ON FUNCTIONS TO frontera_acl_hostil;" | sql
if file /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.up.sql >/dev/null 2>&1; then echo 'UP acepto default ACL hostil' >&2; exit 1; fi
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regnamespace('vec_auditoria_frontera_v1') IS NULL" | grep -qx t
printf '%s\n' "ALTER DEFAULT PRIVILEGES FOR ROLE vec_auditoria_frontera_v1_propietario REVOKE EXECUTE ON FUNCTIONS FROM frontera_acl_hostil; DROP ROLE frontera_acl_hostil;" | sql
printf '%s\n' "CREATE ROLE frontera_schema_acl_hostil NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; ALTER DEFAULT PRIVILEGES FOR ROLE vec_auditoria_frontera_v1_propietario GRANT CREATE ON SCHEMAS TO frontera_schema_acl_hostil;" | sql
if file /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.up.sql >/dev/null 2>&1; then echo 'UP acepto default ACL de esquema hostil' >&2; exit 1; fi
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regnamespace('vec_auditoria_frontera_v1') IS NULL" | grep -qx t
printf '%s\n' "ALTER DEFAULT PRIVILEGES FOR ROLE vec_auditoria_frontera_v1_propietario REVOKE CREATE ON SCHEMAS FROM frontera_schema_acl_hostil; DROP ROLE frontera_schema_acl_hostil; CREATE ROLE frontera_owner_ajeno LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_auditoria_frontera_v1_propietario TO frontera_owner_ajeno WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;" | sql
if file /repo/deploy/postgresql/auditoria_frontera_v1/roles_up.sql >/dev/null 2>&1; then echo 'roles_up acepto SET ROLE hacia propietario' >&2; exit 1; fi
if file /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.up.sql >/dev/null 2>&1; then echo 'UP acepto SET ROLE hacia propietario' >&2; exit 1; fi
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT to_regnamespace('vec_auditoria_frontera_v1') IS NULL" | grep -qx t
printf '%s\n' "REVOKE vec_auditoria_frontera_v1_propietario FROM frontera_owner_ajeno; DROP ROLE frontera_owner_ajeno;" | sql
file /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.up.sql

# El DOWN vacío requiere confirmación explícita y devuelve la base de prueba
# al estado previo; la retirada con historia se comprueba después por separado.
printf '%s\n' 'CREATE DATABASE auditoria_vacia;' | sql
file_vacia /repo/deploy/postgresql/auditoria_frontera_v1/roles_up.sql
file_vacia /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.up.sql
docker exec "$container" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d auditoria_vacia \
  -c "SET vec.confirmar_retirada_denegacion_frontera_identidad_v1='RETIRAR_DENEGACION_FRONTERA_IDENTIDAD_V1'" \
  -f /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.down.sql
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d auditoria_vacia -c "SELECT to_regnamespace('vec_auditoria_frontera_v1') IS NULL" | grep -qx t
printf '%s\n' "CREATE ROLE frontera_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; CREATE ROLE frontera_ajeno LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_auditoria_frontera_identidad_v1_registrador TO frontera_login WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;" | sql

call="SELECT vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1('correlacion_0123456789abcdef0123456789abcdef','externa_personal','/api/vec/bolsa/mi-bolsa','bolsa.participaciones_propias.consultar','autenticacion_requerida','tls-exportador:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',NULL);"
printf '%s\n' "SET ROLE vec_auditoria_frontera_identidad_v1_registrador; $call" | as_role frontera_login | grep -q t
must_fail frontera_login "SELECT * FROM vec_auditoria_frontera_v1.denegacion_identidad;"
must_fail frontera_ajeno "$call"
must_fail frontera_login "SET ROLE vec_auditoria_frontera_identidad_v1_registrador; SELECT vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1('correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','externa_personal','/api/vec/bolsa/mi-bolsa?asercion=x','bolsa.participaciones_propias.consultar','autenticacion_requerida',NULL,NULL);"

printf '%s\n' "SET ROLE vec_auditoria_frontera_identidad_v1_registrador; $call" | as_role frontera_login | grep -q t
must_fail frontera_login "SET ROLE vec_auditoria_frontera_identidad_v1_registrador; SELECT vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1('correlacion_0123456789abcdef0123456789abcdef','externa_personal','/api/vec/bolsa/mi-bolsa','bolsa.participaciones_propias.consultar','acceso_denegado',NULL,NULL);"
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_auditoria_frontera_v1.denegacion_identidad" | grep -qx 1
printf '%s\n' "SET ROLE vec_auditoria_frontera_v1_propietario; UPDATE vec_auditoria_frontera_v1.denegacion_identidad SET motivo='acceso_denegado'; DELETE FROM vec_auditoria_frontera_v1.denegacion_identidad;" | as_role postgres
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_auditoria_frontera_v1.denegacion_identidad WHERE motivo='autenticacion_requerida'" | grep -qx 1
must_fail postgres "SET ROLE vec_auditoria_frontera_v1_propietario; TRUNCATE vec_auditoria_frontera_v1.denegacion_identidad;"

printf '%s\n' "SET ROLE vec_auditoria_frontera_identidad_v1_registrador; BEGIN; SELECT vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1('correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','externa_personal','/api/vec/bolsa/mi-bolsa','bolsa.participaciones_propias.consultar','autenticacion_requerida',NULL,NULL); ROLLBACK;" | as_role frontera_login >/dev/null
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM vec_auditoria_frontera_v1.denegacion_identidad WHERE correlacion_ref='correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'" | grep -qx 0
if docker exec "$container" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SET vec.confirmar_retirada_denegacion_frontera_identidad_v1='RETIRAR_DENEGACION_FRONTERA_IDENTIDAD_V1'" -f /repo/deploy/postgresql/auditoria_frontera_v1/migraciones/000001_denegacion_frontera_identidad_v1.down.sql >/dev/null 2>&1; then echo 'DOWN con historia y confirmacion aceptado' >&2; exit 1; fi
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_auditoria_frontera_v1.denegacion_identidad'::regclass" | grep -qx t
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT count(*) FROM pg_policy WHERE polrelid='vec_auditoria_frontera_v1.denegacion_identidad'::regclass AND polcmd IN ('r','a') AND polroles=ARRAY['vec_auditoria_frontera_v1_propietario'::regrole::oid]" | grep -qx 2
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT has_schema_privilege('vec_auditoria_frontera_identidad_v1_registrador','vec_auditoria_frontera_v1','USAGE') AND NOT has_table_privilege('vec_auditoria_frontera_identidad_v1_registrador','vec_auditoria_frontera_v1.denegacion_identidad','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')" | grep -qx t
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT NOT EXISTS (SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid='vec_auditoria_frontera_v1'::regnamespace AND NOT (a.grantee=n.nspowner OR (a.grantee='vec_auditoria_frontera_identidad_v1_registrador'::regrole AND a.privilege_type='USAGE' AND NOT a.is_grantable)))" | grep -qx t
docker exec "$container" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "SELECT NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace='vec_auditoria_frontera_v1'::regnamespace AND NOT (a.grantee=p.proowner OR (p.oid='vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text)'::regprocedure AND a.grantee='vec_auditoria_frontera_identidad_v1_registrador'::regrole AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)))" | grep -qx t
echo 'PG18 auditoria frontera: default ACLs, membresia owner, ACL/RLS exactos, contaminacion, replay, cross-role, rollback y ambos DOWN OK'
