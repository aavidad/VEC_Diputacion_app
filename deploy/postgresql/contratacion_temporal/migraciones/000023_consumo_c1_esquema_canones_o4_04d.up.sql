-- O4-04D/1: almacenamiento y cánones del consumo C1 durable interno.
-- La barrera permanece en v3 hasta que 000024 publique las operaciones.
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
 WHERE control AND version_esquema = 3
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 3
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_texto_canon(bytea,text)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_microsegundos(timestamp with time zone)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.gobi_o404b_instante_texto_valido(text,boolean)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_lote'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para consumo C1 O4-04D';
    END IF;
END
$prevalidacion$;

-- Dependencia catalogada que impide bajar 000022 mientras exista este corte.
-- No se publica ni se invoca como capacidad funcional de O4-04D.
CREATE FUNCTION
vec_contratacion_temporal.o404d_dependencia_lector_o404c_v1(
    p_coordenadas jsonb
)
RETURNS TABLE (carga_json text)
LANGUAGE sql
STABLE
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT *
      FROM vec_contratacion_temporal
           .leer_terminal_primario_decision_cobertura_o404c_v1(
               p_coordenadas
           );
END;

CREATE TABLE vec_contratacion_temporal.consumo_cobertura_lote (
    lote_ref text PRIMARY KEY,
    lote_huella_sha256 text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    reserva_ref text NOT NULL,
    preparacion_c1_ref text NOT NULL UNIQUE,
    preparacion_c1_huella_sha256 text NOT NULL UNIQUE,
    huella_orden_sha256 text NOT NULL UNIQUE,
    huella_ordenes_c1_sha256 text NOT NULL,
    catalogo_ref text NOT NULL,
    catalogo_version numeric(20, 0) NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    decision_vec_ref text NOT NULL UNIQUE,
    correlacion_vec_ref text NOT NULL UNIQUE,
    contexto_recurso_huella_sha256 text NOT NULL,
    decision_vec_huella_sha256 text NOT NULL UNIQUE,
    prueba_vinculo_vec_sha256 text NOT NULL UNIQUE,
    codigo_probatorio_vec text NOT NULL,
    registrada_vec_en timestamptz(6) NOT NULL,
    revalidada_vec_en timestamptz(6) NOT NULL,
    numero_evidencias integer NOT NULL,
    efecto_en timestamptz(6) NOT NULL,
    persistido_en timestamptz(6) NOT NULL,
    lote_canon bytea NOT NULL,
    CHECK (
        lote_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND preparacion_c1_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND decision_vec_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND correlacion_vec_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND catalogo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    ),
    CHECK (
        version_expediente BETWEEN 2 AND 9007199254740990::numeric
        AND catalogo_version BETWEEN 1 AND 9007199254740991::numeric
        AND numero_evidencias BETWEEN 1 AND 512
        AND codigo_probatorio_vec = 'concedida'
    ),
    CHECK (
        lote_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND preparacion_c1_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_ordenes_c1_sha256 ~ '^[a-f0-9]{64}$'
        AND catalogo_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND contexto_recurso_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND prueba_vinculo_vec_sha256 ~ '^[a-f0-9]{64}$'
        AND lote_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(lote_canon), 'hex'
        )
    ),
    CHECK (
        registrada_vec_en =
            pg_catalog.date_trunc('microseconds', registrada_vec_en)
        AND revalidada_vec_en =
            pg_catalog.date_trunc('microseconds', revalidada_vec_en)
        AND efecto_en = pg_catalog.date_trunc('microseconds', efecto_en)
        AND persistido_en =
            pg_catalog.date_trunc('microseconds', persistido_en)
        AND revalidada_vec_en >= registrada_vec_en
        AND persistido_en >= revalidada_vec_en
    )
);

