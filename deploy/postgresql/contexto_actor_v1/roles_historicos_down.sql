\set ON_ERROR_STOP on
-- Retirar primero fachada; no destruye ni modifica historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:lectura_historica:v2',0));
DO $retirada$
DECLARE r oid:='vec_contexto_actor_v1_lector_historico'::regrole;
 b oid:=(SELECT oid FROM pg_database WHERE datname=current_database());
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'retirada rol historico requiere DBA' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
 AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
 AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r OR roleid=r OR grantor=r)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole=r)
 OR EXISTS(SELECT 1 FROM pg_shdescription WHERE classoid='pg_authid'::regclass AND objoid=r)
 OR EXISTS(SELECT 1 FROM pg_shseclabel WHERE classoid='pg_authid'::regclass AND objoid=r)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_contexto_actor_v1'::regnamespace AND proname='leer_contexto_original_v2')
 OR (SELECT count(*) FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=r)<>1
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=r
 AND (classid<>'pg_database'::regclass OR objid<>b OR objsubid<>0 OR deptype<>'a'))
 THEN RAISE EXCEPTION 'rol historico alterado o con dependencias' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE a.grantee=r)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a
 WHERE d.oid=b AND a.grantee=r AND a.privilege_type='CONNECT' AND NOT a.is_grantable AND a.grantor=d.datdba)
 THEN RAISE EXCEPTION 'ACL de conexion historica alterada' USING ERRCODE='55000'; END IF;
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_contexto_actor_v1_lector_historico RESTRICT',current_database());
END $retirada$;
DROP ROLE vec_contexto_actor_v1_lector_historico;
COMMIT;
