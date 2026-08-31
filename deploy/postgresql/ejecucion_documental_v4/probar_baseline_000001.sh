#!/usr/bin/env bash
set -euo pipefail

if (( $# != 3 )); then
    echo "uso interno: probar_baseline_000001.sh CONTENEDOR BASE RAIZ" >&2
    exit 64
fi
contenedor=$1
base=$2
raiz=$3
if [[ ! "$contenedor" =~ ^vec-ejecucion-v4-pg-[a-zA-Z0-9_.-]+$ \
      || ! "$base" =~ ^[a-zA-Z0-9_]+$ \
      || ! -e "$raiz/.git" \
      || ! -f "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" ]]; then
    echo "coordinacion baseline 000001 no valida" >&2
    exit 64
fi
version=$(docker exec "$contenedor" psql --no-psqlrc --tuples-only \
    --no-align --username postgres --dbname "$base" \
    --command 'SHOW server_version_num')
if [[ "$version" != "180004" ]]; then
    echo "baseline 000001 requiere PostgreSQL 18.4, no ${version}" >&2
    exit 1
fi
salida_observador=$(mktemp)
salida_grant=$(mktemp)
limpiar() { rm -f "$salida_observador" "$salida_grant"; }
trap limpiar EXIT INT TERM

psql_admin() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" "$@"
}

rechazar_roles_down_con_mutacion() {
    local descripcion=$1
    local mutacion=$2
    # Mutacion y down comparten transaccion: el rechazo debe revertir el caso.
    if psql_admin --command "BEGIN; ${mutacion}" --file=- \
        < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
        >/dev/null 2>&1; then
        echo "roles_down V4 acepto ${descripcion}" >&2
        exit 1
    fi
}

# Membresia entrante: debe abortar antes de revocar autoridad.
psql_admin <<'SQL'
CREATE ROLE vec_v4_intruso_propietario_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_v4_intruso_propietario_prueba;
SQL
if psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto un LOGIN miembro del propietario" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $conservado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname =
                   'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_v4_intruso_propietario_prueba'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto autoridad antes de fallar';
    END IF;
END
$conservado$;
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_v4_intruso_propietario_prueba;
DROP ROLE vec_v4_intruso_propietario_prueba;
SQL

# Membresia saliente: un grupo ajeno es una dependencia compartida.
psql_admin <<'SQL'
CREATE ROLE vec_v4_grupo_ajeno_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_v4_grupo_ajeno_prueba
    TO vec_ejecucion_documental_v4_emisor_capacidad;
SQL
if psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto una membresia V4 saliente" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $saliente_conservada$
BEGIN
    IF NOT pg_has_role(
        'vec_ejecucion_documental_v4_emisor_capacidad',
        'vec_v4_grupo_ajeno_prueba',
        'MEMBER'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto una membresia saliente antes de fallar';
    END IF;
END
$saliente_conservada$;
REVOKE vec_v4_grupo_ajeno_prueba
    FROM vec_ejecucion_documental_v4_emisor_capacidad;
DROP ROLE vec_v4_grupo_ajeno_prueba;
SQL

# Opciones y otorgante forman parte de la arista estructural exacta.
psql_admin <<'SQL'
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_ejecucion_documental_v4_migrador
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SQL
if psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto opciones estructurales V4 alteradas" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $opciones_conservadas$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS TRUE
           AND membresia.set_option IS TRUE
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto la arista estructural antes de fallar';
    END IF;
END
$opciones_conservadas$;
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_ejecucion_documental_v4_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL

psql_admin <<'SQL'
DO $grantor_gobernado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_roles AS otorgante ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND otorgante.oid = 10 AND otorgante.rolsuper
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'la arista estructural no conserva el grantor bootstrap';
    END IF;
END
$grantor_gobernado$;
SQL

# Una arista ajena con rol V4 solo como grantor tambien debe conservarse.
psql_admin <<'SQL'
CREATE ROLE vec_v4_grupo_grantor_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_v4_miembro_grantor_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_v4_grupo_grantor_prueba TO vec_v4_miembro_grantor_prueba;
UPDATE pg_catalog.pg_auth_members
   SET grantor = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_ejecucion_documental_v4_emisor_capacidad'
   )
 WHERE roleid = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_grupo_grantor_prueba'
   )
   AND member = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_miembro_grantor_prueba'
   );
