-- BORRADOR: retirada solo en base efímera sin historia; no ejecutar en principal.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000012',0));
DO $guarda$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.sesion_admin_perfiles_v1) THEN RAISE EXCEPTION 'IS12: historia conservada, retirada denegada' USING ERRCODE='55000'; END IF;
END $guarda$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text);
DROP FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text);
DROP FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz);
DROP FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz);
DROP TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1;
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM vec_identidad_sesiones_v1_admin_perfiles;
DO $acl$ BEGIN IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 WHERE NOT uso_ad3_previo) THEN REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM vec_autorizacion_atestada_v3_propietario; END IF; END $acl$;
DROP TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1;
COMMIT;
