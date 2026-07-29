-- CT-000044: materialización privada y minimizada del cuadro RRHH.
--
-- El corte se aplica antes de cualquier ámbito o filtro. Así, un expediente
-- cuya última versión ya no coincide nunca reaparece por una versión antigua.
-- La colección de límite+1 es la única lectura que alimenta salida y canon.

CREATE FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
)
RETURNS vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_candidatas
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[];
    v_publicadas
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1[];
    v_total integer;
    v_total_publicado integer;
    v_hay_mas boolean;
    v_ultima
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_estado IS NULL
       OR p_estado.es_continuacion IS NULL
       OR p_estado.corte_global IS NULL
       OR p_estado.corte_global NOT BETWEEN
          0 AND 9007199254740991::numeric
       OR p_estado.corte_global <>
          pg_catalog.trunc(p_estado.corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'materialización de cuadro RRHH inválida';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );

    IF NOT p_estado.es_continuacion AND (
           p_consulta.cursor <> ''
           OR p_estado.pagina_presentada IS DISTINCT FROM 0
           OR p_estado.familia_ref IS NOT NULL
           OR p_estado.token_presentado_huella_sha256 IS NOT NULL
           OR p_estado.acceso_emision_ref IS NOT NULL
           OR p_estado.cursor_emitida_en IS NOT NULL
           OR p_estado.familia_creada_en IS NOT NULL
           OR p_estado.familia_valida_hasta IS NOT NULL
           OR p_estado.ultimo_actualizado_en IS NOT NULL
           OR p_estado.ultimo_expediente_ref IS NOT NULL
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'estado inicial de cuadro RRHH inválido';
    END IF;

    IF p_estado.es_continuacion AND (
           p_estado.corte_global < 1
           OR p_consulta.cursor = ''
           OR p_estado.familia_ref IS NULL
           OR p_estado.familia_ref !~
              '^familia:cursor:rrhh:[0-9a-f]{32}$'
           OR p_estado.pagina_presentada IS NULL
           OR p_estado.pagina_presentada NOT BETWEEN
              2 AND 9007199254740991::numeric
           OR p_estado.pagina_presentada <>
              pg_catalog.trunc(p_estado.pagina_presentada)
           OR p_estado.token_presentado_huella_sha256 IS NULL
           OR p_estado.token_presentado_huella_sha256 !~
              '^[0-9a-f]{64}$'
           OR p_estado.token_presentado_huella_sha256 =
              pg_catalog.repeat('0', 64)
           OR p_estado.acceso_emision_ref IS NULL
           OR p_estado.acceso_emision_ref !~
              '^acceso:rrhh:[0-9a-f]{32}$'
           OR p_estado.cursor_emitida_en IS NULL
           OR p_estado.familia_creada_en IS NULL
           OR p_estado.familia_valida_hasta IS NULL
           OR p_estado.ultimo_actualizado_en IS NULL
           OR p_estado.ultimo_expediente_ref IS NULL
           OR p_estado.ultimo_expediente_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR p_estado.cursor_emitida_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.cursor_emitida_en
              )
           OR p_estado.familia_creada_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.familia_creada_en
              )
           OR p_estado.familia_valida_hasta <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.familia_valida_hasta
              )
           OR p_estado.ultimo_actualizado_en <>
              pg_catalog.date_trunc(
                  'microseconds', p_estado.ultimo_actualizado_en
              )
           OR p_estado.familia_creada_en >
              p_estado.cursor_emitida_en
           OR p_estado.cursor_emitida_en >=
              p_estado.familia_valida_hasta
           OR p_estado.ultimo_actualizado_en >
              p_estado.cursor_emitida_en
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'continuación de cuadro RRHH inválida';
    END IF;

    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref,
               publicada.organizacion_ref,
               publicada.numero_visible,
               publicada.version,
               publicada.flujo_ref,
               publicada.flujo_version,
               publicada.flujo_huella_sha256,
               publicada.fase_clave,
               publicada.estado_clave,
               publicada.centro_ref,
               publicada.categoria_ref,
               publicada.modalidad_clave,
               publicada.unidad_ref,
               publicada.creado_en,
               publicada.actualizado_en
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= p_estado.corte_global
         ORDER BY publicada.expediente_ref COLLATE "C",
                  publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ROW(
                   ultima.expediente_ref,
                   ultima.organizacion_ref,
                   ultima.numero_visible,
                   ultima.version,
                   ultima.flujo_ref,
                   ultima.flujo_version,
                   ultima.flujo_huella_sha256,
                   ultima.fase_clave,
                   ultima.estado_clave,
                   ultima.centro_ref,
                   ultima.categoria_ref,
                   COALESCE(ultima.modalidad_clave, ''),
                   COALESCE(ultima.unidad_ref, ''),
                   ultima.creado_en,
                   ultima.actualizado_en
               )::vec_contratacion_temporal
                 .resumen_publicacion_rrhh_v1 AS resumen
          FROM ultimas ultima
         WHERE ultima.organizacion_ref COLLATE "C" =
               p_alcance.organizacion_ref COLLATE "C"
           AND (
               p_alcance.clase_ambito = 'organizacion'
               OR p_alcance.clase_ambito = 'centro'
                  AND ultima.centro_ref COLLATE "C" =
                      p_alcance.ambito_ref COLLATE "C"
               OR p_alcance.clase_ambito = 'unidad_gestion'
                  AND ultima.unidad_ref IS NOT NULL
                  AND ultima.unidad_ref COLLATE "C" =
                      p_alcance.ambito_ref COLLATE "C"
           )
           AND (
               p_consulta.texto = ''
               OR pg_catalog.left(
                      ultima.numero_visible,
                      pg_catalog.char_length(p_consulta.texto)
                  ) COLLATE "C" = p_consulta.texto COLLATE "C"
           )
           AND (
               p_consulta.estado_clave = ''
               OR ultima.estado_clave COLLATE "C" =
                  p_consulta.estado_clave COLLATE "C"
           )
           AND (
               p_consulta.fase_clave = ''
               OR ultima.fase_clave COLLATE "C" =
                  p_consulta.fase_clave COLLATE "C"
           )
           AND (
               NOT p_estado.es_continuacion
               OR ultima.actualizado_en <
                  p_estado.ultimo_actualizado_en
               OR ultima.actualizado_en =
                  p_estado.ultimo_actualizado_en
                  AND ultima.expediente_ref COLLATE "C" <
                      p_estado.ultimo_expediente_ref COLLATE "C"
           )
         ORDER BY ultima.actualizado_en DESC,
                  ultima.expediente_ref COLLATE "C" DESC
         LIMIT p_consulta.limite::integer + 1
    )
    SELECT COALESCE(
               pg_catalog.array_agg(
                   filtrada.resumen
                   ORDER BY (filtrada.resumen).actualizado_en DESC,
                            (filtrada.resumen).expediente_ref
                                COLLATE "C" DESC
               ),
               ARRAY[]::vec_contratacion_temporal
                 .resumen_publicacion_rrhh_v1[]
           )
      INTO STRICT v_candidatas
      FROM filtradas filtrada;

    v_total := pg_catalog.cardinality(v_candidatas);
    v_total_publicado := LEAST(
        v_total, p_consulta.limite::integer
    );
    v_hay_mas := v_total > p_consulta.limite::integer;

    IF v_total_publicado = 0 THEN
        v_publicadas := ARRAY[]::vec_contratacion_temporal
          .resumen_publicacion_rrhh_v1[];
        RETURN ROW(
            v_publicadas,
            false,
            NULL::timestamptz,
            NULL::text
        )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    END IF;

    v_publicadas := v_candidatas[1:v_total_publicado];
    v_ultima := v_publicadas[v_total_publicado];

    RETURN ROW(
        v_publicadas,
        v_hay_mas,
        v_ultima.actualizado_en,
        v_ultima.expediente_ref
    )::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range OR array_subscript_error THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'materialización de cuadro RRHH inválida';
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
)
OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

COMMENT ON FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
) IS
'Materializa una sola página minimizada según corte; no autoriza, persiste ni consume cursores.';
