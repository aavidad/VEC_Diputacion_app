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
clave_proyector="vec-proyector-motivos-prueba-$$"
clave_evaluador="vec-evaluador-motivos-prueba-$$"
clave_dba_limitado="vec-dba-limitado-prueba-$$"
base_concurrencia=vec_motivos_v2_concurrencia
cache_go=$(mktemp -d /tmp/vec-autorizacion-gocache-XXXXXX)
error_grant_v2=$(mktemp /tmp/vec-autorizacion-grant-v2-XXXXXX)
error_grant_v1=$(mktemp /tmp/vec-autorizacion-grant-v1-XXXXXX)

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -rf "$cache_go"
    rm -f "$error_grant_v2" "$error_grant_v1"
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
puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efimero de PostgreSQL" >&2
    exit 1
fi

# Ser propietario de la base y tener CREATEROLE no convierte una identidad en
# DBA gobernado. Se usa una conexion LOGIN real: el preflight V1 debe rechazarla
# antes de crear un solo rol o alterar ACL de base/esquema.
docker exec --interactive --env VEC_DBA_LIMITADO_PASSWORD="$clave_dba_limitado" \
    "$contenedor" psql --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" <<'SQL'
\getenv clave_dba_limitado VEC_DBA_LIMITADO_PASSWORD
CREATE ROLE vec_autorizacion_dba_limitado_prueba LOGIN NOSUPERUSER NOCREATEDB
    CREATEROLE INHERIT NOREPLICATION NOBYPASSRLS
    PASSWORD :'clave_dba_limitado';
ALTER DATABASE vec_autorizacion_prueba
    OWNER TO vec_autorizacion_dba_limitado_prueba;
SQL
estado_limitado_antes=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT d.datdba::text || '|' || COALESCE(d.datacl::text, '<NULL>') || '|' || n.nspowner::text || '|' || COALESCE(n.nspacl::text, '<NULL>') FROM pg_catalog.pg_database AS d CROSS JOIN pg_catalog.pg_namespace AS n WHERE d.datname = current_database() AND n.nspname = 'public'")
if docker exec --interactive --env PGPASSWORD="$clave_dba_limitado" \
    "$contenedor" psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_dba_limitado_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up acepto a un propietario de base NOSUPERUSER CREATEROLE" >&2
    exit 1
fi
estado_limitado_despues=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT d.datdba::text || '|' || COALESCE(d.datacl::text, '<NULL>') || '|' || n.nspowner::text || '|' || COALESCE(n.nspacl::text, '<NULL>') FROM pg_catalog.pg_database AS d CROSS JOIN pg_catalog.pg_namespace AS n WHERE d.datname = current_database() AND n.nspname = 'public'")
roles_limitado_v1=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_propietario','vec_autorizacion_migrador','vec_autorizacion_fuente','vec_autorizacion_registro')")
if [[ "$estado_limitado_despues" != "$estado_limitado_antes" \
      || "$roles_limitado_v1" != "0" ]]; then
    echo "roles_up dejo estado tras rechazar al propietario no superusuario" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command 'ALTER DATABASE vec_autorizacion_prueba OWNER TO postgres'

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

# Con V1 ya presente, el mismo propietario limitado alcanzaria todas las
# precondiciones funcionales de V2. Debe fallar especificamente por no ser
# superusuario y conservar tanto V1 como la ACL que encontro.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command 'ALTER DATABASE vec_autorizacion_prueba OWNER TO vec_autorizacion_dba_limitado_prueba'
estado_limitado_v2_antes=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT d.datdba::text || '|' || COALESCE(d.datacl::text, '<NULL>') || '|' || n.nspowner::text || '|' || COALESCE(n.nspacl::text, '<NULL>') FROM pg_catalog.pg_database AS d CROSS JOIN pg_catalog.pg_namespace AS n WHERE d.datname = current_database() AND n.nspname = 'public'")
if docker exec --interactive --env PGPASSWORD="$clave_dba_limitado" \
    "$contenedor" psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_dba_limitado_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_up.sql" >/dev/null 2>&1; then
    echo "roles_v2_up acepto a un propietario NOSUPERUSER CREATEROLE" >&2
    exit 1
fi
estado_limitado_v2_despues=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT d.datdba::text || '|' || COALESCE(d.datacl::text, '<NULL>') || '|' || n.nspowner::text || '|' || COALESCE(n.nspacl::text, '<NULL>') FROM pg_catalog.pg_database AS d CROSS JOIN pg_catalog.pg_namespace AS n WHERE d.datname = current_database() AND n.nspname = 'public'")
roles_limitado_v2=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_motivos_proyector','vec_autorizacion_motivos_evaluador')")
if [[ "$estado_limitado_v2_despues" != "$estado_limitado_v2_antes" \
      || "$roles_limitado_v2" != "0" ]]; then
    echo "roles_v2_up dejo estado tras rechazar al propietario no superusuario" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
ALTER DATABASE vec_autorizacion_prueba OWNER TO postgres;
DROP ROLE vec_autorizacion_dba_limitado_prueba;
SQL

# La evolucion V2 conserva la misma regla anti-adopcion, pero sus roles son
# independientes de V1. Un homonimo contaminado no se corrige ni se reutiliza.
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_autorizacion_motivos_proyector NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT pg_read_all_data TO vec_autorizacion_motivos_proyector;
SQL
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_up.sql" >/dev/null 2>&1; then
    echo "roles_v2_up adopto un rol homonimo contaminado" >&2
    exit 1