CREATE TABLE vec_contratacion_temporal.consumo_cobertura_evidencia (
    lote_ref text NOT NULL,
    posicion integer NOT NULL,
    total integer NOT NULL,
    evidencia_huella_sha256 text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    peticion_ref text NOT NULL,
    huella_peticion_sha256 text NOT NULL,
    huella_resultado_sha256 text NOT NULL,
    autoridad_ref text NOT NULL,
    generacion numeric(10, 0) NOT NULL,
    recibo_respuesta_ref text NOT NULL,
    huella_respuesta_sha256 text NOT NULL,
    catalogo_ref text NOT NULL,
    catalogo_version numeric(20, 0) NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    via_clave text NOT NULL,
    comprobacion_clave text NOT NULL,
    comprobacion_resultado text NOT NULL,
    orden_comprobacion integer NOT NULL,
    comprobacion_obligatoria boolean NOT NULL,
    procedencia_clave text NOT NULL,
    definicion_fuente_ref text NOT NULL,
    categoria_ref text NOT NULL,
    periodo_inicio timestamptz(6) NOT NULL,
    periodo_fin timestamptz(6) NOT NULL,
    solicitada_en timestamptz(6) NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    verificador_ref text NOT NULL,
    publicador_catalogo_ref text NOT NULL,
    peticion_canon bytea NOT NULL,
    resultado_canon bytea NOT NULL,
    atestacion_canon bytea NOT NULL,
    confirmacion_tcb_canon bytea NOT NULL,
    catalogo_canon bytea NOT NULL,
    verificador_canon bytea NOT NULL,
    resumen_canon bytea NOT NULL,
    evidencia_canon bytea NOT NULL,
    PRIMARY KEY (lote_ref, posicion),
    FOREIGN KEY (lote_ref)
        REFERENCES vec_contratacion_temporal.consumo_cobertura_lote
        ON UPDATE RESTRICT
        ON DELETE RESTRICT,
    UNIQUE (organizacion_ref, peticion_ref),
    UNIQUE (autoridad_ref, generacion, recibo_respuesta_ref),
    CHECK (
        posicion BETWEEN 1 AND 512
        AND total BETWEEN 1 AND 512
    ),
    CHECK (
        version_expediente BETWEEN 2 AND 9007199254740990::numeric
        AND generacion BETWEEN 1 AND 4294967295::numeric
        AND catalogo_version BETWEEN 1 AND 9007199254740991::numeric
        AND orden_comprobacion BETWEEN 1 AND 65535
    ),
    CHECK (
        evidencia_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_peticion_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_resultado_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_respuesta_sha256 ~ '^[a-f0-9]{64}$'
        AND catalogo_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND evidencia_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(evidencia_canon), 'hex'
        )
    ),
    CHECK (
        periodo_inicio < periodo_fin
        AND solicitada_en <= emitida_en
        AND emitida_en < valida_hasta
    )
);

