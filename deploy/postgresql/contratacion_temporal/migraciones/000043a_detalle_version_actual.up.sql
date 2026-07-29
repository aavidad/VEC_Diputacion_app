\set ON_ERROR_STOP on
-- CT-000043A: corrige de forma aditiva la lectura de la versión actual.
-- La consulta conserva version_observada=0 en su canon y prueba VEC; solo el
-- cotejo con la versión positiva materializada interpreta ese valor como
-- «actual». No se crean fachadas, permisos, tablas ni otra autoridad.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 23
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 7
 FOR UPDATE;

DO $corrector$
DECLARE
    v_firma pg_catalog.regprocedure :=
        'vec_contratacion_temporal.'
        'cerrar_prueba_resultado_recibo_rrhh_v2('
        'vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,'
        'vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,'
        'vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,'
        'bytea,bytea,bytea,bytea,numeric,numeric,'
        'bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
    v_funcion pg_catalog.pg_proc%ROWTYPE;
    v_definicion text;
    v_dependencias_huella text;
    v_fragmento_anterior text := $anterior$        IF (p_contexto.consulta_detalle).expediente_ref
               <> v_expediente_ref
           OR (p_contexto.consulta_detalle).version_observada
              <> v_version_expediente THEN
$anterior$;
    v_fragmento_corregido text := $corregido$        IF (p_contexto.consulta_detalle).expediente_ref
               <> v_expediente_ref
           OR v_version_expediente IS NULL
           OR v_version_expediente <= 0
           OR (
               (p_contexto.consulta_detalle).version_observada <> 0
               AND (p_contexto.consulta_detalle).version_observada
                   <> v_version_expediente
           ) THEN
