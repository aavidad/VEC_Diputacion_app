\set ON_ERROR_STOP on
-- Solo retira el esquema vacío; nunca borra objetos ni historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relnamespace='vec_calendarios'::regnamespace)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_calendarios'::regnamespace)
 THEN RAISE EXCEPTION 'retirada Calendarios protege objetos' USING ERRCODE='55000'; END IF;
END $$;
DROP SCHEMA vec_calendarios;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_calendarios_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_calendarios_propietario GRANT USAGE ON TYPES TO PUBLIC;
REVOKE vec_calendarios_propietario FROM vec_calendarios_migrador;
DO $conectar$ BEGIN EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_calendarios_migrador, vec_calendarios_lector', current_database()); END $conectar$;
DROP ROLE vec_calendarios_lector;
DROP ROLE vec_calendarios_migrador;
DROP ROLE vec_calendarios_propietario;
COMMIT;
