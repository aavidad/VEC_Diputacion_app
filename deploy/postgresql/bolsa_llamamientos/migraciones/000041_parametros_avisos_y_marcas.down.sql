\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000041:down', 0));

-- Revierte exclusivamente 000041. No ejecutar sobre bases con historia: se
-- rechaza si ya se publicó otra versión además de la inicial. 000020 y su
-- bandeja v1 quedan intactas.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.politica_avisos_bolsa') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para revertir los parametros de avisos 000041' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.politica_avisos_bolsa) <> 1 THEN
  RAISE EXCEPTION 'los parametros de avisos 000041 tienen historia' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DROP TRIGGER llamamiento_emitido_presta_servicios ON vec_bolsa_llamamientos.llamamiento_emitido;
DROP FUNCTION vec_bolsa_llamamientos.exigir_no_presta_servicios();
DROP FUNCTION vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.encadenamiento_en_v1(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.presta_servicios_en_v1(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.situaciones_en_v1(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.consultar_politica_avisos_bolsa_v1();
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1(text,text,integer,integer,integer,integer,text,text[]);
DROP FUNCTION vec_bolsa_llamamientos.politica_avisos_bolsa_vigente();
DROP TABLE vec_bolsa_llamamientos.politica_avisos_bolsa;
COMMIT;
