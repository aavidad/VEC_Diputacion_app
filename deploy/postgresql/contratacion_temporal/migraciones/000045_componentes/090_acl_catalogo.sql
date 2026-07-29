-- CT-000045: privilegio mínimo y huella semántica de las dos fachadas.
\if :ct000045_aplicar_acl
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
),
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

GRANT USAGE ON SCHEMA vec_contratacion_temporal
TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
),
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
\endif

DO $catalogo$
DECLARE
    v_esquema oid :=
        'vec_contratacion_temporal'::pg_catalog.regnamespace;
    v_propietario oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
    v_consultor oid :=
        'vec_contratacion_temporal_consultor_rrhh'::pg_catalog.regrole;
    v_funciones integer;
    v_huella text;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace = v_esquema
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])
           AND funcion.proowner = v_propietario
           AND funcion.prokind = 'f'
           AND funcion.provolatile = 'v'
           AND NOT funcion.proisstrict
           AND funcion.prosecdef
           AND NOT funcion.proleakproof
           AND funcion.proparallel = 'u'
           AND funcion.pronargs = 12
           AND funcion.pronargdefaults = 0
           AND funcion.proretset
           AND funcion.prorettype = 'record'::pg_catalog.regtype
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog',
               'row_security=on',
               'TimeZone=UTC',
               'lock_timeout=1s',
               'statement_timeout=4s',
               'idle_in_transaction_session_timeout=6s'
           ]::text[]
           AND funcion.proargtypes = ARRAY[
               'vec_contratacion_temporal.'
               'alcance_consulta_rrhh_v1'::pg_catalog.regtype::oid,
               CASE
                   WHEN funcion.proname =
                        'consultar_cuadro_rrhh_atestado_v1'
                   THEN 'vec_contratacion_temporal.'
                        'consulta_cuadro_rrhh_v1'::pg_catalog.regtype::oid
                   ELSE 'vec_contratacion_temporal.'
                        'consulta_detalle_rrhh_v1'::pg_catalog.regtype::oid
               END,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'numeric'::pg_catalog.regtype::oid,
               'numeric'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid,
               'bytea'::pg_catalog.regtype::oid
           ]::oidvector
           AND funcion.proargnames[1:12] = ARRAY[
               'p_alcance', 'p_consulta', 'p_capacidad_canonica',
               'p_decision_canonica', 'p_motivo_canonico',
               'p_contexto_actor_canonico', 'p_persona_version',
               'p_perfil_version', 'p_payload_vec_ad_3',
               'p_sobre_cose_sign_1', 'p_evidencia_verificacion',
               'p_raiz_publica_spki'
           ]::text[]
           AND funcion.proargmodes[1:12] =
               pg_catalog.array_fill('i'::"char", ARRAY[12])
           AND (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       acl.grantor::pg_catalog.regrole::text,
                       acl.grantee::pg_catalog.regrole::text,
                       acl.privilege_type, acl.is_grantable
                   ) ORDER BY
                       acl.grantee::pg_catalog.regrole::text COLLATE "C",
                       acl.privilege_type COLLATE "C"
               )
                 FROM pg_catalog.aclexplode(funcion.proacl) acl
           ) = pg_catalog.jsonb_build_array(
               pg_catalog.jsonb_build_array(
                   'vec_contratacion_temporal_propietario',
                   'vec_contratacion_temporal_consultor_rrhh',
                   'EXECUTE', false
               ),
               pg_catalog.jsonb_build_array(
                   'vec_contratacion_temporal_propietario',
                   'vec_contratacion_temporal_propietario',
                   'EXECUTE', false
               )
           )
    ) <> 2
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace = v_esquema
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])
           AND (
               funcion.proargnames[13:] IS DISTINCT FROM
                   CASE
                       WHEN funcion.proname =
                            'consultar_cuadro_rrhh_atestado_v1'
                       THEN ARRAY[
                           'contenido_canonico', 'cursor_siguiente',
                           'esquema', 'acceso_ref', 'secuencia',
                           'anterior_sha256', 'huella_sha256',
                           'vinculo_identidad_huella_sha256',
                           'alcance_huella_sha256', 'registrada_en',
                           'auditoria_vec_ref',
                           'auditoria_vec_huella_sha256',
                           'consumo_vec_huella_sha256',
                           'contenido_huella_sha256',
                           'resultado_huella_sha256',
                           'cursor_huella_sha256', 'generada_en',
                           'expediente_ref', 'version_expediente',
                           'total', 'recibo_sello_sha256'
                       ]::text[]
                       ELSE ARRAY[
                           'contenido_canonico', 'esquema', 'acceso_ref',
                           'secuencia', 'anterior_sha256',
                           'huella_sha256',
                           'vinculo_identidad_huella_sha256',
                           'alcance_huella_sha256', 'registrada_en',
                           'auditoria_vec_ref',
                           'auditoria_vec_huella_sha256',
                           'consumo_vec_huella_sha256',
                           'contenido_huella_sha256',
                           'resultado_huella_sha256',
                           'cursor_huella_sha256', 'generada_en',
                           'expediente_ref', 'version_expediente',
                           'total', 'recibo_sello_sha256'
                       ]::text[]
                   END
               OR funcion.proargmodes[13:] IS DISTINCT FROM
                   pg_catalog.array_fill(
                       't'::"char",
                       ARRAY[CASE
                           WHEN funcion.proname =
                                'consultar_cuadro_rrhh_atestado_v1'
                           THEN 21 ELSE 20
                       END]
                   )
               OR funcion.proallargtypes[13:] IS DISTINCT FROM
                   CASE
                       WHEN funcion.proname =
                            'consultar_cuadro_rrhh_atestado_v1'
                       THEN ARRAY[
                           'bytea', 'text', 'text', 'text', 'numeric',
                           'text', 'text', 'text', 'text', 'timestamptz',
                           'text', 'text', 'text', 'text', 'text',
                           'text', 'timestamptz', 'text', 'numeric',
                           'smallint', 'text'
                       ]::pg_catalog.regtype[]::oid[]
                       ELSE ARRAY[
                           'bytea', 'text', 'text', 'numeric', 'text',
                           'text', 'text', 'text', 'timestamptz', 'text',
                           'text', 'text', 'text', 'text', 'text',
                           'timestamptz', 'text', 'numeric', 'smallint',
                           'text'
                       ]::pg_catalog.regtype[]::oid[]
                   END
           )
    )
    OR NOT pg_catalog.has_database_privilege(
        v_consultor, pg_catalog.current_database(), 'CONNECT'
    )
    OR pg_catalog.has_database_privilege(
        v_consultor, pg_catalog.current_database(), 'CREATE'
    )
    OR pg_catalog.has_database_privilege(
        v_consultor, pg_catalog.current_database(), 'TEMP'
    )
    OR NOT pg_catalog.has_schema_privilege(
        v_consultor, v_esquema, 'USAGE'
    )
    OR pg_catalog.has_schema_privilege(
        v_consultor, v_esquema, 'CREATE'
    )
    OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles grupo
         WHERE grupo.oid = v_consultor
           AND NOT grupo.rolcanlogin
           AND grupo.rolinherit
           AND NOT grupo.rolsuper
           AND NOT grupo.rolcreatedb
           AND NOT grupo.rolcreaterole
           AND NOT grupo.rolreplication
           AND NOT grupo.rolbypassrls
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members membresia
         WHERE membresia.member = v_consultor
    )
    -- Son los 24 tipos nominales creados explícitamente por CT40, CT42,
    -- CT43 y CT44. Los row-types implícitos de sus dos tablas quedan
    -- cubiertos por la ausencia total de privilegios de relación; no se
    -- reescriben ACL pertenecientes a migraciones ya selladas.
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace = v_esquema
           AND pg_catalog.has_function_privilege(
               v_consultor, funcion.oid, 'EXECUTE'
           )
    ) <> 2
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace = v_esquema
           AND tipo.typname = ANY(ARRAY[
               'alcance_consulta_rrhh_v1',
               'consulta_cuadro_rrhh_v1',
               'consulta_detalle_rrhh_v1',
               'evidencia_resultado_rrhh_v1',
               'resumen_publicacion_rrhh_v1',
               'solicitud_operativa_rrhh_v1',
               'analisis_operativo_rrhh_v1',
               'comprobacion_operativa_rrhh_v1',
               'cobertura_operativa_rrhh_v1',
               'asignacion_operativa_rrhh_v1',
               'hito_expediente_rrhh_v1',
               'entrada_detalle_expediente_rrhh_v1',
               'evidencia_recibo_lectura_rrhh_v2',
               'contexto_cierre_prueba_rrhh_v2',
               'contenido_cierre_prueba_rrhh_v2',
               'evidencia_consumo_nuevo_rrhh_v3',
               'resultado_cierre_prueba_rrhh_v2',
               'material_autorizacion_consulta_rrhh_v3',
               'estado_cursor_entrada_cuadro_rrhh_v1',
               'materializacion_cuadro_rrhh_v1',
               'materializacion_detalle_rrhh_v1',
               'salida_cursor_cuadro_rrhh_v1',
               'resultado_motor_cuadro_rrhh_v1',
               'resultado_motor_detalle_rrhh_v1'
           ]::name[])
    ) <> 24
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace = v_esquema
           AND tipo.typname = ANY(ARRAY[
               'alcance_consulta_rrhh_v1',
               'consulta_cuadro_rrhh_v1',
               'consulta_detalle_rrhh_v1',
               'evidencia_resultado_rrhh_v1',
               'resumen_publicacion_rrhh_v1',
               'solicitud_operativa_rrhh_v1',
               'analisis_operativo_rrhh_v1',
               'comprobacion_operativa_rrhh_v1',
               'cobertura_operativa_rrhh_v1',
               'asignacion_operativa_rrhh_v1',
               'hito_expediente_rrhh_v1',
               'entrada_detalle_expediente_rrhh_v1',
               'evidencia_recibo_lectura_rrhh_v2',
               'contexto_cierre_prueba_rrhh_v2',
               'contenido_cierre_prueba_rrhh_v2',
               'evidencia_consumo_nuevo_rrhh_v3',
               'resultado_cierre_prueba_rrhh_v2',
               'material_autorizacion_consulta_rrhh_v3',
               'estado_cursor_entrada_cuadro_rrhh_v1',
               'materializacion_cuadro_rrhh_v1',
               'materializacion_detalle_rrhh_v1',
               'salida_cursor_cuadro_rrhh_v1',
               'resultado_motor_cuadro_rrhh_v1',
               'resultado_motor_detalle_rrhh_v1'
           ]::name[])
           AND pg_catalog.has_type_privilege(
               v_consultor, tipo.oid, 'USAGE'
           )
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class relacion
         WHERE relacion.relnamespace = v_esquema
           AND (
               pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'SELECT'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'INSERT'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'UPDATE'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'DELETE'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'TRUNCATE'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'REFERENCES'
               )
               OR pg_catalog.has_table_privilege(
                   v_consultor, relacion.oid, 'TRIGGER'
               )
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL de fachadas nominales RRHH incompatible';
    END IF;

    WITH funciones AS (
        SELECT funcion.*, lenguaje.lanname
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_language lenguaje
            ON lenguaje.oid = funcion.prolang
         WHERE funcion.pronamespace = v_esquema
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])
    ), manifiesto AS (
        SELECT pg_catalog.jsonb_build_object(
            'funciones', pg_catalog.jsonb_agg(
                pg_catalog.jsonb_build_array(
                    funcion.proname,
                    pg_catalog.pg_get_function_identity_arguments(
                        funcion.oid
                    ),
                    pg_catalog.pg_get_function_result(funcion.oid),
                    funcion.proowner::pg_catalog.regrole::text,
                    funcion.lanname, funcion.prokind::text,
                    funcion.provolatile::text, funcion.proisstrict,
                    funcion.prosecdef, funcion.proleakproof,
                    funcion.proparallel::text, funcion.procost,
                    funcion.prorows, funcion.proargnames,
                    funcion.proargmodes,
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.format_type(
                                argumento.tipo_oid, NULL
                            ) ORDER BY argumento.orden
                        )
                          FROM pg_catalog.unnest(
                              funcion.proallargtypes
                          ) WITH ORDINALITY
                            AS argumento(tipo_oid, orden)
                    ),
                    funcion.proconfig, funcion.proacl,
                    pg_catalog.obj_description(
                        funcion.oid, 'pg_proc'
                    ),
                    pg_catalog.pg_get_functiondef(funcion.oid)
                ) ORDER BY funcion.proname COLLATE "C"
            )
        ) AS valor
          FROM funciones funcion
    )
    SELECT pg_catalog.count(*),
           pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(manifiesto.valor::text, 'UTF8')
           ), 'hex')
      INTO STRICT v_funciones, v_huella
      FROM funciones, manifiesto
     GROUP BY manifiesto.valor;

    RAISE NOTICE 'huella semántica CT45=%', v_huella;
    IF v_funciones <> 2
       OR v_huella <>
          '400de6a2b39a2b65dbd2d32137268e3ceaff784342cd0fb2c058eaec670f6b8a'
       THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo de fachadas nominales RRHH incompatible',
            DETAIL = 'huella=' || v_huella
                || ',funciones=' || v_funciones::text;
    END IF;
END
$catalogo$;
