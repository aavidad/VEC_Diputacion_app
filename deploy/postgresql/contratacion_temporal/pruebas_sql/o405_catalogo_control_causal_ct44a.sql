\set ON_ERROR_STOP on
-- CT-000044A: propietario, ACL y dependencias de los componentes privados.

SET search_path = pg_catalog;

DO $catalogo$
DECLARE
    v_propietario oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
    v_runtime oid :=
        'vec_contratacion_temporal_consultor_rrhh'::pg_catalog.regrole;
    v_funcion oid;
    v_nombre text;
    v_tipo text;
    v_roles oid[] := ARRAY[
        0::oid,
        'vec_contratacion_temporal_migrador'::pg_catalog.regrole::oid,
        'vec_contratacion_temporal_ejecutor'::pg_catalog.regrole::oid,
        'vec_contratacion_temporal_confirmador_cobertura'::
            pg_catalog.regrole::oid,
        'vec_contratacion_temporal_gobernador'::pg_catalog.regrole::oid,
        'vec_contratacion_temporal_consultor_rrhh'::
            pg_catalog.regrole::oid,
        'vec_contratacion_temporal_lector_resultado_cobertura'::
            pg_catalog.regrole::oid
    ]::oid[];
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class tabla
         WHERE tabla.oid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::regclass
           AND tabla.relowner = v_propietario
           AND tabla.relkind = 'r'
           AND tabla.relpersistence = 'p'
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.aclexplode(COALESCE(
                     tabla.relacl,
                     pg_catalog.acldefault('r', tabla.relowner)
                 )) permiso
                WHERE permiso.grantee = ANY(v_roles)
           )
    ) OR pg_catalog.has_table_privilege(
        v_runtime,
        'vec_contratacion_temporal.control_causal_familia_cursor_rrhh',
        'SELECT'
    ) OR pg_catalog.has_table_privilege(
        v_runtime,
        'vec_contratacion_temporal.control_causal_familia_cursor_rrhh',
        'INSERT'
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.'
               'revocacion_familia_cursor_rrhh'::regclass
           AND restriccion.contype = 'p'
           AND pg_catalog.pg_get_constraintdef(
               restriccion.oid, false
           ) = 'PRIMARY KEY (familia_ref)'
    ) <> 1 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid = ANY(ARRAY[
             'vec_contratacion_temporal.'
             'control_causal_familia_cursor_rrhh'::regclass,
             'vec_contratacion_temporal.'
             'revocacion_familia_cursor_rrhh'::regclass
         ])
           AND disparador.tgname = ANY(ARRAY[
             'validar_avance_causal_antes',
             'control_causal_familia_cursor_rrhh_no_borrar',
             'control_causal_familia_cursor_rrhh_no_truncar',
             'avanzar_control_causal_revocacion_antes'
           ])
           AND NOT disparador.tgisinternal
    ) <> 4 THEN
        RAISE EXCEPTION 'catálogo causal CT44A incompatible';
    END IF;

    FOREACH v_nombre IN ARRAY ARRAY[
        'acreditar_contexto_motor_consultas_rrhh_v1('
        || 'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        || 'vec_contratacion_temporal.'
        || 'material_autorizacion_consulta_rrhh_v3)',
        'consumir_autorizacion_motor_consultas_rrhh_v1('
        || 'text,vec_contratacion_temporal.'
        || 'material_autorizacion_consulta_rrhh_v3)',
        'validar_avance_control_causal_cursor_rrhh_v1()',
        'avanzar_control_causal_revocacion_cursor_rrhh_v1()',
        'resolver_estado_cursor_cuadro_rrhh_v1('
        || 'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        || 'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
        || 'text,text,numeric,text,text)',
        'preparar_salida_cursor_cuadro_rrhh_v1('
        || 'vec_contratacion_temporal.'
        || 'estado_cursor_entrada_cuadro_rrhh_v1,'
        || 'vec_contratacion_temporal.materializacion_cuadro_rrhh_v1)'
    ]::text[] LOOP
        v_funcion := pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.' || v_nombre
        );
        IF v_funcion IS NULL
           OR pg_catalog.has_function_privilege(
               v_runtime, v_funcion, 'EXECUTE'
           ) OR NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_proc funcion
                WHERE funcion.oid = v_funcion
                  AND funcion.proowner = v_propietario
                  AND funcion.prosecdef
                  AND funcion.provolatile = 'v'
                  AND funcion.proparallel = 'u'
                  AND funcion.proconfig = ARRAY[
                      'search_path=pg_catalog',
                      'row_security=on',
                      'TimeZone=UTC',
                      'lock_timeout=1s',
                      'statement_timeout=4s',
                      'idle_in_transaction_session_timeout=6s'
                  ]::text[]
                  AND NOT EXISTS (
                      SELECT 1
                        FROM pg_catalog.aclexplode(COALESCE(
                            funcion.proacl,
                            pg_catalog.acldefault(
                                'f', funcion.proowner
                            )
                        )) permiso
                       WHERE permiso.grantee = ANY(v_roles)
                  )
           ) THEN
            RAISE EXCEPTION 'función privada CT44A incompatible: %',
                v_nombre;
        END IF;
    END LOOP;

    FOREACH v_tipo IN ARRAY ARRAY[
        'material_autorizacion_consulta_rrhh_v3',
        'estado_cursor_entrada_cuadro_rrhh_v1',
        'materializacion_cuadro_rrhh_v1',
        'materializacion_detalle_rrhh_v1',
        'salida_cursor_cuadro_rrhh_v1',
        'resultado_motor_cuadro_rrhh_v1',
        'resultado_motor_detalle_rrhh_v1'
    ]::text[] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_type tipo
             WHERE tipo.oid = pg_catalog.to_regtype(
                 'vec_contratacion_temporal.' || v_tipo
             )
               AND tipo.typowner = v_propietario
               AND NOT EXISTS (
                   SELECT 1
                     FROM pg_catalog.aclexplode(COALESCE(
                         tipo.typacl,
                         pg_catalog.acldefault('T', tipo.typowner)
                     )) permiso
                    WHERE permiso.grantee = ANY(v_roles)
               )
        ) THEN
            RAISE EXCEPTION 'tipo privado CT44A incompatible: %', v_tipo;
        END IF;
    END LOOP;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::regclass
           AND restriccion.confrelid =
               'vec_contratacion_temporal.'
               'familia_cursor_cuadro_rrhh'::regclass
           AND restriccion.contype = 'f'
    ) <> 1 THEN
        RAISE EXCEPTION 'dependencia causal CT44A incompatible';
    END IF;
    IF pg_catalog.to_regprocedure(
        'pg_catalog.gen_random_uuid()'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'pg_catalog.uuid_send(uuid)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'pg_catalog.sha256(bytea)'
    ) IS NULL OR pg_catalog.strpos(
        pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.'
            || 'preparar_salida_cursor_cuadro_rrhh_v1('
            || 'vec_contratacion_temporal.'
            || 'estado_cursor_entrada_cuadro_rrhh_v1,'
            || 'vec_contratacion_temporal.'
            || 'materializacion_cuadro_rrhh_v1)'
        )), 'pg_catalog.gen_random_uuid'
    ) = 0 OR pg_catalog.strpos(
        pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.'
            || 'preparar_salida_cursor_cuadro_rrhh_v1('
            || 'vec_contratacion_temporal.'
            || 'estado_cursor_entrada_cuadro_rrhh_v1,'
            || 'vec_contratacion_temporal.'
            || 'materializacion_cuadro_rrhh_v1)'
        )), 'pg_catalog.uuid_send'
    ) = 0 OR pg_catalog.strpos(
        pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.'
            || 'preparar_salida_cursor_cuadro_rrhh_v1('
            || 'vec_contratacion_temporal.'
            || 'estado_cursor_entrada_cuadro_rrhh_v1,'
            || 'vec_contratacion_temporal.'
            || 'materializacion_cuadro_rrhh_v1)'
        )), 'pg_catalog.sha256'
    ) = 0 THEN
        RAISE EXCEPTION 'dependencia CSPRNG CT44A incompatible';
    END IF;
END
$catalogo$;
