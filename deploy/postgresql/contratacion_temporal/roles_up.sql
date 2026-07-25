-- Bootstrap DBA de una sola ejecución. Las cuentas LOGIN se aprovisionan
-- fuera del repositorio y solo reciben los grupos técnicos NOLOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:roles_up:v1',
        0
    )
);

DO $prevalidacion$
DECLARE
    v_encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'bootstrap rechazado: requiere superusuario';
    END IF;
    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO v_encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_contratacion_temporal_propietario',
         'vec_contratacion_temporal_migrador',
         'vec_contratacion_temporal_ejecutor',
         'vec_contratacion_temporal_gobernador'
     ]);
    IF cardinality(v_encontrados) > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: ya existen roles',
            DETAIL = array_to_string(v_encontrados, ',');
    END IF;
    IF to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: el esquema ya existe';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_contratacion_temporal_propietario
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_migrador
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_ejecutor
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_gobernador
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;

GRANT vec_contratacion_temporal_propietario
    TO vec_contratacion_temporal_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

CREATE SCHEMA vec_contratacion_temporal
    AUTHORIZATION vec_contratacion_temporal_propietario;
REVOKE ALL ON SCHEMA vec_contratacion_temporal FROM PUBLIC;

DO $privilegios$
BEGIN
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_migrador',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_ejecutor',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_gobernador',
        current_database()
    );
END
$privilegios$;

COMMIT;