$corregido$;
    v_propietario oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 23
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 7
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace =
                  'vec_contratacion_temporal'::pg_catalog.regnamespace
              AND funcion.proname =
                  'cerrar_prueba_resultado_recibo_rrhh_v2'
       ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para corregir detalle RRHH';
    END IF;

    SELECT *
      INTO STRICT v_funcion
      FROM pg_catalog.pg_proc funcion
     WHERE funcion.oid = v_firma;

    WITH dependencias AS (
        SELECT 'saliente'::text AS sentido,
               dependencia.deptype,
               dependencia.refclassid::pg_catalog.regclass::text AS clase,
               pg_catalog.pg_describe_object(
                   dependencia.refclassid, dependencia.refobjid,
                   dependencia.refobjsubid
               ) AS descripcion
          FROM pg_catalog.pg_depend dependencia
         WHERE dependencia.classid =
               'pg_catalog.pg_proc'::pg_catalog.regclass
           AND dependencia.objid = v_firma
        UNION ALL
        SELECT 'entrante', dependencia.deptype,
               dependencia.classid::pg_catalog.regclass::text,
               pg_catalog.pg_describe_object(
                   dependencia.classid, dependencia.objid,
                   dependencia.objsubid
               )
          FROM pg_catalog.pg_depend dependencia
         WHERE dependencia.refclassid =
               'pg_catalog.pg_proc'::pg_catalog.regclass
           AND dependencia.refobjid = v_firma
    ), manifiesto AS (
        SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                   sentido, deptype, clase, descripcion
               ) ORDER BY sentido COLLATE "C", deptype,
                          clase COLLATE "C", descripcion COLLATE "C") AS valor
          FROM dependencias
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               valor::text, 'UTF8'
           )), 'hex')
      INTO STRICT v_dependencias_huella
      FROM manifiesto;

    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           v_funcion.prosrc, 'UTF8'
       )), 'hex') <>
           '39f819926106443051d91a07f9158803cd58373ce9b60b050abe1a15aa6e5c7f'
       OR v_funcion.proowner <> v_propietario
       OR v_funcion.prolang <> (
           SELECT lenguaje.oid
             FROM pg_catalog.pg_language lenguaje
            WHERE lenguaje.lanname = 'plpgsql'
       )
       OR v_funcion.prorettype <>
          'vec_contratacion_temporal.'
          'resultado_cierre_prueba_rrhh_v2'::pg_catalog.regtype
       OR v_funcion.prosecdef IS DISTINCT FROM true
       OR v_funcion.proleakproof IS DISTINCT FROM false
       OR v_funcion.proisstrict IS DISTINCT FROM false
       OR v_funcion.provolatile <> 'v'
       OR v_funcion.proparallel <> 'u'
       OR v_funcion.proretset IS DISTINCT FROM false
       OR v_funcion.prokind <> 'f'
       OR v_funcion.procost <> 100
       OR v_funcion.prorows <> 0
       OR v_funcion.pronargs <> 13
       OR v_funcion.pronargdefaults <> 0
       OR v_funcion.proargmodes IS NOT NULL
       OR v_funcion.proargdefaults IS NOT NULL
       OR v_funcion.proargnames <> ARRAY[
           'p_contexto', 'p_contenido', 'p_consumo',
           'p_capacidad_canonica', 'p_decision_canonica',
           'p_motivo_canonico', 'p_contexto_actor_canonico',
           'p_persona_version', 'p_perfil_version',
           'p_payload_vec_ad_3', 'p_sobre_cose_sign_1',
           'p_evidencia_verificacion', 'p_raiz_publica_spki'
       ]::text[]
       OR v_funcion.proconfig <> ARRAY[
           'search_path=pg_catalog', 'row_security=on', 'TimeZone=UTC',
           'lock_timeout=1s', 'statement_timeout=12s'
       ]::text[]
       OR (
           SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                      acl.grantor, acl.grantee, acl.privilege_type,
                      acl.is_grantable
                  ) ORDER BY acl.grantor, acl.grantee,
                             acl.privilege_type COLLATE "C",
                             acl.is_grantable)
             FROM pg_catalog.aclexplode(v_funcion.proacl) acl
       ) <> pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_array(
           v_propietario, v_propietario, 'EXECUTE', false
       ))
       OR pg_catalog.obj_description(v_firma, 'pg_proc') IS NOT NULL
       OR v_dependencias_huella <>
          '2a97cba33069a321913d563b939c9d66b7e2a45b1a77061d242bd00d6e6694ca'
       OR pg_catalog.length(v_funcion.prosrc) -
          pg_catalog.length(pg_catalog.replace(
              v_funcion.prosrc, v_fragmento_anterior, ''
          )) <> pg_catalog.length(v_fragmento_anterior)
       OR pg_catalog.strpos(
           v_funcion.prosrc, v_fragmento_corregido
       ) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'definición anterior de detalle RRHH incompatible';
    END IF;

    v_definicion := pg_catalog.pg_get_functiondef(v_firma);
    v_definicion := pg_catalog.replace(
        v_definicion, v_fragmento_anterior, v_fragmento_corregido
    );
    EXECUTE v_definicion;

    SELECT *
      INTO STRICT v_funcion
      FROM pg_catalog.pg_proc funcion
     WHERE funcion.oid = v_firma;
    WITH dependencias AS (
        SELECT 'saliente'::text AS sentido,
               dependencia.deptype,
               dependencia.refclassid::pg_catalog.regclass::text AS clase,
               pg_catalog.pg_describe_object(
                   dependencia.refclassid, dependencia.refobjid,
                   dependencia.refobjsubid
               ) AS descripcion
          FROM pg_catalog.pg_depend dependencia
         WHERE dependencia.classid =
               'pg_catalog.pg_proc'::pg_catalog.regclass
           AND dependencia.objid = v_firma
        UNION ALL
        SELECT 'entrante', dependencia.deptype,
               dependencia.classid::pg_catalog.regclass::text,
               pg_catalog.pg_describe_object(
                   dependencia.classid, dependencia.objid,
                   dependencia.objsubid
               )
          FROM pg_catalog.pg_depend dependencia
         WHERE dependencia.refclassid =
               'pg_catalog.pg_proc'::pg_catalog.regclass
           AND dependencia.refobjid = v_firma
    ), manifiesto AS (
        SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                   sentido, deptype, clase, descripcion
               ) ORDER BY sentido COLLATE "C", deptype,
                          clase COLLATE "C", descripcion COLLATE "C") AS valor
          FROM dependencias
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               valor::text, 'UTF8'
           )), 'hex')
      INTO STRICT v_dependencias_huella
      FROM manifiesto;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           v_funcion.prosrc, 'UTF8'
       )), 'hex') <>
           '3822cb70994240e2c67784ce2b72472d1e4914c209e9166425cc1c3e3c575a46'
       OR pg_catalog.strpos(
           v_funcion.prosrc, v_fragmento_anterior
       ) <> 0
       OR pg_catalog.length(v_funcion.prosrc) -
          pg_catalog.length(pg_catalog.replace(
              v_funcion.prosrc, v_fragmento_corregido, ''
          )) <> pg_catalog.length(v_fragmento_corregido)
       OR v_funcion.proowner <> v_propietario
       OR v_funcion.prolang <> (
           SELECT lenguaje.oid
             FROM pg_catalog.pg_language lenguaje
            WHERE lenguaje.lanname = 'plpgsql'
       )
       OR v_funcion.prorettype <>
          'vec_contratacion_temporal.'
          'resultado_cierre_prueba_rrhh_v2'::pg_catalog.regtype
       OR v_funcion.prosecdef IS DISTINCT FROM true
       OR v_funcion.proleakproof IS DISTINCT FROM false
       OR v_funcion.proisstrict IS DISTINCT FROM false
       OR v_funcion.provolatile <> 'v'
       OR v_funcion.proparallel <> 'u'
       OR v_funcion.proretset IS DISTINCT FROM false
       OR v_funcion.prokind <> 'f'
       OR v_funcion.procost <> 100
       OR v_funcion.prorows <> 0
       OR v_funcion.pronargs <> 13
       OR v_funcion.pronargdefaults <> 0
       OR v_funcion.proargmodes IS NOT NULL
       OR v_funcion.proargdefaults IS NOT NULL
       OR v_funcion.proargnames <> ARRAY[
           'p_contexto', 'p_contenido', 'p_consumo',
           'p_capacidad_canonica', 'p_decision_canonica',
           'p_motivo_canonico', 'p_contexto_actor_canonico',
           'p_persona_version', 'p_perfil_version',
           'p_payload_vec_ad_3', 'p_sobre_cose_sign_1',
           'p_evidencia_verificacion', 'p_raiz_publica_spki'
       ]::text[]
       OR v_funcion.proconfig <> ARRAY[
           'search_path=pg_catalog', 'row_security=on', 'TimeZone=UTC',
           'lock_timeout=1s', 'statement_timeout=12s'
       ]::text[]
       OR (
           SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                      acl.grantor, acl.grantee, acl.privilege_type,
                      acl.is_grantable
                  ) ORDER BY acl.grantor, acl.grantee,
                             acl.privilege_type COLLATE "C",
                             acl.is_grantable)
             FROM pg_catalog.aclexplode(v_funcion.proacl) acl
       ) <> pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_array(
           v_propietario, v_propietario, 'EXECUTE', false
       ))
       OR pg_catalog.obj_description(v_firma, 'pg_proc') IS NOT NULL
       OR v_dependencias_huella <>
          '2a97cba33069a321913d563b939c9d66b7e2a45b1a77061d242bd00d6e6694ca'
       THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'corrección de detalle RRHH incompleta';
    END IF;
END
$corrector$;
COMMIT;
