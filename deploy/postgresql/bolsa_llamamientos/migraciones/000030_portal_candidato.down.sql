\set ON_ERROR_STOP on
-- Bolsa 000030 DOWN. Solo sin historia: con alguna solicitud o respuesta del
-- portal se niega, porque son hechos de la persona que no se borran.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000030', 0));
DO $proteger$
BEGIN
 IF to_regclass('vec_bolsa_llamamientos.solicitud_portal_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL THEN
  RAISE EXCEPTION 'Bolsa 000030 no instalada' USING ERRCODE='55000';
 END IF;
 LOCK TABLE vec_bolsa_llamamientos.solicitud_portal_candidato, vec_bolsa_llamamientos.respuesta_portal_llamamiento IN ACCESS EXCLUSIVE MODE;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento) THEN
  RAISE EXCEPTION 'Bolsa 000030: DOWN denegado con solicitudes o respuestas del portal' USING ERRCODE='55000';
 END IF;
END $proteger$;
DROP FUNCTION vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[]);
DROP FUNCTION vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text);
DROP FUNCTION vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text);
DROP FUNCTION vec_bolsa_llamamientos.exigir_portal_candidato_v1(text,text,text,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(text);
DROP FUNCTION vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(text,timestamptz,text[]);
DROP FUNCTION vec_bolsa_llamamientos.participacion_portal_candidato_v1(text,text);
DROP TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento;
DROP TABLE vec_bolsa_llamamientos.solicitud_portal_candidato;
COMMIT;
