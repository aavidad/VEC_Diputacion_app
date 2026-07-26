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
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 8
 FOR UPDATE;
DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 8
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.prueba_denegacion_decision_cobertura'
    ) IS NOT NULL
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute a
         WHERE a.attrelid = pg_catalog.to_regclass(
             'vec_contratacion_temporal.'
             || 'confirmacion_operacion_decision_cobertura'
         )
           AND a.attname = 'carga_huella_sha256'
           AND NOT a.attisdropped
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint c
         WHERE c.connamespace = pg_catalog.to_regnamespace(
             'vec_contratacion_temporal'
         )
           AND c.conname IN (
             'terminal_ambito_decision_o404e_unico',
             'confirmacion_carga_huella_o404e_valida'
           )
    )
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.o404e_motivo_denegacion_canon_v1(jsonb)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(jsonb)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.o404e_mapa_json_go_v1(jsonb)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.o404e_texto_json_go_v1(text)'
    ) IS NOT NULL
    THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado o capa 000029 impide revertir validadores O4-04E';
    END IF;
END
$prevalidacion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 7,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 8;

DROP FUNCTION
    vec_contratacion_temporal.o404e_material_recibo_v1(jsonb),
    vec_contratacion_temporal.o404e_transicion_exacta_v1(
        jsonb,jsonb,jsonb,jsonb,timestamptz
    ),
    vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text),
    vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb),
    vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(jsonb),
    vec_contratacion_temporal.o404e_mapa_v1(bytea,jsonb),
    vec_contratacion_temporal.o404e_texto_v1(bytea,text),
    vec_contratacion_temporal.o404e_claves_exactas_v1(jsonb,text[]);

COMMIT;
