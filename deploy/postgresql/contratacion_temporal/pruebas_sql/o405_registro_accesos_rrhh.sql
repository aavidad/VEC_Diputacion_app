\set ON_ERROR_STOP on

CREATE FUNCTION pg_temp.registro_rrhh_prueba(
    p_indice integer,
    p_tipo text
)
RETURNS jsonb
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_build_object(
        'accion',
        CASE p_tipo
            WHEN 'cuadro' THEN
                'contratacion_temporal.cuadro.consultar'
            ELSE 'contratacion_temporal.expediente.consultar'
        END,
        'actor_ref',
        'actor:rrhh:' || p_indice::text,
        'ambito_ref',
        'ambito:rrhh:' || p_indice::text,
        'audiencia',
        CASE p_tipo
            WHEN 'cuadro' THEN
                'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
            ELSE
                'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
        END,
        'auditoria_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 80), 64, '8'),
        'auditoria_vec_ref',
        'auditoria:vec:rrhh:' || p_indice::text,
        'capacidad_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 50), 64, '5'),
        'consumo_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 60), 64, '6'),
        'correlacion_ref',
        'correlacion:rrhh:' || p_indice::text,
        'decision_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 10), 64, '1'),
        'decision_ref',
        'decision:rrhh:' || p_indice::text,
        'dominio_huella_consulta',
        CASE p_tipo
            WHEN 'cuadro' THEN
                'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
            ELSE
                'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        END,
        'expediente_ref',
        CASE
            WHEN p_tipo = 'detalle' THEN
                'expediente:rrhh:' || p_indice::text
            ELSE NULL
        END,
        'finalidad',
        CASE p_tipo
            WHEN 'cuadro' THEN
                'gestion_operativa_contratacion_temporal'
            ELSE
                'tramitacion_expediente_contratacion_temporal'
        END,
        'modulo_id',
        'contratacion_temporal',
        'organizacion_ref',
        'organizacion:rrhh:' || p_indice::text,
        'perfil_id',
        'perfil:rrhh:' || p_indice::text,
        'perfil_version',
        1,
        'recurso_ref',
        CASE
            WHEN p_tipo = 'detalle' THEN
                'expediente:rrhh:' || p_indice::text
            ELSE 'ambito:rrhh:' || p_indice::text
        END,
        'recurso_tipo',
        CASE p_tipo
            WHEN 'cuadro' THEN
                'cuadro_rrhh_contratacion_temporal'
            ELSE 'expediente_contratacion_temporal'
        END,
        'resultado_generico',
        'entregado',
        'resultado_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 70), 64, '7'),
        'sesion_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 20), 64, '2'),
        'sesion_id',
        'sesion:rrhh:' || p_indice::text,
        'tipo_consulta',
        p_tipo,
        'total',
        CASE p_tipo WHEN 'cuadro' THEN 2 ELSE 1 END,
        'version_expediente',
        CASE p_tipo WHEN 'detalle' THEN 3 ELSE NULL END,
        'consulta_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 40), 64, '4')
    )
$funcion$;

DO $estructura$
DECLARE
    v_tabla text;
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.cursor_cuadro_rrhh'
    ) IS NOT NULL THEN
        RAISE EXCEPTION 'C2-B no debe crear cursor sin corte global';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 16
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control
           AND version_esquema = 1
    ) THEN
        RAISE EXCEPTION 'barreras de C2-B no están alineadas';
    END IF;
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_migracion_consultas_rrhh',
        'control_cadena_accesos_rrhh',
        'registro_acceso_rrhh'
    ]::text[] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class tabla
              JOIN pg_catalog.pg_roles propietario
                ON propietario.oid = tabla.relowner
              JOIN pg_catalog.pg_namespace esquema
                ON esquema.oid = tabla.relnamespace
             WHERE esquema.nspname = 'vec_contratacion_temporal'
               AND tabla.relname = v_tabla
               AND propietario.rolname =
                   'vec_contratacion_temporal_propietario'
               AND tabla.relrowsecurity
               AND tabla.relforcerowsecurity
        ) OR (
            SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_policies politica
             WHERE politica.schemaname = 'vec_contratacion_temporal'
               AND politica.tablename = v_tabla
               AND politica.policyname = 'propietario_total'
               AND politica.roles =
                   ARRAY['vec_contratacion_temporal_propietario']::name[]
        ) <> 1 THEN
            RAISE EXCEPTION 'propiedad o RLS incorrectas en %', v_tabla;
        END IF;
    END LOOP;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute columna
          JOIN pg_catalog.pg_class tabla
            ON tabla.oid = columna.attrelid
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = tabla.relnamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND tabla.relname = 'registro_acceso_rrhh'
           AND columna.attnum > 0
           AND NOT columna.attisdropped
           AND columna.attname::text ~
               '(cursor|busqueda|nombre|dni|correo|telefono|documento|observacion|material)'
    ) THEN
        RAISE EXCEPTION 'registro RRHH conserva un campo prohibido';
    END IF;
    IF pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal',
        'USAGE'
    ) OR pg_catalog.has_table_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal.registro_acceso_rrhh',
        'SELECT'
    ) OR pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'C2-B concedió acceso antes de la fachada C2-C';
    END IF;
