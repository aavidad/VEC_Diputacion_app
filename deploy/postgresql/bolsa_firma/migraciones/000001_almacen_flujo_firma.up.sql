-- Almacén cerrado de la saga de firma de revisiones de baremación.
BEGIN;
SET LOCAL ROLE vec_bolsa_firma_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_firma') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ya existe el almacén de firma de Bolsa';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_firma AUTHORIZATION vec_bolsa_firma_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_firma FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_firma_propietario
    IN SCHEMA vec_bolsa_firma REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_firma_propietario
    IN SCHEMA vec_bolsa_firma REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_firma_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;

CREATE FUNCTION vec_bolsa_firma.texto_opaco_valido(
    p_valor text, p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[^!-~]'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_bolsa_firma.huella_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_firma.huella_hmac_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~
      '^hmac-sha256:[a-z0-9][a-z0-9._-]{0,127}:[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_firma.entero_canonico_valido(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[1-9][0-9]{0,18}$'
       AND p_valor::numeric <= 9223372036854775807
$funcion$;

CREATE FUNCTION vec_bolsa_firma.instante_utc_valido(p_valor text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE convertido timestamptz;
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,9})?Z$'
    THEN
        RETURN false;
    END IF;
    convertido := p_valor::timestamptz;
    RETURN convertido IS NOT NULL;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'la historia de firma es inmutable';
END
$funcion$;

CREATE TABLE vec_bolsa_firma.estado_cifrado (
    huella_sha256 text PRIMARY KEY,
    cifrado bytea NOT NULL,
    registrado_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CHECK (vec_bolsa_firma.huella_sha256_valida(huella_sha256)),
    CHECK (octet_length(cifrado) BETWEEN 16 AND 67108928),
    CHECK (encode(sha256(cifrado), 'hex') = huella_sha256)
);

