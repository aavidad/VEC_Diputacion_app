-- Persistencia durable cerrada para fuentes autoritativas, propuestas,
-- atestaciones COSE, consumo unico, auditoria encadenada y outbox.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_llamamientos') IS NOT NULL OR
       to_regprocedure(
          'vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar llamamientos';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_llamamientos
    AUTHORIZATION vec_bolsa_llamamientos_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_llamamientos_propietario
    IN SCHEMA vec_bolsa_llamamientos REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_llamamientos_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.texto_opaco_valido(
    p_valor text, p_maximo integer
)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[[:space:][:cntrl:]]'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_bolsa_llamamientos.huella_valida(p_valor text)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_llamamientos.instante_utc_microsegundo_valido(
    p_valor text
)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE convertido timestamptz;
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    convertido := p_valor::timestamptz;
    RETURN to_char(convertido AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') = p_valor;
EXCEPTION WHEN OTHERS THEN RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_inmutable()
RETURNS trigger LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'historia inmutable';
END
$funcion$;

CREATE TABLE vec_bolsa_llamamientos.bolsa_autoritativa (
    bolsa_ref text NOT NULL,
    version bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    bolsa_canonica bytea NOT NULL,
    categoria_ref text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    estado text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (bolsa_ref, version, huella_bolsa_sha256),
    CHECK (version > 0),
    CHECK (estado IN ('vigente', 'suspendida', 'extinguida')),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(bolsa_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(categoria_ref, 512)),
    CHECK (categoria_ref ~ '^[A-Za-z0-9._:/#-]+$'),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_bolsa_sha256)),
    CHECK (octet_length(bolsa_canonica) BETWEEN 2 AND 8388608),
    CHECK (encode(sha256(bolsa_canonica), 'hex') = huella_bolsa_sha256),
    CHECK (vigente_hasta IS NULL OR vigente_desde < vigente_hasta)
);

CREATE TABLE vec_bolsa_llamamientos.necesidad_autoritativa (
    necesidad_ref text NOT NULL,
    version bigint NOT NULL,
    huella_necesidad_sha256 text NOT NULL,
    necesidad_canonica bytea NOT NULL,
    bolsa_ref text NOT NULL,
    version_bolsa bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    categoria_ref text NOT NULL,
    unidad_ref text NOT NULL,
    fin_previsto timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (necesidad_ref, version, huella_necesidad_sha256),
    FOREIGN KEY (bolsa_ref, version_bolsa, huella_bolsa_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_autoritativa(
            bolsa_ref, version, huella_bolsa_sha256
        ),
    CHECK (version > 0 AND version_bolsa > 0),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(necesidad_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(unidad_ref, 512)),
    CHECK (categoria_ref ~ '^[A-Za-z0-9._:/#-]+$'),
    CHECK (unidad_ref ~ '^[A-Za-z0-9._:/#-]+$'),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_necesidad_sha256)),
    CHECK (octet_length(necesidad_canonica) BETWEEN 2 AND 8388608),
    CHECK (encode(sha256(necesidad_canonica), 'hex') = huella_necesidad_sha256)
);

CREATE TABLE vec_bolsa_llamamientos.necesidad_actual (
    necesidad_ref text PRIMARY KEY,
    version bigint NOT NULL,
    huella_necesidad_sha256 text NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (necesidad_ref, version, huella_necesidad_sha256)
        REFERENCES vec_bolsa_llamamientos.necesidad_autoritativa(
            necesidad_ref, version, huella_necesidad_sha256
        ),
    CHECK (estado IN ('abierta', 'cerrada', 'anulada'))
);

CREATE TABLE vec_bolsa_llamamientos.politica_autoritativa (
    politica_ref text NOT NULL,
    version bigint NOT NULL,
    huella_politica_sha256 text NOT NULL,
    politica_canonica bytea NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    estado text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (politica_ref, version, huella_politica_sha256),
    CHECK (version > 0),
    CHECK (estado IN ('publicada', 'retirada')),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(politica_ref, 512)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_politica_sha256)),
    CHECK (octet_length(politica_canonica) BETWEEN 2 AND 8388608),
    CHECK (encode(sha256(politica_canonica), 'hex') = huella_politica_sha256),
    CHECK (vigente_hasta IS NULL OR vigente_desde < vigente_hasta)
);

