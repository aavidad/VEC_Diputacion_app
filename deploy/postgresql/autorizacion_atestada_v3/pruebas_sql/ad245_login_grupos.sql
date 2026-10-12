\set ON_ERROR_STOP on
-- Ejecutar sólo en PostgreSQL 18 desechable tras AUT75/AD245 como superusuario.
-- Roles sintéticos y membresías se revierten por completo.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $pre$
BEGIN
 IF to_regrole('vec_ad245_b1_prueba') IS NOT NULL
 OR to_regrole('vec_ad245_b103_prueba') IS NOT NULL
 THEN RAISE EXCEPTION 'AD245: roles de prueba ocupados' USING ERRCODE='55000';END IF;
END $pre$;
CREATE ROLE vec_ad245_b1_prueba LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ad245_b103_prueba LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
GRANT vec_admin_version_rol_bolsa_ejecutor TO vec_ad245_b1_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_admin_version_rol_bolsa_b103_ejecutor TO vec_ad245_b103_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
 TO vec_ad245_b1_prueba,vec_ad245_b103_prueba;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1(),
 vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1()
 TO vec_ad245_b1_prueba,vec_ad245_b103_prueba;
SET SESSION AUTHORIZATION vec_ad245_b1_prueba;
DO $b1$
BEGIN
 IF vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1() IS NOT TRUE
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1() IS TRUE
 THEN RAISE EXCEPTION 'AD245: LOGIN B1 no aislado' USING ERRCODE='42501';END IF;
END $b1$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ad245_b103_prueba;
DO $b103$
BEGIN
 IF vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1() IS NOT TRUE
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AD245: LOGIN B103 no aislado' USING ERRCODE='42501';END IF;
END $b103$;
RESET SESSION AUTHORIZATION;
GRANT vec_admin_version_rol_bolsa_ejecutor TO vec_ad245_b103_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_ad245_b103_prueba;
DO $doble$
BEGIN
 IF vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1() IS TRUE
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1() IS TRUE
 THEN RAISE EXCEPTION 'AD245: membresia doble aceptada' USING ERRCODE='42501';END IF;
END $doble$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
