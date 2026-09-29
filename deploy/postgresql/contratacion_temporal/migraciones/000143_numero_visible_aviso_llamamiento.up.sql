\set ON_ERROR_STOP on
-- CT-000143: número visible vigente para el correo de llamamiento.
--
-- El correo usaba el número del snapshot fiscalizado, que en los expedientes
-- anteriores a la numeración correlativa es el identificador técnico
-- «AAAA/CT-<hex>». CT-000142 asignó a esos expedientes un número anual sin
-- reescribir su historia. Esta función devuelve el número que se muestra hoy
-- (el anual si existe; si no, el original) para el mismo expediente y
-- llamamiento que ya acredita leer_expediente_aviso_confirmado_v1, con sus
-- mismas guardas de rol. No escribe nada ni cambia recibos o avisos emitidos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000143', 0)
);

DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000143 exige el rol propietario';
    END IF;
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.numero_visible_aviso_confirmado_v1(text,text,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000143 ya instalada';
    END IF;
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(text,text,text)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para CT-000143: faltan CT-000095 o CT-000142';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.numero_visible_aviso_confirmado_v1(
    p_organizacion text, p_expediente text, p_llamamiento text
) RETURNS text
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_expediente jsonb;
    v_numero text;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER') THEN
        RAISE EXCEPTION 'número de aviso no disponible' USING ERRCODE = '42501';
    END IF;
    -- El lector de aviso valida parámetros, vínculo con el llamamiento y canon
    -- de la selección confirmada; falla cerrado si algo no cuadra.
    SELECT leido.expediente_json
      INTO STRICT v_expediente
      FROM vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(
          p_organizacion, p_expediente, p_llamamiento) AS leido;
    IF v_expediente->>'referencia' IS DISTINCT FROM p_expediente
       OR v_expediente->>'organizacion_ref' IS DISTINCT FROM p_organizacion
       OR pg_catalog.jsonb_typeof(v_expediente->'numero_visible') IS DISTINCT FROM 'string' THEN
        RAISE EXCEPTION 'número de aviso no disponible' USING ERRCODE = '42501';
    END IF;
    v_numero := vec_contratacion_temporal.numero_visible_vigente_v1(
        p_expediente, v_expediente->>'numero_visible');
    IF v_numero IS NULL OR v_numero !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$' THEN
        RAISE EXCEPTION 'número de aviso no disponible' USING ERRCODE = '42501';
    END IF;
    RETURN v_numero;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN
    RAISE EXCEPTION 'número de aviso no disponible' USING ERRCODE = '42501';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.numero_visible_aviso_confirmado_v1(
    text, text, text) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.numero_visible_aviso_confirmado_v1(
    text, text, text) TO vec_contratacion_temporal_ejecutor;
COMMIT;
