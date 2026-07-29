\set ON_ERROR_STOP on

-- CT-000044: prueba focal de la única colección según corte del cuadro RRHH.
-- Se ejecuta como administrador dentro de una transacción que siempre revierte.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = 'replica';

CREATE TEMPORARY TABLE ct44_cuadro_control (
    corte_base numeric(20, 0) NOT NULL,
    corte_paginacion numeric(20, 0),
    actualizado_ultimo_pagina timestamptz(6),
    expediente_ultimo_pagina text
) ON COMMIT DROP;

-- El ordinal cero solo existe antes de la primera publicación global. La
-- primera página queda vacía y sin keyset; nunca habilita una continuación.
DO $corpus_vacio_corte_cero$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:ct44:sintetica',
            'organizacion',
            'organizacion:ct44:sintetica'
        )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1 :=
        ROW('', '', '', 100, '')
          ::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1 :=
        ROW(
            false, NULL, 0, 0, NULL, NULL,
            NULL, NULL, NULL, NULL, NULL
        )::vec_contratacion_temporal
          .estado_cursor_entrada_cuadro_rrhh_v1;
    v_material
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
BEGIN
    v_material :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_material.resumenes) <> 0
       OR pg_catalog.array_ndims(v_material.resumenes) IS NOT NULL
       OR v_material.hay_mas
       OR v_material.ultimo_actualizado_en IS NOT NULL
       OR v_material.ultimo_expediente_ref IS NOT NULL THEN
        RAISE EXCEPTION
            'el corte cero publicó filas, paginación o keyset';
    END IF;

    v_consulta.cursor := pg_catalog.repeat('A', 43);
    v_estado := ROW(
        true,
        'familia:cursor:rrhh:11111111111111111111111111111111',
        0,
        2,
        pg_catalog.repeat('f', 64),
        'acceso:rrhh:11111111111111111111111111111111',
        '2026-07-29T08:01:00.000000Z'::timestamptz,
        '2026-07-29T08:00:00.000000Z'::timestamptz,
        '2026-07-29T08:05:00.000000Z'::timestamptz,
        '2026-07-29T08:00:00.000000Z'::timestamptz,
        'expediente:ct44:anterior'
    )::vec_contratacion_temporal
      .estado_cursor_entrada_cuadro_rrhh_v1;
    BEGIN
        PERFORM
            vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
                v_alcance, v_consulta, v_estado
            );
        RAISE EXCEPTION 'continuación con corte cero aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
END
$corpus_vacio_corte_cero$;

INSERT INTO ct44_cuadro_control (corte_base)
SELECT COALESCE(pg_catalog.max(corte_global), 0) + 1
  FROM vec_contratacion_temporal.publicacion_version_rrhh;

DO $capacidad$
BEGIN
    IF (
        SELECT corte_base
          FROM ct44_cuadro_control
    ) > 9007199254740700::numeric THEN
        RAISE EXCEPTION 'sin ordinales sintéticos para probar cuadro CT44';
    END IF;
END
$capacidad$;

CREATE FUNCTION pg_temp.insertar_publicacion_cuadro_ct44(
    p_expediente_ref text,
    p_numero_visible text,
    p_version numeric,
    p_corte numeric,
    p_actualizado_en timestamptz,
    p_centro_ref text,
    p_unidad_ref text,
    p_estado_clave text,
    p_fase_clave text
)
RETURNS void
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
        expediente_ref,
        version,
        corte_global,
        organizacion_ref,
        numero_visible,
        flujo_ref,
        flujo_version,
        flujo_huella_sha256,
        fase_clave,
        estado_clave,
        centro_ref,
        categoria_ref,
        modalidad_clave,
        unidad_ref,
        creado_en,
        actualizado_en,
        agregado_huella_sha256,
        registrada_en
    ) VALUES (
        p_expediente_ref,
        p_version,
        p_corte,
        'organizacion:ct44:sintetica',
        p_numero_visible,
        'flujo:ct44:sintetico',
        1,
        pg_catalog.repeat('a', 64),
        p_fase_clave,
        p_estado_clave,
        p_centro_ref,
        'categoria:ct44:sintetica',
        'interinidad',
        p_unidad_ref,
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        p_actualizado_en,
        pg_catalog.repeat('b', 64),
        p_actualizado_en
    );
