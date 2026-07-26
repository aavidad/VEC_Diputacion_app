\set ON_ERROR_STOP on

\if :preparar
DO $preparacion$
BEGIN
    IF pg_catalog.to_regnamespace(
        'vec_o405_publicacion_prueba'
    ) IS NOT NULL THEN
        RAISE EXCEPTION 'fixture C2-C ya instalado';
    END IF;
END
$preparacion$;
CREATE SCHEMA vec_o405_publicacion_prueba AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA vec_o405_publicacion_prueba FROM PUBLIC;

CREATE FUNCTION vec_o405_publicacion_prueba.agregado(
    p_expediente_ref text,
    p_numero_visible text,
    p_version numeric,
    p_variante text
)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_agregado jsonb;
BEGIN
    v_agregado := pg_catalog.jsonb_build_object(
        'referencia', p_expediente_ref,
        'organizacion_ref', 'organizacion:publicacion:rrhh',
        'numero_visible', p_numero_visible,
        'version', p_version,
        'flujo', pg_catalog.jsonb_build_object(
            'definicion_ref', 'flujo:contratacion:publicacion',
            'version', 3,
            'huella_sha256', pg_catalog.repeat('a', 64)
        ),
        'fase_actual', 'analisis_rrhh',
        'estado_actual', 'en_curso',
        'solicitud', pg_catalog.jsonb_build_object(
            'centro_ref', 'centro:publicacion:rrhh',
            'categoria_ref', 'categoria:solicitud'
        ),
        'creado_en', '2026-01-01T00:00:00.000000Z',
        'actualizado_en', '2026-01-02T00:00:00.000000Z',
        'actuaciones', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'secuencia', p_version,
                'version_expediente', p_version,
                'accion_clave', 'analisis.registrar',
                'actor_ref', 'actor:sintetico:rrhh',
                'realizada_en', '2026-01-02T00:00:00.000000Z'
            )
        )
    );
    IF p_variante = 'completa' THEN
        v_agregado := v_agregado
            || pg_catalog.jsonb_build_object(
                'analisis', pg_catalog.jsonb_build_object(
                    'categoria_ref', 'categoria:analisis',
                    'modalidad_clave', 'interinidad'
                ),
                'asignacion', pg_catalog.jsonb_build_object(
                    'unidad_ref', 'unidad:seleccion:rrhh'
                )
            );
    ELSIF p_variante <> 'base' THEN
        RAISE EXCEPTION 'variante sintética desconocida';
    END IF;
    RETURN v_agregado;
END
$funcion$;

CREATE FUNCTION vec_o405_publicacion_prueba.crear_alta(
    p_expediente_ref text,
    p_numero_visible text
)
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
        p_expediente_ref,
        'reserva:' || p_expediente_ref,
        p_numero_visible,
        'organizacion:publicacion:rrhh',
        'actor:sintetico:rrhh',
        'perfil:sintetico:rrhh',
        'decision:' || p_expediente_ref,
        'efecto:' || p_expediente_ref,
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to('efecto:' || p_expediente_ref, 'UTF8')
        ), 'hex'),
        '2026-01-01T00:00:00Z'::timestamptz,
        'cnf_ct_' || pg_catalog.substr(pg_catalog.encode(
            pg_catalog.sha256(pg_catalog.convert_to(
                'confirmacion:' || p_expediente_ref, 'UTF8'
            )), 'hex'), 1, 32)
    )
    ON CONFLICT (expediente_ref) DO NOTHING;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_o405_publicacion_prueba.agregado(text,text,numeric,text),
    vec_o405_publicacion_prueba.crear_alta(text,text)
FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_o405_publicacion_prueba
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
    vec_o405_publicacion_prueba.agregado(text,text,numeric,text)
    TO vec_contratacion_temporal_propietario;

SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:pub:A', '2026/A'
);
SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:pub:a', '2026/a'
);
SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:pub:empate', '2026/empate'
);

