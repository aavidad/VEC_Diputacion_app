BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000008_efectos_integrales', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_version_integral'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.reserva_operacion_analisis'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.actuacion_expediente_integral'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para efectos integrales';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.actuacion_expediente_integral (
    expediente_ref text NOT NULL,
    secuencia numeric(20, 0) NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    operacion_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    actuacion_json jsonb NOT NULL,
    actuacion_json_huella_sha256 text NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, secuencia),
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (
        secuencia = version_expediente
        AND secuencia BETWEEN 2 AND 9007199254740991::numeric
    ),
    CHECK (pg_catalog.jsonb_typeof(actuacion_json) = 'object'),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(actuacion_json::text, 'UTF8')
        ), 'hex') = actuacion_json_huella_sha256
    ),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            prueba_canonica
        ), 'hex') = prueba_huella_sha256
    ),
    CHECK (pg_catalog.octet_length(prueba_canonica)
        BETWEEN 128 AND 4096),
    CHECK (registrada_en = pg_catalog.date_trunc(
        'microseconds', registrada_en
    ))
);

CREATE TABLE vec_contratacion_temporal.consumo_fuentes_analisis (
    consumo_ref text PRIMARY KEY,
    conjunto_huella_sha256 text NOT NULL UNIQUE,
    ambito_raiz_hmac text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    artefacto_ref text NOT NULL,
    artefacto_huella_sha256 text NOT NULL,
    fuentes_json jsonb NOT NULL,
    fuentes_json_huella_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (conjunto_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (artefacto_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (pg_catalog.jsonb_typeof(fuentes_json) = 'object'),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(fuentes_json::text, 'UTF8')
        ), 'hex') = fuentes_json_huella_sha256
    ),
    CHECK (consumida_en = pg_catalog.date_trunc(
        'microseconds', consumida_en
    ))
);

CREATE TABLE vec_contratacion_temporal.consumo_decision_analisis (
    decision_ref text PRIMARY KEY,
    decision_huella_sha256 text NOT NULL UNIQUE,
    ambito_raiz_hmac text NOT NULL UNIQUE,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_raiz_hmac)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis,
    CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (consumida_en = pg_catalog.date_trunc(
        'microseconds', consumida_en
    ))
);

CREATE TABLE
vec_contratacion_temporal.control_cadenas_expediente_integral (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    secuencia_auditoria numeric(20, 0) NOT NULL,
    cabeza_auditoria_sha256 text NOT NULL,
    secuencia_outbox numeric(20, 0) NOT NULL,
    cabeza_outbox_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CHECK (secuencia_auditoria
        BETWEEN 0 AND 9007199254740991::numeric),
    CHECK (secuencia_outbox
        BETWEEN 0 AND 9007199254740991::numeric),
    CHECK (cabeza_auditoria_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (cabeza_outbox_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (actualizada_en = pg_catalog.date_trunc(
        'microseconds', actualizada_en
    ))
);
INSERT INTO
vec_contratacion_temporal.control_cadenas_expediente_integral
VALUES (
    true, 0, pg_catalog.repeat('0', 64),
    0, pg_catalog.repeat('0', 64),
    pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
);

CREATE TABLE vec_contratacion_temporal.auditoria_expediente_integral (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    operacion_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    consumo_fuentes_ref text NOT NULL UNIQUE,
    prueba_canonica bytea NOT NULL,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    FOREIGN KEY (decision_ref)
        REFERENCES vec_contratacion_temporal.consumo_decision_analisis,
    FOREIGN KEY (consumo_fuentes_ref)
        REFERENCES vec_contratacion_temporal.consumo_fuentes_analisis,
    CHECK (secuencia BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (pg_catalog.octet_length(prueba_canonica)
        BETWEEN 128 AND 16384),
    CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            anterior_sha256::bytea || prueba_canonica
        ), 'hex') = huella_sha256
    ),
    CHECK (registrada_en = pg_catalog.date_trunc(
        'microseconds', registrada_en
    ))
);

CREATE TABLE vec_contratacion_temporal.outbox_expediente_integral (
    evento_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    operacion_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    tipo_evento text NOT NULL,
    payload_canonico bytea NOT NULL,
    payload_huella_sha256 text NOT NULL UNIQUE,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (secuencia BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (tipo_evento ~
        '^[a-z][a-z0-9_.-]{2,159}$'),
    CHECK (pg_catalog.octet_length(payload_canonico)
        BETWEEN 64 AND 32768),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            payload_canonico
        ), 'hex') = payload_huella_sha256
    ),
    CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (
        pg_catalog.encode(pg_catalog.sha256(
            anterior_sha256::bytea || payload_canonico
        ), 'hex') = huella_sha256
    ),
    CHECK (registrada_en = pg_catalog.date_trunc(
        'microseconds', registrada_en
    ))
);

DO $seguridad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'actuacion_expediente_integral',
        'consumo_fuentes_analisis',
        'consumo_decision_analisis',
        'control_cadenas_expediente_integral',
        'auditoria_expediente_integral',
        'outbox_expediente_integral'
    ] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY %I ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',
            v_tabla || '_propietario', v_tabla
        );
        EXECUTE pg_catalog.format(
            'REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor',
            v_tabla
        );
    END LOOP;
END
$seguridad$;

CREATE TRIGGER actuacion_expediente_integral_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.actuacion_expediente_integral
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER consumo_fuentes_analisis_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.consumo_fuentes_analisis
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER consumo_decision_analisis_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.consumo_decision_analisis
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER auditoria_expediente_integral_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.auditoria_expediente_integral
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER outbox_expediente_integral_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.outbox_expediente_integral
FOR EACH ROW EXECUTE FUNCTION
vec_contratacion_temporal.rechazar_mutacion_historia_v1();

COMMIT;
