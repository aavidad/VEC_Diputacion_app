BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:analisis-v3:000001', 0
    )
);

DO $dependencias$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n
            ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_contratacion_temporal'
           AND p.proname = 'confirmar_operacion_analisis_v1'
           AND p.pronargs = 1
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '2BP01',
            MESSAGE = 'retirada de autorización de análisis bloqueada por el consumidor';
    END IF;
END
$dependencias$;

REVOKE EXECUTE ON FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM vec_contratacion_temporal_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_contratacion_temporal_propietario;
DROP FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) RESTRICT;

COMMIT;
