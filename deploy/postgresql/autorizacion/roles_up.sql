-- Bootstrap DBA de una sola ejecucion para una base dedicada de VEC. No crea
-- usuarios LOGIN ni contiene contrasenas. Las identidades de despliegue y de
-- ejecucion se aprovisionan despues y solo heredan el grupo NOLOGIN preciso.
BEGIN;

SET LOCAL search_path = pg_catalog;

-- Serializa intentos concurrentes en esta base. El preflight sucede
-- antes de cualquier mutacion: un nombre preexistente nunca se adopta, corrige
-- ni inspecciona como si perteneciera al subsistema.
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_autorizacion_propietario',
         'vec_autorizacion_migrador',
         'vec_autorizacion_fuente',
         'vec_autorizacion_registro'
     ]);

    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles VEC; no se adoptan ni modifican',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_autorizacion_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_fuente NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_registro NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_autorizacion_propietario TO vec_autorizacion_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- Esta revocacion presupone una base dedicada. No ejecutar este guion en una
-- base compartida sin inventariar antes todos sus consumidores.
DO $privilegios_base$
BEGIN
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC', current_database());
    EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_autorizacion_propietario', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_migrador', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_fuente', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_registro', current_database());
END
$privilegios_base$;

REVOKE ALL ON SCHEMA public FROM PUBLIC;

COMMIT;
