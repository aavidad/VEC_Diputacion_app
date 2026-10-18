-- B132: solo sobre una copia desechable ya migrada. Todo efecto se revierte.
\set ON_ERROR_STOP on
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';

DO $prueba$
DECLARE v_oferta text;v_participacion text;v_previo bigint;v_nuevo bigint;
 v_proyeccion jsonb;v_permisos boolean;
BEGIN
 IF to_regclass('vec_bolsa_llamamientos.reintento_aviso_oferta_v1') IS NULL
   OR to_regprocedure('vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(text,text,text,text,bigint,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)') IS NULL
 THEN RAISE EXCEPTION 'B132 ausente'; END IF;
 SELECT o.oferta_ref,d.participacion_ref,h.revision INTO v_oferta,v_participacion,v_previo
 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario d
 JOIN vec_bolsa_llamamientos.aviso_oferta_cabecera h USING(oferta_ref)
 JOIN vec_bolsa_llamamientos.oferta_publicada o USING(oferta_ref)
 ORDER BY o.publicada_en DESC,d.participacion_ref LIMIT 1;
 IF v_oferta IS NULL THEN RAISE EXCEPTION 'copia sin aviso de oferta para la prueba B132'; END IF;
 SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(v_oferta,clock_timestamp()) INTO v_proyeccion;
 IF (v_proyeccion#>>'{aviso,revision}')::bigint IS DISTINCT FROM v_previo
 THEN RAISE EXCEPTION 'la proyección no conserva la revisión consultada'; END IF;
 SELECT has_table_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.reintento_aviso_oferta_v1','SELECT') INTO v_permisos;
 IF v_permisos IS DISTINCT FROM false OR NOT EXISTS(
   SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.reintento_aviso_oferta_v1'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity
   AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole)
 THEN RAISE EXCEPTION 'tabla de recibos expuesta'; END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario
   SET actualizado_en=actualizado_en+interval '1 microsecond'
   WHERE oferta_ref=v_oferta AND participacion_ref=v_participacion;
 SELECT revision INTO v_nuevo FROM vec_bolsa_llamamientos.aviso_oferta_cabecera WHERE oferta_ref=v_oferta;
 IF v_nuevo IS DISTINCT FROM v_previo+1 THEN RAISE EXCEPTION 'la revisión no avanzó una vez'; END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario
   SET estado=estado WHERE oferta_ref=v_oferta AND participacion_ref=v_participacion;
 SELECT revision INTO v_nuevo FROM vec_bolsa_llamamientos.aviso_oferta_cabecera WHERE oferta_ref=v_oferta;
 IF v_nuevo IS DISTINCT FROM v_previo+1 THEN RAISE EXCEPTION 'cambio idéntico alteró revisión'; END IF;
END $prueba$;
ROLLBACK;