SET session_replication_role = replica;
WITH filas(expediente_ref, numero_visible, version, variante) AS (
    VALUES
      ('expediente:pub:A', '2026/A', 1::numeric, 'base'),
      ('expediente:pub:a', '2026/a', 1::numeric, 'completa'),
      ('expediente:pub:empate', '2026/empate', 1::numeric, 'base'),
      ('expediente:pub:empate', '2026/empate', 2::numeric, 'completa')
), datos AS (
    SELECT filas.*, vec_o405_publicacion_prueba.agregado(
        expediente_ref, numero_visible, version, variante
    ) AS agregado
    FROM filas
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral (
    expediente_ref, version, agregado_json,
    agregado_json_huella_sha256, prueba_canonica,
    prueba_huella_sha256, flujo_ref, flujo_version,
    flujo_huella_sha256, fase_clave, estado, origen_version,
    operacion_ref, registrada_en
)
SELECT expediente_ref, version, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')
       ), 'hex'),
       pg_catalog.convert_to(pg_catalog.repeat(
           'prueba:' || expediente_ref || ':' || version::text || ':', 8
       ), 'UTF8'),
       pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           pg_catalog.repeat(
               'prueba:' || expediente_ref || ':' || version::text || ':', 8
           ), 'UTF8'
       )), 'hex'),
       'flujo:contratacion:publicacion', 3, pg_catalog.repeat('a', 64),
       'analisis_rrhh', 'en_curso',
       CASE WHEN version = 1 THEN 'alta_o2' ELSE 'analisis_o3' END,
       'operacion:publicacion:' || pg_catalog.substr(
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               expediente_ref || ':' || version::text, 'UTF8'
           )), 'hex'), 1, 24
       ),
       '2026-01-03T00:00:00Z'::timestamptz
FROM datos;
SET session_replication_role = origin;
\else

DO $estructura$
DECLARE
    v_tabla text;
    v_rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 17
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control AND corte_base = 4 AND ultimo_corte = 4
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.publicacion_version_rrhh
    ) <> 4 THEN
        RAISE EXCEPTION 'backfill C2-C incompleto';
    END IF;
    IF (
        SELECT pg_catalog.string_agg(
            expediente_ref || '/' || version::text || '=' ||
            corte_global::text, ',' ORDER BY corte_global
        )
        FROM vec_contratacion_temporal.publicacion_version_rrhh
    ) <> 'expediente:pub:A/1=1,expediente:pub:a/1=2,'
          'expediente:pub:empate/1=3,expediente:pub:empate/2=4' THEN
        RAISE EXCEPTION 'orden C o desempate por versión no determinista';
    END IF;
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_publicacion_rrhh', 'publicacion_version_rrhh'
    ]::text[] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class tabla
              JOIN pg_catalog.pg_namespace esquema
                ON esquema.oid = tabla.relnamespace
              JOIN pg_catalog.pg_roles propietario
                ON propietario.oid = tabla.relowner
             WHERE esquema.nspname = 'vec_contratacion_temporal'
               AND tabla.relname = v_tabla
               AND propietario.rolname =
                   'vec_contratacion_temporal_propietario'
               AND tabla.relrowsecurity AND tabla.relforcerowsecurity
        ) OR (
            SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_policies
             WHERE schemaname = 'vec_contratacion_temporal'
               AND tablename = v_tabla
               AND policyname = 'propietario_total'
               AND roles =
                   ARRAY['vec_contratacion_temporal_propietario']::name[]
        ) <> 1 THEN
            RAISE EXCEPTION 'propiedad o RLS incorrectos en %', v_tabla;
        END IF;
    END LOOP;
    FOREACH v_rol IN ARRAY ARRAY[
        'public', 'vec_contratacion_temporal_migrador',
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal_gobernador',
        'vec_contratacion_temporal_confirmador_cobertura',
        'vec_contratacion_temporal_lector_resultado_cobertura',
        'vec_contratacion_temporal_consultor_rrhh'
    ]::text[] LOOP
        IF pg_catalog.has_table_privilege(
               v_rol,
               'vec_contratacion_temporal.control_publicacion_rrhh',
               'SELECT,INSERT,UPDATE,DELETE'
           ) OR pg_catalog.has_table_privilege(
               v_rol,
               'vec_contratacion_temporal.publicacion_version_rrhh',
               'SELECT,INSERT,UPDATE,DELETE'
           ) OR pg_catalog.has_function_privilege(
               v_rol,
               'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)',
               'EXECUTE'
           ) OR pg_catalog.has_function_privilege(
               v_rol,
               'vec_contratacion_temporal.publicar_version_rrhh_v1()',
               'EXECUTE'
           ) THEN
            RAISE EXCEPTION 'ACL excesiva para %', v_rol;
        END IF;
    END LOOP;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_roles propietario
            ON propietario.oid = funcion.proowner
         WHERE funcion.oid =
               'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'
                   ::regprocedure
           AND propietario.rolname =
               'vec_contratacion_temporal_propietario'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_roles propietario
            ON propietario.oid = funcion.proowner
         WHERE funcion.oid =
               'vec_contratacion_temporal.publicar_version_rrhh_v1()'
                   ::regprocedure
           AND propietario.rolname =
               'vec_contratacion_temporal_propietario'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)',
        'EXECUTE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'vec_contratacion_temporal.publicar_version_rrhh_v1()',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'propiedad de funciones C2-C incorrecta';
    END IF;
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class indice
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = indice.relnamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND indice.relname = ANY (ARRAY[
             'publicacion_rrhh_organizacion_orden_idx',
             'publicacion_rrhh_centro_orden_idx',
             'publicacion_rrhh_unidad_orden_idx',
             'publicacion_rrhh_filtros_idx'
           ])
           AND pg_catalog.pg_get_indexdef(indice.oid) LIKE '%COLLATE "C"%'
    ) <> 4 OR pg_catalog.pg_get_expr(
        (
            SELECT indice.indpred
              FROM pg_catalog.pg_index indice
             WHERE indice.indexrelid =
                   'vec_contratacion_temporal.publicacion_rrhh_unidad_orden_idx'
                       ::regclass
        ),
        'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
    ) IS DISTINCT FROM '(unidad_ref IS NOT NULL)' OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE NOT tgisinternal AND tgenabled = 'O'
           AND tgname = ANY (ARRAY[
             'expediente_version_integral_publicar_rrhh',
             'publicacion_version_rrhh_inmutable',
             'publicacion_version_rrhh_no_truncar'
           ])
    ) <> 3 THEN
        RAISE EXCEPTION 'índices C o disparadores incompletos';
    END IF;
