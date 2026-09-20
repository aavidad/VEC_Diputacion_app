-- Bootstrap nominal B11, de una sola ejecucion. El LOGIN se provisiona aparte
-- con una unica membresia INHERIT TRUE, SET FALSE, ADMIN FALSE; sin ACL directas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:perfil-personal-b11:v1',0));
DO $guard$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer < 180000
 OR to_regrole('vec_contexto_actor_perfil_personal_b11_runtime') IS NOT NULL
 OR to_regrole('vec_contexto_actor_v1_propietario') IS NULL
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname=current_database() AND a.grantee=0)
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='alta rol B11 rechazada'; END IF;
END $guard$;
CREATE ROLE vec_contexto_actor_perfil_personal_b11_runtime NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $grant$
BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_perfil_personal_b11_runtime',current_database());
END $grant$;
COMMIT;
