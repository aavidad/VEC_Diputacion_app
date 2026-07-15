#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-pg-prueba-${USER:-usuario}-$$"
base=vec_autorizacion_prueba
clave_admin="vec-admin-prueba-$$"
clave_migrador="vec-migrador-prueba-$$"
clave_fuente="vec-fuente-prueba-$$"
clave_registro="vec-registro-prueba-$$"

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm \
    --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" \
    --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null

for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null

# Un homonimo, aunque parezca tener los atributos previstos, pertenece a un
# tercero. El bootstrap debe rechazarlo antes de corregir membresias, ACL o
# privilegios de PUBLIC. La prueba conserva y compara el estado observable.
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_autorizacion_registro NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;
GRANT pg_read_all_data TO vec_autorizacion_registro;
DO $bloque$
BEGIN
    EXECUTE format(
        'GRANT TEMPORARY ON DATABASE %I TO vec_autorizacion_registro',
        current_database()
    );
END
$bloque$;
SQL

estado_homonimo_antes=$(docker exec "$contenedor" \
    psql --tuples-only --no-align --username postgres --dbname "$base" \
    --command "SELECT r.oid::text || '|' || COALESCE(d.datacl::text, '<NULL>') FROM pg_catalog.pg_roles AS r CROSS JOIN pg_catalog.pg_database AS d WHERE r.rolname = 'vec_autorizacion_registro' AND d.datname = current_database()")

if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up adopto un rol homonimo contaminado" >&2
    exit 1
fi

estado_homonimo_despues=$(docker exec "$contenedor" \
    psql --tuples-only --no-align --username postgres --dbname "$base" \
    --command "SELECT r.oid::text || '|' || COALESCE(d.datacl::text, '<NULL>') FROM pg_catalog.pg_roles AS r CROSS JOIN pg_catalog.pg_database AS d WHERE r.rolname = 'vec_autorizacion_registro' AND d.datname = current_database()")
if [[ "$estado_homonimo_despues" != "$estado_homonimo_antes" ]]; then
    echo "roles_up modifico el rol homonimo o la ACL de base antes de abortar" >&2
    exit 1
fi

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'pg_read_all_data'
           AND miembro.rolname = 'vec_autorizacion_registro'
    ) THEN
        RAISE EXCEPTION 'roles_up modifico la membresia ajena antes de abortar';
    END IF;
END
$bloque$;
DO $bloque$
BEGIN
    EXECUTE format(
        'REVOKE TEMPORARY ON DATABASE %I FROM vec_autorizacion_registro',
        current_database()
    );
END
$bloque$;
REVOKE pg_read_all_data FROM vec_autorizacion_registro;
DROP ROLE vec_autorizacion_registro;
SQL

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_up.sql"

# Un segundo bootstrap no es una reparacion ni una auditoria: falla cerrado y
# exige que el DBA use un verificador de solo lectura o una retirada aprobada.
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up permitio una segunda ejecucion" >&2
    exit 1
fi

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
DECLARE
    cantidad integer;
BEGIN
    SELECT count(*)
      INTO cantidad
      FROM pg_catalog.pg_roles
     WHERE rolname IN (
         'vec_autorizacion_propietario', 'vec_autorizacion_migrador',
         'vec_autorizacion_fuente', 'vec_autorizacion_registro'
     )
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreaterole
       AND NOT rolcreatedb
       AND NOT rolreplication
       AND NOT rolbypassrls;
    IF cantidad <> 4 THEN
        RAISE EXCEPTION 'atributos inesperados tras el bootstrap: %', cantidad;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_autorizacion_propietario'
           AND miembro.rolname = 'vec_autorizacion_migrador'
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'opciones inesperadas en la membresia de migracion';
    END IF;
END
$bloque$;
SQL
docker exec --interactive --env VEC_MIGRADOR_PASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
\getenv clave_migrador VEC_MIGRADOR_PASSWORD
CREATE ROLE vec_autorizacion_migrador_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_migrador';
GRANT vec_autorizacion_migrador TO vec_autorizacion_migrador_prueba;
SQL
docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql"

evolucion_vinculo="$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual"
# El contrato actual de DecisionAutorizacion añade el bloque sellado de
# autenticacion-actor. Se prueba tambien que la evolucion baja limpiamente
# antes de existir evidencia V2 y que puede aplicarse de nuevo desde cero.
docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "${evolucion_vinculo}.up.sql"
docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "${evolucion_vinculo}.down.sql"
docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "${evolucion_vinculo}.up.sql"

docker exec --interactive \
    --env VEC_FUENTE_PASSWORD="$clave_fuente" \
    --env VEC_REGISTRO_PASSWORD="$clave_registro" \
    "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    <<'SQL'
\getenv clave_fuente VEC_FUENTE_PASSWORD
\getenv clave_registro VEC_REGISTRO_PASSWORD
CREATE ROLE vec_autorizacion_fuente_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_fuente';
CREATE ROLE vec_autorizacion_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registro';
GRANT vec_autorizacion_fuente TO vec_autorizacion_fuente_prueba;
GRANT vec_autorizacion_registro TO vec_autorizacion_registro_prueba;
SQL

puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efimero de PostgreSQL" >&2
    exit 1
fi

export VEC_POSTGRES_TEST_FUENTE_DSN="postgresql://vec_autorizacion_fuente_prueba:${clave_fuente}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_REGISTRO_DSN="postgresql://vec_autorizacion_registro_prueba:${clave_registro}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_ADMIN_DSN="postgresql://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"

(cd "$raiz" && go test ./internal/vec/adapters/postgres -run TestIntegracionAutorizacionPostgreSQL -count=1)

docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.down.sql"
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = 'vec_autorizacion'
    ) THEN
        RAISE EXCEPTION 'la migracion descendente no retiro el esquema';
    END IF;
END
$bloque$;
REVOKE vec_autorizacion_fuente FROM vec_autorizacion_fuente_prueba;
REVOKE vec_autorizacion_registro FROM vec_autorizacion_registro_prueba;
REVOKE vec_autorizacion_migrador FROM vec_autorizacion_migrador_prueba;
DROP ROLE vec_autorizacion_fuente_prueba;
DROP ROLE vec_autorizacion_registro_prueba;
DROP ROLE vec_autorizacion_migrador_prueba;
SQL
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_down.sql"
