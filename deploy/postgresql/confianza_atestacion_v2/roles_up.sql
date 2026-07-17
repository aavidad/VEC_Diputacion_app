-- Bootstrap DBA del catalogo publico de confianza VEC-AD-2.
-- No crea cuentas LOGIN, claves privadas, secretos ni membresias de aplicacion.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_confianza_atestacion_v2:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap de confianza V2 rechazado: requiere superusuario';
    END IF;

    -- Este modulo se instala en la base VEC dedicada, despues de retirar el
    -- acceso implicito a public. No intenta endurecer una base compartida.
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
            HINT = 'instale primero el endurecimiento de la base dedicada VEC';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE base.datname = current_database()
           AND privilegio.grantee = 0
           AND privilegio.privilege_type IN (
               'CONNECT', 'CREATE', 'TEMPORARY'
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'la base dedicada conserva privilegios para PUBLIC',
            HINT = 'aplique primero el bootstrap cerrado de la base VEC';
    END IF;

    IF pg_catalog.to_regnamespace('vec_confianza_atestacion_v2') IS NOT NULL
       OR pg_catalog.to_regnamespace(
              'vec_confianza_atestacion_v2_guardia'
          ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_event_trigger
            WHERE evtname = 'vec_confianza_atestacion_v2_cerrar_acl_tipos'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existe un objeto de confianza V2';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname::text COLLATE "C")
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_confianza_atestacion_v2_propietario',
         'vec_confianza_atestacion_v2_migrador',
         'vec_confianza_atestacion_v2_lector_autoridad'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: ya existen roles de confianza V2',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_confianza_atestacion_v2_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_confianza_atestacion_v2_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_confianza_atestacion_v2_lector_autoridad NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_confianza_atestacion_v2_propietario
    TO vec_confianza_atestacion_v2_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- Los tipos fila implicitos de CREATE TABLE no respetan de forma suficiente
-- los default ACL de TYPES. Esta guarda, propiedad del DBA, cierra los tipos
-- presentes y los que creen futuras migraciones dentro del esquema exacto.
CREATE SCHEMA vec_confianza_atestacion_v2_guardia;
REVOKE ALL ON SCHEMA vec_confianza_atestacion_v2_guardia FROM PUBLIC;

CREATE FUNCTION vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos()
RETURNS event_trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $cerrar_acl_tipos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_confianza_atestacion_v2'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I',
            tipo.nspname,
            tipo.typname,
            'vec_confianza_atestacion_v2_lector_autoridad'
        );
    END LOOP;
END
$cerrar_acl_tipos$;
REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos()
    FROM PUBLIC;

CREATE EVENT TRIGGER vec_confianza_atestacion_v2_cerrar_acl_tipos
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
        vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos();

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_confianza_atestacion_v2_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_confianza_atestacion_v2_migrador',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_confianza_atestacion_v2_lector_autoridad',
        current_database()
    );
END
$privilegios_base$;

COMMIT;
