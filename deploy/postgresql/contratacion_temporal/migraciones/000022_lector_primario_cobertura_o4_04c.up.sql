BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 2
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 2
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404c_carga_terminal_v1(text)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para lectores O4-04C';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(
    p_ambitos jsonb
)
RETURNS TABLE (
    ambito_persistido_hmac text,
    huella_semantica_persistida_hmac text,
    carga_json text
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
    v_ambitos text[];
    v_raices text[];
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
       OR p_ambitos IS NULL
       OR pg_catalog.jsonb_typeof(p_ambitos) <> 'array'
       OR pg_catalog.jsonb_array_length(p_ambitos) NOT BETWEEN 1 AND 4 THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consulta terminal O4-04C no autorizada';
    END IF;
    SELECT pg_catalog.array_agg(e.v #>> '{}' ORDER BY e.i)
      INTO STRICT v_ambitos
      FROM pg_catalog.jsonb_array_elements(p_ambitos)
           WITH ORDINALITY e(v, i)
     WHERE pg_catalog.jsonb_typeof(e.v) = 'string'
       AND e.v #>> '{}' ~ (
           '^hmac-sha256:vec[.]contratacion-temporal[.]'
           || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
           || '[a-f0-9]{64}$'
       );
    IF pg_catalog.cardinality(v_ambitos) <>
           pg_catalog.jsonb_array_length(p_ambitos)
       OR pg_catalog.cardinality(v_ambitos) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(v_ambitos) x)
          ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'ámbitos de consulta O4-04C inválidos';
    END IF;
    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema = 3;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera de lectura O4-04C no disponible';
    END IF;
    SELECT pg_catalog.array_agg(DISTINCT a.ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_operacion_decision_cobertura a
     WHERE a.alias_ambito_hmac = ANY(v_ambitos);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'consulta terminal O4-04C divergente';
    END IF;
    RETURN QUERY
    SELECT d.ambito_idempotencia_hmac,
           d.huella_semantica_hmac,
           carga.carga::text
      FROM pg_catalog.unnest(v_ambitos)
           WITH ORDINALITY u(ambito, posicion)
      JOIN vec_contratacion_temporal.alias_operacion_decision_cobertura a
        ON a.alias_ambito_hmac = u.ambito
      JOIN
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura d
        ON d.ambito_raiz_hmac = a.ambito_raiz_hmac
      CROSS JOIN LATERAL (
          SELECT
          vec_contratacion_temporal.o404c_carga_terminal_v1(
              a.ambito_raiz_hmac
          ) AS carga
      ) carga
     WHERE a.ambito_raiz_hmac = v_raices[1]
       AND carga.carga IS NOT NULL
     ORDER BY u.posicion
     LIMIT 1;
END
$funcion$;

-- Frontera SQL interna para el futuro lector push de A/E. Permanece sin
-- GRANT en O4-04C: todavía falta el contrato opaco Go y la verificación de
-- todas las filas funcionales que compondrá O4-04E.
CREATE FUNCTION
vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(
    p_coordenadas jsonb
)
RETURNS TABLE (carga_json text)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_claves text[];
    v_raiz text;
    v_carga jsonb;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.current_setting('transaction_isolation') <>
          'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR p_coordenadas IS NULL
       OR pg_catalog.jsonb_typeof(p_coordenadas) <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'lectura primaria O4-04C no autorizada';
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_coordenadas) c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
        'correlacion_vec_ref',
        'decision_vec_ref',
        'expediente_ref',
        'huella_orden_sha256',
        'organizacion_ref',
        'recibo_ref',
        'reserva_ref',
        'revision_cercado',
        'version_expediente'
    ]::text[]
       OR coalesce(p_coordenadas ->> 'huella_orden_sha256', '') !~
          '^[a-f0-9]{64}$'
       OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
              p_coordenadas -> 'version_expediente',
              2,
              9007199254740990::numeric
          ) IS NOT TRUE
       OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
              p_coordenadas -> 'revision_cercado',
              1,
              9007199254740991::numeric
          ) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'coordenadas primarias O4-04C inválidas';
    END IF;
    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema = 3;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera primaria O4-04C no disponible';
    END IF;
    SELECT b.ambito_raiz_hmac
      INTO v_raiz
      FROM
        vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      JOIN
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura c
        USING (ambito_raiz_hmac)
     WHERE b.organizacion_ref =
               p_coordenadas ->> 'organizacion_ref'
       AND b.expediente_ref = p_coordenadas ->> 'expediente_ref'
       AND b.version_expediente =
               (p_coordenadas ->> 'version_expediente')::numeric
       AND b.reserva_ref = p_coordenadas ->> 'reserva_ref'
       AND b.recibo_ref = p_coordenadas ->> 'recibo_ref'
       AND b.correlacion_vec_ref =
               p_coordenadas ->> 'correlacion_vec_ref'
       AND b.decision_vec_ref = p_coordenadas ->> 'decision_vec_ref'
       AND c.revision_cercado =
               (p_coordenadas ->> 'revision_cercado')::numeric
       AND c.huella_orden_sha256 =
               p_coordenadas ->> 'huella_orden_sha256';
    IF v_raiz IS NULL THEN
        RETURN;
    END IF;
    v_carga :=
        vec_contratacion_temporal.o404c_carga_terminal_v1(v_raiz);
    IF v_carga IS NOT NULL THEN
        RETURN QUERY SELECT v_carga::text;
    END IF;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 3,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 2;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(
        jsonb
    ),
    vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(
        jsonb
    )
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;

GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(
    jsonb
)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
