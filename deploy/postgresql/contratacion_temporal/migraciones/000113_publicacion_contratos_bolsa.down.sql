\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000113', 0)
);

-- CT113 no escribe historia: retirar la publicación no pierde datos. Bolsa
-- conserva lo que ya recibió en su propio histórico.
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT113 no está instalada';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer);
DROP FUNCTION vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz);
COMMIT;
