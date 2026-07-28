\set ON_ERROR_STOP on

DO $estructura$
DECLARE
    v_rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 21
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 5
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
           AND restriccion.conname =
               'publicacion_version_rrhh_estado_clave_valido'
           AND restriccion.contype = 'c'
           AND restriccion.convalidated
           AND restriccion.conenforced
           AND restriccion.conislocal
           AND NOT restriccion.connoinherit
           AND NOT restriccion.condeferrable
           AND NOT restriccion.condeferred
           AND pg_catalog.pg_get_constraintdef(
                   restriccion.oid, false
               ) = $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
    ) <> 1 OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_vocabulario_estados_publicacion_rrhh_v1
         WHERE control
           AND version_esquema = 1
           AND restriccion_nombre =
               'publicacion_version_rrhh_estado_clave_valido'
           AND restriccion_definicion =
               $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
           AND restriccion_validada
    ) <> 1 THEN
        RAISE EXCEPTION 'estructura o manifiesto CT-000041A incompleto';
    END IF;

    FOREACH v_rol IN ARRAY ARRAY[
        'public',
        'vec_contratacion_temporal_migrador',
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal_gobernador',
        'vec_contratacion_temporal_confirmador_cobertura',
        'vec_contratacion_temporal_lector_resultado_cobertura',
        'vec_contratacion_temporal_consultor_rrhh'
    ]::text[] LOOP
        IF pg_catalog.has_table_privilege(
            v_rol,
            'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1',
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) OR pg_catalog.has_table_privilege(
            v_rol,
            'vec_contratacion_temporal.publicacion_version_rrhh',
            'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION 'ACL excesiva para %', v_rol;
        END IF;
    END LOOP;
END
$estructura$;

CREATE TEMP TABLE fixture_estados_publicacion_rrhh (
    LIKE vec_contratacion_temporal.publicacion_version_rrhh
    INCLUDING ALL
);

WITH base AS (
    SELECT *
      FROM vec_contratacion_temporal.publicacion_version_rrhh
     ORDER BY corte_global
     LIMIT 1
), estados(estado_clave, orden) AS (
    VALUES
        ('pendiente'::text, 1::numeric),
        ('en_curso', 2),
        ('espera_externa', 3),
        ('completado', 4),
        ('incidencia', 5),
        ('cancelado', 6)
)
INSERT INTO fixture_estados_publicacion_rrhh (
    expediente_ref, version, corte_global, organizacion_ref,
    numero_visible, flujo_ref, flujo_version, flujo_huella_sha256,
    fase_clave, estado_clave, centro_ref, categoria_ref,
    modalidad_clave, unidad_ref, creado_en, actualizado_en,
    agregado_huella_sha256, registrada_en
)
SELECT base.expediente_ref || ':ct000041:' || estados.orden::text,
       base.version,
       9000000000000000::numeric + estados.orden,
       base.organizacion_ref,
       '2026/C' || pg_catalog.lpad(estados.orden::text, 2, '0'),
       base.flujo_ref,
       base.flujo_version,
       base.flujo_huella_sha256,
       base.fase_clave,
       estados.estado_clave,
       base.centro_ref,
       base.categoria_ref,
       base.modalidad_clave,
       base.unidad_ref,
       base.creado_en,
       base.actualizado_en,
       base.agregado_huella_sha256,
       base.registrada_en
  FROM base
 CROSS JOIN estados;

DO $filtros$
DECLARE
    v_estado text;
BEGIN
    IF (
        SELECT pg_catalog.array_agg(
            estado_clave ORDER BY corte_global
        )
          FROM fixture_estados_publicacion_rrhh
    ) <> ARRAY[
        'pendiente', 'en_curso', 'espera_externa',
        'completado', 'incidencia', 'cancelado'
    ]::text[] THEN
        RAISE EXCEPTION 'la publicación no conserva los seis estados exactos';
    END IF;

    FOREACH v_estado IN ARRAY ARRAY[
        'pendiente', 'en_curso', 'espera_externa',
        'completado', 'incidencia', 'cancelado'
    ]::text[] LOOP
        IF (
            SELECT pg_catalog.count(*)
              FROM fixture_estados_publicacion_rrhh
             WHERE estado_clave = v_estado
        ) <> 1 THEN
            RAISE EXCEPTION 'filtro de estado inexacto: %', v_estado;
        END IF;
        PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            ROW(
                '', v_estado, '', 100, ''
            )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        );
    END LOOP;

    -- Vacío es el valor canónico que pide «todos los estados» en consultas.
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        ROW(
            '', '', '', 100, ''
        )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    );
    PERFORM vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
        ROW(
            '', '', '', 100, ''
        )::vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    );

    IF EXISTS (
        SELECT 1
          FROM fixture_estados_publicacion_rrhh
         WHERE estado_clave IN (
             'PENDIENTE', ' pendiente', 'incidencia '
         )
    ) THEN
        RAISE EXCEPTION 'el filtro normalizó un estado';
    END IF;
