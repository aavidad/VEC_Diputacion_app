\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000023',0));
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;

DO $retirar$
DECLARE
    v_def text; v_acl aclitem[]; v_owner oid;
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.presentar',
                   'contratacion_temporal.peticion_centro.ratificar',
                   'contratacion_temporal.peticion_centro.consultar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
           )
$perfil$;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_contratacion_temporal'
           AND p.proname IN ('registrar_peticion_centro_v1','consultar_peticiones_centro_v1')
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
         WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion' = ANY (ARRAY[
             'contratacion_temporal.peticion_centro.presentar',
             'contratacion_temporal.peticion_centro.ratificar',
             'contratacion_temporal.peticion_centro.consultar'
         ])
    ) THEN
        RAISE EXCEPTION 'reversión denegada: consumidor o historia de petición de centro' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner
      INTO STRICT v_def,v_acl,v_owner
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible; no retirar otros perfiles' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_extension,'');
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl
       OR (SELECT proowner FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_owner THEN
        RAISE EXCEPTION 'retirada alteró autoridad del núcleo' USING ERRCODE='55000';
    END IF;
END
$retirar$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
