-- Bootstrap DBA de una sola ejecucion para la persistencia de baremaciones.
-- Los roles son grupos NOLOGIN: las identidades LOGIN se aprovisionan fuera
-- de este repositorio y reciben solo el grupo tecnico que necesitan.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_baremacion:roles_up:v1', 0)
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
            MESSAGE = 'bootstrap Bolsa rechazado: requiere superusuario';
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
           'vec_bolsa_baremacion_guardia'
       ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_bolsa_baremacion_cerrar_acl_tipos'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existe la guarda DDL Bolsa';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_bolsa_baremacion_propietario',
         'vec_bolsa_baremacion_migrador',
         'vec_bolsa_baremacion_ejecutor',
         'vec_bolsa_baremacion_lector_outbox',
         'vec_bolsa_baremacion_registrador_atestacion'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles Bolsa; no se adoptan ni modifican',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_baremacion_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_baremacion_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_baremacion_ejecutor NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_baremacion_lector_outbox NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
-- Se crea sin privilegios deliberadamente. Solo recibira EXECUTE cuando el
-- registrador criptografico pareado con el verificador PDP este disponible.
CREATE ROLE vec_bolsa_baremacion_registrador_atestacion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_baremacion_propietario TO vec_bolsa_baremacion_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- PostgreSQL no aplica ALTER DEFAULT PRIVILEGES ON TYPES al tipo fila creado
-- implicitamente por CREATE TABLE. Esta guarda, propiedad del DBA, cierra los
-- tipos actuales y futuros del esquema Bolsa al terminar cada DDL relevante.
-- No recibe nombres del comando y su search_path no contiene esquemas
-- controlables por una cuenta de runtime.
CREATE SCHEMA vec_bolsa_baremacion_guardia;
REVOKE ALL ON SCHEMA vec_bolsa_baremacion_guardia FROM PUBLIC;

CREATE FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
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
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I, %I',
            tipo.nspname,
            tipo.typname,
            'vec_bolsa_baremacion_ejecutor',
            'vec_bolsa_baremacion_lector_outbox',
            'vec_bolsa_baremacion_registrador_atestacion'
        );
    END LOOP;
END
$cerrar_acl_tipos$;
REVOKE ALL ON FUNCTION
    vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
    FROM PUBLIC;

CREATE EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos
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
    EXECUTE FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos();

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_baremacion_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_baremacion_migrador',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_baremacion_ejecutor',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_baremacion_lector_outbox',
        current_database()
    );
END
$privilegios_base$;
COMMIT;