END
$estructura$;

SET ROLE vec_contratacion_temporal_propietario;
DO $extraccion$
DECLARE
    v_base record;
    v_completa record;
    v_historia record;
    v_alta record;
    v_agregado jsonb;
    v_caso text;
BEGIN
    SELECT extraida.* INTO STRICT v_base
      FROM vec_contratacion_temporal.expediente_version_integral historia
      JOIN vec_contratacion_temporal.expediente_alta alta USING (
          expediente_ref
      )
      CROSS JOIN LATERAL
      vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
          historia.expediente_ref, historia.version,
          historia.agregado_json, historia.agregado_json_huella_sha256,
          historia.flujo_ref, historia.flujo_version,
          historia.flujo_huella_sha256, historia.fase_clave,
          historia.estado, historia.registrada_en,
          alta.organizacion_ref, alta.numero_visible
      ) extraida
     WHERE historia.expediente_ref = 'expediente:pub:A'
       AND historia.version = 1;
    IF v_base.centro_ref <> 'centro:publicacion:rrhh'
       OR v_base.categoria_ref <> 'categoria:solicitud'
       OR v_base.modalidad_clave IS NOT NULL
       OR v_base.unidad_ref IS NOT NULL
       OR v_base.organizacion_ref <> 'organizacion:publicacion:rrhh'
       OR v_base.numero_visible <> '2026/A'
       OR v_base.creado_en <>
          '2026-01-01T00:00:00Z'::timestamptz
       OR v_base.actualizado_en <>
          '2026-01-02T00:00:00Z'::timestamptz THEN
        RAISE EXCEPTION 'extracción base inexacta';
    END IF;
    SELECT extraida.* INTO STRICT v_completa
      FROM vec_contratacion_temporal.expediente_version_integral historia
      JOIN vec_contratacion_temporal.expediente_alta alta USING (
          expediente_ref
      )
      CROSS JOIN LATERAL
      vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
          historia.expediente_ref, historia.version,
          historia.agregado_json, historia.agregado_json_huella_sha256,
          historia.flujo_ref, historia.flujo_version,
          historia.flujo_huella_sha256, historia.fase_clave,
          historia.estado, historia.registrada_en,
          alta.organizacion_ref, alta.numero_visible
      ) extraida
     WHERE historia.expediente_ref = 'expediente:pub:a'
       AND historia.version = 1;
    IF v_completa.categoria_ref <> 'categoria:analisis'
       OR v_completa.modalidad_clave <> 'interinidad'
       OR v_completa.unidad_ref <> 'unidad:seleccion:rrhh' THEN
        RAISE EXCEPTION 'extracción de análisis/asignación inexacta';
    END IF;

    SELECT historia.*, alta.organizacion_ref, alta.numero_visible
      INTO STRICT v_historia
      FROM vec_contratacion_temporal.expediente_version_integral historia
      JOIN vec_contratacion_temporal.expediente_alta alta USING (
          expediente_ref
      )
     WHERE historia.expediente_ref = 'expediente:pub:a'
       AND historia.version = 1;
    FOREACH v_caso IN ARRAY ARRAY[
        'analisis_null', 'analisis_vacio', 'analisis_array',
        'analisis_texto', 'asignacion_null', 'asignacion_vacia',
        'asignacion_array', 'asignacion_texto'
    ]::text[] LOOP
        v_agregado := CASE v_caso
          WHEN 'analisis_null' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{analisis}', 'null'::jsonb)
          WHEN 'analisis_vacio' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{analisis}', '{}'::jsonb)
          WHEN 'analisis_array' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{analisis}', '[]'::jsonb)
          WHEN 'analisis_texto' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{analisis}', '"x"'::jsonb)
          WHEN 'asignacion_null' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{asignacion}', 'null'::jsonb)
          WHEN 'asignacion_vacia' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{asignacion}', '{}'::jsonb)
          WHEN 'asignacion_array' THEN pg_catalog.jsonb_set(
              v_historia.agregado_json, '{asignacion}', '[]'::jsonb)
          ELSE pg_catalog.jsonb_set(
              v_historia.agregado_json, '{asignacion}', '"x"'::jsonb)
        END;
        BEGIN
            PERFORM 1 FROM
              vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
                v_historia.expediente_ref, v_historia.version,
                v_agregado, pg_catalog.encode(pg_catalog.sha256(
                    pg_catalog.convert_to(v_agregado::text, 'UTF8')
                ), 'hex'), v_historia.flujo_ref,
                v_historia.flujo_version, v_historia.flujo_huella_sha256,
                v_historia.fase_clave, v_historia.estado,
                v_historia.registrada_en, v_historia.organizacion_ref,
                v_historia.numero_visible);
            RAISE EXCEPTION 'caso hostil aceptado: %', v_caso;
        EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
        END;
    END LOOP;
    FOREACH v_caso IN ARRAY ARRAY[
        'version', 'referencia', 'organizacion', 'numero', 'flujo_ref',
        'flujo_version', 'flujo_huella', 'fase', 'estado', 'huella'
    ]::text[] LOOP
        BEGIN
            PERFORM 1 FROM
              vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
                CASE WHEN v_caso = 'referencia' THEN 'expediente:otro'
                     ELSE v_historia.expediente_ref END,
                CASE WHEN v_caso = 'version' THEN 2
                     ELSE v_historia.version END,
                v_historia.agregado_json,
                CASE WHEN v_caso = 'huella' THEN pg_catalog.repeat('0',64)
                     ELSE v_historia.agregado_json_huella_sha256 END,
                CASE WHEN v_caso = 'flujo_ref' THEN 'flujo:otro'
                     ELSE v_historia.flujo_ref END,
                CASE WHEN v_caso = 'flujo_version' THEN 4
                     ELSE v_historia.flujo_version END,
                CASE WHEN v_caso = 'flujo_huella' THEN pg_catalog.repeat('b',64)
                     ELSE v_historia.flujo_huella_sha256 END,
                CASE WHEN v_caso = 'fase' THEN 'fase_distinta'
                     ELSE v_historia.fase_clave END,
                CASE WHEN v_caso = 'estado' THEN 'cancelado'
                     ELSE v_historia.estado END,
                v_historia.registrada_en,
                CASE WHEN v_caso = 'organizacion' THEN 'organizacion:otra'
                     ELSE v_historia.organizacion_ref END,
                CASE WHEN v_caso = 'numero' THEN '2026/otro'
                     ELSE v_historia.numero_visible END);
            RAISE EXCEPTION 'divergencia aceptada: %', v_caso;
        EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
        END;
    END LOOP;
