BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $estructura$
DECLARE
    v_tabla regclass :=
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE oid=v_tabla AND relkind='r' AND relpersistence='p'
           AND relowner='vec_autorizacion_propietario'::regrole
           AND relrowsecurity AND relforcerowsecurity
    ) OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute
           WHERE attrelid=v_tabla AND attnum>0 AND NOT attisdropped)<>15
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid=v_tabla AND contype<>'n' AND convalidated)<>14
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid=v_tabla AND contype='n' AND convalidated)<>15
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
            WHERE indrelid=v_tabla)<>3
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid=v_tabla AND NOT tgisinternal)<>3
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid=v_tabla AND polname='acceso_propietario_exacto'
              AND polpermissive AND polcmd='*')<>1
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
            WHERE pronamespace='vec_autorizacion'::regnamespace
              AND proname=ANY(ARRAY[
                'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1',
                'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
                'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1',
                'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
                'publicar_vinculacion_motivo_cuadro_rrhh_v1',
                'publicar_vinculacion_motivo_detalle_rrhh_v1',
                'retirar_vinculacion_motivo_cuadro_rrhh_v1',
                'retirar_vinculacion_motivo_detalle_rrhh_v1'
              ]::name[]))<>8 THEN
        RAISE EXCEPTION 'estructura 000009 incompleta';
    END IF;
END
$estructura$;

DO $acl$
BEGIN
    IF pg_catalog.has_table_privilege(
         'vec_autorizacion_motivos_proyector',
         'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1',
         'SELECT')
       OR pg_catalog.has_function_privilege(
         'vec_autorizacion_motivos_proyector',
         'vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,text,integer,text,text,timestamptz)',
         'EXECUTE')
       OR NOT pg_catalog.has_function_privilege(
         'vec_autorizacion_motivos_proyector',
         'vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)',
         'EXECUTE')
       OR pg_catalog.has_function_privilege(
         'vec_autorizacion_motivos_evaluador',
         'vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)',
         'EXECUTE') THEN
        RAISE EXCEPTION 'ACL 000009 abierta';
    END IF;
END
$acl$;

DO $inmutabilidad$
BEGIN
    BEGIN
        TRUNCATE vec_autorizacion
          .vinculacion_motivo_consulta_rrhh_evento_v1;
        RAISE EXCEPTION 'TRUNCATE aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        INSERT INTO vec_autorizacion
          .vinculacion_motivo_consulta_rrhh_evento_v1(
            clase_consulta,operacion,publicacion_version,evento_ref,
            evento_huella_sha256,publicacion_ref,
            publicacion_huella_sha256,ocurrida_en,actor_tecnico_ref,
            prueba_vec_secuencia_origen,prueba_vec_evento_origen_ref,
            prueba_vec_evento_huella_sha256,prueba_vec_validada_en)
        VALUES (
          'cuadro','publicacion',1,
          'evento_vinculacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
          repeat('a',64),
          'publicacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
          repeat('b',64),clock_timestamp(),'postgres',1,
          'evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',repeat('c',64),
          clock_timestamp());
        RAISE EXCEPTION 'INSERT directo aceptado';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
END
$inmutabilidad$;

ROLLBACK;
