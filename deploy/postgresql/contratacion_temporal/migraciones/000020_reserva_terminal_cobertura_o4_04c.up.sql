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

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.analisis_rrhh_valido_v3(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_integral_actual'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.reserva_operacion_decision_cobertura'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404c_referencia_derivada_v1(text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para la reserva O4-04C';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404c_referencia_derivada_v1(
    p_prefijo text,
    p_raiz text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT p_prefijo || pg_catalog.substr(
        pg_catalog.encode(
            pg_catalog.sha256(
                pg_catalog.convert_to(p_prefijo || ':' || p_raiz, 'UTF8')
            ),
            'hex'
        ),
        1,
        32
    )
     WHERE p_prefijo ~ '^[a-z][a-z0-9:_-]{2,63}$'
       AND p_raiz ~ (
           '^hmac-sha256:vec[.]contratacion-temporal[.]'
           || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
           || '[a-f0-9]{64}$'
       )
$funcion$;

-- Esta fila es la barrera compartida de O4-04. Las funciones de operación la
-- bloquean FOR SHARE y las migraciones posteriores FOR UPDATE.
CREATE TABLE vec_contratacion_temporal.control_migracion_cobertura_o4 (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    version_esquema integer NOT NULL CHECK (version_esquema BETWEEN 1 AND 999),
    actualizada_en timestamptz(6) NOT NULL CHECK (
        actualizada_en = pg_catalog.date_trunc(
            'microseconds', actualizada_en
        )
    )
);

INSERT INTO vec_contratacion_temporal.control_migracion_cobertura_o4 (
    control,
    version_esquema,
    actualizada_en
) VALUES (
    true,
    1,
    pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
);

-- Identidad inmutable de la intención. No conserva actor, perfil, motivo,
-- clave idempotente ni token en claro; esa semántica queda ligada por HMAC.
CREATE TABLE
vec_contratacion_temporal.reserva_operacion_decision_cobertura (
    ambito_raiz_hmac text PRIMARY KEY,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    actuacion_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    correlacion_vec_ref text NOT NULL UNIQUE,
    decision_vec_ref text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    analisis_ref text NOT NULL,
    analisis_huella_sha256 text NOT NULL,
    huella_semantica_raiz_hmac text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        ambito_raiz_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
            || '[a-f0-9]{64}$'
        )
        AND pg_catalog.right(ambito_raiz_hmac, 64) <>
            pg_catalog.repeat('0', 64)
    ),
    CHECK (
        huella_semantica_raiz_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'cobertura-decision[.]semantica/v[1-9][0-9]{0,8}:'
            || '[a-f0-9]{64}$'
        )
        AND pg_catalog.right(huella_semantica_raiz_hmac, 64) <>
            pg_catalog.repeat('0', 64)
    ),
    CHECK (
        substring(
            ambito_raiz_hmac FROM '/v([1-9][0-9]{0,8}):'
        ) = substring(
            huella_semantica_raiz_hmac FROM '/v([1-9][0-9]{0,8}):'
        )
    ),
    CHECK (
        reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actuacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND correlacion_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND decision_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND analisis_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    ),
    CHECK (
        version_expediente BETWEEN 2 AND 9007199254740990::numeric
        AND analisis_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND analisis_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND creada_en = pg_catalog.date_trunc('microseconds', creada_en)
    )
);

CREATE TABLE
vec_contratacion_temporal.alias_operacion_decision_cobertura (
    alias_ambito_hmac text PRIMARY KEY,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    alias_huella_semantica_hmac text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (ambito_raiz_hmac, generacion),
    UNIQUE (ambito_raiz_hmac, alias_huella_semantica_hmac),
    UNIQUE (
        ambito_raiz_hmac,
        alias_ambito_hmac,
        alias_huella_semantica_hmac
    ),
    CHECK (generacion BETWEEN 1 AND 999999999),
    CHECK (
        alias_ambito_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'cobertura-decision[.]ambito/v[1-9][0-9]{0,8}:'
            || '[a-f0-9]{64}$'
        )
        AND substring(
            alias_ambito_hmac FROM '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
        AND alias_huella_semantica_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'cobertura-decision[.]semantica/v[1-9][0-9]{0,8}:'
            || '[a-f0-9]{64}$'
        )
        AND substring(
            alias_huella_semantica_hmac FROM '/v([1-9][0-9]{0,8}):'
        )::integer = generacion
    ),
    CHECK (
        pg_catalog.right(alias_ambito_hmac, 64) <>
            pg_catalog.repeat('0', 64)
        AND pg_catalog.right(alias_huella_semantica_hmac, 64) <>
            pg_catalog.repeat('0', 64)
        AND registrada_en =
            pg_catalog.date_trunc('microseconds', registrada_en)
    )
);

