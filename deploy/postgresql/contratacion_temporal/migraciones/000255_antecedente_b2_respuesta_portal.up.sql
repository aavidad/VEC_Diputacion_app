\set ON_ERROR_STOP on
-- F1-134: el plan de Personal usa la aceptación real de Mi Bolsa.
-- Después de CT249 y CT253; no modifica las ramas de no incorporación.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000255',0));
DO $pre$
DECLARE f oid:='vec_contratacion_temporal.antecedentes_plan_personal_ct155(text,text)'::regprocedure;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
    IS DISTINCT FROM '2613f213e0da4c31c6d9899d1b5460c60e570f47ef662df0c28e71d5b50460fc'
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND NOT p.prosecdef AND p.provolatile='s'
    AND p.proconfig=ARRAY['search_path=pg_catalog']::text[]
    AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[])
 OR to_regprocedure('vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.antecedentes_plan_personal_ct212(text,text)') IS NULL
 THEN RAISE EXCEPTION 'CT255: preimagen o dependencias divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.antecedentes_plan_personal_ct155(org text, exp text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE a record; p record; v record; r record; e record; rf record; apertura_sha text;propuesta_bolsa text;
BEGIN
 SELECT * INTO a FROM vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(org,exp);
 IF NOT FOUND THEN RETURN NULL; END IF;
 -- Ambas fuentes usan el mismo orden vigente de confirmación y referencia.
 -- La pareja portal conserva su recibo real, sin justificante de correo.
 -- CT212 coteja después el selector, la persona y la respuesta de Bolsa.
 SELECT candidata.* INTO p FROM (
  SELECT pf.* FROM vec_contratacion_temporal.aceptacion_vigente_ct124(org,exp) av
   JOIN vec_contratacion_temporal.propuesta_formalizacion pf ON pf.propuesta_ref=av.propuesta_ref
  UNION ALL
  SELECT pf.*
   FROM vec_contratacion_temporal.propuesta_formalizacion pf
   JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh rr
     ON rr.resolucion_ref=pf.resolucion_ref
   WHERE pf.organizacion_ref=org AND pf.expediente_ref=exp
     AND rr.organizacion_ref=org AND rr.expediente_ref=exp
     AND rr.llamamiento_ref=pf.llamamiento_ref
     AND rr.fuente_respuesta='portal' AND rr.estado='confirmado'
     AND rr.solicitud_json->>'Respuesta'='aceptacion'
     AND rr.comunicacion_ref IS NULL AND rr.justificante_ref IS NULL
     AND rr.continuacion_clave IS NULL
     AND pf.material_json ? 'AceptacionBolsaNativa'
     AND NOT (pf.material_json ? 'AceptacionBolsa')
     AND pf.aceptacion_bolsa_json=pf.material_json->'AceptacionBolsaNativa'
     AND pf.material_json#>>'{Solicitud,OrganizacionRef}'=org
     AND pf.material_json#>>'{Solicitud,ExpedienteRef}'=exp
     AND pf.material_json#>>'{Solicitud,ResolucionLlamamientoAceptadaRef}'=rr.resolucion_ref
     AND pf.material_json#>>'{Solicitud,ReciboResolucionAceptadaRef}'=rr.recibo_ref
     AND pf.material_json#>>'{AceptacionBolsaNativa,RespuestaRef}'=rr.respuesta_portal_ref
     AND pf.material_json#>>'{AceptacionBolsaNativa,Selector,recibo_respuesta_ref}'=rr.recibo_respuesta_portal_ref
 ) candidata
 ORDER BY candidata.confirmada_en DESC,candidata.propuesta_ref COLLATE "C" DESC LIMIT 1;
 IF NOT FOUND OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n WHERE n.organizacion_ref=org AND n.expediente_ref=exp AND n.aceptacion_resolucion_ref=p.resolucion_ref) THEN RETURN NULL; END IF;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE resolucion_ref=p.resolucion_ref;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 WHERE clave_idempotencia=r.seleccion_clave;
 SELECT * INTO v FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 WHERE organizacion_ref=org AND expediente_ref=exp ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT coalesce(x.continuacion_recibo#>>'{ReciboBolsa,RegistroSHA256}',''),x.continuacion_recibo#>>'{ReciboBolsa,PropuestaRef}' INTO apertura_sha,propuesta_bolsa FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh x
 WHERE x.organizacion_ref=org AND x.expediente_ref=exp AND x.continuacion_recibo#>>'{ReciboBolsa,OperacionRef}'=p.aceptacion_bolsa_json->>'AperturaOperacionRef' LIMIT 1;
 SELECT * INTO rf FROM vec_contratacion_temporal.resolucion_formalizacion WHERE organizacion_ref=org AND expediente_ref=exp AND propuesta_ref=p.propuesta_ref;
 RETURN jsonb_build_object('organizacion_ref',org,'expediente_ref',exp,'documento_ref',coalesce(rf.recibo_json->>'DocumentoRef',''),'documento_sha256',coalesce(rf.recibo_json->>'DocumentoSHA256',''),'documento_version',coalesce((rf.recibo_json->>'DocumentoVersion')::numeric,0),'version_expediente',a.version_expediente,
 'analisis_version',a.analisis_version,'analisis_recibo_ref',a.analisis_recibo_ref,'analisis_sha256',a.analisis_huella_sha256,'categoria_ref',a.categoria_ref,
 'propuesta_recibo_ref',p.recibo_ref,'aceptacion_ref',p.resolucion_ref,'aceptacion_recibo_ref',r.recibo_ref,
 'bolsa',jsonb_build_object('unidad_ref',(SELECT vv.agregado_json#>>'{asignacion,unidad_ref}' FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_version_integral vv ON vv.expediente_ref=aa.expediente_ref AND vv.version=aa.version WHERE aa.expediente_ref=exp),'categoria_ref',a.categoria_ref,
 'necesidad_ref',e.recibo_json#>>'{necesidad,referencia}','aceptacion_operacion_ref',p.aceptacion_bolsa_json->>'OperacionRef','aceptacion_registro_sha256',p.aceptacion_bolsa_json->>'RegistroSHA256',
 'apertura_operacion_ref',p.aceptacion_bolsa_json->>'AperturaOperacionRef','apertura_registro_sha256',coalesce(apertura_sha,''),'llamamiento_ref',p.llamamiento_ref,'propuesta_ref',coalesce(propuesta_bolsa,e.recibo_json#>>'{propuesta,referencia}')),
 'vinculo',v.recibo_json->'vinculo');
END $function$;

DO $post$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p
  WHERE p.oid='vec_contratacion_temporal.antecedentes_plan_personal_ct155(text,text)'::regprocedure
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND NOT p.prosecdef AND p.provolatile='s'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
    AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT255: contrato o permisos divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
