-- Bootstrap DBA aislado del registro de accesos T13. Las identidades LOGIN se
-- aprovisionan fuera del repositorio y solo reciben uno de estos grupos.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_registro_accesos:roles:v1', 0)
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap del registro de accesos: requiere superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_migrador'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el nucleo VEC Autorizacion requerido por T13';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = ANY (ARRAY[
             'vec_bolsa_accesos_propietario',
             'vec_bolsa_accesos_migrador',
             'vec_bolsa_accesos_registrador',
             'vec_bolsa_accesos_consultor',
             'vec_bolsa_accesos_gobernador'
         ])
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: ya existen roles T13';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_accesos_propietario NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_accesos_migrador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_accesos_registrador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_accesos_consultor NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_accesos_gobernador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_accesos_propietario TO vec_bolsa_accesos_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $acl_base$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_accesos_propietario',
        current_database()
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_accesos_migrador', 'vec_bolsa_accesos_registrador',
        'vec_bolsa_accesos_consultor', 'vec_bolsa_accesos_gobernador'
    ] LOOP
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',
                       current_database(), rol);
    END LOOP;
END
$acl_base$;
COMMIT;
