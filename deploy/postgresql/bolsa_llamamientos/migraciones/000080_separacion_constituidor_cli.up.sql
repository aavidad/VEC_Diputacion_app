\set ON_ERROR_STOP on
-- B80: la orden CLI de constitución usa un grupo técnico propio. El LOGIN
-- web conserva lecturas del ejecutor y sólo confirma B1 mediante B79/V3.
-- El DBA aprovisiona fuera de Git un LOGIN CLI sin pertenencia al ejecutor web.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000080',0));
DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolsuper)
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_constituidor')
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B80: DBA o B79 ausente; rol no creado' USING ERRCODE='55000'; END IF;
 CREATE ROLE vec_bolsa_llamamientos_constituidor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_constituidor',current_database());
END $rol$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $pre$
DECLARE f1 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)');
        f2 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)');
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR f1 IS NULL OR f2 IS NULL
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid IN (f1,f2)
   AND (proowner<>'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole OR NOT prosecdef))
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f1,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f2,'EXECUTE')
 THEN RAISE EXCEPTION 'B80: preimagen B7/B8 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_constituidor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
 text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)
 FROM vec_bolsa_llamamientos_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)
 FROM vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
 text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz),
 vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)
 TO vec_bolsa_llamamientos_constituidor;
DO $post$
BEGIN
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)','EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',
   'vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',
   'vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)','EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid='vec_bolsa_llamamientos_constituidor'::pg_catalog.regrole)
 THEN RAISE EXCEPTION 'B80: ACL o pertenencia divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
