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
               'canon_consulta_cuadro_rrhh_v1',
               'canon_familia_cuadro_rrhh_v1',
               'canon_consulta_detalle_rrhh_v1',
               'canon_resultado_consulta_rrhh_v1'
           ]::name[])
    ) <> 5 OR (
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
                    funcion.provolatile::text, funcion.proisstrict,
                    funcion.prosecdef, funcion.proparallel::text,
                    funcion.proconfig, funcion.proacl,
                    pg_catalog.pg_get_functiondef(funcion.oid)
                ) ORDER BY funcion.proname COLLATE "C")
                  FROM pg_catalog.pg_proc funcion
                 WHERE funcion.pronamespace =
                       'vec_contratacion_temporal'::pg_catalog.regnamespace
                   AND funcion.proname = ANY(ARRAY[
                       'json_rrhh_seguro_v1',
                       'canon_consulta_cuadro_rrhh_v1',
                       'canon_familia_cuadro_rrhh_v1',
                       'canon_consulta_detalle_rrhh_v1',
                       'canon_resultado_consulta_rrhh_v1'
                   ]::name[])
            ),
            'tipos', (
                SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                    tipo.typname, tipo.typtype::text, tipo.typacl,
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                pg_catalog.format_type(
                                    atributo.atttypid, atributo.atttypmod
                                ), atributo.attnotnull,
                                atributo.attcollation::text
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
                    tabla.relrowsecurity, tabla.relforcerowsecurity,
                    tabla.relacl, tabla.reloptions,
                    (
                        SELECT pg_catalog.jsonb_agg(
                            pg_catalog.jsonb_build_array(
                                atributo.attnum, atributo.attname,
                                pg_catalog.format_type(
                                    atributo.atttypid, atributo.atttypmod
                                ), atributo.attnotnull,
                                pg_catalog.pg_get_expr(
                                    defecto.adbin, defecto.adrelid, false
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
                 WHERE tabla.oid =
                       'vec_contratacion_temporal.'
                       'control_motor_consultas_rrhh_v1'::regclass
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
