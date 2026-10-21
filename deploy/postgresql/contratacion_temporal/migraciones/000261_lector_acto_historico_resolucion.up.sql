\set ON_ERROR_STOP on
-- CT261 / F1-27 A0. Relee el acto confirmado v8 desde un detalle actual posterior.
-- No añade acción, finalidad, permiso, datos personales ni efecto de negocio.
-- Preimagen: definición instalada de main 8fb5c08ce, SHA256 df811064b6199cb5059213a84fb1af3aa89dc07d23f50be56e1ede90303f2203.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000261',0));
DO $pre$
DECLARE f oid;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_preparacion_resolucion_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR f IS NULL
  OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
  OR to_regclass('vec_contratacion_temporal.resolucion_formalizacion') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_contratacion_temporal_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')='df811064b6199cb5059213a84fb1af3aa89dc07d23f50be56e1ede90303f2203')
 THEN RAISE EXCEPTION 'CT261: preimagen instalada o dependencia divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona numeric, p_perfil numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, preparacion jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE v_lectura record;
 v_propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
 v_original vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_resolucion vec_contratacion_temporal.resolucion_formalizacion%ROWTYPE;
 v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_confirmada vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_preparacion jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec:resolucion-formalizacion:dependencia:000025-000069',0));
 IF p_consulta.version_observada IS DISTINCT FROM 0::numeric THEN
 RAISE EXCEPTION 'preparación inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_lectura FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
 p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona,p_perfil,p_payload,p_sobre,p_evidencia,p_raiz);
 -- No consulta tablas de negocio antes de validar y consumir el permiso lector.
 IF v_lectura.expediente_ref IS DISTINCT FROM p_consulta.expediente_ref
 OR v_lectura.version_expediente < 7 THEN
 RAISE EXCEPTION 'preparación fuera de fase' USING ERRCODE='P0692'; END IF;
 SELECT v.* INTO STRICT v_actual FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
 WHERE a.expediente_ref=v_lectura.expediente_ref;
 IF v_actual.version IS DISTINCT FROM v_lectura.version_expediente
 OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref THEN
 RAISE EXCEPTION 'publicación divergente' USING ERRCODE='P0694'; END IF;
 SELECT * INTO STRICT v_original FROM vec_contratacion_temporal.expediente_version_integral ev
 WHERE ev.expediente_ref=v_lectura.expediente_ref AND version=7;
 SELECT * INTO STRICT v_propuesta FROM vec_contratacion_temporal.propuesta_formalizacion pf
 WHERE propuesta_ref=v_original.operacion_ref AND pf.expediente_ref=v_lectura.expediente_ref
 AND organizacion_ref=p_alcance.organizacion_ref AND version_resultante=7;
 IF v_original.agregado_json#>>'{actuaciones,6,accion_clave}' IS DISTINCT FROM 'registrar_propuesta_formalizacion'
 OR v_original.agregado_json#>>'{actuaciones,6,recibo_ref}' IS DISTINCT FROM v_propuesta.recibo_ref
 OR v_propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM v_propuesta.propuesta_ref
 OR v_original.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(v_original.agregado_json::text,'UTF8')),'hex') THEN
 RAISE EXCEPTION 'propuesta no confiable' USING ERRCODE='P0694'; END IF;
 SELECT * INTO v_resolucion FROM vec_contratacion_temporal.resolucion_formalizacion rf
 WHERE rf.expediente_ref=v_lectura.expediente_ref AND organizacion_ref=p_alcance.organizacion_ref
 AND propuesta_ref=v_propuesta.propuesta_ref;
 IF (v_lectura.version_expediente=7 AND FOUND) OR (v_lectura.version_expediente>=8 AND NOT FOUND) THEN
 RAISE EXCEPTION 'recibo no disponible' USING ERRCODE='P0694'; END IF;
 IF v_lectura.version_expediente>=8 THEN
  SELECT * INTO STRICT v_confirmada
  FROM vec_contratacion_temporal.expediente_version_integral ev
  WHERE ev.expediente_ref=v_lectura.expediente_ref AND ev.version=8;
  -- El acto v8 debe seguir en la misma posición y con el mismo recibo.
  -- Un expediente posterior puede haber avanzado de fase o estar cerrado.
  IF v_confirmada.origen_version IS DISTINCT FROM 'resolucion_formalizacion_o6'
   OR v_confirmada.fase_clave IS DISTINCT FROM 'nombramiento'
   OR v_confirmada.operacion_ref IS DISTINCT FROM v_resolucion.resolucion_ref
   OR v_confirmada.agregado_json_huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(v_confirmada.agregado_json::text,'UTF8')),'hex')
   OR v_actual.agregado_json_huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(v_actual.agregado_json::text,'UTF8')),'hex')
   OR v_confirmada.agregado_json#>>'{actuaciones,7,recibo_ref}' IS DISTINCT FROM v_resolucion.recibo_ref
   OR v_confirmada.agregado_json#>>'{actuaciones,7,accion_clave}' IS DISTINCT FROM 'registrar_resolucion_formalizacion'
   OR v_actual.agregado_json#>'{actuaciones,7}' IS DISTINCT FROM
      v_confirmada.agregado_json#>'{actuaciones,7}'
   OR v_resolucion.version_resultante IS DISTINCT FROM 8
   OR v_resolucion.recibo_json->>'ResolucionRef' IS DISTINCT FROM v_resolucion.resolucion_ref
   OR v_resolucion.recibo_json->>'ReciboRef' IS DISTINCT FROM v_resolucion.recibo_ref
   OR v_resolucion.recibo_json->>'AuditoriaRef' IS DISTINCT FROM v_resolucion.auditoria_ref
   OR v_resolucion.recibo_json->>'OutboxRef' IS DISTINCT FROM v_resolucion.evento_ref
   OR v_resolucion.recibo_json->>'DocumentoSHA256' IS DISTINCT FROM v_resolucion.material_json->>'DocumentoSHA256'
   OR v_resolucion.recibo_json->>'DocumentoRef' IS DISTINCT FROM v_resolucion.material_json->>'DocumentoRef'
   OR v_resolucion.recibo_json->'Solicitud' IS DISTINCT FROM v_resolucion.material_json->'Solicitud'
   OR v_resolucion.tipo_validacion IS DISTINCT FROM 'manual_de_ejercicio'
   OR v_resolucion.firma_oficial IS DISTINCT FROM false
   OR v_resolucion.eficacia_administrativa IS DISTINCT FROM false THEN
   RAISE EXCEPTION 'recibo no confiable' USING ERRCODE='P0694';
  END IF;
 END IF;
 v_preparacion:=jsonb_build_object('ExpedienteRef',v_lectura.expediente_ref,
 'PropuestaRef',v_propuesta.propuesta_ref,'VersionEsperada',7,'VersionActual',v_lectura.version_expediente,
 'Recibo',CASE WHEN v_lectura.version_expediente>=8 THEN v_resolucion.recibo_json ELSE NULL END);
 RETURN QUERY SELECT v_lectura.contenido_canonico,
 v_lectura.esquema,
 v_lectura.acceso_ref,
 v_lectura.secuencia,
 v_lectura.anterior_sha256,
 v_lectura.huella_sha256,
 v_lectura.vinculo_identidad_huella_sha256,
 v_lectura.alcance_huella_sha256,
 v_lectura.registrada_en,
 v_lectura.auditoria_vec_ref,
 v_lectura.auditoria_vec_huella_sha256,
 v_lectura.consumo_vec_huella_sha256,
 v_lectura.contenido_huella_sha256,
 v_lectura.resultado_huella_sha256,
 v_lectura.cursor_huella_sha256,
 v_lectura.generada_en,
 v_lectura.expediente_ref,
 v_lectura.version_expediente,
 v_lectura.total,
 v_lectura.recibo_sello_sha256, v_preparacion;
END $function$;


DO $post$
DECLARE f oid;
BEGIN
 f:='vec_contratacion_temporal.consultar_preparacion_resolucion_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='5183aa979f79b5cd637a7778e9012fb838564b57cc7e327a327cff7cb4e147a2'
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT261: contrato o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
