-- Canon, preimagen y comprobacion MAC privadas de la capacidad de fuente V1.
-- La entrada conserva json (no jsonb) para cerrar orden, repeticiones y numeros.

CREATE FUNCTION
vec_autorizacion_atestada_v3.capacidad_fuente_corporativa_v1_canonica(
    p_capacidad pg_catalog.bytea
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_documento pg_catalog.json;
    v_claves pg_catalog.text[];
    v_tipos pg_catalog.text[];
    v_canon pg_catalog.bytea;
    v_claves_esperadas constant pg_catalog.text[] := ARRAY[
        'esquema', 'version', 'fuente_ref', 'fuente_version',
        'evento_fuente_ref', 'huella_evento_fuente_sha256',
        'evento_fuente_emitido_en', 'huella_manifiesto_fuente_sha256',
        'huella_sobre_cose_sign1_sha256',
        'huella_prueba_confianza_sha256', 'audiencia_consumo', 'accion',
        'tipo_efecto', 'operacion_ref', 'efecto_ref',
        'huella_efecto_sha256', 'clave_id', 'clave_version',
        'revision_gobierno', 'huella_gobierno_sha256', 'emisor_id',
        'configuracion_revision', 'configuracion_secuencia',
        'huella_configuracion_sha256', 'raiz_clave_id', 'raiz_version',
        'huella_raiz_spki_sha256', 'audiencia_despliegue', 'suite',
        'nonce', 'emitida_en', 'expira_en', 'mac_sha256'
    ];
    v_tipos_esperados constant pg_catalog.text[] := ARRAY[
        'string', 'number', 'string', 'number', 'string', 'string',
        'string', 'string', 'string', 'string', 'string', 'string',
        'string', 'string', 'string', 'string', 'string', 'number',
        'number', 'string', 'string', 'string', 'number', 'string',
        'string', 'number', 'string', 'string', 'string', 'string',
        'string', 'string', 'string'
    ];
BEGIN
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_bytes_validos(p_capacidad) IS NOT TRUE THEN
        RETURN NULL;
    END IF;

    v_documento := pg_catalog.convert_from(
        p_capacidad, 'UTF8'
    )::pg_catalog.json;
    IF pg_catalog.json_typeof(v_documento) IS DISTINCT FROM 'object' THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.array_agg(e.clave ORDER BY e.orden),
           pg_catalog.array_agg(
               pg_catalog.json_typeof(e.valor) ORDER BY e.orden
           )
      INTO v_claves, v_tipos
      FROM pg_catalog.json_each(v_documento)
           WITH ORDINALITY AS e(clave, valor, orden);

    IF v_claves IS DISTINCT FROM v_claves_esperadas
       OR v_tipos IS DISTINCT FROM v_tipos_esperados
       OR v_documento ->> 'esquema' IS DISTINCT FROM
          'vec.contexto-actor.fuente-corporativa.capacidad.v1'
       OR (v_documento -> 'version')::pg_catalog.text IS DISTINCT FROM '1'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 'version', 'fuente_version', 'clave_version',
                 'revision_gobierno', 'configuracion_secuencia',
                 'raiz_version'
             ]) AS n(clave)
            WHERE vec_autorizacion_atestada_v3
                     .entero_json_seguro_fuente_valido(
                         v_documento -> n.clave
                     ) IS NOT TRUE
       )
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_documento ->> 'fuente_ref'
              ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_documento ->> 'evento_fuente_ref'
              ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_documento ->> 'efecto_ref'
              ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .operacion_ref_fuente_corporativa_valida(
                  v_documento ->> 'operacion_ref'
              ) IS NOT TRUE
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 'huella_evento_fuente_sha256',
                 'huella_manifiesto_fuente_sha256',
                 'huella_sobre_cose_sign1_sha256',
                 'huella_prueba_confianza_sha256',
                 'huella_efecto_sha256', 'huella_gobierno_sha256',
                 'huella_configuracion_sha256',
                 'huella_raiz_spki_sha256', 'nonce', 'mac_sha256'
             ]) AS h(clave)
            WHERE vec_autorizacion_atestada_v3.huella_sha256_valida(
                      v_documento ->> h.clave
                  ) IS NOT TRUE
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 'evento_fuente_emitido_en', 'emitida_en', 'expira_en'
             ]) AS i(clave)
            WHERE vec_autorizacion_atestada_v3
                     .instante_utc_fuente_texto_valido(
                         v_documento ->> i.clave
                     ) IS NOT TRUE
       )
       OR v_documento ->> 'suite' IS DISTINCT FROM
          'VEC-AD-3-COSE-EDDSA-1' THEN
        RETURN NULL;
    END IF;

    IF (CASE v_documento ->> 'audiencia_consumo'
        WHEN 'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
        THEN (v_documento ->> 'accion', v_documento ->> 'tipo_efecto') =
             ('contexto_actor.organizacion_corporativa.publicar',
              'organizacion_corporativa.alta')
        WHEN 'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1'
        THEN (v_documento ->> 'accion', v_documento ->> 'tipo_efecto') =
             ('contexto_actor.organizacion_corporativa.revocar',
              'organizacion_corporativa.revocacion')
        WHEN 'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1'
        THEN (v_documento ->> 'accion', v_documento ->> 'tipo_efecto') =
             ('contexto_actor.vinculo_corporativo.publicar',
              'vinculo_corporativo.alta')
        WHEN 'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        THEN (v_documento ->> 'accion', v_documento ->> 'tipo_efecto') =
             ('contexto_actor.vinculo_corporativo.revocar',
              'vinculo_corporativo.revocacion')
        ELSE false
        END) IS NOT TRUE
       OR (v_documento ->> 'evento_fuente_emitido_en')::pg_catalog.timestamptz
          > (v_documento ->> 'emitida_en')::pg_catalog.timestamptz
       OR (v_documento ->> 'expira_en')::pg_catalog.timestamptz
          <= (v_documento ->> 'emitida_en')::pg_catalog.timestamptz
       OR (v_documento ->> 'expira_en')::pg_catalog.timestamptz
          > (v_documento ->> 'emitida_en')::pg_catalog.timestamptz
            + pg_catalog.make_interval(secs => 5) THEN
        RETURN NULL;
    END IF;

    v_canon := pg_catalog.convert_to(
        '{"esquema":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'esquema') ||
        ',"version":' || (v_documento -> 'version')::pg_catalog.text ||
        ',"fuente_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'fuente_ref') ||
        ',"fuente_version":' ||
            (v_documento -> 'fuente_version')::pg_catalog.text ||
        ',"evento_fuente_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'evento_fuente_ref') ||
        ',"huella_evento_fuente_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_evento_fuente_sha256') ||
        ',"evento_fuente_emitido_en":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'evento_fuente_emitido_en') ||
        ',"huella_manifiesto_fuente_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_manifiesto_fuente_sha256') ||
        ',"huella_sobre_cose_sign1_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_sobre_cose_sign1_sha256') ||
        ',"huella_prueba_confianza_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_prueba_confianza_sha256') ||
        ',"audiencia_consumo":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'audiencia_consumo') ||
        ',"accion":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'accion') ||
        ',"tipo_efecto":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'tipo_efecto') ||
        ',"operacion_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'operacion_ref') ||
        ',"efecto_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'efecto_ref') ||
        ',"huella_efecto_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_efecto_sha256') ||
        ',"clave_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'clave_id') ||
        ',"clave_version":' ||
            (v_documento -> 'clave_version')::pg_catalog.text ||
        ',"revision_gobierno":' ||
            (v_documento -> 'revision_gobierno')::pg_catalog.text ||
        ',"huella_gobierno_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_gobierno_sha256') ||
        ',"emisor_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'emisor_id') ||
        ',"configuracion_revision":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'configuracion_revision') ||
        ',"configuracion_secuencia":' ||
            (v_documento -> 'configuracion_secuencia')::pg_catalog.text ||
        ',"huella_configuracion_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_configuracion_sha256') ||
        ',"raiz_clave_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'raiz_clave_id') ||
        ',"raiz_version":' ||
            (v_documento -> 'raiz_version')::pg_catalog.text ||
        ',"huella_raiz_spki_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_raiz_spki_sha256') ||
        ',"audiencia_despliegue":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'audiencia_despliegue') ||
        ',"suite":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'suite') ||
        ',"nonce":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'nonce') ||
        ',"emitida_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'emitida_en') ||
        ',"expira_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'expira_en') ||
        ',"mac_sha256":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_documento ->> 'mac_sha256') || '}',
        'UTF8'
    );

    IF vec_autorizacion_atestada_v3.bytea_igual_constante(
           v_canon, p_capacidad
       ) IS NOT TRUE THEN
        RETURN NULL;
    END IF;
    RETURN v_canon;
