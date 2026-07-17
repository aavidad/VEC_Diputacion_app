BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;

DO $confirmacion$
BEGIN
    IF current_setting(
        'vec.confirmar_destruccion_bolsa_llamamientos', true
    ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_BOLSA_LLAMAMIENTOS_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destruccion de historia de llamamientos no confirmada';
    END IF;
END
$confirmacion$;

DROP SCHEMA vec_bolsa_llamamientos CASCADE;
COMMIT;
