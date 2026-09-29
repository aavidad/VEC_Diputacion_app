\set ON_ERROR_STOP on
-- Solo clon desechable. Se revierte hasta la creación del LOGIN de prueba.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
SET LOCAL idle_in_transaction_session_timeout='10s';
DO $roles$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_ad3_116_prueba_herencia') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-116: sonda requiere DBA y rol de prueba libre'; END IF;
 IF pg_catalog.to_regrole('vec_externo_bolsa_desarrollo') IS NULL THEN
   CREATE ROLE vec_externo_bolsa_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
   GRANT vec_bolsa_llamamientos_portal_externo TO vec_externo_bolsa_desarrollo WITH INHERIT TRUE, SET FALSE;
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_externo_bolsa_desarrollo'::regrole
      AND m.roleid='vec_bolsa_llamamientos_portal_externo'::regrole
      AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
         WHERE m.member='vec_externo_bolsa_desarrollo'::regrole)<>1
 THEN RAISE EXCEPTION 'AD3-116: LOGIN externo de sonda incompatible'; END IF;
 CREATE ROLE vec_ad3_116_prueba_herencia NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
END $roles$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_116()
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM 1 FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
  'portal_candidato_bolsa',NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
  NULL::numeric,NULL::numeric,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea);
END $f$;
RESET ROLE;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_116() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_externo_bolsa_desarrollo;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_guardia_ad3_116() TO vec_externo_bolsa_desarrollo;

SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $positivo$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.prueba_guardia_ad3_116();
  RAISE EXCEPTION 'AD3-116: guardia positiva no llegó a validar entrada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
 END;
END $positivo$;
RESET SESSION AUTHORIZATION;

GRANT vec_ad3_116_prueba_herencia TO vec_bolsa_llamamientos_portal_externo WITH INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $negativo$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.prueba_guardia_ad3_116();
  RAISE EXCEPTION 'AD3-116: herencia transitiva admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $negativo$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
