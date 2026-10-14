\set ON_ERROR_STOP on
-- Dietas19: desambiguar el recibo en la creación y recuperación catalogadas.
-- Cuerpo obtenido de la copia HZ37 tras Dietas17, con sus permisos intactos.
-- No cambia autorización, cálculo, estado, historia ni significado de la clave.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000019',0));
DO $preimagen$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_dietas_propietario'
    OR to_regprocedure('vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Dietas19: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM '69f7258061b734b13a9cd437483cd6908af1314a31565db84d32523c4bb35ee6'
    OR p.proowner IS DISTINCT FROM 'vec_dietas_propietario'::regrole
    OR p.prosecdef IS NOT TRUE
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
 THEN RAISE EXCEPTION 'Dietas19: preimagen de cuerpo o permisos divergente' USING ERRCODE='55000'; END IF;
END $preimagen$;

CREATE OR REPLACE FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE m jsonb; c jsonb; calc jsonb; salida jsonb; anterior jsonb;
        ref text; v_recibo_ref text; repeticion boolean;
        regla vec_dietas.regla_devengo_registrada%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN
  RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; c:=m->'comando'; calc:=c->'calculo';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END;
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1'
    OR m->>'operacion'<>'crear' OR m->>'recurso_ref'<>'dietas:borradores:propios'
    OR jsonb_typeof(calc) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END IF;
 -- El núcleo v1 consume AD3-49, liga material completo, revalida Personal y
 -- crea el recibo base. Cualquier rechazo posterior revierte TODA la tx.
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
  p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ref:=salida->'comision'->>'referencia';
 v_recibo_ref:=salida->'recibo'->>'referencia';
 repeticion:=(salida->'recibo'->>'repeticion')::boolean;
 IF ref !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR v_recibo_ref !~ '^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR repeticion IS NULL THEN RAISE EXCEPTION 'alta Dietas no reconciliable' USING ERRCODE='PD003'; END IF;
 IF NOT repeticion THEN
  IF calc->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
     OR vec_dietas.validar_calculo_catalogado_v2(c) IS NOT TRUE THEN
   RAISE EXCEPTION 'cálculo catalogado Dietas inválido' USING ERRCODE='PD001'; END IF;
  SELECT * INTO STRICT regla FROM vec_dietas.regla_devengo_registrada
   WHERE regla_ref=calc->>'regla_ref' AND version_ref=calc->>'version_tarifa'
     AND huella_sha256=calc->>'regla_huella_sha256' AND origen='publicada';
  INSERT INTO vec_dietas.calculo_comision(comision_ref,calculo,version_tarifa,registrado_en)
   VALUES(ref,calc,calc->>'version_tarifa',clock_timestamp());
  INSERT INTO vec_dietas.recibo_regla_creacion_comision
   (comision_ref,recibo_ref,regla_ref,regla_huella_sha256,registrada_en)
   VALUES(ref,v_recibo_ref,regla.regla_ref,regla.huella_sha256,clock_timestamp());
 END IF;
 SELECT x.calculo INTO STRICT anterior FROM vec_dietas.calculo_comision x WHERE x.comision_ref=ref;
 IF anterior ? 'regla_ref' AND NOT EXISTS (
   SELECT 1 FROM vec_dietas.recibo_regla_creacion_comision x
   WHERE x.comision_ref=ref AND x.recibo_ref=v_recibo_ref
     AND x.regla_ref=anterior->>'regla_ref'
     AND x.regla_huella_sha256=anterior->>'regla_huella_sha256') THEN
  RAISE EXCEPTION 'recibo de regla Dietas incoherente' USING ERRCODE='PD003'; END IF;
 RETURN jsonb_set(salida,'{comision,calculo}',anterior,true);
END $function$;

DO $postimagen$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_dietas_propietario'
    OR to_regprocedure('vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Dietas19: postimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM 'd58e2f400e5e2413c6916198c66fdc7af082fd8dbba3bc79b405cc4caf4125cd'
    OR p.proowner IS DISTINCT FROM 'vec_dietas_propietario'::regrole
    OR p.prosecdef IS NOT TRUE
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
 THEN RAISE EXCEPTION 'Dietas19: postimagen de cuerpo o permisos divergente' USING ERRCODE='55000'; END IF;
END $postimagen$;
COMMIT;
