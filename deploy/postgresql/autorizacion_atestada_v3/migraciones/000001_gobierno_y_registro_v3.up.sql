BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_autorizacion_atestada_v3.clave_capacidad_version'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migración VEC-AD-3 ya aplicada';
    END IF;
END
$prevalidacion$;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v3_propietario
    IN SCHEMA vec_autorizacion_atestada_v3
    REVOKE ALL ON TABLES FROM PUBLIC,
        vec_autorizacion_atestada_v3_emisor,
        vec_autorizacion_atestada_v3_consumidor;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v3_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC,
        vec_autorizacion_atestada_v3_emisor,
        vec_autorizacion_atestada_v3_consumidor;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v3_propietario
    REVOKE ALL ON TYPES FROM PUBLIC,
        vec_autorizacion_atestada_v3_emisor,
        vec_autorizacion_atestada_v3_consumidor;

CREATE FUNCTION vec_autorizacion_atestada_v3.texto_tecnico_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_maximo BETWEEN 1 AND 4096
       AND pg_catalog.octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor COLLATE "C" ~ '^[!-~]+$'
       AND pg_catalog.strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.huella_sha256_valida(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor COLLATE "C" ~ '^[0-9a-f]{64}$'
       AND p_valor <> pg_catalog.repeat('0', 64)
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'historia VEC-AD-3 inmutable';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'TRUNCATE VEC-AD-3 rechazado';
END
$funcion$;

CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    revision_gobierno numeric(20, 0) NOT NULL,
    huella_gobierno_sha256 text NOT NULL,
    secreto_hmac bytea NOT NULL,
    huella_secreto_sha256 text NOT NULL,
    emisor_id text NOT NULL,
    audiencia_consumo text NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    UNIQUE (revision_gobierno),
    UNIQUE (huella_secreto_sha256),
    CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    CHECK (revision_gobierno BETWEEN 1 AND 18446744073709551615::numeric),
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        clave_id, 512
    )),
    CHECK (vec_autorizacion_atestada_v3.huella_sha256_valida(
        huella_gobierno_sha256
    )),
    CHECK (pg_catalog.octet_length(secreto_hmac) BETWEEN 32 AND 4096),
    CHECK (pg_catalog.encode(
        pg_catalog.sha256(secreto_hmac), 'hex'
    ) = huella_secreto_sha256),
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        emisor_id, 512
    )),
    CHECK (audiencia_consumo =
        'vec_contratacion_temporal.confirmar_alta_atestada.v1'),
    CHECK (valida_hasta > valida_desde),
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        acto_ref, 512
    ))
);

CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision (
    orden numeric(20, 0) PRIMARY KEY,
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL UNIQUE,
    establecida_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version,
    CHECK (orden BETWEEN 1 AND 18446744073709551615::numeric)
);

CREATE TABLE vec_autorizacion_atestada_v3.revocacion_clave_capacidad (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version,
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        motivo_catalogado_ref, 512
    ))
);

CREATE TABLE vec_autorizacion_atestada_v3.configuracion_confianza_version (
    revision text NOT NULL,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    huella_configuracion_sha256 text NOT NULL UNIQUE,
    publicada_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (revision),
    CHECK (secuencia BETWEEN 1 AND 18446744073709551615::numeric),
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        revision, 512
    )),
    CHECK (vec_autorizacion_atestada_v3.huella_sha256_valida(
        huella_configuracion_sha256
    )),
    CHECK (expira_en > publicada_en)
);

CREATE TABLE vec_autorizacion_atestada_v3.raiz_confianza_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    clave_publica_spki bytea NOT NULL,
    huella_spki_sha256 text NOT NULL UNIQUE,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    suite text NOT NULL,
    audiencia_despliegue text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    CHECK (pg_catalog.octet_length(clave_publica_spki) = 44),
    CHECK (pg_catalog.encode(
        pg_catalog.sha256(clave_publica_spki), 'hex'
    ) = huella_spki_sha256),
    CHECK (valida_hasta > valida_desde),
    CHECK (suite = 'VEC-AD-3-COSE-EDDSA-1'),
    CHECK (vec_autorizacion_atestada_v3.texto_tecnico_valido(
        audiencia_despliegue, 512
    ))
);

CREATE TABLE vec_autorizacion_atestada_v3.configuracion_raiz (
    configuracion_revision text NOT NULL,
    raiz_clave_id text NOT NULL,
    raiz_version numeric(20, 0) NOT NULL,
    PRIMARY KEY (
        configuracion_revision, raiz_clave_id, raiz_version
    ),
    FOREIGN KEY (configuracion_revision)
        REFERENCES vec_autorizacion_atestada_v3.configuracion_confianza_version,
    FOREIGN KEY (raiz_clave_id, raiz_version)
        REFERENCES vec_autorizacion_atestada_v3.raiz_confianza_version
);

