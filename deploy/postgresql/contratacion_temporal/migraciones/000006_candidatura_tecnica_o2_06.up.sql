BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000006_candidatura_tecnica_o2_06',
        0
    )
);
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.identidad_reserva_alta'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.alias_ambito_alta'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.candidatura_alta_tecnica'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(text[],text[],text,text,text,text,text,text,text,text)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para candidatura técnica O2-06';
    END IF;
END
$prevalidacion$;
CREATE TABLE vec_contratacion_temporal.candidatura_alta_tecnica (
    ambito_raiz_hmac text PRIMARY KEY,
    huella_raiz_hmac text NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    creada_en timestamptz NOT NULL,
    CONSTRAINT candidatura_tecnica_ambito_valido CHECK (
        ambito_raiz_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND pg_catalog.right(ambito_raiz_hmac, 64) <>
            pg_catalog.repeat('0', 64)
    ),
    CONSTRAINT candidatura_tecnica_huella_valida CHECK (
        huella_raiz_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND pg_catalog.right(huella_raiz_hmac, 64) <>
            pg_catalog.repeat('0', 64)
    ),
    CONSTRAINT candidatura_tecnica_generacion_alineada CHECK (
        pg_catalog.substring(
            ambito_raiz_hmac,
            '/v([1-9][0-9]{0,8}):'
        ) = pg_catalog.substring(
            huella_raiz_hmac,
            '/v([1-9][0-9]{0,8}):'
        )
    ),
    CONSTRAINT candidatura_tecnica_referencias_validas CHECK (
        reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
        AND recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    ),
    CONSTRAINT candidatura_tecnica_instante_canonico CHECK (
        creada_en = pg_catalog.date_trunc('microseconds', creada_en)
    )
);
CREATE TABLE vec_contratacion_temporal.alias_ambito_candidatura_alta (
    alias_hmac text PRIMARY KEY,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    registrada_en timestamptz NOT NULL,
    CONSTRAINT alias_ambito_candidatura_raiz_fk
        FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.candidatura_alta_tecnica(
            ambito_raiz_hmac
        )
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT alias_ambito_candidatura_generacion_unica
        UNIQUE (ambito_raiz_hmac, generacion),
    CONSTRAINT alias_ambito_candidatura_formato CHECK (
        alias_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND pg_catalog.right(alias_hmac, 64) <>
            pg_catalog.repeat('0', 64)
        AND pg_catalog.substring(
            alias_hmac,
            '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
    ),
    CONSTRAINT alias_ambito_candidatura_instante CHECK (
        registrada_en = pg_catalog.date_trunc(
            'microseconds',
            registrada_en
        )
    )
);
CREATE TABLE vec_contratacion_temporal.alias_huella_candidatura_alta (
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    alias_hmac text NOT NULL,
    registrada_en timestamptz NOT NULL,
    PRIMARY KEY (ambito_raiz_hmac, generacion),
    CONSTRAINT alias_huella_candidatura_raiz_fk
        FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.candidatura_alta_tecnica(
            ambito_raiz_hmac
        )
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT alias_huella_candidatura_alias_unico
        UNIQUE (ambito_raiz_hmac, alias_hmac),
    CONSTRAINT alias_huella_candidatura_formato CHECK (
        alias_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND pg_catalog.right(alias_hmac, 64) <>
            pg_catalog.repeat('0', 64)
        AND pg_catalog.substring(
            alias_hmac,
            '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
    ),
    CONSTRAINT alias_huella_candidatura_instante CHECK (
        registrada_en = pg_catalog.date_trunc(
            'microseconds',
            registrada_en
        )
    )
);
INSERT INTO vec_contratacion_temporal.candidatura_alta_tecnica (
    ambito_raiz_hmac,
    huella_raiz_hmac,
    reserva_ref,
    expediente_ref,
    numero_visible,
    recibo_ref,
    organizacion_ref,
    actor_ref,
    perfil_ref,
    creada_en
)
SELECT
    ambito_hmac,
    huella_peticion_hmac,
    reserva_ref,
    expediente_ref,
    numero_visible,
    recibo_ref,
    organizacion_ref,
    actor_ref,
    perfil_ref,
    creada_en
FROM vec_contratacion_temporal.identidad_reserva_alta;
INSERT INTO vec_contratacion_temporal.alias_ambito_candidatura_alta (
    alias_hmac,
    ambito_raiz_hmac,
    generacion,
    registrada_en
)
SELECT alias_hmac, ambito_raiz_hmac, generacion, registrada_en
FROM vec_contratacion_temporal.alias_ambito_alta;
INSERT INTO vec_contratacion_temporal.alias_huella_candidatura_alta (
    ambito_raiz_hmac,
    generacion,
    alias_hmac,
    registrada_en
)
SELECT ambito_raiz_hmac, generacion, alias_hmac, registrada_en
FROM vec_contratacion_temporal.alias_huella_alta;
CREATE TRIGGER candidatura_alta_tecnica_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.candidatura_alta_tecnica
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER alias_ambito_candidatura_alta_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.alias_ambito_candidatura_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER alias_huella_candidatura_alta_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.alias_huella_candidatura_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE FUNCTION
vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
    p_ambitos text[],
    p_huellas text[],
    p_organizacion_ref text,
    p_actor_ref text,
    p_perfil_ref text,
    p_reserva_ref text,
    p_expediente_ref text,
    p_numero_visible text,
    p_recibo_ref text,
    p_ambito_propuesto text
)
RETURNS TABLE (
    resultado text,
    ambito_hmac text,
    huella_peticion_hmac text,
    reserva_ref text,
    expediente_ref text,
    numero_visible text,
    recibo_ref text,
    organizacion_ref text,
    actor_ref text,
    perfil_ref text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_total integer;
    v_indice integer;
    v_generacion integer;
    v_anterior integer;
    v_generaciones integer[] := ARRAY[]::integer[];
    v_generaciones_politica integer[];
    v_raices text[];
    v_raiz text;
    v_ahora timestamptz(6) :=
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
    v_candidatura record;
    v_insertada boolean := false;
    v_coincide boolean := false;
    v_filas bigint;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_ejecutor',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_migrador',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_propietario',
           'MEMBER'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'resolución de candidatura rechazada';
    END IF;
    v_total := pg_catalog.array_length(p_ambitos, 1);
    IF v_total IS NULL
       OR v_total NOT BETWEEN 1 AND 4
       OR v_total <> pg_catalog.array_length(p_huellas, 1)
       OR p_ambito_propuesto IS DISTINCT FROM p_ambitos[1]
       OR p_organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_reserva_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_numero_visible !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR p_recibo_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'candidatura técnica inválida';
    END IF;
    FOR v_indice IN 1..v_total LOOP
        IF p_ambitos[v_indice] !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           )
           OR p_huellas[v_indice] !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           )
           OR pg_catalog.right(p_ambitos[v_indice], 64) =
              pg_catalog.repeat('0', 64)
           OR pg_catalog.right(p_huellas[v_indice], 64) =
              pg_catalog.repeat('0', 64) THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'sellos de candidatura inválidos';
        END IF;
        v_generacion := pg_catalog.substring(
            p_ambitos[v_indice],
            '/v([1-9][0-9]{0,8}):'
        )::integer;
        IF v_generacion <> pg_catalog.substring(
               p_huellas[v_indice],
               '/v([1-9][0-9]{0,8}):'
           )::integer
           OR (v_anterior IS NOT NULL AND v_generacion >= v_anterior) THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'generaciones HMAC no alineadas';
        END IF;
        v_generaciones := pg_catalog.array_append(
            v_generaciones,
            v_generacion
        );
        v_anterior := v_generacion;
    END LOOP;
    SELECT pg_catalog.array_agg(generacion ORDER BY posicion)
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'política HMAC no vigente';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_ct:candidatura:' || alias_hmac,
            0
        )
    )
    FROM pg_catalog.unnest(p_ambitos) AS alias(alias_hmac)
    ORDER BY alias_hmac COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac
           )
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_candidatura_alta
     WHERE alias_hmac = ANY (p_ambitos);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'candidaturas técnicas divergentes';
    END IF;
    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
        SELECT * INTO STRICT v_candidatura
          FROM vec_contratacion_temporal.candidatura_alta_tecnica
         WHERE ambito_raiz_hmac = v_raiz;
        SELECT pg_catalog.count(*) = v_total
               AND pg_catalog.bool_and(
                   aa.ambito_raiz_hmac = v_raiz
                   AND ah.alias_hmac = p_huellas[pares.orden]
               )
          INTO v_coincide
          FROM pg_catalog.unnest(p_ambitos)
               WITH ORDINALITY AS pares(alias_hmac, orden)
          LEFT JOIN
               vec_contratacion_temporal.alias_ambito_candidatura_alta aa
            ON aa.alias_hmac = pares.alias_hmac
          LEFT JOIN
               vec_contratacion_temporal.alias_huella_candidatura_alta ah
            ON ah.ambito_raiz_hmac = aa.ambito_raiz_hmac
           AND ah.generacion = aa.generacion;
        IF NOT coalesce(v_coincide, false)
           OR v_candidatura.organizacion_ref <> p_organizacion_ref
           OR v_candidatura.actor_ref <> p_actor_ref
           OR v_candidatura.perfil_ref <> p_perfil_ref THEN
            RETURN QUERY SELECT
                'idempotencia_reutilizada',
                v_candidatura.ambito_raiz_hmac,
                v_candidatura.huella_raiz_hmac,
                v_candidatura.reserva_ref,
                v_candidatura.expediente_ref,
                v_candidatura.numero_visible,
                v_candidatura.recibo_ref,
                v_candidatura.organizacion_ref,
                v_candidatura.actor_ref,
                v_candidatura.perfil_ref;
            RETURN;
        END IF;
    ELSE
        INSERT INTO vec_contratacion_temporal.candidatura_alta_tecnica (
            ambito_raiz_hmac,
            huella_raiz_hmac,
            reserva_ref,
            expediente_ref,
            numero_visible,
            recibo_ref,
            organizacion_ref,
            actor_ref,
            perfil_ref,
            creada_en
        ) VALUES (
            p_ambitos[1],
            p_huellas[1],
            p_reserva_ref,
            p_expediente_ref,
            p_numero_visible,
            p_recibo_ref,
            p_organizacion_ref,
            p_actor_ref,
            p_perfil_ref,
            v_ahora
        ) ON CONFLICT DO NOTHING;
        GET DIAGNOSTICS v_filas = ROW_COUNT;
        v_insertada := v_filas = 1;
        IF NOT v_insertada THEN
            RETURN QUERY SELECT
                'conflicto_referencias',
                p_ambitos[1],
                p_huellas[1],
                p_reserva_ref,
                p_expediente_ref,
                p_numero_visible,
                p_recibo_ref,
                p_organizacion_ref,
                p_actor_ref,
                p_perfil_ref;
            RETURN;
        END IF;
        v_raiz := p_ambitos[1];
        SELECT * INTO STRICT v_candidatura
          FROM vec_contratacion_temporal.candidatura_alta_tecnica
         WHERE ambito_raiz_hmac = v_raiz;
    END IF;
    FOR v_indice IN 1..v_total LOOP
        v_generacion := v_generaciones[v_indice];
        IF EXISTS (
            SELECT 1
              FROM
                   vec_contratacion_temporal.alias_ambito_candidatura_alta
             WHERE alias_hmac = p_ambitos[v_indice]
               AND ambito_raiz_hmac <> v_raiz
        ) OR EXISTS (
            SELECT 1
              FROM
                   vec_contratacion_temporal.alias_huella_candidatura_alta
             WHERE ambito_raiz_hmac = v_raiz
               AND generacion = v_generacion
               AND alias_hmac <> p_huellas[v_indice]
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'alias de candidatura en conflicto';
        END IF;
        INSERT INTO
            vec_contratacion_temporal.alias_ambito_candidatura_alta (
                alias_hmac,
                ambito_raiz_hmac,
                generacion,
                registrada_en
            ) VALUES (
                p_ambitos[v_indice],
                v_raiz,
                v_generacion,
                v_ahora
            ) ON CONFLICT DO NOTHING;
        INSERT INTO
            vec_contratacion_temporal.alias_huella_candidatura_alta (
                ambito_raiz_hmac,
                generacion,
                alias_hmac,
                registrada_en
            ) VALUES (
                v_raiz,
                v_generacion,
                p_huellas[v_indice],
                v_ahora
            ) ON CONFLICT DO NOTHING;
    END LOOP;
    RETURN QUERY SELECT
        CASE WHEN v_insertada THEN 'estabilizada' ELSE 'recuperada' END,
        v_candidatura.ambito_raiz_hmac,
        v_candidatura.huella_raiz_hmac,
        v_candidatura.reserva_ref,
        v_candidatura.expediente_ref,
        v_candidatura.numero_visible,
        v_candidatura.recibo_ref,
        v_candidatura.organizacion_ref,
        v_candidatura.actor_ref,
        v_candidatura.perfil_ref;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v2(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea,
    p_alta_canonica bytea,
    p_sellos_hmac_canonicos bytea
)
RETURNS TABLE (
    expediente_ref text,
    numero_visible text,
    version numeric,
    recibo_ref text,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz,
    recibo_huella_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    a jsonb;
    s jsonb;
    v_ambitos text[];
    v_huellas text[];
    v_generaciones integer[];
    v_raices text[];
    v_raiz text;
    v_candidatura record;
    v_identidad record;
    v_indice integer;
    v_ahora timestamptz(6) :=
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_ejecutor',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_migrador',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_propietario',
           'MEMBER'
       )
       OR pg_catalog.octet_length(p_alta_canonica)
          NOT BETWEEN 256 AND 32768
       OR pg_catalog.octet_length(p_sellos_hmac_canonicos)
          NOT BETWEEN 256 AND 8192 THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confirmación O2-06 rechazada';
    END IF;
    a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
    s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
    IF pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos O2-06 inválidos';
    END IF;
    SELECT
        pg_catalog.array_agg(
            elemento ->> 'ambito_hmac' ORDER BY orden
        ),
        pg_catalog.array_agg(
            elemento ->> 'huella_hmac' ORDER BY orden
        ),
        pg_catalog.array_agg(
            (elemento ->> 'generacion')::integer ORDER BY orden
        )
      INTO v_ambitos, v_huellas, v_generaciones
      FROM (
          SELECT s -> 'activo' AS elemento, 0::bigint AS orden
          UNION ALL
          SELECT valor, posicion
            FROM pg_catalog.jsonb_array_elements(s -> 'retenidos')
                 WITH ORDINALITY AS retenido(valor, posicion)
      ) AS pares;
    IF pg_catalog.array_length(v_ambitos, 1) NOT BETWEEN 1 AND 4 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'matriz HMAC O2-06 inválida';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_ct:candidatura:' || alias_hmac,
            0
        )
    )
    FROM pg_catalog.unnest(v_ambitos) AS alias(alias_hmac)
    ORDER BY alias_hmac COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac
           )
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_candidatura_alta
     WHERE alias_hmac = ANY (v_ambitos);
    IF pg_catalog.cardinality(v_raices) <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'candidatura técnica no acreditada';
    END IF;
    v_raiz := v_raices[1];
    SELECT * INTO STRICT v_candidatura
      FROM vec_contratacion_temporal.candidatura_alta_tecnica
     WHERE ambito_raiz_hmac = v_raiz;
    IF v_candidatura.reserva_ref <> a ->> 'reserva_ref'
       OR v_candidatura.expediente_ref <> a ->> 'expediente_ref'
       OR v_candidatura.numero_visible <> a ->> 'numero_visible'
       OR v_candidatura.recibo_ref <> a ->> 'recibo_ref'
       OR v_candidatura.organizacion_ref <> a ->> 'organizacion_ref'
       OR v_candidatura.actor_ref <> a ->> 'actor_ref'
       OR v_candidatura.perfil_ref <> a ->> 'perfil_ref'
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.unnest(v_ambitos)
                  WITH ORDINALITY AS pares(alias_hmac, orden)
             JOIN
                  vec_contratacion_temporal.alias_ambito_candidatura_alta aa
               ON aa.alias_hmac = pares.alias_hmac
              AND aa.ambito_raiz_hmac = v_raiz
             JOIN
                  vec_contratacion_temporal.alias_huella_candidatura_alta ah
              ON ah.ambito_raiz_hmac = v_raiz
              AND ah.generacion = aa.generacion
              AND ah.alias_hmac = v_huellas[pares.orden]
       ) <> pg_catalog.cardinality(v_ambitos) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'candidatura técnica divergente';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:alias:' || alias_hmac, 0)
    )
    FROM pg_catalog.unnest(v_ambitos) AS alias(alias_hmac)
    ORDER BY alias_hmac COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac
           )
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_ambitos);
    IF pg_catalog.cardinality(v_raices) > 1
       OR (
           pg_catalog.cardinality(v_raices) = 1
           AND v_raices[1] <> v_raiz
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'candidatura y reserva divergentes';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.identidad_reserva_alta
         WHERE ambito_hmac = v_raiz
    ) THEN
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac,
            reserva_ref,
            expediente_ref,
            numero_visible,
            recibo_ref,
            huella_peticion_hmac,
            organizacion_ref,
            actor_ref,
            perfil_ref,
            creada_en
        ) VALUES (
            v_raiz,
            v_candidatura.reserva_ref,
            v_candidatura.expediente_ref,
            v_candidatura.numero_visible,
            v_candidatura.recibo_ref,
            v_candidatura.huella_raiz_hmac,
            v_candidatura.organizacion_ref,
            v_candidatura.actor_ref,
            v_candidatura.perfil_ref,
            (a #>> '{actuacion,realizada_en}')::timestamptz
        ) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac,
            revision,
            estado,
            registrada_en
        ) VALUES (v_raiz, 1, 'reservada', v_ahora);
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual (
            ambito_hmac,
            revision
        ) VALUES (v_raiz, 1);
    END IF;
    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    IF v_identidad.reserva_ref <> v_candidatura.reserva_ref
       OR v_identidad.expediente_ref <> v_candidatura.expediente_ref
       OR v_identidad.numero_visible <> v_candidatura.numero_visible
       OR v_identidad.recibo_ref <> v_candidatura.recibo_ref
       OR v_identidad.organizacion_ref <> v_candidatura.organizacion_ref
       OR v_identidad.actor_ref <> v_candidatura.actor_ref
       OR v_identidad.perfil_ref <> v_candidatura.perfil_ref THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'identidad de candidatura divergente';
    END IF;
    FOR v_indice IN 1..pg_catalog.array_length(v_ambitos, 1) LOOP
        INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
            alias_hmac,
            ambito_raiz_hmac,
            generacion,
            registrada_en
        ) VALUES (
            v_ambitos[v_indice],
            v_raiz,
            v_generaciones[v_indice],
            v_ahora
        ) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.alias_huella_alta (
            ambito_raiz_hmac,
            generacion,
            alias_hmac,
            registrada_en
        ) VALUES (
            v_raiz,
            v_generaciones[v_indice],
            v_huellas[v_indice],
            v_ahora
        ) ON CONFLICT DO NOTHING;
    END LOOP;
    RETURN QUERY
    SELECT *
      FROM vec_contratacion_temporal.confirmar_alta_atestada_v1(
          p_capacidad_canonica,
          p_decision_canonica,
          p_motivo_canonico,
          p_contexto_actor_canonico,
          p_persona_version,
          p_perfil_version,
          p_payload_vec_ad_3,
          p_sobre_cose_sign1,
          p_evidencia_verificacion,
          p_raiz_publica_spki,
          p_alta_canonica,
          p_sellos_hmac_canonicos
      );
EXCEPTION
    WHEN invalid_text_representation OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada O2-06 inválida';
END
$funcion$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.candidatura_alta_tecnica,
    vec_contratacion_temporal.alias_ambito_candidatura_alta,
    vec_contratacion_temporal.alias_huella_candidatura_alta
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, text
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    )
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    )
    FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, text
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    )
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
