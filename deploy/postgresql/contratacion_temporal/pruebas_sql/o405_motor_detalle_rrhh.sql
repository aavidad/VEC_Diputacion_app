\set ON_ERROR_STOP on

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '20s';

CREATE SCHEMA vec_ct44_detalle_prueba AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA vec_ct44_detalle_prueba FROM PUBLIC;

-- Construye agregados sintéticos progresivos con datos que deben quedar
-- fuera de la proyección. La versión 5 reasigna la unidad para probar que el
-- motor no retrocede a la versión 4 al aplicar el ámbito.
CREATE FUNCTION vec_ct44_detalle_prueba.agregado(p_version integer)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_agregado jsonb;
    v_actuaciones jsonb := '[]'::jsonb;
    v_indice integer;
    v_fase_origen text;
    v_fase_destino text;
    v_actualizado text;
    v_unidad text;
BEGIN
    IF p_version NOT BETWEEN 1 AND 5 THEN
        RAISE EXCEPTION 'versión sintética fuera de rango';
    END IF;
    FOR v_indice IN 1..p_version LOOP
        v_fase_origen := CASE v_indice
            WHEN 1 THEN ''
            WHEN 2 THEN 'solicitud'
            WHEN 3 THEN 'analisis_rrhh'
            ELSE 'unidad_gestora'
        END;
        v_fase_destino := CASE v_indice
            WHEN 1 THEN 'solicitud'
            WHEN 2 THEN 'analisis_rrhh'
            WHEN 3 THEN 'unidad_gestora'
            ELSE 'unidad_gestora'
        END;
        v_actuaciones := v_actuaciones || pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'secuencia', v_indice,
                'version_expediente', v_indice,
                'accion_clave', 'actuacion.sintetica.' || v_indice::text,
                'actor_ref', 'actor:sintetico:no-publicable',
                'unidad_ref', 'unidad:sintetica:tramitadora',
                'recibo_ref', 'recibo:sintetico:no-publicable',
                'realizada_en',
                    '2026-07-29T08:0' || (v_indice - 1)::text
                    || ':00.000000Z',
                'fase_origen', v_fase_origen,
                'fase_destino', v_fase_destino,
                'estado_origen',
                    CASE WHEN v_indice = 1
                        THEN 'pendiente' ELSE 'en_curso' END,
                'estado_destino', 'en_curso',
                'observaciones', 'MARCADOR_PRIVADO_NO_PUBLICABLE',
                'documentos_ref', pg_catalog.jsonb_build_array(
                    'documento:sintetico:no-publicable'
                )
            )
        );
    END LOOP;
    v_actualizado :=
        '2026-07-29T08:0' || (p_version - 1)::text || ':00.000000Z';
    v_agregado := pg_catalog.jsonb_build_object(
        'referencia', 'expediente:ct44:detalle',
        'organizacion_ref', 'organizacion:ct44:sintetica',
        'numero_visible', '2026/CT44-DET',
        'version', p_version,
        'flujo', pg_catalog.jsonb_build_object(
            'definicion_ref', 'flujo:ct44:sintetico',
            'version', 1,
            'huella_sha256', pg_catalog.repeat('a', 64)
        ),
        'fase_actual', CASE p_version
            WHEN 1 THEN 'solicitud'
            WHEN 2 THEN 'analisis_rrhh'
            ELSE 'unidad_gestora'
        END,
        'estado_actual', 'en_curso',
        'solicitud', pg_catalog.jsonb_build_object(
            'centro_ref', 'centro:ct44:sintetico',
            'contacto_ref', 'contacto:sintetico:no-publicable',
            'categoria_ref', 'categoria:ct44:inicial',
            'grupo_subgrupo', 'C2',
            'motivo_clave', 'sustitucion',
            'detalle', 'MARCADOR_PRIVADO_NO_PUBLICABLE',
            'periodo', pg_catalog.jsonb_build_object(
                'inicio', '2026-08-01T00:00:00.000000Z',
                'fin', '2026-09-01T00:00:00.000000Z'
            ),
            'rc', pg_catalog.jsonb_build_object('existe', false),
            'documentos_adjuntos', pg_catalog.jsonb_build_array(
                'documento:sintetico:no-publicable'
            ),
            'observaciones', 'MARCADOR_PRIVADO_NO_PUBLICABLE'
        ),
        'creado_en', '2026-07-29T08:00:00.000000Z',
        'actualizado_en', v_actualizado,
        'actuaciones', v_actuaciones
    );
    IF p_version >= 2 THEN
        v_agregado := v_agregado || pg_catalog.jsonb_build_object(
            'analisis', pg_catalog.jsonb_build_object(
                'modalidad_clave', 'interinidad',
                'categoria_ref', 'categoria:ct44:analizada',
                'grupo_subgrupo', 'C2',
                'causa_clave', 'sustitucion',
                'periodo', pg_catalog.jsonb_build_object(
                    'inicio', '2026-08-01T00:00:00.000000Z',
                    'fin', '2026-09-01T00:00:00.000000Z'
                ),
                'porcentaje_jornada', 7500,
                'actuacion_registro', pg_catalog.jsonb_build_object(
                    'secuencia', 2,
                    'version_expediente', 2,
                    'accion_clave', 'actuacion.sintetica.2',
                    'fase_destino', 'analisis_rrhh',
                    'recibo_ref', 'recibo:sintetico:no-publicable'
                ),
                'validacion_rc', pg_catalog.jsonb_build_object(
                    'resultado', 'no_requerida',
                    'motivo', 'MARCADOR_PRIVADO_NO_PUBLICABLE'
                ),
                'coste_previsto', pg_catalog.jsonb_build_object(
                    'centimos', 125000,
                    'moneda', 'EUR'
                ),
                'fuente_coste_ref', 'fuente:coste:ct44:sintetica',
                'observaciones', 'MARCADOR_PRIVADO_NO_PUBLICABLE'
            )
        );
    END IF;
    IF p_version >= 3 THEN
        v_agregado := v_agregado || pg_catalog.jsonb_build_object(
            'via_cobertura', pg_catalog.jsonb_build_object(
                'via_clave', 'bolsa',
                'decision_gobernada', pg_catalog.jsonb_build_object(
                    'referencia', 'decision:sintetica:no-publicable',
                    'actor_ref', 'actor:sintetico:no-publicable',
                    'actuacion', pg_catalog.jsonb_build_object(
                        'secuencia', 3,
                        'version_expediente', 3,
                        'accion_clave', 'actuacion.sintetica.3',
                        'realizada_en',
                            '2026-07-29T08:02:00.000000Z',
                        'fase_destino', 'unidad_gestora'
                    )
                )
            )
        );
    END IF;
    IF p_version >= 4 THEN
        v_unidad := CASE p_version
            WHEN 4 THEN 'unidad:ct44:anterior'
            ELSE 'unidad:ct44:vigente'
        END;
        v_agregado := v_agregado || pg_catalog.jsonb_build_object(
            'asignacion', pg_catalog.jsonb_build_object(
                'unidad_ref', v_unidad,
                'responsable_ref', 'responsable:sintetico:no-publicable',
                'notificacion_ref', 'notificacion:sintetica:no-publicable',
                'asignada_en', v_actualizado,
                'motivo_clave',
                    CASE WHEN p_version = 4 THEN '' ELSE 'reasignacion' END,
                'observaciones', 'MARCADOR_PRIVADO_NO_PUBLICABLE',
                'actuacion_registro', pg_catalog.jsonb_build_object(
                    'secuencia', p_version,
                    'version_expediente', p_version,
                    'accion_clave',
                        'actuacion.sintetica.' || p_version::text,
                    'fase_destino', 'unidad_gestora',
                    'recibo_ref', 'recibo:sintetico:no-publicable',
                    'unidad_asignada_ref', v_unidad,
                    'responsable_asignado_ref',
                        'responsable:sintetico:no-publicable',
                    'notificacion_ref',
                        'notificacion:sintetica:no-publicable',
                    'motivo_clave',
                        CASE WHEN p_version = 4
                            THEN '' ELSE 'reasignacion' END
                )
            )
        );
    END IF;
    RETURN v_agregado;
