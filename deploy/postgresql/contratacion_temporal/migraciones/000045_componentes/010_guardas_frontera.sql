-- CT-000045: precondiciones cerradas, sin crear una autoridad auxiliar.
DO $guardas$
DECLARE
    v_esquema oid :=
        'vec_contratacion_temporal'::pg_catalog.regnamespace;
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 24
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 8
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace = v_esquema
              AND funcion.proname = ANY(ARRAY[
                  'consultar_cuadro_rrhh_atestado_v1',
                  'consultar_detalle_rrhh_atestado_v1'
              ]::name[])
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'motor_consultar_cuadro_rrhh_v1('
           'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
           'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
           'vec_contratacion_temporal.'
           'material_autorizacion_consulta_rrhh_v3)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'motor_consultar_detalle_rrhh_v1('
           'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
           'vec_contratacion_temporal.consulta_detalle_rrhh_v1,'
           'vec_contratacion_temporal.'
           'material_autorizacion_consulta_rrhh_v3)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'canon_alcance_rrhh_v1('
           'vec_contratacion_temporal.alcance_consulta_rrhh_v1)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'canon_consulta_cuadro_rrhh_v1('
           'vec_contratacion_temporal.consulta_cuadro_rrhh_v1)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'canon_consulta_detalle_rrhh_v1('
           'vec_contratacion_temporal.consulta_detalle_rrhh_v1)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'canon_contenido_cuadro_rrhh_v1('
           'timestamp with time zone,'
           'vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],'
           'boolean,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'canon_contenido_detalle_rrhh_v1('
           'timestamp with time zone,'
           'vec_contratacion_temporal.'
           'entrada_detalle_expediente_rrhh_v1)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles rol
            WHERE rol.rolname =
                  'vec_contratacion_temporal_propietario'
              AND NOT rol.rolcanlogin
              AND NOT rol.rolsuper
              AND NOT rol.rolcreatedb
              AND NOT rol.rolcreaterole
              AND NOT rol.rolinherit
              AND NOT rol.rolreplication
              AND NOT rol.rolbypassrls
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles rol
            WHERE rol.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT rol.rolcanlogin
              AND NOT rol.rolsuper
              AND NOT rol.rolcreatedb
              AND NOT rol.rolcreaterole
              AND rol.rolinherit
              AND NOT rol.rolreplication
              AND NOT rol.rolbypassrls
       )
       OR pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_consultor_rrhh',
           'vec_contratacion_temporal', 'USAGE'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para fachadas nominales RRHH';
    END IF;
END
$guardas$;
