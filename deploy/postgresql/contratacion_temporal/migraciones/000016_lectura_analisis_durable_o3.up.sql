BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000016_lectura_analisis_durable_o3',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.analisis_rrhh_valido_v3(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_integral_actual'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.confirmacion_operacion_analisis'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(text,text,numeric)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para lectura durable O3';
    END IF;
END
$prevalidacion$;

-- Frontera mínima de lectura del agregado vigente. La referencia y la huella
-- del análisis se obtienen de la confirmación y del canon O3; nunca de un
-- artefacto aportado por el llamador.
CREATE FUNCTION
vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(
    p_organizacion_ref text,
    p_expediente_ref text,
    p_version_esperada numeric
)
RETURNS TABLE (
    expediente_json text,
    analisis_huella_sha256 text
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_statement_ms numeric;
    v_idle_ms numeric;
BEGIN
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_statement_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout';
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_idle_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout';

    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR pg_catalog.current_setting('transaction_isolation') <>
          'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR v_statement_ms IS NULL OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL OR v_idle_ms NOT BETWEEN 1 AND 20000
       OR p_organizacion_ref IS NULL
       OR p_organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref IS NULL
       OR p_expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_version_esperada IS NULL
       OR p_version_esperada NOT BETWEEN
          2::numeric AND 9007199254740990::numeric
       OR p_version_esperada <> pg_catalog.trunc(p_version_esperada) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'lectura durable O3 no autorizada';
    END IF;

    RETURN QUERY
    SELECT v.agregado_json::text,
           h.analisis_huella_sha256
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref
       AND v.version = a.version
      JOIN vec_contratacion_temporal.actuacion_expediente_integral x
        ON x.expediente_ref = v.expediente_ref
       AND x.version_expediente = v.version
       AND x.secuencia = v.version
      JOIN vec_contratacion_temporal.reserva_operacion_analisis r
        ON r.expediente_ref = v.expediente_ref
       AND r.organizacion_ref = p_organizacion_ref
       AND r.version_expediente + 1 = v.version
       AND r.recibo_ref = x.recibo_ref
      JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c
        ON c.ambito_raiz_hmac = r.ambito_raiz_hmac
      JOIN vec_contratacion_temporal.reserva_operacion_analisis_actual ra
        ON ra.ambito_raiz_hmac = r.ambito_raiz_hmac
      JOIN vec_contratacion_temporal.reserva_operacion_analisis_version rv
        ON rv.ambito_raiz_hmac = ra.ambito_raiz_hmac
       AND rv.revision = ra.revision
       AND rv.estado = 'confirmada'
       AND rv.confirmada_en = c.confirmada_en
      CROSS JOIN LATERAL (
          SELECT vec_contratacion_temporal.huella_analisis_derivado_v2(
                     v.agregado_json -> 'analisis'
                 ) AS analisis_huella_sha256
      ) h
     WHERE a.expediente_ref = p_expediente_ref
       AND a.version = p_version_esperada
       AND v.origen_version = 'analisis_o3'
       AND pg_catalog.octet_length(v.agregado_json::text)
           BETWEEN 2 AND 262144
       AND v.agregado_json ->> 'organizacion_ref' = p_organizacion_ref
       AND v.agregado_json ->> 'referencia' = p_expediente_ref
       AND v.agregado_json ->> 'version' = p_version_esperada::text
       AND pg_catalog.jsonb_typeof(v.agregado_json -> 'analisis') =
           'object'
       AND vec_contratacion_temporal.analisis_rrhh_valido_v3(
               v.agregado_json -> 'analisis'
           ) IS TRUE
       AND h.analisis_huella_sha256 ~ '^[a-f0-9]{64}$'
       AND h.analisis_huella_sha256 <> pg_catalog.repeat('0', 64)
       AND v.agregado_json #>> '{analisis,actuacion_registro,recibo_ref}' =
           x.recibo_ref
       AND v.agregado_json #>> '{analisis,actuacion_registro,accion_clave}' =
           x.actuacion_json ->> 'accion_clave'
       AND v.agregado_json
               #>> '{analisis,actuacion_registro,version_expediente}' =
           v.version::text
       AND v.agregado_json
               #>> '{analisis,actuacion_registro,secuencia}' =
           x.secuencia::text
       AND CASE
               WHEN pg_catalog.jsonb_typeof(
                        v.agregado_json -> 'actuaciones'
                    ) = 'array'
                AND pg_catalog.jsonb_array_length(
                        v.agregado_json -> 'actuaciones'
                    ) BETWEEN 2 AND 4096
               THEN x.actuacion_json = (
                   v.agregado_json -> 'actuaciones' -> (
                       pg_catalog.jsonb_array_length(
                           v.agregado_json -> 'actuaciones'
                       ) - 1
                   )
               )
               ELSE false
           END
       AND x.actuacion_json ->> 'recibo_ref' = r.recibo_ref
       AND x.actuacion_json ->> 'accion_clave' IN (
           'contratacion_temporal.analisis.registrar',
           'contratacion_temporal.analisis.rectificar'
       )
       AND (
           (
               r.operacion = 'registrar'
               AND x.actuacion_json ->> 'accion_clave' =
                   'contratacion_temporal.analisis.registrar'
           )
           OR (
               r.operacion = 'rectificar'
               AND x.actuacion_json ->> 'accion_clave' =
                   'contratacion_temporal.analisis.rectificar'
           )
       )
       AND a.operacion_ref = v.operacion_ref
       AND v.operacion_ref = x.operacion_ref
       AND a.actualizada_en = v.registrada_en
       AND v.registrada_en = x.registrada_en
       AND x.registrada_en = c.confirmada_en
       AND c.recibo_json ->> 'operacion' = r.operacion
       AND c.recibo_json ->> 'organizacion_ref' = p_organizacion_ref
       AND c.recibo_json ->> 'expediente_ref' = p_expediente_ref
       AND c.recibo_json ->> 'version_resultante' =
           p_version_esperada::text
       AND c.recibo_json ->> 'secuencia_actuacion' = x.secuencia::text
       AND c.recibo_json ->> 'recibo_ref' = x.recibo_ref
       AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(c.recibo_json::text, 'UTF8')
           ), 'hex') = c.recibo_huella_sha256;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(
    text, text, numeric
)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(
    text, text, numeric
)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
