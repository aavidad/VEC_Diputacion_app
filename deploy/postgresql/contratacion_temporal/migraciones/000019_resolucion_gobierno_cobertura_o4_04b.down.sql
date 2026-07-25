BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_resolver(text,text,numeric,text,timestamptz)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actual'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_checkpoint'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada 000019 O4-04B fuera de orden';
    END IF;
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.control_migracion_cobertura_o4'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404c_referencia_derivada_v1(text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada 000019 O4-04B fuera de orden';
    END IF;
END
$prevalidacion$;

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.gobi_o404b_checkpoint
         WHERE ultima_secuencia > 0
    ) AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_gobierno_cobertura_o4_04b', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_GOBIERNO_COBERTURA_O4_04B_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada O4-04B rechazada: existe historia';
    END IF;
END
$proteccion$;

REVOKE USAGE ON SCHEMA vec_contratacion_temporal
FROM vec_contratacion_temporal_gobernador;
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_revalidar_actual(
    text, text, text, numeric, text, text, numeric, text,
    text, numeric, text
);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_resolver(
    text, text, numeric, text, timestamptz
);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_retirar(jsonb);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_publicar(jsonb);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_politica_ligada(
    jsonb, jsonb
);

COMMIT;
