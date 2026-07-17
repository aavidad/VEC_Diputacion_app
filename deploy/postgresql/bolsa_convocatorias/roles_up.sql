-- Bootstrap DBA, de una sola ejecucion, para el gobierno de convocatorias.
-- Las identidades LOGIN se aprovisionan fuera del repositorio y solo reciben
-- uno de estos grupos tecnicos NOLOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_convocatorias:roles_up:v1', 0)
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
            MESSAGE = 'bootstrap de convocatorias rechazado: requiere superusuario';
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

    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_bolsa_convocatorias_propietario',
         'vec_bolsa_convocatorias_migrador',
         'vec_bolsa_convocatorias_ejecutor_consulta',
         'vec_bolsa_convocatorias_proyector_gobierno',
         'vec_bolsa_convocatorias_registrador_atestacion'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles de convocatorias',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_convocatorias_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_convocatorias_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_convocatorias_ejecutor_consulta NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_convocatorias_proyector_gobierno NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_convocatorias_registrador_atestacion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_convocatorias_propietario
    TO vec_bolsa_convocatorias_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_convocatorias_propietario',
        current_database()
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_migrador',
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion'
    ] LOOP
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), rol);
    END LOOP;
END
$privilegios_base$;
COMMIT;