END
$funcion$;

CREATE FUNCTION vec_ct44_detalle_prueba.crear_alta()
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM pg_catalog.set_config(
        'session_replication_role', 'replica', true
    );
    INSERT INTO vec_contratacion_temporal.expediente_alta (
        expediente_ref, reserva_ref, numero_visible, organizacion_ref,
        actor_ref, perfil_ref, decision_ref, efecto_ref,
        huella_efecto_sha256, creada_en, confirmacion_ref
    ) VALUES (
        'expediente:ct44:detalle',
        'reserva:ct44:detalle',
        '2026/CT44-DET',
        'organizacion:ct44:sintetica',
        'actor:sintetico:no-publicable',
        'perfil:sintetico:rrhh',
        'decision:ct44:detalle',
        'efecto:ct44:detalle',
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to('efecto:ct44:detalle', 'UTF8')
        ), 'hex'),
        '2026-07-29T08:00:00Z'::timestamptz,
        'cnf_ct_' || pg_catalog.substr(pg_catalog.encode(
            pg_catalog.sha256(pg_catalog.convert_to(
                'confirmacion:ct44:detalle', 'UTF8'
            )), 'hex'), 1, 32)
    );
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_ct44_detalle_prueba.agregado(integer),
    vec_ct44_detalle_prueba.crear_alta()
FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_ct44_detalle_prueba
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
    vec_ct44_detalle_prueba.agregado(integer)
    TO vec_contratacion_temporal_propietario;

SELECT vec_ct44_detalle_prueba.crear_alta();
SET LOCAL session_replication_role = origin;

SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $cargar_versiones$
DECLARE
    v_version integer;
    v_agregado jsonb;
    v_prueba bytea;
BEGIN
    FOR v_version IN 1..5 LOOP
        v_agregado := vec_ct44_detalle_prueba.agregado(v_version);
        v_prueba := pg_catalog.convert_to(
            pg_catalog.repeat(
                'prueba:ct44:detalle:' || v_version::text || ':',
                8
            ),
            'UTF8'
        );
        INSERT INTO vec_contratacion_temporal.expediente_version_integral (
            expediente_ref, version, agregado_json,
            agregado_json_huella_sha256, prueba_canonica,
            prueba_huella_sha256, flujo_ref, flujo_version,
            flujo_huella_sha256, fase_clave, estado, origen_version,
            operacion_ref, registrada_en
        ) VALUES (
            'expediente:ct44:detalle',
            v_version,
            v_agregado,
            pg_catalog.encode(pg_catalog.sha256(
                pg_catalog.convert_to(v_agregado::text, 'UTF8')
            ), 'hex'),
            v_prueba,
            pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
            'flujo:ct44:sintetico',
            1,
            pg_catalog.repeat('a', 64),
            CASE v_version
                WHEN 1 THEN 'solicitud'
                WHEN 2 THEN 'analisis_rrhh'
                ELSE 'unidad_gestora'
            END,
            'en_curso',
            CASE v_version
                WHEN 1 THEN 'alta_o2'
                WHEN 2 THEN 'analisis_o3'
                WHEN 3 THEN 'cobertura_o4'
                ELSE 'asignacion_o5'
            END,
            'operacion:ct44:detalle:' || v_version::text,
            (
                '2026-07-29T08:0' || (v_version - 1)::text
                || ':00.000000Z'
            )::timestamptz
        );
    END LOOP;
END
$cargar_versiones$;

DO $detalle$
DECLARE
    v_corte_4 numeric(20, 0);
    v_corte_5 numeric(20, 0);
    v_material
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_salida jsonb;
    v_codigo text;
    v_mensaje text;
    v_caso record;
    v_efectos_antes bigint;
    v_efectos_despues bigint;
