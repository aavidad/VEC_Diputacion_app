\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000034:down', 0));
-- La traza de versiones y situaciones se puede reconstruir desde las historias
-- B2 y B4; la de subcampos de contacto no. Si ya existe alguna, no se deshace.
DO $f$ BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para retirar la traza de valores' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE campo IN ('correo','telefono_1','telefono_2')) THEN
  RAISE EXCEPTION 'hay traza de campos de contacto; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_historial_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP TRIGGER datos_contacto_participacion_traza_valores ON vec_bolsa_llamamientos.datos_contacto_participacion;
DROP TRIGGER situacion_participacion_traza_valores ON vec_bolsa_llamamientos.situacion_participacion;
DROP FUNCTION vec_bolsa_llamamientos.trazar_datos_contacto_participacion_v1() RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.trazar_situacion_participacion_v1() RESTRICT;
DROP TABLE vec_bolsa_llamamientos.traza_valor_participacion RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.fecha_traza_v1(timestamptz) RESTRICT;
COMMIT;
