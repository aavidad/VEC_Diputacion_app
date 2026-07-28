-- O4-05/C2-D2: contrato tipado; ninguna función se concede al runtime.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog; SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 19 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 3 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 19
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 3
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_motor_consultas_rrhh_v1'
    ) IS NOT NULL OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'alcance_consulta_rrhh_v1', 'consulta_cuadro_rrhh_v1',
               'consulta_detalle_rrhh_v1',
               'evidencia_resultado_rrhh_v1'
           ]::name[])
    ) OR EXISTS (
        SELECT 1
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
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para motor de consultas RRHH';
    END IF;
END
$prevalidacion$;

CREATE TYPE vec_contratacion_temporal.alcance_consulta_rrhh_v1 AS (
    organizacion_ref text,
    clase_ambito text,
    ambito_ref text
);
CREATE TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v1 AS (
    texto text,
    estado_clave text,
    fase_clave text,
    limite smallint,
    cursor text
);
CREATE TYPE vec_contratacion_temporal.consulta_detalle_rrhh_v1 AS (
    expediente_ref text,
    version_observada numeric(20, 0)
);
CREATE TYPE vec_contratacion_temporal.evidencia_resultado_rrhh_v1 AS (
    tipo_consulta text,
    generada_en timestamptz,
    total smallint,
    contenido_huella_sha256 text,
    cursor_huella_sha256 text
);

CREATE TABLE vec_contratacion_temporal.control_motor_consultas_rrhh_v1 (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    version_esquema integer NOT NULL CHECK (version_esquema = 1),
    catalogo_huella_sha256 text NOT NULL CHECK (
        catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND catalogo_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    creada_en timestamptz(6) NOT NULL CHECK (
        creada_en = pg_catalog.date_trunc('microseconds', creada_en)
    )
);

CREATE FUNCTION vec_contratacion_temporal.json_rrhh_seguro_v1(p_valor jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog AS $funcion$
    WITH RECURSIVE nodos(valor, profundidad) AS (
        SELECT p_valor, 1
        UNION ALL
        SELECT hijo.valor, padre.profundidad + 1
          FROM nodos padre
          CROSS JOIN LATERAL (
              SELECT objeto.value AS valor
                FROM pg_catalog.jsonb_each(
                    CASE pg_catalog.jsonb_typeof(padre.valor)
                    WHEN 'object' THEN padre.valor ELSE '{}'::jsonb END
                ) objeto
              UNION ALL
              SELECT matriz.value
                FROM pg_catalog.jsonb_array_elements(
                    CASE pg_catalog.jsonb_typeof(padre.valor)
                    WHEN 'array' THEN padre.valor ELSE '[]'::jsonb END
                ) matriz
          ) hijo
         WHERE padre.profundidad < 25
    )
    SELECT pg_catalog.octet_length(p_valor::text) <= 262144
       AND pg_catalog.count(*) <= 16384
       AND COALESCE(pg_catalog.max(profundidad), 0) <= 24
      FROM nodos
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1
)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog AS $funcion$
BEGIN
    IF p_consulta.texto IS NULL
       OR p_consulta.texto <> pg_catalog.btrim(p_consulta.texto)
       OR p_consulta.texto !~
          '^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$'
       OR pg_catalog.strpos(p_consulta.texto, '%') > 0
       OR p_consulta.estado_clave IS NULL
       OR p_consulta.estado_clave NOT IN (
           '', 'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       ) OR p_consulta.fase_clave IS NULL
       OR p_consulta.fase_clave !~ '^$|^[a-z][a-z0-9._-]{1,79}$'
       OR p_consulta.limite IS NULL
       OR p_consulta.limite NOT BETWEEN 1 AND 100
       OR p_consulta.cursor IS NULL
       OR (
           p_consulta.cursor <> ''
           AND (
               p_consulta.cursor !~ '^[A-Za-z0-9_-]{43}$'
               OR pg_catalog.rtrim(pg_catalog.translate(pg_catalog.encode(
                   pg_catalog.decode(
                       pg_catalog.translate(p_consulta.cursor, '-_', '+/')
                       || '=', 'base64'
                   ), 'base64'
               ), '+/', '-_'), E'=\n') <> p_consulta.cursor
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta RRHH inválida';
    END IF;
    RETURN pg_catalog.convert_to(
        '{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v1"'
        || ',"version":1,"texto":'
        || vec_contratacion_temporal.texto_json_go_v1(p_consulta.texto)
        || ',"estado_clave":'
        || vec_contratacion_temporal.texto_json_go_v1(
            p_consulta.estado_clave
        ) || ',"fase_clave":'
        || vec_contratacion_temporal.texto_json_go_v1(
            p_consulta.fase_clave
        ) || ',"limite":' || p_consulta.limite::text || ',"cursor":'
        || vec_contratacion_temporal.texto_json_go_v1(p_consulta.cursor)
        || '}', 'UTF8'
    );
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'consulta RRHH inválida';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1
)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog AS $funcion$
BEGIN
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );
    RETURN pg_catalog.convert_to(
        '{"dominio":"vec.contratacion_temporal.filtros_rrhh.cuadro.v1"'
        || ',"version":1,"texto":'
        || vec_contratacion_temporal.texto_json_go_v1(p_consulta.texto)
        || ',"estado_clave":'
        || vec_contratacion_temporal.texto_json_go_v1(
            p_consulta.estado_clave
        ) || ',"fase_clave":'
        || vec_contratacion_temporal.texto_json_go_v1(
            p_consulta.fase_clave
        ) || ',"limite":' || p_consulta.limite::text || '}', 'UTF8'
    );
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1
)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog AS $funcion$
BEGIN
    IF p_consulta.expediente_ref IS NULL
       OR p_consulta.expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consulta.version_observada IS NULL
       OR p_consulta.version_observada NOT BETWEEN
          0 AND 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta RRHH inválida';
    END IF;
    RETURN pg_catalog.convert_to(
        '{"dominio":"vec.contratacion_temporal.consulta_rrhh.detalle.v1"'
        || ',"version":1,"expediente_ref":'
        || vec_contratacion_temporal.texto_json_go_v1(
            p_consulta.expediente_ref
        ) || ',"version_observada":'
        || p_consulta.version_observada::text || '}', 'UTF8'
    );
END
$funcion$;

-- Este canon encuadrado no serializa el contenido ni el token: enlaza las
-- huellas que 000041 deberá recalcular desde la proyección tipada.
CREATE FUNCTION
vec_contratacion_temporal.canon_resultado_consulta_rrhh_v1(
    p_evidencia vec_contratacion_temporal.evidencia_resultado_rrhh_v1
)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog AS $funcion$
BEGIN
    IF p_evidencia.tipo_consulta IS NULL
       OR p_evidencia.tipo_consulta NOT IN ('cuadro', 'detalle')
       OR p_evidencia.generada_en IS NULL
       OR p_evidencia.generada_en <>
          pg_catalog.date_trunc(
              'microseconds', p_evidencia.generada_en
          )
       OR p_evidencia.total IS NULL
       OR p_evidencia.total < 0
       OR p_evidencia.total > (
           CASE WHEN p_evidencia.tipo_consulta = 'detalle' THEN 1 ELSE 100 END
       )
       OR p_evidencia.contenido_huella_sha256 IS NULL
       OR p_evidencia.contenido_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.contenido_huella_sha256 =
          pg_catalog.repeat('0', 64)
       OR p_evidencia.cursor_huella_sha256 IS NULL
       OR (
           p_evidencia.cursor_huella_sha256 <> ''
           AND (
               p_evidencia.tipo_consulta <> 'cuadro'
               OR p_evidencia.total = 0
               OR p_evidencia.cursor_huella_sha256 !~ '^[0-9a-f]{64}$'
               OR p_evidencia.cursor_huella_sha256 =
                  pg_catalog.repeat('0', 64)
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'resultado RRHH inválido';
    END IF;
    RETURN pg_catalog.convert_to(
        'VEC-CT-RESULTADO-CONSULTA-RRHH-V1'
        || pg_catalog.chr(10), 'UTF8'
    )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_evidencia.tipo_consulta
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(
                p_evidencia.generada_en
            )
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_evidencia.total::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_evidencia.contenido_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_evidencia.cursor_huella_sha256
        );
END
$funcion$;

CREATE TRIGGER control_motor_consultas_rrhh_v1_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.control_motor_consultas_rrhh_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER control_motor_consultas_rrhh_v1_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.control_motor_consultas_rrhh_v1
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.control_motor_consultas_rrhh_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.control_motor_consultas_rrhh_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
ON vec_contratacion_temporal.control_motor_consultas_rrhh_v1
TO vec_contratacion_temporal_propietario
USING (true) WITH CHECK (true);

REVOKE ALL ON TABLE
    vec_contratacion_temporal.control_motor_consultas_rrhh_v1
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
REVOKE ALL ON TYPE
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.evidencia_resultado_rrhh_v1
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
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
    )
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

INSERT INTO vec_contratacion_temporal.control_motor_consultas_rrhh_v1
    (control, version_esquema, catalogo_huella_sha256, creada_en)
SELECT true, 1, pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.convert_to(pg_catalog.jsonb_build_object(
        'funciones', (
            SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                funcion.proname, pg_catalog.pg_get_function_identity_arguments(
                    funcion.oid
                ), pg_catalog.pg_get_function_result(funcion.oid),
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
                        pg_catalog.pg_get_triggerdef(disparador.oid, false)
                        ORDER BY disparador.tgname COLLATE "C"
                    ) FROM pg_catalog.pg_trigger disparador
                     WHERE disparador.tgrelid = tabla.oid
                       AND NOT disparador.tgisinternal
                ),
                (
                    SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                        politica.polname, politica.polcmd::text,
                        politica.polpermissive, politica.polroles,
                        pg_catalog.pg_get_expr(
                            politica.polqual, politica.polrelid, false
                        ), pg_catalog.pg_get_expr(
                            politica.polwithcheck,
                            politica.polrelid, false
                        )
                    ) ORDER BY politica.polname COLLATE "C")
                      FROM pg_catalog.pg_policy politica
                     WHERE politica.polrelid = tabla.oid
                )
            )
              FROM pg_catalog.pg_class tabla
             WHERE tabla.oid =
                   'vec_contratacion_temporal.'
                   'control_motor_consultas_rrhh_v1'::regclass
        )
    )::text, 'UTF8')
), 'hex'), pg_catalog.date_trunc(
    'microseconds', pg_catalog.clock_timestamp()
);

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 4,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 3;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 20,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 19;
COMMIT;