CREATE TABLE vec_bolsa_llamamientos.instantanea_autoritativa (
    instantanea_ref text NOT NULL,
    version bigint NOT NULL,
    huella_instantanea_sha256 text NOT NULL,
    instantanea_canonica bytea NOT NULL,
    bolsa_ref text NOT NULL,
    version_bolsa bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    total_participaciones bigint NOT NULL,
    referida_en timestamptz(6) NOT NULL,
    generada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (instantanea_ref, version, huella_instantanea_sha256),
    FOREIGN KEY (bolsa_ref, version_bolsa, huella_bolsa_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_autoritativa(
            bolsa_ref, version, huella_bolsa_sha256
        ),
    CHECK (version > 0 AND total_participaciones > 0),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(instantanea_ref, 512)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_instantanea_sha256)),
    CHECK (octet_length(instantanea_canonica) BETWEEN 2 AND 33554432),
    CHECK (encode(sha256(instantanea_canonica), 'hex') = huella_instantanea_sha256),
    CHECK (referida_en <= generada_en)
);

CREATE TABLE vec_bolsa_llamamientos.evaluacion_autoritativa (
    instantanea_ref text NOT NULL,
    version_instantanea bigint NOT NULL,
    huella_instantanea_sha256 text NOT NULL,
    orden bigint NOT NULL,
    evaluacion_canonica jsonb NOT NULL,
    participacion_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    resultado text NOT NULL,
    entrada_evaluacion_ref text NOT NULL UNIQUE,
    huella_entrada_sha256 text NOT NULL,
    resultado_evaluacion_ref text NOT NULL UNIQUE,
    huella_resultado_sha256 text NOT NULL,
    evaluada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (instantanea_ref, version_instantanea, orden),
    FOREIGN KEY (
        instantanea_ref, version_instantanea, huella_instantanea_sha256
    ) REFERENCES vec_bolsa_llamamientos.instantanea_autoritativa(
        instantanea_ref, version, huella_instantanea_sha256
    ),
    CHECK (orden > 0 AND resultado IN ('elegible', 'no_elegible')),
    CHECK (jsonb_typeof(evaluacion_canonica) = 'object'),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(participacion_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(sujeto_ref, 512)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_entrada_sha256)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_resultado_sha256)),
    CHECK (entrada_evaluacion_ref <> resultado_evaluacion_ref)
);

CREATE TABLE vec_bolsa_llamamientos.atestacion_autorizacion_version (
    decision_ref text NOT NULL,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    evidencia_canonica bytea NOT NULL,
    huella_evidencia_sha256 text NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    clave_id text NOT NULL,
    revision_confianza text NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version, estado),
    CHECK (version > 0 AND estado IN ('activa', 'revocada')),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(decision_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(atestacion_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(clave_id, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(revision_confianza, 128)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_decision_sha256)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_evidencia_sha256)),
    CHECK (encode(sha256(evidencia_canonica), 'hex') = huella_evidencia_sha256),
    CHECK (octet_length(evidencia_canonica) BETWEEN 2 AND 2097152),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_sobre_sha256)),
    CHECK (encode(sha256(sobre_cose_sign1), 'hex') = huella_sobre_sha256),
    CHECK (octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384),
    CHECK (valida_desde < valida_hasta)
);

CREATE TABLE vec_bolsa_llamamientos.atestacion_autorizacion_actual (
    decision_ref text PRIMARY KEY,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref, atestacion_ref, version, estado)
        REFERENCES vec_bolsa_llamamientos.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version, estado
        ),
    CHECK (estado IN ('activa', 'revocada'))
);

CREATE TABLE vec_bolsa_llamamientos.propuesta (
    propuesta_ref text PRIMARY KEY,
    necesidad_ref text NOT NULL,
    version_necesidad bigint NOT NULL,
    huella_necesidad_sha256 text NOT NULL,
    instantanea_ref text NOT NULL,
    version_instantanea bigint NOT NULL,
    huella_instantanea_sha256 text NOT NULL,
    propuesta_canonica bytea NOT NULL,
    huella_propuesta_sha256 text NOT NULL,
    huella_documento_sha256 text NOT NULL,
    decision_ref text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    CONSTRAINT propuesta_necesidad_unica UNIQUE (
        necesidad_ref, version_necesidad, huella_necesidad_sha256
    ),
    CONSTRAINT propuesta_instantanea_unica UNIQUE (
        instantanea_ref, version_instantanea, huella_instantanea_sha256
    ),
    CONSTRAINT propuesta_decision_unica UNIQUE (decision_ref),
    FOREIGN KEY (necesidad_ref, version_necesidad, huella_necesidad_sha256)
        REFERENCES vec_bolsa_llamamientos.necesidad_autoritativa(
            necesidad_ref, version, huella_necesidad_sha256
        ),
    FOREIGN KEY (instantanea_ref, version_instantanea, huella_instantanea_sha256)
        REFERENCES vec_bolsa_llamamientos.instantanea_autoritativa(
            instantanea_ref, version, huella_instantanea_sha256
    ),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(propuesta_ref, 512)),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(decision_ref, 512)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_propuesta_sha256)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_documento_sha256)),
    CHECK (octet_length(propuesta_canonica) BETWEEN 2 AND 33554432),
    CHECK (encode(sha256(propuesta_canonica), 'hex') = huella_documento_sha256)
);

