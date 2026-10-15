\set ON_ERROR_STOP on
-- Ejecutar en copia desechable después de CT221.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p,
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
   WHERE p.oid='vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)'::regprocedure
     AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)',
  'EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT221: concesión entre módulos incorrecta'; END IF;
END $test$;
ROLLBACK;
