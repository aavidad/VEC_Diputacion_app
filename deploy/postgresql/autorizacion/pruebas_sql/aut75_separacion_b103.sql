\set ON_ERROR_STOP on
-- Ejecutar sólo en PostgreSQL 18 desechable tras AUT75 y AD245. ROLLBACK final.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE b1 text:='vec_admin_version_rol_bolsa_ejecutor';
 b103 text:='vec_admin_version_rol_bolsa_b103_ejecutor';
 nueva text:='vec_autorizacion.proponer_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 cierre text:='vec_autorizacion.cerrar_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 anterior text:='vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 tabla text;
BEGIN
 IF to_regrole(b103) IS NULL OR to_regprocedure(nueva) IS NULL
 OR to_regprocedure(cierre) IS NULL OR to_regprocedure(anterior) IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AUT75: falta pieza del corte' USING ERRCODE='55000';END IF;
 IF has_function_privilege(b1,nueva,'EXECUTE') IS TRUE
 OR has_function_privilege(b1,cierre,'EXECUTE') IS TRUE
 OR has_function_privilege(b103,anterior,'EXECUTE') IS TRUE
 OR has_function_privilege('public',nueva,'EXECUTE') IS TRUE
 OR has_function_privilege('public',cierre,'EXECUTE') IS TRUE
 OR has_function_privilege(b103,nueva,'EXECUTE') IS NOT TRUE
 OR has_function_privilege(b103,cierre,'EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT75: ACL B1/B103 mezclada' USING ERRCODE='42501';END IF;
 FOREACH tabla IN ARRAY ARRAY['propuesta_version_rol_bolsa_v2',
  'cierre_version_rol_bolsa_v2','outbox_version_rol_bolsa_v2'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_autorizacion.'||tabla)
   AND c.relowner=to_regrole('vec_autorizacion_propietario')
   AND c.relrowsecurity AND c.relforcerowsecurity)
  OR has_table_privilege(b1,'vec_autorizacion.'||tabla,'SELECT,INSERT,UPDATE,DELETE') IS TRUE
  OR has_table_privilege(b103,'vec_autorizacion.'||tabla,'SELECT,INSERT,UPDATE,DELETE') IS TRUE
  OR has_table_privilege('public','vec_autorizacion.'||tabla,'SELECT,INSERT,UPDATE,DELETE') IS TRUE
  THEN RAISE EXCEPTION 'AUT75: tabla o RLS divergente %',tabla USING ERRCODE='42501';END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
