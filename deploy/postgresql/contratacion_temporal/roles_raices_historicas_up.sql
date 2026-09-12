\set ON_ERROR_STOP on
-- Bootstrap exclusivo de rol; no membresías/login ni capacidad de resolución.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000078:lectura_raices:v2',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'bootstrap historico requiere DBA' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_lector_raices_historicas')
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal'
 AND nspowner='vec_contratacion_temporal_propietario'::regrole)
 THEN RAISE EXCEPTION 'estado incompatible para rol historico' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_database b CROSS JOIN LATERAL aclexplode(coalesce(b.datacl,acldefault('d',b.datdba))) a
 WHERE b.datname=current_database() AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
 WHERE n.nspname='public' AND a.grantee=0)
 THEN RAISE EXCEPTION 'base de ensayo debe estar cerrada a PUBLIC' USING ERRCODE='42501'; END IF;
END $pre$;
CREATE ROLE vec_contratacion_temporal_lector_raices_historicas NOLOGIN NOINHERIT NOSUPERUSER
 NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$
BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_lector_raices_historicas',current_database());
END $conexion$;
COMMIT;
