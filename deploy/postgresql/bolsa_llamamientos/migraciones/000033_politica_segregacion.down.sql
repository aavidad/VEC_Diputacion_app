\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000033:down', 0));

-- Revierte exclusivamente 000033. No ejecutar sobre bases con historia: se
-- rechaza si alguna operación ya aplicó la política o si se publicó otra
-- versión además de la inicial. El CHECK de 000019 sigue intacto.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.politica_segregacion') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para revertir la politica de segregacion B8' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE politica_segregacion_version IS NOT NULL)
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.politica_segregacion) <> 1 THEN
  RAISE EXCEPTION 'la politica de segregacion B8 tiene historia' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DROP TRIGGER operacion_situacion_politica_segregacion ON vec_bolsa_llamamientos.operacion_situacion_participacion;
DROP FUNCTION vec_bolsa_llamamientos.aplicar_politica_segregacion();
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_segregacion_v1(text,text,text[]);
DROP FUNCTION vec_bolsa_llamamientos.consultar_politica_segregacion_v1();
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion DROP CONSTRAINT operacion_situacion_politica_segregacion_obligatoria;
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion DROP COLUMN politica_segregacion_version;
DROP TABLE vec_bolsa_llamamientos.politica_segregacion;
COMMIT;
