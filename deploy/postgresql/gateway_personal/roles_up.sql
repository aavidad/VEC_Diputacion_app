-- Bootstrap DBA para la sesion opaca del gateway de Personal. No crea LOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_gateway_personal:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap gateway Personal rechazado: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_gateway_personal') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap gateway Personal rechazado: esquema preexistente';
    END IF;
    SELECT pg_catalog.array_agg(rolname::text ORDER BY rolname::text)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
        'vec_gateway_personal_propietario',
        'vec_gateway_personal_migrador',
        'vec_gateway_personal_ejecutor'
     ]);
    IF pg_catalog.cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap gateway Personal rechazado: roles preexistentes',
            DETAIL = pg_catalog.array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_gateway_personal_propietario NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_gateway_personal_migrador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_gateway_personal_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_gateway_personal_propietario TO vec_gateway_personal_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_gateway_personal_propietario',
        current_database()
    );
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_gateway_personal_migrador, vec_gateway_personal_ejecutor',
        current_database()
    );
END
$privilegios_base$;
COMMIT;
