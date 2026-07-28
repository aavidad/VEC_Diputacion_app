\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
SET search_path = pg_catalog;

DO $codec_y_guc$
DECLARE
    v_area bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1('Área_Ñ');
    v_instante text;
    v_locale text := pg_catalog.current_setting('lc_time');
BEGIN
    IF vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(v_area)
       IS DISTINCT FROM 'Área_Ñ'
       OR vec_contratacion_temporal.encuadrar_valor_rrhh_v1(v_area)
          IS DISTINCT FROM pg_catalog.convert_to(
              '8:Área_Ñ' || pg_catalog.chr(10), 'UTF8'
          ) THEN
        RAISE EXCEPTION 'vector UTF-8 divergente';
    END IF;
    IF pg_catalog.octet_length(
        vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            pg_catalog.decode(pg_catalog.repeat('00', 262136), 'hex')
        )
    ) <> 262144 THEN
        RAISE EXCEPTION 'límite exacto del encuadrador divergente';
    END IF;
    BEGIN
        PERFORM vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            pg_catalog.decode(pg_catalog.repeat('00', 262137), 'hex')
        );
        RAISE EXCEPTION 'exceso del encuadrador aceptado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    FOREACH v_area IN ARRAY ARRAY[
        '\x80', '\xc080', '\xeda080', '\xf4908080', '\x00'
    ]::bytea[] LOOP
        BEGIN
            PERFORM
                vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(
                    v_area
                );
            RAISE EXCEPTION 'UTF-8 inválido aceptado';
        EXCEPTION WHEN SQLSTATE '22023' THEN
            NULL;
        END;
    END LOOP;
    v_instante :=
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            '2026-07-28T08:09:10.123456Z'
        );
    PERFORM pg_catalog.set_config('TimeZone', 'Pacific/Chatham', true);
    PERFORM pg_catalog.set_config('DateStyle', 'German, DMY', true);
    PERFORM pg_catalog.set_config(
        'lc_time',
        CASE WHEN v_locale = 'C' THEN 'C.UTF-8' ELSE 'C' END,
        true
    );
    PERFORM pg_catalog.set_config(
        'search_path', 'pg_temp, public', true
    );
    IF vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
           '2026-07-28T08:09:10.123456Z'
       ) IS DISTINCT FROM v_instante
       OR v_instante <> '2026-07-28T08:09:10.123456Z' THEN
        RAISE EXCEPTION 'instante dependiente de GUC';
    END IF;
END
$codec_y_guc$;

DO $arrays_hostiles$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1 := ROW(
        'expediente:rrhh:array', 'organizacion:diputacion-granada',
        '2026/CT-ARRAY', 1, 'flujo:rrhh:array', 1,
        pg_catalog.repeat('a', 64), 'solicitud', 'en_curso',
        'centro:rrhh:array', 'categoria:rrhh:array', '', '',
        '2026-07-26T08:00:00Z', '2026-07-26T08:00:00Z'
    );
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1 :=
        ROW(
            'C2', 'sustitucion',
            '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z'
        );
    v_hito vec_contratacion_temporal.hito_expediente_rrhh_v1 := ROW(
        1, 1, 'actuacion.array.1', '2026-07-26T08:00:00Z',
        '', 'solicitud', 'pendiente', 'en_curso'
    );
    v_entrada
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
BEGIN
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
                '2026-07-26T08:01:00Z',
                ARRAY[[v_resumen]], false, ''::bytea
            );
        RAISE EXCEPTION 'cuadro multidimensional aceptado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
                '2026-07-26T08:01:00Z',
                ARRAY[
                    NULL::vec_contratacion_temporal
                        .resumen_publicacion_rrhh_v1
                ],
                false, ''::bytea
            );
        RAISE EXCEPTION 'cuadro con elemento nulo aceptado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
                '2026-07-26T08:01:00Z',
                pg_catalog.array_fill(
                    v_resumen, ARRAY[1], ARRAY[0]
                ),
                false, ''::bytea
            );
        RAISE EXCEPTION 'cuadro con límite inferior cero aceptado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    v_entrada := ROW(
        v_resumen, v_solicitud,
        false, NULL::vec_contratacion_temporal.analisis_operativo_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
        0, ARRAY[[v_hito]]
    );
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                '2026-07-26T08:01:00Z', v_entrada
            );
        RAISE EXCEPTION 'hitos multidimensionales aceptados';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    v_entrada := ROW(
        v_resumen, v_solicitud,
        false, NULL::vec_contratacion_temporal.analisis_operativo_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
        0, pg_catalog.array_fill(v_hito, ARRAY[1], ARRAY[0])
    );
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                '2026-07-26T08:01:00Z', v_entrada
            );
        RAISE EXCEPTION 'hitos con límite inferior cero aceptados';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    v_entrada := ROW(
        v_resumen, v_solicitud,
        false, NULL::vec_contratacion_temporal.analisis_operativo_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
        0,
        ARRAY[
            NULL::vec_contratacion_temporal.hito_expediente_rrhh_v1
        ]
    );
    BEGIN
        PERFORM
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                '2026-07-26T08:01:00Z', v_entrada
            );
        RAISE EXCEPTION 'hitos con elemento nulo aceptados';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
