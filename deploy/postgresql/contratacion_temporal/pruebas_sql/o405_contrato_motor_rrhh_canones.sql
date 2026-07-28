\set ON_ERROR_STOP on

SET ROLE vec_contratacion_temporal_propietario;

DO $estructura$
DECLARE
    nombre text;
    oid_funcion oid;
    rechazo oid := pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    );
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 20
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 4
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.control_motor_consultas_rrhh_v1
         WHERE control AND version_esquema = 1
           AND catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'
           AND catalogo_huella_sha256 <> pg_catalog.repeat('0', 64)
    ) <> 1 THEN
        RAISE EXCEPTION 'barreras o control CT-000040 incorrectos';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class tabla
         WHERE tabla.oid = 'vec_contratacion_temporal.'
               'control_motor_consultas_rrhh_v1'::regclass
           AND tabla.relowner =
               'vec_contratacion_temporal_propietario'::regrole
           AND tabla.relrowsecurity AND tabla.relforcerowsecurity
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policies
         WHERE schemaname = 'vec_contratacion_temporal'
           AND tablename = 'control_motor_consultas_rrhh_v1'
           AND policyname = 'propietario_total'
           AND roles =
               ARRAY['vec_contratacion_temporal_propietario']::name[]
           AND cmd = 'ALL' AND qual = 'true' AND with_check = 'true'
    ) <> 1 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policies
         WHERE schemaname = 'vec_contratacion_temporal'
           AND tablename = 'control_motor_consultas_rrhh_v1'
    ) <> 1 THEN
        RAISE EXCEPTION 'propietario, RLS o política incorrectos';
    END IF;

    IF rechazo IS NULL OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid = 'vec_contratacion_temporal.'
               'control_motor_consultas_rrhh_v1'::regclass
           AND NOT disparador.tgisinternal
           AND disparador.tgenabled = 'O'
           AND disparador.tgfoid = rechazo
           AND (
               (
                   disparador.tgname =
                       'control_motor_consultas_rrhh_v1_inmutable'
                   AND disparador.tgtype = 27
               ) OR (
                   disparador.tgname =
                       'control_motor_consultas_rrhh_v1_no_truncar'
                   AND disparador.tgtype = 34
               )
           )
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid = 'vec_contratacion_temporal.'
               'control_motor_consultas_rrhh_v1'::regclass
           AND NOT disparador.tgisinternal
    ) <> 2 THEN
        RAISE EXCEPTION 'inmutabilidad del control incorrecta';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class tabla
          CROSS JOIN LATERAL pg_catalog.aclexplode(tabla.relacl) permiso
         WHERE tabla.oid = 'vec_contratacion_temporal.'
               'control_motor_consultas_rrhh_v1'::regclass
           AND permiso.grantee <>
               'vec_contratacion_temporal_propietario'::regrole
    ) THEN
        RAISE EXCEPTION 'tabla de control expuesta';
    END IF;

    FOREACH nombre IN ARRAY ARRAY[
        'alcance_consulta_rrhh_v1',
        'consulta_cuadro_rrhh_v1',
        'consulta_detalle_rrhh_v1',
        'evidencia_resultado_rrhh_v1'
    ]::text[] LOOP
        IF pg_catalog.has_type_privilege(
            'public', 'vec_contratacion_temporal.' || nombre, 'USAGE'
        ) OR pg_catalog.has_type_privilege(
            'vec_contratacion_temporal_consultor_rrhh',
            'vec_contratacion_temporal.' || nombre, 'USAGE'
        ) OR EXISTS (
            SELECT 1
              FROM pg_catalog.pg_type tipo
              CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) permiso
             WHERE tipo.oid =
                   ('vec_contratacion_temporal.' || nombre)::regtype
               AND permiso.grantee <>
                   'vec_contratacion_temporal_propietario'::regrole
        ) THEN
            RAISE EXCEPTION 'tipo % expuesto', nombre;
        END IF;
    END LOOP;

    FOREACH nombre IN ARRAY ARRAY[
        'json_rrhh_seguro_v1(jsonb)',
        'canon_alcance_rrhh_v1(' ||
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1)',
        'canon_consulta_cuadro_rrhh_v1(' ||
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1)',
        'canon_familia_cuadro_rrhh_v1(' ||
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1)',
        'canon_consulta_detalle_rrhh_v1(' ||
        'vec_contratacion_temporal.consulta_detalle_rrhh_v1)',
        'canon_resultado_consulta_rrhh_v1(' ||
        'vec_contratacion_temporal.evidencia_resultado_rrhh_v1)'
    ]::text[] LOOP
        oid_funcion := pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.' || nombre
        );
        IF oid_funcion IS NULL OR pg_catalog.has_function_privilege(
            'public', oid_funcion, 'EXECUTE'
        ) OR pg_catalog.has_function_privilege(
            'vec_contratacion_temporal_consultor_rrhh',
            oid_funcion, 'EXECUTE'
        ) OR EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc funcion
              JOIN pg_catalog.pg_roles rol
                ON rol.oid = funcion.proowner
             WHERE funcion.oid = oid_funcion
               AND (
                   rol.rolname <>
                       'vec_contratacion_temporal_propietario'
                   OR funcion.provolatile <> 'i'
                   OR NOT funcion.proisstrict
                   OR funcion.prosecdef
                   OR funcion.proparallel <> 's'
                   OR funcion.proconfig IS DISTINCT FROM
                      ARRAY['search_path=pg_catalog']::text[]
               )
        ) OR EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc funcion
              CROSS JOIN LATERAL
                   pg_catalog.aclexplode(funcion.proacl) permiso
             WHERE funcion.oid = oid_funcion
               AND permiso.grantee <>
                   'vec_contratacion_temporal_propietario'::regrole
        ) THEN
            RAISE EXCEPTION 'función % expuesta o no determinista', nombre;
        END IF;
    END LOOP;