fi
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
BEGIN
    IF NOT pg_has_role(
        'vec_autorizacion_motivos_proyector', 'pg_read_all_data', 'MEMBER'
    ) THEN
        RAISE EXCEPTION 'roles_v2_up modifico la membresia ajena al abortar';
    END IF;
END
$bloque$;
REVOKE pg_read_all_data FROM vec_autorizacion_motivos_proyector;
DROP ROLE vec_autorizacion_motivos_proyector;
SQL
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_up.sql"
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_up.sql" >/dev/null 2>&1; then
    echo "roles_v2_up permitio una segunda ejecucion" >&2
    exit 1
fi

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
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql"
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
    --env VEC_PROYECTOR_PASSWORD="$clave_proyector" \
    --env VEC_EVALUADOR_PASSWORD="$clave_evaluador" \
    "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    <<'SQL'
\getenv clave_fuente VEC_FUENTE_PASSWORD
\getenv clave_registro VEC_REGISTRO_PASSWORD
\getenv clave_proyector VEC_PROYECTOR_PASSWORD
\getenv clave_evaluador VEC_EVALUADOR_PASSWORD
CREATE ROLE vec_autorizacion_fuente_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_fuente';
CREATE ROLE vec_autorizacion_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registro';
CREATE ROLE vec_autorizacion_motivos_proyector_prueba LOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_proyector';
CREATE ROLE vec_autorizacion_motivos_evaluador_prueba LOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_evaluador';
GRANT vec_autorizacion_fuente TO vec_autorizacion_fuente_prueba;
GRANT vec_autorizacion_registro TO vec_autorizacion_registro_prueba;
GRANT vec_autorizacion_motivos_proyector
    TO vec_autorizacion_motivos_proyector_prueba;
GRANT vec_autorizacion_motivos_evaluador
    TO vec_autorizacion_motivos_evaluador_prueba;
SQL

export VEC_POSTGRES_TEST_FUENTE_DSN="postgresql://vec_autorizacion_fuente_prueba:${clave_fuente}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_REGISTRO_DSN="postgresql://vec_autorizacion_registro_prueba:${clave_registro}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_ADMIN_DSN="postgresql://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"

(cd "$raiz" && GOCACHE="$cache_go" go test ./internal/vec/adapters/postgres \
    -run TestIntegracionAutorizacionPostgreSQL -count=1)

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/acl_motivos_v2.sql"
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/integracion_motivos_v2.sql"

docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.down.sql"

# La barrera actual mantiene FOR SHARE sobre el checkpoint hasta el COMMIT del
# consumidor. Una base efimera adicional permite confirmar la carrera y la
# negativa de la migracion descendente sin borrar evidencia para limpiar.
docker exec "$contenedor" createdb --username postgres "$base_concurrencia"
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --set base_concurrencia="$base_concurrencia" <<'SQL'
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'base_concurrencia') \gexec
SELECT format(
    'GRANT CONNECT, CREATE ON DATABASE %I TO vec_autorizacion_propietario',
    :'base_concurrencia'
) \gexec
SELECT format(
    'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_migrador, vec_autorizacion_motivos_proyector, vec_autorizacion_motivos_evaluador',
    :'base_concurrencia'
) \gexec
-- Fuente V1 solo necesita alcanzar esta base para demostrar que sus
-- privilegios V1 no permiten resolver motivos V2.
SELECT format(
    'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_fuente',
    :'base_concurrencia'
) \gexec
SQL
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql" >/dev/null
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql" >/dev/null

# Reproduce la carrera que existiria si el down comprobase en vacio antes de
# esperar el DROP: la publicacion ya ha escrito pero aun no ha confirmado. El
# down corregido espera sus bloqueos, ve la evidencia al despertar y aborta.
docker exec --interactive --env PGPASSWORD="$clave_proyector" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_motivos_proyector_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/mantener_publicacion_para_down_motivos_v2.sql" &
pid_publicacion=$!
publicacion_observada=false
for _ in $(seq 1 40); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base_concurrencia" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE usename = 'vec_autorizacion_motivos_proyector_prueba' AND state = 'active' AND query LIKE 'SELECT pg_sleep(4)%'")
    if [[ "$estado" == "1" ]]; then
        publicacion_observada=true
        break
    fi
    sleep 0.1
done
if [[ "$publicacion_observada" != true ]]; then
    wait "$pid_publicacion" || true
    echo "no se observo la publicacion concurrente contra 000003 down" >&2
    exit 1
fi
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.down.sql" \
    >/dev/null 2>&1; then
    wait "$pid_publicacion" || true
    echo "000003 down borro una publicacion concurrente" >&2
    exit 1
fi
wait "$pid_publicacion"

# El replay exacto confirma que la evidencia sobrevivio sin duplicarse.
docker exec --interactive --env PGPASSWORD="$clave_proyector" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_motivos_proyector_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/preparar_concurrencia_motivos_v2.sql"