END
$filtros$;

DO $rechazos$
DECLARE
    v_estado text;
BEGIN
    FOREACH v_estado IN ARRAY ARRAY[
        '',
        'inventado',
        'PENDIENTE',
        ' pendiente',
        'incidencia ',
        'espera_externá',
        'cancelado_extra'
    ]::text[] LOOP
        BEGIN
            UPDATE fixture_estados_publicacion_rrhh
               SET estado_clave = v_estado
             WHERE corte_global = 9000000000000001::numeric;
            RAISE EXCEPTION 'estado inválido aceptado: %', v_estado;
        EXCEPTION WHEN check_violation THEN NULL;
        END;

        BEGIN
            INSERT INTO fixture_estados_publicacion_rrhh (
                expediente_ref, version, corte_global, organizacion_ref,
                numero_visible, flujo_ref, flujo_version,
                flujo_huella_sha256, fase_clave, estado_clave,
                centro_ref, categoria_ref, modalidad_clave, unidad_ref,
                creado_en, actualizado_en, agregado_huella_sha256,
                registrada_en
            )
            SELECT expediente_ref || ':rechazo_insert',
                   version,
                   9100000000000000::numeric + pg_catalog.length(v_estado),
                   organizacion_ref, numero_visible, flujo_ref,
                   flujo_version, flujo_huella_sha256, fase_clave,
                   v_estado, centro_ref, categoria_ref, modalidad_clave,
                   unidad_ref, creado_en, actualizado_en,
                   agregado_huella_sha256, registrada_en
              FROM fixture_estados_publicacion_rrhh
             ORDER BY corte_global
             LIMIT 1;
            RAISE EXCEPTION 'estado inválido aceptado por INSERT: %',
                v_estado;
        EXCEPTION WHEN check_violation THEN NULL;
        END;

        -- La cadena vacía es el filtro canónico «sin estado» en CT-000040;
        -- no es, en cambio, un estado publicable válido.
        IF v_estado <> '' THEN
            BEGIN
                PERFORM
                    vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
                        ROW(
                            '', v_estado, '', 100, ''
                        )::vec_contratacion_temporal
                            .consulta_cuadro_rrhh_v1
                    );
                RAISE EXCEPTION 'canon aceptó estado no canónico: %',
                    v_estado;
            EXCEPTION WHEN invalid_parameter_value THEN NULL;
            END;

            BEGIN
                PERFORM
                    vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
                        ROW(
                            '', v_estado, '', 100, ''
                        )::vec_contratacion_temporal
                            .consulta_cuadro_rrhh_v1
                    );
                RAISE EXCEPTION
                    'canon de familia aceptó estado no canónico: %',
                    v_estado;
            EXCEPTION WHEN invalid_parameter_value THEN NULL;
            END;
        END IF;
    END LOOP;

    BEGIN
        UPDATE fixture_estados_publicacion_rrhh
           SET estado_clave = NULL
         WHERE corte_global = 9000000000000001::numeric;
        RAISE EXCEPTION 'estado nulo aceptado';
    EXCEPTION WHEN not_null_violation THEN NULL;
    END;
END
$rechazos$;

SET ROLE vec_contratacion_temporal_propietario;
DO $historia$
BEGIN
    BEGIN
        UPDATE vec_contratacion_temporal
               .control_vocabulario_estados_publicacion_rrhh_v1
           SET version_esquema = 1
         WHERE control;
        RAISE EXCEPTION 'UPDATE del manifiesto aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_contratacion_temporal
              .control_vocabulario_estados_publicacion_rrhh_v1
         WHERE control;
        RAISE EXCEPTION 'DELETE del manifiesto aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal
            .control_vocabulario_estados_publicacion_rrhh_v1;
        RAISE EXCEPTION 'TRUNCATE del manifiesto aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$historia$;
RESET ROLE;
