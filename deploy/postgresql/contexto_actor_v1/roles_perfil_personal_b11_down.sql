-- Tras retirar 000007 y los LOGIN nominales. Nunca DROP OWNED ni CASCADE.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:perfil-personal-b11:v1',0));
DO $guard$
DECLARE grupo oid:=to_regrole('vec_contexto_actor_perfil_personal_b11_runtime'); base oid;
BEGIN
 SELECT oid INTO base FROM pg_database WHERE datname=current_database();
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR grupo IS NULL
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid=grupo OR member=grupo)
 OR to_regclass('vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole=grupo)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=grupo
 AND NOT(classid='pg_catalog.pg_database'::regclass AND objid=base AND objsubid=0 AND deptype='a'))
 OR (SELECT count(*) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=base AND a.grantee=grupo AND a.privilege_type='CONNECT' AND NOT a.is_grantable)<>1
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='retirada rol B11 rechazada'; END IF;
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_contexto_actor_perfil_personal_b11_runtime',current_database());
END $guard$;
DROP ROLE vec_contexto_actor_perfil_personal_b11_runtime;
COMMIT;