SQL
if psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto un rol V4 usado solo como grantor" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $grantor_solo_conservado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_roles AS otorgante ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_v4_grupo_grantor_prueba'
           AND miembro.rolname = 'vec_v4_miembro_grantor_prueba'
           AND otorgante.rolname =
                   'vec_ejecucion_documental_v4_emisor_capacidad'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar el grantor V4';
    END IF;
END
$grantor_solo_conservado$;
UPDATE pg_catalog.pg_auth_members SET grantor = 10
 WHERE roleid = (SELECT oid FROM pg_catalog.pg_roles
                  WHERE rolname = 'vec_v4_grupo_grantor_prueba')
   AND member = (SELECT oid FROM pg_catalog.pg_roles
                  WHERE rolname = 'vec_v4_miembro_grantor_prueba');
REVOKE vec_v4_grupo_grantor_prueba FROM vec_v4_miembro_grantor_prueba;
DROP ROLE vec_v4_miembro_grantor_prueba;
DROP ROLE vec_v4_grupo_grantor_prueba;
SQL

# Matriz hostil de atributos, guarda, ACL, DDL y privilegios por defecto.
psql_admin <<'SQL'
CREATE ROLE vec_v4_guardia_propietario_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE SCHEMA vec_v4_acl_externa_prueba;
CREATE TABLE vec_v4_acl_externa_prueba.documento (identificador bigint);
SQL
mutaciones=(
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado PASSWORD 'fixture_no_secreta';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado VALID UNTIL '2027-01-01 00:00:00+00';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado SET statement_timeout = '1s';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado IN DATABASE ${base} SET statement_timeout = '1s';"
)
for mutacion in "${mutaciones[@]}"; do
    rechazar_roles_down_con_mutacion "atributos de rol ajenos al bootstrap" "$mutacion"
done
rechazar_roles_down_con_mutacion "otro propietario del esquema guarda" \
    "ALTER SCHEMA vec_ejecucion_documental_v4_guardia OWNER TO vec_v4_guardia_propietario_prueba;"
rechazar_roles_down_con_mutacion "otro propietario de la funcion guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() OWNER TO vec_v4_guardia_propietario_prueba;"
rechazar_roles_down_con_mutacion "USAGE publico en la guarda" \
    "GRANT USAGE ON SCHEMA vec_ejecucion_documental_v4_guardia TO PUBLIC;"
rechazar_roles_down_con_mutacion "EXECUTE publico en la guarda" \
    "GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() TO PUBLIC;"
rechazar_roles_down_con_mutacion "proconfig manipulado en la guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() SET search_path = pg_catalog;"
rechazar_roles_down_con_mutacion "atributos manipulados en la funcion guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() SECURITY INVOKER;"
rechazar_roles_down_con_mutacion "etiquetas manipuladas en la guarda" \
    "DROP EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos; CREATE EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos ON ddl_command_end WHEN TAG IN ('CREATE TABLE') EXECUTE FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();"
rechazar_roles_down_con_mutacion "una funcion adicional en la guarda" \
    "CREATE FUNCTION vec_ejecucion_documental_v4_guardia.extra_prueba() RETURNS event_trigger LANGUAGE plpgsql AS 'BEGIN END;'; REVOKE ALL ON FUNCTION vec_ejecucion_documental_v4_guardia.extra_prueba() FROM PUBLIC;"
rechazar_roles_down_con_mutacion "un uso adicional de la funcion guarda" \
    "CREATE EVENT TRIGGER vec_ejecucion_documental_v4_extra_prueba ON ddl_command_end WHEN TAG IN ('CREATE TABLE') EXECUTE FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();"
rechazar_roles_down_con_mutacion "TEMPORARY adicional en la base" \
    "GRANT TEMPORARY ON DATABASE ${base} TO vec_ejecucion_documental_v4_emisor_capacidad;"
rechazar_roles_down_con_mutacion "CREATE adicional sobre public" \
    "GRANT CREATE ON SCHEMA public TO vec_ejecucion_documental_v4_propietario;"
rechazar_roles_down_con_mutacion "HMAC con opcion de concesion" \
    "GRANT EXECUTE ON FUNCTION public.hmac(bytea,bytea,text) TO vec_ejecucion_documental_v4_propietario WITH GRANT OPTION;"
rechazar_roles_down_con_mutacion "una ACL de tabla externa" \
    "GRANT SELECT ON vec_v4_acl_externa_prueba.documento TO vec_ejecucion_documental_v4_emisor_capacidad;"
rechazar_roles_down_con_mutacion "un beneficiario adicional en defaults" \
    "ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario GRANT EXECUTE ON FUNCTIONS TO vec_ejecucion_documental_v4_emisor_capacidad;"