EXCEPTION
    WHEN data_exception THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.preimagen_mac_fuente_corporativa_v1(
    p_capacidad pg_catalog.bytea
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_documento pg_catalog.json;
    v_preimagen pg_catalog.bytea;
BEGIN
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(p_capacidad)
       IS DISTINCT FROM p_capacidad THEN
        RETURN NULL;
    END IF;
    v_documento := pg_catalog.convert_from(
        p_capacidad, 'UTF8'
    )::pg_catalog.json;
    SELECT pg_catalog.convert_to(
               'VEC-CONTEXTO-ACTOR-FUENTE-CORPORATIVA-V1', 'UTF8'
           ) || pg_catalog.string_agg(
               vec_autorizacion_atestada_v3.encuadrar_mac(v.valor),
               ''::pg_catalog.bytea ORDER BY v.orden
           )
      INTO v_preimagen
      FROM (VALUES
        (1, v_documento ->> 'esquema'),
        (2, v_documento ->> 'version'),
        (3, v_documento ->> 'fuente_ref'),
        (4, v_documento ->> 'fuente_version'),
        (5, v_documento ->> 'evento_fuente_ref'),
        (6, v_documento ->> 'huella_evento_fuente_sha256'),
        (7, v_documento ->> 'evento_fuente_emitido_en'),
        (8, v_documento ->> 'huella_manifiesto_fuente_sha256'),
        (9, v_documento ->> 'huella_sobre_cose_sign1_sha256'),
        (10, v_documento ->> 'huella_prueba_confianza_sha256'),
        (11, v_documento ->> 'audiencia_consumo'),
        (12, v_documento ->> 'accion'),
        (13, v_documento ->> 'tipo_efecto'),
        (14, v_documento ->> 'operacion_ref'),
        (15, v_documento ->> 'efecto_ref'),
        (16, v_documento ->> 'huella_efecto_sha256'),
        (17, v_documento ->> 'clave_id'),
        (18, v_documento ->> 'clave_version'),
        (19, v_documento ->> 'revision_gobierno'),
        (20, v_documento ->> 'huella_gobierno_sha256'),
        (21, v_documento ->> 'emisor_id'),
        (22, v_documento ->> 'configuracion_revision'),
        (23, v_documento ->> 'configuracion_secuencia'),
        (24, v_documento ->> 'huella_configuracion_sha256'),
        (25, v_documento ->> 'raiz_clave_id'),
        (26, v_documento ->> 'raiz_version'),
        (27, v_documento ->> 'huella_raiz_spki_sha256'),
        (28, v_documento ->> 'audiencia_despliegue'),
        (29, v_documento ->> 'suite'),
        (30, v_documento ->> 'nonce'),
        (31, v_documento ->> 'emitida_en'),
        (32, v_documento ->> 'expira_en')
      ) AS v(orden, valor);
    RETURN v_preimagen;
EXCEPTION
    WHEN data_exception THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.mac_capacidad_fuente_corporativa_v1_valido(
    p_capacidad pg_catalog.bytea,
    p_secreto_hmac pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_documento pg_catalog.json;
    v_preimagen pg_catalog.bytea;
BEGIN
    IF vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
           p_secreto_hmac, 32, 4096
       ) IS NOT TRUE
       OR p_secreto_hmac = pg_catalog.decode(
           pg_catalog.repeat(
               '00', pg_catalog.octet_length(p_secreto_hmac)
           ),
           'hex'
       ) THEN
        RETURN false;
    END IF;
    v_preimagen := vec_autorizacion_atestada_v3
        .preimagen_mac_fuente_corporativa_v1(p_capacidad);
    IF v_preimagen IS NULL THEN
        RETURN false;
    END IF;
    v_documento := pg_catalog.convert_from(
        p_capacidad, 'UTF8'
    )::pg_catalog.json;
    RETURN vec_autorizacion_atestada_v3.bytea_igual_constante(
        public.hmac(v_preimagen, p_secreto_hmac, 'sha256'),
        pg_catalog.decode(v_documento ->> 'mac_sha256', 'hex')
    ) IS TRUE;
EXCEPTION
    WHEN data_exception THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .capacidad_fuente_corporativa_v1_canonica(pg_catalog.bytea),
    vec_autorizacion_atestada_v3
        .preimagen_mac_fuente_corporativa_v1(pg_catalog.bytea),
    vec_autorizacion_atestada_v3
        .mac_capacidad_fuente_corporativa_v1_valido(
            pg_catalog.bytea, pg_catalog.bytea
        )
    FROM PUBLIC;
