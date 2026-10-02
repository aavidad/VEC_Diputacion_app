\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000011',0));
DO $guarda$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_perfiles_v1') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.sesion_admin_copias_v1) THEN RAISE EXCEPTION 'IS11: historia o consumidor posterior conservado, retirada denegada' USING ERRCODE='55000'; END IF;
END $guarda$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text);
DROP FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz,text,text);
DROP FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz);
DROP FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz);
DROP TABLE vec_identidad_sesiones_v1.sesion_admin_copias_v1;
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM vec_identidad_sesiones_v1_admin_copias;
DO $acl$ BEGIN IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 WHERE NOT uso_ad3_previo) THEN REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM vec_autorizacion_atestada_v3_propietario; END IF; END $acl$;
DROP TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1;
COMMIT;
