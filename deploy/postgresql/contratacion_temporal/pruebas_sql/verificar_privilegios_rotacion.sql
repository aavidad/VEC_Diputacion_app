\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY;

DO $prueba$
DECLARE
    v_ejecutor oid := (
        SELECT oid
        FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_contratacion_temporal_ejecutor'
    );
BEGIN
    IF pg_catalog.has_function_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.preparar_alta_v1(jsonb)',
        'EXECUTE'
    ) OR NOT pg_catalog.has_function_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.preparar_alta_v2(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'la exclusión v1/v2 de runtime es incorrecta';
    END IF;
    IF NOT (
        SELECT prosecdef
           AND provolatile = 'v'
           AND proparallel = 'u'
           AND proconfig @> ARRAY[
               'search_path=pg_catalog',
               'lock_timeout=2s'
           ]::text[]
           AND cardinality(proconfig) = 2
        FROM pg_catalog.pg_proc
        WHERE oid =
            'vec_contratacion_temporal.preparar_alta_v2(jsonb)'::regprocedure
    ) THEN
        RAISE EXCEPTION 'la función v2 no conserva su cierre';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM pg_catalog.pg_class AS objeto
        JOIN pg_catalog.pg_namespace AS espacio
          ON espacio.oid = objeto.relnamespace
        WHERE espacio.nspname = 'vec_contratacion_temporal'
          AND objeto.relkind IN ('r', 'p', 'S', 'v', 'm')
          AND pg_catalog.has_table_privilege(
              v_ejecutor,
              objeto.oid,
              'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
          )
    ) THEN
        RAISE EXCEPTION 'runtime conserva privilegios directos tras rotación';
    END IF;
    IF (
        SELECT array_agg(generacion ORDER BY posicion)
        FROM vec_contratacion_temporal.politica_generaciones_hmac_alta
    ) IS DISTINCT FROM ARRAY[2, 1]::integer[] THEN
        RAISE EXCEPTION 'política de generaciones inesperada';
    END IF;
END
$prueba$;

COMMIT;
