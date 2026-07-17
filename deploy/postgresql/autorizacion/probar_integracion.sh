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
base_concurrencia=vec_motivos_v2_concurrencia

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

puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efimero de PostgreSQL" >&2
    exit 1
fi

export VEC_POSTGRES_TEST_FUENTE_DSN="postgresql://vec_autorizacion_fuente_prueba:${clave_fuente}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_REGISTRO_DSN="postgresql://vec_autorizacion_registro_prueba:${clave_registro}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_ADMIN_DSN="postgresql://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"

(cd "$raiz" && go test ./internal/vec/adapters/postgres -run TestIntegracionAutorizacionPostgreSQL -count=1)

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

# Conserva los OID para demostrar que tampoco queda una arista huerfana. La
# carrera se lanza desde dos bases: down en la dedicada y GRANT en postgres.
oids_roles_v2=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_autorizacion_motivos_proyector', 'vec_autorizacion_motivos_evaluador')")
if [[ ! "$oids_roles_v2" =~ ^[0-9]+,[0-9]+$ ]]; then
    echo "no se pudieron fijar los OID de roles V2" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v2_bloqueo "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/mantener_bloqueo_catalogo_roles_motivos_v2.sql" &
pid_bloqueo_catalogo=$!
bloqueo_advisory_observado=false
for _ in $(seq 1 40); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = 'vec_roles_v2_bloqueo' AND state = 'active' AND query LIKE 'SELECT pg_sleep(4)%'")
    if [[ "$estado" == "1" ]]; then
        bloqueo_advisory_observado=true
        break
    fi
    sleep 0.1
done
if [[ "$bloqueo_advisory_observado" != true ]]; then
    wait "$pid_bloqueo_catalogo" || true
    echo "no se observo el bloqueo preparatorio de roles_v2_down" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_roles_v2_down_carrera "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion/roles_v2_down.sql" &
pid_roles_down=$!
docker exec --interactive --env PGAPPNAME=vec_roles_v2_grant_carrera "$contenedor" \
    psql --set ON_ERROR_STOP=1 --username postgres --dbname postgres \
    < "$raiz/deploy/postgresql/autorizacion/pruebas_sql/conceder_membresia_concurrente_motivos_v2.sql" \
    >/dev/null 2>&1 &
pid_grant=$!

wait "$pid_bloqueo_catalogo"
wait "$pid_roles_down"
if wait "$pid_grant"; then
    echo "un GRANT concurrente sobrevivio al DROP ROLE" >&2
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