BEGIN
    SELECT corte_global INTO STRICT v_corte_4
      FROM vec_contratacion_temporal.publicacion_version_rrhh
     WHERE expediente_ref = 'expediente:ct44:detalle'
       AND version = 4;
    SELECT corte_global INTO STRICT v_corte_5
      FROM vec_contratacion_temporal.publicacion_version_rrhh
     WHERE expediente_ref = 'expediente:ct44:detalle'
       AND version = 5;

    SELECT pg_catalog.count(*)
      INTO v_efectos_antes
      FROM vec_contratacion_temporal.registro_acceso_rrhh;

    v_material :=
        vec_contratacion_temporal.materializar_detalle_rrhh_v1(
            ROW(
                'organizacion:ct44:sintetica',
                'organizacion',
                'organizacion:ct44:sintetica'
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
            ROW(
                'expediente:ct44:detalle', 0
            )::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
            v_corte_4
        );
    v_detalle := v_material.detalle;
    IF (v_detalle.resumen).version <> 4
       OR NOT v_detalle.analisis_presente
       OR v_detalle.referencia_analisis <> 2
       OR NOT v_detalle.cobertura_presente
       OR v_detalle.referencia_cobertura <> 3
       OR NOT (v_detalle.cobertura).decision_gobernada
       OR NOT v_detalle.asignacion_presente
       OR v_detalle.referencia_asignacion <> 4
       OR (v_detalle.asignacion).unidad_ref <>
          'unidad:ct44:anterior'
       OR pg_catalog.cardinality(v_detalle.hitos) <> 4 THEN
        RAISE EXCEPTION 'primera carga al corte 4 divergente';
    END IF;

    v_material :=
        vec_contratacion_temporal.materializar_detalle_rrhh_v1(
            ROW(
                'organizacion:ct44:sintetica',
                'unidad_gestion',
                'unidad:ct44:vigente'
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
            ROW(
                'expediente:ct44:detalle', 5
            )::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
            v_corte_5
        );
    v_detalle := v_material.detalle;
    v_salida := pg_catalog.to_jsonb(v_detalle);
    IF (v_detalle.resumen).version <> 5
       OR (v_detalle.asignacion).unidad_ref <>
          'unidad:ct44:vigente'
       OR (v_detalle.asignacion).motivo_clave <> 'reasignacion'
       OR v_salida::text ~
          '(MARCADOR_PRIVADO|actor:sintetico|contacto:sintetico|documento:sintetico|responsable:sintetico|notificacion:sintetica)'
       OR v_salida ? 'agregado_json' THEN
        RAISE EXCEPTION 'detalle vigente no es íntegro o minimizado';
    END IF;

    -- Todos estos casos comparten exactamente código y mensaje: inexistente,
    -- organización ajena, ámbito ajeno, versión futura y versión obsoleta.
    FOR v_caso IN
        SELECT *
          FROM (VALUES
              (
                  'ausente',
                  'organizacion:ct44:sintetica',
                  'organizacion',
                  'organizacion:ct44:sintetica',
                  'expediente:ct44:ausente',
                  0::numeric,
                  v_corte_5
              ),
              (
                  'organizacion_ajena',
                  'organizacion:ct44:ajena',
                  'organizacion',
                  'organizacion:ct44:ajena',
                  'expediente:ct44:detalle',
                  0::numeric,
                  v_corte_5
              ),
              (
                  'unidad_anterior',
                  'organizacion:ct44:sintetica',
                  'unidad_gestion',
                  'unidad:ct44:anterior',
                  'expediente:ct44:detalle',
                  0::numeric,
                  v_corte_5
              ),
              (
                  'version_futura_al_corte',
                  'organizacion:ct44:sintetica',
                  'organizacion',
                  'organizacion:ct44:sintetica',
                  'expediente:ct44:detalle',
                  5::numeric,
                  v_corte_4
              ),
              (
                  'version_obsoleta',
                  'organizacion:ct44:sintetica',
                  'organizacion',
                  'organizacion:ct44:sintetica',
                  'expediente:ct44:detalle',
                  4::numeric,
                  v_corte_5
              )
          ) AS casos(
              nombre, organizacion_ref, clase_ambito, ambito_ref,
              expediente_ref, version_observada, corte_global
          )
    LOOP
        BEGIN
            PERFORM
                vec_contratacion_temporal.materializar_detalle_rrhh_v1(
                    ROW(
                        v_caso.organizacion_ref,
                        v_caso.clase_ambito,
                        v_caso.ambito_ref
                    )::vec_contratacion_temporal
                        .alcance_consulta_rrhh_v1,
                    ROW(
                        v_caso.expediente_ref,
                        v_caso.version_observada
                    )::vec_contratacion_temporal
                        .consulta_detalle_rrhh_v1,
                    v_caso.corte_global
                );
            RAISE EXCEPTION 'caso no observable aceptado: %',
                v_caso.nombre;
        EXCEPTION WHEN SQLSTATE '42501' THEN
            GET STACKED DIAGNOSTICS
                v_codigo = RETURNED_SQLSTATE,
                v_mensaje = MESSAGE_TEXT;
            IF v_codigo <> '42501'
               OR v_mensaje <> 'detalle RRHH no disponible' THEN
                RAISE EXCEPTION
                    'oráculo exterior en %: %/%',
                    v_caso.nombre, v_codigo, v_mensaje;
            END IF;
        END;
    END LOOP;

    SELECT pg_catalog.count(*)
      INTO v_efectos_despues
      FROM vec_contratacion_temporal.registro_acceso_rrhh;
    IF v_efectos_despues <> v_efectos_antes THEN
        RAISE EXCEPTION
            'el materializador produjo efectos de acceso';
    END IF;
END
$detalle$;

ROLLBACK;