END
$arrays_hostiles$;

DO $compuestos_parcialmente_nulos$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1 := ROW(
        'expediente:rrhh:nulos', 'organizacion:diputacion-granada',
        '2026/CT-NULOS', 4, 'flujo:rrhh:nulos', 1,
        pg_catalog.repeat('a', 64), 'unidad_gestora', 'en_curso',
        'centro:rrhh:nulos', 'categoria:rrhh:nulos',
        'interinidad', 'unidad:rrhh:nulos',
        '2026-07-26T07:57:00Z', '2026-07-26T08:00:00Z'
    );
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1 :=
        ROW(
            'C2', 'sustitucion',
            '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z'
        );
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_comprobacion
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1;
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion
        vec_contratacion_temporal.asignacion_operativa_rrhh_v1 := ROW(
            'unidad:rrhh:nulos', '2026-07-26T08:00:00Z', ''
        );
    v_hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_entrada
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_caso integer;
BEGIN
    FOR v_caso IN 1..5 LOOP
        v_analisis := ROW(
            'interinidad', 'categoria:rrhh:nulos', 'sustitucion',
            '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z',
            CASE WHEN v_caso = 1 THEN NULL::smallint ELSE 10000 END,
            CASE WHEN v_caso = 2 THEN NULL::text
                 ELSE 'no_requerida' END,
            true, 125000, 'EUR', 'fuente:coste:nulos'
        );
        v_comprobacion := ROW(
            'disponibilidad',
            CASE WHEN v_caso = 3 THEN NULL::text ELSE 'afirmativa' END
        );
        v_cobertura := ROW(
            'bolsa', false, 'procedimiento:rrhh:nulos',
            'bolsa:rrhh:nulos', ARRAY[v_comprobacion]
        );
        v_hitos := ARRAY[
            ROW(
                1, 1, 'actuacion.nulos.1',
                '2026-07-26T07:57:00Z', '', 'solicitud',
                CASE WHEN v_caso = 4 THEN NULL::text ELSE 'pendiente' END,
                CASE WHEN v_caso = 5 THEN NULL::text ELSE 'en_curso' END
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
            ROW(
                2, 2, 'actuacion.nulos.2',
                '2026-07-26T07:58:00Z', 'solicitud', 'gestion_bolsa',
                'en_curso', 'en_curso'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
            ROW(
                3, 3, 'actuacion.nulos.3',
                '2026-07-26T07:59:00Z', 'gestion_bolsa',
                'asignacion_unidad', 'en_curso', 'en_curso'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
            ROW(
                4, 4, 'actuacion.nulos.4',
                '2026-07-26T08:00:00Z', 'asignacion_unidad',
                'unidad_gestora', 'en_curso', 'en_curso'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        ];
        v_entrada := ROW(
            v_resumen, v_solicitud, true, v_analisis, 2,
            true, v_cobertura, 3, true, v_asignacion, 4, v_hitos
        );
        BEGIN
            PERFORM
                vec_contratacion_temporal
                    .canon_contenido_detalle_rrhh_v1(
                        '2026-07-26T08:01:00Z', v_entrada
                    );
            RAISE EXCEPTION 'compuesto parcial nulo aceptado: %', v_caso;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            IF SQLERRM <> 'contenido de detalle RRHH inválido' THEN
                RAISE;
            END IF;
        END;
    END LOOP;
    v_analisis := ROW(
        'interinidad', 'categoria:rrhh:nulos', 'sustitucion',
        '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z',
        10000, 'no_requerida', true, 125000, 'EUR',
        'fuente:coste:nulos'
    );
    v_comprobacion := ROW('disponibilidad', 'afirmativa');
    v_hitos := ARRAY[
        ROW(
            1, 1, 'actuacion.nulos.1',
            '2026-07-26T07:57:00Z', '', 'solicitud',
            'pendiente', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            2, 2, 'actuacion.nulos.2',
            '2026-07-26T07:58:00Z', 'solicitud', 'gestion_bolsa',
            'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            3, 3, 'actuacion.nulos.3',
            '2026-07-26T07:59:00Z', 'gestion_bolsa',
            'asignacion_unidad', 'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            4, 4, 'actuacion.nulos.4',
            '2026-07-26T08:00:00Z', 'asignacion_unidad',
            'unidad_gestora', 'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1
    ];
    FOR v_caso IN 1..3 LOOP
        v_comprobaciones := CASE v_caso
            WHEN 1 THEN ARRAY[[v_comprobacion]]
            WHEN 2 THEN pg_catalog.array_fill(
                v_comprobacion, ARRAY[1], ARRAY[0]
            )
            ELSE ARRAY[
                NULL::vec_contratacion_temporal
                    .comprobacion_operativa_rrhh_v1
            ]
        END;
        v_cobertura := ROW(
            'bolsa', false, 'procedimiento:rrhh:nulos',
            'bolsa:rrhh:nulos', v_comprobaciones
        );
        v_entrada := ROW(
            v_resumen, v_solicitud, true, v_analisis, 2,
            true, v_cobertura, 3, true, v_asignacion, 4, v_hitos
        );
        BEGIN
            PERFORM
                vec_contratacion_temporal
                    .canon_contenido_detalle_rrhh_v1(
                        '2026-07-26T08:01:00Z', v_entrada
                    );
            RAISE EXCEPTION 'array de comprobaciones hostil aceptado: %',
                v_caso;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            IF SQLERRM <> 'contenido de detalle RRHH inválido' THEN
                RAISE;
            END IF;
        END;
    END LOOP;
END
$compuestos_parcialmente_nulos$;

DO $catalogo_funciones$
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'codificar_texto_utf8_rrhh_v1',
               'decodificar_texto_utf8_rrhh_v1',
               'texto_instante_canonico_rrhh_v1',
               'encuadrar_valor_rrhh_v1',
               'canon_resumen_publicacion_rrhh_v1',
               'canon_resultado_consulta_rrhh_puro_v1',
               'canon_recibo_lectura_rrhh_v2',
               'huella_material_consumo_rrhh_v3',
               'canon_contenido_cuadro_rrhh_v1',
               'canon_contenido_detalle_rrhh_v1'
           ]::name[])
           AND funcion.provolatile = 'i'
           AND funcion.proisstrict
           AND NOT funcion.prosecdef
           AND NOT funcion.proleakproof
           AND funcion.proparallel = 's'
           AND funcion.proconfig = ARRAY['search_path=pg_catalog']
           AND funcion.proowner =
               'vec_contratacion_temporal_propietario'::regrole
    ) <> 10 THEN
        RAISE EXCEPTION 'contrato puro de funciones divergente';
    END IF;
END
$catalogo_funciones$;

RESET ROLE;
