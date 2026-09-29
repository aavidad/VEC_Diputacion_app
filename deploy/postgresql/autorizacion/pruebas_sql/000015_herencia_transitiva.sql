\set ON_ERROR_STOP on
-- La prueba de grupos usa un LOGIN sintético nominal y se revierte entera.
BEGIN;
CREATE ROLE vec_externo_v3_fuente_autorizacion_desarrollo LOGIN INHERIT
 NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_fuente_externa TO vec_externo_v3_fuente_autorizacion_desarrollo
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text)
 TO vec_externo_v3_fuente_autorizacion_desarrollo;
SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
DO $positivo$ BEGIN
 IF vec_autorizacion.login_candidato_externo_v1(
      'vec_externo_v3_fuente_autorizacion_desarrollo','vec_autorizacion_fuente_externa') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT-15: login nominal rechazado'; END IF;
END $positivo$;
RESET SESSION AUTHORIZATION;
CREATE ROLE vec_autorizacion_prueba_herencia NOLOGIN;
GRANT vec_autorizacion_prueba_herencia TO vec_autorizacion_fuente_externa;
SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
DO $negativo$ BEGIN
 IF vec_autorizacion.login_candidato_externo_v1(
      'vec_externo_v3_fuente_autorizacion_desarrollo','vec_autorizacion_fuente_externa') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT-15: herencia transitiva admitida'; END IF;
END $negativo$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
