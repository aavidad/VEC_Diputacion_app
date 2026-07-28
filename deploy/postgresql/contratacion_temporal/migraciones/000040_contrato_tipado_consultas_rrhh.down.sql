-- Reversión segura del contrato 000040; no retira 000041 ni deriva local.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog; SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 20 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 4 FOR UPDATE;
LOCK TABLE vec_contratacion_temporal.control_motor_consultas_rrhh_v1
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    calculada text;
    guardada text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 20
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 4
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.control_motor_consultas_rrhh_v1
         WHERE control AND version_esquema = 1
    ) <> 1 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'json_rrhh_seguro_v1',
               'canon_alcance_rrhh_v1',
               'canon_consulta_cuadro_rrhh_v1',
               'canon_familia_cuadro_rrhh_v1',
               'canon_consulta_detalle_rrhh_v1',
               'canon_resultado_consulta_rrhh_v1'
           ]::name[])
    ) <> 6 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'alcance_consulta_rrhh_v1',
               'consulta_cuadro_rrhh_v1',
               'consulta_detalle_rrhh_v1',
               'evidencia_resultado_rrhh_v1'
           ]::name[])
    ) <> 4 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir contrato RRHH';
    END IF;

    SELECT catalogo_huella_sha256 INTO STRICT guardada
      FROM vec_contratacion_temporal.control_motor_consultas_rrhh_v1
     WHERE control;
    SELECT pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(pg_catalog.jsonb_build_object(
            'funciones', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    funcion.proname,
                    pg_catalog.pg_get_function_identity_arguments(
                        funcion.oid
                    ),
                    pg_catalog.pg_get_function_result(funcion.oid),
                    funcion.proowner::regrole::text,
                    lenguaje.lanname, funcion.prokind::text,
                    funcion.provolatile::text, funcion.proisstrict,
                    funcion.prosecdef, funcion.proleakproof,
                    funcion.proparallel::text, funcion.procost, funcion.prorows,
                    funcion.proconfig, funcion.proacl,
                    pg_catalog.obj_description(funcion.oid, 'pg_proc'),
                    pg_catalog.pg_get_functiondef(funcion.oid)
                ) ORDER BY funcion.proname COLLATE "C")
                  FROM pg_catalog.pg_proc funcion
                  JOIN pg_catalog.pg_language lenguaje
                    ON lenguaje.oid = funcion.prolang
                 WHERE funcion.pronamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND funcion.proname = ANY(ARRAY[
                       'json_rrhh_seguro_v1',
                       'canon_alcance_rrhh_v1',
                       'canon_consulta_cuadro_rrhh_v1',
                       'canon_familia_cuadro_rrhh_v1',
                       'canon_consulta_detalle_rrhh_v1',
                       'canon_resultado_consulta_rrhh_v1'
                   ]::name[])
            ),
            'tipos', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    tipo.typname, tipo.typtype::text,
                    tipo.typowner::regrole::text, tipo.typacl,
                    pg_catalog.obj_description(tipo.oid, 'pg_type'),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                pg_catalog.format_type(
                                    atributo.atttypid, atributo.atttypmod
                                ), atributo.attnotnull,
                                atributo.attcollation::text,
                                atributo.attacl,
                                pg_catalog.col_description(
                                    atributo.attrelid, atributo.attnum
                                )
                            ) ORDER BY atributo.attnum
                        )
                          FROM pg_catalog.pg_attribute atributo
                         WHERE atributo.attrelid = tipo.typrelid
                           AND atributo.attnum > 0
                           AND NOT atributo.attisdropped
                    )
                ) ORDER BY tipo.typname COLLATE "C")
                  FROM pg_catalog.pg_type tipo
                 WHERE tipo.typnamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND tipo.typname = ANY(ARRAY[
                       'alcance_consulta_rrhh_v1',
                       'consulta_cuadro_rrhh_v1',
                       'consulta_detalle_rrhh_v1',
                       'evidencia_resultado_rrhh_v1'
                   ]::name[])
            ),
            'relacion', (
                SELECT pg_catalog.jsonb_build_array(
                    tabla.relkind::text, tabla.relowner::regrole::text,
                    tabla.relpersistence::text,
                    metodo.amname, espacio.spcname,
                    tabla.relrowsecurity, tabla.relforcerowsecurity,
                    tabla.relhasrules, tabla.relhastriggers,
                    tabla.relhassubclass, tabla.relispartition,
                    tabla.relreplident::text,
                    pg_catalog.pg_get_expr(
                        tabla.relpartbound, tabla.oid, false
                    ),
                    tabla.relacl, tabla.reloptions,
                    pg_catalog.obj_description(tabla.oid, 'pg_class'),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                pg_catalog.format_type(
                                    atributo.atttypid, atributo.atttypmod
                                ), atributo.attnotnull,
                                atributo.attidentity::text,
                                atributo.attgenerated::text,
                                atributo.attacl, atributo.attstorage::text,
                                atributo.attcompression::text,
                                pg_catalog.pg_get_expr(
                                    defecto.adbin, defecto.adrelid, false
                                ), pg_catalog.col_description(
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
                           AND NOT atributo.attisdropped
                    ),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.pg_get_constraintdef(
                                restriccion.oid, false
                            ) ORDER BY restriccion.conname COLLATE "C"
                        ) FROM pg_catalog.pg_constraint restriccion
                         WHERE restriccion.conrelid = tabla.oid
                    ),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.pg_get_triggerdef(
                                disparador.oid, false
                            ) ORDER BY disparador.tgname COLLATE "C"
                        ) FROM pg_catalog.pg_trigger disparador
                         WHERE disparador.tgrelid = tabla.oid
                           AND NOT disparador.tgisinternal
                    ),
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                politica.polname, politica.polcmd::text,
                                politica.polpermissive,
                                politica.polroles,
                                pg_catalog.pg_get_expr(
                                    politica.polqual,
                                    politica.polrelid, false
                                ), pg_catalog.pg_get_expr(
                                    politica.polwithcheck,
                                    politica.polrelid, false
                                )
                            ) ORDER BY politica.polname COLLATE "C"
                        ) FROM pg_catalog.pg_policy politica
                         WHERE politica.polrelid = tabla.oid
                    )
                )
                  FROM pg_catalog.pg_class tabla
                  LEFT JOIN pg_catalog.pg_am metodo
                    ON metodo.oid = tabla.relam
                  LEFT JOIN pg_catalog.pg_tablespace espacio
                    ON espacio.oid = tabla.reltablespace
                 WHERE tabla.oid =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
            ),
            'indices', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    indice.indexrelid::regclass::text,
                    clase.relowner::regrole::text, metodo.amname,
                    espacio.spcname, indice.indisunique, indice.indisprimary,
                    indice.indisexclusion, indice.indimmediate,
                    indice.indisclustered, indice.indisvalid,
                    indice.indisready, indice.indislive,
                    indice.indisreplident, indice.indnullsnotdistinct,
                    pg_catalog.pg_get_indexdef(indice.indexrelid, 0, false),
                    clase.relacl, clase.reloptions,
                    pg_catalog.obj_description(
                        indice.indexrelid, 'pg_class'
                    )
                ) ORDER BY (
                    indice.indexrelid::regclass::text
                ) COLLATE "C")
                  FROM pg_catalog.pg_index indice
                  JOIN pg_catalog.pg_class clase
                    ON clase.oid = indice.indexrelid
                  JOIN pg_catalog.pg_am metodo ON metodo.oid = clase.relam
                  LEFT JOIN pg_catalog.pg_tablespace espacio
                    ON espacio.oid = clase.reltablespace
                 WHERE indice.indrelid =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
            ),
            'reglas', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    regla.rulename, regla.ev_type::text,
                    regla.ev_enabled::text, regla.is_instead,
                    pg_catalog.pg_get_ruledef(regla.oid, false)
                ) ORDER BY regla.rulename COLLATE "C")
                  FROM pg_catalog.pg_rewrite regla
                 WHERE regla.ev_class =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
            ),
            'herencia', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    herencia.inhrelid::regclass::text,
                    herencia.inhparent::regclass::text, herencia.inhseqno,
                    hija.relispartition,
                    pg_catalog.pg_get_expr(
                        hija.relpartbound, hija.oid, false
                    )
                ) ORDER BY herencia.inhseqno)
                  FROM pg_catalog.pg_inherits herencia
                  JOIN pg_catalog.pg_class hija
                    ON hija.oid = herencia.inhrelid
                 WHERE herencia.inhrelid =
                           'vec_contratacion_temporal.'
                           'control_motor_consultas_rrhh_v1'::regclass
                    OR herencia.inhparent =
                           'vec_contratacion_temporal.'
                           'control_motor_consultas_rrhh_v1'::regclass
            ),
            'publicaciones', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    publicacion.pubname, pertenencia.prattrs::text,
                    pg_catalog.pg_get_expr(
                        pertenencia.prqual, pertenencia.prrelid, false
                    )
                ) ORDER BY publicacion.pubname COLLATE "C")
                  FROM pg_catalog.pg_publication_rel pertenencia
                  JOIN pg_catalog.pg_publication publicacion
                    ON publicacion.oid = pertenencia.prpubid
                 WHERE pertenencia.prrelid =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
            ),
            'estadisticas', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    estadistica.stxname,
                    estadistica.stxowner::regrole::text,
                    estadistica.stxkeys::text, estadistica.stxkind::text,
                    pg_catalog.pg_get_expr(
                        estadistica.stxexprs, estadistica.stxrelid, false
                    )
                ) ORDER BY estadistica.stxname COLLATE "C")
                  FROM pg_catalog.pg_statistic_ext estadistica
                 WHERE estadistica.stxrelid =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
            ),
            'etiquetas', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    objetivo.clase, objetivo.nombre, etiqueta.objsubid,
                    etiqueta.provider, etiqueta.label
                ) ORDER BY objetivo.clase COLLATE "C",
                           objetivo.nombre COLLATE "C",
                           etiqueta.objsubid, etiqueta.provider COLLATE "C")
                  FROM pg_catalog.pg_seclabel etiqueta
                  JOIN (
                      SELECT 'relacion'::text clase,
                             clase.oid objeto,
                             clase.oid::regclass::text nombre
                        FROM pg_catalog.pg_class clase
                       WHERE clase.oid =
                             'vec_contratacion_temporal.'
                             'control_motor_consultas_rrhh_v1'::regclass
                          OR clase.oid IN (
                              SELECT indice.indexrelid
                                FROM pg_catalog.pg_index indice
                               WHERE indice.indrelid =
                                     'vec_contratacion_temporal.'
                                     'control_motor_consultas_rrhh_v1'::regclass
                          )
                      UNION ALL
                      SELECT 'funcion', funcion.oid,
                             funcion.oid::regprocedure::text
                        FROM pg_catalog.pg_proc funcion
                       WHERE funcion.pronamespace =
                             'vec_contratacion_temporal'::regnamespace
                         AND funcion.proname = ANY(ARRAY[
                             'json_rrhh_seguro_v1',
                             'canon_alcance_rrhh_v1',
                             'canon_consulta_cuadro_rrhh_v1',
                             'canon_familia_cuadro_rrhh_v1',
                             'canon_consulta_detalle_rrhh_v1',
                             'canon_resultado_consulta_rrhh_v1'
                         ]::name[])
                      UNION ALL
                      SELECT 'tipo', tipo.oid, tipo.oid::regtype::text
                        FROM pg_catalog.pg_type tipo
                       WHERE (
                           tipo.typnamespace =
                               'vec_contratacion_temporal'::regnamespace
                           AND tipo.typname = ANY(ARRAY[
                               'alcance_consulta_rrhh_v1',
                               'consulta_cuadro_rrhh_v1',
                               'consulta_detalle_rrhh_v1',
                               'evidencia_resultado_rrhh_v1'
                           ]::name[])
                       ) OR tipo.typrelid =
                           'vec_contratacion_temporal.'
                           'control_motor_consultas_rrhh_v1'::regclass
                  ) objetivo
                    ON etiqueta.objoid = objetivo.objeto
                   AND etiqueta.classoid = CASE objetivo.clase
                       WHEN 'relacion' THEN
                           'pg_catalog.pg_class'::regclass
                       WHEN 'funcion' THEN
                           'pg_catalog.pg_proc'::regclass
                       ELSE 'pg_catalog.pg_type'::regclass END
            ),
            'dependencias', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    dependencia.deptype::text,
                    pg_catalog.pg_describe_object(
                        dependencia.classid, dependencia.objid,
                        dependencia.objsubid
                    ), pg_catalog.pg_describe_object(
                        dependencia.refclassid, dependencia.refobjid,
                        dependencia.refobjsubid
                    )
                ) ORDER BY pg_catalog.pg_describe_object(
                    dependencia.classid, dependencia.objid,
                    dependencia.objsubid
                ) COLLATE "C", dependencia.deptype)
                  FROM pg_catalog.pg_depend dependencia
                 WHERE (
                     dependencia.refclassid =
                         'pg_catalog.pg_class'::regclass
                     AND dependencia.refobjid =
                         'vec_contratacion_temporal.'
                         'control_motor_consultas_rrhh_v1'::regclass
                 ) OR (
                     dependencia.refclassid =
                         'pg_catalog.pg_proc'::regclass
                     AND dependencia.refobjid IN (
                         SELECT funcion.oid
                           FROM pg_catalog.pg_proc funcion
                          WHERE funcion.pronamespace =
                                'vec_contratacion_temporal'::regnamespace
                            AND funcion.proname = ANY(ARRAY[
                                'json_rrhh_seguro_v1',
                                'canon_alcance_rrhh_v1',
                                'canon_consulta_cuadro_rrhh_v1',
                                'canon_familia_cuadro_rrhh_v1',
                                'canon_consulta_detalle_rrhh_v1',
                                'canon_resultado_consulta_rrhh_v1'
                            ]::name[])
                     )
                 ) OR (
                     dependencia.refclassid =
                         'pg_catalog.pg_type'::regclass
                     AND dependencia.refobjid IN (
                         SELECT tipo.oid FROM pg_catalog.pg_type tipo
                          WHERE tipo.typnamespace =
                                'vec_contratacion_temporal'::regnamespace
                            AND tipo.typname = ANY(ARRAY[
                                'alcance_consulta_rrhh_v1',
                                'consulta_cuadro_rrhh_v1',
                                'consulta_detalle_rrhh_v1',
                                'evidencia_resultado_rrhh_v1'
                            ]::name[])
                     )
                 )
            )
        )::text, 'UTF8')
    ), 'hex') INTO STRICT calculada;
    IF calculada <> guardada THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo derivado impide revertir contrato RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.json_rrhh_seguro_v1(jsonb),
    vec_contratacion_temporal.canon_alcance_rrhh_v1(
        vec_contratacion_temporal.alcance_consulta_rrhh_v1
    ),
    vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    ),
    vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    ),
    vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        vec_contratacion_temporal.consulta_detalle_rrhh_v1
    ),
    vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
        vec_contratacion_temporal.evidencia_resultado_rrhh_v1
    ) FROM PUBLIC;
DROP FUNCTION
    vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
        vec_contratacion_temporal.evidencia_resultado_rrhh_v1
    ) RESTRICT;
DROP FUNCTION
    vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        vec_contratacion_temporal.consulta_detalle_rrhh_v1
    ) RESTRICT;
DROP FUNCTION
    vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    ) RESTRICT;
DROP FUNCTION
    vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1
    ) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.canon_alcance_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1
) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.json_rrhh_seguro_v1(jsonb)
    RESTRICT;
DROP TABLE vec_contratacion_temporal.control_motor_consultas_rrhh_v1
    RESTRICT;
DROP TYPE vec_contratacion_temporal.evidencia_resultado_rrhh_v1
    RESTRICT;
DROP TYPE vec_contratacion_temporal.consulta_detalle_rrhh_v1 RESTRICT;
DROP TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v1 RESTRICT;
DROP TYPE vec_contratacion_temporal.alcance_consulta_rrhh_v1 RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 3,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 4;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 19,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 20;
COMMIT;
