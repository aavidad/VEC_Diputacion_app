\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
SET search_path = pg_catalog;

DO $vectores_contenido$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_entrada
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_canon bytea;
    v_huella text;
    v_resultado bytea;
    v_caso_coste integer;
BEGIN
    v_canon :=
        vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
            '2026-07-26T08:00:00Z'::timestamptz,
            ARRAY[]::vec_contratacion_temporal
                .resumen_publicacion_rrhh_v1[],
            false, ''::bytea
        );
    IF v_canon IS DISTINCT FROM
       vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
           'VEC-CT-CONTENIDO-CUADRO-RRHH-V1' || pg_catalog.chr(10)
           || '27:2026-07-26T08:00:00.000000Z' || pg_catalog.chr(10)
           || '1:0' || pg_catalog.chr(10)
           || '1:0' || pg_catalog.chr(10)
           || '0:' || pg_catalog.chr(10)
       )
       OR pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex') <>
          '568056c5d1a9b0651d2bc85f7dcc6e6dc3a71b12ffe8f831cb7ea5ffd51aa0c4'
    THEN
        RAISE EXCEPTION 'vector de cuadro vacío divergente';
    END IF;
    v_resultado :=
        vec_contratacion_temporal.canon_resultado_consulta_rrhh_puro_v1(
            ROW(
                'cuadro', '2026-07-26T08:00:00Z'::timestamptz, 0,
                '568056c5d1a9b0651d2bc85f7dcc6e6dc3a71b12ffe8f831cb7ea5ffd51aa0c4',
                ''
            )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1
        );
    IF pg_catalog.encode(pg_catalog.sha256(v_resultado), 'hex') <>
       'cb8ad45d7c31faa5100a249a840e66671b0de2319d23fc1c8878e56da7076ee0'
       OR pg_catalog.octet_length(v_resultado) <> 149 THEN
        RAISE EXCEPTION 'vector de resultado de cuadro divergente';
    END IF;

    v_resumen := ROW(
        'expediente:rrhh:minimizado',
        'organizacion:diputacion-granada',
        '2026/CT-MIN', 1, 'flujo:rrhh:minimizado', 1,
        pg_catalog.repeat('a', 64), 'solicitud', 'en_curso',
        'centro:rrhh:minimizado', 'categoria:rrhh:minimizada',
        '', '', '2026-07-26T08:00:00Z', '2026-07-26T08:00:00Z'
    );
    v_solicitud := ROW(
        'C2', 'sustitucion',
        '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z'
    );
    v_hitos := ARRAY[
        ROW(
            1, 1, 'actuacion.minimizada.1',
            '2026-07-26T08:00:00Z', '', 'solicitud',
            'pendiente', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1
    ];
    v_entrada := ROW(
        v_resumen, v_solicitud,
        false, NULL::vec_contratacion_temporal.analisis_operativo_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
        0,
        false, NULL::vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
        0, v_hitos
    );
    v_canon :=
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            '2026-07-26T08:01:00Z', v_entrada
        );
    v_huella := pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex');
    IF v_huella <>
       '8e63fb2710c43306f709e8236715537526b5831ba3db95470280cb069cbf136a'
       OR pg_catalog.octet_length(v_canon) <> 577 THEN
        RAISE EXCEPTION 'vector de detalle mínimo divergente: %/%',
            v_huella, pg_catalog.octet_length(v_canon);
    END IF;
    v_resultado :=
        vec_contratacion_temporal.canon_resultado_consulta_rrhh_puro_v1(
            ROW(
                'detalle', '2026-07-26T08:01:00Z'::timestamptz,
                1, v_huella, ''
            )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1
        );
    IF pg_catalog.encode(pg_catalog.sha256(v_resultado), 'hex') <>
       '0b7d78f6d34cd87f3da98fc32a0830ba953c83f36c2a7423fba9810baff78e31'
       OR pg_catalog.octet_length(v_resultado) <> 150 THEN
        RAISE EXCEPTION 'vector de resultado de detalle divergente';
    END IF;

    v_resumen := ROW(
        'expediente:rrhh:minimizado',
        'organizacion:diputacion-granada',
        '2026/CT-MIN', 4, 'flujo:rrhh:minimizado', 1,
        pg_catalog.repeat('a', 64), 'unidad_gestora', 'en_curso',
        'centro:rrhh:minimizado', 'categoria:rrhh:minimizada',
        'interinidad', 'unidad:rrhh:minimizada',
        '2026-07-26T07:57:00Z', '2026-07-26T08:00:00Z'
    );
    v_analisis := ROW(
        'interinidad', 'categoria:rrhh:minimizada', 'sustitucion',
        '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z',
        10000, 'no_requerida', true, 125000, 'EUR',
        'fuente:coste:minimizada'
    );
    v_cobertura := ROW(
        'bolsa', false, 'procedimiento:rrhh:minimizado',
        'bolsa:rrhh:minimizada',
        ARRAY[
            ROW('disponibilidad', 'afirmativa')
                ::vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
        ]
    );
    v_asignacion := ROW(
        'unidad:rrhh:minimizada', '2026-07-26T08:00:00Z', ''
    );
    v_hitos := ARRAY[
        ROW(
            1, 1, 'actuacion.minimizada.1',
            '2026-07-26T07:57:00Z', '', 'solicitud',
            'pendiente', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            2, 2, 'actuacion.minimizada.2',
            '2026-07-26T07:58:00Z', 'solicitud', 'gestion_bolsa',
            'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            3, 3, 'actuacion.minimizada.3',
            '2026-07-26T07:59:00Z', 'gestion_bolsa',
            'asignacion_unidad', 'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            4, 4, 'actuacion.minimizada.4',
            '2026-07-26T08:00:00Z', 'asignacion_unidad',
            'unidad_gestora', 'en_curso', 'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1
    ];
    v_entrada := ROW(
        v_resumen, v_solicitud, true, v_analisis, 2,
        true, v_cobertura, 3, true, v_asignacion, 4, v_hitos
    );
    v_canon :=
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            '2026-07-26T08:01:00Z', v_entrada
        );
    v_huella := pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex');
    IF v_huella <>
       '97b2d440c764090e452e51fb3623900a2cac78d97f337a42437b599ec6335e9b'
       OR pg_catalog.octet_length(v_canon) <> 1342 THEN
        RAISE EXCEPTION 'vector de detalle completo divergente: %/%',
            v_huella, pg_catalog.octet_length(v_canon);
    END IF;
    FOR v_caso_coste IN 1..3 LOOP
        v_analisis := ROW(
            'interinidad', 'categoria:rrhh:minimizada', 'sustitucion',
            '2026-08-26T00:00:00Z', '2026-09-26T00:00:00Z',
            10000, 'no_requerida', true,
            CASE v_caso_coste
                WHEN 1 THEN 0
                WHEN 2 THEN -1
                ELSE 125000
            END,
            CASE WHEN v_caso_coste = 3 THEN 'USD' ELSE 'EUR' END,
            'fuente:coste:minimizada'
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
            RAISE EXCEPTION 'coste inválido aceptado: %', v_caso_coste;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            IF SQLERRM <> 'contenido de detalle RRHH inválido' THEN
                RAISE;
            END IF;
        END;
    END LOOP;
END
$vectores_contenido$;

DO $vector_cuadro_cursor$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_canon bytea;
    v_huella text;
    v_cursor bytea := pg_catalog.decode(
        'af9613760f72635fbdb44a5a0a63c39f12af30f950a6ee5c971be188e89c4051',
        'hex'
    );
BEGIN
    v_resumen := ROW(
        'expediente:rrhh:001', 'organizacion:diputacion-granada',
        '2026/CT-001', 1, 'flujo:rrhh:001', 1,
        pg_catalog.repeat('a', 64), 'solicitud', 'en_curso',
        'centro:rrhh:001', 'categoria:rrhh:001',
        'interinidad', 'unidad:rrhh:001',
        '2026-07-26T07:00:00Z', '2026-07-26T07:30:00Z'
    );
    v_canon :=
        vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
            '2026-07-26T08:00:00Z',
            ARRAY[v_resumen], false, ''::bytea
        );
    v_huella := pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex');
    IF v_huella <>
       '7a235e6bbaa9bc265b09814ad08ffc4f35056954c939a6b465518b60acd74795'
       OR pg_catalog.octet_length(v_canon) <> 401 THEN
        RAISE EXCEPTION 'vector de cuadro sin cursor divergente: %/%',
            v_huella, pg_catalog.octet_length(v_canon);
    END IF;
    v_canon :=
        vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
            '2026-07-26T08:00:00Z',
            ARRAY[v_resumen], true, v_cursor
        );
    IF pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex') <>
       'acf18cd8e268f93f451f6ef6566e617c9128ba84c068b5b3124132f5f1ef5f07'
       OR pg_catalog.octet_length(v_canon) <> 434 THEN
        RAISE EXCEPTION 'vector de cuadro con cursor divergente';
    END IF;
END
$vector_cuadro_cursor$;

RESET ROLE;
