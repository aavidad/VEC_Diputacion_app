BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000007_preparacion_analisis', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.reserva_operacion_analisis') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para preparar análisis';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.reserva_operacion_analisis (
    ambito_raiz_hmac text PRIMARY KEY,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    operacion text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    artefacto_ref text NOT NULL,
    artefacto_huella_sha256 text NOT NULL,
    huella_semantica_raiz_hmac text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (ambito_raiz_hmac ~ (
        '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
        || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) AND pg_catalog.right(ambito_raiz_hmac, 64) <> repeat('0', 64)),
    CHECK (huella_semantica_raiz_hmac ~ (
        '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
        || 'huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) AND pg_catalog.right(huella_semantica_raiz_hmac, 64) <>
        repeat('0', 64)),
    CHECK (operacion IN ('registrar', 'rectificar')),
    CHECK (version_expediente BETWEEN 1 AND 9007199254740990::numeric),
    CHECK (artefacto_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (creada_en = pg_catalog.date_trunc('microseconds', creada_en))
);

CREATE TABLE vec_contratacion_temporal.alias_operacion_analisis (
    alias_ambito_hmac text PRIMARY KEY,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    alias_huella_semantica_hmac text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    UNIQUE (ambito_raiz_hmac, generacion),
    UNIQUE (ambito_raiz_hmac, alias_huella_semantica_hmac),
    CHECK (generacion BETWEEN 1 AND 999999999),
    CHECK (alias_ambito_hmac ~ (
        '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
        || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) AND substring(
        alias_ambito_hmac FROM '/v([1-9][0-9]{0,8}):'
    )::integer = generacion),
    CHECK (alias_huella_semantica_hmac ~ (
        '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
        || 'huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) AND substring(
        alias_huella_semantica_hmac FROM '/v([1-9][0-9]{0,8}):'
    )::integer = generacion)
);

CREATE TABLE vec_contratacion_temporal.reserva_operacion_analisis_version (
    ambito_raiz_hmac text NOT NULL,
    revision bigint NOT NULL,
    estado text NOT NULL,
    confirmada_en timestamptz(6),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (ambito_raiz_hmac, revision),
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    CHECK (revision BETWEEN 1 AND 9007199254740991),
    CHECK (
        (estado = 'reservada' AND confirmada_en IS NULL)
        OR (estado = 'confirmada' AND confirmada_en IS NOT NULL)
    ),
    CHECK (registrada_en = pg_catalog.date_trunc(
        'microseconds', registrada_en
    ))
);

CREATE TABLE vec_contratacion_temporal.reserva_operacion_analisis_actual (
    ambito_raiz_hmac text PRIMARY KEY,
    revision bigint NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac, revision)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis_version
);

-- O3-04 insertará esta confirmación en el mismo COMMIT que el agregado.
CREATE TABLE vec_contratacion_temporal.confirmacion_operacion_analisis (
    ambito_raiz_hmac text PRIMARY KEY,
    recibo_json jsonb NOT NULL,
    recibo_huella_sha256 text NOT NULL UNIQUE,
    confirmada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    CHECK (pg_catalog.jsonb_typeof(recibo_json) = 'object'),
    CHECK (pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(recibo_json::text, 'UTF8')
    ), 'hex') = recibo_huella_sha256),
    CHECK (confirmada_en = pg_catalog.date_trunc(
        'microseconds', confirmada_en
    ))
);

CREATE TABLE vec_contratacion_temporal.alias_consulta_operacion_analisis (
    alias_ambito_consulta_hmac text PRIMARY KEY,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.confirmacion_operacion_analisis,
    UNIQUE (ambito_raiz_hmac, generacion),
    CHECK (generacion BETWEEN 1 AND 999999999),
    CHECK (alias_ambito_consulta_hmac ~ (
        '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
        || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) AND substring(
        alias_ambito_consulta_hmac FROM '/v([1-9][0-9]{0,8}):'
    )::integer = generacion)
);

