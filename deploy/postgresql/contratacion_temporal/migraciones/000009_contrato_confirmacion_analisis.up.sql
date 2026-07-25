BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000009_contrato_confirmacion_analisis', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_fuentes_analisis'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.texto_json_go_v1(text)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb)'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.consumo_fuentes_analisis
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para contrato de confirmación';
    END IF;
END
$prevalidacion$;

-- Conserva exactamente los bytes TLV que originaron la huella del conjunto.
-- El JSON de lectura no se usa para reconstruir esta evidencia probatoria.
ALTER TABLE vec_contratacion_temporal.consumo_fuentes_analisis
    ADD COLUMN prueba_canonica bytea NOT NULL,
    ADD CONSTRAINT consumo_fuentes_prueba_canonica_huella CHECK (
        pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = conjunto_huella_sha256
    ),
    ADD CONSTRAINT consumo_fuentes_prueba_canonica_tamano CHECK (
        pg_catalog.octet_length(prueba_canonica) BETWEEN 64 AND 65536
    );

-- Reproduce el JSON compacto y ordenado que encoding/json genera para
-- RecursoAutorizable. Solo admite el subconjunto exacto de análisis O3.
CREATE FUNCTION
vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(
    p_operacion jsonb
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    p jsonb := p_operacion -> 'politica';
    v_json text;
BEGIN
    IF pg_catalog.jsonb_typeof(p_operacion) <> 'object'
       OR pg_catalog.jsonb_typeof(p) <> 'object' THEN
        RETURN NULL;
    END IF;
    v_json :=
        '{"ambitos":{"estado_previo":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'estado_previo'
        ) || ',"expediente_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'expediente_ref'
        ) || ',"fase_previa":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'fase_previa'
        ) || ',"organizacion_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'organizacion_ref'
        ) || '},"atributos":{"analisis_derivado_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'analisis_derivado_huella_sha256'
        ) || ',"artefacto_analisis_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'artefacto_huella_sha256'
        ) || ',"artefacto_analisis_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'artefacto_ref'
        ) || ',"exige_actor_distinto":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            CASE WHEN (p ->> 'exige_actor_distinto')::boolean
                 THEN 'true' ELSE 'false' END
        ) || ',"huella_semantica_hmac":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'huella_semantica_hmac'
        ) || ',"operacion":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'operacion'
        ) || ',"politica_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'huella_sha256'
        ) || ',"politica_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'definicion_ref'
        ) || ',"politica_version":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'version'
        ) || ',"version_expediente_esperada":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p_operacion ->> 'version_anterior'
        ) || '}}';
    RETURN pg_catalog.encode(
        pg_catalog.sha256(pg_catalog.convert_to(v_json, 'UTF8')),
        'hex'
    );
EXCEPTION
    WHEN data_exception OR invalid_text_representation THEN
        RETURN NULL;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
