-- O4-04B: proyección durable, append-only y de mínimo privilegio del gobierno
-- de cobertura. Esta migración no compone todavía la confirmación O4-04E.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000017_gobierno_cobertura_o4_04b', 0
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
           'vec_contratacion_temporal.expediente_version_integral'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.numero_entero_json_canonico_v2(jsonb,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.texto_instante_utc_go_v2(text)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.gobi_o404b_catalogo'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar gobierno O4-04B';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_claves_exactas(
    p_valor jsonb,
    p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_valor) = 'object'
       AND (
           SELECT pg_catalog.array_agg(clave ORDER BY clave)
             FROM pg_catalog.jsonb_object_keys(p_valor) AS k(clave)
       ) = p_claves;
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_texto_canon(
    p_preimagen bytea,
    p_valor text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT CASE
        WHEN pg_catalog.octet_length(p_valor) <= 1048576
        THEN p_preimagen
             || pg_catalog.int4send(pg_catalog.octet_length(p_valor))
             || pg_catalog.convert_to(p_valor, 'UTF8')
    END;
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_microsegundos(
    p_instante timestamptz
)
RETURNS bigint
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT (
        extract(epoch FROM p_instante)::numeric * 1000000
    )::bigint;
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_entorno_valido(
    p_solo_lectura boolean
)
RETURNS boolean
LANGUAGE sql
STABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.current_setting('transaction_isolation') =
               'serializable'
       AND pg_catalog.current_setting('transaction_read_only') =
               CASE WHEN p_solo_lectura THEN 'on' ELSE 'off' END
       AND (
           SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                       THEN setting::numeric END
             FROM pg_catalog.pg_settings
            WHERE name = 'statement_timeout'
       ) BETWEEN 1 AND 15000
       AND (
           SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                       THEN setting::numeric END
             FROM pg_catalog.pg_settings
            WHERE name = 'idle_in_transaction_session_timeout'
       ) BETWEEN 1 AND 20000;
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
    p_valor text,
    p_permite_cero boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF p_valor = '0001-01-01T00:00:00Z' THEN
        RETURN p_permite_cero;
    END IF;
    IF p_valor !~ (
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}T'
           || '([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]'
           || '([.][0-9]{0,5}[1-9])?Z$'
       ) THEN
        RETURN false;
    END IF;
    RETURN vec_contratacion_temporal.texto_instante_utc_go_v2(
               p_valor
           ) = p_valor;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_material_catalogo(
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

CREATE TABLE vec_contratacion_temporal.gobi_o404b_evento (
    secuencia bigint PRIMARY KEY,
    evento_ref text NOT NULL UNIQUE,
    tipo text NOT NULL,
    huella_evento_sha256 text NOT NULL,
    contenido_evento jsonb NOT NULL,
    registrado_en timestamptz(6) NOT NULL,
    UNIQUE (secuencia, evento_ref),
    CHECK (secuencia BETWEEN 1 AND 9007199254740991),
    CHECK (evento_ref ~ '^evento_gobi_o404b_[a-f0-9]{32}$'),
    CHECK (tipo IN ('publicacion', 'retirada')),
    CHECK (huella_evento_sha256 ~ '^[a-f0-9]{64}$'
           AND huella_evento_sha256 <> pg_catalog.repeat('0', 64)),
    CHECK (pg_catalog.jsonb_typeof(contenido_evento) = 'object'),
    CHECK (pg_catalog.octet_length(contenido_evento::text) <= 3145728),
    CHECK (registrado_en = pg_catalog.date_trunc(
        'microseconds', registrado_en
    ))
);

CREATE TABLE vec_contratacion_temporal.gobi_o404b_catalogo (
    referencia text NOT NULL,
    version numeric(20, 0) NOT NULL,
    huella_sha256 text NOT NULL,
    publicacion_json jsonb NOT NULL,
    publicado_en timestamptz(6) NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    evento_ref text NOT NULL,
    secuencia bigint NOT NULL,
    PRIMARY KEY (referencia, version),
    UNIQUE (referencia, version, huella_sha256),
    FOREIGN KEY (secuencia, evento_ref)
        REFERENCES vec_contratacion_temporal.gobi_o404b_evento
            (secuencia, evento_ref),
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (huella_sha256 ~ '^[a-f0-9]{64}$'
           AND huella_sha256 <> pg_catalog.repeat('0', 64)),
    CHECK (pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.gobi_o404b_material_catalogo(
            publicacion_json
        )
    ), 'hex') = huella_sha256),
    CHECK (publicacion_json ->> 'referencia' = referencia),
    CHECK ((publicacion_json ->> 'version')::numeric = version),
    CHECK (publicacion_json ->> 'huella_sha256' = huella_sha256),
    CHECK (publicado_en = (publicacion_json ->> 'publicado_en')::timestamptz),
    CHECK (vigente_desde =
           (publicacion_json #>> '{vigencia,desde}')::timestamptz),
    CHECK (vigente_hasta IS NOT DISTINCT FROM CASE
        WHEN publicacion_json #>> '{vigencia,hasta}' =
             '0001-01-01T00:00:00Z' THEN NULL
        ELSE (publicacion_json #>> '{vigencia,hasta}')::timestamptz END)
);

CREATE TABLE vec_contratacion_temporal.gobi_o404b_checkpoint (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    ultima_secuencia bigint NOT NULL,
    ultimo_evento_ref text,
    ultima_huella_evento_sha256 text,
    actualizado_en timestamptz(6) NOT NULL,
    CHECK (ultima_secuencia BETWEEN 0 AND 9007199254740991),
    CHECK (
        (ultima_secuencia = 0 AND ultimo_evento_ref IS NULL
         AND ultima_huella_evento_sha256 IS NULL)
        OR
        (ultima_secuencia > 0
         AND ultimo_evento_ref ~ '^evento_gobi_o404b_[a-f0-9]{32}$'
         AND ultima_huella_evento_sha256 ~ '^[a-f0-9]{64}$'
         AND ultima_huella_evento_sha256 <>
             pg_catalog.repeat('0', 64))
    )
);

INSERT INTO vec_contratacion_temporal.gobi_o404b_checkpoint (
    control, ultima_secuencia, actualizado_en
) VALUES (
    true, 0, pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    )
);

CREATE FUNCTION vec_contratacion_temporal.gobi_o404b_bloquear_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia de gobierno O4-04B inmutable';
END
$funcion$;

CREATE TRIGGER bloquear_mutacion
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.gobi_o404b_evento
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER bloquear_truncado
BEFORE TRUNCATE ON vec_contratacion_temporal.gobi_o404b_evento
FOR EACH STATEMENT EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER bloquear_mutacion
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.gobi_o404b_catalogo
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();
CREATE TRIGGER bloquear_truncado
BEFORE TRUNCATE ON vec_contratacion_temporal.gobi_o404b_catalogo
FOR EACH STATEMENT EXECUTE FUNCTION
vec_contratacion_temporal.gobi_o404b_bloquear_inmutable();

ALTER TABLE vec_contratacion_temporal.gobi_o404b_evento
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_evento
    FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_catalogo
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_catalogo
    FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_checkpoint
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_checkpoint
    FORCE ROW LEVEL SECURITY;

CREATE POLICY propietario_total ON
    vec_contratacion_temporal.gobi_o404b_evento
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
CREATE POLICY propietario_total ON
    vec_contratacion_temporal.gobi_o404b_catalogo
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
CREATE POLICY propietario_total ON
    vec_contratacion_temporal.gobi_o404b_checkpoint
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);

REVOKE ALL ON
    vec_contratacion_temporal.gobi_o404b_evento,
    vec_contratacion_temporal.gobi_o404b_catalogo,
    vec_contratacion_temporal.gobi_o404b_checkpoint
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contratacion_temporal
FROM PUBLIC;

COMMIT;
