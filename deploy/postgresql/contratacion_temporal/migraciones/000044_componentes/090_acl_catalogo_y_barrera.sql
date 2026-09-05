-- CT-000044: frontera privada, ACL exacta y huella semántica del paquete.
\if :ct000044_aplicar_acl
REVOKE ALL ON TABLE
vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON TYPE
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_detalle_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1,
    vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
),
vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    text,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
),
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    numeric
),
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
),
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1(),
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1(),
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    text, text, numeric, text, text
),
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
),
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
),
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
),
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;
\endif

DO $catalogo$
DECLARE
    v_propietario oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
    v_funciones integer;
    v_tipos integer;
    v_restricciones integer;
    v_indices integer;
    v_disparadores integer;
    v_huella text;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'acreditar_contexto_motor_consultas_rrhh_v1',
               'consumir_autorizacion_motor_consultas_rrhh_v1',
               'materializar_detalle_rrhh_v1',
               'materializar_cuadro_rrhh_v1',
               'validar_avance_control_causal_cursor_rrhh_v1',
               'avanzar_control_causal_revocacion_cursor_rrhh_v1',
               'resolver_estado_cursor_cuadro_rrhh_v1',
               'preparar_salida_cursor_cuadro_rrhh_v1',
               'aplicar_efectos_cursor_cuadro_rrhh_v1',
               'motor_consultar_cuadro_rrhh_v1',
               'motor_consultar_detalle_rrhh_v1'
           ]::name[])
           AND funcion.proowner = v_propietario
           AND funcion.prosecdef
           AND NOT funcion.proisstrict
           AND funcion.proparallel = 'u'
           AND funcion.provolatile = CASE
               WHEN funcion.proname = 'materializar_detalle_rrhh_v1'
               THEN 's'::"char" ELSE 'v'::"char" END
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog', 'row_security=on',
               'TimeZone=UTC', 'lock_timeout=1s',
               'statement_timeout=4s',
               'idle_in_transaction_session_timeout=6s'
           ]::text[]
           AND (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       acl.grantor, acl.grantee, acl.privilege_type,
                       acl.is_grantable
                   ) ORDER BY acl.grantor, acl.grantee,
                              acl.privilege_type COLLATE "C",
                              acl.is_grantable
               )
                 FROM pg_catalog.aclexplode(funcion.proacl) acl
           ) = pg_catalog.jsonb_build_array(
               pg_catalog.jsonb_build_array(
                   v_propietario, v_propietario, 'EXECUTE', false
               )
           )
    ) <> 11
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'material_autorizacion_consulta_rrhh_v3',
               'estado_cursor_entrada_cuadro_rrhh_v1',
               'materializacion_cuadro_rrhh_v1',
               'materializacion_detalle_rrhh_v1',
               'salida_cursor_cuadro_rrhh_v1',
               'resultado_motor_cuadro_rrhh_v1',
               'resultado_motor_detalle_rrhh_v1'
           ]::name[])
           AND (
               tipo.typowner <> v_propietario
               OR tipo.typtype <> 'c'
               OR (
                   SELECT pg_catalog.jsonb_agg(
                       pg_catalog.jsonb_build_array(
                           acl.grantor, acl.grantee, acl.privilege_type,
                           acl.is_grantable
                       ) ORDER BY acl.grantor, acl.grantee
                   )
                     FROM pg_catalog.aclexplode(tipo.typacl) acl
               ) IS DISTINCT FROM pg_catalog.jsonb_build_array(
                   pg_catalog.jsonb_build_array(
                       v_propietario, v_propietario, 'USAGE', false
                   )
               )
           )
    )
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class tabla
         WHERE tabla.oid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
           AND tabla.relowner = v_propietario
           AND tabla.relkind = 'r'
           AND tabla.relpersistence = 'p'
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity
           AND (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       acl.grantor, acl.grantee, acl.privilege_type,
                       acl.is_grantable
                   ) ORDER BY acl.privilege_type COLLATE "C"
               )
                 FROM pg_catalog.aclexplode(tabla.relacl) acl
           ) = (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       v_propietario, v_propietario, privilegio, false
                   ) ORDER BY privilegio COLLATE "C"
               )
                 FROM pg_catalog.unnest(ARRAY[
                     'DELETE', 'INSERT', 'MAINTAIN', 'REFERENCES',
                     'SELECT', 'TRIGGER', 'TRUNCATE', 'UPDATE'
                 ]::text[]) privilegio
           )
    ) <> 1
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policy politica
         WHERE politica.polrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
           AND politica.polname = 'propietario_total'
           AND politica.polcmd = '*'
           AND politica.polpermissive
           AND politica.polroles = ARRAY[v_propietario]
           AND pg_catalog.pg_get_expr(
                   politica.polqual, politica.polrelid, false
               ) = 'true'
           AND pg_catalog.pg_get_expr(
                   politica.polwithcheck, politica.polrelid, false
               ) = 'true'
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL del motor de consultas RRHH incompatible';
    END IF;

    WITH funciones AS (
        SELECT funcion.*
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'acreditar_contexto_motor_consultas_rrhh_v1',
               'consumir_autorizacion_motor_consultas_rrhh_v1',
               'materializar_detalle_rrhh_v1',
               'materializar_cuadro_rrhh_v1',
               'validar_avance_control_causal_cursor_rrhh_v1',
               'avanzar_control_causal_revocacion_cursor_rrhh_v1',
               'resolver_estado_cursor_cuadro_rrhh_v1',
               'preparar_salida_cursor_cuadro_rrhh_v1',
               'aplicar_efectos_cursor_cuadro_rrhh_v1',
               'motor_consultar_cuadro_rrhh_v1',
               'motor_consultar_detalle_rrhh_v1'
           ]::name[])
    ), tipos AS (
        SELECT tipo.*
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'material_autorizacion_consulta_rrhh_v3',
               'estado_cursor_entrada_cuadro_rrhh_v1',
               'materializacion_cuadro_rrhh_v1',
               'materializacion_detalle_rrhh_v1',
               'salida_cursor_cuadro_rrhh_v1',
               'resultado_motor_cuadro_rrhh_v1',
               'resultado_motor_detalle_rrhh_v1',
               '_material_autorizacion_consulta_rrhh_v3',
               '_estado_cursor_entrada_cuadro_rrhh_v1',
               '_materializacion_cuadro_rrhh_v1',
               '_materializacion_detalle_rrhh_v1',
               '_salida_cursor_cuadro_rrhh_v1',
               '_resultado_motor_cuadro_rrhh_v1',
               '_resultado_motor_detalle_rrhh_v1',
               'control_causal_familia_cursor_rrhh',
               '_control_causal_familia_cursor_rrhh'
           ]::name[])
    ), restricciones AS (
        SELECT restriccion.*
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
    ), indices AS (
        SELECT indice.*, clase.relowner, clase.relacl, clase.reloptions
          FROM pg_catalog.pg_index indice
          JOIN pg_catalog.pg_class clase
            ON clase.oid = indice.indexrelid
         WHERE indice.indrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
    ), disparadores AS (
        SELECT disparador.*
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
            OR disparador.tgconstraint IN (
                SELECT restriccion.oid FROM restricciones restriccion
            )
            OR (
                disparador.tgrelid =
                    'vec_contratacion_temporal.'
                    'revocacion_familia_cursor_rrhh'::pg_catalog.regclass
                AND disparador.tgname =
                    'avanzar_control_causal_revocacion_antes'
            )
    ), politicas AS (
        SELECT politica.*
          FROM pg_catalog.pg_policy politica
         WHERE politica.polrelid =
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
    ), objetos AS (
        SELECT 'pg_catalog.pg_proc'::pg_catalog.regclass clase,
               funcion.oid objeto FROM funciones funcion
        UNION ALL
        SELECT 'pg_catalog.pg_type'::pg_catalog.regclass, tipo.oid
          FROM tipos tipo
        UNION ALL
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               'vec_contratacion_temporal.'
               'control_causal_familia_cursor_rrhh'::pg_catalog.regclass
        UNION ALL
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               indice.indexrelid FROM indices indice
        UNION ALL
        SELECT 'pg_catalog.pg_constraint'::pg_catalog.regclass,
               restriccion.oid FROM restricciones restriccion
        UNION ALL
        SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass,
               disparador.oid FROM disparadores disparador
        UNION ALL
        SELECT 'pg_catalog.pg_policy'::pg_catalog.regclass,
               politica.oid FROM politicas politica
    ), manifiesto AS (
        SELECT pg_catalog.jsonb_build_object(
            'funciones', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        funcion.proname,
                        pg_catalog.pg_get_function_identity_arguments(
                            funcion.oid
                        ),
                        pg_catalog.pg_get_function_arguments(funcion.oid),
                        pg_catalog.pg_get_function_result(funcion.oid),
                        funcion.proowner::pg_catalog.regrole::text,
                        lenguaje.lanname, funcion.provolatile::text,
                        funcion.proisstrict, funcion.prosecdef,
                        funcion.proparallel::text, funcion.proconfig,
                        funcion.proacl,
                        pg_catalog.obj_description(funcion.oid, 'pg_proc'),
                        pg_catalog.pg_get_functiondef(funcion.oid)
                    ) ORDER BY funcion.proname COLLATE "C",
                               pg_catalog.pg_get_function_identity_arguments(
                                   funcion.oid
                               ) COLLATE "C"
                )
                  FROM funciones funcion
                  JOIN pg_catalog.pg_language lenguaje
                    ON lenguaje.oid = funcion.prolang
            ),
            'tipos', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        tipo.typname, tipo.typtype::text,
                        tipo.typcategory::text,
                        tipo.typowner::pg_catalog.regrole::text,
                        tipo.typacl,
                        tipo.typelem::pg_catalog.regtype::text,
                        tipo.typarray::pg_catalog.regtype::text,
                        tipo.typrelid::pg_catalog.regclass::text,
                        pg_catalog.obj_description(tipo.oid, 'pg_type'),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    atributo.attnum, atributo.attname,
                                    pg_catalog.format_type(
                                        atributo.atttypid,
                                        atributo.atttypmod
                                    ),
                                    atributo.attnotnull,
                                    atributo.attcollation,
                                    atributo.attacl,
                                    atributo.attidentity::text,
                                    atributo.attgenerated::text,
                                    pg_catalog.col_description(
                                        atributo.attrelid,
                                        atributo.attnum
                                    )
                                ) ORDER BY atributo.attnum
                            )
                              FROM pg_catalog.pg_attribute atributo
                             WHERE atributo.attrelid = tipo.typrelid
                               AND atributo.attnum > 0
                        )
                    ) ORDER BY tipo.typname COLLATE "C"
                ) FROM tipos tipo
            ),
            'tabla', (
                SELECT pg_catalog.jsonb_build_array(
                    tabla.relkind::text,
                    tabla.relowner::pg_catalog.regrole::text,
                    tabla.relpersistence::text,
                    tabla.relrowsecurity, tabla.relforcerowsecurity,
                    tabla.relreplident::text, tabla.relacl,
                    tabla.reloptions,
                    pg_catalog.obj_description(tabla.oid, 'pg_class'),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                pg_catalog.format_type(
                                    atributo.atttypid,
                                    atributo.atttypmod
                                ),
                                atributo.attnotnull,
                                atributo.attidentity::text,
                                atributo.attgenerated::text,
                                atributo.attcollation,
                                atributo.attacl, atributo.attoptions,
                                atributo.attstorage::text,
                                atributo.attcompression::text,
                                pg_catalog.pg_get_expr(
                                    defecto.adbin, defecto.adrelid, false
                                ),
                                pg_catalog.col_description(
                                    atributo.attrelid, atributo.attnum
                                )
                            ) ORDER BY atributo.attnum
                        )
                          FROM pg_catalog.pg_attribute atributo
                          LEFT JOIN pg_catalog.pg_attrdef defecto
                            ON defecto.adrelid = atributo.attrelid
                           AND defecto.adnum = atributo.attnum
                         WHERE atributo.attrelid = tabla.oid
                           AND atributo.attnum > 0
                    )
                )
                  FROM pg_catalog.pg_class tabla
                 WHERE tabla.oid =
                       'vec_contratacion_temporal.'
                       'control_causal_familia_cursor_rrhh'
                       ::pg_catalog.regclass
            ),
            'restricciones', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        restriccion.conname,
                        restriccion.contype::text,
                        restriccion.condeferrable,
                        restriccion.condeferred,
                        restriccion.convalidated,
                        restriccion.conenforced,
                        restriccion.confmatchtype::text,
                        restriccion.confupdtype::text,
                        restriccion.confdeltype::text,
                        pg_catalog.pg_get_constraintdef(
                            restriccion.oid, false
                        ),
                        pg_catalog.obj_description(
                            restriccion.oid, 'pg_constraint'
                        )
                    ) ORDER BY restriccion.conname COLLATE "C"
                ) FROM restricciones restriccion
            ),
            'indices', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        indice.indexrelid::pg_catalog.regclass::text,
                        indice.relowner::pg_catalog.regrole::text,
                        indice.indisunique, indice.indisprimary,
                        indice.indisvalid, indice.indisready,
                        indice.indislive, indice.indnullsnotdistinct,
                        pg_catalog.pg_get_indexdef(
                            indice.indexrelid, 0, false
                        ),
                        indice.relacl, indice.reloptions,
                        pg_catalog.obj_description(
                            indice.indexrelid, 'pg_class'
                        )
                    ) ORDER BY
                        indice.indexrelid::pg_catalog.regclass::text
                        COLLATE "C"
                ) FROM indices indice
            ),
            'disparadores', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        disparador.tgrelid::pg_catalog.regclass::text,
                        CASE WHEN disparador.tgisinternal
                             THEN restriccion.conname
                             ELSE disparador.tgname END,
                        disparador.tgisinternal,
                        disparador.tgenabled::text,
                        disparador.tgtype,
                        disparador.tgdeferrable,
                        disparador.tginitdeferred,
                        disparador.tgfoid::pg_catalog.regprocedure::text,
                        CASE WHEN disparador.tgisinternal THEN NULL
                             ELSE pg_catalog.pg_get_triggerdef(
                                 disparador.oid, false
                             ) END,
                        pg_catalog.obj_description(
                            disparador.oid, 'pg_trigger'
                        )
                    ) ORDER BY
                        disparador.tgrelid::pg_catalog.regclass::text
                        COLLATE "C",
                        CASE WHEN disparador.tgisinternal
                             THEN restriccion.conname
                             ELSE disparador.tgname END COLLATE "C"
                )
                  FROM disparadores disparador
                  LEFT JOIN pg_catalog.pg_constraint restriccion
                    ON restriccion.oid = disparador.tgconstraint
            ),
            'politicas', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        politica.polname, politica.polcmd::text,
                        politica.polpermissive,
                        (
                            SELECT pg_catalog.jsonb_agg(
                                CASE WHEN rol = 0 THEN 'PUBLIC'
                                     ELSE rol::pg_catalog.regrole::text
                                END
                                ORDER BY (
                                    CASE WHEN rol = 0 THEN 'PUBLIC'
                                         ELSE rol::pg_catalog.regrole::text
                                    END
                                ) COLLATE "C"
                            )
                            FROM pg_catalog.unnest(politica.polroles) rol
                        ),
                        pg_catalog.pg_get_expr(
                            politica.polqual, politica.polrelid, false
                        ),
                        pg_catalog.pg_get_expr(
                            politica.polwithcheck,
                            politica.polrelid, false
                        ),
                        pg_catalog.obj_description(
                            politica.oid, 'pg_policy'
                        )
                    ) ORDER BY politica.polname COLLATE "C"
                ) FROM politicas politica
            ),
            'dependencias', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        pg_catalog.pg_describe_object(
                            dependencia.classid,
                            dependencia.objid,
                            dependencia.objsubid
                        ),
                        pg_catalog.pg_describe_object(
                            dependencia.refclassid,
                            dependencia.refobjid,
                            dependencia.refobjsubid
                        ),
                        dependencia.deptype::text
                    ) ORDER BY
                        pg_catalog.pg_describe_object(
                            dependencia.classid,
                            dependencia.objid,
                            dependencia.objsubid
                        ) COLLATE "C",
                        pg_catalog.pg_describe_object(
                            dependencia.refclassid,
                            dependencia.refobjid,
                            dependencia.refobjsubid
                        ) COLLATE "C",
                        dependencia.deptype
                )
                  FROM pg_catalog.pg_depend dependencia
                 WHERE EXISTS (
                     SELECT 1 FROM objetos objeto
                      WHERE objeto.clase = dependencia.classid
                        AND objeto.objeto = dependencia.objid
                 )
                   AND NOT (
                       dependencia.classid =
                           'pg_catalog.pg_trigger'::pg_catalog.regclass
                       AND EXISTS (
                           SELECT 1
                             FROM disparadores disparador
                            WHERE disparador.oid = dependencia.objid
                              AND disparador.tgisinternal
                       )
                   )
            ),
            'derivas_relacion', pg_catalog.jsonb_build_object(
                'herencia', (
                    SELECT pg_catalog.jsonb_agg(
                        pg_catalog.jsonb_build_array(
                            herencia.inhrelid::pg_catalog.regclass::text,
                            herencia.inhparent::pg_catalog.regclass::text,
                            herencia.inhseqno
                        )
                    )
                      FROM pg_catalog.pg_inherits herencia
                     WHERE herencia.inhrelid =
                               'vec_contratacion_temporal.'
                               'control_causal_familia_cursor_rrhh'
                               ::pg_catalog.regclass
                        OR herencia.inhparent =
                               'vec_contratacion_temporal.'
                               'control_causal_familia_cursor_rrhh'
                               ::pg_catalog.regclass
                ),
                'publicaciones', (
                    SELECT pg_catalog.jsonb_agg(
                        pg_catalog.jsonb_build_array(
                            publicacion.pubname,
                            pertenencia.prattrs::text,
                            pg_catalog.pg_get_expr(
                                pertenencia.prqual,
                                pertenencia.prrelid, false
                            )
                        ) ORDER BY publicacion.pubname COLLATE "C"
                    )
                      FROM pg_catalog.pg_publication_rel pertenencia
                      JOIN pg_catalog.pg_publication publicacion
                        ON publicacion.oid = pertenencia.prpubid
                     WHERE pertenencia.prrelid =
                           'vec_contratacion_temporal.'
                           'control_causal_familia_cursor_rrhh'
                           ::pg_catalog.regclass
                ),
                'estadisticas', (
                    SELECT pg_catalog.jsonb_agg(
                        pg_catalog.jsonb_build_array(
                            estadistica.stxname,
                            estadistica.stxowner::pg_catalog.regrole::text,
                            estadistica.stxkeys::text,
                            estadistica.stxkind::text,
                            pg_catalog.pg_get_expr(
                                estadistica.stxexprs,
                                estadistica.stxrelid, false
                            )
                        ) ORDER BY estadistica.stxname COLLATE "C"
                    )
                      FROM pg_catalog.pg_statistic_ext estadistica
                     WHERE estadistica.stxrelid =
                           'vec_contratacion_temporal.'
                           'control_causal_familia_cursor_rrhh'
                           ::pg_catalog.regclass
                ),
                'reglas', (
                    SELECT pg_catalog.jsonb_agg(
                        pg_catalog.pg_get_ruledef(regla.oid, false)
                        ORDER BY regla.rulename COLLATE "C"
                    )
                      FROM pg_catalog.pg_rewrite regla
                     WHERE regla.ev_class =
                           'vec_contratacion_temporal.'
                           'control_causal_familia_cursor_rrhh'
                           ::pg_catalog.regclass
                )
            ),
            'etiquetas', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        etiqueta.classoid::pg_catalog.regclass::text,
                        etiqueta.objsubid, etiqueta.provider,
                        etiqueta.label,
                        pg_catalog.pg_describe_object(
                            etiqueta.classoid, etiqueta.objoid,
                            etiqueta.objsubid
                        )
                    ) ORDER BY
                        etiqueta.classoid::pg_catalog.regclass::text
                        COLLATE "C",
                        etiqueta.objsubid, etiqueta.provider COLLATE "C"
                )
                  FROM pg_catalog.pg_seclabel etiqueta
                 WHERE EXISTS (
                     SELECT 1 FROM objetos objeto
                      WHERE objeto.clase = etiqueta.classoid
                        AND objeto.objeto = etiqueta.objoid
                 )
            )
        ) AS valor
    )
    SELECT (SELECT pg_catalog.count(*) FROM funciones),
           (SELECT pg_catalog.count(*) FROM tipos),
           (SELECT pg_catalog.count(*) FROM restricciones),
           (SELECT pg_catalog.count(*) FROM indices),
           (SELECT pg_catalog.count(*) FROM disparadores),
           pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(manifiesto.valor::text, 'UTF8')
           ), 'hex')
      INTO STRICT v_funciones, v_tipos, v_restricciones,
                  v_indices, v_disparadores, v_huella
      FROM manifiesto;

    IF v_funciones <> 11 OR v_tipos <> 16
       OR v_restricciones <> 10 OR v_indices <> 2
       OR v_disparadores <> 8
       OR v_huella <>
          'a01d3cdc140fac44d3db2073964644a6a031bc536ea0c93ed86908077b8b0a09'
       THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo del motor de consultas RRHH incompatible',
            DETAIL = 'huella=' || v_huella
                || ',funciones=' || v_funciones::text
                || ',tipos=' || v_tipos::text
                || ',restricciones=' || v_restricciones::text
                || ',indices=' || v_indices::text
                || ',disparadores=' || v_disparadores::text;
    END IF;
END
$catalogo$;
