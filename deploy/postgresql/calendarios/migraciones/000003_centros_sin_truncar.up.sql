\set ON_ERROR_STOP on
-- Calendarios 000003: la lista de centros con calendario ya no se trunca en
-- silencio a 1000 filas; si hay más de las que admite una respuesta, falla con
-- 54000. Sustituye solo el cuerpo de centros_con_calendario_v1 (misma firma,
-- mismo propietario y misma ACL, que CREATE OR REPLACE conserva).
-- Criterio de 000002, sin reescribirla: se clasifica como nacional lo que el
-- anexo de BOE-A-2025-21667 fija para todas las comunidades en 2026 (incluido
-- el 6 de enero, sustituible pero no sustituido este año).
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000003:centros:v1',0));
DO $pre$ BEGIN
 IF current_user<>'vec_calendarios_propietario'
    OR to_regprocedure('vec_calendarios.centros_con_calendario_v1(integer,timestamptz)') IS NULL
    OR position('LIMIT 1000' in pg_get_functiondef('vec_calendarios.centros_con_calendario_v1(integer,timestamptz)'::regprocedure))=0
 THEN RAISE EXCEPTION 'Calendarios 000003: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_calendarios.centros_con_calendario_v1(p_anio integer, p_conocido_en timestamptz)
RETURNS TABLE (id text, ambito_tipo text, ambito_ref text, anio integer, numero integer, sustituye_id text, denominacion text,
               procedencia_norma text, procedencia_referencia text, procedencia_publicada_en date, sintetica boolean,
               comunidad_ref text, municipio_ref text, conocido_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
#variable_conflict use_column
BEGIN
  IF p_anio IS NULL OR p_anio NOT BETWEEN 2000 AND 2100 OR p_conocido_en IS NULL OR p_conocido_en>statement_timestamp() THEN
    RAISE EXCEPTION 'Calendarios: consulta inválida' USING ERRCODE='22023';
  END IF;
  -- Nunca se trunca en silencio: más centros de los que admite una respuesta es un error explícito.
  IF (SELECT count(DISTINCT x.ambito_ref) FROM vec_calendarios.version_calendario x
       WHERE x.ambito_tipo='centro' AND x.anio=p_anio AND x.conocido_desde<=p_conocido_en) > 1000 THEN
    RAISE EXCEPTION 'Calendarios: demasiados centros para una respuesta' USING ERRCODE='54000';
  END IF;
  RETURN QUERY
  SELECT v.id, v.ambito_tipo, v.ambito_ref, v.anio, v.numero, v.sustituye_id, v.denominacion,
         v.procedencia_norma, v.procedencia_referencia, v.procedencia_publicada_en, v.sintetica,
         v.comunidad_ref, v.municipio_ref, v.conocido_desde
    FROM (SELECT DISTINCT ON (x.ambito_ref) x.*
            FROM vec_calendarios.version_calendario x
           WHERE x.ambito_tipo='centro' AND x.anio=p_anio AND x.conocido_desde<=p_conocido_en
           ORDER BY x.ambito_ref, x.numero DESC) v
   ORDER BY v.ambito_ref;
END $f$;

DO $post$ BEGIN
 IF position('54000' in pg_get_functiondef('vec_calendarios.centros_con_calendario_v1(integer,timestamptz)'::regprocedure))=0
    OR has_function_privilege('public','vec_calendarios.centros_con_calendario_v1(integer,timestamptz)','EXECUTE')
    OR NOT has_function_privilege('vec_calendarios_lector','vec_calendarios.centros_con_calendario_v1(integer,timestamptz)','EXECUTE')
 THEN RAISE EXCEPTION 'Calendarios 000003: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