-- secuencia es la versión append-only del estado. revision_cercado cambia
-- únicamente al adquirir o reapropiar; el terminal conserva el fence dueño.
CREATE TABLE
vec_contratacion_temporal.reserva_operacion_decision_cobertura_version (
    ambito_raiz_hmac text NOT NULL,
    secuencia numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    revision_cercado numeric(20, 0) NOT NULL,
    token_propietario_sha256 text NOT NULL,
    observada_en timestamptz(6) NOT NULL,
    propiedad_hasta timestamptz(6) NOT NULL,
    huella_orden_sha256 text,
    confirmada_en timestamptz(6),
    PRIMARY KEY (ambito_raiz_hmac, secuencia),
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        secuencia BETWEEN 1 AND 9007199254740991::numeric
        AND revision_cercado BETWEEN 1 AND 9007199254740991::numeric
        AND token_propietario_sha256 ~ '^[a-f0-9]{64}$'
        AND token_propietario_sha256 <> pg_catalog.repeat('0', 64)
        AND observada_en =
            pg_catalog.date_trunc('microseconds', observada_en)
        AND propiedad_hasta =
            pg_catalog.date_trunc('microseconds', propiedad_hasta)
        AND propiedad_hasta > observada_en
        AND propiedad_hasta <= observada_en + interval '5 seconds'
    ),
    CHECK (
        (
            estado = 'reservada'
            AND huella_orden_sha256 IS NULL
            AND confirmada_en IS NULL
        )
        OR (
            estado IN ('aplicada', 'denegada_vec')
            AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
            AND huella_orden_sha256 <> pg_catalog.repeat('0', 64)
            AND confirmada_en IS NOT NULL
            AND confirmada_en =
                pg_catalog.date_trunc('microseconds', confirmada_en)
            AND confirmada_en >= observada_en
        )
    )
);

CREATE TABLE
vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual (
    ambito_raiz_hmac text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac, secuencia)
        REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version
        ON UPDATE RESTRICT ON DELETE RESTRICT
        DEFERRABLE INITIALLY IMMEDIATE
);

-- Recibo y marker se separan para evitar ciclos. O4-04E será el único corte
-- que podrá insertarlos, después de crear toda la prueba funcional.
CREATE TABLE
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura (
    ambito_raiz_hmac text PRIMARY KEY,
    recibo_ref text NOT NULL UNIQUE,
    reserva_ref text NOT NULL UNIQUE,
    huella_orden_sha256 text NOT NULL UNIQUE,
    rama text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    correlacion_vec_ref text NOT NULL UNIQUE,
    decision_vec_ref text NOT NULL UNIQUE,
    decision_vec_huella_sha256 text NOT NULL,
    codigo_probatorio_vec text NOT NULL,
    revision_cercado numeric(20, 0) NOT NULL,
    ambito_idempotencia_hmac text NOT NULL,
    huella_semantica_hmac text NOT NULL,
    decision_cobertura_ref text,
    decision_cobertura_huella_sha256 text,
    version_resultante numeric(20, 0),
    evento_ref text,
    actuacion_ref text,
    confirmada_en timestamptz(6) NOT NULL,
    recibo_huella_sha256 text NOT NULL UNIQUE,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        ambito_raiz_hmac,
        ambito_idempotencia_hmac,
        huella_semantica_hmac
    ) REFERENCES
        vec_contratacion_temporal.alias_operacion_decision_cobertura (
            ambito_raiz_hmac,
            alias_ambito_hmac,
            alias_huella_semantica_hmac
        )
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND correlacion_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND decision_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND recibo_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND revision_cercado BETWEEN 1 AND 9007199254740991::numeric
        AND confirmada_en =
            pg_catalog.date_trunc('microseconds', confirmada_en)
    ),
    CHECK (
        pg_catalog.right(huella_orden_sha256, 64) <>
            pg_catalog.repeat('0', 64)
        AND pg_catalog.right(decision_vec_huella_sha256, 64) <>
            pg_catalog.repeat('0', 64)
        AND pg_catalog.right(recibo_huella_sha256, 64) <>
            pg_catalog.repeat('0', 64)
    ),
    CHECK (
        (
            rama = 'concedida'
            AND codigo_probatorio_vec = 'concedida'
            AND decision_cobertura_ref ~
                '^decision-cobertura:sha256:[a-f0-9]{64}$'
            AND decision_cobertura_huella_sha256 ~ '^[a-f0-9]{64}$'
            AND decision_cobertura_ref =
                'decision-cobertura:sha256:'
                || decision_cobertura_huella_sha256
            AND version_resultante BETWEEN
                3 AND 9007199254740991::numeric
            AND evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND actuacion_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        )
        OR (
            rama = 'denegada'
            AND codigo_probatorio_vec IN (
                'perfil_no_vigente',
                'ambito_no_autorizado',
                'rol_no_publicado',
                'rol_retirado',
                'accion_no_concedida',
                'finalidad_no_autorizada',
                'denegada_por_politica',
                'restriccion_abac_incumplida',
                'garantia_insuficiente'
            )
            AND decision_cobertura_ref IS NULL
            AND decision_cobertura_huella_sha256 IS NULL
            AND version_resultante IS NULL
            AND evento_ref IS NULL
            AND actuacion_ref IS NULL
        )
    )
);