rechazar_roles_down_con_mutacion "otra clase de default ACL" \
    "ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario GRANT USAGE ON SCHEMAS TO vec_ejecucion_documental_v4_emisor_capacidad;"

psql_admin <<'SQL'
DO $estado_canonico_tras_negativas$
DECLARE oid_ejecutor oid;
BEGIN
    SELECT oid INTO STRICT oid_ejecutor
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_ejecutor_atestado'
       AND rolpassword IS NULL AND rolvaliduntil IS NULL;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole = oid_ejecutor
    ) OR pg_catalog.to_regprocedure(
        'vec_ejecucion_documental_v4_guardia.extra_prueba()'
    ) IS NOT NULL OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_extra_prueba'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
           AND cardinality(evttags) = 12
    ) THEN
        RAISE EXCEPTION 'una prueba negativa V4 no revirtio su transaccion';
    END IF;
END
$estado_canonico_tras_negativas$;
DROP TABLE vec_v4_acl_externa_prueba.documento;
DROP SCHEMA vec_v4_acl_externa_prueba;
DROP ROLE vec_v4_guardia_propietario_prueba;
SQL

# Carrera GRANT/roles_down. El observador se preconecta y usa las funciones
# de estadisticas directas para no bloquearse al resolver pg_authid.
psql_admin <<'SQL'
CREATE ROLE vec_v4_grant_concurrente_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SQL
oids=$(psql_admin --tuples-only --no-align --command \
    "SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_ejecucion_documental_v4_propietario', 'vec_ejecucion_documental_v4_migrador', 'vec_ejecucion_documental_v4_emisor_capacidad', 'vec_ejecucion_documental_v4_ejecutor_atestado')")
if [[ ! "$oids" =~ ^[0-9]+,[0-9]+,[0-9]+,[0-9]+$ ]]; then
    echo "no se pudieron fijar los OID de roles V4" >&2
    exit 1
fi
docker exec --interactive --env PGAPPNAME=vec_v4_roles_down_bloqueo \
    "$contenedor" psql --no-psqlrc --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL' &
SELECT pg_advisory_lock(
    hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);
SELECT pg_sleep(4);
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_unlock(
    hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);
SELECT pg_sleep(4);
ROLLBACK;
SQL
pid_bloqueo=$!
bloqueo_observado=false
for _ in $(seq 1 40); do
    estado=$(psql_admin --tuples-only --no-align --command \
        "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = 'vec_v4_roles_down_bloqueo' AND state = 'active' AND query LIKE 'SELECT pg_sleep(4)%'")
    if [[ "$estado" == "1" ]]; then bloqueo_observado=true; break; fi
    sleep 0.1
done
if [[ "$bloqueo_observado" != true ]]; then
    wait "$pid_bloqueo" || true
    echo "no se observo el bloqueo preparatorio de roles_down V4" >&2
    exit 1
fi
docker exec --interactive --env PGAPPNAME=vec_v4_roles_down_carrera \
    "$contenedor" psql --no-psqlrc --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" &
pid_down=$!
docker exec --interactive --env PGAPPNAME=vec_v4_grant_carrera \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose --username postgres --dbname postgres \
    >"$salida_grant" 2>&1 <<'SQL' &
SELECT pg_sleep(5);
GRANT vec_ejecucion_documental_v4_ejecutor_atestado
    TO vec_v4_grant_concurrente_prueba;
SQL
pid_grant=$!
docker exec --interactive --env PGAPPNAME=vec_v4_observador_carrera \
    "$contenedor" psql -X --quiet --tuples-only --no-align \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    >"$salida_observador" <<'SQL' &
SELECT pg_sleep(7);
WITH actividad AS MATERIALIZED (
    SELECT * FROM pg_catalog.pg_stat_get_activity(NULL::integer)
)
SELECT count(*)
  FROM actividad AS concesion
  JOIN actividad AS retirada
    ON retirada.application_name = 'vec_v4_roles_down_carrera'
   AND retirada.pid = ANY (pg_catalog.pg_blocking_pids(concesion.pid))
 WHERE concesion.application_name = 'vec_v4_grant_carrera'
   AND concesion.state = 'active'
   AND concesion.wait_event_type = 'Lock'
   AND (
       SELECT count(*) FROM pg_catalog.pg_lock_status() AS bloqueo
        WHERE bloqueo.pid = retirada.pid
          AND bloqueo.locktype = 'relation'
          AND bloqueo.relation IN (
              'pg_catalog.pg_authid'::regclass::oid,
              'pg_catalog.pg_auth_members'::regclass::oid
          )
          AND bloqueo.mode = 'AccessExclusiveLock' AND bloqueo.granted
   ) = 2;
