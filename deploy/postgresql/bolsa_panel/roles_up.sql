-- Bootstrap DBA de una sola ejecucion para el panel interno de bolsas.
-- Las identidades LOGIN se aprovisionan fuera del repositorio y solo reciben
-- uno de estos grupos tecnicos NOLOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_panel:roles_up:v1', 0)
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
            MESSAGE = 'bootstrap del panel rechazado: requiere superusuario';
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
            MESSAGE = 'falta el nucleo de autorizacion requerido';
    END IF;

    SELECT array_agg(
               rolname::text ORDER BY rolname::text COLLATE "C"
           )
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_panel_propietario',
         'vec_bolsa_panel_migrador',
         'vec_bolsa_panel_proyector',
         'vec_bolsa_panel_ejecutor_consulta',
         'vec_bolsa_panel_registrador_atestacion'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: ya existen roles del panel',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_panel_propietario NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_panel_migrador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_panel_proyector NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_panel_ejecutor_consulta NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_panel_registrador_atestacion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_panel_propietario TO vec_bolsa_panel_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_panel_propietario',
        current_database()
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_panel_migrador',
        'vec_bolsa_panel_proyector',
        'vec_bolsa_panel_ejecutor_consulta',
        'vec_bolsa_panel_registrador_atestacion'
    ] LOOP
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',
                       current_database(), rol);
    END LOOP;
END
$privilegios_base$;
COMMIT;