# El adaptador Go se prueba sobre la misma evidencia confirmada y antes de la
# retirada. Cada DSN corresponde a una LOGIN distinta; nunca se imprime.
export VEC_POSTGRES_TEST_MOTIVOS_EVALUADOR_DSN="postgresql://vec_autorizacion_motivos_evaluador_prueba:${clave_evaluador}@127.0.0.1:${puerto}/${base_concurrencia}?sslmode=disable"
export VEC_POSTGRES_TEST_MOTIVOS_PROYECTOR_DSN="postgresql://vec_autorizacion_motivos_proyector_prueba:${clave_proyector}@127.0.0.1:${puerto}/${base_concurrencia}?sslmode=disable"
export VEC_POSTGRES_TEST_MOTIVOS_FUENTE_V1_DSN="postgresql://vec_autorizacion_fuente_prueba:${clave_fuente}@127.0.0.1:${puerto}/${base_concurrencia}?sslmode=disable"
(cd "$raiz" && GOCACHE="$cache_go" go test ./internal/vec/adapters/postgres \
    -run '^TestIntegracionMotivosAutorizacionV2PostgreSQL$' -count=1)
unset VEC_POSTGRES_TEST_MOTIVOS_EVALUADOR_DSN
unset VEC_POSTGRES_TEST_MOTIVOS_PROYECTOR_DSN
unset VEC_POSTGRES_TEST_MOTIVOS_FUENTE_V1_DSN

docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/mantener_barrera_motivos_v2.sql" &
pid_barrera=$!
barrera_observada=false
for _ in $(seq 1 40); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base_concurrencia" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE usename = 'vec_autorizacion_migrador_prueba' AND state = 'active' AND query LIKE 'SELECT pg_sleep(4)%'")
    if [[ "$estado" == "1" ]]; then
        barrera_observada=true
        break
    fi
    sleep 0.1
done
if [[ "$barrera_observada" != true ]]; then
    wait "$pid_barrera" || true
    echo "no se observo la transaccion que mantenia la barrera V2" >&2
    exit 1
fi

if docker exec --interactive \
    --env PGPASSWORD="$clave_proyector" \
    --env PGOPTIONS="-c statement_timeout=500ms" \
    "$contenedor" psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_motivos_proyector_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/retirar_concurrencia_motivos_v2.sql" \
    >/dev/null 2>&1; then
    wait "$pid_barrera" || true
    echo "una retirada adelanto al COMMIT protegido por la barrera V2" >&2
    exit 1
fi
wait "$pid_barrera"

docker exec --interactive --env PGPASSWORD="$clave_proyector" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_motivos_proyector_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/retirar_concurrencia_motivos_v2.sql"
docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba \
    --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/verificar_concurrencia_motivos_v2.sql"
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/auditar_concurrencia_motivos_v2.sql"
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base_concurrencia" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.down.sql" \
    >/dev/null 2>&1; then
    echo "000003 down borro evidencia V2" >&2
    exit 1
fi
docker exec "$contenedor" dropdb --username postgres "$base_concurrencia"

# roles_v2_down debe fallar antes de tocar una membresia LOGIN ajena. Las dos
# membresias de prueba siguen vivas en este punto y permiten verificarlo.
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_v2_down elimino roles con membresias pendientes" >&2
    exit 1
fi
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
BEGIN
    IF NOT pg_has_role(
        'vec_autorizacion_motivos_proyector_prueba',
        'vec_autorizacion_motivos_proyector',
        'MEMBER'
    ) OR NOT pg_has_role(
        'vec_autorizacion_motivos_evaluador_prueba',
        'vec_autorizacion_motivos_evaluador',
        'MEMBER'
    ) THEN
        RAISE EXCEPTION 'roles_v2_down toco membresias antes de abortar';
    END IF;
END
$bloque$;
SQL

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_autorizacion_motivos_proyector
    FROM vec_autorizacion_motivos_proyector_prueba;
REVOKE vec_autorizacion_motivos_evaluador
    FROM vec_autorizacion_motivos_evaluador_prueba;
DROP ROLE vec_autorizacion_motivos_proyector_prueba;
DROP ROLE vec_autorizacion_motivos_evaluador_prueba;
SQL

# Cada atributo persistente forma parte del inventario V2. La mutacion y el
# down se ejecutan en una misma conexion/transaccion; el rechazo deja la
# transaccion abortada y el cierre confirma que no se adopto el cambio.
mutaciones_atributos_v2=(
    "ALTER ROLE vec_autorizacion_motivos_proyector PASSWORD 'fixture_no_secreta'"
    "ALTER ROLE vec_autorizacion_motivos_proyector VALID UNTIL '2030-01-01 00:00:00+00'"
    "ALTER ROLE vec_autorizacion_motivos_proyector IN DATABASE $base SET statement_timeout = '1s'"
)
for mutacion in "${mutaciones_atributos_v2[@]}"; do
    if docker exec --interactive "$contenedor" \
        psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "BEGIN; $mutacion" --file=- \
        < "$raiz/deploy/postgresql/autorizacion/roles_v2_down.sql" \
        >/dev/null 2>&1; then
        echo "roles_v2_down acepto atributos persistentes inesperados" >&2
        exit 1
    fi
done
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $atributos_limpios$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid AS rol
         WHERE rol.rolname = 'vec_autorizacion_motivos_proyector'
           AND (
               rol.rolpassword IS NOT NULL
               OR rol.rolvaliduntil IS NOT NULL
               OR EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_db_role_setting AS ajuste
                    WHERE ajuste.setrole = rol.oid
               )
           )
    ) THEN
        RAISE EXCEPTION 'una regresion de atributos V2 dejo estado';
    END IF;
