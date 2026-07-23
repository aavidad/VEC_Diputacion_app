BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000002_rotacion_hmac',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
        'vec_contratacion_temporal.preparar_alta_v1(jsonb)'
    ) IS NULL
       OR to_regclass(
           'vec_contratacion_temporal.identidad_reserva_alta'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'rotación HMAC rechazada: falta la migración 000001';
    END IF;
    IF to_regprocedure(
        'vec_contratacion_temporal.preparar_alta_v2(jsonb)'
    ) IS NOT NULL
       OR to_regclass(
           'vec_contratacion_temporal.alias_ambito_alta'
       ) IS NOT NULL
       OR to_regclass(
           'vec_contratacion_temporal.alias_huella_alta'
       ) IS NOT NULL
       OR to_regclass(
           'vec_contratacion_temporal.politica_generaciones_hmac_alta'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'rotación HMAC rechazada: ya existen objetos v2';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    DROP CONSTRAINT identidad_reserva_ambito_hmac_valido,
    DROP CONSTRAINT identidad_reserva_huella_peticion_valida;
ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    ADD CONSTRAINT identidad_reserva_ambito_hmac_valido CHECK (
        ambito_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND right(ambito_hmac, 64) <> repeat('0', 64)
    ),
    ADD CONSTRAINT identidad_reserva_huella_peticion_valida CHECK (
        huella_peticion_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND right(huella_peticion_hmac, 64) <> repeat('0', 64)
    );

CREATE TABLE vec_contratacion_temporal.politica_generaciones_hmac_alta (
    generacion integer PRIMARY KEY,
    posicion smallint NOT NULL UNIQUE,
    estado text NOT NULL,
    registrada_en timestamptz NOT NULL,
    CONSTRAINT politica_generacion_rango CHECK (
        generacion BETWEEN 1 AND 999999999
        AND posicion BETWEEN 0 AND 3
    ),
    CONSTRAINT politica_generacion_estado CHECK (
        (posicion = 0 AND estado = 'activa')
        OR (posicion > 0 AND estado = 'retenida')
    ),
    CONSTRAINT politica_generacion_instante CHECK (
        registrada_en = date_trunc('microseconds', registrada_en)
    )
);

INSERT INTO vec_contratacion_temporal.politica_generaciones_hmac_alta (
    generacion,
    posicion,
    estado,
    registrada_en
)
VALUES
    (2, 0, 'activa', date_trunc('microseconds', clock_timestamp())),
    (1, 1, 'retenida', date_trunc('microseconds', clock_timestamp()));

CREATE TABLE vec_contratacion_temporal.alias_ambito_alta (
    alias_hmac text PRIMARY KEY,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    registrada_en timestamptz NOT NULL,
    CONSTRAINT alias_ambito_raiz_fk
        FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.identidad_reserva_alta(ambito_hmac)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT alias_ambito_raiz_generacion_unica
        UNIQUE (ambito_raiz_hmac, generacion),
    CONSTRAINT alias_ambito_formato CHECK (
        alias_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND right(alias_hmac, 64) <> repeat('0', 64)
        AND substring(
            alias_hmac FROM '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
    ),
    CONSTRAINT alias_ambito_instante CHECK (
        registrada_en = date_trunc('microseconds', registrada_en)
    )
);

CREATE TABLE vec_contratacion_temporal.alias_huella_alta (
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    alias_hmac text NOT NULL,
    registrada_en timestamptz NOT NULL,
    PRIMARY KEY (ambito_raiz_hmac, generacion),
    CONSTRAINT alias_huella_raiz_fk
        FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.identidad_reserva_alta(ambito_hmac)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT alias_huella_raiz_alias_unico
        UNIQUE (ambito_raiz_hmac, alias_hmac),
    CONSTRAINT alias_huella_formato CHECK (
        alias_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
        )
        AND right(alias_hmac, 64) <> repeat('0', 64)
        AND substring(
            alias_hmac FROM '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
    ),
    CONSTRAINT alias_huella_instante CHECK (
        registrada_en = date_trunc('microseconds', registrada_en)
    )
);

INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
    alias_hmac,
    ambito_raiz_hmac,
    generacion,
    registrada_en
)
SELECT
    ambito_hmac,
    ambito_hmac,
    1,
    creada_en
FROM vec_contratacion_temporal.identidad_reserva_alta;

INSERT INTO vec_contratacion_temporal.alias_huella_alta (
    ambito_raiz_hmac,
    generacion,
    alias_hmac,
    registrada_en
)
SELECT
    ambito_hmac,
    1,
    huella_peticion_hmac,
    creada_en
FROM vec_contratacion_temporal.identidad_reserva_alta;

CREATE TRIGGER alias_ambito_alta_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.alias_ambito_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER alias_huella_alta_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.alias_huella_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.preparar_alta_v2(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text,
    ambito_hmac text,
    reserva_ref text,
    expediente_ref text,
    numero_visible text,
    recibo_ref text,
    huella_peticion_hmac text,
    organizacion_ref text,
    actor_ref text,
    perfil_ref text,
    estado text,
    version_expediente bigint,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_insertada boolean := false;
    v_filas bigint;
    v_identidad vec_contratacion_temporal.identidad_reserva_alta%ROWTYPE;
    v_version vec_contratacion_temporal.reserva_alta_version%ROWTYPE;
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_claves text[];
    v_claves_referencias text[];
    v_generaciones integer[];
    v_generaciones_politica integer[];
    v_ambitos text[];
    v_huellas text[];
    v_raices text[];
    v_raiz text;
    v_coincide_huella boolean;
    v_elementos_invalidos boolean;
    v_indice integer;
    v_statement_timeout_ms numeric;
    v_idle_timeout_ms numeric;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_ejecutor',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_propietario',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_migrador',
           'MEMBER'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'identidad de ejecución no autorizada';
    END IF;

    SELECT CASE
               WHEN unidad = 'ms'
                AND valor ~ '^[0-9]{1,18}$'
               THEN valor::numeric
               ELSE NULL
           END
      INTO v_statement_timeout_ms
      FROM (
          SELECT setting AS valor, unit AS unidad
          FROM pg_catalog.pg_settings
          WHERE name = 'statement_timeout'
      ) AS configuracion;
    SELECT CASE
               WHEN unidad = 'ms'
                AND valor ~ '^[0-9]{1,18}$'
               THEN valor::numeric
               ELSE NULL
           END
      INTO v_idle_timeout_ms
      FROM (
          SELECT setting AS valor, unit AS unidad
          FROM pg_catalog.pg_settings
          WHERE name = 'idle_in_transaction_session_timeout'
      ) AS configuracion;
    IF v_statement_timeout_ms IS NULL
       OR v_statement_timeout_ms <= 0
       OR v_statement_timeout_ms > 15000
       OR v_idle_timeout_ms IS NULL
       OR v_idle_timeout_ms <= 0
       OR v_idle_timeout_ms > 20000 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'límites ambientales de ejecución ausentes o inválidos';
    END IF;

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'operación de preparación inválida';
    END IF;
    SELECT array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM jsonb_object_keys(p_operacion) AS claves(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
        'actor_ref',
        'esquema',
        'organizacion_ref',
        'perfil_ref',
        'referencias_candidatas',
        'reserva_ref_candidata',
        'sellos_hmac'
    ]::text[] THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'campos de preparación inválidos';
    END IF;

    IF jsonb_typeof(p_operacion -> 'referencias_candidatas') <> 'object'
       OR jsonb_typeof(p_operacion -> 'sellos_hmac') <> 'object'
       OR jsonb_typeof(p_operacion #> '{sellos_hmac,activo}') <> 'object'
       OR jsonb_typeof(p_operacion #> '{sellos_hmac,retenidos}') <> 'array'
       OR jsonb_array_length(
           p_operacion #> '{sellos_hmac,retenidos}'
       ) > 3 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'estructura de preparación inválida';
    END IF;

    SELECT array_agg(clave ORDER BY clave)
      INTO v_claves_referencias
      FROM jsonb_object_keys(
          p_operacion -> 'referencias_candidatas'
      ) AS claves(clave);
    IF v_claves_referencias IS DISTINCT FROM ARRAY[
        'expediente_ref',
        'numero_visible',
        'recibo_ref'
    ]::text[] THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'campos de referencias inválidos';
    END IF;

    SELECT array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM jsonb_object_keys(p_operacion -> 'sellos_hmac') AS claves(clave);
    IF v_claves IS DISTINCT FROM ARRAY['activo', 'retenidos']::text[] THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'campos de sellos HMAC inválidos';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM (
            SELECT
                p_operacion #> '{sellos_hmac,activo}' AS elemento,
                0::bigint AS posicion
            UNION ALL
            SELECT valor, orden
            FROM jsonb_array_elements(
                p_operacion #> '{sellos_hmac,retenidos}'
            ) WITH ORDINALITY AS retenido(valor, orden)
        ) AS matriz
        WHERE jsonb_typeof(elemento) <> 'object'
           OR jsonb_typeof(elemento -> 'generacion') <> 'number'
           OR coalesce(elemento ->> 'generacion', '')
                !~ '^[1-9][0-9]{0,8}$'
           OR (
               SELECT array_agg(clave ORDER BY clave)
               FROM jsonb_object_keys(elemento) AS claves(clave)
           ) IS DISTINCT FROM ARRAY[
               'ambito_hmac',
               'generacion',
               'huella_peticion_hmac'
           ]::text[]
    ) INTO v_elementos_invalidos;
    IF v_elementos_invalidos THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'matriz de generaciones HMAC inválida';
    END IF;

    SELECT
        array_agg(
            (elemento ->> 'generacion')::integer ORDER BY posicion
        ),
        array_agg(elemento ->> 'ambito_hmac' ORDER BY posicion),
        array_agg(elemento ->> 'huella_peticion_hmac' ORDER BY posicion)
      INTO v_generaciones, v_ambitos, v_huellas
      FROM (
          SELECT
              p_operacion #> '{sellos_hmac,activo}' AS elemento,
              0::bigint AS posicion
          UNION ALL
          SELECT valor, orden
          FROM jsonb_array_elements(
              p_operacion #> '{sellos_hmac,retenidos}'
          ) WITH ORDINALITY AS retenido(valor, orden)
      ) AS matriz;

    SELECT array_agg(generacion ORDER BY posicion)
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica
       OR cardinality(v_generaciones) NOT BETWEEN 1 AND 4
       OR cardinality(v_ambitos) <> cardinality(
           ARRAY(SELECT DISTINCT unnest(v_ambitos))
       )
       OR cardinality(v_huellas) <> cardinality(
           ARRAY(SELECT DISTINCT unnest(v_huellas))
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'generaciones HMAC ausentes, desordenadas o repetidas';
    END IF;

    FOR v_indice IN 1..cardinality(v_generaciones) LOOP
        IF coalesce(v_ambitos[v_indice], '') !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           )
           OR coalesce(v_huellas[v_indice], '') !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           )
           OR right(v_ambitos[v_indice], 64) = repeat('0', 64)
           OR right(v_huellas[v_indice], 64) = repeat('0', 64)
           OR substring(
               v_ambitos[v_indice] FROM '/v([1-9][0-9]{0,8}):'
           )::integer <> v_generaciones[v_indice]
           OR substring(
               v_huellas[v_indice] FROM '/v([1-9][0-9]{0,8}):'
           )::integer <> v_generaciones[v_indice] THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'sellos HMAC inválidos o mal ligados';
        END IF;
    END LOOP;

    IF p_operacion ->> 'esquema'
           <> 'vec.contratacion-temporal.preparar-alta.v2'
       OR coalesce(p_operacion ->> 'reserva_ref_candidata', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,expediente_ref}',
           ''
       ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,numero_visible}',
           ''
       ) !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,recibo_ref}',
           ''
       ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'organizacion_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'actor_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'perfil_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'contenido de preparación inválido';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_ct:ambito:' || alias_hmac,
            0
        )
    )
    FROM unnest(v_ambitos) AS alias(alias_hmac)
    ORDER BY alias_hmac;

    SELECT array_agg(DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_ambitos);
    IF cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'convergencia de ámbitos HMAC rechazada';
    END IF;

    IF cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac,
            reserva_ref,
            expediente_ref,
            numero_visible,
            recibo_ref,
            huella_peticion_hmac,
            organizacion_ref,
            actor_ref,
            perfil_ref,
            creada_en
        )
        VALUES (
            v_ambitos[1],
            p_operacion ->> 'reserva_ref_candidata',
            p_operacion #>> '{referencias_candidatas,expediente_ref}',
            p_operacion #>> '{referencias_candidatas,numero_visible}',
            p_operacion #>> '{referencias_candidatas,recibo_ref}',
            v_huellas[1],
            p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'actor_ref',
            p_operacion ->> 'perfil_ref',
            v_ahora
        )
        ON CONFLICT ON CONSTRAINT identidad_reserva_alta_pkey DO NOTHING;
        GET DIAGNOSTICS v_filas = ROW_COUNT;
        v_insertada := v_filas = 1;
        v_raiz := v_ambitos[1];
    END IF;

    SELECT *
      INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta AS identidad
     WHERE identidad.ambito_hmac = v_raiz
     FOR UPDATE;

    IF NOT v_insertada THEN
        SELECT EXISTS (
            SELECT 1
            FROM vec_contratacion_temporal.alias_huella_alta
            WHERE ambito_raiz_hmac = v_raiz
              AND alias_hmac = ANY (v_huellas)
        ) INTO v_coincide_huella;
        IF NOT v_coincide_huella
           OR v_identidad.organizacion_ref
                <> p_operacion ->> 'organizacion_ref'
           OR v_identidad.actor_ref <> p_operacion ->> 'actor_ref'
           OR v_identidad.perfil_ref <> p_operacion ->> 'perfil_ref' THEN
            RETURN QUERY
            SELECT
                'idempotencia_reutilizada'::text,
                v_ambitos[1],
                p_operacion ->> 'reserva_ref_candidata',
                p_operacion #>> '{referencias_candidatas,expediente_ref}',
                p_operacion #>> '{referencias_candidatas,numero_visible}',
                p_operacion #>> '{referencias_candidatas,recibo_ref}',
                v_huellas[1],
                p_operacion ->> 'organizacion_ref',
                p_operacion ->> 'actor_ref',
                p_operacion ->> 'perfil_ref',
                'reservada'::text,
                NULL::bigint,
                NULL::text,
                NULL::text,
                NULL::timestamptz;
            RETURN;
        END IF;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM unnest(v_generaciones, v_ambitos)
            AS recibido(generacion, alias_hmac)
        JOIN vec_contratacion_temporal.alias_ambito_alta AS guardado
          ON guardado.ambito_raiz_hmac = v_raiz
         AND guardado.generacion = recibido.generacion
        WHERE guardado.alias_hmac <> recibido.alias_hmac
    ) OR EXISTS (
        SELECT 1
        FROM unnest(v_generaciones, v_huellas)
            AS recibido(generacion, alias_hmac)
        JOIN vec_contratacion_temporal.alias_huella_alta AS guardado
          ON guardado.ambito_raiz_hmac = v_raiz
         AND guardado.generacion = recibido.generacion
        WHERE guardado.alias_hmac <> recibido.alias_hmac
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'colisión de generación HMAC rechazada';
    END IF;

    INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
        alias_hmac,
        ambito_raiz_hmac,
        generacion,
        registrada_en
    )
    SELECT alias_hmac, v_raiz, generacion, v_ahora
    FROM unnest(v_generaciones, v_ambitos)
        AS recibido(generacion, alias_hmac)
    ON CONFLICT (alias_hmac) DO NOTHING;

    IF EXISTS (
        SELECT 1
        FROM vec_contratacion_temporal.alias_ambito_alta
        WHERE alias_hmac = ANY (v_ambitos)
          AND ambito_raiz_hmac <> v_raiz
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'alias de ámbito HMAC ya asignado';
    END IF;

    INSERT INTO vec_contratacion_temporal.alias_huella_alta (
        ambito_raiz_hmac,
        generacion,
        alias_hmac,
        registrada_en
    )
    SELECT v_raiz, generacion, alias_hmac, v_ahora
    FROM unnest(v_generaciones, v_huellas)
        AS recibido(generacion, alias_hmac)
    ON CONFLICT (ambito_raiz_hmac, generacion) DO NOTHING;

    IF v_insertada THEN
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac,
            revision,
            estado,
            registrada_en
        )
        VALUES (v_raiz, 1, 'reservada', v_ahora);
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual (
            ambito_hmac,
            revision
        )
        VALUES (v_raiz, 1);
    END IF;

    SELECT version.*
      INTO STRICT v_version
      FROM vec_contratacion_temporal.reserva_alta_actual AS actual
      JOIN vec_contratacion_temporal.reserva_alta_version AS version
        ON version.ambito_hmac = actual.ambito_hmac
       AND version.revision = actual.revision
     WHERE actual.ambito_hmac = v_raiz;

    RETURN QUERY
    SELECT
        CASE
            WHEN v_version.estado = 'confirmada' THEN 'confirmada'
            WHEN v_insertada THEN 'reservada'
            ELSE 'reutilizada'
        END,
        v_identidad.ambito_hmac,
        v_identidad.reserva_ref,
        v_identidad.expediente_ref,
        v_identidad.numero_visible,
        v_identidad.recibo_ref,
        v_identidad.huella_peticion_hmac,
        v_identidad.organizacion_ref,
        v_identidad.actor_ref,
        v_identidad.perfil_ref,
        v_version.estado,
        v_version.version_expediente,
        v_version.auditoria_ref,
        v_version.evento_ref,
        v_version.confirmada_en;
END
$funcion$;

ALTER TABLE vec_contratacion_temporal.politica_generaciones_hmac_alta
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.alias_ambito_alta
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.alias_huella_alta
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.preparar_alta_v2(jsonb)
    OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_contratacion_temporal FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contratacion_temporal FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v1(jsonb)
    FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v2(jsonb)
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
