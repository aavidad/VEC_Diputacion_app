CREATE FUNCTION vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
    p_ambitos_hmac text[], p_huellas_hmac text[],
    p_organizacion_ref text, p_actor_ref text, p_perfil_ref text,
    p_reserva_ref_propuesta text, p_expediente_ref_propuesto text,
    p_numero_visible_propuesto text, p_recibo_ref_propuesto text,
    p_instante_efecto_propuesto timestamptz
)
RETURNS TABLE (
    resultado text, reserva_ref text, expediente_ref text,
    numero_visible text, recibo_ref text, ambito_hmac text,
    huella_peticion_hmac text, organizacion_ref text, actor_ref text,
    perfil_ref text, instante_efecto timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_generaciones integer[];
    v_politica integer[];
    v_raices text[];
    v_raiz text;
    v_insertada boolean := false;
    v_candidatura vec_contratacion_temporal.candidatura_alta_tecnica%ROWTYPE;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user,
           'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
           'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR p_ambitos_hmac IS NULL
       OR p_huellas_hmac IS NULL
       OR pg_catalog.array_ndims(p_ambitos_hmac) IS DISTINCT FROM 1
       OR pg_catalog.array_ndims(p_huellas_hmac) IS DISTINCT FROM 1
       OR pg_catalog.array_lower(p_ambitos_hmac, 1) IS DISTINCT FROM 1
       OR pg_catalog.array_lower(p_huellas_hmac, 1) IS DISTINCT FROM 1
       OR pg_catalog.cardinality(p_ambitos_hmac) NOT BETWEEN 1 AND 4
       OR pg_catalog.cardinality(p_huellas_hmac) IS DISTINCT FROM
          pg_catalog.cardinality(p_ambitos_hmac)
       OR p_organizacion_ref IS NULL
       OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_actor_ref IS NULL
       OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_perfil_ref IS NULL
       OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_reserva_ref_propuesta IS NULL
       OR p_reserva_ref_propuesta !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref_propuesto IS NULL
       OR p_expediente_ref_propuesto !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_numero_visible_propuesto IS NULL
       OR p_numero_visible_propuesto !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR p_recibo_ref_propuesto IS NULL
       OR p_recibo_ref_propuesto !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_instante_efecto_propuesto IS NULL
       OR p_instante_efecto_propuesto IS DISTINCT FROM
          pg_catalog.date_trunc('microseconds', p_instante_efecto_propuesto) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'candidatura de alta invalida';
    END IF;
    IF pg_catalog.array_position(p_ambitos_hmac, NULL) IS NOT NULL
       OR pg_catalog.array_position(p_huellas_hmac, NULL) IS NOT NULL
       OR EXISTS (
        SELECT 1
          FROM ROWS FROM (
                   pg_catalog.unnest(p_ambitos_hmac),
                   pg_catalog.unnest(p_huellas_hmac)
               ) WITH ORDINALITY AS p(ambito, huella, orden)
         WHERE p.ambito IS NULL
            OR p.huella IS NULL
            OR p.ambito !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
                   'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR p.huella !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
                   'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR pg_catalog.right(p.ambito, 64) = pg_catalog.repeat('0', 64)
            OR pg_catalog.right(p.huella, 64) = pg_catalog.repeat('0', 64)
            OR substring(p.ambito FROM '/v([1-9][0-9]{0,8}):') IS DISTINCT FROM
               substring(p.huella FROM '/v([1-9][0-9]{0,8}):')
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = CASE
                WHEN pg_catalog.array_position(p_ambitos_hmac, NULL) IS NOT NULL
                  OR pg_catalog.array_position(p_huellas_hmac, NULL) IS NOT NULL
                THEN 'candidatura de alta invalida'
                ELSE 'pares HMAC invalidos'
            END;
    END IF;
    SELECT pg_catalog.array_agg(substring(ambito FROM
               '/v([1-9][0-9]{0,8}):')::integer ORDER BY orden)
      INTO v_generaciones
      FROM pg_catalog.unnest(p_ambitos_hmac) WITH ORDINALITY AS p(ambito, orden);
    SELECT pg_catalog.array_agg(generacion ORDER BY posicion)
      INTO v_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_politica OR
       pg_catalog.cardinality(v_generaciones) <>
       pg_catalog.cardinality(ARRAY(
           SELECT DISTINCT x FROM pg_catalog.unnest(v_generaciones) AS u(x)
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'politica HMAC no satisfecha';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:candidatura:' || valor, 0))
      FROM pg_catalog.unnest(p_ambitos_hmac || p_huellas_hmac) AS u(valor)
     ORDER BY valor COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT a.ambito_raiz_hmac
               ORDER BY a.ambito_raiz_hmac
           )
      INTO v_raices
      FROM vec_contratacion_temporal.candidatura_alta_alias AS a
     WHERE a.ambito_hmac = ANY(p_ambitos_hmac)
        OR a.huella_hmac = ANY(p_huellas_hmac);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'candidatura de alta en conflicto';
    ELSIF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        v_raiz := p_ambitos_hmac[1];
        INSERT INTO vec_contratacion_temporal.candidatura_alta_tecnica (
            ambito_raiz_hmac, huella_raiz_hmac, reserva_ref, expediente_ref,
            numero_visible, recibo_ref, organizacion_ref, actor_ref, perfil_ref,
            instante_efecto, origen, creada_en
        ) VALUES (
            v_raiz, p_huellas_hmac[1], p_reserva_ref_propuesta,
            p_expediente_ref_propuesto, p_numero_visible_propuesto,
            p_recibo_ref_propuesto, p_organizacion_ref, p_actor_ref,
            p_perfil_ref, p_instante_efecto_propuesto, 'resolucion',
            pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
        );
        v_insertada := true;
    END IF;
    SELECT * INTO STRICT v_candidatura
      FROM vec_contratacion_temporal.candidatura_alta_tecnica
     WHERE ambito_raiz_hmac = v_raiz;
    IF v_candidatura.organizacion_ref IS DISTINCT FROM p_organizacion_ref
       OR v_candidatura.actor_ref IS DISTINCT FROM p_actor_ref
       OR v_candidatura.perfil_ref IS DISTINCT FROM p_perfil_ref THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'candidatura de alta en conflicto';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM ROWS FROM (
                   pg_catalog.unnest(v_generaciones),
                   pg_catalog.unnest(p_ambitos_hmac),
                   pg_catalog.unnest(p_huellas_hmac)
               ) AS p(generacion, ambito, huella)
          JOIN vec_contratacion_temporal.candidatura_alta_alias a
            ON a.ambito_raiz_hmac = v_raiz
           AND (a.generacion = p.generacion OR a.ambito_hmac = p.ambito
                OR a.huella_hmac = p.huella)
         WHERE a.generacion IS DISTINCT FROM p.generacion
            OR a.ambito_hmac IS DISTINCT FROM p.ambito
            OR a.huella_hmac IS DISTINCT FROM p.huella
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'par HMAC de candidatura en conflicto';
    END IF;
    INSERT INTO vec_contratacion_temporal.candidatura_alta_alias (
        ambito_hmac, huella_hmac, ambito_raiz_hmac, generacion, registrada_en
    )
    SELECT ambito, huella, v_raiz, generacion,
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
      FROM ROWS FROM (
               pg_catalog.unnest(v_generaciones),
               pg_catalog.unnest(p_ambitos_hmac),
               pg_catalog.unnest(p_huellas_hmac)
           ) AS p(generacion, ambito, huella)
    ON CONFLICT ON CONSTRAINT candidatura_alta_alias_pkey DO NOTHING;
    RETURN QUERY SELECT
        CASE WHEN v_insertada THEN 'estabilizada' ELSE 'recuperada' END,
        v_candidatura.reserva_ref, v_candidatura.expediente_ref,
        v_candidatura.numero_visible, v_candidatura.recibo_ref,
        v_candidatura.ambito_raiz_hmac, v_candidatura.huella_raiz_hmac,
        v_candidatura.organizacion_ref, v_candidatura.actor_ref,
        v_candidatura.perfil_ref, v_candidatura.instante_efecto;
END
$funcion$;