CREATE FUNCTION vec_contratacion_temporal.preparar_operacion_analisis_v1(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text, expediente_json text, recibo_json text,
    reserva_ref text, recibo_ref text, operacion text,
    organizacion_ref text, expediente_ref text,
    version_expediente bigint, actor_ref text, perfil_ref text,
    artefacto_ref text, artefacto_huella_sha256 text,
    ambito_hmac text, huella_semantica_hmac text, estado text
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_claves text[];
    v_pares jsonb;
    v_generaciones integer[];
    v_ambitos text[];
    v_huellas text[];
    v_raices text[];
    v_raiz text;
    v_elementos_invalidos boolean;
    v_filas bigint;
    v_identidad vec_contratacion_temporal.reserva_operacion_analisis%ROWTYPE;
    v_estado text;
    v_insertada boolean := false;
    v_expediente jsonb;
    v_recibo jsonb;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_statement_ms numeric;
    v_idle_ms numeric;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad de ejecución no autorizada';
    END IF;
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_statement_ms FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout';
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_idle_ms FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout';
    IF v_statement_ms IS NULL OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL OR v_idle_ms NOT BETWEEN 1 AND 20000 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'límites de ejecución ausentes o inválidos';
    END IF;
    IF p_operacion IS NULL
       OR pg_catalog.jsonb_typeof(p_operacion) <> 'object' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'preparación de análisis inválida';
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_operacion) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
        'actor_ref', 'artefacto_huella_sha256', 'artefacto_ref',
        'esquema', 'expediente_ref', 'operacion', 'organizacion_ref',
        'perfil_ref', 'sellos_hmac', 'version_expediente'
    ]::text[] THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'campos de preparación de análisis inválidos';
    END IF;
    IF p_operacion ->> 'esquema' <>
           'vec.contratacion-temporal.preparar-operacion-analisis.v1'
       OR p_operacion ->> 'operacion' NOT IN ('registrar', 'rectificar')
       OR pg_catalog.jsonb_typeof(p_operacion -> 'version_expediente')
            <> 'number'
       OR (p_operacion ->> 'version_expediente')::numeric
            NOT BETWEEN 1 AND 9007199254740990::numeric
       OR coalesce(p_operacion ->> 'organizacion_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'expediente_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'actor_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'perfil_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'artefacto_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'artefacto_huella_sha256', '')
            !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de preparación de análisis inválido';
    END IF;
    IF pg_catalog.jsonb_typeof(p_operacion -> 'sellos_hmac') <> 'object'
       OR pg_catalog.jsonb_typeof(
           p_operacion #> '{sellos_hmac,activo}'
       ) <> 'object'
       OR pg_catalog.jsonb_typeof(
           p_operacion #> '{sellos_hmac,retenidos}'
       ) <> 'array' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'sellos de análisis inválidos';
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(
          p_operacion -> 'sellos_hmac'
      ) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY['activo', 'retenidos']::text[]
       OR pg_catalog.jsonb_array_length(
           p_operacion #> '{sellos_hmac,retenidos}'
       ) > 3 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'colección de sellos de análisis inválida';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(
        p_operacion #> '{sellos_hmac,activo}'
    ) || (p_operacion #> '{sellos_hmac,retenidos}');
    SELECT
        pg_catalog.array_agg((e.v ->> 'generacion')::integer ORDER BY e.i),
        pg_catalog.array_agg(e.v ->> 'ambito_hmac' ORDER BY e.i),
        pg_catalog.array_agg(e.v ->> 'huella_peticion_hmac' ORDER BY e.i),
        pg_catalog.bool_or(
            pg_catalog.jsonb_typeof(e.v) <> 'object'
            OR (SELECT pg_catalog.array_agg(k ORDER BY k)
                  FROM pg_catalog.jsonb_object_keys(e.v) AS x(k))
               IS DISTINCT FROM ARRAY[
                   'ambito_hmac', 'generacion', 'huella_peticion_hmac'
               ]::text[]
            OR pg_catalog.jsonb_typeof(e.v -> 'generacion') <> 'number'
            OR coalesce(e.v ->> 'ambito_hmac', '') !~ (
                '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
                || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR coalesce(e.v ->> 'huella_peticion_hmac', '') !~ (
                '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
                || 'huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR substring(e.v ->> 'ambito_hmac'
                FROM '/v([1-9][0-9]{0,8}):')::integer <>
                (e.v ->> 'generacion')::integer
            OR substring(e.v ->> 'huella_peticion_hmac'
                FROM '/v([1-9][0-9]{0,8}):')::integer <>
                (e.v ->> 'generacion')::integer
        )
      INTO STRICT v_generaciones, v_ambitos, v_huellas,
          v_elementos_invalidos
      FROM pg_catalog.jsonb_array_elements(v_pares)
           WITH ORDINALITY AS e(v, i);
    IF v_elementos_invalidos
       OR pg_catalog.cardinality(v_generaciones) NOT BETWEEN 1 AND 4
       OR v_generaciones[1] IS NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.generate_subscripts(
               v_generaciones, 1
           ) AS s(i)
           WHERE s.i > 1
             AND v_generaciones[s.i] >= v_generaciones[s.i - 1]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'pares HMAC de análisis inválidos';
    END IF;
    v_insertada := false;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:analisis:' || a, 0)
    ) FROM pg_catalog.unnest(v_ambitos) AS u(a)
      ORDER BY a COLLATE "C";
    SELECT pg_catalog.array_agg(DISTINCT x.ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_operacion_analisis x
     WHERE x.alias_ambito_hmac = ANY(v_ambitos);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'alias de análisis divergentes';
    END IF;
    v_raiz := CASE WHEN pg_catalog.cardinality(v_raices) = 1
                   THEN v_raices[1] ELSE v_ambitos[1] END;
    SELECT v.agregado_json INTO STRICT v_expediente
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref
     WHERE a.expediente_ref = p_operacion ->> 'expediente_ref'
       AND v.version = (p_operacion ->> 'version_expediente')::numeric
       AND v.agregado_json ->> 'organizacion_ref' =
           p_operacion ->> 'organizacion_ref'
       AND (
           a.version = v.version
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.reserva_operacion_analisis r
                WHERE r.ambito_raiz_hmac = v_raiz
                  AND r.expediente_ref = v.expediente_ref
                  AND r.version_expediente = v.version
           )
       )
     FOR SHARE OF a, v;
    INSERT INTO vec_contratacion_temporal.reserva_operacion_analisis (
        ambito_raiz_hmac, reserva_ref, recibo_ref, operacion,
        organizacion_ref, expediente_ref, version_expediente,
        actor_ref, perfil_ref, artefacto_ref, artefacto_huella_sha256,
        huella_semantica_raiz_hmac, creada_en
    ) VALUES (
        v_raiz,
        'res_ct_an_' || pg_catalog.substr(pg_catalog.encode(
            pg_catalog.sha256(pg_catalog.convert_to(
                v_raiz || ':' || (p_operacion ->> 'expediente_ref')
                    || ':' || (p_operacion ->> 'version_expediente'), 'UTF8'
            )), 'hex'), 1, 32),
        'rec_ct_an_' || pg_catalog.substr(pg_catalog.encode(
            pg_catalog.sha256(pg_catalog.convert_to(
                'recibo:' || v_raiz || ':' ||
                    (p_operacion ->> 'expediente_ref'), 'UTF8'
            )), 'hex'), 1, 32),
        p_operacion ->> 'operacion', p_operacion ->> 'organizacion_ref',
        p_operacion ->> 'expediente_ref',
        (p_operacion ->> 'version_expediente')::numeric,
        p_operacion ->> 'actor_ref', p_operacion ->> 'perfil_ref',
        p_operacion ->> 'artefacto_ref',
        p_operacion ->> 'artefacto_huella_sha256', v_huellas[1], v_ahora
    ) ON CONFLICT (ambito_raiz_hmac) DO NOTHING;
    GET DIAGNOSTICS v_filas = ROW_COUNT;
    v_insertada := v_filas = 1;
    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.reserva_operacion_analisis r
     WHERE r.ambito_raiz_hmac = v_raiz FOR UPDATE;
    IF v_identidad.operacion <> p_operacion ->> 'operacion'
       OR v_identidad.organizacion_ref <> p_operacion ->> 'organizacion_ref'
       OR v_identidad.expediente_ref <> p_operacion ->> 'expediente_ref'
       OR v_identidad.version_expediente <>
            (p_operacion ->> 'version_expediente')::numeric
       OR v_identidad.actor_ref <> p_operacion ->> 'actor_ref'
       OR v_identidad.perfil_ref <> p_operacion ->> 'perfil_ref'
       OR v_identidad.artefacto_ref <> p_operacion ->> 'artefacto_ref'
       OR v_identidad.artefacto_huella_sha256 <>
            p_operacion ->> 'artefacto_huella_sha256'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.generate_subscripts(v_ambitos, 1) s(i)
           WHERE v_ambitos[s.i] = v_raiz
             AND v_huellas[s.i] = v_identidad.huella_semantica_raiz_hmac
       ) THEN
        RETURN QUERY SELECT 'idempotencia_reutilizada', '', '',
            v_identidad.reserva_ref, v_identidad.recibo_ref,
            p_operacion ->> 'operacion', p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'expediente_ref',
            (p_operacion ->> 'version_expediente')::bigint,
            p_operacion ->> 'actor_ref', p_operacion ->> 'perfil_ref',
            p_operacion ->> 'artefacto_ref',
            p_operacion ->> 'artefacto_huella_sha256',
            v_ambitos[1], v_huellas[1], 'reservada';
        RETURN;
    END IF;
    FOR i IN 1..pg_catalog.cardinality(v_ambitos) LOOP
        INSERT INTO vec_contratacion_temporal.alias_operacion_analisis
        VALUES (
            v_ambitos[i], v_raiz, v_generaciones[i], v_huellas[i], v_ahora
        ) ON CONFLICT DO NOTHING;
        IF NOT EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_operacion_analisis x
             WHERE x.alias_ambito_hmac = v_ambitos[i]
               AND x.ambito_raiz_hmac = v_raiz
               AND x.generacion = v_generaciones[i]
               AND x.alias_huella_semantica_hmac = v_huellas[i]
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'par HMAC de análisis divergente';
        END IF;
    END LOOP;
    IF v_insertada THEN
        INSERT INTO vec_contratacion_temporal.reserva_operacion_analisis_version
        VALUES (v_raiz, 1, 'reservada', NULL, v_ahora);
        INSERT INTO vec_contratacion_temporal.reserva_operacion_analisis_actual
        VALUES (v_raiz, 1);
    END IF;
    SELECT v.estado INTO STRICT v_estado
      FROM vec_contratacion_temporal.reserva_operacion_analisis_actual a
      JOIN vec_contratacion_temporal.reserva_operacion_analisis_version v
        USING (ambito_raiz_hmac, revision)
     WHERE a.ambito_raiz_hmac = v_raiz;
    SELECT c.recibo_json INTO v_recibo
      FROM vec_contratacion_temporal.confirmacion_operacion_analisis c
     WHERE c.ambito_raiz_hmac = v_raiz;
    RETURN QUERY SELECT
        CASE WHEN v_estado = 'confirmada' THEN 'confirmada'
             WHEN v_insertada THEN 'reservada' ELSE 'reutilizada' END,
        CASE WHEN v_estado = 'reservada' THEN v_expediente::text ELSE '' END,
        CASE WHEN v_estado = 'confirmada' THEN v_recibo::text ELSE '' END,
        v_identidad.reserva_ref, v_identidad.recibo_ref,
        v_identidad.operacion, v_identidad.organizacion_ref,
        v_identidad.expediente_ref, v_identidad.version_expediente::bigint,
        v_identidad.actor_ref, v_identidad.perfil_ref,
        v_identidad.artefacto_ref, v_identidad.artefacto_huella_sha256,
        v_identidad.ambito_raiz_hmac,
        v_identidad.huella_semantica_raiz_hmac, v_estado;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_operacion_analisis_v1(
    p_ambitos jsonb
)
RETURNS TABLE (recibo_json text)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_raices text[];
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR p_ambitos IS NULL
       OR pg_catalog.jsonb_typeof(p_ambitos) <> 'array'
       OR pg_catalog.jsonb_array_length(p_ambitos) NOT BETWEEN 1 AND 4
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v)
           WHERE pg_catalog.jsonb_typeof(e.v) <> 'string'
              OR e.v #>> '{}' !~ (
                  '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
                  || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta de análisis inválida';
    END IF;
    SELECT pg_catalog.array_agg(DISTINCT a.ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_consulta_operacion_analisis a
     WHERE a.alias_ambito_consulta_hmac IN (
         SELECT e.v #>> '{}'
           FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v)
     );
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'consulta de análisis divergente';
    END IF;
    RETURN QUERY
    SELECT c.recibo_json::text
      FROM vec_contratacion_temporal.confirmacion_operacion_analisis c
     WHERE c.ambito_raiz_hmac = v_raices[1];
END
$funcion$;

CREATE TRIGGER reserva_operacion_analisis_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.reserva_operacion_analisis
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER alias_operacion_analisis_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.alias_operacion_analisis
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER reserva_operacion_analisis_version_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.reserva_operacion_analisis_version
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER confirmacion_operacion_analisis_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.confirmacion_operacion_analisis
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER alias_consulta_operacion_analisis_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.alias_consulta_operacion_analisis
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'reserva_operacion_analisis',
        'alias_operacion_analisis',
        'reserva_operacion_analisis_version',
        'reserva_operacion_analisis_actual',
        'confirmacion_operacion_analisis',
        'alias_consulta_operacion_analisis'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'ENABLE ROW LEVEL SECURITY', v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY %I ON vec_contratacion_temporal.%I '
            || 'TO vec_contratacion_temporal_propietario '
            || 'USING (true) WITH CHECK (true)',
            v_tabla || '_propietario', v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'FORCE ROW LEVEL SECURITY', v_tabla
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_contratacion_temporal
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb),
    vec_contratacion_temporal.consultar_operacion_analisis_v1(jsonb)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb),
    vec_contratacion_temporal.consultar_operacion_analisis_v1(jsonb)
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
