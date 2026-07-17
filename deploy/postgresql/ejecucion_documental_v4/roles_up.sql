-- Bootstrap DBA. Requiere que el nucleo vec_autorizacion ya este instalado.
-- No crea LOGIN ni concede membresia alguna en vec_autorizacion_propietario.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_ejecucion_documental_v4:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    -- CREATEROLE y la propiedad de la base no equivalen a autoridad de
    -- bootstrap. Se comprueba antes de inspeccionar dependencias y, sobre
    -- todo, antes de crear el primer rol u objeto gobernado.
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap documental V4 rechazado: requiere superusuario';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = extension.extnamespace
         WHERE extension.extname = 'pgcrypto'
           AND espacio.nspname = 'public'
    ) OR pg_catalog.to_regprocedure(
        'public.hmac(bytea,bytea,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta pgcrypto instalado en el esquema public',
            HINT = 'el DBA debe ejecutar CREATE EXTENSION pgcrypto antes del despliegue';
    END IF;

    -- La base VEC es dedicada y su bootstrap de autorizacion debe haber
    -- retirado el acceso implicito a public. Sin esta precondicion, revocar
    -- EXECUTE no bastaria para mantener cerrada la superficie de extensiones
    -- que una futura actualizacion pudiera volver a crear con ACL por defecto.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.nspname = 'public'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'USAGE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el esquema public conserva USAGE para PUBLIC',
            HINT = 'aplique antes el endurecimiento de la base dedicada VEC';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario' AND NOT rolcanlogin
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_migrador' AND NOT rolcanlogin
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el nucleo de autorizacion requerido';
    END IF;

    IF pg_catalog.to_regnamespace(
           'vec_ejecucion_documental_v4_guardia'
       ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existe la guarda DDL V4';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_ejecucion_documental_v4_propietario',
         'vec_ejecucion_documental_v4_migrador',
         'vec_ejecucion_documental_v4_emisor_capacidad',
         'vec_ejecucion_documental_v4_ejecutor_atestado'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles V4; no se adoptan ni modifican',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_ejecucion_documental_v4_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ejecucion_documental_v4_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ejecucion_documental_v4_emisor_capacidad NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ejecucion_documental_v4_ejecutor_atestado NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_ejecucion_documental_v4_propietario
    TO vec_ejecucion_documental_v4_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- CREATE TABLE no aplica los default ACL de TYPES al tipo fila implicito. Una
-- guarda DDL propiedad del DBA cierra automaticamente todos los tipos V4 tras
-- crear o mover tablas, vistas, dominios o tipos. La funcion es SECURITY
-- DEFINER, no recibe identificadores del DDL y solo actua sobre el esquema V4.
CREATE SCHEMA vec_ejecucion_documental_v4_guardia;
REVOKE ALL ON SCHEMA vec_ejecucion_documental_v4_guardia FROM PUBLIC;

CREATE FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()
RETURNS event_trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $cerrar_acl_tipos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I',
            tipo.nspname,
            tipo.typname,
            'vec_ejecucion_documental_v4_emisor_capacidad',
            'vec_ejecucion_documental_v4_ejecutor_atestado'
        );
    END LOOP;
END
$cerrar_acl_tipos$;
REVOKE ALL ON FUNCTION
    vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()
    FROM PUBLIC;

CREATE EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos
    ON ddl_command_end
    WHEN TAG IN (
        'CREATE TABLE',
        'CREATE TABLE AS',
        'CREATE FOREIGN TABLE',
        'CREATE VIEW',
        'CREATE MATERIALIZED VIEW',
        'CREATE TYPE',
        'CREATE DOMAIN',
        'ALTER TABLE',
        'ALTER VIEW',
        'ALTER MATERIALIZED VIEW',
        'ALTER TYPE',
        'ALTER DOMAIN'
    )
    EXECUTE FUNCTION
        vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();

-- pgcrypto se mantiene fuera del esquema de producto. La instalacion de una
-- extension concede normalmente EXECUTE a PUBLIC; se retira de cada funcion
-- que pertenece realmente a pgcrypto, sin afectar a otras funciones de public.
-- Solo el propietario V4 recibe despues el overload bytea exacto usado dentro
-- de ejecutar_plan_atestado(), que es SECURITY DEFINER y no filtra el secreto.
DO $cerrar_pgcrypto$
DECLARE
    funcion record;
BEGIN
    FOR funcion IN
        SELECT espacio.nspname,
               procedimiento.proname,
               pg_catalog.pg_get_function_identity_arguments(
                   procedimiento.oid
               ) AS argumentos
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS procedimiento
            ON procedimiento.oid = dependencia.objid
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = procedimiento.pronamespace
         WHERE extension.extname = 'pgcrypto'
           AND procedimiento.prokind = 'f'
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON FUNCTION %I.%I(%s) FROM PUBLIC',
            funcion.nspname,
            funcion.proname,
            funcion.argumentos
        );
    END LOOP;
END
$cerrar_pgcrypto$;

GRANT USAGE ON SCHEMA public
    TO vec_ejecucion_documental_v4_propietario;
GRANT EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    TO vec_ejecucion_documental_v4_propietario;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_ejecucion_documental_v4_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_ejecucion_documental_v4_migrador',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_ejecucion_documental_v4_emisor_capacidad',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_ejecucion_documental_v4_ejecutor_atestado',
        current_database()
    );
END
$privilegios_base$;

COMMIT;