END
$estructura$;

DO $canones$
DECLARE
    alcance_organizacion
        vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:diputacion-granada',
            'organizacion',
            'organizacion:diputacion-granada'
        );
    alcance_centro vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:diputacion-granada',
            'centro',
            'centro:rrhh:001'
        );
    alcance_unidad vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:diputacion-granada',
            'unidad_gestion',
            'unidad:seleccion:001'
        );
    pagina_uno vec_contratacion_temporal.consulta_cuadro_rrhh_v1 :=
        ROW('ÁREA_Ñ 2026/CT', 'en_curso', 'solicitud', 37, '');
    pagina_dos vec_contratacion_temporal.consulta_cuadro_rrhh_v1 :=
        ROW(
            'ÁREA_Ñ 2026/CT', 'en_curso', 'solicitud', 37,
            pg_catalog.translate(pg_catalog.rtrim(pg_catalog.encode(
                pg_catalog.decode(pg_catalog.repeat('ff', 32), 'hex'),
                'base64'
            ), E'=\n'), '+/', '-_')
        );
    detalle vec_contratacion_temporal.consulta_detalle_rrhh_v1 :=
        ROW('EXP-2026/0001', 7);
    resultado vec_contratacion_temporal.evidencia_resultado_rrhh_v1 :=
        ROW(
            'cuadro', '2026-07-28T10:20:30.123456Z'::timestamptz, 2,
            pg_catalog.repeat('a', 64), pg_catalog.repeat('b', 64)
        );
    huella bytea;
    nombre text;
BEGIN
    IF pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_alcance_rrhh_v1(
            alcance_organizacion
        )
    ), 'hex') <>
       'cc4772d9abe85886f10f5ee98e304ae190929de3602e69c8281bcd8e3ea7fd4b'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_alcance_rrhh_v1(
            alcance_centro
        )
    ), 'hex') <>
       'f864ac24878bf8cd93701254a8a64f27dfaf4702e1b44f6ee0ddf30177abe7d9'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_alcance_rrhh_v1(
            alcance_unidad
        )
    ), 'hex') <>
       '0e9140f36357f299b8f2606ff07e1e74e12cdb3cfe97839da8ad372ba733b6d0'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            pagina_uno
        )
    ), 'hex') <>
       'c145a15fa8ac964f779d805c159540e9cf71bb55340e78af2a7460c4f67b6e08'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
            pagina_uno
        )
    ), 'hex') <>
       'a52a2e97b7e5ad15c558d84348d8de2d629f0ed9234e586b72b23776ca63bc4d'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            pagina_dos
        )
    ), 'hex') <>
       '9c13a2c2a09e6ae7257f2d22cc1dda0566369ea57cf5bc25f30b703833c0a4a8'
    OR vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
        pagina_uno
    ) <> vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
        pagina_dos
    ) OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(detalle)
    ), 'hex') <>
       'd2f2434e133db0d0298d1ca20a6a5af1c9f95ea7c2c26020c95309836396bed4'
    OR pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
            resultado
        )
    ), 'hex') <>
       'd84005a4999413a65a2df596a8abb86e18d14973d8bfad9621d5974c02e8fcd5'
    THEN
        RAISE EXCEPTION 'un canon no coincide con su vector de oro';
    END IF;

    huella := pg_catalog.sha256(
        vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
            resultado
        )
    );
    resultado.total := 3;
    IF huella = pg_catalog.sha256(
        vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
            resultado
        )
    ) THEN
        RAISE EXCEPTION 'el total no altera el canon del resultado';
    END IF;

    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        ROW(
            'FILTRO_LITERAL_CON_GUION_BAJO', '', '', 1, ''
        )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    );
    FOREACH nombre IN ARRAY ARRAY[
        '', 'pendiente', 'en_curso', 'espera_externa',
        'completado', 'incidencia', 'cancelado'
    ]::text[] LOOP
        PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            ROW(
                '', nombre, '', 1, ''
            )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        );
    END LOOP;
