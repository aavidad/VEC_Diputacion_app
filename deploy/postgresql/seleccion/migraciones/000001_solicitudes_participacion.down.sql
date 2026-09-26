\set ON_ERROR_STOP on
-- Selección 000001 DOWN: solo sin historia de solicitudes ni accesos. Las
-- versiones publicadas de las convocatorias son catálogo reproducible desde
-- su fichero y se retiran con el esquema.
BEGIN;
SET LOCAL ROLE vec_seleccion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL client_min_messages = warning;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_seleccion:migracion:000001', 0));
DO $pre$
BEGIN
 IF to_regclass('vec_seleccion.solicitud') IS NULL THEN
  RAISE EXCEPTION 'Selección 000001 DOWN: no instalada' USING ERRCODE = '55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_seleccion.solicitud) OR EXISTS (SELECT 1 FROM vec_seleccion.acceso_rrhh) THEN
  RAISE EXCEPTION 'Selección 000001 DOWN: no admitido con solicitudes o accesos registrados' USING ERRCODE = '55000';
 END IF;
END $pre$;
DROP SCHEMA vec_seleccion CASCADE;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_seleccion_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
COMMIT;