END
$atributos_limpios$;
SQL

# Un rol V2 tambien puede aparecer como otorgante: se le concede ADMIN de un
# grupo ajeno, se usa esa identidad para crear otra arista y se comprueba que
# el down la detecta por grantor sin borrar ninguna de las dos relaciones.
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_autorizacion_otorgado_v2_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT pg_read_all_data TO vec_autorizacion_motivos_proyector
    WITH ADMIN TRUE, INHERIT FALSE, SET TRUE;
SET ROLE vec_autorizacion_motivos_proyector;
GRANT pg_read_all_data TO vec_autorizacion_otorgado_v2_prueba
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
RESET ROLE;
DO $grantor_v2$
DECLARE
    oid_proyector oid := 'vec_autorizacion_motivos_proyector'::regrole::oid;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members
         WHERE grantor = oid_proyector
    ) THEN
        RAISE EXCEPTION 'no se construyo la regresion V2 sobre grantor';
    END IF;
END
$grantor_v2$;
SQL
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_v2_down retiro un rol usado como otorgante" >&2
    exit 1
fi
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $grantor_conservado$
DECLARE
    oid_proyector oid := 'vec_autorizacion_motivos_proyector'::regrole::oid;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members WHERE grantor = oid_proyector
    ) THEN
        RAISE EXCEPTION 'roles_v2_down muto la arista de grantor antes de abortar';
    END IF;
END
$grantor_conservado$;
SET ROLE vec_autorizacion_motivos_proyector;
REVOKE pg_read_all_data FROM vec_autorizacion_otorgado_v2_prueba;
RESET ROLE;
REVOKE pg_read_all_data FROM vec_autorizacion_motivos_proyector;
DROP ROLE vec_autorizacion_otorgado_v2_prueba;
SQL

# La prueba del otorgante debe quedar completamente desmontada antes de la
# carrera; de otro modo una negativa posterior no demostraría la intercalación.
membresias_v2_previas=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS m WHERE m.roleid IN ('vec_autorizacion_motivos_proyector'::regrole::oid,'vec_autorizacion_motivos_evaluador'::regrole::oid) OR m.member IN ('vec_autorizacion_motivos_proyector'::regrole::oid,'vec_autorizacion_motivos_evaluador'::regrole::oid) OR m.grantor IN ('vec_autorizacion_motivos_proyector'::regrole::oid,'vec_autorizacion_motivos_evaluador'::regrole::oid)")
if [[ "$membresias_v2_previas" != "0" ]]; then
    echo "la regresion de grantor V2 no restauro el inventario vacio" >&2
    exit 1
fi

# Conserva los OID para demostrar que tampoco queda una arista huerfana. La
# carrera se lanza desde dos bases y se gobierna con tres barreras, sin sleeps.
oids_roles_v2=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_motivos_proyector', 'vec_autorizacion_motivos_evaluador')")
if [[ ! "$oids_roles_v2" =~ ^[0-9]+,[0-9]+$ ]]; then
    echo "no se pudieron fijar los OID de roles V2" >&2
    exit 1
fi

# El observador de la base VEC conecta antes de congelar pg_database, posee las
# barreras de inicio y fin y las libera al acreditar cada intercalacion exacta.
# La barrera GRANT necesita otra sesion en postgres: los advisory locks estan
# ligados a la base y no servirian como sincronizacion entre ambas conexiones.
docker exec --interactive --env PGAPPNAME=vec_roles_v2_observador "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:inicio', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:fin', 0));
DO $observador$
DECLARE
    limite timestamptz := clock_timestamp() + interval '120 seconds';
    oid_authid oid := 'pg_catalog.pg_authid'::regclass::oid;
    oid_membresias oid := 'pg_catalog.pg_auth_members'::regclass::oid;
    oid_bases oid := 'pg_catalog.pg_database'::regclass::oid;
    encontrado boolean;
BEGIN
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
             WHERE retirada.application_name = 'vec_roles_v2_down_carrera'
               AND retirada.wait_event_type = 'Lock'
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'advisory'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando el down V2 tras el preparador';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:inicio', 0)) THEN
        RAISE EXCEPTION 'no se libero la barrera inicial V2';
    END IF;

    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
              JOIN actividad AS preparador
                ON preparador.application_name = 'vec_roles_v2_bloqueo'
               AND preparador.pid = ANY(pg_catalog.pg_blocking_pids(retirada.pid))
             WHERE retirada.application_name = 'vec_roles_v2_down_carrera'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation = oid_bases
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando los bloqueos exactos del down V2';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS concesion
              JOIN actividad AS retirada
                ON retirada.application_name = 'vec_roles_v2_down_carrera'
               AND retirada.pid = ANY(pg_catalog.pg_blocking_pids(concesion.pid))
             WHERE concesion.application_name = 'vec_roles_v2_grant_carrera'
               AND concesion.wait_event_type = 'Lock'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando que el GRANT V2 dependa del down';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:fin', 0)) THEN
        RAISE EXCEPTION 'no se libero la barrera final V2';
    END IF;
END
$observador$;
SQL
pid_observador_v2=$!

