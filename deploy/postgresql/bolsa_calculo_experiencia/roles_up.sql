-- Bootstrap DBA de los roles tecnicos del calculo oficial de experiencia.
-- Las identidades LOGIN se aprovisionan fuera del repositorio y reciben un
-- solo grupo NOLOGIN conforme a su funcion.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_calculo_experiencia:roles_up:v1', 0
    )
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
            MESSAGE = 'bootstrap de calculo de experiencia: requiere superusuario';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_calculo_experiencia_propietario',
         'vec_bolsa_calculo_experiencia_migrador',
         'vec_bolsa_calculo_experiencia_aplicacion',
         'vec_bolsa_calculo_experiencia_lector_operativo',
         'vec_bolsa_calculo_experiencia_publicador'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles del calculo',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_calculo_experiencia_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_calculo_experiencia_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_calculo_experiencia_aplicacion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_calculo_experiencia_lector_operativo NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_calculo_experiencia_publicador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_calculo_experiencia_propietario
    TO vec_bolsa_calculo_experiencia_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO %I',
        current_database(),
        'vec_bolsa_calculo_experiencia_propietario'
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_calculo_experiencia_migrador',
        'vec_bolsa_calculo_experiencia_aplicacion',
        'vec_bolsa_calculo_experiencia_lector_operativo',
        'vec_bolsa_calculo_experiencia_publicador'
    ] LOOP
        EXECUTE format(
            'GRANT CONNECT ON DATABASE %I TO %I', current_database(), rol
        );
    END LOOP;
END
$privilegios_base$;
COMMIT;
