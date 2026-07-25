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
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_resolver(text,text,numeric,text,timestamptz)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_revalidar_actual(text,text,text,numeric,text,text,numeric,text,text,numeric,text)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_politica_ligada(jsonb,jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.control_migracion_cobertura_o4'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_politica'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actuacion'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_actual'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_retirada'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_checkpoint'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada 000018 O4-04B fuera de orden';
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
            MESSAGE = 'retirada 000018 O4-04B protegida por historia';
    END IF;
END
$proteccion$;

DROP TRIGGER validar_avance
    ON vec_contratacion_temporal.gobi_o404b_checkpoint;
DROP TRIGGER bloquear_insercion_borrado
    ON vec_contratacion_temporal.gobi_o404b_checkpoint;
DROP TRIGGER bloquear_truncado
    ON vec_contratacion_temporal.gobi_o404b_checkpoint;
DROP TABLE vec_contratacion_temporal.gobi_o404b_retirada RESTRICT;
DROP TABLE vec_contratacion_temporal.gobi_o404b_actual RESTRICT;
DROP TABLE vec_contratacion_temporal.gobi_o404b_actuacion RESTRICT;
DROP TABLE vec_contratacion_temporal.gobi_o404b_politica RESTRICT;
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_validar_checkpoint();
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_validar_actual();
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_material_actuacion(jsonb);
DROP FUNCTION vec_contratacion_temporal.gobi_o404b_material_politica(jsonb);

COMMIT;
