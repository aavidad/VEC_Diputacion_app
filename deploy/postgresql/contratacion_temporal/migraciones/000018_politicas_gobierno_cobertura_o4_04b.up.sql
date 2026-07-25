-- O4-04B: políticas de decisión y actuación ligadas al catálogo durable.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000018_politicas_gobierno_o4_04b', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_catalogo'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_material_catalogo(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_politica'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para políticas O4-04B';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_material_politica(
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
    v_via record;
    v_comprobacion record;
    v_resultado record;
    v_publicada timestamptz;
    v_desde timestamptz;
    v_hasta timestamptz;
    v_prioridad integer := 0;
    v_vias text[] := ARRAY[]::text[];
    v_comprobaciones text[];
    v_resultados text[];
BEGIN
    IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion,
           ARRAY[
               'canon', 'catalogo', 'finalidad_clave', 'finalidad_ref',
               'huella_sha256', 'organizacion_ref', 'procedencia_ref',
               'publicada_en', 'referencia', 'version', 'vias', 'vigencia'
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
           p_publicacion -> 'vigencia', ARRAY['desde', 'hasta']::text[]
       )
       OR p_publicacion #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.politica-decision-cobertura'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{canon,version_esquema}', 1, 1)
       OR p_publicacion #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_publicacion ->> 'referencia') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'version', 1, 9007199254740991::numeric)
       OR (p_publicacion ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_publicacion ->> 'huella_sha256' = pg_catalog.repeat('0', 64)
       OR (p_publicacion #>> '{catalogo,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{catalogo,version}',
           1, 9007199254740991::numeric)
       OR (p_publicacion #>> '{catalogo,huella_sha256}') !~
          '^[a-f0-9]{64}$'
       OR p_publicacion #>> '{catalogo,huella_sha256}' =
          pg_catalog.repeat('0', 64)
       OR (p_publicacion ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'finalidad_clave') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_publicacion ->> 'finalidad_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'procedencia_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR pg_catalog.jsonb_typeof(p_publicacion -> 'vias') <> 'array'
       OR pg_catalog.jsonb_array_length(p_publicacion -> 'vias')
          NOT BETWEEN 1 AND 64
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'publicada_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,desde}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,hasta}', false
       ) THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_publicada := (p_publicacion ->> 'publicada_en')::timestamptz;
        v_desde := (p_publicacion #>> '{vigencia,desde}')::timestamptz;
        v_hasta := (p_publicacion #>> '{vigencia,hasta}')::timestamptz;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF v_hasta <= v_desde
       OR v_publicada > v_desde THEN
        RETURN NULL;
    END IF;
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{canon,dominio}'
    ) || pg_catalog.decode('0001', 'hex');
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'sha-256'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'referencia'
    ) || pg_catalog.int8send((p_publicacion ->> 'version')::bigint);
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{catalogo,referencia}'
    ) || pg_catalog.int8send(
             (p_publicacion #>> '{catalogo,version}')::bigint
         );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{catalogo,huella_sha256}'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'organizacion_ref'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'finalidad_clave'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'finalidad_ref'
    ) || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_publicada)
         )
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_desde)
         )
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_hasta)
         );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'procedencia_ref'
    ) || pg_catalog.int4send(
        pg_catalog.jsonb_array_length(p_publicacion -> 'vias')
    );
    FOR v_via IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(p_publicacion -> 'vias')
               WITH ORDINALITY AS v(valor, ordinalidad)
    LOOP
        IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
               v_via.valor,
               ARRAY['comprobaciones', 'prioridad', 'via_clave']::text[]
           )
           OR (v_via.valor ->> 'via_clave') !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR (v_via.valor ->> 'via_clave') = ANY(v_vias)
           OR NOT vec_contratacion_temporal
              .numero_entero_json_canonico_v2(
                  v_via.valor -> 'prioridad', 1, 65535)
           OR (v_via.valor ->> 'prioridad')::integer <= v_prioridad
           OR pg_catalog.jsonb_typeof(
                  v_via.valor -> 'comprobaciones'
              ) <> 'array'
           OR pg_catalog.jsonb_array_length(
                  v_via.valor -> 'comprobaciones'
              ) NOT BETWEEN 1 AND 32 THEN
            RETURN NULL;
        END IF;
        v_vias := pg_catalog.array_append(
            v_vias, v_via.valor ->> 'via_clave'
        );
        v_prioridad := (v_via.valor ->> 'prioridad')::integer;
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_via.valor ->> 'via_clave'
        ) || pg_catalog.decode(
                 pg_catalog.lpad(pg_catalog.to_hex(v_prioridad), 4, '0'),
                 'hex'
             )
          || pg_catalog.int4send(
                 pg_catalog.jsonb_array_length(
                     v_via.valor -> 'comprobaciones'
                 )
             );
        v_comprobaciones := ARRAY[]::text[];
        FOR v_comprobacion IN
            SELECT valor, ordinalidad
              FROM pg_catalog.jsonb_array_elements(
                       v_via.valor -> 'comprobaciones'
                   ) WITH ORDINALITY AS c(valor, ordinalidad)
        LOOP
            IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_comprobacion.valor,
                   ARRAY[
                       'clave', 'resultados_habilitantes',
                       'tratamiento_ausencia'
                   ]::text[]
               )
               OR (v_comprobacion.valor ->> 'clave') !~
                  '^[a-z][a-z0-9._-]{1,79}$'
               OR (v_comprobacion.valor ->> 'clave') =
                  ANY(v_comprobaciones)
               OR v_comprobacion.valor ->> 'tratamiento_ausencia'
                  NOT IN ('bloquea', 'admitida')
               OR pg_catalog.jsonb_typeof(
                      v_comprobacion.valor -> 'resultados_habilitantes'
                  ) <> 'array'
               OR pg_catalog.jsonb_array_length(
                      v_comprobacion.valor -> 'resultados_habilitantes'
                  ) NOT BETWEEN 1 AND 3 THEN
                RETURN NULL;
            END IF;
            v_comprobaciones := pg_catalog.array_append(
                v_comprobaciones, v_comprobacion.valor ->> 'clave'
            );
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, v_comprobacion.valor ->> 'clave'
            );
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material,
                v_comprobacion.valor ->> 'tratamiento_ausencia'
            ) || pg_catalog.int4send(
                     pg_catalog.jsonb_array_length(
                         v_comprobacion.valor -> 'resultados_habilitantes'
                     )
                 );
            v_resultados := ARRAY[]::text[];
            FOR v_resultado IN
                SELECT valor #>> '{}' AS valor, ordinalidad
                  FROM pg_catalog.jsonb_array_elements(
                           v_comprobacion.valor ->
                           'resultados_habilitantes'
                       ) WITH ORDINALITY AS r(valor, ordinalidad)
            LOOP
                IF v_resultado.valor NOT IN (
                       'afirmativa', 'negativa', 'no_aplica'
                   )
                   OR v_resultado.valor = ANY(v_resultados)
                   OR (
                       pg_catalog.cardinality(v_resultados) > 0
                       AND v_resultado.valor <=
                           v_resultados[
                               pg_catalog.cardinality(v_resultados)
                           ]
                   ) THEN
                    RETURN NULL;
                END IF;
                v_resultados := pg_catalog.array_append(
                    v_resultados, v_resultado.valor
                );
                v_material :=
                    vec_contratacion_temporal.gobi_o404b_texto_canon(
                        v_material, v_resultado.valor
                    );
            END LOOP;
        END LOOP;
    END LOOP;
    IF pg_catalog.octet_length(v_material) > 1048576 THEN
        RETURN NULL;
    END IF;
    RETURN v_material;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_material_actuacion(
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
    v_motivo jsonb;
    v_publicada timestamptz;
    v_desde timestamptz;
    v_hasta timestamptz;
    v_claves text[];
    v_equivalencia text := COALESCE(
        p_publicacion ->> 'equivalencia_motivos_ref', ''
    );
BEGIN
    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_publicacion) AS k(clave);
    IF v_claves NOT IN (
           ARRAY[
               'accion', 'canon', 'catalogo', 'estado_destino',
               'fase_destino', 'finalidad_autorizacion_vec',
               'finalidad_contratacion_clave',
               'finalidad_contratacion_ref',
               'huella_sha256', 'motivo_autorizacion_decidir',
               'motivo_autorizacion_rectificar', 'organizacion_ref',
               'politica', 'publicada_en', 'referencia',
               'unidad_ejecutora_ref', 'version', 'vigencia'
           ]::text[],
           ARRAY[
               'accion', 'canon', 'catalogo', 'equivalencia_motivos_ref',
               'estado_destino', 'fase_destino',
               'finalidad_autorizacion_vec',
               'finalidad_contratacion_clave', 'finalidad_contratacion_ref',
               'huella_sha256',
               'motivo_autorizacion_decidir',
               'motivo_autorizacion_rectificar', 'organizacion_ref',
               'politica', 'publicada_en', 'referencia',
               'unidad_ejecutora_ref', 'version', 'vigencia'
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
           p_publicacion -> 'vigencia', ARRAY['desde', 'hasta']::text[]
       )
       OR p_publicacion #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.politica-actuacion-cobertura'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{canon,version_esquema}', 1, 1)
       OR p_publicacion #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_publicacion ->> 'referencia') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'version', 1, 9007199254740991::numeric)
       OR (p_publicacion ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_publicacion ->> 'huella_sha256' = pg_catalog.repeat('0', 64)
       OR (p_publicacion ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_publicacion ->> 'accion' NOT IN (
          'contratacion_temporal.cobertura.decidir',
          'contratacion_temporal.cobertura.rectificar'
       )
       OR (p_publicacion ->> 'finalidad_contratacion_clave') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_publicacion ->> 'finalidad_contratacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'finalidad_autorizacion_vec') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_publicacion ->> 'unidad_ejecutora_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_publicacion ->> 'fase_destino') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR p_publicacion ->> 'estado_destino' NOT IN (
          'pendiente', 'en_curso', 'espera_externa', 'completado',
          'incidencia', 'cancelado'
       )
       OR (p_publicacion #>> '{catalogo,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{catalogo,version}',
           1, 9007199254740991::numeric)
       OR (p_publicacion #>> '{catalogo,huella_sha256}') !~
          '^[a-f0-9]{64}$'
       OR p_publicacion #>> '{catalogo,huella_sha256}' =
          pg_catalog.repeat('0', 64)
       OR (p_publicacion #>> '{politica,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{politica,version}',
           1, 9007199254740991::numeric)
       OR (p_publicacion #>> '{politica,huella_sha256}') !~
          '^[a-f0-9]{64}$'
       OR p_publicacion #>> '{politica,huella_sha256}' =
          pg_catalog.repeat('0', 64)
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'publicada_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,desde}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,hasta}', false
       ) THEN
        RETURN NULL;
    END IF;
    FOREACH v_motivo IN ARRAY ARRAY[
        p_publicacion -> 'motivo_autorizacion_decidir',
        p_publicacion -> 'motivo_autorizacion_rectificar'
    ] LOOP
        IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
               v_motivo,
               ARRAY[
                   'catalogo_huella_sha256', 'catalogo_id',
                   'catalogo_version', 'entrada_clave'
               ]::text[]
           )
           OR (v_motivo ->> 'catalogo_id') !~
              '^[a-z][a-z0-9._-]{0,127}$'
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
                  v_motivo -> 'catalogo_version', 1, 2147483647)
           OR (v_motivo ->> 'catalogo_huella_sha256') !~
              '^[a-f0-9]{64}$'
           OR v_motivo ->> 'catalogo_huella_sha256' =
              pg_catalog.repeat('0', 64)
           OR (v_motivo ->> 'entrada_clave') !~
              '^motivo_[a-f0-9]{32}$' THEN
            RETURN NULL;
        END IF;
    END LOOP;
    IF (
           p_publicacion -> 'motivo_autorizacion_decidir' =
           p_publicacion -> 'motivo_autorizacion_rectificar'
       ) <> (v_equivalencia <> '')
       OR (v_equivalencia <> '' AND v_equivalencia !~
           '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_publicada := (p_publicacion ->> 'publicada_en')::timestamptz;
        v_desde := (p_publicacion #>> '{vigencia,desde}')::timestamptz;
        v_hasta := (p_publicacion #>> '{vigencia,hasta}')::timestamptz;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF v_hasta <= v_desde OR v_publicada > v_desde THEN
        RETURN NULL;
    END IF;
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{canon,dominio}'
    ) || pg_catalog.int8send(1);
    FOREACH v_claves SLICE 1 IN ARRAY ARRAY[
        ARRAY[p_publicacion #>> '{canon,algoritmo}'],
        ARRAY[p_publicacion ->> 'referencia']
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_claves[1]
        );
    END LOOP;
    v_material := v_material
      || pg_catalog.int8send((p_publicacion ->> 'version')::bigint);
    FOREACH v_claves SLICE 1 IN ARRAY ARRAY[
        ARRAY[p_publicacion ->> 'organizacion_ref'],
        ARRAY[p_publicacion ->> 'accion'],
        ARRAY[p_publicacion #>> '{catalogo,referencia}']
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_claves[1]
        );
    END LOOP;
    v_material := v_material
      || pg_catalog.int8send(
             (p_publicacion #>> '{catalogo,version}')::bigint
         );
    FOREACH v_claves SLICE 1 IN ARRAY ARRAY[
        ARRAY[p_publicacion #>> '{catalogo,huella_sha256}'],
        ARRAY[p_publicacion #>> '{politica,referencia}']
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_claves[1]
        );
    END LOOP;
    v_material := v_material
      || pg_catalog.int8send(
             (p_publicacion #>> '{politica,version}')::bigint
         );
    FOREACH v_claves SLICE 1 IN ARRAY ARRAY[
        ARRAY[p_publicacion #>> '{politica,huella_sha256}'],
        ARRAY[p_publicacion ->> 'finalidad_contratacion_clave'],
        ARRAY[p_publicacion ->> 'finalidad_contratacion_ref'],
        ARRAY[p_publicacion ->> 'finalidad_autorizacion_vec'],
        ARRAY[p_publicacion ->> 'unidad_ejecutora_ref'],
        ARRAY[p_publicacion ->> 'fase_destino'],
        ARRAY[p_publicacion ->> 'estado_destino']
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_claves[1]
        );
    END LOOP;
    FOREACH v_motivo IN ARRAY ARRAY[
        p_publicacion -> 'motivo_autorizacion_decidir',
        p_publicacion -> 'motivo_autorizacion_rectificar'
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_motivo ->> 'catalogo_id'
        ) || pg_catalog.int8send(
                 (v_motivo ->> 'catalogo_version')::bigint
             );
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_motivo ->> 'catalogo_huella_sha256'
        );
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_motivo ->> 'entrada_clave'
        );
    END LOOP;
    FOREACH v_claves SLICE 1 IN ARRAY ARRAY[
        ARRAY[v_equivalencia],
        ARRAY[p_publicacion ->> 'publicada_en'],
        ARRAY[p_publicacion #>> '{vigencia,desde}'],
        ARRAY[p_publicacion #>> '{vigencia,hasta}']
    ] LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_claves[1]
        );
    END LOOP;
    RETURN v_material;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE TABLE vec_contratacion_temporal.gobi_o404b_politica (
    referencia text NOT NULL,
    version numeric(20, 0) NOT NULL,
    huella_sha256 text NOT NULL,
    catalogo_ref text NOT NULL,
    catalogo_version numeric(20, 0) NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    organizacion_ref text NOT NULL,
    finalidad_clave text NOT NULL,
    finalidad_ref text NOT NULL,
    publicacion_json jsonb NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    evento_ref text NOT NULL,
    secuencia bigint NOT NULL,
    PRIMARY KEY (referencia, version),
    UNIQUE (referencia, version, huella_sha256),
    FOREIGN KEY (catalogo_ref, catalogo_version, catalogo_huella_sha256)
        REFERENCES vec_contratacion_temporal.gobi_o404b_catalogo
            (referencia, version, huella_sha256),
    FOREIGN KEY (secuencia, evento_ref)
        REFERENCES vec_contratacion_temporal.gobi_o404b_evento
            (secuencia, evento_ref),
    CHECK (pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.gobi_o404b_material_politica(
            publicacion_json
        )
    ), 'hex') = huella_sha256),
    CHECK (publicacion_json ->> 'referencia' = referencia),
    CHECK ((publicacion_json ->> 'version')::numeric = version),
    CHECK (publicacion_json ->> 'huella_sha256' = huella_sha256),
    CHECK (publicacion_json #>> '{catalogo,referencia}' = catalogo_ref),
    CHECK ((publicacion_json #>> '{catalogo,version}')::numeric =
           catalogo_version),
    CHECK (publicacion_json #>> '{catalogo,huella_sha256}' =
           catalogo_huella_sha256),
    CHECK (publicacion_json ->> 'organizacion_ref' = organizacion_ref),
    CHECK (publicacion_json ->> 'finalidad_clave' = finalidad_clave),
    CHECK (publicacion_json ->> 'finalidad_ref' = finalidad_ref)
);

CREATE TABLE vec_contratacion_temporal.gobi_o404b_actuacion (
    referencia text NOT NULL,
    version numeric(20, 0) NOT NULL,
    huella_sha256 text NOT NULL,
    organizacion_ref text NOT NULL,
    accion text NOT NULL,
    catalogo_ref text NOT NULL,
    catalogo_version numeric(20, 0) NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    politica_ref text NOT NULL,
    politica_version numeric(20, 0) NOT NULL,
    politica_huella_sha256 text NOT NULL,
    publicacion_json jsonb NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    evento_ref text NOT NULL,
    secuencia bigint NOT NULL,
    PRIMARY KEY (referencia, version),
    UNIQUE (referencia, version, huella_sha256),
    FOREIGN KEY (catalogo_ref, catalogo_version, catalogo_huella_sha256)
        REFERENCES vec_contratacion_temporal.gobi_o404b_catalogo
            (referencia, version, huella_sha256),
    FOREIGN KEY (politica_ref, politica_version, politica_huella_sha256)
        REFERENCES vec_contratacion_temporal.gobi_o404b_politica
            (referencia, version, huella_sha256),
    FOREIGN KEY (secuencia, evento_ref)
        REFERENCES vec_contratacion_temporal.gobi_o404b_evento
            (secuencia, evento_ref),
    CHECK (accion IN (
        'contratacion_temporal.cobertura.decidir',
        'contratacion_temporal.cobertura.rectificar'
    )),
    CHECK (pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.gobi_o404b_material_actuacion(
            publicacion_json
        )
    ), 'hex') = huella_sha256),
    CHECK (publicacion_json ->> 'referencia' = referencia),
    CHECK ((publicacion_json ->> 'version')::numeric = version),
    CHECK (publicacion_json ->> 'huella_sha256' = huella_sha256),
    CHECK (publicacion_json ->> 'organizacion_ref' = organizacion_ref),
    CHECK (publicacion_json ->> 'accion' = accion)
);

CREATE TABLE vec_contratacion_temporal.gobi_o404b_actual (
    organizacion_ref text NOT NULL,
    accion text NOT NULL,
    actuacion_ref text NOT NULL,
    actuacion_version numeric(20, 0) NOT NULL,
    actuacion_huella_sha256 text NOT NULL,
    secuencia bigint NOT NULL,
    evento_ref text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (organizacion_ref, accion),
    FOREIGN KEY (actuacion_ref, actuacion_version, actuacion_huella_sha256)
        REFERENCES vec_contratacion_temporal.gobi_o404b_actuacion
            (referencia, version, huella_sha256),
    FOREIGN KEY (secuencia, evento_ref)
        REFERENCES vec_contratacion_temporal.gobi_o404b_evento
            (secuencia, evento_ref),
    CHECK (accion IN (
        'contratacion_temporal.cobertura.decidir',
        'contratacion_temporal.cobertura.rectificar'
    ))
);

CREATE TABLE vec_contratacion_temporal.gobi_o404b_retirada (
    organizacion_ref text NOT NULL,
    accion text NOT NULL,
    actuacion_ref text NOT NULL,
    actuacion_version numeric(20, 0) NOT NULL,
    actuacion_huella_sha256 text NOT NULL,
    retirada_en timestamptz(6) NOT NULL,
    secuencia bigint NOT NULL,
    evento_ref text NOT NULL,
    PRIMARY KEY (organizacion_ref, accion, actuacion_ref, actuacion_version),
    FOREIGN KEY (actuacion_ref, actuacion_version, actuacion_huella_sha256)
        REFERENCES vec_contratacion_temporal.gobi_o404b_actuacion
            (referencia, version, huella_sha256),
    FOREIGN KEY (secuencia, evento_ref)
        REFERENCES vec_contratacion_temporal.gobi_o404b_evento
            (secuencia, evento_ref)
);

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_validar_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF (
           TG_OP = 'UPDATE'
           AND (
               NEW.organizacion_ref <> OLD.organizacion_ref
               OR NEW.accion <> OLD.accion
               OR NEW.secuencia <= OLD.secuencia
               OR NEW.actualizada_en < OLD.actualizada_en
           )
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.gobi_o404b_evento e
            WHERE e.secuencia = NEW.secuencia
              AND e.evento_ref = NEW.evento_ref
              AND e.tipo = 'publicacion'
              AND e.contenido_evento #>> '{actuacion,organizacion_ref}' =
                  NEW.organizacion_ref
              AND e.contenido_evento #>> '{actuacion,accion}' = NEW.accion
              AND e.contenido_evento #>> '{actuacion,referencia}' =
                  NEW.actuacion_ref
              AND (e.contenido_evento #>> '{actuacion,version}')::numeric =
                  NEW.actuacion_version
              AND e.contenido_evento
                      #>> '{actuacion,huella_sha256}' =
                  NEW.actuacion_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de puntero O4-04B no atómico';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_validar_checkpoint()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF NEW.control IS DISTINCT FROM OLD.control
       OR NEW.ultima_secuencia <> OLD.ultima_secuencia + 1
       OR NEW.actualizado_en < OLD.actualizado_en
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.gobi_o404b_evento e
            WHERE e.secuencia = NEW.ultima_secuencia
              AND e.evento_ref = NEW.ultimo_evento_ref
              AND e.huella_evento_sha256 =
                  NEW.ultima_huella_evento_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de checkpoint O4-04B no atómico';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_avance
BEFORE INSERT OR UPDATE ON vec_contratacion_temporal.gobi_o404b_actual
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_validar_actual();
CREATE TRIGGER bloquear_borrado
BEFORE DELETE ON vec_contratacion_temporal.gobi_o404b_actual
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER bloquear_truncado
BEFORE TRUNCATE ON vec_contratacion_temporal.gobi_o404b_actual
FOR EACH STATEMENT EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER validar_avance
BEFORE UPDATE ON vec_contratacion_temporal.gobi_o404b_checkpoint
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_validar_checkpoint();
CREATE TRIGGER bloquear_insercion_borrado
BEFORE INSERT OR DELETE ON vec_contratacion_temporal.gobi_o404b_checkpoint
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER bloquear_truncado
BEFORE TRUNCATE ON vec_contratacion_temporal.gobi_o404b_checkpoint
FOR EACH STATEMENT EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();

DO $protecciones$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'gobi_o404b_politica', 'gobi_o404b_actuacion',
        'gobi_o404b_retirada'
    ] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_mutacion BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.gobi_o404b_bloquear_inmutable()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_truncado BEFORE TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.gobi_o404b_bloquear_inmutable()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',
            v_tabla
        );
    END LOOP;
END
$protecciones$;

ALTER TABLE vec_contratacion_temporal.gobi_o404b_actual ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_actual FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON
    vec_contratacion_temporal.gobi_o404b_actual
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);

REVOKE ALL ON
    vec_contratacion_temporal.gobi_o404b_politica,
    vec_contratacion_temporal.gobi_o404b_actuacion,
    vec_contratacion_temporal.gobi_o404b_actual,
    vec_contratacion_temporal.gobi_o404b_retirada
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contratacion_temporal
FROM PUBLIC;

COMMIT;
