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
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_politica'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actuacion'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actual'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_checkpoint'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_catalogo'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_evento'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada 000017 O4-04B fuera de orden';
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
            MESSAGE = 'retirada 000017 O4-04B protegida por historia';
    END IF;
END
$proteccion$;

DROP TABLE vec_contratacion_temporal.gobi_o404b_checkpoint RESTRICT;
DROP TABLE vec_contratacion_temporal.gobi_o404b_catalogo RESTRICT;
DROP TABLE vec_contratacion_temporal.gobi_o404b_evento RESTRICT;
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_material_catalogo(jsonb);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
    text, boolean
);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_entorno_valido(boolean);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_microsegundos(timestamptz);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_texto_canon(bytea, text);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_claves_exactas(
    jsonb, text[]
);

COMMIT;
