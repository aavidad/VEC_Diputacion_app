\set ON_ERROR_STOP on
-- Solo para instalación vacía desechable. Una oferta B57, preparación,
-- adjudicación, evento o política nueva hace irreversible esta evolución.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000057',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.capacidad_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.adjudicacion_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.evento_adjudicacion_oferta)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version
              WHERE politica#>'{adjudicacion,requiere_segunda_validacion}' IS NOT NULL)
 THEN RAISE EXCEPTION 'B57 DOWN: historia o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.resolver_oferta_v2(text,text,text,text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text);
DROP FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v2(text,timestamptz,integer);
DROP FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text);
DROP FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz);
DROP TABLE vec_bolsa_llamamientos.evento_adjudicacion_oferta;
DROP TABLE vec_bolsa_llamamientos.adjudicacion_oferta;
DROP TABLE vec_bolsa_llamamientos.preparacion_adjudicacion_oferta;
DROP TABLE vec_bolsa_llamamientos.capacidad_oferta;

DO $politica$
DECLARE f regprocedure:='vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 actual text; antiguo text;
 marca text:=$m$OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion')) NOT IN (2,3)
    OR (p_politica->'adjudicacion' ? 'requiere_segunda_validacion' AND
        p_politica#>>'{adjudicacion,requiere_segunda_validacion}' IS DISTINCT FROM 'true')
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))=3
       AND NOT (p_politica->'adjudicacion' ? 'requiere_segunda_validacion')$m$;
 reemplazo text:=$r$OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))<>2$r$;
BEGIN
 actual:=pg_get_functiondef(f);
 IF (length(actual)-length(replace(actual,marca,'')))<>length(marca)
 THEN RAISE EXCEPTION 'B57 DOWN: publicador cambiado' USING ERRCODE='55000'; END IF;
 antiguo:=replace(actual,marca,reemplazo);
 EXECUTE antiguo;
 IF pg_get_functiondef(f) IS DISTINCT FROM antiguo THEN
  RAISE EXCEPTION 'B57 DOWN: publicador no restaurado' USING ERRCODE='55000'; END IF;
END $politica$;

GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
