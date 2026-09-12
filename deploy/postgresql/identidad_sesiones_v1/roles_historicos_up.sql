\set ON_ERROR_STOP on
-- Bootstrap exclusivo de rol; no membresías/login ni capacidad de resolución.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:lectura_historica:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'bootstrap historico requiere DBA' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_lector_historico')
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_identidad_sesiones_v1'
 AND nspowner='vec_identidad_sesiones_v1_propietario'::regrole)
 THEN RAISE EXCEPTION 'estado incompatible para rol historico' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_database b CROSS JOIN LATERAL aclexplode(coalesce(b.datacl,acldefault('d',b.datdba))) a
 WHERE b.datname=current_database() AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
 WHERE n.nspname='public' AND a.grantee=0)
 THEN RAISE EXCEPTION 'base de ensayo debe estar cerrada a PUBLIC' USING ERRCODE='42501'; END IF;
END $pre$;
CREATE ROLE vec_identidad_sesiones_v1_lector_historico NOLOGIN NOINHERIT NOSUPERUSER
 NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$
BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_lector_historico',current_database());
END $conexion$;
COMMIT;
