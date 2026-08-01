-- Canon privado del manifiesto minimizado de fuente corporativa V1.
-- Un resultado no nulo acredita forma, semantica y bytes canonicos exactos.

CREATE FUNCTION
vec_autorizacion_atestada_v3.manifiesto_fuente_corporativa_v1_canonico(
    p_manifiesto pg_catalog.bytea
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_documento pg_catalog.json;
    v_claves text[];
    v_tipos text[];
    v_canon pg_catalog.bytea;
    v_claves_esperadas constant text[] := ARRAY[
        'esquema', 'version', 'fuente_ref', 'fuente_version',
        'evento_fuente_ref', 'huella_evento_fuente_sha256',
        'evento_fuente_emitido_en', 'audiencia_consumo', 'accion',
        'tipo_efecto', 'operacion_ref', 'efecto_ref',
        'huella_efecto_sha256'
    ];
    v_tipos_esperados constant text[] := ARRAY[
        'string', 'number', 'string', 'number', 'string', 'string',
        'string', 'string', 'string', 'string', 'string', 'string',
        'string'
    ];
BEGIN
    -- A1 aplica el limite y valida UTF-8 antes de convertir o interpretar.
    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_bytes_validos(p_manifiesto) IS NOT TRUE THEN
        RETURN NULL;
    END IF;

    v_documento := pg_catalog.convert_from(p_manifiesto, 'UTF8')::pg_catalog.json;
    IF pg_catalog.json_typeof(v_documento) IS DISTINCT FROM 'object' THEN
        RETURN NULL;
    END IF;

    -- json conserva orden y repeticiones: la matriz cierra claves sobrantes,
    -- ausentes, duplicadas o reordenadas antes de extraer sus valores.
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
          'vec.contexto-actor.fuente-corporativa.manifiesto.v1'
       OR (v_documento -> 'version')::pg_catalog.text IS DISTINCT FROM '1'
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              v_documento -> 'version'
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              v_documento -> 'fuente_version'
          ) IS NOT TRUE
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
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              v_documento ->> 'huella_evento_fuente_sha256'
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              v_documento ->> 'huella_efecto_sha256'
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              v_documento ->> 'evento_fuente_emitido_en'
          ) IS NOT TRUE THEN
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
        END) IS NOT TRUE THEN
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
        ',"evento_fuente_ref":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'evento_fuente_ref') ||
        ',"huella_evento_fuente_sha256":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'huella_evento_fuente_sha256') ||
        ',"evento_fuente_emitido_en":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                v_documento ->> 'evento_fuente_emitido_en') ||
        ',"audiencia_consumo":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
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
                v_documento ->> 'huella_efecto_sha256') || '}',
        'UTF8'
    );

    IF vec_autorizacion_atestada_v3.bytea_igual_constante(
           v_canon, p_manifiesto
       ) IS NOT TRUE THEN
        RETURN NULL;
    END IF;
    RETURN v_canon;
EXCEPTION
    WHEN invalid_text_representation OR
         character_not_in_repertoire OR
         untranslatable_character THEN
        RETURN NULL;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    FROM PUBLIC;
