\set ON_ERROR_STOP on
-- Sonda transaccional: crea solo actores sintéticos y revierte todo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $roles$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_ad3_118_prueba_herencia') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-118: sonda requiere DBA y rol libre'; END IF;
 IF pg_catalog.to_regrole('vec_externo_usuarios_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_usuarios_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_usuarios_ejecutor_externo TO vec_externo_usuarios_desarrollo WITH INHERIT TRUE, SET FALSE;
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
   WHERE m.member='vec_externo_usuarios_desarrollo'::regrole
    AND m.roleid='vec_usuarios_ejecutor_externo'::regrole
    AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
      WHERE m.member='vec_externo_usuarios_desarrollo'::regrole)<>1
 THEN RAISE EXCEPTION 'AD3-118: LOGIN externo de sonda incompatible'; END IF;
 CREATE ROLE vec_ad3_118_prueba_herencia NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
END $roles$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_118(p_perfil text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM 1 FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(
  p_perfil,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
  NULL::numeric,NULL::numeric,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea);
END $f$;
RESET ROLE;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_118(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_externo_usuarios_desarrollo;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_118(text) TO vec_externo_usuarios_desarrollo;
SET SESSION AUTHORIZATION vec_externo_usuarios_desarrollo;
DO $pruebas$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.prueba_guardia_ad3_118('portal_candidato_bolsa');
  RAISE EXCEPTION 'AD3-118: perfil Bolsa admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.prueba_guardia_ad3_118('correos_retirar_usuarios');
  RAISE EXCEPTION 'AD3-118: guardia positiva no llegó a validar entrada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
 END;
END $pruebas$;
RESET SESSION AUTHORIZATION;
GRANT vec_ad3_118_prueba_herencia TO vec_usuarios_ejecutor_externo WITH INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_externo_usuarios_desarrollo;
DO $negativo$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.prueba_guardia_ad3_118('correos_retirar_usuarios');
  RAISE EXCEPTION 'AD3-118: herencia transitiva admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $negativo$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
