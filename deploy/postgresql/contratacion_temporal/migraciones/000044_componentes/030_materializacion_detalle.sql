-- CT-000044: materialización privada y minimizada del detalle RRHH.
--
-- La versión se elige una sola vez sobre el corte global que ya fijó el
-- motor. El valor cero significa «última versión visible en ese corte»; una
-- versión positiva actúa como control de obsolescencia y debe coincidir con
-- esa última versión. El ámbito se comprueba después de elegirla para impedir
-- que un cambio de centro o unidad haga reaparecer una versión anterior.
CREATE FUNCTION
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
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_hitos
        vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_analisis_presente boolean;
    v_cobertura_presente boolean;
    v_asignacion_presente boolean;
    v_decision_gobernada boolean;
    v_coste_presente boolean;
    v_referencia_analisis numeric(20, 0) := 0;
    v_referencia_cobertura numeric(20, 0) := 0;
    v_referencia_asignacion numeric(20, 0) := 0;
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
            ELSE '' END
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
            v_referencia_cobertura := (
                v_agregado #>>
                '{via_cobertura,decision_gobernada,actuacion,version_expediente}'
            )::numeric(20, 0);
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
        v_hitos
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

ALTER FUNCTION
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    numeric
) OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    numeric
)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

COMMENT ON FUNCTION
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    numeric
) IS
'Materializa una única versión actual al corte y un detalle RRHH minimizado.';
