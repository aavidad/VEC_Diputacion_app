\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000029', 0));
-- Con disposiciones manifestadas por las personas no se deshace: cada una
-- consumió una autorización que ya consta como efecto.
DO $f$ BEGIN
 IF to_regclass('vec_bolsa_llamamientos.disposicion_oferta_candidato') IS NULL THEN
  RAISE EXCEPTION 'migracion 000029 no aplicada' USING ERRCODE='55000';
 END IF;
 -- 000030 y 000040 usan la participación vigente y la marca de consumo.
 IF to_regclass('vec_bolsa_llamamientos.solicitud_portal_candidato') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NOT NULL THEN
  RAISE EXCEPTION '000030 o 000040 instaladas; deshacerlas antes' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta_candidato) THEN
  RAISE EXCEPTION 'hay disposiciones manifestadas; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(text,text,text,text,text,timestamptz,text);
DROP FUNCTION vec_bolsa_llamamientos.participacion_oferta_candidato_v1(text,text);
DROP FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text);
DROP FUNCTION vec_bolsa_llamamientos.anotar_consumo_candidato_v1(text,text,text);
DROP FUNCTION vec_bolsa_llamamientos.firma_marca_consumo_v1(xid8,text,text,text);
DROP FUNCTION vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text);
DROP TABLE vec_bolsa_llamamientos.secreto_marca_consumo;
DROP TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato;
COMMIT;
