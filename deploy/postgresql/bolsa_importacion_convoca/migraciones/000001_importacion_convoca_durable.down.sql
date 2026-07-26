-- Destrucción protegida: el acta y su historia son prueba administrativa.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);

DO $confirmacion$
BEGIN
    IF current_setting(
        'vec.confirmar_destruccion_bolsa_importacion_convoca', true
    ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_IMPORTACION_CONVOCA_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destruccion de importacion Convoca no confirmada';
    END IF;
END
$confirmacion$;

DROP SCHEMA vec_bolsa_importacion_convoca CASCADE;
COMMIT;