END
$extraccion$;
RESET ROLE;

SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:pub:nuevo', '2026/nuevo'
);
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
WITH datos AS (
    SELECT vec_o405_publicacion_prueba.agregado(
        'expediente:pub:nuevo', '2026/nuevo', 1, 'completa'
    ) AS agregado
), prueba AS (
    SELECT agregado, pg_catalog.convert_to(
        pg_catalog.repeat('prueba:nuevo:', 16), 'UTF8'
    ) AS bytes FROM datos
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral
SELECT 'expediente:pub:nuevo', 1, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')), 'hex'),
       bytes, pg_catalog.encode(pg_catalog.sha256(bytes), 'hex'),
       'flujo:contratacion:publicacion', 3, pg_catalog.repeat('a',64),
       'analisis_rrhh', 'en_curso', 'analisis_o3',
       'operacion:publicacion:nuevo',
       '2026-01-03T00:00:00Z'::timestamptz
FROM prueba;
COMMIT;

DO $primer_insert$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE expediente_ref = 'expediente:pub:nuevo'
           AND version = 1 AND corte_global = 5
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control AND corte_base = 4 AND ultimo_corte = 5
    ) THEN
        RAISE EXCEPTION 'primer INSERT no publicó base+1';
    END IF;
END
$primer_insert$;

SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:pub:rollback', '2026/rollback'
);
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
WITH datos AS (
    SELECT vec_o405_publicacion_prueba.agregado(
        'expediente:pub:rollback', '2026/rollback', 1, 'base'
    ) AS agregado
), prueba AS (
    SELECT agregado, pg_catalog.convert_to(
        pg_catalog.repeat('prueba:rollback:', 12), 'UTF8'
    ) AS bytes FROM datos
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral
SELECT 'expediente:pub:rollback', 1, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')), 'hex'),
       bytes, pg_catalog.encode(pg_catalog.sha256(bytes), 'hex'),
       'flujo:contratacion:publicacion', 3, pg_catalog.repeat('a',64),
       'analisis_rrhh', 'en_curso', 'analisis_o3',
       'operacion:publicacion:rollback',
       '2026-01-03T00:00:00Z'::timestamptz
FROM prueba;
ROLLBACK;

DO $rollback$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
         WHERE expediente_ref = 'expediente:pub:rollback'
    ) OR EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE expediente_ref = 'expediente:pub:rollback'
    ) OR NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control AND ultimo_corte = 5
    ) THEN
        RAISE EXCEPTION 'rollback dejó fila u ordinal';
    END IF;
END
$rollback$;

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
WITH datos AS (
    SELECT vec_o405_publicacion_prueba.agregado(
        'expediente:pub:rollback', '2026/rollback', 1, 'base'
    ) AS agregado
), prueba AS (
    SELECT agregado, pg_catalog.convert_to(
        pg_catalog.repeat('prueba:rollback:', 12), 'UTF8'
    ) AS bytes FROM datos
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral
SELECT 'expediente:pub:rollback', 1, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')), 'hex'),
       bytes, pg_catalog.encode(pg_catalog.sha256(bytes), 'hex'),
       'flujo:contratacion:publicacion', 3, pg_catalog.repeat('a',64),
       'analisis_rrhh', 'en_curso', 'analisis_o3',
       'operacion:publicacion:rollback',
       '2026-01-03T00:00:00Z'::timestamptz
FROM prueba;
COMMIT;

SET ROLE vec_contratacion_temporal_propietario;
DO $inmutabilidad$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE expediente_ref = 'expediente:pub:rollback'
           AND corte_global = 6
    ) THEN
        RAISE EXCEPTION 'ordinal de rollback no fue reutilizado';
    END IF;
    BEGIN
        UPDATE vec_contratacion_temporal.publicacion_version_rrhh
           SET centro_ref = 'centro:adulterado'
         WHERE corte_global = 1;
        RAISE EXCEPTION 'UPDATE de publicación aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE corte_global = 1;
        RAISE EXCEPTION 'DELETE de publicación aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal.publicacion_version_rrhh;
        RAISE EXCEPTION 'TRUNCATE de publicación aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$inmutabilidad$;
RESET ROLE;
\endif
