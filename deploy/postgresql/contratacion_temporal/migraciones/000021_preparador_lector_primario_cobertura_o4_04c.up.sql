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
 WHERE control
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 1
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.terminal_operacion_decision_cobertura'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404c_carga_terminal_v1(text)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para funciones O4-04C';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404c_carga_terminal_v1(p_raiz text)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_carga jsonb;
BEGIN
    SELECT pg_catalog.jsonb_build_object(
        'reserva_terminal',
        pg_catalog.jsonb_build_object(
            'organizacion_ref', b.organizacion_ref,
            'expediente_ref', b.expediente_ref,
            'version_expediente', b.version_expediente,
            'reserva_ref', b.reserva_ref,
            'recibo_ref', b.recibo_ref,
            'actuacion_ref', b.actuacion_ref,
            'auditoria_ref', b.auditoria_ref,
            'evento_ref', b.evento_ref,
            'correlacion_vec_ref', b.correlacion_vec_ref,
            'decision_vec_ref', b.decision_vec_ref,
            'ambito_idempotencia_hmac',
                c.ambito_idempotencia_hmac,
            'huella_semantica_hmac', c.huella_semantica_hmac,
            'revision_cercado', c.revision_cercado,
            'observada_en',
                vec_contratacion_temporal.texto_instante_utc_go_v2(
                    v.observada_en::text
                )
        ),
        'recibo',
        pg_catalog.jsonb_build_object(
            'recibo_ref', c.recibo_ref,
            'reserva_ref', c.reserva_ref,
            'auditoria_ref', c.auditoria_ref,
            'correlacion_vec_ref', c.correlacion_vec_ref,
            'decision_vec_ref', c.decision_vec_ref,
            'decision_vec_huella_sha256',
                c.decision_vec_huella_sha256,
            'codigo_probatorio_vec', c.codigo_probatorio_vec,
            'concedida_vec', c.rama = 'concedida',
            'revision_cercado', c.revision_cercado,
            'ambito_idempotencia_hmac',
                c.ambito_idempotencia_hmac,
            'huella_semantica_hmac', c.huella_semantica_hmac,
            'confirmada_en',
                vec_contratacion_temporal.texto_instante_utc_go_v2(
                    c.confirmada_en::text
                ),
            'aplicada', c.rama = 'concedida',
            'denegada_vec', c.rama = 'denegada',
            'decision_cobertura_ref',
                coalesce(c.decision_cobertura_ref, ''),
            'decision_cobertura_huella_sha256',
                coalesce(
                    c.decision_cobertura_huella_sha256, ''
                ),
            'version_resultante',
                coalesce(c.version_resultante, 0),
            'evento_ref', coalesce(c.evento_ref, ''),
            'actuacion_ref', coalesce(c.actuacion_ref, '')
        )
    )
      INTO v_carga
      FROM
        vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      JOIN
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual a
        USING (ambito_raiz_hmac)
      JOIN
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
        USING (ambito_raiz_hmac, secuencia)
      JOIN
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura c
        USING (ambito_raiz_hmac)
      JOIN
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        USING (ambito_raiz_hmac)
     WHERE b.ambito_raiz_hmac = p_raiz
       AND v.estado = CASE c.rama
           WHEN 'concedida' THEN 'aplicada'
           WHEN 'denegada' THEN 'denegada_vec'
           ELSE ''
       END
       AND t.secuencia_terminal = a.secuencia
       AND t.recibo_ref = c.recibo_ref
       AND t.huella_orden_sha256 = c.huella_orden_sha256
       AND t.rama = c.rama
       AND t.decision_vec_ref = c.decision_vec_ref
       AND t.auditoria_ref = c.auditoria_ref
       AND t.outbox_ref = b.evento_ref
       AND c.recibo_ref = b.recibo_ref
       AND c.reserva_ref = b.reserva_ref
       AND c.auditoria_ref = b.auditoria_ref
       AND c.correlacion_vec_ref = b.correlacion_vec_ref
       AND c.decision_vec_ref = b.decision_vec_ref
       AND c.revision_cercado = v.revision_cercado
       AND c.huella_orden_sha256 = v.huella_orden_sha256
       AND c.confirmada_en = v.confirmada_en
       AND t.marcada_en = c.confirmada_en
       AND (
           (
               c.rama = 'concedida'
               AND t.decision_cobertura_ref =
                   c.decision_cobertura_ref
               AND t.actuacion_ref = c.actuacion_ref
               AND t.version_resultante = c.version_resultante
               AND c.evento_ref = b.evento_ref
               AND c.actuacion_ref = b.actuacion_ref
           )
           OR (
               c.rama = 'denegada'
               AND t.decision_cobertura_ref IS NULL
               AND t.actuacion_ref IS NULL
               AND t.version_resultante IS NULL
               AND c.evento_ref IS NULL
               AND c.actuacion_ref IS NULL
           )
       );
    RETURN v_carga;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(
    p_operacion jsonb,
    p_par_persistido_validado jsonb DEFAULT NULL
)
RETURNS TABLE (
    resultado text,
    ambito_persistido_hmac text,
    huella_semantica_persistida_hmac text,
    carga_json text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_claves text[];
    v_ambitos text[];
    v_raices text[];
    v_raiz text;
    v_ambito_persistido text;
    v_semantica_persistida text;
    v_generacion_activa integer;
    v_filas bigint;
    v_nueva boolean := false;
    v_base
        vec_contratacion_temporal.reserva_operacion_decision_cobertura%ROWTYPE;
    v_actual
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual%ROWTYPE;
    v_version
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version%ROWTYPE;
    v_agregado jsonb;
    v_analisis_ref text;
    v_analisis_huella text;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_carga jsonb;
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
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR v_statement_ms IS NULL OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL OR v_idle_ms NOT BETWEEN 1 AND 20000
       OR p_operacion IS NULL
       OR pg_catalog.jsonb_typeof(p_operacion) <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'preparación de cobertura no autorizada';
    END IF;

    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_operacion) c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
        'ambito_activo_hmac',
        'ambitos_consulta_hmac',
        'esquema',
        'expediente_ref',
        'huella_semantica_activa_hmac',
        'organizacion_ref',
        'token_propietario_sha256',
        'version_expediente'
    ]::text[]
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.preparar-decision-cobertura.v1'
       OR coalesce(p_operacion ->> 'organizacion_ref', '') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'expediente_ref', '') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
              p_operacion -> 'version_expediente',
              2,
              9007199254740990::numeric
          ) IS NOT TRUE
       OR coalesce(p_operacion ->> 'token_propietario_sha256', '') !~
          '^[a-f0-9]{64}$'
       OR p_operacion ->> 'token_propietario_sha256' =
          pg_catalog.repeat('0', 64)
       OR coalesce(p_operacion ->> 'ambito_activo_hmac', '') !~
          (
              '^hmac-sha256:vec[.]contratacion-temporal[.]'
              || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
              || '[a-f0-9]{64}$'
          )
       OR coalesce(
              p_operacion ->> 'huella_semantica_activa_hmac', ''
          ) !~ (
              '^hmac-sha256:vec[.]contratacion-temporal[.]'
              || 'cobertura-decision[.]semantica/v[1-9][0-9]{0,8}:'
              || '[a-f0-9]{64}$'
          )
       OR pg_catalog.right(p_operacion ->> 'ambito_activo_hmac', 64) =
          pg_catalog.repeat('0', 64)
       OR pg_catalog.right(
              p_operacion ->> 'huella_semantica_activa_hmac', 64
          ) = pg_catalog.repeat('0', 64)
       OR substring(
              p_operacion ->> 'ambito_activo_hmac'
              FROM '/v([1-9][0-9]{0,8}):'
          ) <> substring(
              p_operacion ->> 'huella_semantica_activa_hmac'
              FROM '/v([1-9][0-9]{0,8}):'
          )
       OR pg_catalog.jsonb_typeof(
              p_operacion -> 'ambitos_consulta_hmac'
          ) <> 'array'
       OR pg_catalog.jsonb_array_length(
              p_operacion -> 'ambitos_consulta_hmac'
          ) NOT BETWEEN 1 AND 4 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'carga de preparación de cobertura inválida';
    END IF;

    SELECT pg_catalog.array_agg(e.valor #>> '{}' ORDER BY e.posicion)
      INTO STRICT v_ambitos
      FROM pg_catalog.jsonb_array_elements(
          p_operacion -> 'ambitos_consulta_hmac'
      ) WITH ORDINALITY e(valor, posicion)
     WHERE pg_catalog.jsonb_typeof(e.valor) = 'string'
       AND e.valor #>> '{}' ~ (
           '^hmac-sha256:vec[.]contratacion-temporal[.]'
           || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
           || '[a-f0-9]{64}$'
       )
       AND pg_catalog.right(e.valor #>> '{}', 64) <>
           pg_catalog.repeat('0', 64);
    IF pg_catalog.cardinality(v_ambitos) <>
           pg_catalog.jsonb_array_length(
               p_operacion -> 'ambitos_consulta_hmac'
           )
       OR v_ambitos[1] <> p_operacion ->> 'ambito_activo_hmac'
       OR pg_catalog.cardinality(v_ambitos) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(v_ambitos) x)
          )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.generate_subscripts(v_ambitos, 1) s(i)
            WHERE s.i > 1
              AND substring(
                  v_ambitos[s.i] FROM '/v([1-9][0-9]{0,8}):'
              )::integer >= substring(
                  v_ambitos[s.i - 1] FROM '/v([1-9][0-9]{0,8}):'
              )::integer
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'ámbitos HMAC de cobertura inválidos';
    END IF;
    v_generacion_activa := substring(
        v_ambitos[1] FROM '/v([1-9][0-9]{0,8}):'
    )::integer;

    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema IN (2, 3)
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera O4-04C no disponible';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:o404c:' || a, 0)
    )
      FROM pg_catalog.unnest(v_ambitos) u(a)
     ORDER BY a COLLATE "C";

    SELECT pg_catalog.array_agg(DISTINCT x.ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_operacion_decision_cobertura x
     WHERE x.alias_ambito_hmac = ANY(v_ambitos);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'aliases de cobertura divergentes';
    END IF;

    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
        SELECT x.alias_ambito_hmac, x.alias_huella_semantica_hmac
          INTO STRICT v_ambito_persistido, v_semantica_persistida
          FROM
            vec_contratacion_temporal.alias_operacion_decision_cobertura x
          JOIN pg_catalog.unnest(v_ambitos)
               WITH ORDINALITY u(ambito, posicion)
            ON u.ambito = x.alias_ambito_hmac
         WHERE x.ambito_raiz_hmac = v_raiz
         ORDER BY u.posicion
         LIMIT 1;
        SELECT * INTO STRICT v_base
          FROM
            vec_contratacion_temporal.reserva_operacion_decision_cobertura b
         WHERE b.ambito_raiz_hmac = v_raiz
         FOR UPDATE;
        IF v_base.organizacion_ref <>
               p_operacion ->> 'organizacion_ref'
           OR v_base.expediente_ref <> p_operacion ->> 'expediente_ref'
           OR v_base.version_expediente <>
              (p_operacion ->> 'version_expediente')::numeric THEN
            RETURN QUERY SELECT
                'colision'::text,
                v_ambito_persistido,
                v_semantica_persistida,
                ''::text;
            RETURN;
        END IF;
        IF p_par_persistido_validado IS NULL THEN
            RETURN QUERY SELECT
                'requiere_validacion'::text,
                v_ambito_persistido,
                v_semantica_persistida,
                ''::text;
            RETURN;
        END IF;
        SELECT pg_catalog.array_agg(clave ORDER BY clave)
          INTO v_claves
          FROM pg_catalog.jsonb_object_keys(
              p_par_persistido_validado
          ) c(clave);
        IF v_claves IS DISTINCT FROM ARRAY[
               'ambito_hmac', 'huella_semantica_hmac'
           ]::text[]
           OR p_par_persistido_validado ->> 'ambito_hmac' <>
              v_ambito_persistido
           OR p_par_persistido_validado ->> 'huella_semantica_hmac' <>
              v_semantica_persistida THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'validación HMAC persistida inválida';
        END IF;
    ELSE
        IF p_par_persistido_validado IS NOT NULL THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'validación HMAC inesperada';
        END IF;
        v_raiz := p_operacion ->> 'ambito_activo_hmac';
        v_ambito_persistido := v_raiz;
        v_semantica_persistida :=
            p_operacion ->> 'huella_semantica_activa_hmac';
        v_nueva := true;
    END IF;

    IF v_nueva THEN
        SELECT v.agregado_json,
               v.agregado_json
                   #>> '{analisis,actuacion_registro,recibo_ref}',
               vec_contratacion_temporal.huella_analisis_derivado_v2(
                   v.agregado_json -> 'analisis'
               )
          INTO STRICT v_agregado, v_analisis_ref, v_analisis_huella
          FROM vec_contratacion_temporal.expediente_integral_actual a
          JOIN vec_contratacion_temporal.expediente_version_integral v
            ON v.expediente_ref = a.expediente_ref
           AND v.version = a.version
         WHERE a.expediente_ref = p_operacion ->> 'expediente_ref'
           AND a.version =
               (p_operacion ->> 'version_expediente')::numeric
           AND v.origen_version = 'analisis_o3'
           AND v.agregado_json ->> 'organizacion_ref' =
               p_operacion ->> 'organizacion_ref'
           AND v.agregado_json ->> 'referencia' =
               p_operacion ->> 'expediente_ref'
           AND v.agregado_json ->> 'version' =
               p_operacion ->> 'version_expediente'
           AND vec_contratacion_temporal.analisis_rrhh_valido_v3(
                   v.agregado_json -> 'analisis'
               ) IS TRUE
           AND v.agregado_json
                   #>> '{analisis,actuacion_registro,recibo_ref}'
               ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
         FOR SHARE OF a, v;
        IF v_analisis_huella !~ '^[a-f0-9]{64}$'
           OR v_analisis_huella = pg_catalog.repeat('0', 64) THEN
            RAISE EXCEPTION USING
                ERRCODE = '55000',
                MESSAGE = 'análisis O3 no confiable';
        END IF;
        INSERT INTO
        vec_contratacion_temporal.reserva_operacion_decision_cobertura (
            ambito_raiz_hmac,
            reserva_ref,
            recibo_ref,
            actuacion_ref,
            auditoria_ref,
            evento_ref,
            correlacion_vec_ref,
            decision_vec_ref,
            organizacion_ref,
            expediente_ref,
            version_expediente,
            analisis_ref,
            analisis_huella_sha256,
            huella_semantica_raiz_hmac,
            creada_en
        ) VALUES (
            v_raiz,
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'reserva:ct:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'recibo:ct:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'actuacion:ct:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'auditoria:ct:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'evento:ct:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'correlacion:vec:cobertura:', v_raiz
            ),
            vec_contratacion_temporal.o404c_referencia_derivada_v1(
                'decision:vec:cobertura:', v_raiz
            ),
            p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'expediente_ref',
            (p_operacion ->> 'version_expediente')::numeric,
            v_analisis_ref,
            v_analisis_huella,
            p_operacion ->> 'huella_semantica_activa_hmac',
            v_ahora
        );
        INSERT INTO
        vec_contratacion_temporal.alias_operacion_decision_cobertura
        VALUES (
            p_operacion ->> 'ambito_activo_hmac',
            v_raiz,
            v_generacion_activa,
            p_operacion ->> 'huella_semantica_activa_hmac',
            v_ahora
        );
        INSERT INTO
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version
        VALUES (
            v_raiz,
            1,
            'reservada',
            1,
            p_operacion ->> 'token_propietario_sha256',
            v_ahora,
            v_ahora + interval '5 seconds',
            NULL,
            NULL
        );
        INSERT INTO
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual
        VALUES (v_raiz, 1);
    ELSE
        INSERT INTO
        vec_contratacion_temporal.alias_operacion_decision_cobertura
        VALUES (
            p_operacion ->> 'ambito_activo_hmac',
            v_raiz,
            v_generacion_activa,
            p_operacion ->> 'huella_semantica_activa_hmac',
            v_ahora
        )
        ON CONFLICT (alias_ambito_hmac) DO NOTHING;
        IF NOT EXISTS (
            SELECT 1
              FROM
                vec_contratacion_temporal.alias_operacion_decision_cobertura x
             WHERE x.alias_ambito_hmac =
                   p_operacion ->> 'ambito_activo_hmac'
               AND x.ambito_raiz_hmac = v_raiz
               AND x.generacion = v_generacion_activa
               AND x.alias_huella_semantica_hmac =
                   p_operacion ->> 'huella_semantica_activa_hmac'
        ) THEN
            RETURN QUERY SELECT
                'colision'::text,
                v_ambito_persistido,
                v_semantica_persistida,
                ''::text;
            RETURN;
        END IF;
    END IF;

    SELECT * INTO STRICT v_base
      FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura b
     WHERE b.ambito_raiz_hmac = v_raiz
     FOR UPDATE;
    SELECT * INTO STRICT v_actual
      FROM
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual a
     WHERE a.ambito_raiz_hmac = v_raiz
     FOR UPDATE;
    SELECT * INTO STRICT v_version
      FROM
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
     WHERE v.ambito_raiz_hmac = v_raiz
       AND v.secuencia = v_actual.secuencia
     FOR UPDATE;

    IF v_version.estado IN ('aplicada', 'denegada_vec') THEN
        SELECT c.ambito_idempotencia_hmac,
               c.huella_semantica_hmac
          INTO STRICT v_ambito_persistido, v_semantica_persistida
          FROM
            vec_contratacion_temporal.confirmacion_operacion_decision_cobertura c
         WHERE c.ambito_raiz_hmac = v_raiz;
        v_carga :=
            vec_contratacion_temporal.o404c_carga_terminal_v1(v_raiz);
        IF v_carga IS NULL THEN
            RAISE EXCEPTION USING
                ERRCODE = '55000',
                MESSAGE = 'prueba terminal O4-04C divergente';
        END IF;
        RETURN QUERY SELECT
            'confirmada'::text,
            v_ambito_persistido,
            v_semantica_persistida,
            v_carga::text;
        RETURN;
    END IF;
    IF v_version.estado <> 'reservada' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado de reserva O4-04C desconocido';
    END IF;

    IF v_version.token_propietario_sha256 =
           p_operacion ->> 'token_propietario_sha256'
       AND v_ahora < v_version.propiedad_hasta THEN
        NULL;
    ELSIF v_ahora < v_version.propiedad_hasta
       OR v_version.token_propietario_sha256 =
          p_operacion ->> 'token_propietario_sha256' THEN
        RETURN QUERY SELECT
            'ocupada'::text,
            v_ambito_persistido,
            v_semantica_persistida,
            ''::text;
        RETURN;
    ELSE
        IF v_version.revision_cercado >=
           9007199254740991::numeric
           OR v_actual.secuencia >= 9007199254740991::numeric THEN
            RAISE EXCEPTION USING
                ERRCODE = '22003',
                MESSAGE = 'cercado O4-04C agotado';
        END IF;
        INSERT INTO
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version
        VALUES (
            v_raiz,
            v_actual.secuencia + 1,
            'reservada',
            v_version.revision_cercado + 1,
            p_operacion ->> 'token_propietario_sha256',
            v_ahora,
            v_ahora + interval '5 seconds',
            NULL,
            NULL
        );
        UPDATE
            vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual
           SET secuencia = v_actual.secuencia + 1
         WHERE ambito_raiz_hmac = v_raiz
           AND secuencia = v_actual.secuencia;
        GET DIAGNOSTICS v_filas = ROW_COUNT;
        IF v_filas <> 1 THEN
            RAISE EXCEPTION USING
                ERRCODE = '40001',
                MESSAGE = 'CAS de reapropiación O4-04C perdido';
        END IF;
        SELECT * INTO STRICT v_version
          FROM
            vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
         WHERE v.ambito_raiz_hmac = v_raiz
           AND v.secuencia = v_actual.secuencia + 1;
    END IF;

    SELECT v.agregado_json
      INTO STRICT v_agregado
      FROM vec_contratacion_temporal.expediente_version_integral v
     WHERE v.expediente_ref = v_base.expediente_ref
       AND v.version = v_base.version_expediente
       AND v.origen_version = 'analisis_o3'
       AND v.agregado_json ->> 'organizacion_ref' =
           v_base.organizacion_ref
       AND v.agregado_json ->> 'referencia' = v_base.expediente_ref
       AND v.agregado_json ->> 'version' =
           v_base.version_expediente::text
       AND v.agregado_json
               #>> '{analisis,actuacion_registro,recibo_ref}' =
           v_base.analisis_ref
       AND vec_contratacion_temporal.huella_analisis_derivado_v2(
               v.agregado_json -> 'analisis'
           ) = v_base.analisis_huella_sha256
       AND vec_contratacion_temporal.analisis_rrhh_valido_v3(
               v.agregado_json -> 'analisis'
           ) IS TRUE
     FOR SHARE;

    v_carga := pg_catalog.jsonb_build_object(
        'reserva_ref', v_base.reserva_ref,
        'recibo_ref', v_base.recibo_ref,
        'actuacion_ref', v_base.actuacion_ref,
        'auditoria_ref', v_base.auditoria_ref,
        'evento_ref', v_base.evento_ref,
        'correlacion_vec_ref', v_base.correlacion_vec_ref,
        'decision_vec_ref', v_base.decision_vec_ref,
        'analisis_ref', v_base.analisis_ref,
        'analisis_huella_sha256', v_base.analisis_huella_sha256,
        'token_propietario_sha256',
            v_version.token_propietario_sha256,
        'ambito_idempotencia_hmac',
            p_operacion ->> 'ambito_activo_hmac',
        'huella_semantica_hmac',
            p_operacion ->> 'huella_semantica_activa_hmac',
        'agregado_anterior', v_agregado,
        'revision_cercado_anterior',
            v_version.revision_cercado - 1,
        'revision_cercado', v_version.revision_cercado,
        'observada_en',
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                v_version.observada_en::text
            ),
        'propiedad_hasta',
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                v_version.propiedad_hasta::text
            )
    );
    RETURN QUERY SELECT
        'propietaria'::text,
        p_operacion ->> 'ambito_activo_hmac',
        p_operacion ->> 'huella_semantica_activa_hmac',
        v_carga::text;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 2,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 1;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404c_carga_terminal_v1(text),
    vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(
        jsonb, jsonb
    )
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;

GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(
        jsonb, jsonb
    )
TO vec_contratacion_temporal_ejecutor;

COMMIT;
