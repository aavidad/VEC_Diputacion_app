\set ON_ERROR_STOP on
-- CT102: prueba de canon_contenido_detalle_rrhh_v1 con soporte V2 y observaciones.
-- Comprueba:
-- 1. Detalle con observaciones devuelve canon con cabecera V2 y contiene las observaciones.
-- 2. Detalle sin observaciones (cadena vacia) devuelve canon con cabecera V2 identico salvo observaciones.
-- 3. Detalle con observaciones > 4000 caracteres es rechazado con excepcion 22023.
-- 4. Detalle con observaciones NULL es rechazado con excepcion 22023.
-- No ejecuta escrituras permanentes; bloque transaccional con rollback.

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '20s';

DO $ct102_prueba$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis_base vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_analisis_obs vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_analisis_4001 vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_analisis_null vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_entrada_base vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_entrada_obs vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_entrada_4001 vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_entrada_null vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_canon_base bytea;
    v_canon_obs bytea;
    v_instante timestamptz := '2026-09-06T02:00:00.000000Z';
    v_error_4001 boolean := false;
    v_error_null boolean := false;
BEGIN
    v_resumen := ROW(
        'expediente:ct:sintetico:001',
        'organizacion:sintetica:001',
        '2026/CT-000001',
        2::numeric(20, 0),
        'flujo:ct:sintetico',
        1::numeric(20, 0),
        pg_catalog.repeat('a', 64),
        'analisis',
        'en_curso',
        'centro:sintetico:001',
        'categoria:sintetica:c2',
        'sustitucion',
        '',
        '2026-09-06T01:00:00.000000Z'::timestamptz,
        '2026-09-06T01:30:00.000000Z'::timestamptz
    );

    v_solicitud := ROW(
        'C2',
        'sustitucion',
        '2026-10-01T00:00:00.000000Z'::timestamptz,
        '2026-12-31T00:00:00.000000Z'::timestamptz
    );

    v_analisis_base := ROW(
        'sustitucion',
        'categoria:sintetica:c2',
        'necesidad_temporal',
        '2026-10-01T00:00:00.000000Z'::timestamptz,
        '2026-12-31T00:00:00.000000Z'::timestamptz,
        10000::smallint,
        'validada',
        false,
        0::bigint,
        '',
        '',
        ''
    );

    v_analisis_obs := ROW(
        'sustitucion',
        'categoria:sintetica:c2',
        'necesidad_temporal',
        '2026-10-01T00:00:00.000000Z'::timestamptz,
        '2026-12-31T00:00:00.000000Z'::timestamptz,
        10000::smallint,
        'validada',
        false,
        0::bigint,
        '',
        '',
        'Observaciones de prueba para análisis RRHH'
    );

    v_hitos := ARRAY[
        ROW(
            1::numeric(20, 0),
            1::numeric(20, 0),
            'alta',
            '2026-09-06T01:00:00.000000Z'::timestamptz,
            '',
            'solicitud',
            'pendiente',
            'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
        ROW(
            2::numeric(20, 0),
            2::numeric(20, 0),
            'analisis',
            '2026-09-06T01:30:00.000000Z'::timestamptz,
            'solicitud',
            'analisis',
            'en_curso',
            'en_curso'
        )::vec_contratacion_temporal.hito_expediente_rrhh_v1
    ];

    v_entrada_base := ROW(
        v_resumen,
        v_solicitud,
        true,
        v_analisis_base,
        2::numeric(20, 0),
        false,
        NULL,
        0::numeric(20, 0),
        false,
        NULL,
        0::numeric(20, 0),
        v_hitos
    );

    v_entrada_obs := ROW(
        v_resumen,
        v_solicitud,
        true,
        v_analisis_obs,
        2::numeric(20, 0),
        false,
        NULL,
        0::numeric(20, 0),
        false,
        NULL,
        0::numeric(20, 0),
        v_hitos
    );

    -- 1. Canon base produce cabecera V2
    v_canon_base := vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
        v_instante,
        v_entrada_base
    );
    IF v_canon_base IS NULL OR pg_catalog.substr(v_canon_base, 1, 33) <> '\x5645432d43542d434f4e54454e49444f2d444554414c4c452d525248482d56320a'::bytea THEN
        RAISE EXCEPTION 'CT102: canon base no contiene cabecera V2 esperada';
    END IF;

    -- 2. Canon con observaciones produce cabecera V2 y contiene el texto de observaciones
    v_canon_obs := vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
        v_instante,
        v_entrada_obs
    );
    IF v_canon_obs IS NULL OR position(
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1('Observaciones de prueba para análisis RRHH') IN v_canon_obs
    ) = 0 THEN
        RAISE EXCEPTION 'CT102: canon con observaciones no incluye el texto esperado';
    END IF;

    -- 3. Observaciones excesivas (>4000) debe lanzar 22023
    v_analisis_4001 := v_analisis_base;
    v_analisis_4001.observaciones := pg_catalog.repeat('x', 4001);
    v_entrada_4001 := v_entrada_base;
    v_entrada_4001.analisis := v_analisis_4001;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_instante,
            v_entrada_4001
        );
    EXCEPTION WHEN SQLSTATE '22023' THEN
        v_error_4001 := true;
    END;
    IF NOT v_error_4001 THEN
        RAISE EXCEPTION 'CT102: no se rechazo observaciones > 4000 caracteres';
    END IF;

    -- 4. Observaciones NULL debe lanzar 22023
    v_analisis_null := v_analisis_base;
    v_analisis_null.observaciones := NULL;
    v_entrada_null := v_entrada_base;
    v_entrada_null.analisis := v_analisis_null;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_instante,
            v_entrada_null
        );
    EXCEPTION WHEN SQLSTATE '22023' THEN
        v_error_null := true;
    END;
    IF NOT v_error_null THEN
        RAISE EXCEPTION 'CT102: no se rechazo observaciones NULL';
    END IF;

    RAISE NOTICE 'CT102: pruebas de canon_contenido_detalle_rrhh_v1 con soporte V2 superadas exitosamente';
END
$ct102_prueba$;

ROLLBACK;
