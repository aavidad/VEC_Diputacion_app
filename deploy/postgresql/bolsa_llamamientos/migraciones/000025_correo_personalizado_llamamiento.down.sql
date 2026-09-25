\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000025:down',0));

-- Revierte 000025 solo sin historia: ni huellas registradas ni reservas con
-- una plantilla distinta de bolsa-llamamiento-v1. Con historia no se deshace.
DO $pre$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR to_regclass('vec_bolsa_llamamientos.cuerpo_llamamiento_contacto') IS NULL THEN RAISE EXCEPTION 'estado incompatible para deshacer 000025' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto) OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE configuracion->>'plantilla_version'<>'bolsa-llamamiento-v1') THEN RAISE EXCEPTION 'hay historia B7 personalizada; no se deshace' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1(text,text,text,bytea,jsonb);
DROP TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto;
DO $ajuste_b7$
DECLARE
 v_def text;
 v_ant text := $sql$p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$'$sql$;
 v_nueva text := $sql$p_configuracion->>'plantilla_version'<>'bolsa-llamamiento-v1'$sql$;
BEGIN
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO STRICT v_def;
 IF strpos(v_def,v_ant)=0 OR strpos(substr(v_def,strpos(v_def,v_ant)+length(v_ant)),v_ant)<>0 THEN
  RAISE EXCEPTION 'reserva B7 sin plantillas versionadas' USING ERRCODE='55000';
 END IF;
 EXECUTE replace(v_def,v_ant,v_nueva);
END $ajuste_b7$;
COMMIT;
