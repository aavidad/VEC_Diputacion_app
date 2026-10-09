\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:orq1:ct199', 0));
DO $pre$
DECLARE v_huella text; v_acl aclitem[]; v_config text[]; v_prop name;
        v_definer boolean; v_estricta boolean; v_volatil char; v_paralelo char;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'CT199: PARO clave=migrador actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';
 END IF;
 IF to_regprocedure('vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'CT199: PARO clave=dependencia actual=CT198_ausente esperado=CT198_instalada' USING ERRCODE='55000';
 END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),p.proacl,p.proconfig,r.rolname,
        p.prosecdef,p.proisstrict,p.provolatile,p.proparallel
 INTO v_huella,v_acl,v_config,v_prop,v_definer,v_estricta,v_volatil,v_paralelo
 FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
 WHERE p.oid='vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb)'::regprocedure;
 IF v_huella IS DISTINCT FROM '525da10714d5564cb04837e687237f32f546fab31ed01732100e3f0f351e94bb' THEN
  RAISE EXCEPTION 'CT199: PARO clave=huellas_materiales.prosrc actual=% esperado=525da10714d5564cb04837e687237f32f546fab31ed01732100e3f0f351e94bb',coalesce(v_huella,'ausente') USING ERRCODE='55000';
 END IF;
 IF v_prop IS DISTINCT FROM 'vec_contratacion_temporal_propietario'
    OR v_acl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario'::aclitem]
    OR v_config IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR v_definer IS DISTINCT FROM false OR v_estricta IS DISTINCT FROM true
    OR v_volatil IS DISTINCT FROM 'i' OR v_paralelo IS DISTINCT FROM 's' THEN
  RAISE EXCEPTION 'CT199: PARO clave=metadatos_funcion actual=divergente esperado=owner_CT_ACL_owner_search_path_pg_catalog_immutable_parallel_safe_strict_invoker' USING ERRCODE='55000';
 END IF;
END
$pre$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(p_artefacto jsonb)
 RETURNS text[]
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_comando jsonb := p_artefacto->'comando';
    v_datos jsonb := p_artefacto #> '{comando,contexto,datos}';
    v_recibo jsonb := p_artefacto->'recibo';
    v_procedencia jsonb := v_recibo->'procedencia';
    v_evidencia_nominal jsonb := v_procedencia->'evidencia';
    v_prefijo text;
    v_peticion text;
    v_respuesta text;
BEGIN
    v_prefijo :=
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'esquema', 'vec.contratacion-temporal.integracion-bolsa.v1');
    v_peticion := v_prefijo ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'tipo', 'peticion-llamamiento') ||
        vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(v_datos) ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'necesidad', v_comando->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'bolsa', v_comando->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'orden', v_comando->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'politica', v_comando->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'total_posiciones_orden', v_comando->>'total_posiciones_orden') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'maxima_posicion_evaluable', v_comando->>'maxima_posicion_evaluable') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'huella_recibo_orden', v_comando->>'huella_recibo_orden');

    v_respuesta := v_prefijo ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'tipo', 'recibo-llamamiento-durable') ||
        vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(v_datos) ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_operacion_ref', v_recibo->>'operacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_organizacion_ref', v_recibo->>'organizacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_expediente_ref', v_recibo->>'expediente_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_version_expediente', v_recibo->>'version_expediente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_correlacion_ref', v_recibo->>'correlacion_ref') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'respuesta_necesidad', v_recibo->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'respuesta_resultado', v_recibo->'resultado');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'autoridad_ref', v_procedencia->>'autoridad_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_ref', v_procedencia->>'respuesta_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'procedencia_contrato_version', v_procedencia->>'contrato_version') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'fuente', v_procedencia->'fuente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_ref', v_evidencia_nominal->>'evidencia_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'clave_verificacion_ref', v_evidencia_nominal->>'clave_verificacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_emitida_en', v_evidencia_nominal->>'emitida_en') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_valida_hasta', v_evidencia_nominal->>'valida_hasta') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_retener_hasta', v_evidencia_nominal->>'retener_hasta');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_necesidad', v_comando->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_bolsa', v_comando->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_orden', v_comando->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_politica', v_comando->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_total_posiciones', v_comando->>'total_posiciones_orden') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_maxima_posicion', v_comando->>'maxima_posicion_evaluable') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_huella_recibo_orden', v_comando->>'huella_recibo_orden');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_bolsa', v_recibo->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_orden', v_recibo->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_politica', v_recibo->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recibo_propuesta_generada', v_recibo->>'propuesta_generada') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_propuesta', v_recibo->'propuesta') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_accion_evento', v_recibo->'accion_evento') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'llamamiento_ref', v_recibo->>'llamamiento_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'seleccion_ref_seudonimizada', v_recibo->>'seleccion_ref') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'retencion_seleccion', v_recibo->'retencion_seleccion') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'orden_seleccionado', v_recibo->>'orden_seleccionado') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recibo_ref', v_recibo->>'recibo_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'auditoria_ref', v_recibo->>'auditoria_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evento_ref', v_recibo->>'evento_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'confirmada_en', v_recibo->>'confirmada_en') ||
        CASE WHEN v_recibo->'llamamiento_recuperado' = 'true'::jsonb THEN
            vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
                'recibo_llamamiento_recuperado', 'true')
        ELSE '' END;
    RETURN ARRAY[
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_peticion, 'UTF8')
        ), 'hex'),
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_respuesta, 'UTF8')
        ), 'hex')
    ];
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$function$;
DO $post$
DECLARE v_huella text; v_acl aclitem[]; v_config text[]; v_prop name;
BEGIN
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),p.proacl,p.proconfig,r.rolname
 INTO v_huella,v_acl,v_config,v_prop
 FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner
 WHERE p.oid='vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb)'::regprocedure;
 IF v_huella IS DISTINCT FROM 'f0c459c9d626818e109ab90d1c0687353db199a07aeb2325010ffdd0ea7e59f0' THEN
  RAISE EXCEPTION 'CT199: PARO clave=huellas_materiales.nueva actual=% esperado=f0c459c9d626818e109ab90d1c0687353db199a07aeb2325010ffdd0ea7e59f0',coalesce(v_huella,'ausente') USING ERRCODE='55000';
 END IF;
 IF v_prop IS DISTINCT FROM 'vec_contratacion_temporal_propietario'
    OR v_acl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario'::aclitem]
    OR v_config IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[] THEN
  RAISE EXCEPTION 'CT199: PARO clave=metadatos_finales actual=divergente esperado=sin_cambios' USING ERRCODE='55000';
 END IF;
END
$post$;
RESET ROLE;
COMMIT;
