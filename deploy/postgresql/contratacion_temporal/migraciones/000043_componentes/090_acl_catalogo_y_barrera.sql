-- CT-000043: frontera privada, catálogo semántico y avance atómico.
\if :ct000043_aplicar_acl
REVOKE ALL ON TABLE
    vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON TYPE
    vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,
    vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
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
    v_huella text;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname =
               'cerrar_prueba_resultado_recibo_rrhh_v2'
           AND funcion.proowner = v_propietario
           AND funcion.prosecdef
           AND NOT funcion.proisstrict
           AND funcion.provolatile = 'v'
           AND funcion.proparallel = 'u'
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog',
               'row_security=on',
               'TimeZone=UTC',
               'lock_timeout=1s',
               'statement_timeout=12s'
           ]::text[]
           AND (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       acl.grantor, acl.grantee,
                       acl.privilege_type, acl.is_grantable
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
    ) <> 1
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class tabla
         WHERE tabla.oid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
           AND tabla.relowner = v_propietario
           AND tabla.relkind = 'r'
           AND tabla.relpersistence = 'p'
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity
           AND (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       acl.grantor, acl.grantee,
                       acl.privilege_type, acl.is_grantable
                   ) ORDER BY acl.grantor, acl.grantee,
                              acl.privilege_type COLLATE "C",
                              acl.is_grantable
               )
                 FROM pg_catalog.aclexplode(tabla.relacl) acl
           ) = (
               SELECT pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       v_propietario, v_propietario,
                       privilegio, false
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
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
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
    ) <> 1
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'contexto_cierre_prueba_rrhh_v2',
               'contenido_cierre_prueba_rrhh_v2',
               'evidencia_consumo_nuevo_rrhh_v3',
               'resultado_cierre_prueba_rrhh_v2'
           ]::name[])
           AND (
               tipo.typowner <> v_propietario
               OR (
                   SELECT pg_catalog.jsonb_agg(
                       pg_catalog.jsonb_build_array(
                           acl.grantor, acl.grantee,
                           acl.privilege_type, acl.is_grantable
                       ) ORDER BY acl.grantor, acl.grantee,
                                  acl.privilege_type COLLATE "C",
                                  acl.is_grantable
                   )
                     FROM pg_catalog.aclexplode(tipo.typacl) acl
               ) IS DISTINCT FROM pg_catalog.jsonb_build_array(
                   pg_catalog.jsonb_build_array(
                       v_propietario, v_propietario, 'USAGE', false
                   )
               )
           )
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               '_contexto_cierre_prueba_rrhh_v2',
               '_contenido_cierre_prueba_rrhh_v2',
               '_evidencia_consumo_nuevo_rrhh_v3',
               '_resultado_cierre_prueba_rrhh_v2'
           ]::name[])
           AND (
               tipo.typowner <> v_propietario
               OR tipo.typacl IS NOT NULL
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL de prueba durable RRHH incompatible';
    END IF;

    WITH funciones AS (
        SELECT funcion.*
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname =
               'cerrar_prueba_resultado_recibo_rrhh_v2'
    ), tipos AS (
        SELECT tipo.*
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'contexto_cierre_prueba_rrhh_v2',
               'contenido_cierre_prueba_rrhh_v2',
               'evidencia_consumo_nuevo_rrhh_v3',
               'resultado_cierre_prueba_rrhh_v2',
               '_contexto_cierre_prueba_rrhh_v2',
               '_contenido_cierre_prueba_rrhh_v2',
               '_evidencia_consumo_nuevo_rrhh_v3',
               '_resultado_cierre_prueba_rrhh_v2',
               'prueba_resultado_recibo_rrhh_v2',
               '_prueba_resultado_recibo_rrhh_v2'
           ]::name[])
    ), restricciones AS (
        SELECT restriccion.*
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
            OR (
                restriccion.conrelid = ANY(ARRAY[
                    'vec_contratacion_temporal.'
                    'registro_acceso_rrhh'::pg_catalog.regclass,
                    'vec_contratacion_temporal.'
                    'vinculo_identidad_acceso_rrhh_v2'::pg_catalog.regclass,
                    'vec_contratacion_temporal.'
                    'alcance_acceso_rrhh'::pg_catalog.regclass
                ])
                AND restriccion.conname LIKE '%prueba%v2%'
            )
    ), indices AS (
        SELECT indice.*, clase.*
          FROM pg_catalog.pg_index indice
          JOIN pg_catalog.pg_class clase
            ON clase.oid = indice.indexrelid
         WHERE indice.indrelid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
            OR indice.indexrelid IN (
                SELECT restriccion.conindid
                  FROM restricciones restriccion
                 WHERE restriccion.conindid <> 0
            )
    ), objetos AS (
        SELECT 'pg_catalog.pg_proc'::pg_catalog.regclass AS clase,
               funcion.oid AS objeto
          FROM funciones funcion
        UNION
        SELECT 'pg_catalog.pg_type'::pg_catalog.regclass, tipo.oid
          FROM tipos tipo
        UNION
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
        UNION
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               indice.indexrelid
          FROM indices indice
        UNION
        SELECT 'pg_catalog.pg_constraint'::pg_catalog.regclass,
               restriccion.oid
          FROM restricciones restriccion
        UNION
        SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass,
               disparador.oid
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
           AND NOT disparador.tgisinternal
        UNION
        SELECT 'pg_catalog.pg_policy'::pg_catalog.regclass,
               politica.oid
          FROM pg_catalog.pg_policy politica
         WHERE politica.polrelid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
        UNION
        SELECT 'pg_catalog.pg_rewrite'::pg_catalog.regclass,
               regla.oid
          FROM pg_catalog.pg_rewrite regla
         WHERE regla.ev_class =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
        UNION
        SELECT 'pg_catalog.pg_statistic_ext'::pg_catalog.regclass,
               estadistica.oid
          FROM pg_catalog.pg_statistic_ext estadistica
         WHERE estadistica.stxrelid =
               'vec_contratacion_temporal.'
               'prueba_resultado_recibo_rrhh_v2'::pg_catalog.regclass
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
                        lenguaje.lanname, funcion.prokind::text,
                        funcion.proretset, funcion.provolatile::text,
                        funcion.proisstrict, funcion.prosecdef,
                        funcion.proleakproof, funcion.proparallel::text,
                        funcion.procost, funcion.prorows,
                        funcion.prosupport::pg_catalog.regprocedure::text,
                        funcion.proconfig, funcion.proacl,
                        pg_catalog.obj_description(funcion.oid, 'pg_proc'),
                        pg_catalog.pg_get_functiondef(funcion.oid)
                    ) ORDER BY funcion.oid
                )
                  FROM funciones funcion
                  JOIN pg_catalog.pg_language lenguaje
                    ON lenguaje.oid = funcion.prolang
            ),
            'tipos', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        tipo.typname, tipo.typtype::text,
                        tipo.typcategory::text, tipo.typispreferred,
                        tipo.typisdefined, tipo.typdelim::text,
                        tipo.typowner::pg_catalog.regrole::text,
                        tipo.typacl, tipo.typelem::pg_catalog.regtype::text,
                        tipo.typarray::pg_catalog.regtype::text,
                        tipo.typrelid::pg_catalog.regclass::text,
                        pg_catalog.obj_description(tipo.oid, 'pg_type'),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    atributo.attnum, atributo.attname,
                                    atributo.attisdropped,
                                    CASE WHEN atributo.attisdropped
                                         THEN NULL ELSE
                                         pg_catalog.format_type(
                                             atributo.atttypid,
                                             atributo.atttypmod
                                         ) END,
                                    atributo.atttypmod,
                                    atributo.attnotnull,
                                    CASE WHEN atributo.attcollation = 0
                                         THEN NULL ELSE
                                         atributo.attcollation
                                         ::pg_catalog.regcollation::text
                                    END,
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
                    (
                        SELECT metodo.amname
                          FROM pg_catalog.pg_am metodo
                         WHERE metodo.oid = tabla.relam
                    ),
                    (
                        SELECT espacio.spcname
                          FROM pg_catalog.pg_tablespace espacio
                         WHERE espacio.oid = tabla.reltablespace
                    ),
                    tabla.relrowsecurity,
                    tabla.relforcerowsecurity, tabla.relhastriggers,
                    tabla.relispartition, tabla.relreplident::text,
                    tabla.relacl, tabla.reloptions,
                    pg_catalog.obj_description(tabla.oid, 'pg_class'),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                atributo.attisdropped,
                                CASE WHEN atributo.attisdropped
                                     THEN NULL ELSE pg_catalog.format_type(
                                         atributo.atttypid,
                                         atributo.atttypmod
                                     ) END,
                                atributo.atttypmod,
                                atributo.attnotnull,
                                atributo.attidentity::text,
                                atributo.attgenerated::text,
                                CASE WHEN atributo.attcollation = 0
                                     THEN NULL ELSE
                                     atributo.attcollation
                                     ::pg_catalog.regcollation::text
                                END,
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
                    ),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                CASE WHEN disparador.tgisinternal
                                     THEN restriccion.conname
                                     ELSE disparador.tgname END,
                                disparador.tgisinternal,
                                disparador.tgenabled::text,
                                disparador.tgtype,
                                disparador.tgdeferrable,
                                disparador.tginitdeferred,
                                disparador.tgfoid
                                    ::pg_catalog.regprocedure::text,
                                CASE WHEN disparador.tgisinternal
                                     THEN NULL ELSE
                                     pg_catalog.pg_get_triggerdef(
                                         disparador.oid, false
                                     ) END,
                                pg_catalog.obj_description(
                                    disparador.oid, 'pg_trigger'
                                )
                            ) ORDER BY
                                disparador.tgisinternal,
                                CASE WHEN disparador.tgisinternal
                                     THEN restriccion.conname
                                     ELSE disparador.tgname END COLLATE "C",
                                disparador.tgfoid
                                    ::pg_catalog.regprocedure::text
                                    COLLATE "C",
                                disparador.tgtype
                        )
                          FROM pg_catalog.pg_trigger disparador
                          LEFT JOIN pg_catalog.pg_constraint restriccion
                            ON restriccion.oid =
                               disparador.tgconstraint
                         WHERE disparador.tgrelid = tabla.oid
                    ),
                    (
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
                                                 ELSE rol::pg_catalog.regrole
                                                      ::text END
                                        ) COLLATE "C"
                                    )
                                      FROM pg_catalog.unnest(
                                          politica.polroles
                                      ) rol
                                ),
                                pg_catalog.pg_get_expr(
                                    politica.polqual,
                                    politica.polrelid, false
                                ),
                                pg_catalog.pg_get_expr(
                                    politica.polwithcheck,
                                    politica.polrelid, false
                                ),
                                pg_catalog.obj_description(
                                    politica.oid, 'pg_policy'
                                )
                            ) ORDER BY politica.polname COLLATE "C"
                        )
                          FROM pg_catalog.pg_policy politica
                         WHERE politica.polrelid = tabla.oid
                    ),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                pg_catalog.pg_get_ruledef(regla.oid, false),
                                pg_catalog.obj_description(
                                    regla.oid, 'pg_rewrite'
                                )
                            )
                            ORDER BY regla.rulename COLLATE "C"
                        )
                          FROM pg_catalog.pg_rewrite regla
                         WHERE regla.ev_class = tabla.oid
                    )
                )
                  FROM pg_catalog.pg_class tabla
                 WHERE tabla.oid =
                       'vec_contratacion_temporal.'
                       'prueba_resultado_recibo_rrhh_v2'
                       ::pg_catalog.regclass
            ),
            'restricciones', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        restriccion.conrelid::pg_catalog.regclass::text,
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
                    ) ORDER BY
                        restriccion.conrelid::pg_catalog.regclass::text
                        COLLATE "C",
                        restriccion.conname COLLATE "C"
                ) FROM restricciones restriccion
            ),
            'indices', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        indice.indrelid::pg_catalog.regclass::text,
                        indice.indexrelid::pg_catalog.regclass::text,
                        indice.relowner::pg_catalog.regrole::text,
                        indice.indisunique, indice.indisprimary,
                        indice.indisexclusion, indice.indimmediate,
                        indice.indisclustered, indice.indisvalid,
                        indice.indisready, indice.indislive,
                        indice.indisreplident,
                        indice.indnullsnotdistinct,
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
            'columnas_existentes', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        atributo.attrelid::pg_catalog.regclass::text,
                        atributo.attname,
                        pg_catalog.format_type(
                            atributo.atttypid, atributo.atttypmod
                        ),
                        atributo.attnotnull,
                        atributo.attgenerated::text,
                        atributo.attstattarget,
                        atributo.attislocal,
                        atributo.attinhcount,
                        CASE WHEN atributo.attcollation = 0
                             THEN NULL ELSE
                             atributo.attcollation
                             ::pg_catalog.regcollation::text
                        END,
                        atributo.attacl,
                        atributo.attoptions,
                        atributo.attstorage::text,
                        atributo.attcompression::text,
                        pg_catalog.pg_get_expr(
                            defecto.adbin, defecto.adrelid, false
                        ),
                        pg_catalog.col_description(
                            atributo.attrelid, atributo.attnum
                        )
                    ) ORDER BY atributo.attname COLLATE "C"
                )
                  FROM pg_catalog.pg_attribute atributo
                  LEFT JOIN pg_catalog.pg_attrdef defecto
                    ON defecto.adrelid = atributo.attrelid
                   AND defecto.adnum = atributo.attnum
                 WHERE atributo.attrelid =
                       'vec_contratacion_temporal.'
                       'registro_acceso_rrhh'::pg_catalog.regclass
                   AND atributo.attname IN (
                       'expediente_ref_prueba_v2',
                       'version_expediente_prueba_v2'
                   )
                   AND NOT atributo.attisdropped
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
                 WHERE NOT (
                     dependencia.classid =
                         'pg_catalog.pg_trigger'::pg_catalog.regclass
                     AND EXISTS (
                         SELECT 1
                           FROM pg_catalog.pg_trigger disparador
                          WHERE disparador.oid = dependencia.objid
                            AND disparador.tgisinternal
                     )
                 ) AND NOT (
                     dependencia.classid =
                         'pg_catalog.pg_class'::pg_catalog.regclass
                     AND dependencia.objid = (
                         SELECT tabla.reltoastrelid
                           FROM pg_catalog.pg_class tabla
                          WHERE tabla.oid =
                                'vec_contratacion_temporal.'
                                'prueba_resultado_recibo_rrhh_v2'
                                ::pg_catalog.regclass
                     )
                 )
                   AND (
                     EXISTS (
                         SELECT 1 FROM objetos objeto
                          WHERE objeto.clase = dependencia.classid
                            AND objeto.objeto = dependencia.objid
                     ) OR EXISTS (
                         SELECT 1 FROM objetos objeto
                          WHERE objeto.clase = dependencia.refclassid
                            AND objeto.objeto = dependencia.refobjid
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
                               'prueba_resultado_recibo_rrhh_v2'
                               ::pg_catalog.regclass
                        OR herencia.inhparent =
                               'vec_contratacion_temporal.'
                               'prueba_resultado_recibo_rrhh_v2'
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
                           'prueba_resultado_recibo_rrhh_v2'
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
                            ),
                            pg_catalog.obj_description(
                                estadistica.oid, 'pg_statistic_ext'
                            )
                        ) ORDER BY estadistica.stxname COLLATE "C"
                    )
                      FROM pg_catalog.pg_statistic_ext estadistica
                     WHERE estadistica.stxrelid =
                           'vec_contratacion_temporal.'
                           'prueba_resultado_recibo_rrhh_v2'
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
                 ) OR (
                     etiqueta.classoid =
                         'pg_catalog.pg_class'::pg_catalog.regclass
                     AND etiqueta.objoid =
                         'vec_contratacion_temporal.'
                         'registro_acceso_rrhh'::pg_catalog.regclass
                     AND etiqueta.objsubid IN (
                         SELECT atributo.attnum
                           FROM pg_catalog.pg_attribute atributo
                          WHERE atributo.attrelid = etiqueta.objoid
                            AND atributo.attname IN (
                                'expediente_ref_prueba_v2',
                                'version_expediente_prueba_v2'
                            )
                            AND NOT atributo.attisdropped
                     )
                 )
            )
        ) AS valor
    )
    SELECT pg_catalog.count(*) FILTER (
               WHERE objeto.clase =
                     'pg_catalog.pg_proc'::pg_catalog.regclass
           ),
           (SELECT pg_catalog.count(*) FROM tipos),
           (SELECT pg_catalog.count(*) FROM restricciones),
           pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(manifiesto.valor::text, 'UTF8')
           ), 'hex')
      INTO STRICT v_funciones, v_tipos, v_restricciones, v_huella
      FROM manifiesto
      CROSS JOIN objetos objeto
     GROUP BY manifiesto.valor;

    IF v_funciones <> 1 OR v_tipos <> 10
       OR v_restricciones <> 63
       OR v_huella <>
          'e8a4cbadc41fb73d4381dff9b8aa20a19093ce53a97058af39312957906473a3'
       THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo de prueba durable RRHH incompatible',
            DETAIL = 'huella=' || v_huella
                || ',funciones=' || v_funciones::text
                || ',tipos=' || v_tipos::text
                || ',restricciones=' || v_restricciones::text;
    END IF;
END
$catalogo$;
