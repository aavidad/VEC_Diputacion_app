\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000009:asignacion-dietas:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL
    OR EXISTS (SELECT 1 FROM vec_personal.asignacion_dietas) THEN
   RAISE EXCEPTION 'Personal 000009 DOWN: asignación ausente o con historia' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
DROP TABLE vec_personal.asignacion_dietas;
ALTER TABLE vec_personal.relacion_empleado_dietas DROP CONSTRAINT relacion_dietas_identidad_unidad;
COMMIT;
