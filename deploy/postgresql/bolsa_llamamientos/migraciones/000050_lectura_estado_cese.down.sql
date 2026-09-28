\set ON_ERROR_STOP on
-- Solo se revierte sin hechos B45 ni versiones de política publicadas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000050',0));
LOCK TABLE vec_bolsa_llamamientos.restriccion_cese_bolsa,
 vec_bolsa_llamamientos.cese_ajeno_bolsa,
 vec_bolsa_llamamientos.politica_cese_bolsa IN SHARE ROW EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa)
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.politica_cese_bolsa)<>1 THEN
  RAISE EXCEPTION 'Bolsa 000050 DOWN: instalación o historia incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

DO $lectura$
DECLARE v_oid oid:='vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz)'::regprocedure;
 v_def text; v_acl aclitem[];
 v_antes text:='SELECT r.disponible_desde,r.fecha_efecto,r.recibo_ref,r.politica_version';
 v_despues text:='SELECT e.disponible_desde,e.fecha_efecto,r.recibo_ref,r.politica_version';
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
 WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_despues,''))<>length(v_despues) THEN
  RAISE EXCEPTION 'Bolsa 000050 DOWN: lector de restricción incompatible' USING ERRCODE='55000';
 END IF;
 v_def:=replace(v_def,v_despues,v_antes);
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def
    OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000050 DOWN: lector alterado' USING ERRCODE='55000';
 END IF;
END $lectura$;
DROP FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz);
COMMIT;
