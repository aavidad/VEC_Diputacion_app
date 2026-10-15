\set ON_ERROR_STOP on
-- CT221: lectura mínima de modalidad y versión para el orden contextual de Bolsa.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000221',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int < 180000
 OR current_user <> 'vec_contratacion_temporal_propietario'
 OR to_regclass('vec_contratacion_temporal.expediente_alta') IS NULL
 OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
 OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT221: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La ACL limita esta lectura a los dueños que la consumen en una función
-- SECURITY DEFINER. No entrega el agregado ni otra información del expediente.
CREATE FUNCTION vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
 p_expediente text,p_categoria text,p_organizacion text,p_version bigint)
RETURNS TABLE(modalidad_clave text,version_expediente bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE v_organizacion text;v_categoria text;v_modalidad text;v_version bigint;
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario'
 OR p_expediente IS NULL OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
 OR p_categoria<>btrim(p_categoria)
 OR (p_organizacion IS NOT NULL AND
     (p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'))
 OR (p_version IS NOT NULL AND (p_version<1 OR p_version>9007199254740991))
 THEN RAISE EXCEPTION 'CT221: origen no disponible' USING ERRCODE='42501'; END IF;
 SELECT e.organizacion_ref,a.version::bigint,
        v.agregado_json#>>'{analisis,categoria_ref}',
        v.agregado_json#>>'{analisis,modalidad_clave}'
 INTO v_organizacion,v_version,v_categoria,v_modalidad
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a
   ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.expediente_ref=p_expediente;
 IF NOT FOUND OR v_organizacion IS NULL
 OR (p_organizacion IS NOT NULL AND v_organizacion IS DISTINCT FROM p_organizacion)
 OR (p_version IS NOT NULL AND v_version IS DISTINCT FROM p_version)
 OR v_categoria IS DISTINCT FROM p_categoria
 OR v_modalidad IS NULL OR v_modalidad !~ '^[a-z][a-z0-9._-]{1,79}$'
 THEN RAISE EXCEPTION 'CT221: origen no disponible' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v_modalidad,v_version;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)
 TO vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario;
DO $post$ BEGIN
 IF (SELECT p.prosecdef AND p.proowner='vec_contratacion_temporal_propietario'::regrole
            AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
      FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)'::regprocedure) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT221: función no cerrada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
