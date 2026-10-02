\set ON_ERROR_STOP on
-- Revertir primero Méritos 000001 y el consumidor AD nominal, sin historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:bootstrap:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR EXISTS (SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_meritos'::regnamespace)
 OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_meritos'::regnamespace)
 OR EXISTS (SELECT 1 FROM pg_auth_members a WHERE a.roleid IN ('vec_meritos_ejecutor'::regrole,'vec_meritos_externo'::regrole,'vec_meritos_interno'::regrole))
 THEN RAISE EXCEPTION 'meritos.error.roles_down_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP SCHEMA vec_meritos;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario GRANT USAGE ON TYPES TO PUBLIC;
DO $db$ BEGIN EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_meritos_ejecutor,vec_meritos_migrador',current_database()); END $db$;
REVOKE vec_meritos_propietario FROM vec_meritos_migrador;
-- Sin CASCADE ni DROP OWNED: una dependencia inesperada aborta todo el DOWN.
DROP ROLE vec_meritos_externo,vec_meritos_interno,vec_meritos_ejecutor,vec_meritos_migrador,vec_meritos_propietario;
COMMIT;
