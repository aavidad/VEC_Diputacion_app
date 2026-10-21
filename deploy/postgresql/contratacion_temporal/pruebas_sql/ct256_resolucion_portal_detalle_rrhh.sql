\set ON_ERROR_STOP on
-- CT256 / F1-127. prueba focal del lector post CT256; solo base desechable, ROLLBACK.
-- Solo focal de metadatos, fuente del lector y denegación sin contexto V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $ct256$
DECLARE p pg_proc%ROWTYPE; fuente text; rechazado boolean:=false;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY[
  'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s',
  'statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
  '7748e4d63deb1766850aeb8596c5eca996705176e2de0cb90312acbd2a14cf0d'
 THEN RAISE EXCEPTION 'CT256 prueba: definición, owner o ACL divergentes' USING ERRCODE='55000'; END IF;
 fuente:=p.prosrc;
 IF strpos(fuente,'''resolucion_ct''')=0
 OR strpos(fuente,'jsonb_array_length(opacas)=1')=0
 OR strpos(fuente,'leer_resultados_emisiones_ct_v1')=0
 OR strpos(fuente,'consultar_detalle_rrhh_atestado_v1')=0
 OR strpos(fuente,'r.recibo_json->>''ResolucionRef''=r.resolucion_ref')=0
 OR strpos(fuente,'r.recibo_respuesta_portal_ref=part.valor->>''recibo_respuesta_ref''')=0
 THEN RAISE EXCEPTION 'CT256 prueba: proyección o lectura previa ausente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
   NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
   NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
   NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
   NULL::numeric,NULL::numeric,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,NULL::text);
 EXCEPTION WHEN insufficient_privilege THEN rechazado:=true;
 END;
 IF NOT rechazado THEN RAISE EXCEPTION 'CT256 prueba: lectura sin contexto admitida' USING ERRCODE='42501'; END IF;
END $ct256$;
ROLLBACK;
