\set ON_ERROR_STOP on
-- La reversión del núcleo AD3 requiere un procedimiento supervisado que
-- conserve decisiones, consumos y auditoría; nunca se borra en una instalación.
DO $f$ BEGIN RAISE EXCEPTION 'AD3-59: DOWN no admitido con historia' USING ERRCODE='55000'; END $f$;