observador_v2_preparado=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS actividad ON actividad.pid = bloqueo.pid WHERE actividad.application_name = 'vec_roles_v2_observador' AND bloqueo.locktype = 'advisory' AND bloqueo.granted")
    if [[ "$estado" == "2" ]]; then
        observador_v2_preparado=true
        break
    fi
    sleep 0.05
done
if [[ "$observador_v2_preparado" != true ]]; then
    echo "no se preconecto el observador de la carrera V2" >&2
    exit 1
fi

# Esta sesion comparte base con el cliente GRANT y retiene su advisory lock
# hasta que roles_v2_down posea ambos catalogos de roles y espere pg_database.
# Se conecta antes de congelar ese catalogo y observa mediante funciones
# directas para no bloquearse al leer pg_authid.
docker exec --interactive --env PGAPPNAME=vec_roles_v2_puerta_grant "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:grant', 0));
DO $puerta_grant$
DECLARE
    limite timestamptz := clock_timestamp() + interval '120 seconds';
    oid_authid oid := 'pg_catalog.pg_authid'::regclass::oid;
    oid_membresias oid := 'pg_catalog.pg_auth_members'::regclass::oid;
    oid_bases oid := 'pg_catalog.pg_database'::regclass::oid;
    encontrado boolean;
BEGIN
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
             WHERE retirada.application_name = 'vec_roles_v2_down_carrera'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation = oid_bases
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando retirar la barrera GRANT V2';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:grant', 0)) THEN
        RAISE EXCEPTION 'no se libero la puerta GRANT V2';
    END IF;
END
$puerta_grant$;
SQL
pid_puerta_grant_v2=$!
puerta_grant_v2_preparada=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname postgres \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS actividad ON actividad.pid = bloqueo.pid WHERE actividad.application_name = 'vec_roles_v2_puerta_grant' AND bloqueo.locktype = 'advisory' AND bloqueo.granted")
    if [[ "$estado" == "1" ]]; then
        puerta_grant_v2_preparada=true
        break
    fi
    sleep 0.05
done
if [[ "$puerta_grant_v2_preparada" != true ]]; then
    wait "$pid_puerta_grant_v2" || true
    echo "no se preparo la puerta interbase del GRANT V2" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v2_bloqueo "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:inicio', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:inicio', 0));
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:fin', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:fin', 0));
ROLLBACK;
SQL
pid_bloqueo_catalogo=$!
preparador_v2_bloqueado=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity AS preparador JOIN pg_catalog.pg_stat_activity AS observador ON observador.application_name = 'vec_roles_v2_observador' AND observador.pid = ANY(pg_catalog.pg_blocking_pids(preparador.pid)) WHERE preparador.application_name = 'vec_roles_v2_bloqueo' AND preparador.wait_event_type = 'Lock'")
    if [[ "$estado" == "1" ]]; then
        preparador_v2_bloqueado=true
        break
    fi
    sleep 0.05
done
if [[ "$preparador_v2_bloqueado" != true ]]; then
    echo "no se preparo la barrera determinista de roles_v2_down" >&2
    exit 1
fi

# El cliente GRANT tambien se preconecta mientras los catalogos siguen libres.
# Su primera consulta queda retenida por la puerta de su propia base hasta que
# el down posea exactamente los dos bloqueos de roles.
if ! kill -0 "$pid_observador_v2" 2>/dev/null; then
    wait "$pid_observador_v2" || true
    echo "el observador V2 termino antes de preconectar el GRANT" >&2
    exit 1
fi
docker exec --interactive --env PGAPPNAME=vec_roles_v2_grant_carrera "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    --username postgres \
    --dbname postgres >/dev/null 2>"$error_grant_v2" <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v2:grant', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v2:grant', 0));
GRANT vec_autorizacion_motivos_evaluador
    TO vec_autorizacion_migrador_prueba;
SQL
pid_grant=$!
grant_v2_preconectado=false
for _ in $(seq 1 200); do
    if ! kill -0 "$pid_grant" 2>/dev/null; then
        wait "$pid_grant" || true
        echo "el cliente GRANT V2 termino antes de alcanzar su barrera" >&2
        sed -n '1,20p' "$error_grant_v2" >&2
        exit 1
    fi
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS concesion ON concesion.pid = bloqueo.pid WHERE concesion.application_name = 'vec_roles_v2_grant_carrera' AND concesion.wait_event_type = 'Lock' AND bloqueo.locktype = 'advisory' AND NOT bloqueo.granted")
    if [[ "$estado" == "1" ]]; then
        grant_v2_preconectado=true
        break
    fi
    sleep 0.05
done
if [[ "$grant_v2_preconectado" != true ]]; then
    echo "no se preconecto el GRANT de la carrera V2" >&2
    sed -n '1,20p' "$error_grant_v2" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v2_down_carrera "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_down.sql" &
pid_roles_down=$!

if ! wait "$pid_observador_v2"; then
    echo "fallo el observador preconectado de la carrera V2" >&2
    exit 1
fi
if ! wait "$pid_puerta_grant_v2"; then
    echo "fallo la puerta interbase del GRANT V2" >&2
    exit 1
fi
wait "$pid_bloqueo_catalogo"
wait "$pid_roles_down"
if wait "$pid_grant"; then
    echo "un GRANT concurrente sobrevivio al DROP ROLE" >&2
    exit 1
