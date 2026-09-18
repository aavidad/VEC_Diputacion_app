\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
-- CT-000106: detalle RRHH V3 con fiscalización minimizada y atestada.
-- Solo proyecta datos ya durables del agregado: resultado, observaciones,
-- instante y, si existe, la actuación de subsanación enlazada.
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regtype(
           'vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_attribute atributo
            WHERE atributo.attrelid =
                  'vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'
                  ::pg_catalog.regclass
              AND atributo.attname = ANY (ARRAY[
                  'fiscalizacion_presente', 'fiscalizacion',
                  'referencia_fiscalizacion', 'referencia_subsanacion'
              ]::pg_catalog.name[])
              AND atributo.attnum > 0
              AND NOT atributo.attisdropped
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para detalle RRHH V3';
    END IF;
END
$prevalidacion$;

CREATE TYPE vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1 AS (
    resultado_clave text,
    reparo_clave text,
    reparo_texto text,
    registrada_en timestamptz(6),
    subsanacion_registrada_en timestamptz(6),
    subsanacion_texto text
);

ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
    ADD ATTRIBUTE fiscalizacion_presente boolean;
ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
    ADD ATTRIBUTE fiscalizacion
        vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
    ADD ATTRIBUTE referencia_fiscalizacion numeric(20, 0);
ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
    ADD ATTRIBUTE referencia_subsanacion numeric(20, 0);

CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    p_corte_global numeric
)
RETURNS vec_contratacion_temporal.materializacion_detalle_rrhh_v1
LANGUAGE plpgsql
STABLE
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
    v_fila record;
    v_agregado jsonb;
    v_nodo jsonb;
    v_indice integer;
    v_total integer;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud
        vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis
        vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura
        vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion
        vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion
        vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_subsanacion_nodo jsonb;
    v_subsanaciones integer;
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_hitos
        vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_analisis_presente boolean;
    v_cobertura_presente boolean;
    v_asignacion_presente boolean;
    v_fiscalizacion_presente boolean;
    v_subsanacion_presente boolean;
    v_decision_gobernada boolean;
    v_coste_presente boolean;
    v_referencia_analisis numeric(20, 0) := 0;
    v_referencia_cobertura numeric(20, 0) := 0;
    v_referencia_asignacion numeric(20, 0) := 0;
    v_referencia_fiscalizacion numeric(20, 0) := 0;
    v_referencia_subsanacion numeric(20, 0) := 0;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        p_consulta
    );

    -- MATERIALIZED fija primero la versión actual relativa al corte. El
    -- predicado exterior aplica después el ámbito y el control de versión.
    WITH elegida AS MATERIALIZED (
        SELECT publicacion.*, historia.agregado_json
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicacion
          JOIN vec_contratacion_temporal.expediente_version_integral historia
            ON historia.expediente_ref = publicacion.expediente_ref
           AND historia.version = publicacion.version
           AND historia.flujo_ref = publicacion.flujo_ref
           AND historia.flujo_version = publicacion.flujo_version
           AND historia.flujo_huella_sha256 =
               publicacion.flujo_huella_sha256
           AND historia.fase_clave = publicacion.fase_clave
           AND historia.estado = publicacion.estado_clave
           AND historia.agregado_json_huella_sha256 =
               publicacion.agregado_huella_sha256
           AND historia.registrada_en = publicacion.registrada_en
         WHERE publicacion.expediente_ref = p_consulta.expediente_ref
           AND publicacion.corte_global <= p_corte_global
         ORDER BY publicacion.corte_global DESC
         LIMIT 1
    )
    SELECT elegida.*
      INTO STRICT v_fila
      FROM elegida
     WHERE elegida.organizacion_ref = p_alcance.organizacion_ref
       AND (
           p_consulta.version_observada = 0
           OR p_consulta.version_observada = elegida.version
       )
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN
               elegida.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN
               elegida.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN
               elegida.unidad_ref = p_alcance.ambito_ref
           ELSE false
       END;

    v_agregado := v_fila.agregado_json;
    IF pg_catalog.octet_length(v_agregado::text) > 262144
       OR NOT vec_contratacion_temporal.json_rrhh_seguro_v1(v_agregado)
       OR pg_catalog.jsonb_typeof(v_agregado -> 'solicitud') <> 'object'
       OR pg_catalog.jsonb_typeof(
           v_agregado #> '{solicitud,periodo}'
       ) <> 'object'
       OR pg_catalog.jsonb_typeof(v_agregado -> 'actuaciones') <> 'array'
       OR v_agregado ->> 'referencia' IS DISTINCT FROM
          v_fila.expediente_ref
       OR v_agregado ->> 'organizacion_ref' IS DISTINCT FROM
          v_fila.organizacion_ref
       OR v_agregado ->> 'numero_visible' IS DISTINCT FROM
          v_fila.numero_visible
       OR v_agregado ->> 'version' IS DISTINCT FROM
          v_fila.version::text
       OR v_agregado #>> '{flujo,definicion_ref}' IS DISTINCT FROM
          v_fila.flujo_ref
       OR v_agregado #>> '{flujo,version}' IS DISTINCT FROM
          v_fila.flujo_version::text
       OR v_agregado #>> '{flujo,huella_sha256}' IS DISTINCT FROM
          v_fila.flujo_huella_sha256
       OR v_agregado ->> 'fase_actual' IS DISTINCT FROM
          v_fila.fase_clave
       OR v_agregado ->> 'estado_actual' IS DISTINCT FROM
          v_fila.estado_clave THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        v_fila.numero_visible,
        v_fila.version,
        v_fila.flujo_ref,
        v_fila.flujo_version,
        v_fila.flujo_huella_sha256,
        v_fila.fase_clave,
        v_fila.estado_clave,
        v_fila.centro_ref,
        v_fila.categoria_ref,
        COALESCE(v_fila.modalidad_clave, ''),
        COALESCE(v_fila.unidad_ref, ''),
        v_fila.creado_en,
        v_fila.actualizado_en
    );
    v_solicitud := ROW(
        v_agregado #>> '{solicitud,grupo_subgrupo}',
        v_agregado #>> '{solicitud,motivo_clave}',
        (v_agregado #>> '{solicitud,periodo,inicio}')::timestamptz,
        (v_agregado #>> '{solicitud,periodo,fin}')::timestamptz
    );

    v_analisis_presente := v_agregado ? 'analisis';
    IF v_analisis_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'analisis') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,periodo}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,validacion_rc}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_coste_presente :=
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object';
        v_referencia_analisis := (
            v_agregado #>>
            '{analisis,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_analisis := ROW(
            v_agregado #>> '{analisis,modalidad_clave}',
            v_agregado #>> '{analisis,categoria_ref}',
            v_agregado #>> '{analisis,causa_clave}',
            (v_agregado #>> '{analisis,periodo,inicio}')::timestamptz,
            (v_agregado #>> '{analisis,periodo,fin}')::timestamptz,
            (v_agregado #>> '{analisis,porcentaje_jornada}')::smallint,
            v_agregado #>> '{analisis,validacion_rc,resultado}',
            v_coste_presente,
            CASE WHEN v_coste_presente THEN
                (v_agregado #>>
                 '{analisis,coste_previsto,centimos}')::bigint
            ELSE 0::bigint END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,coste_previsto,moneda}'
            ELSE '' END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,fuente_coste_ref}'
            ELSE '' END,
            COALESCE(v_agregado #>> '{analisis,observaciones}', '')
        );
    ELSE
        v_analisis := NULL;
    END IF;

    v_cobertura_presente := v_agregado ? 'via_cobertura';
    IF v_cobertura_presente THEN
        IF pg_catalog.jsonb_typeof(
               v_agregado -> 'via_cobertura'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_decision_gobernada :=
            pg_catalog.jsonb_typeof(
                v_agregado #>
                '{via_cobertura,decision_gobernada}'
            ) = 'object';
        v_comprobaciones := ARRAY[]::vec_contratacion_temporal
            .comprobacion_operativa_rrhh_v1[];
        IF v_decision_gobernada THEN
            IF NOT v_analisis_presente THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := (
                v_agregado #>>
                '{via_cobertura,decision_gobernada,actuacion,version_expediente}'
            )::numeric(20, 0);
            SELECT ARRAY(
                SELECT ROW(
                    evidencia.comprobacion_clave,
                    evidencia.comprobacion_resultado
                )::vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
                  FROM vec_contratacion_temporal.consumo_cobertura_lote lote
                  JOIN vec_contratacion_temporal.consumo_cobertura_evidencia evidencia
                    ON evidencia.lote_ref = lote.lote_ref
                 WHERE lote.preparacion_c1_ref = v_agregado #>>
                       '{via_cobertura,decision_gobernada,preparacion_evidencias_ref}'
                   AND lote.expediente_ref = v_fila.expediente_ref
                   AND lote.version_expediente = v_referencia_analisis
                   AND evidencia.expediente_ref = lote.expediente_ref
                   AND evidencia.version_expediente = lote.version_expediente
                 ORDER BY evidencia.posicion
            ) INTO v_comprobaciones;
            IF pg_catalog.cardinality(v_comprobaciones) = 0
               OR pg_catalog.cardinality(v_comprobaciones) > 32 THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
        ELSE
            IF NOT v_analisis_presente
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{via_cobertura,comprobaciones}'
               ) <> 'array' THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := v_referencia_analisis + 1;
            v_total := pg_catalog.jsonb_array_length(
                v_agregado #> '{via_cobertura,comprobaciones}'
            );
            FOR v_indice IN 0..v_total - 1 LOOP
                v_nodo := (
                    v_agregado #> '{via_cobertura,comprobaciones}'
                ) -> v_indice;
                v_comprobaciones := pg_catalog.array_append(
                    v_comprobaciones,
                    ROW(
                        v_nodo ->> 'clave',
                        v_nodo ->> 'resultado'
                    )::vec_contratacion_temporal
                        .comprobacion_operativa_rrhh_v1
                );
            END LOOP;
        END IF;
        v_cobertura := ROW(
            v_agregado #>> '{via_cobertura,via_clave}',
            v_decision_gobernada,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                v_agregado #>> '{via_cobertura,procedimiento_ref}' END,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                COALESCE(
                    v_agregado #>> '{via_cobertura,bolsa_ref}', ''
                ) END,
            v_comprobaciones
        );
    ELSE
        v_cobertura := NULL;
    END IF;

    v_asignacion_presente := v_agregado ? 'asignacion';
    IF v_asignacion_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'asignacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{asignacion,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_asignacion := (
            v_agregado #>>
            '{asignacion,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_asignacion := ROW(
            v_agregado #>> '{asignacion,unidad_ref}',
            (v_agregado #>> '{asignacion,asignada_en}')::timestamptz,
            COALESCE(v_agregado #>> '{asignacion,motivo_clave}', '')
        );
    ELSE
        v_asignacion := NULL;
    END IF;

    v_fiscalizacion_presente := v_agregado ? 'fiscalizacion';
    v_subsanacion_presente := false;
    IF v_fiscalizacion_presente THEN
        IF NOT v_asignacion_presente
           OR pg_catalog.jsonb_typeof(v_agregado -> 'fiscalizacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,actuacion_registro}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,resultado}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,fiscalizada_en}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,observaciones}'
           ) NOT IN ('string', 'null') THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_fiscalizacion := (v_agregado #>>
            '{fiscalizacion,actuacion_registro,secuencia}')::numeric(20, 0);
        IF v_referencia_fiscalizacion <= v_referencia_asignacion
           OR v_referencia_fiscalizacion > v_fila.version
           OR v_agregado #>> '{fiscalizacion,resultado}' NOT IN (
                'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
              ~ '[\x00-\x1F]'
           OR pg_catalog.char_length(
                COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
           ) > 2000
           OR (v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
               AND COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '') <> '')
           OR (v_agregado #>> '{fiscalizacion,resultado}' <> 'favorable'
               AND pg_catalog.char_length(
                   COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
               ) = 0) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_fiscalizacion := ROW(
            v_agregado #>> '{fiscalizacion,resultado}',
            CASE WHEN v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
                 THEN '' ELSE 'observaciones_fiscalizacion' END,
            COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', ''),
            (v_agregado #>> '{fiscalizacion,fiscalizada_en}')::timestamptz,
            NULL::timestamptz, NULL::text
        );
        IF v_agregado #>> '{fiscalizacion,resultado}' = 'desfavorable' THEN
            IF pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno}'
               ) <> 'object'
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno,retorno_ref}'
               ) <> 'string' THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            SELECT pg_catalog.count(*)
              INTO v_subsanaciones
              FROM pg_catalog.jsonb_array_elements(
                   v_agregado -> 'actuaciones'
              ) nodo
             WHERE nodo ->> 'accion_clave' =
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND nodo ->> 'retorno_ref' =
                   v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
            IF v_subsanaciones > 1 THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            IF v_subsanaciones = 1 THEN
                SELECT nodo INTO STRICT v_subsanacion_nodo
                  FROM pg_catalog.jsonb_array_elements(
                       v_agregado -> 'actuaciones'
                  ) nodo
                 WHERE nodo ->> 'accion_clave' =
                       'contratacion_temporal.subsanacion_reparos.registrar'
                   AND nodo ->> 'retorno_ref' =
                       v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
                IF pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'realizada_en'
                   ) <> 'string'
                   OR pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'observaciones'
                   ) <> 'string'
                   OR pg_catalog.char_length(
                       v_subsanacion_nodo ->> 'observaciones'
                   ) NOT BETWEEN 1 AND 2000
                   OR v_subsanacion_nodo ->> 'observaciones' ~ '[\x00-\x1F]' THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_referencia_subsanacion := (v_subsanacion_nodo ->>
                    'secuencia')::numeric(20, 0);
                IF v_referencia_subsanacion <= v_referencia_fiscalizacion
                   OR v_referencia_subsanacion > v_fila.version THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_subsanacion_presente := true;
                v_fiscalizacion.subsanacion_registrada_en :=
                    (v_subsanacion_nodo ->> 'realizada_en')::timestamptz;
                v_fiscalizacion.subsanacion_texto :=
                    v_subsanacion_nodo ->> 'observaciones';
            END IF;
        END IF;
    ELSE
        v_fiscalizacion := NULL;
    END IF;

    v_hitos := ARRAY[]::vec_contratacion_temporal
        .hito_expediente_rrhh_v1[];
    v_total := pg_catalog.jsonb_array_length(
        v_agregado -> 'actuaciones'
    );
    IF v_total < 1 OR v_total::numeric <> v_fila.version THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;
    FOR v_indice IN 0..v_total - 1 LOOP
        v_nodo := (v_agregado -> 'actuaciones') -> v_indice;
        v_hitos := pg_catalog.array_append(
            v_hitos,
            ROW(
                (v_nodo ->> 'secuencia')::numeric(20, 0),
                (v_nodo ->> 'version_expediente')::numeric(20, 0),
                v_nodo ->> 'accion_clave',
                (v_nodo ->> 'realizada_en')::timestamptz,
                COALESCE(v_nodo ->> 'fase_origen', ''),
                v_nodo ->> 'fase_destino',
                v_nodo ->> 'estado_origen',
                v_nodo ->> 'estado_destino'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        );
    END LOOP;

    v_detalle := ROW(
        v_resumen,
        v_solicitud,
        v_analisis_presente,
        v_analisis,
        v_referencia_analisis,
        v_cobertura_presente,
        v_cobertura,
        v_referencia_cobertura,
        v_asignacion_presente,
        v_asignacion,
        v_referencia_asignacion,
        v_hitos,
        v_fiscalizacion_presente,
        v_fiscalizacion,
        v_referencia_fiscalizacion,
        v_referencia_subsanacion
    );

    -- Reutiliza la validación del canon nominal sin volver a consultar ni
    -- conservar el agregado. El instante de la versión basta para esta
    -- validación estructural; el motor canoniza después su instante real.
    PERFORM
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resumen.actualizado_en,
            v_detalle
        );
    RETURN ROW(v_detalle)::
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
END
$funcion$;

CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
    p_generada_en timestamptz,
    p_entrada vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' || pg_catalog.chr(10)
        );
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_hito vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_anterior vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_comprobacion
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1;
    v_campos text[];
    v_campo text;
    v_claves_vistas text[] := ARRAY[]::text[];
    v_indice integer;
    v_total_hitos integer;
    v_total_comprobaciones integer;
    v_mascara smallint := 0;
    v_analisis_nulo boolean;
    v_cobertura_nula boolean;
    v_asignacion_nula boolean;
    v_fiscalizacion_nula boolean;
    v_subsanacion_presente boolean;
    v_utc timestamp;
BEGIN
    IF p_entrada.resumen IS NOT DISTINCT FROM NULL
       OR p_entrada.solicitud IS NOT DISTINCT FROM NULL
       OR p_entrada.analisis_presente IS NULL
       OR p_entrada.referencia_analisis IS NULL
       OR p_entrada.cobertura_presente IS NULL
       OR p_entrada.referencia_cobertura IS NULL
       OR p_entrada.asignacion_presente IS NULL
       OR p_entrada.referencia_asignacion IS NULL
       OR p_entrada.hitos IS NULL
       OR p_entrada.fiscalizacion_presente IS NULL
       OR p_entrada.referencia_fiscalizacion IS NULL
       OR p_entrada.referencia_subsanacion IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_resumen := p_entrada.resumen;
    v_solicitud := p_entrada.solicitud;
    v_analisis_nulo := p_entrada.analisis IS NOT DISTINCT FROM NULL;
    v_cobertura_nula := p_entrada.cobertura IS NOT DISTINCT FROM NULL;
    v_asignacion_nula := p_entrada.asignacion IS NOT DISTINCT FROM NULL;
    v_fiscalizacion_nula := p_entrada.fiscalizacion IS NOT DISTINCT FROM NULL;

    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        p_generada_en
    );
    v_canon := v_canon
        || vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
            v_resumen
        );
    IF p_generada_en < v_resumen.actualizado_en
       OR v_solicitud.grupo_subgrupo IS NULL
       OR v_solicitud.grupo_subgrupo !~
          '^[A-Z][A-Z0-9/+.-]{0,19}$'
       OR v_solicitud.motivo_clave IS NULL
       OR v_solicitud.motivo_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR v_solicitud.periodo_inicio IS NULL
       OR v_solicitud.periodo_fin IS NULL
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_inicio
    );
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_fin
    );
    v_utc := v_solicitud.periodo_inicio AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_utc := v_solicitud.periodo_fin AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    v_total_hitos := pg_catalog.cardinality(p_entrada.hitos);
    IF v_total_hitos < 1
       OR v_total_hitos::numeric <> v_resumen.version
       OR pg_catalog.array_ndims(p_entrada.hitos) <> 1
       OR pg_catalog.array_lower(p_entrada.hitos, 1) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF pg_catalog.array_position(p_entrada.hitos, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis_nulo
           OR p_entrada.referencia_analisis
              NOT BETWEEN 2 AND v_resumen.version THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 1;
        v_analisis := p_entrada.analisis;
    ELSIF NOT v_analisis_nulo OR p_entrada.referencia_analisis <> 0
          OR v_resumen.modalidad_clave <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.cobertura_presente THEN
        IF v_cobertura_nula OR NOT p_entrada.analisis_presente
           OR p_entrada.referencia_cobertura
              NOT BETWEEN 2 AND v_resumen.version
           OR p_entrada.referencia_cobertura
              <= p_entrada.referencia_analisis THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 2;
        v_cobertura := p_entrada.cobertura;
    ELSIF NOT v_cobertura_nula OR p_entrada.referencia_cobertura <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.asignacion_presente THEN
        IF v_asignacion_nula OR NOT p_entrada.cobertura_presente
           OR p_entrada.referencia_asignacion
              NOT BETWEEN 2 AND v_resumen.version
           OR p_entrada.referencia_asignacion
              <= p_entrada.referencia_cobertura THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 4;
        v_asignacion := p_entrada.asignacion;
    ELSIF NOT v_asignacion_nula OR p_entrada.referencia_asignacion <> 0
          OR v_resumen.unidad_ref <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.fiscalizacion_presente THEN
        IF v_fiscalizacion_nula OR NOT p_entrada.asignacion_presente
           OR p_entrada.referencia_fiscalizacion
              NOT BETWEEN 1 AND v_resumen.version
           OR p_entrada.referencia_fiscalizacion
              <= p_entrada.referencia_asignacion
           OR p_entrada.referencia_subsanacion < 0
           OR p_entrada.referencia_subsanacion > v_resumen.version THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 8;
        v_fiscalizacion := p_entrada.fiscalizacion;
    ELSIF NOT v_fiscalizacion_nula
          OR p_entrada.referencia_fiscalizacion <> 0
          OR p_entrada.referencia_subsanacion <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF v_mascara NOT IN (0, 1, 3, 7, 15) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis.modalidad_clave IS NULL
           OR v_analisis.modalidad_clave !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.modalidad_clave <> v_resumen.modalidad_clave
           OR v_analisis.categoria_ref IS NULL
           OR v_analisis.categoria_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_analisis.categoria_ref <> v_resumen.categoria_ref
           OR v_analisis.causa_clave IS NULL
           OR v_analisis.causa_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.periodo_inicio IS NULL
           OR v_analisis.periodo_fin IS NULL
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio
           OR v_analisis.porcentaje_jornada IS NULL
           OR v_analisis.porcentaje_jornada NOT BETWEEN 1 AND 10000
           OR v_analisis.resultado_rc IS NULL
           OR v_analisis.resultado_rc NOT IN (
               'validada', 'no_requerida', 'rechazada'
           )
           OR v_analisis.coste_presente IS NULL
           OR v_analisis.coste_centimos IS NULL
           OR v_analisis.coste_moneda IS NULL
           OR v_analisis.fuente_coste_ref IS NULL
           OR v_analisis.observaciones IS NULL
           OR pg_catalog.length(v_analisis.observaciones) > 4000 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_inicio
        );
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_fin
        );
        v_utc := v_analisis.periodo_inicio AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_utc := v_analisis.periodo_fin AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        IF v_analisis.coste_presente THEN
            IF v_analisis.coste_centimos <= 0
               OR v_analisis.coste_moneda <> 'EUR'
               OR v_analisis.fuente_coste_ref !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_analisis.coste_centimos <> 0
              OR v_analisis.coste_moneda <> ''
              OR v_analisis.fuente_coste_ref <> '' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END IF;

    IF p_entrada.cobertura_presente THEN
        IF v_cobertura.via_clave IS NULL
           OR v_cobertura.via_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_cobertura.decision_gobernada IS NULL
           OR v_cobertura.procedimiento_ref IS NULL
           OR v_cobertura.bolsa_ref IS NULL
           OR v_cobertura.comprobaciones IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_total_comprobaciones :=
            pg_catalog.cardinality(v_cobertura.comprobaciones);
        IF v_cobertura.decision_gobernada THEN
            IF v_cobertura.procedimiento_ref <> ''
               OR v_cobertura.bolsa_ref <> '' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            IF v_total_comprobaciones = 0 THEN
                IF pg_catalog.array_ndims(v_cobertura.comprobaciones)
                   IS NOT NULL THEN
                    RAISE EXCEPTION USING ERRCODE = '22023',
                        MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
            ELSIF v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(v_cobertura.comprobaciones) <> 1
               OR pg_catalog.array_lower(v_cobertura.comprobaciones, 1) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSE
            IF v_cobertura.procedimiento_ref !~
               '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_cobertura.bolsa_ref !~
                  '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(
                   v_cobertura.comprobaciones
               ) <> 1
               OR pg_catalog.array_lower(
                   v_cobertura.comprobaciones, 1
               ) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_total_comprobaciones > 0 THEN
            IF pg_catalog.array_position(v_cobertura.comprobaciones, NULL) IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            FOR v_indice IN 1..v_total_comprobaciones LOOP
                v_comprobacion := v_cobertura.comprobaciones[v_indice];
                IF v_comprobacion.clave IS NULL OR v_comprobacion.clave !~ '^[a-z][a-z0-9._-]{1,79}$'
                   OR v_comprobacion.resultado IS NULL OR v_comprobacion.resultado NOT IN ('afirmativa', 'negativa', 'no_aplica', 'no_consta')
                   OR v_comprobacion.clave = ANY(v_claves_vistas) THEN
                    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
                v_claves_vistas := pg_catalog.array_append(v_claves_vistas, v_comprobacion.clave);
            END LOOP;
        END IF;
    END IF;

    IF p_entrada.asignacion_presente THEN
        IF v_asignacion.unidad_ref IS NULL
           OR v_asignacion.unidad_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_asignacion.unidad_ref <> v_resumen.unidad_ref
           OR v_asignacion.asignada_en IS NULL
           OR v_asignacion.asignada_en < v_resumen.creado_en
           OR v_asignacion.asignada_en > v_resumen.actualizado_en
           OR v_asignacion.motivo_clave IS NULL
           OR v_asignacion.motivo_clave !~
              '^$|^[a-z][a-z0-9._-]{1,79}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_asignacion.asignada_en
        );
    END IF;

    FOR v_indice IN 1..v_total_hitos LOOP
        v_hito := p_entrada.hitos[v_indice];
        IF v_hito.secuencia IS NULL
           OR v_hito.secuencia <> v_indice
           OR v_hito.version_expediente IS NULL
           OR v_hito.version_expediente <> v_indice
           OR v_hito.accion_clave IS NULL
           OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.realizada_en IS NULL
           OR v_hito.realizada_en > v_resumen.actualizado_en
           OR v_hito.fase_origen IS NULL
           OR v_hito.fase_origen !~
              '^$|^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.fase_destino IS NULL
           OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.estado_origen IS NULL
           OR v_hito.estado_origen NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           )
           OR v_hito.estado_destino IS NULL
           OR v_hito.estado_destino NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_hito.realizada_en
        );
        IF p_entrada.fiscalizacion_presente
           AND v_indice = p_entrada.referencia_fiscalizacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.fiscalizacion.registrar'
               OR v_hito.realizada_en <> v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF p_entrada.fiscalizacion_presente
           AND p_entrada.referencia_subsanacion <> 0
           AND v_indice = p_entrada.referencia_subsanacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.subsanacion_reparos.registrar'
               OR v_hito.realizada_en <>
                  v_fiscalizacion.subsanacion_registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_indice = 1 THEN
            IF v_hito.fase_origen <> ''
               OR v_hito.estado_origen <> 'pendiente' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_hito.fase_origen <> v_anterior.fase_destino
              OR v_hito.estado_origen <> v_anterior.estado_destino
              OR v_hito.realizada_en < v_anterior.realizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_anterior := v_hito;
    END LOOP;
    IF v_anterior.fase_destino <> v_resumen.fase_clave
       OR v_anterior.estado_destino <> v_resumen.estado_clave
       OR v_anterior.realizada_en <> v_resumen.actualizado_en
       OR (
           p_entrada.asignacion_presente
           AND (p_entrada.hitos[
               p_entrada.referencia_asignacion::integer
           ]).realizada_en <> v_asignacion.asignada_en
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    v_campos := ARRAY[
        v_solicitud.grupo_subgrupo,
        v_solicitud.motivo_clave,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_inicio
        ),
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_fin
        ),
        v_mascara::text
    ]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.analisis_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_analisis::text
    ]::text[];
    IF p_entrada.analisis_presente THEN
        v_campos := v_campos || ARRAY[
            v_analisis.modalidad_clave, v_analisis.categoria_ref,
            v_analisis.causa_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_inicio
            ),
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_fin
            ),
            v_analisis.porcentaje_jornada::text,
            v_analisis.resultado_rc,
            CASE WHEN v_analisis.coste_presente THEN '1' ELSE '0' END
        ]::text[];
        IF v_analisis.coste_presente THEN
            v_campos := v_campos || ARRAY[
                v_analisis.coste_centimos::text,
                v_analisis.coste_moneda
            ]::text[];
        END IF;
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.fuente_coste_ref
        );
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.observaciones
        );
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.cobertura_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_cobertura::text
    ]::text[];
    IF p_entrada.cobertura_presente THEN
        v_campos := v_campos || ARRAY[
            v_cobertura.via_clave,
            CASE WHEN v_cobertura.decision_gobernada THEN '1' ELSE '0' END,
            v_cobertura.procedimiento_ref, v_cobertura.bolsa_ref,
            v_total_comprobaciones::text
        ]::text[];
        FOR v_indice IN 1..v_total_comprobaciones LOOP
            v_comprobacion := v_cobertura.comprobaciones[v_indice];
            v_campos := v_campos || ARRAY[
                v_comprobacion.clave, v_comprobacion.resultado
            ]::text[];
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.asignacion_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_asignacion::text
    ]::text[];
    IF p_entrada.asignacion_presente THEN
        v_campos := v_campos || ARRAY[
            v_asignacion.unidad_ref,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_asignacion.asignada_en
            ),
            v_asignacion.motivo_clave
        ]::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_campos := ARRAY[
            '1', p_entrada.referencia_fiscalizacion::text,
            p_entrada.referencia_subsanacion::text,
            v_fiscalizacion.resultado_clave,
            CASE WHEN v_fiscalizacion.reparo_clave = '' THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.reparo_clave <> '' THEN
            v_campos := v_campos || ARRAY[
                v_fiscalizacion.reparo_clave, v_fiscalizacion.reparo_texto
            ]::text[];
        END IF;
        v_campos := v_campos || ARRAY[
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.registrada_en
            ),
            CASE WHEN v_fiscalizacion.subsanacion_registrada_en IS NULL
                 THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.subsanacion_registrada_en IS NOT NULL THEN
            v_campos := v_campos || ARRAY[
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_fiscalizacion.subsanacion_registrada_en
                ), v_fiscalizacion.subsanacion_texto
            ]::text[];
        END IF;
    ELSE
        v_campos := ARRAY['0', '0', '0']::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[v_total_hitos::text]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_subsanacion_presente :=
            v_fiscalizacion.subsanacion_registrada_en IS NOT NULL
            OR v_fiscalizacion.subsanacion_texto IS NOT NULL;
        IF v_fiscalizacion.resultado_clave NOT IN (
               'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR v_fiscalizacion.reparo_clave IS NULL
           OR v_fiscalizacion.reparo_texto IS NULL
           OR v_fiscalizacion.registrada_en IS NULL
           OR (v_fiscalizacion.resultado_clave = 'favorable'
               AND (v_fiscalizacion.reparo_clave <> ''
                    OR v_fiscalizacion.reparo_texto <> ''))
           OR (v_fiscalizacion.resultado_clave <> 'favorable'
               AND (v_fiscalizacion.reparo_clave <> 'observaciones_fiscalizacion'
                    OR pg_catalog.char_length(v_fiscalizacion.reparo_texto)
                       NOT BETWEEN 1 AND 2000))
           OR (v_subsanacion_presente
               AND (v_fiscalizacion.resultado_clave <> 'desfavorable'
                    OR v_fiscalizacion.subsanacion_registrada_en IS NULL
                    OR v_fiscalizacion.subsanacion_texto IS NULL
                    OR pg_catalog.char_length(v_fiscalizacion.subsanacion_texto)
                       NOT BETWEEN 1 AND 2000
                    OR p_entrada.referencia_subsanacion
                       <= p_entrada.referencia_fiscalizacion))
           OR (NOT v_subsanacion_presente
               AND p_entrada.referencia_subsanacion <> 0) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_fiscalizacion.registrada_en
        );
        IF v_subsanacion_presente THEN
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.subsanacion_registrada_en
            );
            IF v_fiscalizacion.subsanacion_registrada_en
               <= v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
    END IF;

    FOR v_indice IN 1..v_total_hitos LOOP
        v_hito := p_entrada.hitos[v_indice];
        v_campos := ARRAY[
            v_hito.secuencia::text,
            v_hito.version_expediente::text,
            v_hito.accion_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_hito.realizada_en
            ),
            v_hito.fase_origen, v_hito.fase_destino,
            v_hito.estado_origen, v_hito.estado_destino
        ]::text[];
        FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon := v_canon
                || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                    vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                        v_campo
                    )
                );
        END LOOP;
        IF pg_catalog.octet_length(v_canon) > 262144 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END LOOP;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'contenido de detalle RRHH inválido';
END
$funcion$;

COMMIT;
