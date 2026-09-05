\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000012', 0));

-- Amplía solo la lista nominal de operaciones del consumidor único. Conserva
-- íntegros verificación, revocaciones y perfiles añadidos por 000011; no copia
-- otra implementación criptográfica ni restaura una definición histórica.
DO $ampliar$
DECLARE
    v_def text;
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'comunicacion_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'comunicacion_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
BEGIN
    IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'consumidor de comunicación ya instalado' USING ERRCODE = '55000';
    END IF;
    SELECT pg_catalog.pg_get_functiondef(
        'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
        INTO STRICT v_def;
    IF pg_catalog.strpos(v_def, v_marca) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_marca) + pg_catalog.length(v_marca)), v_marca) <> 0
       OR pg_catalog.strpos(v_def, 'p_perfil_mutacion IS NOT DISTINCT FROM ''comunicacion_llamamiento''') <> 0 THEN
        RAISE EXCEPTION 'forma del consumidor incompatible' USING ERRCODE = '55000';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_marca, v_extension || v_marca);
END
$ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE (
    decision_ref text, efecto_ref text, huella_efecto_sha256 text,
    consumo_huella_sha256 text, auditoria_ref text,
    consumida_en timestamptz, consumo_nuevo boolean
)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
    SELECT * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'comunicacion_llamamiento', p_capacidad, p_decision, p_motivo, p_contexto,
        p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz)
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor,
    vec_autorizacion_atestada_v3_emisor, vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) TO vec_contratacion_temporal_propietario;
COMMIT;