fi
if ! grep -Fq 'ERROR:  42704:' "$error_grant_v2" || \
   ! grep -Fq 'role "vec_autorizacion_motivos_evaluador" does not exist' \
       "$error_grant_v2"; then
    echo "el GRANT V2 no termino por la ausencia exacta del rol retirado" >&2
    sed -n '1,20p' "$error_grant_v2" >&2
    exit 1
fi

restos_roles_v2=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia WHERE membresia.roleid IN ($oids_roles_v2) OR membresia.member IN ($oids_roles_v2) OR membresia.grantor IN ($oids_roles_v2)")
huerfanas=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia LEFT JOIN pg_catalog.pg_authid AS grupo ON grupo.oid = membresia.roleid LEFT JOIN pg_catalog.pg_authid AS miembro ON miembro.oid = membresia.member LEFT JOIN pg_catalog.pg_authid AS otorgante ON otorgante.oid = membresia.grantor WHERE grupo.oid IS NULL OR miembro.oid IS NULL OR otorgante.oid IS NULL")
roles_v2_restantes=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_motivos_proyector', 'vec_autorizacion_motivos_evaluador')")
if [[ "$restos_roles_v2" != "0" || "$huerfanas" != "0" || "$roles_v2_restantes" != "0" ]]; then
    echo "roles_v2_down dejo roles o membresias huerfanas" >&2
    exit 1
fi

docker exec --interactive --env PGPASSWORD="$clave_migrador" "$contenedor" \
    psql --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_autorizacion_migrador_prueba --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.down.sql"

# El down V1 debe inventariar antes de mutar: las tres cuentas LOGIN de prueba
# siguen siendo miembros y obligan a abortar conservando todas las aristas.
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down elimino roles V1 con membresias pendientes" >&2
    exit 1
fi
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bloque$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = 'vec_autorizacion'
    ) THEN
        RAISE EXCEPTION 'la migracion descendente no retiro el esquema';
    END IF;
    IF NOT pg_has_role(
        'vec_autorizacion_fuente_prueba',
        'vec_autorizacion_fuente',
        'MEMBER'
    ) OR NOT pg_has_role(
        'vec_autorizacion_registro_prueba',
        'vec_autorizacion_registro',
        'MEMBER'
    ) OR NOT pg_has_role(
        'vec_autorizacion_migrador_prueba',
        'vec_autorizacion_migrador',
        'MEMBER'
    ) THEN
        RAISE EXCEPTION 'roles_down toco membresias V1 antes de abortar';
    END IF;
END
$bloque$;
REVOKE vec_autorizacion_fuente FROM vec_autorizacion_fuente_prueba;
REVOKE vec_autorizacion_registro FROM vec_autorizacion_registro_prueba;
REVOKE vec_autorizacion_migrador FROM vec_autorizacion_migrador_prueba;
DROP ROLE vec_autorizacion_fuente_prueba;
DROP ROLE vec_autorizacion_registro_prueba;
SQL

# V1 tambien rechaza roles con atributos o ajustes que no creo su bootstrap.
# Cada mutacion comparte transaccion con el down: el error debe revertirla y
# conservar tanto los cuatro roles como su relacion estructural original.
mutaciones_atributos_v1=(
    "ALTER ROLE vec_autorizacion_registro NOINHERIT"
    "ALTER ROLE vec_autorizacion_registro PASSWORD 'fixture_no_secreta'"
    "ALTER ROLE vec_autorizacion_registro IN DATABASE $base SET statement_timeout = '1s'"
)
for mutacion in "${mutaciones_atributos_v1[@]}"; do
    if docker exec --interactive "$contenedor" \
        psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "BEGIN; $mutacion" --file=- \
        < "$raiz/deploy/postgresql/autorizacion/roles_down.sql" \
        >/dev/null 2>&1; then
        echo "roles_down V1 acepto atributos persistentes inesperados" >&2
        exit 1
    fi
done
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $atributos_v1_limpios$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid AS rol
         WHERE rol.rolname = 'vec_autorizacion_registro'
           AND (
               rol.rolinherit IS FALSE
               OR rol.rolpassword IS NOT NULL
               OR EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_db_role_setting AS ajuste
                    WHERE ajuste.setrole = rol.oid
               )
           )
    ) THEN
        RAISE EXCEPTION 'una regresion de atributos V1 dejo estado';
    END IF;
END
$atributos_v1_limpios$;
SQL

# Una concesion TEMPORARY adicional es una ACL ajena, no algo que el down pueda
# borrar con REVOKE ALL. Debe abortar conservando tanto esa concesion como la
# arista estructural; tras retirarla expresamente vuelve el inventario exacto.
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
GRANT TEMPORARY ON DATABASE vec_autorizacion_prueba
    TO vec_autorizacion_registro;
SQL
if docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down V1 revoco una ACL TEMPORARY no inventariada" >&2
    exit 1
