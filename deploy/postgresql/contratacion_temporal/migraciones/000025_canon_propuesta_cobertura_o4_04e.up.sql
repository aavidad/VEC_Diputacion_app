-- O4-04E/2: canon SQL V1 de la propuesta de decisión de cobertura.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 4
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 4
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_texto_canon(bytea,text)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_microsegundos(timestamptz)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para canon de propuesta O4-04E';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(
    p_publicacion jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_material bytea := ''::bytea;
    v_resultados jsonb;
    v_evaluaciones jsonb;
    v_resultado record;
    v_evidencia record;
    v_evaluacion record;
    v_clave text;
    v_lista text;
    v_inicio timestamptz;
    v_fin timestamptz;
    v_generada timestamptz;
    v_valida_hasta timestamptz;
    v_total_evidencias integer := 0;
    v_claves text[];
BEGIN
    IF pg_catalog.pg_column_size(p_publicacion) > 1048576
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion,
           ARRAY[
               'analisis_huella_sha256', 'analisis_ref', 'canon',
               'catalogo', 'categoria_ref', 'estado', 'evaluaciones',
               'expediente_ref', 'finalidad_clave', 'finalidad_ref',
               'generada_en', 'huella_sha256', 'organizacion_ref',
               'periodo', 'politica', 'preparacion_evidencias_huella_sha256',
               'preparacion_evidencias_ref', 'referencia', 'resultados',
               'valida_hasta', 'version_expediente', 'via_propuesta'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'canon',
           ARRAY['algoritmo', 'dominio', 'version_esquema']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'catalogo',
           ARRAY['huella_sha256', 'referencia', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'politica',
           ARRAY['huella_sha256', 'referencia', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )
       OR p_publicacion #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.propuesta-decision-cobertura'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{canon,version_esquema}', 1, 1
       )
       OR p_publicacion #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_publicacion ->> 'referencia') !~
          '^propuesta-cobertura:sha256:[a-f0-9]{64}$'
       OR (p_publicacion ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_publicacion ->> 'referencia' <>
          'propuesta-cobertura:sha256:' ||
          (p_publicacion ->> 'huella_sha256')
       OR (p_publicacion ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'expediente_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'version_expediente',
           1, 9007199254740991::numeric
       )
       OR (p_publicacion ->> 'analisis_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'preparacion_evidencias_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'finalidad_clave') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_publicacion ->> 'finalidad_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'categoria_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion #>> '{catalogo,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion #>> '{politica,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{catalogo,version}',
           1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{politica,version}',
           1, 9007199254740991::numeric
       )
       OR (p_publicacion ->> 'estado') NOT IN (
           'viable', 'incompleta', 'conflictiva', 'sin_via'
       )
       OR (
           (p_publicacion ->> 'estado') = 'viable'
           AND (p_publicacion ->> 'via_propuesta') !~
               '^[a-z][a-z0-9._-]{1,79}$'
       )
       OR (
           (p_publicacion ->> 'estado') <> 'viable'
           AND (p_publicacion ->> 'via_propuesta') <> ''
       ) THEN
        RETURN NULL;
    END IF;

    FOREACH v_clave IN ARRAY ARRAY[
        p_publicacion ->> 'huella_sha256',
        p_publicacion ->> 'analisis_huella_sha256',
        p_publicacion ->> 'preparacion_evidencias_huella_sha256',
        p_publicacion #>> '{catalogo,huella_sha256}',
        p_publicacion #>> '{politica,huella_sha256}'
    ]::text[]
    LOOP
        IF v_clave !~ '^[a-f0-9]{64}$'
           OR v_clave = pg_catalog.repeat('0', 64) THEN
            RETURN NULL;
        END IF;
    END LOOP;

    IF NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{periodo,inicio}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{periodo,fin}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'generada_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'valida_hasta', false
       ) THEN
        RETURN NULL;
    END IF;
    v_inicio := (p_publicacion #>> '{periodo,inicio}')::timestamptz;
    v_fin := (p_publicacion #>> '{periodo,fin}')::timestamptz;
    v_generada := (p_publicacion ->> 'generada_en')::timestamptz;
    v_valida_hasta := (p_publicacion ->> 'valida_hasta')::timestamptz;
    IF v_inicio <> pg_catalog.date_trunc('day', v_inicio)
       OR v_fin <> pg_catalog.date_trunc('day', v_fin)
       OR v_fin < v_inicio
       OR v_fin > v_inicio + interval '100 years'
       OR v_valida_hasta <= v_generada THEN
        RETURN NULL;
    END IF;

    v_resultados := p_publicacion -> 'resultados';
    v_evaluaciones := p_publicacion -> 'evaluaciones';
    IF pg_catalog.jsonb_typeof(v_resultados) <> 'array'
       OR pg_catalog.jsonb_array_length(v_resultados) > 512
       OR pg_catalog.jsonb_typeof(v_evaluaciones) <> 'array'
       OR pg_catalog.jsonb_array_length(v_evaluaciones) > 64 THEN
        RETURN NULL;
    END IF;

    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{canon,dominio}'
    ) || pg_catalog.decode('0001', 'hex');
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'sha-256'
    );
    FOREACH v_clave IN ARRAY ARRAY[
        p_publicacion ->> 'organizacion_ref',
        p_publicacion ->> 'expediente_ref'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_clave
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_publicacion ->> 'version_expediente')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        p_publicacion ->> 'analisis_ref',
        p_publicacion ->> 'analisis_huella_sha256',
        p_publicacion ->> 'preparacion_evidencias_ref',
        p_publicacion ->> 'preparacion_evidencias_huella_sha256',
        p_publicacion #>> '{catalogo,referencia}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_clave
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_publicacion #>> '{catalogo,version}')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        p_publicacion #>> '{catalogo,huella_sha256}',
        p_publicacion #>> '{politica,referencia}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_clave
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_publicacion #>> '{politica,version}')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        p_publicacion #>> '{politica,huella_sha256}',
        p_publicacion ->> 'finalidad_clave',
        p_publicacion ->> 'finalidad_ref',
        p_publicacion ->> 'categoria_ref'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_clave
        );
    END LOOP;
    v_material := v_material
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_inicio)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_fin)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_generada)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_valida_hasta)
        );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'estado'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'via_propuesta'
    ) || pg_catalog.int4send(pg_catalog.jsonb_array_length(v_resultados));

    FOR v_resultado IN
        SELECT valor
          FROM pg_catalog.jsonb_array_elements(v_resultados) AS r(valor)
    LOOP
        IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
               v_resultado.valor, ARRAY['clave', 'evidencias']::text[]
           )
           OR (v_resultado.valor ->> 'clave') !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR pg_catalog.jsonb_typeof(
                  v_resultado.valor -> 'evidencias'
              ) <> 'array'
           OR pg_catalog.jsonb_array_length(
                  v_resultado.valor -> 'evidencias'
              ) NOT BETWEEN 1 AND 4 THEN
            RETURN NULL;
        END IF;
        v_total_evidencias := v_total_evidencias +
            pg_catalog.jsonb_array_length(v_resultado.valor -> 'evidencias');
        IF v_total_evidencias > 1024 THEN
            RETURN NULL;
        END IF;
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_resultado.valor ->> 'clave'
        ) || pg_catalog.int4send(
            pg_catalog.jsonb_array_length(v_resultado.valor -> 'evidencias')
        );
        FOR v_evidencia IN
            SELECT valor
              FROM pg_catalog.jsonb_array_elements(
                       v_resultado.valor -> 'evidencias'
                   ) AS e(valor)
        LOOP
            IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_evidencia.valor,
                   ARRAY[
                       'evaluada_en', 'fuente_ref', 'recibo_ref', 'resultado'
                   ]::text[]
               )
               OR (v_evidencia.valor ->> 'resultado') NOT IN (
                   'afirmativa', 'negativa', 'no_aplica', 'no_consta'
               )
               OR (v_evidencia.valor ->> 'fuente_ref') !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR (v_evidencia.valor ->> 'recibo_ref') !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR NOT vec_contratacion_temporal
                  .gobi_o404b_instante_texto_valido(
                      v_evidencia.valor ->> 'evaluada_en', false
                  ) THEN
                RETURN NULL;
            END IF;
            FOREACH v_clave IN ARRAY ARRAY[
                v_evidencia.valor ->> 'resultado',
                v_evidencia.valor ->> 'fuente_ref',
                v_evidencia.valor ->> 'recibo_ref'
            ]::text[]
            LOOP
                v_material :=
                    vec_contratacion_temporal.gobi_o404b_texto_canon(
                        v_material, v_clave
                    );
            END LOOP;
            v_material := v_material || pg_catalog.int8send(
                vec_contratacion_temporal.gobi_o404b_microsegundos(
                    (v_evidencia.valor ->> 'evaluada_en')::timestamptz
                )
            );
        END LOOP;
    END LOOP;

    v_material := v_material ||
        pg_catalog.int4send(pg_catalog.jsonb_array_length(v_evaluaciones));
    FOR v_evaluacion IN
        SELECT valor
          FROM pg_catalog.jsonb_array_elements(v_evaluaciones) AS e(valor)
    LOOP
        SELECT pg_catalog.array_agg(clave ORDER BY clave)
          INTO v_claves
          FROM pg_catalog.jsonb_object_keys(v_evaluacion.valor) AS k(clave);
        IF NOT (
               v_evaluacion.valor ?& ARRAY[
                   'estado', 'prioridad', 'via_clave'
               ]
           )
           OR EXISTS (
               SELECT 1 FROM pg_catalog.unnest(v_claves) AS k(clave)
                WHERE clave <> ALL(ARRAY[
                    'ausencias_admitidas', 'ausencias_bloqueantes',
                    'conflictos', 'estado', 'no_habilitantes', 'prioridad',
                    'resultados_omitidos', 'via_clave'
                ]::text[])
           )
           OR (v_evaluacion.valor ->> 'via_clave') !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               v_evaluacion.valor -> 'prioridad', 0, 65535
           )
           OR (v_evaluacion.valor ->> 'estado') NOT IN (
               'viable', 'incompleta', 'conflictiva', 'no_viable'
           ) THEN
            RETURN NULL;
        END IF;
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_evaluacion.valor ->> 'via_clave'
        ) || pg_catalog.decode(
            pg_catalog.lpad(
                pg_catalog.to_hex(
                    (v_evaluacion.valor ->> 'prioridad')::integer
                ), 4, '0'
            ), 'hex'
        );
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_evaluacion.valor ->> 'estado'
        );
        FOREACH v_lista IN ARRAY ARRAY[
            'resultados_omitidos', 'ausencias_bloqueantes',
            'ausencias_admitidas', 'no_habilitantes', 'conflictos'
        ]::text[]
        LOOP
            IF v_evaluacion.valor ? v_lista THEN
                IF pg_catalog.jsonb_typeof(
                       v_evaluacion.valor -> v_lista
                   ) <> 'array'
                   OR pg_catalog.jsonb_array_length(
                       v_evaluacion.valor -> v_lista
                   ) > 512 THEN
                    RETURN NULL;
                END IF;
            END IF;
            v_material := v_material || pg_catalog.int4send(
                CASE WHEN v_evaluacion.valor ? v_lista
                     THEN pg_catalog.jsonb_array_length(
                         v_evaluacion.valor -> v_lista
                     )
                     ELSE 0 END
            );
            IF v_evaluacion.valor ? v_lista THEN
                FOR v_clave IN
                    SELECT pg_catalog.jsonb_array_elements_text(
                        v_evaluacion.valor -> v_lista
                    )
                LOOP
                    IF v_clave !~ '^[a-z][a-z0-9._-]{1,79}$' THEN
                        RETURN NULL;
                    END IF;
                    v_material :=
                        vec_contratacion_temporal.gobi_o404b_texto_canon(
                            v_material, v_clave
                        );
                END LOOP;
            END IF;
        END LOOP;
    END LOOP;
    IF pg_catalog.octet_length(v_material) > 1048576 THEN
        RETURN NULL;
    END IF;
    RETURN v_material;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_propuesta_cobertura_exacta_v1(
    p_publicacion jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT vec_contratacion_temporal
               .o404e_material_propuesta_cobertura_v1(p_publicacion)
               IS NOT NULL
       AND pg_catalog.encode(
               pg_catalog.sha256(
                   vec_contratacion_temporal
                       .o404e_material_propuesta_cobertura_v1(p_publicacion)
               ), 'hex'
           ) = p_publicacion ->> 'huella_sha256';
END;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb),
    vec_contratacion_temporal.o404e_propuesta_cobertura_exacta_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 5,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 4;

COMMIT;