CREATE TABLE vec_bolsa_llamamientos.referencia_consumida (
    referencia text PRIMARY KEY,
    clase text NOT NULL,
    propuesta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.propuesta,
    consumida_en timestamptz(6) NOT NULL,
    CHECK (clase IN ('instantanea', 'entrada_evaluacion', 'resultado_evaluacion')),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(referencia, 512))
);

CREATE TABLE vec_bolsa_llamamientos.uso_decision (
    decision_ref text PRIMARY KEY,
    consumo_ref text NOT NULL UNIQUE,
    propuesta_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.propuesta,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    consumo_canonico bytea NOT NULL,
    huella_consumo_sha256 text NOT NULL UNIQUE,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref, atestacion_ref, atestacion_version)
        REFERENCES vec_bolsa_llamamientos.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version
        ),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(consumo_ref, 512)),
    CHECK (octet_length(consumo_canonico) BETWEEN 2 AND 2097152),
    CHECK (encode(sha256(consumo_canonico), 'hex') = huella_consumo_sha256)
);

CREATE TABLE vec_bolsa_llamamientos.auditoria (
    auditoria_ref text PRIMARY KEY,
    secuencia bigint NOT NULL UNIQUE,
    consumo_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.uso_decision(consumo_ref),
    registro_canonico bytea NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_auditoria_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (secuencia > 0),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(auditoria_ref, 512)),
    CHECK (vec_bolsa_llamamientos.huella_valida(huella_anterior_sha256)),
    CHECK (octet_length(registro_canonico) BETWEEN 2 AND 2097152),
    CHECK (encode(sha256(registro_canonico), 'hex') = huella_auditoria_sha256)
);

CREATE TABLE vec_bolsa_llamamientos.auditoria_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultima_huella_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CHECK (vec_bolsa_llamamientos.huella_valida(ultima_huella_sha256))
);
INSERT INTO vec_bolsa_llamamientos.auditoria_actual VALUES (
    true, 0, repeat('0', 64), statement_timestamp()
);

CREATE TABLE vec_bolsa_llamamientos.outbox (
    evento_ref text PRIMARY KEY,
    propuesta_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.propuesta,
    evento_canonico bytea NOT NULL,
    huella_evento_sha256 text NOT NULL UNIQUE,
    creada_en timestamptz(6) NOT NULL,
    entregada_en timestamptz(6),
    CHECK (vec_bolsa_llamamientos.texto_opaco_valido(evento_ref, 512)),
    CHECK (octet_length(evento_canonico) BETWEEN 2 AND 2097152),
    CHECK (entregada_en IS NULL OR entregada_en >= creada_en),
    CHECK (encode(sha256(evento_canonico), 'hex') = huella_evento_sha256)
);

DO $proteccion$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
        'politica_autoritativa', 'instantanea_autoritativa',
        'evaluacion_autoritativa', 'atestacion_autorizacion_version',
        'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
        'uso_decision', 'auditoria', 'auditoria_actual', 'outbox'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format(
            'CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.%I USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_llamamientos_propietario', 'vec_bolsa_llamamientos_propietario'
        );
        EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC', tabla);
    END LOOP;
END
$proteccion$;

DO $inmutabilidad$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'bolsa_autoritativa', 'necesidad_autoritativa',
        'politica_autoritativa', 'instantanea_autoritativa',
        'evaluacion_autoritativa', 'atestacion_autorizacion_version',
        'propuesta', 'referencia_consumida', 'uso_decision', 'auditoria'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER negar_mutacion BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_inmutable()',
            tabla
        );
    END LOOP;
END
$inmutabilidad$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
COMMIT;