SQL
pid_observador=$!
if ! wait "$pid_observador" || ! grep -qx '1' "$salida_observador"; then
    wait "$pid_bloqueo" || true; wait "$pid_down" || true; wait "$pid_grant" || true
    echo "fallo el observador preconectado de la carrera roles_down V4" >&2
    exit 1
fi
wait "$pid_bloqueo"
wait "$pid_down"
grant_rechazado=false
if ! wait "$pid_grant"; then grant_rechazado=true; fi
if [[ "$grant_rechazado" != true ]] \
   || ! grep -Eq 'ERROR: +42704:' "$salida_grant" \
   || ! grep -Fq 'vec_ejecucion_documental_v4_ejecutor_atestado' "$salida_grant"; then
    echo "el GRANT V4 no acredito SQLSTATE 42704 y el rol retirado" >&2
    sed -n '1,20p' "$salida_grant" >&2
    exit 1
fi
restos=$(psql_admin --tuples-only --no-align --command \
    "SELECT count(*) FROM pg_catalog.pg_auth_members WHERE roleid IN ($oids) OR member IN ($oids) OR grantor IN ($oids)")
huerfanas=$(psql_admin --tuples-only --no-align --command \
    "SELECT count(*) FROM pg_catalog.pg_auth_members AS m LEFT JOIN pg_catalog.pg_authid AS r ON r.oid=m.roleid LEFT JOIN pg_catalog.pg_authid AS u ON u.oid=m.member LEFT JOIN pg_catalog.pg_authid AS g ON g.oid=m.grantor WHERE r.oid IS NULL OR u.oid IS NULL OR g.oid IS NULL")
roles=$(psql_admin --tuples-only --no-align --command \
    "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'")
if [[ "$restos" != 0 || "$huerfanas" != 0 || "$roles" != 0 ]]; then
    echo "roles_down V4 dejo roles o membresias huerfanas" >&2
    exit 1
fi
psql_admin --command 'DROP ROLE vec_v4_grant_concurrente_prueba'

# El desmontaje no reabre pgcrypto a PUBLIC.
psql_admin <<'SQL'
DO $cierre_persistente$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS funcion ON funcion.oid = dependencia.objid
          CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
              funcion.proacl, pg_catalog.acldefault('f', funcion.proowner)
          )) AS privilegio
         WHERE extension.extname = 'pgcrypto' AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down reabrio pgcrypto a PUBLIC';
    END IF;
END
$cierre_persistente$;
SQL

# Un DBA nominativo alternativo debe poseer guarda y completar el ciclo.
psql_admin <<'SQL'
CREATE ROLE vec_v4_dba_alternativo_prueba NOLOGIN SUPERUSER;
SQL
psql_admin --command 'SET SESSION AUTHORIZATION vec_v4_dba_alternativo_prueba' \
    --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql"
psql_admin <<'SQL'
DO $instalacion_dba_alternativo$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_roles AS propietario ON propietario.oid = espacio.nspowner
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4_guardia'
           AND propietario.rolname = 'vec_v4_dba_alternativo_prueba'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_roles AS propietario ON propietario.oid = funcion.proowner
         WHERE funcion.oid = pg_catalog.to_regprocedure(
                   'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
               ) AND propietario.rolname = 'vec_v4_dba_alternativo_prueba'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_authid AS otorgante ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND otorgante.oid = 10 AND otorgante.rolsuper
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'el DBA alternativo no separo propiedad y grantor V4';
    END IF;
END
$instalacion_dba_alternativo$;
SQL
psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.up.sql"
psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql"
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
psql_admin --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.down.sql"
psql_admin --command 'SET SESSION AUTHORIZATION vec_v4_dba_alternativo_prueba' \
    --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql"
psql_admin <<'SQL'
DO $retirada_dba_alternativo$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR pg_catalog.to_regnamespace(
        'vec_ejecucion_documental_v4_guardia'
    ) IS NOT NULL THEN
        RAISE EXCEPTION 'el DBA alternativo no completo la retirada V4';
    END IF;
END
$retirada_dba_alternativo$;
DROP ROLE vec_v4_dba_alternativo_prueba;
SQL
echo "baseline 000001/PostgreSQL 18.4: correcto"