END
$estructura$;

SET ROLE vec_contratacion_temporal_propietario;

DO $casos$
DECLARE
    v_antes bigint;
    v_recibo jsonb;
    v_registro jsonb;
BEGIN
    v_recibo :=
        vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(
            pg_temp.registro_rrhh_prueba(1, 'cuadro')
        );
    IF v_recibo ->> 'esquema' <>
           'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v1'
       OR (v_recibo ->> 'secuencia')::integer <> 1
       OR v_recibo ->> 'anterior_sha256' <>
           pg_catalog.repeat('0', 64) THEN
        RAISE EXCEPTION 'recibo de cuadro inválido';
    END IF;
    v_recibo :=
        vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(
            pg_temp.registro_rrhh_prueba(2, 'detalle')
        );
    IF (v_recibo ->> 'secuencia')::integer <> 2 THEN
        RAISE EXCEPTION 'recibo de detalle inválido';
    END IF;

    SELECT pg_catalog.count(*) INTO v_antes
      FROM vec_contratacion_temporal.registro_acceso_rrhh;
    FOREACH v_registro IN ARRAY ARRAY[
        pg_temp.registro_rrhh_prueba(10, 'cuadro')
            || pg_catalog.jsonb_build_object(
                'accion',
                'contratacion_temporal.expediente.consultar'
            ),
        pg_temp.registro_rrhh_prueba(11, 'cuadro')
            || pg_catalog.jsonb_build_object('total', 101),
        pg_temp.registro_rrhh_prueba(12, 'detalle')
            || pg_catalog.jsonb_build_object(
                'version_expediente',
                NULL
            ),
        pg_temp.registro_rrhh_prueba(13, 'cuadro')
            || pg_catalog.jsonb_build_object('perfil_version', 1.5),
        pg_temp.registro_rrhh_prueba(14, 'cuadro')
            || pg_catalog.jsonb_build_object(
                'capacidad_huella_sha256',
                pg_catalog.repeat('0', 64)
            ),
        pg_temp.registro_rrhh_prueba(15, 'cuadro')
            || pg_catalog.jsonb_build_object('campo_ajeno', true),
        pg_temp.registro_rrhh_prueba(16, 'cuadro')
            || pg_catalog.jsonb_build_object(
                'actor_ref',
                pg_catalog.repeat('a', 33000)
            ),
        pg_temp.registro_rrhh_prueba(17, 'cuadro')
            || pg_catalog.jsonb_build_object(
                'consulta_huella_sha256',
                1111111111111111111111111111111111111111111111111111111111111111
            ),
        pg_temp.registro_rrhh_prueba(18, 'detalle')
            || pg_catalog.jsonb_build_object(
                'total',
                0,
                'resultado_generico',
                'sin_resultado'
            )
    ]::jsonb[] LOOP
        BEGIN
            PERFORM
                vec_contratacion_temporal
                    .registrar_acceso_rrhh_interno_v1(v_registro);
            RAISE EXCEPTION 'registro hostil aceptado';
        EXCEPTION
            WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN
                NULL;
        END;
    END LOOP;
    IF (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.registro_acceso_rrhh
    ) <> v_antes THEN
        RAISE EXCEPTION 'un rechazo dejó escritura parcial';
    END IF;
END
$casos$;

DO $cadena$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM (
              SELECT registro.*,
                     pg_catalog.lag(
                         huella_sha256,
                         1,
                         pg_catalog.repeat('0', 64)
                     ) OVER (ORDER BY secuencia) AS anterior_esperado
                FROM vec_contratacion_temporal.registro_acceso_rrhh registro
          ) ordenado
         WHERE anterior_sha256 <> anterior_esperado
            OR huella_sha256 <> pg_catalog.encode(
                pg_catalog.sha256(
                    pg_catalog.decode(anterior_sha256, 'hex')
                    || prueba_canonica
                ),
                'hex'
            )
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_cadena_accesos_rrhh control
          JOIN vec_contratacion_temporal.registro_acceso_rrhh ultimo
            ON ultimo.secuencia = control.ultima_secuencia
           AND ultimo.huella_sha256 = control.cabeza_sha256
         WHERE control.control
           AND control.ultima_secuencia = 2
    ) THEN
        RAISE EXCEPTION 'cadena de accesos RRHH inválida';
    END IF;
END
$cadena$;

DO $inmutabilidad$
BEGIN
    BEGIN
        UPDATE vec_contratacion_temporal.registro_acceso_rrhh
           SET resultado_generico = 'sin_resultado'
         WHERE secuencia = 1;
        RAISE EXCEPTION 'UPDATE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_contratacion_temporal.registro_acceso_rrhh
         WHERE secuencia = 1;
        RAISE EXCEPTION 'DELETE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal.registro_acceso_rrhh;
        RAISE EXCEPTION 'TRUNCATE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$inmutabilidad$;

RESET ROLE;
