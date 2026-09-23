\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR EXISTS(SELECT 1 FROM pg_class WHERE relnamespace='vec_dietas'::regnamespace) OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_dietas'::regnamespace) THEN RAISE EXCEPTION 'retirada Dietas protege objetos' USING ERRCODE='55000'; END IF;
END $$;
DROP SCHEMA vec_dietas;
-- Deshace únicamente las ACL por defecto creadas por roles_up; no toca ACL
-- de otros esquemas ni intenta borrar una dependencia externa.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_dietas_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_dietas_propietario GRANT USAGE ON TYPES TO PUBLIC;
REVOKE vec_dietas_propietario FROM vec_dietas_migrador;
DROP ROLE vec_dietas_ejecutor;
DROP ROLE vec_dietas_migrador;
DROP ROLE vec_dietas_propietario;
COMMIT;
