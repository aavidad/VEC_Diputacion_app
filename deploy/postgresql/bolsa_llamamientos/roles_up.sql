-- Bootstrap DBA de una sola ejecucion. Las identidades LOGIN se gestionan
-- fuera del repositorio y solo heredan un grupo tecnico minimo NOLOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:roles_up:v1', 0)
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
            MESSAGE = 'bootstrap de llamamientos requiere superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario' AND NOT rolcanlogin
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el nucleo de autorizacion';
    END IF;
    SELECT array_agg(rolname::text ORDER BY rolname) INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_bolsa_llamamientos_propietario',
         'vec_bolsa_llamamientos_migrador',
         'vec_bolsa_llamamientos_ejecutor',
         'vec_bolsa_llamamientos_proyector_autoritativo',
         'vec_bolsa_llamamientos_registrador_atestacion',
         'vec_bolsa_llamamientos_despachador_outbox'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ya existen roles de llamamientos',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_llamamientos_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_ejecutor NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_proyector_autoritativo NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_registrador_atestacion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_despachador_outbox NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_llamamientos_propietario
    TO vec_bolsa_llamamientos_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $conexiones$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_llamamientos_propietario',
        current_database()
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_llamamientos_migrador',
        'vec_bolsa_llamamientos_ejecutor',
        'vec_bolsa_llamamientos_proyector_autoritativo',
        'vec_bolsa_llamamientos_registrador_atestacion',
        'vec_bolsa_llamamientos_despachador_outbox'
    ] LOOP
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), rol);
    END LOOP;
END
$conexiones$;
COMMIT;
