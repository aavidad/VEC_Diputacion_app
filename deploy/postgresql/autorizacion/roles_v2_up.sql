-- Evolucion DBA independiente para la proyeccion de motivos V2. No modifica
-- ni adopta los roles historicos. Debe ejecutarse despues de roles_up.sql.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:roles_motivos_v2:up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_autorizacion_motivos_proyector',
         'vec_autorizacion_motivos_evaluador'
     ]);

    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap V2 rechazado: existen roles de motivos; no se adoptan ni modifican',
            DETAIL = array_to_string(encontrados, ',');
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario'
           AND NOT rolcanlogin
           AND NOT rolsuper
           AND NOT rolcreaterole
           AND NOT rolcreatedb
           AND NOT rolreplication
           AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap V2 rechazado: falta el propietario V1 esperado';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_autorizacion_motivos_proyector
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_motivos_evaluador
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;

-- Los roles de ejecucion no son miembros del propietario ni del migrador.
-- Sus unicos privilegios de esquema se conceden desde 000003.
DO $privilegios_base$
BEGIN
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_motivos_proyector',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_motivos_evaluador',
        current_database()
    );
END
$privilegios_base$;

COMMIT;