$funcion$;

-- Ciento una filas permiten demostrar límites 1/100 y la página final 101.
WITH base AS (
    SELECT corte_base FROM ct44_cuadro_control
), filas AS (
    SELECT indice,
           'expediente:ct44:paginacion:'
               || pg_catalog.lpad(indice::text, 3, '0') AS expediente_ref,
           '2026/PAG-'
               || pg_catalog.lpad(indice::text, 3, '0') AS numero_visible,
           base.corte_base + indice - 1 AS corte,
           '2026-07-29T09:00:00.000000Z'::timestamptz
               + indice * interval '1 microsecond' AS actualizado_en
      FROM base
     CROSS JOIN pg_catalog.generate_series(1, 101) indice
)
SELECT pg_temp.insertar_publicacion_cuadro_ct44(
           expediente_ref,
           numero_visible,
           1,
           corte,
           actualizado_en,
           'centro:ct44:paginacion',
           'unidad:ct44:paginacion',
           'en_curso',
           'solicitud'
       )
  FROM filas;

UPDATE ct44_cuadro_control
   SET corte_paginacion = corte_base + 100,
       actualizado_ultimo_pagina =
           '2026-07-29T09:00:00.000002Z'::timestamptz,
       expediente_ultimo_pagina =
           'expediente:ct44:paginacion:002';

-- Una versión antigua coincide con prefijo y centro; la última deja de
-- coincidir. Los filtros deben aplicarse después de elegir versión.
WITH base AS (
    SELECT corte_base + 110 AS corte FROM ct44_cuadro_control
)
SELECT pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:evolucion',
           '2026/ANTIGUA',
           1,
           corte,
           '2026-07-29T09:10:00.000000Z',
           'centro:ct44:antiguo',
           'unidad:ct44:antigua',
           'en_curso',
           'solicitud'
       )
  FROM base;
WITH base AS (
    SELECT corte_base + 111 AS corte FROM ct44_cuadro_control
)
SELECT pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:evolucion',
           '2026/NUEVA',
           2,
           corte,
           '2026-07-29T09:11:00.000000Z',
           'centro:ct44:nuevo',
           'unidad:ct44:nueva',
           'completado',
           'cierre'
       )
  FROM base;

-- El guion bajo debe conservarse como literal y no convertirse en comodín.
WITH base AS (
    SELECT corte_base FROM ct44_cuadro_control
)
SELECT pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:literal:guion',
           '2026/CT44_001',
           1,
           corte_base + 112,
           '2026-07-29T09:12:00.000000Z',
           'centro:ct44:literal',
           'unidad:ct44:literal',
           'en_curso',
           'analisis'
       ),
       pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:literal:letra',
           '2026/CT44X001',
           1,
           corte_base + 113,
           '2026-07-29T09:13:00.000000Z',
           'centro:ct44:literal',
           'unidad:ct44:literal',
           'en_curso',
           'analisis'
       )
  FROM base;

-- La colación C debe decidir empates, no la colación de la base.
WITH base AS (
    SELECT corte_base FROM ct44_cuadro_control
)
SELECT pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:empate:A',
           '2026/EMPATE-A',
           1,
           corte_base + 114,
           '2026-07-29T09:14:00.000000Z',
           'centro:ct44:empate',
           'unidad:ct44:empate',
           'en_curso',
           'analisis'
       ),
       pg_temp.insertar_publicacion_cuadro_ct44(
           'expediente:ct44:empate:a',
           '2026/EMPATE-a',
           1,
           corte_base + 115,
           '2026-07-29T09:14:00.000000Z',
           'centro:ct44:empate',
           'unidad:ct44:empate',
           'en_curso',
           'analisis'
       )
  FROM base;