CREATE FUNCTION vec_contratacion_temporal.o404d_prueba_canon_v1(
    p_evidencia jsonb,
    p_clave text
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_hex text;
BEGIN
    v_hex := p_evidencia ->> p_clave;
    IF pg_catalog.jsonb_typeof(p_evidencia -> p_clave) <> 'string'
       OR pg_catalog.length(v_hex) NOT BETWEEN 2 AND 131072
       OR pg_catalog.length(v_hex) % 2 <> 0
       OR v_hex !~ '^[a-f0-9]+$' THEN
        RETURN NULL;
    END IF;
    RETURN pg_catalog.decode(v_hex, 'hex');
EXCEPTION
    WHEN data_exception OR invalid_parameter_value THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404d_material_evidencia_v1(
    p_evidencia jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_claves text[];
    v_material bytea := ''::bytea;
    v_prueba bytea;
    v_clave text;
    v_inicio timestamptz;
    v_fin timestamptz;
    v_solicitada timestamptz;
    v_emitida timestamptz;
    v_valida timestamptz;
BEGIN
    IF pg_catalog.jsonb_typeof(p_evidencia) <> 'object'
       OR pg_catalog.pg_column_size(p_evidencia) > 1048576
       OR pg_catalog.octet_length(p_evidencia::text) > 1048576 THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_evidencia) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
           'atestacion_canon_hex', 'autoridad_ref',
           'catalogo_canon_hex', 'catalogo_huella_sha256',
           'catalogo_ref', 'catalogo_version', 'categoria_ref',
           'comprobacion_clave', 'comprobacion_obligatoria',
           'comprobacion_resultado', 'confirmacion_tcb_canon_hex',
           'definicion_fuente_ref', 'emitida_en',
           'evidencia_huella_sha256', 'expediente_ref', 'generacion',
           'huella_peticion_sha256', 'huella_respuesta_sha256',
           'huella_resultado_sha256', 'orden_comprobacion',
           'organizacion_ref', 'periodo_fin', 'periodo_inicio',
           'peticion_canon_hex', 'peticion_ref', 'posicion',
           'procedencia_clave', 'publicador_catalogo_ref',
           'recibo_respuesta_ref', 'resultado_canon_hex',
           'resumen_canon_hex', 'solicitada_en', 'total',
           'valida_hasta', 'verificador_canon_hex', 'verificador_ref',
           'version_expediente', 'via_clave'
       ]::text[] THEN
        RETURN NULL;
    END IF;

    IF NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'posicion', 1, 512
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'total', 1, 512
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'version_expediente',
           2,
           9007199254740990::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'generacion', 1, 4294967295::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'catalogo_version',
           1,
           9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_evidencia -> 'orden_comprobacion', 1, 65535
       )
       OR pg_catalog.jsonb_typeof(
              p_evidencia -> 'comprobacion_obligatoria'
          ) <> 'boolean' THEN
        RETURN NULL;
    END IF;

    FOREACH v_clave IN ARRAY ARRAY[
        'organizacion_ref', 'expediente_ref', 'peticion_ref',
        'autoridad_ref', 'recibo_respuesta_ref', 'catalogo_ref',
        'definicion_fuente_ref', 'categoria_ref', 'verificador_ref',
        'publicador_catalogo_ref'
    ]::text[]
    LOOP
        IF pg_catalog.jsonb_typeof(p_evidencia -> v_clave) <> 'string'
           OR (p_evidencia ->> v_clave) !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RETURN NULL;
        END IF;
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'via_clave', 'comprobacion_clave', 'comprobacion_resultado',
        'procedencia_clave'
    ]::text[]
    LOOP
        IF pg_catalog.jsonb_typeof(p_evidencia -> v_clave) <> 'string'
           OR (p_evidencia ->> v_clave) !~
              '^[a-z][a-z0-9._-]{1,79}$' THEN
            RETURN NULL;
        END IF;
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'evidencia_huella_sha256', 'huella_peticion_sha256',
        'huella_resultado_sha256', 'huella_respuesta_sha256',
        'catalogo_huella_sha256'
    ]::text[]
    LOOP
        IF pg_catalog.jsonb_typeof(p_evidencia -> v_clave) <> 'string'
           OR (p_evidencia ->> v_clave) !~ '^[a-f0-9]{64}$'
           OR (p_evidencia ->> v_clave) =
              pg_catalog.repeat('0', 64) THEN
            RETURN NULL;
        END IF;
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'periodo_inicio', 'periodo_fin', 'solicitada_en', 'emitida_en',
        'valida_hasta'
    ]::text[]
    LOOP
        IF NOT vec_contratacion_temporal
               .gobi_o404b_instante_texto_valido(
                   p_evidencia ->> v_clave, false
               ) THEN
            RETURN NULL;
        END IF;
    END LOOP;

    v_inicio := (p_evidencia ->> 'periodo_inicio')::timestamptz;
    v_fin := (p_evidencia ->> 'periodo_fin')::timestamptz;
    v_solicitada := (p_evidencia ->> 'solicitada_en')::timestamptz;
    v_emitida := (p_evidencia ->> 'emitida_en')::timestamptz;
    v_valida := (p_evidencia ->> 'valida_hasta')::timestamptz;
    IF v_inicio >= v_fin
       OR v_fin > v_inicio + interval '100 years'
       OR v_solicitada > v_emitida
       OR v_emitida >= v_valida
       OR v_valida > v_emitida + interval '5 seconds' THEN
        RETURN NULL;
    END IF;

    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'VEC-CT-CONSUMO-C1-EVIDENCIA-O4-04D-V1'
    );
    v_material := v_material
        || pg_catalog.int8send((p_evidencia ->> 'posicion')::bigint)
        || pg_catalog.int8send((p_evidencia ->> 'total')::bigint);
    FOREACH v_clave IN ARRAY ARRAY[
        'organizacion_ref', 'expediente_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_evidencia ->> 'version_expediente')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        'peticion_ref', 'huella_peticion_sha256',
        'huella_resultado_sha256', 'autoridad_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_evidencia ->> 'generacion')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        'recibo_respuesta_ref', 'huella_respuesta_sha256',
        'catalogo_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_evidencia ->> 'catalogo_version')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        'catalogo_huella_sha256', 'via_clave', 'comprobacion_clave',
        'comprobacion_resultado'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_evidencia ->> 'orden_comprobacion')::bigint
    );
    v_material := v_material || CASE
        WHEN (p_evidencia ->> 'comprobacion_obligatoria')::boolean
        THEN E'\\x01'::bytea
        ELSE E'\\x00'::bytea
    END;
    FOREACH v_clave IN ARRAY ARRAY[
        'procedencia_clave', 'definicion_fuente_ref', 'categoria_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
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
            vec_contratacion_temporal.gobi_o404b_microsegundos(
                v_solicitada
            )
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_emitida)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_valida)
        );
    FOREACH v_clave IN ARRAY ARRAY[
        'verificador_ref', 'publicador_catalogo_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_evidencia ->> v_clave
            );
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'peticion_canon_hex', 'resultado_canon_hex',
        'atestacion_canon_hex', 'confirmacion_tcb_canon_hex',
        'catalogo_canon_hex', 'verificador_canon_hex',
        'resumen_canon_hex'
    ]::text[]
    LOOP
        v_prueba := vec_contratacion_temporal.o404d_prueba_canon_v1(
            p_evidencia, v_clave
        );
        IF v_prueba IS NULL THEN
            RETURN NULL;
        END IF;
        v_material := v_material || pg_catalog.sha256(v_prueba);
    END LOOP;
    RETURN v_material;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404d_material_lote_v1(
    p_lote jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_claves text[];
    v_clave text;
    v_material bytea := ''::bytea;
    v_evidencia record;
    v_material_evidencia bytea;
    v_total integer;
    v_peticiones text[] := ARRAY[]::text[];
    v_respuestas text[] := ARRAY[]::text[];
    v_efecto timestamptz;
BEGIN
    IF pg_catalog.jsonb_typeof(p_lote) <> 'object'
       OR pg_catalog.pg_column_size(p_lote) > 67108864
       OR pg_catalog.octet_length(p_lote::text) > 67108864 THEN
        RETURN NULL;
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_lote) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
           'catalogo_huella_sha256', 'catalogo_ref',
           'catalogo_version', 'correlacion_vec_ref', 'decision_vec_ref',
           'efecto_en', 'esquema', 'evidencias', 'expediente_ref',
           'huella_orden_sha256', 'huella_ordenes_c1_sha256',
           'lote_huella_sha256', 'lote_ref', 'organizacion_ref',
           'preparacion_c1_huella_sha256', 'preparacion_c1_ref',
           'reserva_ref', 'version_expediente'
       ]::text[]
       OR p_lote ->> 'esquema' <>
          'vec.contratacion-temporal.consumo-c1.o4-04d.v1'
       OR pg_catalog.jsonb_typeof(p_lote -> 'evidencias') <> 'array'
       OR pg_catalog.jsonb_array_length(
              p_lote -> 'evidencias'
          ) NOT BETWEEN 1 AND 512 THEN
        RETURN NULL;
    END IF;
    v_total := pg_catalog.jsonb_array_length(p_lote -> 'evidencias');
    IF NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_lote -> 'version_expediente',
           2,
           9007199254740990::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_lote -> 'catalogo_version',
           1,
           9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal
              .gobi_o404b_instante_texto_valido(
                  p_lote ->> 'efecto_en', false
              ) THEN
        RETURN NULL;
    END IF;
    FOREACH v_clave IN ARRAY ARRAY[
        'lote_ref', 'organizacion_ref', 'expediente_ref',
        'reserva_ref', 'preparacion_c1_ref', 'decision_vec_ref',
        'correlacion_vec_ref', 'catalogo_ref'
    ]::text[]
    LOOP
        IF (p_lote ->> v_clave) !~
           '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RETURN NULL;
        END IF;
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'lote_huella_sha256', 'preparacion_c1_huella_sha256',
        'huella_orden_sha256', 'huella_ordenes_c1_sha256',
        'catalogo_huella_sha256'
    ]::text[]
    LOOP
        IF (p_lote ->> v_clave) !~ '^[a-f0-9]{64}$'
           OR (p_lote ->> v_clave) = pg_catalog.repeat('0', 64) THEN
            RETURN NULL;
        END IF;
    END LOOP;

    v_efecto := (p_lote ->> 'efecto_en')::timestamptz;
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'VEC-CT-CONSUMO-C1-LOTE-O4-04D-V1'
    );
    FOREACH v_clave IN ARRAY ARRAY[
        'lote_ref', 'organizacion_ref', 'expediente_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_lote ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_lote ->> 'version_expediente')::bigint
    );
    FOREACH v_clave IN ARRAY ARRAY[
        'reserva_ref', 'preparacion_c1_ref',
        'preparacion_c1_huella_sha256',
        'huella_orden_sha256', 'huella_ordenes_c1_sha256',
        'decision_vec_ref', 'correlacion_vec_ref', 'catalogo_ref'
    ]::text[]
    LOOP
        v_material :=
            vec_contratacion_temporal.gobi_o404b_texto_canon(
                v_material, p_lote ->> v_clave
            );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_lote ->> 'catalogo_version')::bigint
    );
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_lote ->> 'catalogo_huella_sha256'
    );
    v_material := v_material || pg_catalog.int8send(
        vec_contratacion_temporal.gobi_o404b_microsegundos(v_efecto)
    );
    v_material := v_material || pg_catalog.int4send(v_total);

    FOR v_evidencia IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(
                   p_lote -> 'evidencias'
               ) WITH ORDINALITY AS e(valor, ordinalidad)
         ORDER BY ordinalidad
    LOOP
        v_material_evidencia :=
            vec_contratacion_temporal.o404d_material_evidencia_v1(
                v_evidencia.valor
            );
        IF v_material_evidencia IS NULL
           OR (v_evidencia.valor ->> 'posicion')::integer <>
              v_evidencia.ordinalidad
           OR (v_evidencia.valor ->> 'total')::integer <> v_total
           OR v_evidencia.valor ->> 'organizacion_ref'
              IS DISTINCT FROM
              p_lote ->> 'organizacion_ref'
           OR v_evidencia.valor ->> 'expediente_ref'
              IS DISTINCT FROM
              p_lote ->> 'expediente_ref'
           OR v_evidencia.valor ->> 'version_expediente'
              IS DISTINCT FROM
              p_lote ->> 'version_expediente'
           OR v_evidencia.valor ->> 'catalogo_ref'
              IS DISTINCT FROM
              p_lote ->> 'catalogo_ref'
           OR v_evidencia.valor ->> 'catalogo_version'
              IS DISTINCT FROM
              p_lote ->> 'catalogo_version'
           OR v_evidencia.valor ->> 'catalogo_huella_sha256'
              IS DISTINCT FROM
              p_lote ->> 'catalogo_huella_sha256'
           OR pg_catalog.encode(
                  pg_catalog.sha256(v_material_evidencia), 'hex'
              ) <> v_evidencia.valor ->> 'evidencia_huella_sha256'
           OR v_efecto <
              (v_evidencia.valor ->> 'emitida_en')::timestamptz
           OR v_efecto >=
              (v_evidencia.valor ->> 'valida_hasta')::timestamptz
           OR v_evidencia.valor ->> 'peticion_ref' =
              ANY(v_peticiones)
           OR (
                  (v_evidencia.valor ->> 'autoridad_ref') || E'\\000'
                  || (v_evidencia.valor ->> 'generacion') || E'\\000'
                  || (v_evidencia.valor ->> 'recibo_respuesta_ref')
              ) = ANY(v_respuestas) THEN
            RETURN NULL;
        END IF;
        v_peticiones := pg_catalog.array_append(
            v_peticiones, v_evidencia.valor ->> 'peticion_ref'
        );
        v_respuestas := pg_catalog.array_append(
            v_respuestas,
            (v_evidencia.valor ->> 'autoridad_ref') || E'\\000'
            || (v_evidencia.valor ->> 'generacion') || E'\\000'
            || (v_evidencia.valor ->> 'recibo_respuesta_ref')
        );
        v_material :=
            v_material || pg_catalog.sha256(v_material_evidencia);
    END LOOP;
    RETURN v_material;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

REVOKE ALL ON
    vec_contratacion_temporal.consumo_cobertura_lote,
    vec_contratacion_temporal.consumo_cobertura_evidencia
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404d_dependencia_lector_o404c_v1(jsonb),
    vec_contratacion_temporal.o404d_prueba_canon_v1(jsonb, text),
    vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb),
    vec_contratacion_temporal.o404d_material_lote_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

COMMIT;
