-- CT000141: requisitos documentales y de datos dentro del catálogo O4-04B.
-- La rama V1 conserva exactamente la definición de 000017; ninguna fila
-- histórica se modifica. El CHECK y publicar siguen usando el mismo OID.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones', 0
));
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
         'vec_contratacion_temporal.gobi_o404b_material_catalogo(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
         'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
            WHERE p.oid = pg_catalog.to_regprocedure(
              'vec_contratacion_temporal.gobi_o404b_material_catalogo(jsonb)'
            )
              AND p.proowner =
                  'vec_contratacion_temporal_propietario'::pg_catalog.regrole
              AND p.provolatile = 'i'
              AND p.proisstrict
              AND NOT p.prosecdef
              AND pg_catalog.encode(pg_catalog.sha256(
                    pg_catalog.convert_to(p.prosrc, 'UTF8')
                  ), 'hex') =
                  'daac1fec7f04618a41337ea0d6d6bff494178da52dcbd51e84f50ab0868c15c6'
       )
       OR pg_catalog.to_regprocedure(
         'vec_contratacion_temporal.gobi_o404b_material_catalogo_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
         'vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='estado incompatible para CT000141';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_material_catalogo_v1(
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
    v_publicado timestamptz;
    v_desde timestamptz;
    v_hasta timestamptz;
    v_orden_anterior integer := 0;
    v_orden_comprobacion integer;
    v_total integer := 0;
    v_vistas text[] := ARRAY[]::text[];
    v_comprobaciones text[] := ARRAY[]::text[];
    v_procedencias jsonb := '{}'::jsonb;
BEGIN
    IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion,
           ARRAY[
               'canon', 'huella_sha256', 'procedencia_ref', 'publicado_en',
               'referencia', 'version', 'vias', 'vigencia'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'canon',
           ARRAY['algoritmo', 'dominio', 'version_esquema']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'vigencia', ARRAY['desde', 'hasta']::text[]
       )
       OR p_publicacion #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.catalogo-vias-cobertura'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{canon,version_esquema}', 1, 1
       )
       OR p_publicacion #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_publicacion ->> 'referencia') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'version',
           1,
           9007199254740991::numeric
       )
       OR (p_publicacion ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_publicacion ->> 'huella_sha256' =
          pg_catalog.repeat('0', 64)
       OR (p_publicacion ->> 'procedencia_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR pg_catalog.jsonb_typeof(p_publicacion -> 'vias') <> 'array'
       OR pg_catalog.jsonb_array_length(p_publicacion -> 'vias')
          NOT BETWEEN 1 AND 64
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'publicado_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,desde}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,hasta}', true
       )
       THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_publicado := (p_publicacion ->> 'publicado_en')::timestamptz;
        v_desde := (p_publicacion #>> '{vigencia,desde}')::timestamptz;
        IF p_publicacion #>> '{vigencia,hasta}' =
           '0001-01-01T00:00:00Z' THEN
            v_hasta := NULL;
        ELSE
            v_hasta := (p_publicacion #>> '{vigencia,hasta}')::timestamptz;
        END IF;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF pg_catalog.date_trunc('microseconds', v_publicado) <> v_publicado
       OR pg_catalog.date_trunc('microseconds', v_desde) <> v_desde
       OR (v_hasta IS NOT NULL AND (
           pg_catalog.date_trunc('microseconds', v_hasta) <> v_hasta
           OR v_hasta <= v_desde
       )) THEN
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
    ) || pg_catalog.int8send((p_publicacion ->> 'version')::bigint)
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_publicado)
         )
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_desde)
         )
      || CASE WHEN v_hasta IS NULL THEN E'\\x00'::bytea
              ELSE E'\\x01'::bytea || pg_catalog.int8send(
                  vec_contratacion_temporal.gobi_o404b_microsegundos(v_hasta)
              ) END;
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
               ARRAY['clave', 'comprobaciones', 'orden']::text[]
           )
           OR (v_via.valor ->> 'clave') !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR (v_via.valor ->> 'clave') = ANY(v_vistas)
           OR NOT vec_contratacion_temporal
              .numero_entero_json_canonico_v2(
                  v_via.valor -> 'orden', 1, 65535
              )
           OR (v_via.valor ->> 'orden')::integer <= v_orden_anterior
           OR pg_catalog.jsonb_typeof(
                  v_via.valor -> 'comprobaciones'
              ) <> 'array'
           OR pg_catalog.jsonb_array_length(
                  v_via.valor -> 'comprobaciones'
              ) NOT BETWEEN 1 AND 32 THEN
            RETURN NULL;
        END IF;
        v_vistas := pg_catalog.array_append(
            v_vistas, v_via.valor ->> 'clave'
        );
        v_orden_anterior := (v_via.valor ->> 'orden')::integer;
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_via.valor ->> 'clave'
        ) || pg_catalog.decode(
                 pg_catalog.lpad(pg_catalog.to_hex(v_orden_anterior), 4, '0'),
                 'hex'
             )
          || pg_catalog.int4send(
                 pg_catalog.jsonb_array_length(
                     v_via.valor -> 'comprobaciones'
                 )
             );
        v_orden_comprobacion := 0;
        v_comprobaciones := ARRAY[]::text[];
        FOR v_comprobacion IN
            SELECT valor, ordinalidad
              FROM pg_catalog.jsonb_array_elements(
                       v_via.valor -> 'comprobaciones'
                   ) WITH ORDINALITY AS c(valor, ordinalidad)
        LOOP
            IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_comprobacion.valor,
                   ARRAY['clave', 'obligatoria', 'orden', 'procedencia']::text[]
               )
               OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_comprobacion.valor -> 'procedencia',
                   ARRAY['clave', 'definicion_fuente_ref']::text[]
               )
               OR (v_comprobacion.valor ->> 'clave') !~
                  '^[a-z][a-z0-9._-]{1,79}$'
               OR (v_comprobacion.valor ->> 'clave') =
                  ANY(v_comprobaciones)
               OR pg_catalog.jsonb_typeof(
                      v_comprobacion.valor -> 'obligatoria'
                  ) <> 'boolean'
               OR NOT vec_contratacion_temporal
                  .numero_entero_json_canonico_v2(
                      v_comprobacion.valor -> 'orden', 1, 65535
                  )
               OR (v_comprobacion.valor ->> 'orden')::integer <=
                  v_orden_comprobacion
               OR (v_comprobacion.valor #>> '{procedencia,clave}') !~
                  '^[a-z][a-z0-9._-]{1,79}$'
               OR (v_comprobacion.valor
                       #>> '{procedencia,definicion_fuente_ref}') !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RETURN NULL;
            END IF;
            IF v_procedencias ? (v_comprobacion.valor ->> 'clave')
               AND v_procedencias -> (v_comprobacion.valor ->> 'clave')
                   IS DISTINCT FROM
                   v_comprobacion.valor -> 'procedencia' THEN
                RETURN NULL;
            END IF;
            v_procedencias := pg_catalog.jsonb_set(
                v_procedencias,
                ARRAY[v_comprobacion.valor ->> 'clave'],
                v_comprobacion.valor -> 'procedencia',
                true
            );
            v_comprobaciones := pg_catalog.array_append(
                v_comprobaciones, v_comprobacion.valor ->> 'clave'
            );
            v_orden_comprobacion :=
                (v_comprobacion.valor ->> 'orden')::integer;
            v_total := v_total + 1;
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, v_comprobacion.valor ->> 'clave'
            ) || pg_catalog.decode(
                     pg_catalog.lpad(
                         pg_catalog.to_hex(v_orden_comprobacion), 4, '0'
                     ),
                     'hex'
                 )
              || CASE WHEN (v_comprobacion.valor ->> 'obligatoria')::boolean
                      THEN E'\\x01'::bytea ELSE E'\\x00'::bytea END;
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material,
                v_comprobacion.valor #>> '{procedencia,clave}'
            );
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material,
                v_comprobacion.valor
                    #>> '{procedencia,definicion_fuente_ref}'
            );
        END LOOP;
    END LOOP;
    IF v_total > 512 OR pg_catalog.octet_length(v_material) > 1048576 THEN
        RETURN NULL;
    END IF;
    RETURN v_material;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(
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
    v_requisito record;
    v_tipo text;
    v_orden_requisito integer;
    v_claves_requisito text[];
    v_total_requisitos integer := 0;
    v_publicado timestamptz;
    v_desde timestamptz;
    v_hasta timestamptz;
    v_orden_anterior integer := 0;
    v_orden_comprobacion integer;
    v_total integer := 0;
    v_vistas text[] := ARRAY[]::text[];
    v_comprobaciones text[] := ARRAY[]::text[];
    v_procedencias jsonb := '{}'::jsonb;
    v_claves_via text[];
BEGIN
    IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion,
           CASE WHEN p_publicacion ? 'es_ejemplo' THEN ARRAY[
               'canon', 'es_ejemplo', 'huella_sha256', 'procedencia_ref',
               'publicado_en', 'referencia', 'version', 'vias', 'vigencia'
           ]::text[] ELSE ARRAY[
               'canon', 'huella_sha256', 'procedencia_ref', 'publicado_en',
               'referencia', 'version', 'vias', 'vigencia'
           ]::text[] END
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'canon',
           ARRAY['algoritmo', 'dominio', 'version_esquema']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_publicacion -> 'vigencia', ARRAY['desde', 'hasta']::text[]
       )
       OR pg_catalog.jsonb_typeof(
            p_publicacion #> '{canon,dominio}'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion #> '{canon,algoritmo}'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion -> 'referencia'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion -> 'huella_sha256'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion -> 'procedencia_ref'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion -> 'publicado_en'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion #> '{vigencia,desde}'
          ) IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(
            p_publicacion #> '{vigencia,hasta}'
          ) IS DISTINCT FROM 'string'
       OR (p_publicacion ? 'es_ejemplo' AND
           p_publicacion -> 'es_ejemplo' IS DISTINCT FROM 'true'::jsonb)
       OR p_publicacion #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.catalogo-vias-cobertura'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion #> '{canon,version_esquema}', 2, 2
       )
       OR p_publicacion #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_publicacion ->> 'referencia') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_publicacion -> 'version',
           1,
           9007199254740991::numeric
       )
       OR (p_publicacion ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_publicacion ->> 'huella_sha256' =
          pg_catalog.repeat('0', 64)
       OR (p_publicacion ->> 'procedencia_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR pg_catalog.jsonb_typeof(p_publicacion -> 'vias') <> 'array'
       OR pg_catalog.jsonb_array_length(p_publicacion -> 'vias')
          NOT BETWEEN 1 AND 64
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion ->> 'publicado_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,desde}', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{vigencia,hasta}', true
       )
       THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_publicado := (p_publicacion ->> 'publicado_en')::timestamptz;
        v_desde := (p_publicacion #>> '{vigencia,desde}')::timestamptz;
        IF p_publicacion #>> '{vigencia,hasta}' =
           '0001-01-01T00:00:00Z' THEN
            v_hasta := NULL;
        ELSE
            v_hasta := (p_publicacion #>> '{vigencia,hasta}')::timestamptz;
        END IF;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF pg_catalog.date_trunc('microseconds', v_publicado) <> v_publicado
       OR pg_catalog.date_trunc('microseconds', v_desde) <> v_desde
       OR (v_hasta IS NOT NULL AND (
           pg_catalog.date_trunc('microseconds', v_hasta) <> v_hasta
           OR v_hasta <= v_desde
       )) THEN
        RETURN NULL;
    END IF;
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion #>> '{canon,dominio}'
    ) || pg_catalog.decode('0002', 'hex');
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'sha-256'
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'referencia'
    ) || pg_catalog.int8send((p_publicacion ->> 'version')::bigint)
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_publicado)
         )
      || pg_catalog.int8send(
             vec_contratacion_temporal.gobi_o404b_microsegundos(v_desde)
         )
      || CASE WHEN v_hasta IS NULL THEN E'\\x00'::bytea
              ELSE E'\\x01'::bytea || pg_catalog.int8send(
                  vec_contratacion_temporal.gobi_o404b_microsegundos(v_hasta)
              ) END;
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_publicacion ->> 'procedencia_ref'
    ) || CASE WHEN p_publicacion ? 'es_ejemplo'
              THEN pg_catalog.decode('01','hex')
              ELSE pg_catalog.decode('00','hex') END
      || pg_catalog.int4send(
        pg_catalog.jsonb_array_length(p_publicacion -> 'vias')
    );
    FOR v_via IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(p_publicacion -> 'vias')
               WITH ORDINALITY AS v(valor, ordinalidad)
    LOOP
        v_claves_via := ARRAY['clave', 'comprobaciones', 'orden']::text[];
        IF v_via.valor ? 'datos' THEN
            v_claves_via := pg_catalog.array_append(v_claves_via, 'datos');
        END IF;
        IF v_via.valor ? 'documentos' THEN
            v_claves_via := pg_catalog.array_append(
                v_claves_via, 'documentos'
            );
        END IF;
        SELECT pg_catalog.array_agg(k ORDER BY k) INTO v_claves_via
          FROM pg_catalog.unnest(v_claves_via) AS k;
        IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
               v_via.valor, v_claves_via
           )
           OR pg_catalog.jsonb_typeof(v_via.valor -> 'clave')
              IS DISTINCT FROM 'string'
           OR (v_via.valor ->> 'clave') !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR (v_via.valor ->> 'clave') = ANY(v_vistas)
           OR NOT vec_contratacion_temporal
              .numero_entero_json_canonico_v2(
                  v_via.valor -> 'orden', 1, 65535
              )
           OR (v_via.valor ->> 'orden')::integer <= v_orden_anterior
           OR pg_catalog.jsonb_typeof(
                  v_via.valor -> 'comprobaciones'
              ) <> 'array'
           OR pg_catalog.jsonb_array_length(
                  v_via.valor -> 'comprobaciones'
              ) NOT BETWEEN 1 AND 32 THEN
            RETURN NULL;
        END IF;
        v_vistas := pg_catalog.array_append(
            v_vistas, v_via.valor ->> 'clave'
        );
        v_orden_anterior := (v_via.valor ->> 'orden')::integer;
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_via.valor ->> 'clave'
        ) || pg_catalog.decode(
                 pg_catalog.lpad(pg_catalog.to_hex(v_orden_anterior), 4, '0'),
                 'hex'
             )
          || pg_catalog.int4send(
                 pg_catalog.jsonb_array_length(
                     v_via.valor -> 'comprobaciones'
                 )
             );
        v_orden_comprobacion := 0;
        v_comprobaciones := ARRAY[]::text[];
        FOR v_comprobacion IN
            SELECT valor, ordinalidad
              FROM pg_catalog.jsonb_array_elements(
                       v_via.valor -> 'comprobaciones'
                   ) WITH ORDINALITY AS c(valor, ordinalidad)
        LOOP
            IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_comprobacion.valor,
                   ARRAY['clave', 'obligatoria', 'orden', 'procedencia']::text[]
               )
               OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                   v_comprobacion.valor -> 'procedencia',
                   ARRAY['clave', 'definicion_fuente_ref']::text[]
               )
               OR pg_catalog.jsonb_typeof(
                    v_comprobacion.valor -> 'clave'
                  ) IS DISTINCT FROM 'string'
               OR pg_catalog.jsonb_typeof(
                    v_comprobacion.valor #> '{procedencia,clave}'
                  ) IS DISTINCT FROM 'string'
               OR pg_catalog.jsonb_typeof(
                    v_comprobacion.valor #> '{procedencia,definicion_fuente_ref}'
                  ) IS DISTINCT FROM 'string'
               OR (v_comprobacion.valor ->> 'clave') !~
                  '^[a-z][a-z0-9._-]{1,79}$'
               OR (v_comprobacion.valor ->> 'clave') =
                  ANY(v_comprobaciones)
               OR pg_catalog.jsonb_typeof(
                      v_comprobacion.valor -> 'obligatoria'
                  ) <> 'boolean'
               OR NOT vec_contratacion_temporal
                  .numero_entero_json_canonico_v2(
                      v_comprobacion.valor -> 'orden', 1, 65535
                  )
               OR (v_comprobacion.valor ->> 'orden')::integer <=
                  v_orden_comprobacion
               OR (v_comprobacion.valor #>> '{procedencia,clave}') !~
                  '^[a-z][a-z0-9._-]{1,79}$'
               OR (v_comprobacion.valor
                       #>> '{procedencia,definicion_fuente_ref}') !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RETURN NULL;
            END IF;
            IF v_procedencias ? (v_comprobacion.valor ->> 'clave')
               AND v_procedencias -> (v_comprobacion.valor ->> 'clave')
                   IS DISTINCT FROM
                   v_comprobacion.valor -> 'procedencia' THEN
                RETURN NULL;
            END IF;
            v_procedencias := pg_catalog.jsonb_set(
                v_procedencias,
                ARRAY[v_comprobacion.valor ->> 'clave'],
                v_comprobacion.valor -> 'procedencia',
                true
            );
            v_comprobaciones := pg_catalog.array_append(
                v_comprobaciones, v_comprobacion.valor ->> 'clave'
            );
            v_orden_comprobacion :=
                (v_comprobacion.valor ->> 'orden')::integer;
            v_total := v_total + 1;
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, v_comprobacion.valor ->> 'clave'
            ) || pg_catalog.decode(
                     pg_catalog.lpad(
                         pg_catalog.to_hex(v_orden_comprobacion), 4, '0'
                     ),
                     'hex'
                 )
              || CASE WHEN (v_comprobacion.valor ->> 'obligatoria')::boolean
                      THEN E'\\x01'::bytea ELSE E'\\x00'::bytea END;
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material,
                v_comprobacion.valor #>> '{procedencia,clave}'
            );
            v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material,
                v_comprobacion.valor
                    #>> '{procedencia,definicion_fuente_ref}'
            );
        END LOOP;
        -- La presencia de cada lista forma parte del JSON; en material se
        -- codifica su cardinalidad, incluyendo cero si se omite.
        FOREACH v_tipo IN ARRAY ARRAY['documentos', 'datos']::text[] LOOP
            IF v_via.valor ? v_tipo THEN
                IF pg_catalog.jsonb_typeof(v_via.valor -> v_tipo) <> 'array'
                   OR pg_catalog.jsonb_array_length(
                          v_via.valor -> v_tipo
                      ) NOT BETWEEN 1 AND 32 THEN
                    RETURN NULL;
                END IF;
                v_material := v_material || pg_catalog.int4send(
                    pg_catalog.jsonb_array_length(v_via.valor -> v_tipo)
                );
                v_total_requisitos := v_total_requisitos +
                    pg_catalog.jsonb_array_length(v_via.valor -> v_tipo);
                v_orden_requisito := 0;
                v_claves_requisito := ARRAY[]::text[];
                FOR v_requisito IN
                    SELECT valor
                      FROM pg_catalog.jsonb_array_elements(
                               v_via.valor -> v_tipo
                           ) AS r(valor)
                LOOP
                    IF NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
                           v_requisito.valor, ARRAY['clave', 'clave_i18n', 'orden']::text[]
                       )
                       OR pg_catalog.jsonb_typeof(
                           v_requisito.valor -> 'clave'
                          ) IS DISTINCT FROM 'string'
                       OR pg_catalog.jsonb_typeof(
                           v_requisito.valor -> 'clave_i18n'
                          ) IS DISTINCT FROM 'string'
                       OR (v_requisito.valor ->> 'clave') !~
                          '^[a-z][a-z0-9._-]{1,79}$'
                       OR (v_requisito.valor ->> 'clave_i18n') !~
                          '^[a-z][a-z0-9._-]{1,79}$'
                       OR pg_catalog.strpos(
                          v_requisito.valor ->> 'clave_i18n', '.'
                          ) = 0
                       OR (v_requisito.valor ->> 'clave') =
                          ANY(v_claves_requisito)
                       OR NOT vec_contratacion_temporal
                          .numero_entero_json_canonico_v2(
                              v_requisito.valor -> 'orden', 1, 65535
                          )
                       OR (v_requisito.valor ->> 'orden')::integer <=
                          v_orden_requisito THEN
                        RETURN NULL;
                    END IF;
                    v_claves_requisito := pg_catalog.array_append(
                        v_claves_requisito, v_requisito.valor ->> 'clave'
                    );
                    v_orden_requisito :=
                        (v_requisito.valor ->> 'orden')::integer;
                    v_material :=
                        vec_contratacion_temporal.gobi_o404b_texto_canon(
                            v_material, v_requisito.valor ->> 'clave'
                        ) || pg_catalog.decode(
                            pg_catalog.lpad(
                                pg_catalog.to_hex(v_orden_requisito),
                                4, '0'
                            ), 'hex'
                        );
                    v_material :=
                        vec_contratacion_temporal.gobi_o404b_texto_canon(
                            v_material,
                            v_requisito.valor ->> 'clave_i18n'
                        );
                END LOOP;
            ELSE
                v_material := v_material || pg_catalog.int4send(0);
            END IF;
        END LOOP;
    END LOOP;
    IF v_total_requisitos NOT BETWEEN 1 AND 512
       OR v_total > 512
       OR pg_catalog.octet_length(v_material) > 1048576 THEN
        RETURN NULL;
    END IF;
    RETURN v_material;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.gobi_o404b_material_catalogo(
    p_publicacion jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF vec_contratacion_temporal.numero_entero_json_canonico_v2(
        p_publicacion #> '{canon,version_esquema}', 1, 1
    ) THEN
        RETURN vec_contratacion_temporal.gobi_o404b_material_catalogo_v1(
            p_publicacion
        );
    ELSIF vec_contratacion_temporal.numero_entero_json_canonico_v2(
        p_publicacion #> '{canon,version_esquema}', 2, 2
    ) THEN
        RETURN vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(
            p_publicacion
        );
    END IF;
    RETURN NULL;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.gobi_o404b_material_catalogo_v1(jsonb),
    vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;
COMMIT;
