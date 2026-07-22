-- Bootstrap DBA para la base fisicamente separada de proyeccion publica.
-- Las cuentas LOGIN se aprovisionan fuera del repositorio y reciben como
-- unica membresia tecnica vec_bolsa_publica_consulta.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap de bolsa publica rechazado: requiere superusuario';
    END IF;

    -- Es una precondicion de aprovisionamiento de esta base dedicada. La
    -- migracion no revoca ACL globales para no apropiarse de permisos ajenos
    -- ni tener que adivinar su estado al revertir.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base,
               LATERAL pg_catalog.aclexplode(
                   COALESCE(base.datacl, pg_catalog.acldefault('d', base.datdba))
               ) AS permiso
         WHERE base.datname = current_database()
           AND permiso.grantee = 0
           AND permiso.privilege_type IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: la base dedicada conserva privilegios para PUBLIC';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio,
               LATERAL pg_catalog.aclexplode(
                   COALESCE(espacio.nspacl, pg_catalog.acldefault('n', espacio.nspowner))
               ) AS permiso
         WHERE espacio.nspname = 'public'
           AND permiso.grantee = 0
           AND permiso.privilege_type = 'CREATE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: el esquema public permite CREATE a PUBLIC';
    END IF;

    IF pg_catalog.to_regnamespace('vec_bolsa_publica_datos') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_bolsa_publica_lectura') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_bolsa_publica_publicacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen esquemas de bolsa publica';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_bolsa_publica_propietario',
         'vec_bolsa_publica_publicacion_propietario',
         'vec_bolsa_publica_migrador',
         'vec_bolsa_publica_consulta',
         'vec_bolsa_publica_publicador'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles de bolsa publica',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_publica_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_publica_publicacion_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_publica_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_publica_consulta NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_publica_publicador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

CREATE SCHEMA vec_bolsa_publica_datos AUTHORIZATION vec_bolsa_publica_propietario;
CREATE SCHEMA vec_bolsa_publica_lectura AUTHORIZATION vec_bolsa_publica_propietario;
CREATE SCHEMA vec_bolsa_publica_publicacion
    AUTHORIZATION vec_bolsa_publica_publicacion_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_publica_datos FROM PUBLIC;
REVOKE ALL ON SCHEMA vec_bolsa_publica_lectura FROM PUBLIC;
REVOKE ALL ON SCHEMA vec_bolsa_publica_publicacion FROM PUBLIC;

GRANT vec_bolsa_publica_propietario TO vec_bolsa_publica_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT vec_bolsa_publica_publicacion_propietario TO vec_bolsa_publica_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_publica_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_publica_migrador, vec_bolsa_publica_consulta, vec_bolsa_publica_publicador',
        current_database()
    );
END
$privilegios_base$;

ALTER ROLE vec_bolsa_publica_consulta SET default_transaction_read_only = on;
ALTER ROLE vec_bolsa_publica_consulta SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_publica_consulta SET statement_timeout = '10s';
ALTER ROLE vec_bolsa_publica_consulta SET lock_timeout = '2s';
ALTER ROLE vec_bolsa_publica_consulta SET idle_in_transaction_session_timeout = '10s';
ALTER ROLE vec_bolsa_publica_publicador SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_publica_publicador SET statement_timeout = '2min';
ALTER ROLE vec_bolsa_publica_publicador SET lock_timeout = '10s';
ALTER ROLE vec_bolsa_publica_publicador SET idle_in_transaction_session_timeout = '2min';
-- Los clientes deben invocar el publicador con parametros enlazados. Si una
-- validacion falla, PostgreSQL no vuelca el JSON potencialmente sensible.
ALTER ROLE vec_bolsa_publica_publicador SET log_parameter_max_length_on_error = 0;

COMMIT;