CREATE TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual (
    orden numeric(20, 0) PRIMARY KEY,
    configuracion_revision text NOT NULL UNIQUE,
    establecida_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (configuracion_revision)
        REFERENCES vec_autorizacion_atestada_v3.configuracion_confianza_version,
    CHECK (orden BETWEEN 1 AND 18446744073709551615::numeric)
);

CREATE TABLE vec_autorizacion_atestada_v3.revocacion_configuracion (
    configuracion_revision text PRIMARY KEY,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (configuracion_revision)
        REFERENCES vec_autorizacion_atestada_v3.configuracion_confianza_version
);

CREATE TABLE vec_autorizacion_atestada_v3.revocacion_raiz (
    raiz_clave_id text NOT NULL,
    raiz_version numeric(20, 0) NOT NULL,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (raiz_clave_id, raiz_version),
    FOREIGN KEY (raiz_clave_id, raiz_version)
        REFERENCES vec_autorizacion_atestada_v3.raiz_confianza_version
);

CREATE TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    revision numeric(20, 0) NOT NULL,
    configuracion_secuencia_minima numeric(20, 0) NOT NULL,
    raiz_version_minima numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CHECK (revision BETWEEN 0 AND 18446744073709551615::numeric),
    CHECK (configuracion_secuencia_minima BETWEEN
        0 AND 18446744073709551615::numeric),
    CHECK (raiz_version_minima BETWEEN
        0 AND 18446744073709551615::numeric)
);
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno VALUES (
    true, 0, 0, 0, clock_timestamp()
);

CREATE TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL UNIQUE,
    decision_canonica bytea NOT NULL,
    motivo_canonico bytea NOT NULL,
    contexto_actor_canonico bytea NOT NULL,
    payload_vec_ad_3 bytea NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    evidencia_verificacion bytea NOT NULL,
    raiz_publica_spki bytea NOT NULL,
    capacidad_canonica bytea NOT NULL,
    huella_capacidad_sha256 text NOT NULL UNIQUE,
    efecto_ref text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (decision_ref, efecto_ref, huella_efecto_sha256),
    FOREIGN KEY (decision_ref)
        REFERENCES vec_autorizacion.decision_concedida_contexto_actor_v3(
            decision_ref
        )
);

CREATE TABLE vec_autorizacion_atestada_v3.consumo_decision_v3 (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL UNIQUE,
    nonce text NOT NULL UNIQUE,
    efecto_ref text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    consumida_en timestamptz(6) NOT NULL,
    UNIQUE (decision_ref, efecto_ref, huella_efecto_sha256),
    FOREIGN KEY (decision_ref, efecto_ref, huella_efecto_sha256)
        REFERENCES vec_autorizacion_atestada_v3.atestacion_decision_v3(
            decision_ref, efecto_ref, huella_efecto_sha256
        )
);

CREATE TABLE vec_autorizacion_atestada_v3.control_cadena_auditoria (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    secuencia numeric(20, 0) NOT NULL,
    cabeza_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL
);
INSERT INTO vec_autorizacion_atestada_v3.control_cadena_auditoria VALUES (
    true, 0, pg_catalog.repeat('0', 64), clock_timestamp()
);

CREATE TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref, efecto_ref, huella_efecto_sha256)
        REFERENCES vec_autorizacion_atestada_v3.consumo_decision_v3(
            decision_ref, efecto_ref, huella_efecto_sha256
        )
);

DO $proteccion$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'clave_capacidad_version', 'puntero_clave_emision',
        'revocacion_clave_capacidad', 'configuracion_confianza_version',
        'raiz_confianza_version', 'configuracion_raiz',
        'puntero_configuracion_actual', 'revocacion_configuracion',
        'revocacion_raiz', 'atestacion_decision_v3',
        'consumo_decision_v3', 'auditoria_consumo_v3'
    ] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I FOR ALL TO vec_autorizacion_atestada_v3_propietario USING (current_user = ''vec_autorizacion_atestada_v3_propietario'') WITH CHECK (current_user = ''vec_autorizacion_atestada_v3_propietario'')',
            v_tabla
        );
    END LOOP;
END
$proteccion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_autorizacion_atestada_v3
    FROM PUBLIC, vec_autorizacion_atestada_v3_emisor,
         vec_autorizacion_atestada_v3_consumidor;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_autorizacion_atestada_v3
    FROM PUBLIC, vec_autorizacion_atestada_v3_emisor,
         vec_autorizacion_atestada_v3_consumidor;

COMMIT;
