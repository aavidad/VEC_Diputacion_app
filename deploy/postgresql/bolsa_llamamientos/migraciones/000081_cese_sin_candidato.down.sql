\set ON_ERROR_STOP on
-- Bolsa 000081 DOWN. Solo sin ningún cese sin candidato registrado: con
-- historia no se revierte. Devuelve el cursor exacto de 000045.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000081',0));
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.cese_sin_candidato_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint)') IS NULL
    OR (SELECT md5(prosrc) FROM pg_proc WHERE oid='vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()'::regprocedure)
       IS DISTINCT FROM '393d7534b44ca5db82ef3ed77ecbdafd' THEN
  RAISE EXCEPTION 'Bolsa 000081 DOWN: instalación incompatible' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa) THEN
  RAISE EXCEPTION 'Bolsa 000081 DOWN: historia conservada' USING ERRCODE='55000';
 END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()
 RETURNS TABLE(origen_posicion bigint, origen_ref text)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'cursor de cese no autorizado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.posicion,x.ref FROM (
  SELECT r.origen_posicion AS posicion,r.origen_ref AS ref FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  UNION ALL
  SELECT a.origen_posicion,a.origen_ref FROM vec_bolsa_llamamientos.cese_ajeno_bolsa a
 ) x ORDER BY x.posicion DESC,x.ref DESC LIMIT 1;
END $function$;
DO $post$
BEGIN
 IF (SELECT md5(prosrc) FROM pg_proc WHERE oid='vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()'::regprocedure)
    IS DISTINCT FROM '1654ac60956eaa685f13683466fb51cd' THEN
  RAISE EXCEPTION 'Bolsa 000081 DOWN: cursor no restaurado' USING ERRCODE='55000';
 END IF;
END $post$;
DROP FUNCTION vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint);
DROP TABLE vec_bolsa_llamamientos.cese_sin_candidato_bolsa;
COMMIT;