END
$canones$;

DO $rechazos$
BEGIN
    BEGIN
        PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
            ROW(
                'organizacion:diputacion-granada',
                'organizacion',
                'organizacion:ajena'
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó alcance organizativo cruzado';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        IF SQLERRM <> 'alcance RRHH inválido' THEN RAISE; END IF;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
            ROW(
                'organizacion:diputacion-granada',
                'ambito_inventado',
                'centro:rrhh:001'
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó clase de ámbito no gobernada';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            ROW(
                'Área 100%', '', '', 20, ''
            )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó el comodín %%';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        IF SQLERRM <> 'consulta RRHH inválida' THEN RAISE; END IF;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            ROW(
                '', '', '', NULL, ''
            )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó un límite nulo';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            ROW(
                '', '', '', 20, pg_catalog.repeat('a', 43)
            )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó un cursor no canónico';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            ROW(
                'EXP-2026/0001', NULL
            )::vec_contratacion_temporal.consulta_detalle_rrhh_v1
        );
        RAISE EXCEPTION 'se aceptó una versión nula';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
            ROW(
                'detalle', pg_catalog.clock_timestamp(), 1,
                pg_catalog.repeat('a', 64), pg_catalog.repeat('b', 64)
            )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1
        );
        RAISE EXCEPTION 'detalle aceptó cursor';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        IF SQLERRM <> 'resultado RRHH inválido' THEN RAISE; END IF;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
            ROW(
                'cuadro', pg_catalog.clock_timestamp(), 0,
                pg_catalog.repeat('a', 64), pg_catalog.repeat('b', 64)
            )::vec_contratacion_temporal.evidencia_resultado_rrhh_v1
        );
        RAISE EXCEPTION 'resultado vacío aceptó cursor';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
END
$rechazos$;

DO $limites_json$
DECLARE
    anidado jsonb := '0'::jsonb;
    indice integer;
    matriz jsonb;
BEGIN
    IF NOT vec_contratacion_temporal.json_rrhh_seguro_v1(
        '{"resultado":[1,2,3]}'::jsonb
    ) OR vec_contratacion_temporal.json_rrhh_seguro_v1(
        pg_catalog.to_jsonb(pg_catalog.repeat('x', 262145))
    ) THEN
        RAISE EXCEPTION 'límite de tamaño JSON incorrecto';
    END IF;
    FOR indice IN 1..24 LOOP
        anidado := pg_catalog.jsonb_build_array(anidado);
    END LOOP;
    IF vec_contratacion_temporal.json_rrhh_seguro_v1(anidado) THEN
        RAISE EXCEPTION 'se aceptó profundidad JSON superior a 24';
    END IF;
    SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(valor))
      INTO matriz
      FROM pg_catalog.generate_series(1, 16384) valor;
    IF vec_contratacion_temporal.json_rrhh_seguro_v1(matriz) THEN
        RAISE EXCEPTION 'se aceptaron más de 16.384 nodos JSON';
    END IF;
END
$limites_json$;

DO $inmutabilidad$
BEGIN
    BEGIN
        UPDATE vec_contratacion_temporal.control_motor_consultas_rrhh_v1
           SET version_esquema = 1
         WHERE control;
        RAISE EXCEPTION 'control mutable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal.control_motor_consultas_rrhh_v1;
        RAISE EXCEPTION 'control truncable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$inmutabilidad$;

RESET ROLE;
