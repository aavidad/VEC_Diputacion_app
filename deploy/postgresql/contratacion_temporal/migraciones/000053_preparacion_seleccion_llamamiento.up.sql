BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

-- Solo lee el agregado propio fiscalizado. No crea una propuesta en Bolsa,
-- no consume permisos y no habilita por sí sola el llamamiento.
CREATE FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v1(
    p_organizacion text, p_expediente text, p_version bigint
) RETURNS TABLE(expediente_json jsonb, version_actual bigint)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_expediente jsonb;
    v_actual bigint;
BEGIN
    IF NOT pg_catalog.pg_has_role(session_user,
            'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR p_organizacion IS NULL OR p_expediente IS NULL
       OR p_version IS DISTINCT FROM 6
       OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparacion de seleccion denegada';
    END IF;
    SELECT v.agregado_json, a.version::bigint INTO v_expediente, v_actual
      FROM vec_contratacion_temporal.expediente_version_integral v
      JOIN vec_contratacion_temporal.expediente_integral_actual a
        USING (expediente_ref)
     WHERE v.expediente_ref = p_expediente AND v.version = p_version
       AND v.agregado_json->>'organizacion_ref' = p_organizacion
       AND v.agregado_json->>'fase_actual' = 'fiscalizacion'
       AND v.agregado_json->>'estado_actual' = 'en_curso'
       AND v.agregado_json->'fiscalizacion'->>'resultado' IN
           ('favorable', 'favorable_con_observaciones');
    IF NOT FOUND OR v_actual < p_version THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparacion de seleccion denegada';
    END IF;
    RETURN QUERY SELECT v_expediente, v_actual;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v1(
    text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v1(
    text,text,bigint) TO vec_contratacion_temporal_ejecutor;
COMMIT;
