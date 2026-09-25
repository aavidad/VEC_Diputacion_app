\set ON_ERROR_STOP on
-- Retirada solo sin historia: una proyección publicada nunca se borra.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000016:proyeccion-empleado-persona:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL THEN
   RAISE EXCEPTION 'Personal 000016 DOWN: proyección ausente' USING ERRCODE='55000';
 END IF;
END $pre$;
LOCK TABLE vec_personal.proyeccion_empleado_persona_historia,
 vec_personal.proyeccion_empleado_persona_control IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia) THEN
   RAISE EXCEPTION 'Personal 000016 DOWN: la proyección tiene historia' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_control) THEN
   RAISE EXCEPTION 'Personal 000016 DOWN: la generación por persona tiene filas' USING ERRCODE='55000';
 END IF;
END $historia$;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text);
DROP FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz);
DROP FUNCTION vec_personal.publicar_proyeccion_empleado_persona_v1(text,bigint,text,text,text,timestamptz,timestamptz,text,text,bigint,text);
DROP TABLE vec_personal.proyeccion_empleado_persona_control;
DROP TABLE vec_personal.proyeccion_empleado_persona_historia;
DROP FUNCTION vec_personal.validar_version_proyeccion_empleado_v1();
DROP FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1();
COMMIT;
