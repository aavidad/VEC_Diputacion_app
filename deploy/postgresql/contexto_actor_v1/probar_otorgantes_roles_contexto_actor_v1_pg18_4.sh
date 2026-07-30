#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-otorgantes-pg-${USER:-usuario}-$$"
base=ct117_otorgantes
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
roles_up=deploy/postgresql/contexto_actor_v1/roles_up.sql
roles_down=deploy/postgresql/contexto_actor_v1/roles_down.sql

limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
for _ in $(seq 1 200); do
  version=$(docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "SELECT current_setting('server_version_num') || '|' || pg_is_in_recovery()" \
    2>/dev/null || true)
  [[ $version == '180004|false' ]] && break
  sleep 0.05
done
[[ ${version:-} == '180004|false' ]] || {
  echo 'se requiere PostgreSQL 18.4 primario' >&2
  exit 1
}

psql_archivo() {
  local usuario=$1 archivo=$2
  shift 2
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 --username "$usuario" --dbname "$base" "$@" \
    < "$raiz/$archivo"
}
psql_admin() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base"
}
consulta() {
  local sql=$1 usuario=${2:-postgres}
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username "$usuario" --dbname "$base" --command "$sql"
}
retirar_roles() {
  local usuario=$1
  psql_archivo "$usuario" "$roles_down" \
    --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
}
rechazar_retirada() {
  local descripcion=$1 estado
  set +e
  retirar_roles ct117_retirador_tercero >/dev/null 2>&1
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || {
    echo "retirada aceptó $descripcion" >&2
    exit 1
  }
}

psql_admin <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
CREATE ROLE ct117_bootstrap_alterno LOGIN SUPERUSER;
CREATE ROLE ct117_retirador_tercero LOGIN SUPERUSER;
SQL
psql_archivo ct117_bootstrap_alterno "$roles_up"

