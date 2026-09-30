\set ON_ERROR_STOP on
-- Solo reversión de ensayo sin hechos. Nunca ejecutar sobre historia instalada.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000004',0));
LOCK TABLE vec_catalogos_configurables.propuesta_gobierno,
 vec_catalogos_configurables.control_gobierno,
 vec_catalogos_configurables.aprobacion_gobierno,
 vec_catalogos_configurables.confirmacion_gobierno,
 vec_catalogos_configurables.outbox_gobierno IN ACCESS EXCLUSIVE MODE;
DO $guardia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.propuesta_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.control_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.aprobacion_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.confirmacion_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.outbox_gobierno) THEN
  RAISE EXCEPTION 'Cat4: historia conservada; reversion prohibida' USING ERRCODE='55000';
 END IF;
END $guardia$;
DROP FUNCTION vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text);
DROP FUNCTION vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text);
DROP TABLE vec_catalogos_configurables.outbox_gobierno;
DROP TABLE vec_catalogos_configurables.confirmacion_gobierno;
DROP TABLE vec_catalogos_configurables.aprobacion_gobierno;
DROP TABLE vec_catalogos_configurables.control_gobierno;
DROP TABLE vec_catalogos_configurables.propuesta_gobierno;
COMMIT;
