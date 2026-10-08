\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:orq1:ad225',0));
-- Parche del bloque completo de reanudación sobre dos preimágenes instaladas:
-- postHX+HZ14 y post AD211/214/216/218. El resto del núcleo queda literal.
DO $parche$
DECLARE
 f oid := to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; meta jsonb; deps jsonb; compartidas jsonb;
 actual text; nuevo text; marca text := $marca225$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_seleccion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'reanudacion_seleccion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$marca225$;
 ampliacion text := $ampliacion225$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_seleccion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'reanudacion_seleccion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_solicitud_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_solicitud'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_solicitud'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'reanudacion_seleccion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$ampliacion225$;
 def_sha text; src_sha text;
BEGIN
 IF f IS NULL THEN
  RAISE EXCEPTION 'AD225: PARO clave=nucleo actual=ausente esperado=funcion_instalada' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD225: PARO clave=fachada actual=presente esperado=ausente' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc'
 INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 def_sha:=encode(sha256(convert_to(original,'UTF8')),'hex');
 src_sha:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF NOT ((def_sha='092367a3c6be54e163eceb26d58a86442f83addc044e0cae2e85e572c7e56faf'
          AND src_sha='559555ec535ad40cc3aad6361286899c28aede91ff73b2901eb30970d95ac986')
       OR (def_sha='c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714'
          AND src_sha='79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5')) THEN
  RAISE EXCEPTION 'AD225: PARO clave=nucleo_preimagen actual=def:%/src:% esperado=postHX:092367a3/559555ec_o_postAD218:c74551ea/79d2f297',def_sha,src_sha USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND a.grantee=p.proowner AND a.grantor=p.proowner
    AND a.privilege_type='EXECUTE' AND NOT a.is_grantable) THEN
  RAISE EXCEPTION 'AD225: PARO clave=nucleo_meta_acl actual=divergente esperado=owner_definer_volatil_una_acl_owner_search_path_fijo' USING ERRCODE='55000';
 END IF;
 IF strpos(original,marca)=0
    OR strpos(substr(original,strpos(original,marca)+length(marca)),marca)<>0
    OR strpos(original,'reanudacion_solicitud_llamamiento')<>0
    OR strpos(original,'contratacion_temporal.llamamiento.reanudar_solicitud')<>0 THEN
  RAISE EXCEPTION 'AD225: PARO clave=marca actual=ausente_duplicada_o_accion_existente esperado=una_marca_y_accion_ausente' USING ERRCODE='55000';
 END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(original,marca,ampliacion);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(actual,ampliacion,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
         AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'AD225: PARO clave=nucleo_postimagen actual=divergente esperado=solo_ampliacion_y_meta_acl_dependencias_identicas' USING ERRCODE='55000';
 END IF;
END
$parche$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET lock_timeout = '2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'reanudacion_solicitud_llamamiento',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'reanudación requiere consumo nuevo' USING ERRCODE = '42501';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor, vec_autorizacion_atestada_v3_emisor,
    vec_contratacion_temporal_ejecutor, vec_contratacion_temporal_migrador,
    vec_bolsa_llamamientos_propietario, vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;
COMMIT;