fi
docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $acl_conservada$
BEGIN
    IF NOT pg_catalog.has_database_privilege(
        'vec_autorizacion_registro', current_database(), 'TEMPORARY'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
         WHERE membresia.roleid = 'vec_autorizacion_propietario'::regrole::oid
           AND membresia.member = 'vec_autorizacion_migrador'::regrole::oid
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'roles_down V1 muto estado antes de rechazar la ACL';
    END IF;
END
$acl_conservada$;
REVOKE TEMPORARY ON DATABASE vec_autorizacion_prueba
    FROM vec_autorizacion_registro;
SQL

# Conserva los cuatro OID V1. La carrera usa barreras deterministas: acredita
# que el down retiene ambos catalogos de roles y que el GRANT espera a su PID.
oids_roles_v1=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_propietario', 'vec_autorizacion_migrador', 'vec_autorizacion_fuente', 'vec_autorizacion_registro')")
if [[ ! "$oids_roles_v1" =~ ^[0-9]+,[0-9]+,[0-9]+,[0-9]+$ ]]; then
    echo "no se pudieron fijar los OID de roles V1" >&2
    exit 1
fi

# El observador de VEC conserva solo las barreras de inicio y fin. La puerta
# del GRANT vive en postgres porque los advisory locks no cruzan bases.
docker exec --interactive --env PGAPPNAME=vec_roles_v1_observador "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:inicio', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:fin', 0));
DO $observador$
DECLARE
    limite timestamptz := clock_timestamp() + interval '120 seconds';
    oid_authid oid := 'pg_catalog.pg_authid'::regclass::oid;
    oid_membresias oid := 'pg_catalog.pg_auth_members'::regclass::oid;
    oid_bases oid := 'pg_catalog.pg_database'::regclass::oid;
    encontrado boolean;
BEGIN
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
             WHERE retirada.application_name = 'vec_roles_v1_down_carrera'
               AND retirada.wait_event_type = 'Lock'
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'advisory'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando el down V1 tras el preparador';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:inicio', 0)) THEN
        RAISE EXCEPTION 'no se libero la barrera inicial V1';
    END IF;

    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
              JOIN actividad AS preparador
                ON preparador.application_name = 'vec_roles_v1_bloqueo'
               AND preparador.pid = ANY(pg_catalog.pg_blocking_pids(retirada.pid))
             WHERE retirada.application_name = 'vec_roles_v1_down_carrera'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation = oid_bases
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando los bloqueos exactos del down V1';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS concesion
              JOIN actividad AS retirada
                ON retirada.application_name = 'vec_roles_v1_down_carrera'
               AND retirada.pid = ANY(pg_catalog.pg_blocking_pids(concesion.pid))
             WHERE concesion.application_name = 'vec_roles_v1_grant_carrera'
               AND concesion.wait_event_type = 'Lock'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando que el GRANT V1 dependa del down';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:fin', 0)) THEN
        RAISE EXCEPTION 'no se libero la barrera final V1';
    END IF;
END
$observador$;
SQL
pid_observador_v1=$!

observador_v1_preparado=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS actividad ON actividad.pid = bloqueo.pid WHERE actividad.application_name = 'vec_roles_v1_observador' AND bloqueo.locktype = 'advisory' AND bloqueo.granted")
    if [[ "$estado" == "2" ]]; then
        observador_v1_preparado=true
        break
    fi
    sleep 0.05
done
if [[ "$observador_v1_preparado" != true ]]; then
    echo "no se preconecto el observador de la carrera V1" >&2
    exit 1
fi

# La sesion se establece en postgres antes de bloquear pg_database. Retiene al
# cliente GRANT hasta acreditar que roles_down ya posee los catalogos globales
# de roles y espera el catalogo de bases; despues el GRANT solo puede continuar
# contra el down, nunca colarse entre su inventario y sus DROP ROLE.
docker exec --interactive --env PGAPPNAME=vec_roles_v1_puerta_grant "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:grant', 0));
DO $puerta_grant$
DECLARE
    limite timestamptz := clock_timestamp() + interval '120 seconds';
    oid_authid oid := 'pg_catalog.pg_authid'::regclass::oid;
    oid_membresias oid := 'pg_catalog.pg_auth_members'::regclass::oid;
    oid_bases oid := 'pg_catalog.pg_database'::regclass::oid;
    encontrado boolean;
BEGIN
    LOOP
        PERFORM pg_catalog.pg_stat_clear_snapshot();
        WITH actividad AS MATERIALIZED (
            SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
        )
        SELECT EXISTS (
            SELECT 1
              FROM actividad AS retirada
             WHERE retirada.application_name = 'vec_roles_v1_down_carrera'
               AND (
                   SELECT count(*)
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation IN (oid_authid, oid_membresias)
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND bloqueo.granted
               ) = 2
               AND EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_lock_status() AS bloqueo
                    WHERE bloqueo.pid = retirada.pid
                      AND bloqueo.locktype = 'relation'
                      AND bloqueo.relation = oid_bases
                      AND bloqueo.mode = 'AccessExclusiveLock'
                      AND NOT bloqueo.granted
               )
        ) INTO encontrado;
        EXIT WHEN encontrado;
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION 'timeout esperando retirar la barrera GRANT V1';
        END IF;
        PERFORM pg_sleep(0.05);
    END LOOP;
    IF NOT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:grant', 0)) THEN
        RAISE EXCEPTION 'no se libero la puerta GRANT V1';
    END IF;
END
$puerta_grant$;
SQL
pid_puerta_grant_v1=$!
puerta_grant_v1_preparada=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname postgres \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS actividad ON actividad.pid = bloqueo.pid WHERE actividad.application_name = 'vec_roles_v1_puerta_grant' AND bloqueo.locktype = 'advisory' AND bloqueo.granted")
    if [[ "$estado" == "1" ]]; then
        puerta_grant_v1_preparada=true
        break
    fi
    sleep 0.05