autoridad_membresia=$(consulta \
  "SELECT pg_catalog.pg_get_userbyid(grantor)
     FROM pg_catalog.pg_auth_members
    WHERE roleid='vec_contexto_actor_v1_propietario'::regrole
      AND member='vec_contexto_actor_v1_migrador'::regrole")
[[ $(consulta \
  "SELECT rolsuper FROM pg_catalog.pg_roles
    WHERE rolname='$autoridad_membresia'") == t ]] || {
  echo "otorgante de membresía no superusuario: $autoridad_membresia" >&2
  exit 1
}
[[ $autoridad_membresia != ct117_bootstrap_alterno ]]
[[ $(consulta \
  "SELECT pg_catalog.pg_get_userbyid(datdba)
     FROM pg_catalog.pg_database WHERE datname=current_database()") != \
  ct117_bootstrap_alterno ]]
numero_acl_bootstrap=$(consulta \
  "SELECT count(*) FROM pg_catalog.pg_database AS b
   CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) AS a
   WHERE b.datname=current_database()
     AND a.grantee=ANY(ARRAY[
       'vec_contexto_actor_v1_propietario'::regrole,
       'vec_contexto_actor_v1_migrador'::regrole,
       'vec_contexto_actor_v1_runtime'::regrole])
     AND pg_catalog.pg_get_userbyid(a.grantor)='$autoridad_membresia'")
[[ $numero_acl_bootstrap == 3 ]] || {
  echo "ACL CONNECT del bootstrap inesperada: $numero_acl_bootstrap" >&2
  exit 1
}

# Una ACL CONNECT concedida por otra autoridad no puede aprovechar el down.
psql_admin <<'SQL'
CREATE ROLE ct117_otorgante_acl LOGIN;
SQL
consulta "GRANT CONNECT ON DATABASE $base TO ct117_otorgante_acl WITH GRANT OPTION" \
  ct117_bootstrap_alterno >/dev/null
consulta "REVOKE CONNECT ON DATABASE $base FROM vec_contexto_actor_v1_propietario,vec_contexto_actor_v1_migrador,vec_contexto_actor_v1_runtime" \
  ct117_bootstrap_alterno >/dev/null
consulta "GRANT CONNECT ON DATABASE $base TO vec_contexto_actor_v1_propietario,vec_contexto_actor_v1_migrador,vec_contexto_actor_v1_runtime" \
  ct117_otorgante_acl >/dev/null
[[ $(consulta \
  "SELECT count(*) FROM pg_catalog.pg_database AS b
   CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) AS a
   WHERE b.datname=current_database()
     AND a.grantee=ANY(ARRAY[
       'vec_contexto_actor_v1_propietario'::regrole,
       'vec_contexto_actor_v1_migrador'::regrole,
       'vec_contexto_actor_v1_runtime'::regrole])
     AND pg_catalog.pg_get_userbyid(a.grantor)='ct117_otorgante_acl'") == 3 ]]
rechazar_retirada 'ACL CONNECT con otorgante divergente'
consulta "REVOKE CONNECT ON DATABASE $base FROM vec_contexto_actor_v1_propietario,vec_contexto_actor_v1_migrador,vec_contexto_actor_v1_runtime" \
  ct117_otorgante_acl >/dev/null
consulta "GRANT CONNECT ON DATABASE $base TO vec_contexto_actor_v1_propietario,vec_contexto_actor_v1_migrador,vec_contexto_actor_v1_runtime" \
  ct117_bootstrap_alterno >/dev/null
consulta "REVOKE CONNECT ON DATABASE $base FROM ct117_otorgante_acl" \
  ct117_bootstrap_alterno >/dev/null
psql_admin <<'SQL'
DROP ROLE ct117_otorgante_acl;
SQL

# Una membresía concedida por otra autoridad tampoco sustituye a la canónica.
psql_admin <<'SQL'
CREATE ROLE ct117_otorgante_membresia LOGIN SUPERUSER;
SQL
consulta 'REVOKE vec_contexto_actor_v1_propietario FROM vec_contexto_actor_v1_migrador' \
  ct117_bootstrap_alterno >/dev/null
consulta 'GRANT vec_contexto_actor_v1_propietario TO ct117_otorgante_membresia WITH ADMIN TRUE,INHERIT FALSE,SET FALSE' \
  ct117_bootstrap_alterno >/dev/null
consulta 'GRANT vec_contexto_actor_v1_propietario TO vec_contexto_actor_v1_migrador WITH ADMIN FALSE,INHERIT FALSE,SET TRUE GRANTED BY ct117_otorgante_membresia' \
  ct117_bootstrap_alterno >/dev/null
[[ $(consulta \
  "SELECT pg_catalog.pg_get_userbyid(grantor)
     FROM pg_catalog.pg_auth_members
    WHERE roleid='vec_contexto_actor_v1_propietario'::regrole
      AND member='vec_contexto_actor_v1_migrador'::regrole") == \
  ct117_otorgante_membresia ]]
rechazar_retirada 'membresía con otorgante divergente'
consulta 'REVOKE vec_contexto_actor_v1_propietario FROM vec_contexto_actor_v1_migrador GRANTED BY ct117_otorgante_membresia' \
  ct117_bootstrap_alterno >/dev/null
consulta 'REVOKE vec_contexto_actor_v1_propietario FROM ct117_otorgante_membresia' \
  ct117_bootstrap_alterno >/dev/null
consulta 'GRANT vec_contexto_actor_v1_propietario TO vec_contexto_actor_v1_migrador WITH ADMIN FALSE,INHERIT FALSE,SET TRUE' \
  ct117_bootstrap_alterno >/dev/null
psql_admin <<'SQL'
DROP ROLE ct117_otorgante_membresia;
SQL

retirar_roles ct117_retirador_tercero
[[ $(consulta \
  "SELECT count(*) FROM pg_catalog.pg_roles
    WHERE rolname LIKE 'vec_contexto_actor_v1_%'") == 0 ]]
psql_admin <<'SQL'
DROP ROLE ct117_retirador_tercero;
DROP ROLE ct117_bootstrap_alterno;
SQL
echo 'ContextoActor V1: otorgantes semánticos de roles superados'
