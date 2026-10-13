\set ON_ERROR_STOP on
-- P49: prueba vigente del mismo plan P23 para el propietario CT, cuando
-- la sesión nominal ejecutora de Bolsa prepara el primer alta B115.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000049',0));
DO $pre$
DECLARE f regprocedure:=to_regprocedure('vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb)');
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM 'c9703a02ea06ea9bd90077e9678354973fd50b1a39352dcca8e16bebaa944ba4'
    OR (SELECT proowner<>'vec_personal_propietario'::regrole OR NOT prosecdef
       OR proacl IS DISTINCT FROM ARRAY[
         'vec_personal_propietario=X/vec_personal_propietario',
         'vec_contratacion_temporal_propietario=X/vec_personal_propietario']::aclitem[]
       OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp',
         'row_security=on','lock_timeout=2s']::text[]
       FROM pg_proc WHERE oid=f) IS NOT FALSE
    OR to_regclass('vec_personal.plan_incorporacion_ct') IS NULL
    OR to_regprocedure('vec_personal.estado_plan_ct_interno(vec_personal.plan_incorporacion_ct)') IS NULL
    OR to_regprocedure('vec_personal.probar_origen_incorporacion_plan_bolsa_v1(text,text,text,bigint,text,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'P49: preimagen P23 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_personal.probar_origen_incorporacion_plan_bolsa_v1(
 p_org text,p_exp text,p_ct_plan_ref text,p_ct_plan_version bigint,p_ct_sha text,p_hechos jsonb
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE p vec_personal.plan_incorporacion_ct%ROWTYPE; est jsonb; a jsonb; o jsonb;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'P49: sesión delegada denegada' USING ERRCODE='42501'; END IF;
 IF p_org IS NULL OR p_exp IS NULL OR p_ct_plan_ref IS NULL OR p_ct_plan_version IS DISTINCT FROM 1
    OR p_ct_sha IS NULL OR p_ct_sha !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_hechos) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'P49: selector de origen inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO p FROM vec_personal.plan_incorporacion_ct
 WHERE plan_ref=p_hechos->>'personal_plan_ref' AND organismo_ref=p_org;
 IF NOT FOUND THEN RETURN NULL; END IF;
 est:=vec_personal.estado_plan_ct_interno(p); a:=est->'recibo_alta_relacion'; o:=est->'recibo_ocupacion';
 IF est->>'estado' IS DISTINCT FROM 'ejecutado'
    OR p.datos->>'expediente_ref' IS DISTINCT FROM p_exp
    OR p.datos->>'origen_ct_ref' IS DISTINCT FROM p_ct_plan_ref
    OR p.datos->>'origen_ct_huella_sha256' IS DISTINCT FROM p_ct_sha
    OR p_hechos->>'personal_plan_version' IS DISTINCT FROM '1'
    OR p_hechos->>'personal_plan_recibo_ref' IS DISTINCT FROM p.recibo_ref
    OR p_hechos->>'personal_plan_sha256' IS DISTINCT FROM p.huella_sha256
    OR p_hechos->>'modo_personal' IS DISTINCT FROM p.modo
    OR p_hechos->>'alta_recibo_ref' IS DISTINCT FROM (CASE WHEN p.modo='alta_empleado' THEN a->>'recibo_ref' ELSE '' END)
    OR p_hechos->>'empleado_ref' IS DISTINCT FROM a->>'empleado_ref'
    OR p_hechos->>'relacion_ref' IS DISTINCT FROM a->>'relacion_ref'
    OR p_hechos->>'relacion_version' IS DISTINCT FROM a->>'version'
    OR p_hechos->>'relacion_recibo_ref' IS DISTINCT FROM a->>'recibo_ref'
    OR p_hechos->>'ocupacion_ref' IS DISTINCT FROM o->>'hecho_ref'
    OR p_hechos->>'ocupacion_version' IS DISTINCT FROM o->>'version'
    OR p_hechos->>'ocupacion_recibo_ref' IS DISTINCT FROM o->>'recibo_ref'
    OR p_hechos->>'rpt_confirmacion_ref' IS DISTINCT FROM p.confirmacion_rpt_ref
    OR p_hechos->>'rpt_recibo_ref' IS NULL OR p_hechos->>'rpt_recibo_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR (SELECT max(revision) FROM vec_personal.relacion_servicio_historia WHERE relacion_ref=a->>'relacion_ref')::text IS DISTINCT FROM a->>'version'
    OR (SELECT max(revision) FROM vec_personal.ocupacion_empleado_historia WHERE ocupacion_ref=o->>'hecho_ref')::text IS DISTINCT FROM o->>'version' THEN
  RAISE EXCEPTION 'P49: origen o hechos vigentes divergentes' USING ERRCODE='42501'; END IF;
 RETURN p_hechos;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.probar_origen_incorporacion_plan_bolsa_v1(text,text,text,bigint,text,jsonb)
 FROM PUBLIC,vec_personal_ejecutor,vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.probar_origen_incorporacion_plan_bolsa_v1(text,text,text,bigint,text,jsonb)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_personal.probar_origen_incorporacion_plan_bolsa_v1(text,text,text,bigint,text,jsonb)'::regprocedure;
BEGIN
 IF (SELECT proowner<>'vec_personal_propietario'::regrole OR NOT prosecdef
       OR proacl IS DISTINCT FROM ARRAY[
        'vec_personal_propietario=X/vec_personal_propietario',
        'vec_contratacion_temporal_propietario=X/vec_personal_propietario']::aclitem[]
       FROM pg_proc WHERE oid=f) IS NOT FALSE
    OR has_function_privilege('public',f,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'P49: helper expuesto' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
