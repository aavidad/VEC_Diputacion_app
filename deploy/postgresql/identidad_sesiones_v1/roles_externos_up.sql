-- DBA: capacidades exclusivas para la identidad del proceso externo.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_provisionador') IS NOT NULL
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_registrador') IS NOT NULL
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_revalidador') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_identidad_externa_v1') IS NOT NULL THEN
        RAISE EXCEPTION 'roles externos: precondiciones incumplidas' USING ERRCODE='55000';
    END IF;
END $pre$;

CREATE ROLE vec_identidad_externa_v1_provisionador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_externa_v1_registrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_externa_v1_revalidador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;

CREATE SCHEMA vec_identidad_externa_v1 AUTHORIZATION vec_identidad_sesiones_v1_propietario;
REVOKE ALL ON SCHEMA vec_identidad_externa_v1 FROM PUBLIC;

DO $conectar$
DECLARE rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_identidad_externa_v1_provisionador',
        'vec_identidad_externa_v1_registrador',
        'vec_identidad_externa_v1_revalidador'
    ] LOOP
        EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO %I',
                                  pg_catalog.current_database(), rol);
    END LOOP;
END $conectar$;
COMMIT;