CREATE TABLE
vec_contratacion_temporal.terminal_operacion_decision_cobertura (
    ambito_raiz_hmac text PRIMARY KEY,
    secuencia_terminal numeric(20, 0) NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    huella_orden_sha256 text NOT NULL UNIQUE,
    rama text NOT NULL,
    decision_vec_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    outbox_ref text NOT NULL UNIQUE,
    gobierno_ref text,
    gobierno_huella_sha256 text,
    consumo_c1_lote_ref text,
    consumo_c1_lote_huella_sha256 text,
    decision_cobertura_ref text,
    actuacion_ref text,
    version_resultante numeric(20, 0),
    marcada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (ambito_raiz_hmac, secuencia_terminal)
        REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura_version
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        secuencia_terminal BETWEEN 2 AND 9007199254740991::numeric
        AND recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND outbox_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND marcada_en = pg_catalog.date_trunc('microseconds', marcada_en)
    ),
    CHECK (
        (
            rama = 'concedida'
            AND gobierno_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND gobierno_huella_sha256 ~ '^[a-f0-9]{64}$'
            AND consumo_c1_lote_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND consumo_c1_lote_huella_sha256 ~ '^[a-f0-9]{64}$'
            AND decision_cobertura_ref ~
                '^decision-cobertura:sha256:[a-f0-9]{64}$'
            AND actuacion_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND version_resultante BETWEEN
                3 AND 9007199254740991::numeric
        )
        OR (
            rama = 'denegada'
            AND gobierno_ref IS NULL
            AND gobierno_huella_sha256 IS NULL
            AND consumo_c1_lote_ref IS NULL
            AND consumo_c1_lote_huella_sha256 IS NULL
            AND decision_cobertura_ref IS NULL
            AND actuacion_ref IS NULL
            AND version_resultante IS NULL
        )
    )
);

DO $inmutabilidad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'reserva_operacion_decision_cobertura',
        'alias_operacion_decision_cobertura',
        'reserva_operacion_decision_cobertura_version',
        'confirmacion_operacion_decision_cobertura',
        'terminal_operacion_decision_cobertura'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_mutacion '
            || 'BEFORE UPDATE OR DELETE '
            || 'ON vec_contratacion_temporal.%I '
            || 'FOR EACH ROW EXECUTE FUNCTION '
            || 'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_truncado '
            || 'BEFORE TRUNCATE ON vec_contratacion_temporal.%I '
            || 'FOR EACH STATEMENT EXECUTE FUNCTION '
            || 'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
    END LOOP;
END
$inmutabilidad$;

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_migracion_cobertura_o4',
        'reserva_operacion_decision_cobertura',
        'alias_operacion_decision_cobertura',
        'reserva_operacion_decision_cobertura_version',
        'reserva_operacion_decision_cobertura_actual',
        'confirmacion_operacion_decision_cobertura',
        'terminal_operacion_decision_cobertura'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total ON '
            || 'vec_contratacion_temporal.%I '
            || 'TO vec_contratacion_temporal_propietario '
            || 'USING (true) WITH CHECK (true)',
            v_tabla
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON
    vec_contratacion_temporal.control_migracion_cobertura_o4,
    vec_contratacion_temporal.reserva_operacion_decision_cobertura,
    vec_contratacion_temporal.alias_operacion_decision_cobertura,
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_version,
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual,
    vec_contratacion_temporal.confirmacion_operacion_decision_cobertura,
    vec_contratacion_temporal.terminal_operacion_decision_cobertura
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.o404c_referencia_derivada_v1(text, text)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador;

COMMIT;