done
if [[ "$puerta_grant_v1_preparada" != true ]]; then
    wait "$pid_puerta_grant_v1" || true
    echo "no se preparo la puerta interbase del GRANT V1" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v1_bloqueo "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" >/dev/null <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:roles_down:v2', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:inicio', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:inicio', 0));
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:roles_down:v2', 0));
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:fin', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:fin', 0));
ROLLBACK;
SQL
pid_bloqueo_catalogo_v1=$!
preparador_v1_bloqueado=false
for _ in $(seq 1 100); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity AS preparador JOIN pg_catalog.pg_stat_activity AS observador ON observador.application_name = 'vec_roles_v1_observador' AND observador.pid = ANY(pg_catalog.pg_blocking_pids(preparador.pid)) WHERE preparador.application_name = 'vec_roles_v1_bloqueo' AND preparador.wait_event_type = 'Lock'")
    if [[ "$estado" == "1" ]]; then
        preparador_v1_bloqueado=true
        break
    fi
    sleep 0.05
done
if [[ "$preparador_v1_bloqueado" != true ]]; then
    echo "no se preparo la barrera determinista de roles_down V1" >&2
    exit 1
fi

if ! kill -0 "$pid_observador_v1" 2>/dev/null; then
    wait "$pid_observador_v1" || true
    echo "el observador V1 termino antes de preconectar el GRANT" >&2
    exit 1
fi
docker exec --interactive --env PGAPPNAME=vec_roles_v1_grant_carrera "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    --username postgres \
    --dbname postgres >/dev/null 2>"$error_grant_v1" <<'SQL' &
SELECT pg_advisory_lock(hashtextextended('vec_autorizacion:prueba:v1:grant', 0));
SELECT pg_advisory_unlock(hashtextextended('vec_autorizacion:prueba:v1:grant', 0));
GRANT vec_autorizacion_registro TO vec_autorizacion_migrador_prueba;
SQL
pid_grant_v1=$!
grant_v1_preconectado=false
for _ in $(seq 1 200); do
    if ! kill -0 "$pid_grant_v1" 2>/dev/null; then
        wait "$pid_grant_v1" || true
        echo "el cliente GRANT V1 termino antes de alcanzar su barrera" >&2
        sed -n '1,20p' "$error_grant_v1" >&2
        exit 1
    fi
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_locks AS bloqueo JOIN pg_catalog.pg_stat_activity AS concesion ON concesion.pid = bloqueo.pid WHERE concesion.application_name = 'vec_roles_v1_grant_carrera' AND concesion.wait_event_type = 'Lock' AND bloqueo.locktype = 'advisory' AND NOT bloqueo.granted")
    if [[ "$estado" == "1" ]]; then
        grant_v1_preconectado=true
        break
    fi
    sleep 0.05
done
if [[ "$grant_v1_preconectado" != true ]]; then
    echo "no se preconecto el GRANT de la carrera V1" >&2
    sed -n '1,20p' "$error_grant_v1" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v1_down_carrera "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_down.sql" &
pid_roles_down_v1=$!

if ! wait "$pid_observador_v1"; then
    echo "fallo el observador preconectado de la carrera V1" >&2
    exit 1
fi
if ! wait "$pid_puerta_grant_v1"; then
    echo "fallo la puerta interbase del GRANT V1" >&2
    exit 1
fi
wait "$pid_bloqueo_catalogo_v1"
wait "$pid_roles_down_v1"
if wait "$pid_grant_v1"; then
    echo "un GRANT V1 concurrente sobrevivio al DROP ROLE" >&2
    exit 1
fi
if ! grep -Fq 'ERROR:  42704:' "$error_grant_v1" || \
   ! grep -Fq 'role "vec_autorizacion_registro" does not exist' \
       "$error_grant_v1"; then
    echo "el GRANT V1 no termino por la ausencia exacta del rol retirado" >&2
    sed -n '1,20p' "$error_grant_v1" >&2
    exit 1
fi

restos_roles_v1=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia WHERE membresia.roleid IN ($oids_roles_v1) OR membresia.member IN ($oids_roles_v1) OR membresia.grantor IN ($oids_roles_v1)")
huerfanas_v1=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia LEFT JOIN pg_catalog.pg_authid AS grupo ON grupo.oid = membresia.roleid LEFT JOIN pg_catalog.pg_authid AS miembro ON miembro.oid = membresia.member LEFT JOIN pg_catalog.pg_authid AS otorgante ON otorgante.oid = membresia.grantor WHERE grupo.oid IS NULL OR miembro.oid IS NULL OR otorgante.oid IS NULL")
roles_v1_restantes=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_propietario', 'vec_autorizacion_migrador', 'vec_autorizacion_fuente', 'vec_autorizacion_registro')")
if [[ "$restos_roles_v1" != "0" || "$huerfanas_v1" != "0" || "$roles_v1_restantes" != "0" ]]; then
    echo "roles_down V1 dejo roles o membresias huerfanas" >&2
    exit 1
fi

docker exec --interactive "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname postgres \
    --command 'DROP ROLE vec_autorizacion_migrador_prueba'
