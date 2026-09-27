\set ON_ERROR_STOP on
-- Bolsa 000050: lectura técnica nominal del estado B45 para consumidores
-- autorizados y fecha de cese coherente en la restricción visible. 000045 ya
-- tiene historia: no se modifica ni se reaplica. El lector interno B45 sigue
-- cerrado; el endpoint RRHH debe consumir su autorización V3 antes de llamar.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000050',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa 000050: estado de B45 incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

-- La función B45 devuelve fecha_efecto del último cese y el máximo de las
-- fechas de disponibilidad que aún afectan al candidato. No concede acceso
-- directo al lector interno ni a las tablas.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(
 p_participacion_ref text,p_corte timestamptz)
RETURNS TABLE(fecha_efecto date,disponible_desde date,en_restriccion boolean,trabajo_cesado boolean)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_participacion_ref IS NULL OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 512
    OR p_corte IS NULL OR NOT isfinite(p_corte) THEN
  RAISE EXCEPTION 'consulta de estado de cese denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT e.fecha_efecto,e.disponible_desde,e.en_restriccion,e.trabajo_cesado
  FROM vec_bolsa_llamamientos.estado_cese_bolsa_v1(p_participacion_ref,p_corte) e;
END $f$;

-- B45 elegía la fila cuya disponibilidad terminaba más tarde y mostraba su
-- fecha de cese, que puede ser anterior al cese más reciente. Conservamos
-- recibo y versión de esa fila determinante del máximo; solo la fecha
-- efectiva visible se obtiene del último cese del lector interno.
DO $lectura$
DECLARE v_oid oid:='vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz)'::regprocedure;
 v_def text; v_acl aclitem[];
 v_antes text:='SELECT r.disponible_desde,r.fecha_efecto,r.recibo_ref,r.politica_version';
 v_despues text:='SELECT e.disponible_desde,e.fecha_efecto,r.recibo_ref,r.politica_version';
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
 WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_antes,''))<>length(v_antes)
    OR strpos(v_def,'JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(p_participacion_ref,p_corte) e ON e.en_restriccion')=0 THEN
  RAISE EXCEPTION 'Bolsa 000050: lector de restricción incompatible' USING ERRCODE='55000';
 END IF;
 v_def:=replace(v_def,v_antes,v_despues);
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def
    OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000050: lector de restricción alterado' USING ERRCODE='55000';
 END IF;
END $lectura$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)
 TO vec_bolsa_llamamientos_ejecutor;
DO $acl$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese',
      'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.restriccion_cese_bolsa','SELECT,INSERT,UPDATE,DELETE') THEN
  RAISE EXCEPTION 'Bolsa 000050: ACL incompatible' USING ERRCODE='42501';
 END IF;
END $acl$;
COMMIT;
