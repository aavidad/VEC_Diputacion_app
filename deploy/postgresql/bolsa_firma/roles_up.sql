-- Bootstrap DBA de una sola ejecución para la saga durable de firma de Bolsa.
-- Los LOGIN se aprovisionan fuera del repositorio y solo reciben el grupo
-- técnico NOLOGIN estrictamente necesario.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_firma:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap de firma de Bolsa requiere superusuario';
    END IF;
    SELECT array_agg(rolname::text ORDER BY rolname) INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
        'vec_bolsa_firma_propietario',
        'vec_bolsa_firma_migrador',
        'vec_bolsa_firma_ejecutor'
     ]);
    IF cardinality(encontrados) > 0 OR
       pg_catalog.to_regnamespace('vec_bolsa_firma_guardia') IS NOT NULL OR
       EXISTS (
           SELECT 1 FROM pg_catalog.pg_event_trigger
            WHERE evtname = 'vec_bolsa_firma_cerrar_acl_tipos'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: ya existe la frontera de firma';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_firma_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_firma_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_firma_ejecutor NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_firma_propietario TO vec_bolsa_firma_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

CREATE SCHEMA vec_bolsa_firma_guardia;
REVOKE ALL ON SCHEMA vec_bolsa_firma_guardia FROM PUBLIC;

CREATE FUNCTION vec_bolsa_firma_guardia.cerrar_acl_tipos()
RETURNS event_trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_bolsa_firma'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I',
            tipo.nspname, tipo.typname, 'vec_bolsa_firma_ejecutor'
        );
    END LOOP;
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_bolsa_firma_guardia.cerrar_acl_tipos() FROM PUBLIC;

CREATE EVENT TRIGGER vec_bolsa_firma_cerrar_acl_tipos
    ON ddl_command_end
    WHEN TAG IN (
        'CREATE TABLE', 'CREATE TABLE AS', 'CREATE FOREIGN TABLE',
        'CREATE VIEW', 'CREATE MATERIALIZED VIEW', 'CREATE TYPE',
        'CREATE DOMAIN', 'ALTER TABLE', 'ALTER VIEW',
        'ALTER MATERIALIZED VIEW', 'ALTER TYPE', 'ALTER DOMAIN'
    )
    EXECUTE FUNCTION vec_bolsa_firma_guardia.cerrar_acl_tipos();

DO $conexiones$
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_firma_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_firma_migrador',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_firma_ejecutor',
        current_database()
    );
END
$conexiones$;
COMMIT;
