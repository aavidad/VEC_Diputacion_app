\set ON_ERROR_STOP on
-- CT233 / F1-09: cotejo privado; no concede lectura ni devuelve el agregado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000233',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT233: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(p jsonb,p_nueva boolean)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE a jsonb;version_actual bigint;organizacion text;necesidad text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR p_nueva IS NULL
 THEN RETURN false; END IF;
 SELECT v.agregado_json,i.version,e.organizacion_ref INTO a,version_actual,organizacion
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual i USING(expediente_ref)
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=i.expediente_ref AND v.version=i.version
 WHERE e.expediente_ref=p->>'expediente_ref' FOR SHARE OF e,i;
 IF NOT FOUND OR organizacion IS DISTINCT FROM p->>'organizacion_ref'
 OR a#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p->>'unidad_ref'
 OR p->>'unidad_ref' IS NULL THEN RETURN false; END IF;
 -- Una recuperación revalida el ámbito actual sin reinterpretar el acto antiguo.
 IF NOT p_nueva THEN RETURN true; END IF;
 -- Misma derivación canónica que prepararReferenciasLlamamientoDesarrollo.
 necesidad:='necesidad:'||translate(encode(sha256(convert_to(
  replace(jsonb_build_array('vec.ct.bolsa.desarrollo.v1','necesidad',organizacion,
   p->>'expediente_ref','6')::text,', ',','),'UTF8')),'hex'),'0123456789','ghijklmnop');
 RETURN version_actual IS NOT DISTINCT FROM (p->>'version_expediente')::bigint
  AND a->>'fase_actual'='fiscalizacion' AND a->>'estado_actual'='en_curso'
  AND a#>>'{fiscalizacion,resultado}' IN ('favorable','favorable_con_observaciones')
  AND a#>>'{analisis,categoria_ref}' IS NOT DISTINCT FROM p->>'categoria_ref'
  AND a#>>'{analisis,modalidad_clave}' IS NOT DISTINCT FROM p->>'modalidad_ofrecida'
  AND coalesce(a#>>'{analisis,causa_clave}','') IS NOT DISTINCT FROM p->>'causa_ofrecida'
  AND necesidad IS NOT DISTINCT FROM p->>'necesidad_ref';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)
 TO vec_bolsa_llamamientos_propietario;
COMMIT;
