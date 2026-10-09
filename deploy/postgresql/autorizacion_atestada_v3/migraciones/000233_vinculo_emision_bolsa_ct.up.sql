\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000233',0));

-- Partimos del cuerpo instalado en el clon postHX+HZ15. Una definición
-- distinta se detiene y obliga a reconstruir el parche desde esa preimagen.
DO $parche$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; marca text; ampliacion text; meta jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD233: PARO clave=preimagen esperado=nucleo_sin_fachada actual=incompatible' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
      '03f151286ed21039cfe2f96840f474062a8aecb61c546601ef71cf20cc716212'
    OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM
      'fc80e4851d7a63a147cce6a40ca924d24d0b5e671686134be90e98e0f53c906c'
    OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'AD233: PARO clave=nucleo_preimagen esperado=postHZ15:03f15128/fc80e485 actual=otro_hash_o_meta' USING ERRCODE='55000';
 END IF;
 marca:=$m$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'asignacion'$m$;
 ampliacion:=$a$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_emision_bolsa_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.vinculo_emision_bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.bolsa.vincular'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_bolsa_expediente'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'asignacion'$a$;
 IF length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'vinculo_emision_bolsa_ct')<>0 THEN
  RAISE EXCEPTION 'AD233: PARO clave=marca esperado=una_sin_perfil actual=ausente_duplicada' USING ERRCODE='55000';
 END IF;
 nuevo:=replace(original,marca,ampliacion);
 EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta THEN
  RAISE EXCEPTION 'AD233: PARO clave=postimagen esperado=solo_perfil_nuevo actual=meta_o_definicion_divergente' USING ERRCODE='55000';
 END IF;
END $parche$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'vinculo_emision_bolsa_ct',p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD233: consumo previo rechazado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,
  v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;
COMMIT;
