\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000016', 0));
DO $f$ BEGIN IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.datos_contacto_participacion) OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento WHERE accion='registrar_datos_contacto' OR ruta_clase='datos_contacto') THEN RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='hay datos de contacto B4; no se deshace'; END IF; END $f$;
DROP FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(text,text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.datos_contacto_participacion RESTRICT;
DO $bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 accion := replace(accion, ', ''registrar_datos_contacto''::text', '');
 ruta := replace(ruta, ', ''datos_contacto''::text', '');
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check '||accion;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check '||ruta;
END $bitacora$;
COMMIT;
