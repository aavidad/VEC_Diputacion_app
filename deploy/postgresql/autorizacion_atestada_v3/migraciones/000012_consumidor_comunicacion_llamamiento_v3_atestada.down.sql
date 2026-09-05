\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000012', 0));
DO $proteger$
DECLARE
    v_def text;
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
    IF pg_catalog.to_regclass('vec_contratacion_temporal.comunicacion_llamamiento_local') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
          WHERE pg_catalog.convert_from(a.capacidad_canonica, 'UTF8')::jsonb ->> 'operacion' =
             'contratacion_temporal.llamamiento.comunicacion.registrar') THEN
        RAISE EXCEPTION 'reversión protegida: existe consumidor o historia de comunicación' USING ERRCODE = '55000';
    END IF;
    SELECT pg_catalog.pg_get_functiondef(
        'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
        INTO STRICT v_def;
    IF pg_catalog.strpos(v_def, v_extension) = 0 THEN
        RAISE EXCEPTION 'forma del consumidor incompatible' USING ERRCODE = '55000';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_extension, '');
END
$proteger$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