CREATE TABLE vec_bolsa_firma.flujo (
    flujo_ref text PRIMARY KEY,
    indice_idempotencia_hmac text NOT NULL UNIQUE,
    huella_solicitud_hmac text NOT NULL,
    vinculo_actor_hmac text NOT NULL,
    perfil_actor_clave text NOT NULL,
    proceso_ref text NOT NULL,
    solicitud_ref text NOT NULL,
    baremacion_merito_ref text NOT NULL,
    decision_ref text NOT NULL,
    version_actual bigint NOT NULL,
    secuencia_cercado bigint NOT NULL DEFAULT 0,
    creado_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CHECK (vec_bolsa_firma.texto_opaco_valido(flujo_ref, 512)),
    CHECK (vec_bolsa_firma.huella_hmac_valida(indice_idempotencia_hmac)),
    CHECK (vec_bolsa_firma.huella_hmac_valida(huella_solicitud_hmac)),
    CHECK (vec_bolsa_firma.huella_hmac_valida(vinculo_actor_hmac)),
    CHECK (perfil_actor_clave ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CHECK (vec_bolsa_firma.texto_opaco_valido(proceso_ref, 512)),
    CHECK (vec_bolsa_firma.texto_opaco_valido(solicitud_ref, 512)),
    CHECK (vec_bolsa_firma.texto_opaco_valido(baremacion_merito_ref, 512)),
    CHECK (vec_bolsa_firma.texto_opaco_valido(decision_ref, 512)),
    CHECK (version_actual > 0 AND secuencia_cercado >= 0)
);

CREATE TABLE vec_bolsa_firma.version_flujo (
    flujo_ref text NOT NULL REFERENCES vec_bolsa_firma.flujo(flujo_ref),
    version bigint NOT NULL,
    expediente_documento jsonb NOT NULL,
    huella_documento_sha256 text NOT NULL,
    huella_cifrado_sha256 text NOT NULL
        REFERENCES vec_bolsa_firma.estado_cifrado(huella_sha256),
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (flujo_ref, version),
    CHECK (version > 0),
    CHECK (jsonb_typeof(expediente_documento) = 'object'),
    CHECK (octet_length(convert_to(expediente_documento::text, 'UTF8'))
           BETWEEN 2 AND 262144),
    CHECK (vec_bolsa_firma.huella_sha256_valida(huella_documento_sha256)),
    CHECK (
        encode(
            sha256(convert_to(expediente_documento::text, 'UTF8')),
            'hex'
        ) = huella_documento_sha256
    )
);

CREATE TABLE vec_bolsa_firma.arrendamiento (
    flujo_ref text PRIMARY KEY REFERENCES vec_bolsa_firma.flujo(flujo_ref),
    propietario_ref text NOT NULL,
    secuencia_cercado bigint NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    huella_token_hmac bytea NOT NULL,
    adquirido_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CHECK (vec_bolsa_firma.texto_opaco_valido(propietario_ref, 512)),
    CHECK (secuencia_cercado > 0),
    CHECK (octet_length(huella_token_hmac) = 32),
    CHECK (expira_en > adquirido_en),
    CHECK (expira_en - adquirido_en <= interval '5 minutes')
);

-- Cierra también la proyección mutable: al confirmar cada transacción,
-- version_actual debe apuntar a una versión inmutable realmente existente.
ALTER TABLE vec_bolsa_firma.flujo
    ADD CONSTRAINT flujo_version_actual_fk
    FOREIGN KEY (flujo_ref, version_actual)
    REFERENCES vec_bolsa_firma.version_flujo(flujo_ref, version)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE vec_bolsa_firma.auditoria (
    secuencia bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    flujo_ref text NOT NULL,
    version bigint,
    tipo_evento text NOT NULL,
    detalle jsonb NOT NULL,
    ocurrido_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text,
    huella_evento_sha256 text NOT NULL UNIQUE,
    CHECK (version IS NULL OR version > 0),
    CHECK (tipo_evento IN (
        'flujo_creado', 'version_guardada',
        'arrendamiento_adquirido', 'arrendamiento_liberado'
    )),
    CHECK (jsonb_typeof(detalle) = 'object'),
    CHECK (
        huella_anterior_sha256 IS NULL OR
        vec_bolsa_firma.huella_sha256_valida(huella_anterior_sha256)
    ),
    CHECK (vec_bolsa_firma.huella_sha256_valida(huella_evento_sha256))
);

CREATE TABLE vec_bolsa_firma.outbox (
    evento_ref text PRIMARY KEY,
    flujo_ref text NOT NULL,
    version bigint NOT NULL,
    tipo_evento text NOT NULL,
    contenido jsonb NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    creado_en timestamptz(6) NOT NULL,
    publicado_en timestamptz(6),
    CHECK (vec_bolsa_firma.texto_opaco_valido(evento_ref, 512)),
    CHECK (version > 0),
    CHECK (tipo_evento IN ('flujo_creado', 'version_guardada')),
    CHECK (jsonb_typeof(contenido) = 'object'),
    CHECK (
        encode(sha256(convert_to(contenido::text, 'UTF8')), 'hex') =
        huella_contenido_sha256
    ),
    CHECK (publicado_en IS NULL OR publicado_en >= creado_en)
);

ALTER TABLE vec_bolsa_firma.estado_cifrado ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.estado_cifrado FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.flujo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.flujo FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.version_flujo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.version_flujo FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.arrendamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.arrendamiento FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.auditoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.auditoria FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_firma.outbox FORCE ROW LEVEL SECURITY;

CREATE POLICY propietario_estado ON vec_bolsa_firma.estado_cifrado
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');
CREATE POLICY propietario_flujo ON vec_bolsa_firma.flujo
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');
CREATE POLICY propietario_version ON vec_bolsa_firma.version_flujo
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');
CREATE POLICY propietario_arrendamiento ON vec_bolsa_firma.arrendamiento
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');
CREATE POLICY propietario_auditoria ON vec_bolsa_firma.auditoria
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');
CREATE POLICY propietario_outbox ON vec_bolsa_firma.outbox
    FOR ALL TO vec_bolsa_firma_propietario
    USING (current_user = 'vec_bolsa_firma_propietario')
    WITH CHECK (current_user = 'vec_bolsa_firma_propietario');

CREATE TRIGGER estado_cifrado_inmutable
    BEFORE UPDATE OR DELETE ON vec_bolsa_firma.estado_cifrado
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable();
CREATE TRIGGER version_flujo_inmutable
    BEFORE UPDATE OR DELETE ON vec_bolsa_firma.version_flujo
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable();
CREATE TRIGGER auditoria_inmutable
    BEFORE UPDATE OR DELETE ON vec_bolsa_firma.auditoria
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable();
CREATE TRIGGER outbox_contenido_inmutable
    BEFORE UPDATE OF evento_ref, flujo_ref, version, tipo_evento,
                     contenido, huella_contenido_sha256, creado_en
    ON vec_bolsa_firma.outbox
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable();
CREATE TRIGGER outbox_no_borrable
    BEFORE DELETE ON vec_bolsa_firma.outbox
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_firma.rechazar_mutacion_inmutable();

CREATE FUNCTION vec_bolsa_firma.registrar_evidencia(
    p_flujo_ref text,
    p_version bigint,
    p_tipo_evento text,
    p_detalle jsonb,
    p_publicar boolean
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_instante timestamptz(6);
    v_anterior text;
    v_huella text;
    v_contenido jsonb;
BEGIN
    PERFORM pg_advisory_xact_lock(
        hashtextextended('vec_bolsa_firma:cadena_auditoria:v1', 0)
    );
    v_instante := clock_timestamp();
    SELECT huella_evento_sha256 INTO v_anterior
      FROM vec_bolsa_firma.auditoria
     ORDER BY secuencia DESC LIMIT 1;
    v_huella := encode(sha256(convert_to(
        concat_ws(chr(31), 'vec.bolsa.firma.auditoria.v1',
            coalesce(v_anterior, ''), p_flujo_ref,
            coalesce(p_version::text, ''), p_tipo_evento,
            p_detalle::text,
            to_char(v_instante AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
        'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_firma.auditoria(
        flujo_ref, version, tipo_evento, detalle, ocurrido_en,
        huella_anterior_sha256, huella_evento_sha256
    ) VALUES (
        p_flujo_ref, p_version, p_tipo_evento, p_detalle, v_instante,
        v_anterior, v_huella
    );
    IF p_publicar THEN
        v_contenido := jsonb_build_object(
            'esquema', 'vec.bolsa.firma.evento.v1',
            'flujo_ref', p_flujo_ref,
            'version', p_version::text,
            'tipo_evento', p_tipo_evento,
            'auditoria_sha256', v_huella,
            'ocurrido_en', to_char(
                v_instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        );
        INSERT INTO vec_bolsa_firma.outbox(
            evento_ref, flujo_ref, version, tipo_evento, contenido,
            huella_contenido_sha256, creado_en
        ) VALUES (
            'firma:' || p_flujo_ref || ':v' || p_version::text,
            p_flujo_ref, p_version, p_tipo_evento, v_contenido,
            encode(sha256(convert_to(v_contenido::text, 'UTF8')), 'hex'),
            v_instante
        );
    END IF;
END
$funcion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_firma FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_firma FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_firma FROM PUBLIC;
REVOKE ALL ON SCHEMA vec_bolsa_firma FROM vec_bolsa_firma_ejecutor;
COMMIT;
