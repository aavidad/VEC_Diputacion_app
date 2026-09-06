\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $retirar$
DECLARE
    v_def text; v_acl aclitem[];
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'organizacion_preparacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'estructura_organizativa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_estructura_organizativa'
           )
$perfil$;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_contratacion_temporal' AND p.proname='registrar_cambio_organizacion_v1'
    ) THEN
        RAISE EXCEPTION 'retirar primero consumidor de negocio, sin datos' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible; no retirar otros perfiles' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_extension,'');
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'retirada alteró permisos del núcleo' USING ERRCODE='55000';
    END IF;
END
$retirar$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_organizacion_preparacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
