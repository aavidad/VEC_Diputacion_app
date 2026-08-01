-- Canon y huella privados del consumo de fuente corporativa V1.
-- El llamante no aporta capacidad_ref, canon de consumo ni su huella.

CREATE FUNCTION
vec_autorizacion_atestada_v3.canon_y_huella_consumo_fuente_corporativa_v1(
    p_capacidad_canonica pg_catalog.bytea,
    p_consumida_en pg_catalog.timestamptz
)
RETURNS TABLE (
    consumo_canonico pg_catalog.bytea,
    consumo_huella_sha256 pg_catalog.text
)
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_capacidad pg_catalog.json;
    v_claves pg_catalog.text[];
    v_tipos pg_catalog.text[];
    v_capacidad_reconstruida pg_catalog.bytea;
    v_capacidad_ref pg_catalog.text;
    v_consumida_texto pg_catalog.text;
    v_emitida pg_catalog.timestamptz;
    v_expira pg_catalog.timestamptz;
    v_evento_emitido pg_catalog.timestamptz;
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
    -- A1 limita y valida UTF-8 antes de convertir, recorrer o calcular huellas.
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_bytes_validos(p_capacidad_canonica) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .instante_fuente_finito_valido(p_consumida_en) IS NOT TRUE THEN
        RETURN;
    END IF;

    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::pg_catalog.json;
    IF pg_catalog.json_typeof(v_capacidad) IS DISTINCT FROM 'object' THEN
        RETURN;
    END IF;

    SELECT pg_catalog.array_agg(e.clave ORDER BY e.orden),
           pg_catalog.array_agg(
               pg_catalog.json_typeof(e.valor) ORDER BY e.orden
           )
      INTO v_claves, v_tipos
      FROM pg_catalog.json_each(v_capacidad)
           WITH ORDINALITY AS e(clave, valor, orden);

    IF v_claves IS DISTINCT FROM v_claves_esperadas
       OR v_tipos IS DISTINCT FROM v_tipos_esperados
       OR v_capacidad ->> 'esquema' IS DISTINCT FROM
          'vec.contexto-actor.fuente-corporativa.capacidad.v1'
       OR (v_capacidad -> 'version')::pg_catalog.text IS DISTINCT FROM '1'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 v_capacidad -> 'version',
                 v_capacidad -> 'fuente_version',
                 v_capacidad -> 'clave_version',
                 v_capacidad -> 'revision_gobierno',
                 v_capacidad -> 'configuracion_secuencia',
                 v_capacidad -> 'raiz_version'
             ]) AS n(valor)
            WHERE vec_autorizacion_atestada_v3
                      .entero_json_seguro_fuente_valido(n.valor) IS NOT TRUE
       )
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_capacidad ->> 'fuente_ref') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_capacidad ->> 'evento_fuente_ref') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .referencia_opaca_fuente_corporativa_valida(
                  v_capacidad ->> 'efecto_ref') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .operacion_ref_fuente_corporativa_valida(
                  v_capacidad ->> 'operacion_ref') IS NOT TRUE
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 v_capacidad ->> 'huella_evento_fuente_sha256',
                 v_capacidad ->> 'huella_manifiesto_fuente_sha256',
                 v_capacidad ->> 'huella_sobre_cose_sign1_sha256',
                 v_capacidad ->> 'huella_prueba_confianza_sha256',
                 v_capacidad ->> 'huella_efecto_sha256',
                 v_capacidad ->> 'huella_gobierno_sha256',
                 v_capacidad ->> 'huella_configuracion_sha256',
                 v_capacidad ->> 'huella_raiz_spki_sha256',
                 v_capacidad ->> 'nonce',
                 v_capacidad ->> 'mac_sha256'
             ]) AS h(valor)
            WHERE vec_autorizacion_atestada_v3
                      .huella_sha256_valida(h.valor) IS NOT TRUE
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 v_capacidad ->> 'clave_id',
                 v_capacidad ->> 'emisor_id',
                 v_capacidad ->> 'configuracion_revision',
                 v_capacidad ->> 'raiz_clave_id',
                 v_capacidad ->> 'audiencia_despliegue'
             ]) AS t(valor)
            WHERE vec_autorizacion_atestada_v3
                      .texto_tecnico_valido(t.valor, 512) IS NOT TRUE
       )
       OR v_capacidad ->> 'suite' IS DISTINCT FROM
          'VEC-AD-3-COSE-EDDSA-1'
       OR vec_autorizacion_atestada_v3
              .instante_utc_fuente_texto_valido(
                  v_capacidad ->> 'evento_fuente_emitido_en') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .instante_utc_fuente_texto_valido(
                  v_capacidad ->> 'emitida_en') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
              .instante_utc_fuente_texto_valido(
                  v_capacidad ->> 'expira_en') IS NOT TRUE THEN
        RETURN;
    END IF;

    IF (CASE v_capacidad ->> 'audiencia_consumo'
        WHEN 'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
        THEN (v_capacidad ->> 'accion', v_capacidad ->> 'tipo_efecto') =
             ('contexto_actor.organizacion_corporativa.publicar',
              'organizacion_corporativa.alta')
        WHEN 'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1'
        THEN (v_capacidad ->> 'accion', v_capacidad ->> 'tipo_efecto') =
             ('contexto_actor.organizacion_corporativa.revocar',
              'organizacion_corporativa.revocacion')
        WHEN 'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1'
        THEN (v_capacidad ->> 'accion', v_capacidad ->> 'tipo_efecto') =
             ('contexto_actor.vinculo_corporativo.publicar',
              'vinculo_corporativo.alta')
        WHEN 'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        THEN (v_capacidad ->> 'accion', v_capacidad ->> 'tipo_efecto') =
             ('contexto_actor.vinculo_corporativo.revocar',
              'vinculo_corporativo.revocacion')
        ELSE false
        END) IS NOT TRUE THEN
        RETURN;
    END IF;

    v_evento_emitido := (v_capacidad ->> 'evento_fuente_emitido_en')
        ::pg_catalog.timestamptz;
    v_emitida := (v_capacidad ->> 'emitida_en')::pg_catalog.timestamptz;
    v_expira := (v_capacidad ->> 'expira_en')::pg_catalog.timestamptz;
    IF v_evento_emitido > v_emitida
       OR v_expira <= v_emitida
       OR v_expira > v_emitida + pg_catalog.make_interval(secs => 5)
       OR p_consumida_en < v_emitida
       OR p_consumida_en >= v_expira THEN
        RETURN;
    END IF;

    -- Reconstruir también la capacidad impide que escapes, orden o números
    -- equivalentes pero no canónicos alteren la referencia que se deriva.
    v_capacidad_reconstruida := pg_catalog.convert_to(
        '{"esquema":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'esquema') ||
        ',"version":' || (v_capacidad -> 'version')::pg_catalog.text ||
        ',"fuente_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'fuente_ref') ||
        ',"fuente_version":' ||
            (v_capacidad -> 'fuente_version')::pg_catalog.text ||
        ',"evento_fuente_ref":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'evento_fuente_ref') ||
        ',"huella_evento_fuente_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_evento_fuente_sha256') ||
        ',"evento_fuente_emitido_en":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'evento_fuente_emitido_en') ||
        ',"huella_manifiesto_fuente_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_manifiesto_fuente_sha256') ||
        ',"huella_sobre_cose_sign1_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_sobre_cose_sign1_sha256') ||
        ',"huella_prueba_confianza_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_prueba_confianza_sha256') ||
        ',"audiencia_consumo":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'audiencia_consumo') ||
        ',"accion":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'accion') ||
        ',"tipo_efecto":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'tipo_efecto') ||
        ',"operacion_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'operacion_ref') ||
        ',"efecto_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'efecto_ref') ||
        ',"huella_efecto_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_efecto_sha256') ||
        ',"clave_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'clave_id') ||
        ',"clave_version":' ||
            (v_capacidad -> 'clave_version')::pg_catalog.text ||
        ',"revision_gobierno":' ||
            (v_capacidad -> 'revision_gobierno')::pg_catalog.text ||
        ',"huella_gobierno_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_gobierno_sha256') ||
        ',"emisor_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'emisor_id') ||
        ',"configuracion_revision":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'configuracion_revision') ||
        ',"configuracion_secuencia":' ||
            (v_capacidad -> 'configuracion_secuencia')::pg_catalog.text ||
        ',"huella_configuracion_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_configuracion_sha256') ||
        ',"raiz_clave_id":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'raiz_clave_id') ||
        ',"raiz_version":' ||
            (v_capacidad -> 'raiz_version')::pg_catalog.text ||
        ',"huella_raiz_spki_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'huella_raiz_spki_sha256') ||
        ',"audiencia_despliegue":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_capacidad ->> 'audiencia_despliegue') ||
        ',"suite":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'suite') ||
        ',"nonce":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'nonce') ||
        ',"emitida_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'emitida_en') ||
        ',"expira_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'expira_en') ||
        ',"mac_sha256":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_capacidad ->> 'mac_sha256') || '}',
        'UTF8'
    );
    IF vec_autorizacion_atestada_v3.bytea_igual_constante(
           v_capacidad_reconstruida, p_capacidad_canonica
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    v_capacidad_ref := 'cfc_' || pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex');
    v_consumida_texto := vec_autorizacion_atestada_v3
        .representacion_instante_utc_fuente(p_consumida_en);
    consumo_canonico := pg_catalog.convert_to(
        '{"esquema":"vec.contexto-actor.fuente-corporativa.consumo.v1"' ||
        ',"version":1,"capacidad_ref":' ||
            vec_autorizacion_atestada_v3.texto_json_go(v_capacidad_ref) ||
        pg_catalog.left(
            pg_catalog.substr(
                pg_catalog.convert_from(v_capacidad_reconstruida, 'UTF8'),
                pg_catalog.strpos(
                    pg_catalog.convert_from(
                        v_capacidad_reconstruida, 'UTF8'),
                    ',"fuente_ref":'
                )
            ),
            pg_catalog.length(pg_catalog.substr(
                pg_catalog.convert_from(v_capacidad_reconstruida, 'UTF8'),
                pg_catalog.strpos(
                    pg_catalog.convert_from(
                        v_capacidad_reconstruida, 'UTF8'),
                    ',"fuente_ref":'
                )
            )) - 1
        ) ||
        ',"consumida_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            v_consumida_texto) || '}',
        'UTF8'
    );
    consumo_huella_sha256 := pg_catalog.encode(
        pg_catalog.sha256(consumo_canonico), 'hex');
    RETURN NEXT;
EXCEPTION
    WHEN invalid_text_representation OR
         character_not_in_repertoire OR
         untranslatable_character OR
         datetime_field_overflow OR
         numeric_value_out_of_range THEN
        RETURN;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .canon_y_huella_consumo_fuente_corporativa_v1(
            pg_catalog.bytea, pg_catalog.timestamptz
        ) FROM PUBLIC;
