-- CT-000042: frontera privada, catálogo semántico y avance de barreras.
\if :ct000042_aplicar_acl
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(text)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(bytea)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(timestamptz)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
    vec_contratacion_temporal.resumen_publicacion_rrhh_v1
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.canon_resultado_consulta_rrhh_puro_v1(
    vec_contratacion_temporal.evidencia_resultado_rrhh_v1
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
    vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    boolean, bytea
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
)
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;

REVOKE ALL ON TYPE
vec_contratacion_temporal.resumen_publicacion_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.solicitud_operativa_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.analisis_operativo_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.cobertura_operativa_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.asignacion_operativa_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.hito_expediente_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE ALL ON TYPE
vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
FROM PUBLIC, vec_contratacion_temporal_migrador,
vec_contratacion_temporal_ejecutor,
vec_contratacion_temporal_confirmador_cobertura,
vec_contratacion_temporal_gobernador,
vec_contratacion_temporal_consultor_rrhh,
vec_contratacion_temporal_lector_resultado_cobertura;

\endif
DO $catalogo$
DECLARE
    v_huella text;
    v_funciones integer;
    v_tipos integer;
    v_compuestos integer;
BEGIN
    WITH objetos AS (
        SELECT 'pg_proc'::text AS clase, funcion.oid AS objeto
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'codificar_texto_utf8_rrhh_v1',
               'decodificar_texto_utf8_rrhh_v1',
               'texto_instante_canonico_rrhh_v1',
               'encuadrar_valor_rrhh_v1',
               'canon_resumen_publicacion_rrhh_v1',
               'canon_resultado_consulta_rrhh_puro_v1',
               'canon_recibo_lectura_rrhh_v2',
               'huella_material_consumo_rrhh_v3',
               'canon_contenido_cuadro_rrhh_v1',
               'canon_contenido_detalle_rrhh_v1'
           ]::name[])
        UNION ALL
        SELECT 'pg_type', tipo.oid
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'resumen_publicacion_rrhh_v1',
               'solicitud_operativa_rrhh_v1',
               'analisis_operativo_rrhh_v1',
               'comprobacion_operativa_rrhh_v1',
               'cobertura_operativa_rrhh_v1',
               'asignacion_operativa_rrhh_v1',
               'hito_expediente_rrhh_v1',
               'entrada_detalle_expediente_rrhh_v1',
               'evidencia_recibo_lectura_rrhh_v2',
               '_resumen_publicacion_rrhh_v1',
               '_solicitud_operativa_rrhh_v1',
               '_analisis_operativo_rrhh_v1',
               '_comprobacion_operativa_rrhh_v1',
               '_cobertura_operativa_rrhh_v1',
               '_asignacion_operativa_rrhh_v1',
               '_hito_expediente_rrhh_v1',
               '_entrada_detalle_expediente_rrhh_v1',
               '_evidencia_recibo_lectura_rrhh_v2'
           ]::name[])
        UNION ALL
        SELECT 'pg_class', tipo.typrelid
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typrelid <> 0
           AND tipo.typname = ANY(ARRAY[
               'resumen_publicacion_rrhh_v1',
               'solicitud_operativa_rrhh_v1',
               'analisis_operativo_rrhh_v1',
               'comprobacion_operativa_rrhh_v1',
               'cobertura_operativa_rrhh_v1',
               'asignacion_operativa_rrhh_v1',
               'hito_expediente_rrhh_v1',
               'entrada_detalle_expediente_rrhh_v1',
               'evidencia_recibo_lectura_rrhh_v2'
           ]::name[])
    ), catalogo AS (
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
                        funcion.proowner::regrole::text,
                        lenguaje.lanname, funcion.prokind::text,
                        funcion.proretset, funcion.provolatile::text,
                        funcion.proisstrict, funcion.prosecdef,
                        funcion.proleakproof, funcion.proparallel::text,
                        funcion.procost, funcion.prorows,
                        funcion.prosupport::regprocedure::text,
                        funcion.proconfig,
                        funcion.proacl IS NULL,
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    acl.grantor::regrole::text,
                                    acl.grantee::regrole::text,
                                    acl.privilege_type, acl.is_grantable
                                ) ORDER BY acl.grantor::regrole::text
                                  COLLATE "C",
                                  acl.grantee::regrole::text COLLATE "C",
                                  acl.privilege_type COLLATE "C",
                                  acl.is_grantable
                            )
                              FROM pg_catalog.aclexplode(
                                  COALESCE(
                                      funcion.proacl,
                                      pg_catalog.acldefault(
                                          'f', funcion.proowner
                                      )
                                  )
                              ) acl
                        ),
                        pg_catalog.obj_description(
                            funcion.oid, 'pg_proc'
                        ),
                        pg_catalog.pg_get_functiondef(funcion.oid)
                    )
                    ORDER BY funcion.proname COLLATE "C",
                             pg_catalog.pg_get_function_identity_arguments(
                                 funcion.oid
                             ) COLLATE "C"
                )
                  FROM pg_catalog.pg_proc funcion
                  JOIN pg_catalog.pg_language lenguaje
                    ON lenguaje.oid = funcion.prolang
                 WHERE funcion.pronamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND funcion.proname = ANY(ARRAY[
                       'codificar_texto_utf8_rrhh_v1',
                       'decodificar_texto_utf8_rrhh_v1',
                       'texto_instante_canonico_rrhh_v1',
                       'encuadrar_valor_rrhh_v1',
                       'canon_resumen_publicacion_rrhh_v1',
                       'canon_resultado_consulta_rrhh_puro_v1',
                       'canon_recibo_lectura_rrhh_v2',
                       'huella_material_consumo_rrhh_v3',
                       'canon_contenido_cuadro_rrhh_v1',
                       'canon_contenido_detalle_rrhh_v1'
                   ]::name[])
            ),
            'tipos', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        tipo.typname, tipo.typtype::text,
                        tipo.typcategory::text, tipo.typispreferred,
                        tipo.typisdefined, tipo.typdelim::text,
                        tipo.typowner::regrole::text,
                        tipo.typacl IS NULL,
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    acl.grantor::regrole::text,
                                    acl.grantee::regrole::text,
                                    acl.privilege_type, acl.is_grantable
                                ) ORDER BY acl.grantor::regrole::text
                                  COLLATE "C",
                                  acl.grantee::regrole::text COLLATE "C",
                                  acl.privilege_type COLLATE "C",
                                  acl.is_grantable
                            )
                              FROM pg_catalog.aclexplode(
                                  COALESCE(
                                      tipo.typacl,
                                      pg_catalog.acldefault(
                                          'T', tipo.typowner
                                      )
                                  )
                              ) acl
                        ),
                        elemento.typname, matriz.typname,
                        tipo.typrelid::regclass::text,
                        pg_catalog.obj_description(tipo.oid, 'pg_type'),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    atributo.attnum, atributo.attname,
                                    atributo.attisdropped,
                                    pg_catalog.format_type(
                                        atributo.atttypid,
                                        atributo.atttypmod
                                    ),
                                    atributo.atttypmod,
                                    atributo.attnotnull,
                                    atributo.attcollation::regcollation::text,
                                    atributo.attacl, atributo.attoptions,
                                    atributo.attstorage::text,
                                    atributo.attcompression::text,
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
                )
                  FROM pg_catalog.pg_type tipo
                  LEFT JOIN pg_catalog.pg_type elemento
                    ON elemento.oid = tipo.typelem
                  LEFT JOIN pg_catalog.pg_type matriz
                    ON matriz.oid = tipo.typarray
                 WHERE tipo.typnamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND tipo.typname = ANY(ARRAY[
                       'resumen_publicacion_rrhh_v1',
                       'solicitud_operativa_rrhh_v1',
                       'analisis_operativo_rrhh_v1',
                       'comprobacion_operativa_rrhh_v1',
                       'cobertura_operativa_rrhh_v1',
                       'asignacion_operativa_rrhh_v1',
                       'hito_expediente_rrhh_v1',
                       'entrada_detalle_expediente_rrhh_v1',
                       'evidencia_recibo_lectura_rrhh_v2',
                       '_resumen_publicacion_rrhh_v1',
                       '_solicitud_operativa_rrhh_v1',
                       '_analisis_operativo_rrhh_v1',
                       '_comprobacion_operativa_rrhh_v1',
                       '_cobertura_operativa_rrhh_v1',
                       '_asignacion_operativa_rrhh_v1',
                       '_hito_expediente_rrhh_v1',
                       '_entrada_detalle_expediente_rrhh_v1',
                       '_evidencia_recibo_lectura_rrhh_v2'
                   ]::name[])
            ),
            'compuestos', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        clase.relname, clase.relkind::text,
                        clase.relowner::regrole::text,
                        clase.relpersistence::text,
                        acceso.amname, espacio.spcname,
                        clase.relrowsecurity, clase.relforcerowsecurity,
                        clase.relhastriggers, clase.relispartition,
                        clase.relreplident::text,
                        pg_catalog.pg_get_expr(
                            clase.relpartbound, clase.oid, false
                        ),
                        clase.relacl, clase.reloptions,
                        pg_catalog.obj_description(clase.oid, 'pg_class'),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.jsonb_build_array(
                                    restriccion.conname,
                                    restriccion.contype::text,
                                    restriccion.convalidated,
                                    restriccion.conenforced,
                                    pg_catalog.pg_get_constraintdef(
                                        restriccion.oid, false
                                    )
                                ) ORDER BY restriccion.conname COLLATE "C"
                            )
                              FROM pg_catalog.pg_constraint restriccion
                             WHERE restriccion.conrelid = clase.oid
                                OR restriccion.contypid = tipo.oid
                        ),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.pg_get_triggerdef(
                                    disparador.oid, false
                                ) ORDER BY disparador.tgname COLLATE "C"
                            )
                              FROM pg_catalog.pg_trigger disparador
                             WHERE disparador.tgrelid = clase.oid
                        ),
                        (
                            SELECT pg_catalog.jsonb_agg(
                                pg_catalog.pg_get_ruledef(regla.oid, false)
                                ORDER BY regla.rulename COLLATE "C"
                            )
                              FROM pg_catalog.pg_rewrite regla
                             WHERE regla.ev_class = clase.oid
                        )
                    ) ORDER BY clase.relname COLLATE "C"
                )
                  FROM pg_catalog.pg_type tipo
                  JOIN pg_catalog.pg_class clase
                    ON clase.oid = tipo.typrelid
                  LEFT JOIN pg_catalog.pg_am acceso
                    ON acceso.oid = clase.relam
                  LEFT JOIN pg_catalog.pg_tablespace espacio
                    ON espacio.oid = clase.reltablespace
                 WHERE tipo.typnamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND tipo.typname = ANY(ARRAY[
                       'resumen_publicacion_rrhh_v1',
                       'solicitud_operativa_rrhh_v1',
                       'analisis_operativo_rrhh_v1',
                       'comprobacion_operativa_rrhh_v1',
                       'cobertura_operativa_rrhh_v1',
                       'asignacion_operativa_rrhh_v1',
                       'hito_expediente_rrhh_v1',
                       'entrada_detalle_expediente_rrhh_v1',
                       'evidencia_recibo_lectura_rrhh_v2'
                   ]::name[])
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
                     SELECT 1 FROM objetos
                      WHERE clase::regclass = dependencia.classid
                        AND objeto = dependencia.objid
                 ) OR EXISTS (
                     SELECT 1 FROM objetos
                      WHERE clase::regclass = dependencia.refclassid
                        AND objeto = dependencia.refobjid
                 )
            ),
            'dependencias_compartidas', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        pg_catalog.pg_describe_object(
                            dependencia.classid,
                            dependencia.objid,
                            dependencia.objsubid
                        ),
                        dependencia.refclassid::regclass::text,
                        pg_catalog.pg_describe_object(
                            dependencia.refclassid,
                            dependencia.refobjid,
                            0
                        ),
                        dependencia.deptype::text
                    ) ORDER BY
                        pg_catalog.pg_describe_object(
                            dependencia.classid,
                            dependencia.objid,
                            dependencia.objsubid
                        ) COLLATE "C",
                        dependencia.refclassid::regclass::text COLLATE "C",
                        pg_catalog.pg_describe_object(
                            dependencia.refclassid,
                            dependencia.refobjid,
                            0
                        ) COLLATE "C",
                        dependencia.deptype
                )
                  FROM pg_catalog.pg_shdepend dependencia
                 WHERE dependencia.dbid = (
                     SELECT base.oid
                       FROM pg_catalog.pg_database base
                      WHERE base.datname = pg_catalog.current_database()
                 )
                   AND EXISTS (
                     SELECT 1 FROM objetos
                      WHERE clase::regclass = dependencia.classid
                        AND objeto = dependencia.objid
                 )
            ),
            'etiquetas', (
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        etiqueta.classoid::regclass::text,
                        etiqueta.objsubid,
                        etiqueta.provider, etiqueta.label,
                        pg_catalog.pg_describe_object(
                            etiqueta.classoid, etiqueta.objoid,
                            etiqueta.objsubid
                        )
                    ) ORDER BY etiqueta.classoid::regclass::text COLLATE "C",
                             etiqueta.objsubid,
                             etiqueta.provider COLLATE "C"
                )
                  FROM pg_catalog.pg_seclabel etiqueta
                 WHERE EXISTS (
                     SELECT 1 FROM objetos
                      WHERE clase::regclass = etiqueta.classoid
                        AND objeto = etiqueta.objoid
                 )
            )
        ) AS valor
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(valor::text, 'UTF8')
           ), 'hex'),
           (SELECT pg_catalog.count(*) FROM objetos
             WHERE clase = 'pg_proc'),
           (SELECT pg_catalog.count(*) FROM objetos
             WHERE clase = 'pg_type')
           ,
           (SELECT pg_catalog.count(*) FROM objetos
             WHERE clase = 'pg_class')
      INTO STRICT v_huella, v_funciones, v_tipos, v_compuestos
      FROM catalogo;
    IF v_funciones <> 10 OR v_tipos <> 18 OR v_compuestos <> 9
       OR v_huella <>
          'c75500cf0fede2157b31e13ca702a3a5122d01026862a6437f5f16562e24ec2d'
       THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo canónico RRHH incompatible',
            DETAIL = 'huella=' || v_huella;
    END IF;
END
$catalogo$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 22
 WHERE control AND version_esquema = 21;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 6
 WHERE control AND version_esquema = 5;

DO $barrera$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 22
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 6
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'avance de cánones RRHH incompleto';
    END IF;
END
$barrera$;