DO $paginas$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:ct44:sintetica',
            'organizacion',
            'organizacion:ct44:sintetica'
        )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_resultado
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_control record;
BEGIN
    SELECT * INTO STRICT v_control FROM ct44_cuadro_control;
    v_estado := ROW(
        false, NULL, v_control.corte_paginacion, 0,
        NULL, NULL, NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;

    v_consulta := ROW(
        '2026/PAG-', '', '', 1, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1
       OR NOT v_resultado.hay_mas
       OR (v_resultado.resumenes[1]).expediente_ref <>
          'expediente:ct44:paginacion:101'
       OR v_resultado.ultimo_expediente_ref <>
          'expediente:ct44:paginacion:101' THEN
        RAISE EXCEPTION 'límite uno o centinela incorrectos';
    END IF;

    v_consulta := ROW(
        '2026/PAG-', '', '', 100, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 100
       OR NOT v_resultado.hay_mas
       OR (v_resultado.resumenes[1]).expediente_ref <>
          'expediente:ct44:paginacion:101'
       OR (v_resultado.resumenes[100]).expediente_ref <>
          v_control.expediente_ultimo_pagina
       OR v_resultado.ultimo_actualizado_en <>
          v_control.actualizado_ultimo_pagina THEN
        RAISE EXCEPTION 'página de cien o posición de continuación incorrectas';
    END IF;

    v_consulta := ROW(
        '2026/PAG-', '', '', 100, pg_catalog.repeat('A', 43)
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado := ROW(
        true,
        'familia:cursor:rrhh:11111111111111111111111111111111',
        v_control.corte_paginacion,
        2,
        pg_catalog.repeat('f', 64),
        'acceso:rrhh:11111111111111111111111111111111',
        '2026-07-29T10:00:00.000000Z'::timestamptz,
        '2026-07-29T09:59:00.000000Z'::timestamptz,
        '2026-07-29T10:04:00.000000Z'::timestamptz,
        v_control.actualizado_ultimo_pagina,
        v_control.expediente_ultimo_pagina
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1
       OR v_resultado.hay_mas
       OR (v_resultado.resumenes[1]).expediente_ref <>
          'expediente:ct44:paginacion:001'
       OR v_resultado.ultimo_expediente_ref <>
          'expediente:ct44:paginacion:001' THEN
        RAISE EXCEPTION 'fila 101 o continuación por posición incorrecta';
    END IF;
END
$paginas$;

DO $corte_y_filtros$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_resultado
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_base numeric(20, 0);
BEGIN
    SELECT corte_base INTO STRICT v_base FROM ct44_cuadro_control;
    v_alcance := ROW(
        'organizacion:ct44:sintetica',
        'organizacion',
        'organizacion:ct44:sintetica'
    )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;

    v_estado := ROW(
        false, NULL, v_base + 110, 0,
        NULL, NULL, NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_consulta := ROW(
        '2026/ANTIGUA', '', '', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1
       OR (v_resultado.resumenes[1]).version <> 1 THEN
        RAISE EXCEPTION 'corte anterior no recuperó versión uno';
    END IF;

    v_estado.corte_global := v_base + 111;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 0
       OR v_resultado.hay_mas
       OR v_resultado.ultimo_actualizado_en IS NOT NULL
       OR v_resultado.ultimo_expediente_ref IS NOT NULL THEN
        RAISE EXCEPTION 'filtro anterior resucitó versión ya superada';
    END IF;

    v_estado.corte_global := v_base + 115;
    v_consulta := ROW(
        '2026/CT44_', 'en_curso', 'analisis', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1
       OR (v_resultado.resumenes[1]).numero_visible <>
          '2026/CT44_001' THEN
        RAISE EXCEPTION 'guion bajo actuó como comodín';
    END IF;

    v_consulta := ROW(
        '2026/EMPATE', '', '', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 2
       OR (v_resultado.resumenes[1]).expediente_ref <>
          'expediente:ct44:empate:a'
       OR (v_resultado.resumenes[2]).expediente_ref <>
          'expediente:ct44:empate:A' THEN
        RAISE EXCEPTION 'desempate con colación C incorrecto';
    END IF;

    v_consulta := ROW(
        '2026/AUSENTE', '', '', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 0
       OR v_resultado.hay_mas
       OR v_resultado.ultimo_actualizado_en IS NOT NULL
       OR v_resultado.ultimo_expediente_ref IS NOT NULL THEN
        RAISE EXCEPTION 'resultado vacío no es canónico';
    END IF;
END
$corte_y_filtros$;

DO $ambitos_y_corte_posterior$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_resultado
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_base numeric(20, 0);
BEGIN
    SELECT corte_base INTO STRICT v_base FROM ct44_cuadro_control;
    v_consulta := ROW(
        '2026/NUEVA', 'completado', 'cierre', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado := ROW(
        false, NULL, v_base + 111, 0,
        NULL, NULL, NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;

    v_alcance := ROW(
        'organizacion:ct44:sintetica',
        'centro',
        'centro:ct44:nuevo'
    )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1 THEN
        RAISE EXCEPTION 'ámbito de centro no materializado';
    END IF;

    v_alcance := ROW(
        'organizacion:ct44:sintetica',
        'unidad_gestion',
        'unidad:ct44:nueva'
    )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1 THEN
        RAISE EXCEPTION 'ámbito de unidad no materializado';
    END IF;

    -- Simula una publicación que confirma después de capturar el corte. La
    -- misma familia debe permanecer estable aunque esa versión ya sea visible
    -- para una consulta nueva.
    PERFORM pg_temp.insertar_publicacion_cuadro_ct44(
        'expediente:ct44:posterior',
        '2026/POSTERIOR',
        1,
        v_base + 116,
        '2026-07-29T09:16:00.000000Z',
        'centro:ct44:nuevo',
        'unidad:ct44:nueva',
        'en_curso',
        'solicitud'
    );
    v_consulta := ROW(
        '2026/POSTERIOR', '', '', 10, ''
    )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 0 THEN
        RAISE EXCEPTION 'publicación posterior atravesó el corte capturado';
    END IF;
    v_estado.corte_global := v_base + 116;
    v_resultado :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            v_alcance, v_consulta, v_estado
        );
    IF pg_catalog.cardinality(v_resultado.resumenes) <> 1 THEN
        RAISE EXCEPTION 'corte nuevo no hizo visible publicación posterior';
    END IF;
END
$ambitos_y_corte_posterior$;

DO $estado_falla_cerrado$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:ct44:sintetica',
            'organizacion',
            'organizacion:ct44:sintetica'
        )::vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1 :=
        ROW('', '', '', 10, '')
          ::vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_base numeric(20, 0);
BEGIN
    SELECT corte_base INTO STRICT v_base FROM ct44_cuadro_control;
    v_estado := ROW(
        false, 'familia:cursor:rrhh:11111111111111111111111111111111',
        v_base, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    BEGIN
        PERFORM
            vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
                v_alcance, v_consulta, v_estado
            );
        RAISE EXCEPTION 'estado inicial parcial aceptado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;

    v_consulta.cursor := pg_catalog.repeat('A', 43);
    v_estado := ROW(
        true,
        'familia:cursor:rrhh:11111111111111111111111111111111',
        v_base, 2, pg_catalog.repeat('f', 64),
        'acceso:rrhh:11111111111111111111111111111111',
        NULL, NULL, NULL, NULL, NULL
    )::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    BEGIN
        PERFORM
            vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
                v_alcance, v_consulta, v_estado
            );
        RAISE EXCEPTION 'continuación parcial aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
END
$estado_falla_cerrado$;

-- El material publicado solo contiene los quince campos operativos tipados.
DO $minimizacion$
DECLARE
    v_definicion text;
    v_configuracion text[];
BEGIN
    SELECT pg_catalog.pg_get_functiondef(
               'vec_contratacion_temporal.'
               'materializar_cuadro_rrhh_v1('
               'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
               'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
               'vec_contratacion_temporal.'
               'estado_cursor_entrada_cuadro_rrhh_v1)'
                   ::pg_catalog.regprocedure
           )
      INTO STRICT v_definicion;
    SELECT funcion.proconfig
      INTO STRICT v_configuracion
      FROM pg_catalog.pg_proc funcion
     WHERE funcion.oid =
           'vec_contratacion_temporal.'
           'materializar_cuadro_rrhh_v1('
           'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
           'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
           'vec_contratacion_temporal.'
           'estado_cursor_entrada_cuadro_rrhh_v1)'
               ::pg_catalog.regprocedure;
    IF v_definicion ~* '\m(dni|correo|telefono|domicilio|persona_ref)\M'
       OR v_definicion ~
          '\m(registro_acceso_rrhh|familia_cursor_cuadro_rrhh|'
          'cursor_cuadro_rrhh|vec_autorizacion_atestada_v3)\M'
       OR NOT (
           'idle_in_transaction_session_timeout=6s' =
           ANY(v_configuracion)
       ) THEN
        RAISE EXCEPTION 'materializador contiene dato o autoridad ajenos';
    END IF;
END
$minimizacion$;

ROLLBACK;
