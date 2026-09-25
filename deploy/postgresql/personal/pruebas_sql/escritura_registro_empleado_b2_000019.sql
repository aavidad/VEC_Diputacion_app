\set ON_ERROR_STOP on
-- Ejecutar en PostgreSQL 18 tras ContextoActor 000008, AD3-54 y Personal
-- 000016/000017/000019. Los casos positivos V3 requieren atestación real y
-- se prueban por el adaptador HTTP con identidad sintética.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
DO $focal$
DECLARE firma text:='text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea';
 f regprocedure; p record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  ('vec_personal.registrar_empleado_rrhh_v1('||firma||')')::regprocedure,
  ('vec_personal.registrar_hecho_empleado_rrhh_v1('||firma||')')::regprocedure]
 LOOP
  SELECT proowner,prosecdef,provolatile,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_personal_propietario'::regrole OR NOT p.prosecdef
     OR p.provolatile<>'v' OR NOT ('row_security=on'=ANY(p.proconfig))
     OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc px CROSS JOIN LATERAL
         aclexplode(coalesce(px.proacl,acldefault('f',px.proowner))) a
         WHERE px.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE') THEN
    RAISE EXCEPTION '000019: firma o ACL de escritura incompatible %',f;
  END IF;
 END LOOP;
 IF has_function_privilege('vec_personal_ejecutor',
     'vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
     'EXECUTE')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.registro_empleado_b2_recibo','SELECT')
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
       WHERE oid='vec_personal.registro_empleado_b2_recibo'::regclass)
    OR (SELECT count(*) FROM pg_policies WHERE schemaname='vec_personal'
        AND tablename='registro_empleado_b2_recibo')<>1 THEN
   RAISE EXCEPTION '000019: recibo o helper expuesto';
 END IF;
END $focal$;
-- Incluso con transacción SERIALIZABLE, una solicitud sin material V3 se deniega.
DO $focal$
BEGIN
 BEGIN
  PERFORM vec_personal.registrar_empleado_rrhh_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION '000019: alta sin V3 admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.registrar_hecho_empleado_rrhh_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION '000019: hecho sin V3 admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 IF (SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo)<>0 THEN
  RAISE EXCEPTION '000019: denegación dejó recibos'; END IF;
END $focal$;
ROLLBACK;
