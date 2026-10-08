\set ON_ERROR_STOP on
-- CT197. Testigo técnico de la auditoría común del único acto CT115.
-- Bolsa recibe la proyección de un cese publicado, sin decidir ni auditar un
-- segundo acto. CT129 acredita el origen antes de revelar su auditoria_ref.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000197',0));

DO $pre$
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario'
    OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
    OR to_regrole('vec_bolsa_llamamientos_relevo_cese') IS NULL
    OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
       WHERE p.oid='vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)'::regprocedure
         AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
         AND 'row_security=on'=ANY(p.proconfig)) THEN
  RAISE EXCEPTION 'CT197: clave=preimagen_cese_bolsa esperado=CT129_instalada actual=incompatible_o_ya_instalada'
   USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(
 p_origen_ref text,p_huella_sha256 text,p_posicion bigint)
RETURNS TABLE(origen_ref text,auditoria_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE v_ct record; v_cese record; v_hay_cese boolean;
 v_marca_previa text:=current_setting('vec.ct129.origen_ref',true);
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,to_regrole('vec_bolsa_llamamientos_relevo_cese'),'MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'CT197: testigo de cese no autorizado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO v_ct FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(
  p_origen_ref,p_huella_sha256,p_posicion);
 IF NOT FOUND THEN
  RAISE EXCEPTION 'CT197: cese publicado no acreditado' USING ERRCODE='42501';
 END IF;
 -- La política RLS CT129 exige esta marca y el LOGIN exclusivo del relevo.
 -- Nunca se concede SELECT de cese_nombramiento_v1 al propietario de Bolsa.
 PERFORM set_config('vec.ct129.origen_ref',p_origen_ref,true);
 SELECT c.evento_ref,c.recibo_ref,c.organizacion_ref,c.expediente_ref,
        c.auditoria_ref,c.recibo_json->>'auditoria_ref' AS auditoria_recibo
   INTO v_cese
 FROM vec_contratacion_temporal.cese_nombramiento_v1 c
 WHERE c.evento_ref=p_origen_ref AND c.estado='confirmada';
 v_hay_cese:=FOUND;
 PERFORM set_config('vec.ct129.origen_ref',coalesce(v_marca_previa,''),true);
 IF NOT v_hay_cese OR v_cese.evento_ref IS DISTINCT FROM p_origen_ref
    OR v_cese.recibo_ref IS DISTINCT FROM v_ct.recibo_ref
    OR v_cese.organizacion_ref IS DISTINCT FROM v_ct.organizacion_ref
    OR v_cese.expediente_ref IS DISTINCT FROM v_ct.expediente_ref
    OR v_cese.auditoria_ref IS NULL OR v_cese.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
    OR v_cese.auditoria_recibo IS DISTINCT FROM v_cese.auditoria_ref THEN
  RAISE EXCEPTION 'CT197: auditoría del cese divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT p_origen_ref,v_cese.auditoria_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)
 TO vec_bolsa_llamamientos_propietario;

DO $post$
DECLARE f regprocedure:='vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT 'search_path=pg_catalog, pg_temp'=ANY(proconfig) FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR EXISTS (SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee NOT IN
        (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole))
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese',f,'EXECUTE')
    OR has_table_privilege('vec_bolsa_llamamientos_propietario',
       'vec_contratacion_temporal.cese_nombramiento_v1','SELECT,INSERT,UPDATE,DELETE') THEN
  RAISE EXCEPTION 'CT197: clave=postimagen_testigo esperado=solo_owner_CT_y_owner_Bolsa actual=ACL_o_definicion_divergente'
   USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
