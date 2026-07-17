-- Puerta durable y de uso unico para decisiones VEC-AD-2 ya verificadas.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion_dependencias$
BEGIN
    IF to_regprocedure(
           'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el cotejo transaccional del catalogo VEC-AD-2';
    END IF;
END
$prevalidacion_dependencias$;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000001', 0
    )
);

CREATE SCHEMA vec_autorizacion_atestada_v2
    AUTHORIZATION vec_autorizacion_atestada_v2_propietario;
REVOKE ALL ON SCHEMA vec_autorizacion_atestada_v2 FROM PUBLIC;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    IN SCHEMA vec_autorizacion_atestada_v2
    REVOKE ALL ON TABLES FROM PUBLIC,
        vec_autorizacion_atestada_v2_emisor_capacidad,
        vec_autorizacion_atestada_v2_consumidor;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    IN SCHEMA vec_autorizacion_atestada_v2
    REVOKE ALL ON SEQUENCES FROM PUBLIC,
        vec_autorizacion_atestada_v2_emisor_capacidad,
        vec_autorizacion_atestada_v2_consumidor;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC,
        vec_autorizacion_atestada_v2_emisor_capacidad,
        vec_autorizacion_atestada_v2_consumidor;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    REVOKE ALL ON TYPES FROM PUBLIC,
        vec_autorizacion_atestada_v2_emisor_capacidad,
        vec_autorizacion_atestada_v2_consumidor;

CREATE FUNCTION vec_autorizacion_atestada_v2.texto_tecnico_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor ~ '^[!-~]+$'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.huella_sha256_valida(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.sujeto_hmac_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor COLLATE "C" ~
        '^hmac-sha256:[a-z][a-z0-9._-]{0,127}:[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.instante_texto_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RETURN p_valor ~
        '^[0-9]{4}-(0[1-9]|1[0-2])-([0-2][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]{1,6})?Z$'
       AND (p_valor::timestamptz)::text IS NOT NULL;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Longitud binaria big-endian de 64 bits seguida de UTF-8. La preimagen no
-- depende de separadores ni del orden textual que jsonb elija al imprimir.
CREATE FUNCTION vec_autorizacion_atestada_v2.encuadrar_capacidad(
    p_valor text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT int8send(octet_length(convert_to(p_valor, 'UTF8'))::bigint)
        || convert_to(p_valor, 'UTF8')
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.preimagen_capacidad(
    p_capacidad jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'esquema'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'clave_id'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'clave_version'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'emisor_id'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'audiencia'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'nonce'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'emitida_en'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'expira_en'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'registro_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'consumo_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'decision_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_decision_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_motivo_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_payload_vec_ad_2_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_sobre_cose_sign1_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_evidencia_verificacion_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'principal_id'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'accion'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'finalidad'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'sujeto_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'recurso_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'contexto_recurso_huella_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'correlacion_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'decision_valida_hasta'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'efecto_ref'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_efecto_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'verificada_en'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'revision_confianza'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_configuracion_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'configuracion_publicada_en'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'configuracion_expira_en'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'raiz_clave_id'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'raiz_version'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'huella_raiz_spki_sha256'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'raiz_valida_desde'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'raiz_valida_hasta'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'suite'
           ) ||
           vec_autorizacion_atestada_v2.encuadrar_capacidad(
               p_capacidad ->> 'audiencia_despliegue'
           )
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.bytea_igual_constante(
    p_esperado bytea,
    p_recibido bytea
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    indice integer;
    maximo integer;
    diferencia integer;
BEGIN
    IF octet_length(p_esperado) = 0 OR octet_length(p_recibido) = 0 THEN
        RETURN octet_length(p_esperado) = octet_length(p_recibido);
    END IF;
    maximo := greatest(octet_length(p_esperado), octet_length(p_recibido));
    diferencia := octet_length(p_esperado) # octet_length(p_recibido);
    FOR indice IN 0..maximo - 1 LOOP
        diferencia := diferencia |
            (get_byte(p_esperado, indice % octet_length(p_esperado)) #
             get_byte(p_recibido, indice % octet_length(p_recibido)));
    END LOOP;
    RETURN diferencia = 0;
END
$funcion$;

CREATE TABLE vec_autorizacion_atestada_v2.clave_capacidad_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    secreto_hmac bytea NOT NULL,
    huella_secreto_sha256 text NOT NULL,
    emisor_id text NOT NULL,
    audiencia text NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    UNIQUE (version),
    CHECK (version BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        clave_id, 512
    )),
    CHECK (octet_length(secreto_hmac) BETWEEN 32 AND 128),
    CHECK (encode(sha256(secreto_hmac), 'hex') = huella_secreto_sha256),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        emisor_id, 512
    )),
    CHECK (audiencia =
        'vec_autorizacion_atestada_v2.registrar_y_consumir'),
    CHECK (valida_hasta > valida_desde),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        acto_ref, 512
    ))
);

CREATE TABLE vec_autorizacion_atestada_v2.revocacion_clave_capacidad (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_autorizacion_atestada_v2.clave_capacidad_version(
            clave_id, version
        ),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        motivo_catalogado_ref, 512
    ))
);

CREATE TABLE vec_autorizacion_atestada_v2.puntero_clave_capacidad (
    orden numeric(20, 0) PRIMARY KEY,
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL UNIQUE,
    establecida_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_autorizacion_atestada_v2.clave_capacidad_version(
            clave_id, version
        ),
    CHECK (orden BETWEEN 1 AND 18446744073709551615)
);

CREATE TABLE vec_autorizacion_atestada_v2.atestacion_decision_v2 (
    registro_ref text PRIMARY KEY,
    decision_ref text NOT NULL UNIQUE,
    huella_decision_sha256 text NOT NULL UNIQUE,
    decision_canonica bytea NOT NULL,
    huella_motivo_sha256 text NOT NULL,
    motivo_canonico bytea NOT NULL,
    payload_vec_ad_2 bytea NOT NULL,
    huella_payload_vec_ad_2_sha256 text NOT NULL UNIQUE,
    sobre_cose_sign1 bytea NOT NULL,
    huella_sobre_cose_sign1_sha256 text NOT NULL UNIQUE,
    evidencia_verificacion_canonica bytea NOT NULL,
    huella_evidencia_verificacion_sha256 text NOT NULL UNIQUE,
    formato_vec_ad_version smallint NOT NULL DEFAULT 2,
    suite text NOT NULL,
    audiencia_despliegue text NOT NULL,
    principal_id text NOT NULL,
    accion text NOT NULL,
    finalidad text NOT NULL,
    sujeto_ref text NOT NULL,
    recurso_ref text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    correlacion_ref text NOT NULL,
    decision_valida_hasta timestamptz(6) NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    revision_confianza text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    configuracion_publicada_en timestamptz(6) NOT NULL,
    configuracion_expira_en timestamptz(6) NOT NULL,
    raiz_clave_id text NOT NULL,
    raiz_version numeric(20, 0) NOT NULL,
    raiz_publica_spki bytea NOT NULL,
    huella_raiz_spki_sha256 text NOT NULL,
    raiz_valida_desde timestamptz(6) NOT NULL,
    raiz_valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref)
        REFERENCES vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref
        ),
    FOREIGN KEY (revision_confianza, raiz_clave_id, raiz_version)
        REFERENCES vec_confianza_atestacion_v2.configuracion_raiz(
            configuracion_revision, clave_id, version
        ),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        registro_ref, 512
    )),
    CHECK (formato_vec_ad_version = 2
        AND suite = 'VEC-AD-2-COSE-EDDSA-1'),
    CHECK (vec_autorizacion_atestada_v2.sujeto_hmac_valido(sujeto_ref)),
    CHECK (correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'),
    CHECK (encode(sha256(decision_canonica), 'hex') =
        huella_decision_sha256),
    CHECK (encode(sha256(motivo_canonico), 'hex') =
        huella_motivo_sha256),
    CHECK (encode(sha256(payload_vec_ad_2), 'hex') =
        huella_payload_vec_ad_2_sha256),
    CHECK (encode(sha256(sobre_cose_sign1), 'hex') =
        huella_sobre_cose_sign1_sha256),
    CHECK (encode(sha256(evidencia_verificacion_canonica), 'hex') =
        huella_evidencia_verificacion_sha256),
    CHECK (encode(sha256(raiz_publica_spki), 'hex') =
        huella_raiz_spki_sha256),
    CHECK (octet_length(decision_canonica) BETWEEN 1 AND 524288
        AND octet_length(motivo_canonico) BETWEEN 1 AND 65536
        AND octet_length(payload_vec_ad_2) BETWEEN 1 AND 524288
        AND octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384
        AND octet_length(evidencia_verificacion_canonica)
            BETWEEN 1 AND 2097152
        AND octet_length(raiz_publica_spki) = 44),
    CHECK (verificada_en >= configuracion_publicada_en
        AND verificada_en < configuracion_expira_en
        AND verificada_en >= raiz_valida_desde
        AND verificada_en < raiz_valida_hasta
        AND registrada_en >= verificada_en
        AND registrada_en < decision_valida_hasta)
);

CREATE TABLE vec_autorizacion_atestada_v2.consumo_capacidad_v2 (
    clave_id text NOT NULL,
    clave_version numeric(20, 0) NOT NULL,
    nonce text NOT NULL,
    registro_ref text NOT NULL UNIQUE,
    huella_capacidad_sha256 text NOT NULL UNIQUE,
    capacidad jsonb NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    PRIMARY KEY (clave_id, clave_version, nonce),
    FOREIGN KEY (clave_id, clave_version)
        REFERENCES vec_autorizacion_atestada_v2.clave_capacidad_version(
            clave_id, version
        ),
    FOREIGN KEY (registro_ref)
        REFERENCES vec_autorizacion_atestada_v2.atestacion_decision_v2(
            registro_ref
        ),
    CHECK (nonce ~ '^[0-9a-f]{64}$'),
    CHECK (vec_autorizacion_atestada_v2.huella_sha256_valida(
        huella_capacidad_sha256
    )),
    CHECK (expira_en > emitida_en
        AND expira_en <= emitida_en + interval '5 seconds'
        AND consumida_en >= emitida_en AND consumida_en < expira_en)
);

CREATE TABLE vec_autorizacion_atestada_v2.consumo_decision_v2 (
    consumo_ref text PRIMARY KEY,
    registro_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    huella_decision_sha256 text NOT NULL,
    efecto_ref text NOT NULL UNIQUE,
    huella_efecto_sha256 text NOT NULL UNIQUE,
    principal_id text NOT NULL,
    accion text NOT NULL,
    finalidad text NOT NULL,
    sujeto_ref text NOT NULL,
    recurso_ref text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    correlacion_ref text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (registro_ref)
        REFERENCES vec_autorizacion_atestada_v2.atestacion_decision_v2(
            registro_ref
        ),
    FOREIGN KEY (decision_ref)
        REFERENCES vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref
        ),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        consumo_ref, 512
    )),
    CHECK (vec_autorizacion_atestada_v2.sujeto_hmac_valido(sujeto_ref)),
    CHECK (correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'),
    CHECK (vec_autorizacion_atestada_v2.huella_sha256_valida(
        huella_decision_sha256
    ) AND vec_autorizacion_atestada_v2.huella_sha256_valida(
        huella_efecto_sha256
    ) AND vec_autorizacion_atestada_v2.huella_sha256_valida(
        contexto_recurso_huella_sha256
    ))
);

CREATE TABLE vec_autorizacion_atestada_v2.control_cadena_auditoria (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia numeric(20, 0) NOT NULL,
    ultima_huella_sha256 text NOT NULL,
    CHECK (ultima_secuencia BETWEEN 0 AND 18446744073709551615),
    CHECK ((ultima_secuencia = 0 AND ultima_huella_sha256 = repeat('0', 64))
        OR (ultima_secuencia > 0 AND
            vec_autorizacion_atestada_v2.huella_sha256_valida(
                ultima_huella_sha256
            )))
);
INSERT INTO vec_autorizacion_atestada_v2.control_cadena_auditoria(
    control_id, ultima_secuencia, ultima_huella_sha256
) VALUES (true, 0, repeat('0', 64));

CREATE TABLE vec_autorizacion_atestada_v2.auditoria_consumo_v2 (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    consumo_ref text NOT NULL UNIQUE,
    registro_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL UNIQUE,
    accion text NOT NULL,
    finalidad text NOT NULL,
    correlacion_ref text NOT NULL,
    ocurrida_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    FOREIGN KEY (consumo_ref)
        REFERENCES vec_autorizacion_atestada_v2.consumo_decision_v2(
            consumo_ref
        ),
    CHECK (vec_autorizacion_atestada_v2.texto_tecnico_valido(
        auditoria_ref, 512
    )),
    CHECK (vec_autorizacion_atestada_v2.huella_sha256_valida(
        huella_anterior_sha256
    ) AND vec_autorizacion_atestada_v2.huella_sha256_valida(
        huella_registro_sha256
    ))
);

CREATE FUNCTION vec_autorizacion_atestada_v2.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia atestada V2 inmutable';
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v2.validar_gobierno_clave()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    anterior record;
    destino record;
BEGIN
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v2:gobierno_clave:v1', 0
        )
    );
    IF TG_TABLE_NAME = 'revocacion_clave_capacidad' THEN
        SELECT valida_desde, valida_hasta INTO STRICT destino
          FROM vec_autorizacion_atestada_v2.clave_capacidad_version
         WHERE clave_id = NEW.clave_id AND version = NEW.version
         FOR SHARE;
        IF NEW.revocada_en < destino.valida_desde THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'revocacion anterior a la clave';
        END IF;
        RETURN NEW;
    END IF;
    SELECT version, valida_desde, valida_hasta INTO STRICT destino
      FROM vec_autorizacion_atestada_v2.clave_capacidad_version
     WHERE clave_id = NEW.clave_id AND version = NEW.version
     FOR SHARE;
    IF NEW.establecida_en < destino.valida_desde
       OR NEW.establecida_en >= destino.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v2.revocacion_clave_capacidad
            WHERE clave_id = NEW.clave_id AND version = NEW.version
              AND revocada_en <= NEW.establecida_en
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destino de clave de capacidad invalido';
    END IF;
    SELECT orden, version, establecida_en INTO anterior
      FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad
     ORDER BY orden DESC LIMIT 1 FOR SHARE;
    IF FOUND AND (NEW.orden <= anterior.orden
       OR NEW.version <= anterior.version
       OR NEW.establecida_en <= anterior.establecida_en) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'transicion de clave no monotona';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER puntero_clave_validar
    BEFORE INSERT ON vec_autorizacion_atestada_v2.puntero_clave_capacidad
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion_atestada_v2.validar_gobierno_clave();
CREATE TRIGGER revocacion_clave_validar
    BEFORE INSERT ON vec_autorizacion_atestada_v2.revocacion_clave_capacidad
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion_atestada_v2.validar_gobierno_clave();

DO $inmutabilidad$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'clave_capacidad_version', 'revocacion_clave_capacidad',
        'puntero_clave_capacidad', 'atestacion_decision_v2',
        'consumo_capacidad_v2', 'consumo_decision_v2',
        'auditoria_consumo_v2'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v2.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v2.rechazar_mutacion_inmutable()',
            tabla || '_inmutable', tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I BEFORE TRUNCATE ON vec_autorizacion_atestada_v2.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v2.rechazar_mutacion_inmutable()',
            tabla || '_no_truncar', tabla
        );
    END LOOP;
END
$inmutabilidad$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'clave_capacidad_version', 'revocacion_clave_capacidad',
        'puntero_clave_capacidad', 'atestacion_decision_v2',
        'consumo_capacidad_v2', 'consumo_decision_v2',
        'control_cadena_auditoria', 'auditoria_consumo_v2'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion_atestada_v2.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_autorizacion_atestada_v2.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v2.%I FOR ALL TO vec_autorizacion_atestada_v2_propietario USING (current_user = ''vec_autorizacion_atestada_v2_propietario'') WITH CHECK (current_user = ''vec_autorizacion_atestada_v2_propietario'')',
            tabla
        );
    END LOOP;
END
$rls$;

CREATE FUNCTION vec_autorizacion_atestada_v2.identidad_runtime_valida(
    p_rol text,
    p_exclusiva boolean
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    identidad record;
    oid_sesion oid;
BEGIN
    SELECT oid, rolcanlogin, rolsuper, rolcreatedb, rolcreaterole,
           rolreplication, rolbypassrls
      INTO identidad
      FROM pg_catalog.pg_roles WHERE rolname = session_user;
    oid_sesion := identidad.oid;
    RETURN identidad IS NOT NULL AND identidad.rolcanlogin
       AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
       AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
       AND NOT identidad.rolbypassrls
       AND pg_catalog.pg_has_role(session_user, p_rol, 'MEMBER')
       AND (NOT p_exclusiva OR (
           SELECT count(*) FROM pg_catalog.pg_auth_members
            WHERE member = oid_sesion
       ) = 1);
END
$funcion$;

-- Solo el broker aislado obtiene el secreto. Su LOGIN debe tener exactamente
-- una membresia: emisor_capacidad. El verificador del catalogo usa otra cuenta.
CREATE FUNCTION
vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()
RETURNS TABLE (
    clave_id text,
    clave_version numeric(20, 0),
    secreto_hmac bytea,
    huella_secreto_sha256 text,
    emisor_id text,
    audiencia text,
    valida_desde timestamptz(6),
    valida_hasta timestamptz(6)
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    instante timestamptz(6);
BEGIN
    IF vec_autorizacion_atestada_v2.identidad_runtime_valida(
           'vec_autorizacion_atestada_v2_emisor_capacidad', true
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad emisora de capacidad rechazada';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v2:gobierno_clave:v1', 0
        )
    );
    instante := clock_timestamp();
    RETURN QUERY
    SELECT clave.clave_id, clave.version, clave.secreto_hmac,
           clave.huella_secreto_sha256, clave.emisor_id, clave.audiencia,
           clave.valida_desde, clave.valida_hasta
      FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad AS puntero
      JOIN vec_autorizacion_atestada_v2.clave_capacidad_version AS clave
        ON clave.clave_id = puntero.clave_id
       AND clave.version = puntero.version
     WHERE puntero.orden = (
               SELECT max(ultimo.orden)
                 FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad
                     AS ultimo
           )
       AND puntero.establecida_en <= instante
       AND instante >= clave.valida_desde
       AND instante < clave.valida_hasta
       AND NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v2.revocacion_clave_capacidad
                 AS revocacion
            WHERE revocacion.clave_id = clave.clave_id
              AND revocacion.version = clave.version
              AND revocacion.revocada_en <= instante
       )
     FOR SHARE OF puntero, clave;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_payload_vec_ad_2 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea,
    p_capacidad jsonb
)
RETURNS TABLE (
    registro_ref text,
    consumo_ref text,
    auditoria_ref text,
    registrada_en timestamptz(6)
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    decision jsonb;
    clave record;
    instante timestamptz(6);
    mac_esperado bytea;
    secuencia numeric(20, 0);
    huella_anterior text;
    huella_auditoria text;
    referencia_auditoria text;
    claves_texto text[] := ARRAY[
        'esquema', 'clave_id', 'emisor_id', 'audiencia', 'nonce',
        'emitida_en', 'expira_en', 'registro_ref', 'consumo_ref',
        'decision_ref', 'huella_decision_sha256', 'huella_motivo_sha256',
        'huella_payload_vec_ad_2_sha256',
        'huella_sobre_cose_sign1_sha256',
        'huella_evidencia_verificacion_sha256', 'principal_id', 'accion',
        'finalidad', 'sujeto_ref', 'recurso_ref',
        'contexto_recurso_huella_sha256', 'correlacion_ref',
        'decision_valida_hasta', 'efecto_ref', 'huella_efecto_sha256',
        'verificada_en', 'revision_confianza',
        'huella_configuracion_sha256', 'configuracion_publicada_en',
        'configuracion_expira_en', 'raiz_clave_id',
        'huella_raiz_spki_sha256', 'raiz_valida_desde',
        'raiz_valida_hasta', 'suite', 'audiencia_despliegue', 'mac_sha256'
    ];
BEGIN
    IF vec_autorizacion_atestada_v2.identidad_runtime_valida(
           'vec_autorizacion_atestada_v2_consumidor', false
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad consumidora atestada rechazada';
    END IF;
    -- Antes de interpretar bytes de decision se validan forma, hashes y MAC.
    IF p_capacidad IS NULL OR jsonb_typeof(p_capacidad) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_capacidad)) <> 39
       OR NOT (p_capacidad ?& (claves_texto || ARRAY[
           'clave_version', 'raiz_version'
       ]))
       OR EXISTS (
           SELECT 1 FROM unnest(claves_texto) AS clave_texto
            WHERE jsonb_typeof(p_capacidad -> clave_texto) <> 'string'
       )
       OR jsonb_typeof(p_capacidad -> 'clave_version') <> 'number'
       OR jsonb_typeof(p_capacidad -> 'raiz_version') <> 'number'
       OR (p_capacidad ->> 'clave_version') !~ '^[1-9][0-9]{0,19}$'
       OR (p_capacidad ->> 'raiz_version') !~ '^[1-9][0-9]{0,19}$'
       OR (p_capacidad ->> 'clave_version')::numeric NOT BETWEEN
          1 AND 18446744073709551615
       OR (p_capacidad ->> 'raiz_version')::numeric NOT BETWEEN
          1 AND 18446744073709551615
       OR p_capacidad ->> 'esquema' <>
          'vec.autorizacion.capacidad-registro-consumo-atestado.v2'
       OR p_capacidad ->> 'audiencia' <>
          'vec_autorizacion_atestada_v2.registrar_y_consumir'
       OR p_capacidad ->> 'nonce' !~ '^[0-9a-f]{64}$'
       OR p_capacidad ->> 'correlacion_ref' !~
          '^correlacion_[0-9a-f]{32}$'
       OR vec_autorizacion_atestada_v2.sujeto_hmac_valido(
              p_capacidad ->> 'sujeto_ref'
          ) IS NOT TRUE
       OR p_capacidad ->> 'suite' <> 'VEC-AD-2-COSE-EDDSA-1'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_decision_sha256', 'huella_motivo_sha256',
               'huella_payload_vec_ad_2_sha256',
               'huella_sobre_cose_sign1_sha256',
               'huella_evidencia_verificacion_sha256',
               'contexto_recurso_huella_sha256', 'huella_efecto_sha256',
               'huella_configuracion_sha256', 'huella_raiz_spki_sha256',
               'mac_sha256'
           ]) AS clave_huella
            WHERE vec_autorizacion_atestada_v2.huella_sha256_valida(
                p_capacidad ->> clave_huella
            ) IS NOT TRUE
       )
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'emitida_en', 'expira_en', 'decision_valida_hasta',
               'verificada_en', 'configuracion_publicada_en',
               'configuracion_expira_en', 'raiz_valida_desde',
               'raiz_valida_hasta'
           ]) AS clave_instante
            WHERE vec_autorizacion_atestada_v2.instante_texto_valido(
                p_capacidad ->> clave_instante
            ) IS NOT TRUE
       )
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'clave_id', 'emisor_id', 'registro_ref', 'consumo_ref',
               'decision_ref', 'principal_id', 'accion', 'finalidad',
               'recurso_ref', 'efecto_ref', 'revision_confianza',
               'raiz_clave_id', 'audiencia_despliegue'
           ]) AS clave_referencia
            WHERE vec_autorizacion_atestada_v2.texto_tecnico_valido(
                p_capacidad ->> clave_referencia, 512
            ) IS NOT TRUE
       )
       OR p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR p_payload_vec_ad_2 IS NULL OR p_sobre_cose_sign1 IS NULL
       OR p_evidencia_verificacion IS NULL OR p_raiz_publica_spki IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR octet_length(p_payload_vec_ad_2) NOT BETWEEN 1 AND 524288
       OR octet_length(p_sobre_cose_sign1) NOT BETWEEN 16 AND 528384
       OR octet_length(p_evidencia_verificacion) NOT BETWEEN 1 AND 2097152
       OR octet_length(p_raiz_publica_spki) <> 44 THEN
        RETURN;
    END IF;

    IF encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_decision_sha256'
       OR encode(sha256(p_motivo_canonico), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_motivo_sha256'
       OR encode(sha256(p_payload_vec_ad_2), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_payload_vec_ad_2_sha256'
       OR encode(sha256(p_sobre_cose_sign1), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_sobre_cose_sign1_sha256'
       OR encode(sha256(p_evidencia_verificacion), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_evidencia_verificacion_sha256'
       OR encode(sha256(p_raiz_publica_spki), 'hex') IS DISTINCT FROM
           p_capacidad ->> 'huella_raiz_spki_sha256' THEN
        RETURN;
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v2:gobierno_clave:v1', 0
        )
    );
    instante := clock_timestamp();
    SELECT capacidad.* INTO clave
      FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad AS puntero
      JOIN vec_autorizacion_atestada_v2.clave_capacidad_version AS capacidad
        ON capacidad.clave_id = puntero.clave_id
       AND capacidad.version = puntero.version
     WHERE puntero.orden = (
               SELECT max(ultimo.orden)
                 FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad
                     AS ultimo
           )
       AND puntero.establecida_en <= instante
       AND NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v2.revocacion_clave_capacidad
                 AS revocacion
            WHERE revocacion.clave_id = capacidad.clave_id
              AND revocacion.version = capacidad.version
              AND revocacion.revocada_en <= instante
       )
     FOR SHARE OF puntero, capacidad;
    IF NOT FOUND
       OR clave.clave_id IS DISTINCT FROM p_capacidad ->> 'clave_id'
       OR clave.version IS DISTINCT FROM
          (p_capacidad ->> 'clave_version')::numeric
       OR clave.emisor_id IS DISTINCT FROM p_capacidad ->> 'emisor_id'
       OR clave.audiencia IS DISTINCT FROM p_capacidad ->> 'audiencia'
       OR instante < clave.valida_desde OR instante >= clave.valida_hasta
       OR (p_capacidad ->> 'emitida_en')::timestamptz < clave.valida_desde
       OR (p_capacidad ->> 'expira_en')::timestamptz > clave.valida_hasta
       OR (p_capacidad ->> 'expira_en')::timestamptz <=
          (p_capacidad ->> 'emitida_en')::timestamptz
       OR (p_capacidad ->> 'expira_en')::timestamptz >
          (p_capacidad ->> 'emitida_en')::timestamptz + interval '5 seconds'
       OR instante < (p_capacidad ->> 'emitida_en')::timestamptz
       OR instante >= (p_capacidad ->> 'expira_en')::timestamptz THEN
        RETURN;
    END IF;
    mac_esperado := public.hmac(
        vec_autorizacion_atestada_v2.preimagen_capacidad(p_capacidad),
        clave.secreto_hmac,
        'sha256'
    );
    IF vec_autorizacion_atestada_v2.bytea_igual_constante(
           mac_esperado,
           decode(p_capacidad ->> 'mac_sha256', 'hex')
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    -- El cotejo toma el mismo advisory xact lock compartido que la lectura
    -- oficial del catálogo. Una revocación o rotación queda serializada hasta
    -- el COMMIT que también consumirá la decisión y aplicará el efecto.
    IF vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
           p_capacidad ->> 'revision_confianza',
           p_capacidad ->> 'huella_configuracion_sha256',
           (p_capacidad ->> 'configuracion_publicada_en')::timestamptz,
           (p_capacidad ->> 'configuracion_expira_en')::timestamptz,
           p_capacidad ->> 'raiz_clave_id',
           (p_capacidad ->> 'raiz_version')::numeric,
           p_capacidad ->> 'huella_raiz_spki_sha256',
           (p_capacidad ->> 'raiz_valida_desde')::timestamptz,
           (p_capacidad ->> 'raiz_valida_hasta')::timestamptz,
           p_capacidad ->> 'suite',
           p_capacidad ->> 'audiencia_despliegue',
           (p_capacidad ->> 'verificada_en')::timestamptz
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    BEGIN
        decision := convert_from(p_decision_canonica, 'UTF8')::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN;
    END;
    IF jsonb_typeof(decision) <> 'object'
       OR decision ->> 'decision_ref' IS DISTINCT FROM
          p_capacidad ->> 'decision_ref'
       OR decision ->> 'motivo_huella_sha256' IS DISTINCT FROM
          p_capacidad ->> 'huella_motivo_sha256'
       OR decision ->> 'principal_id' IS DISTINCT FROM
          p_capacidad ->> 'principal_id'
       OR decision ->> 'accion' IS DISTINCT FROM p_capacidad ->> 'accion'
       OR decision ->> 'finalidad' IS DISTINCT FROM
          p_capacidad ->> 'finalidad'
       OR decision ->> 'recurso_ref' IS DISTINCT FROM
          p_capacidad ->> 'recurso_ref'
       OR decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
          p_capacidad ->> 'contexto_recurso_huella_sha256'
       OR decision ->> 'correlacion_ref' IS DISTINCT FROM
          p_capacidad ->> 'correlacion_ref'
       OR (decision ->> 'valida_hasta')::timestamptz IS DISTINCT FROM
          (p_capacidad ->> 'decision_valida_hasta')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz <
          (decision ->> 'emitida_en')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz >=
          (decision ->> 'valida_hasta')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz <
          (p_capacidad ->> 'configuracion_publicada_en')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz >=
          (p_capacidad ->> 'configuracion_expira_en')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz <
          (p_capacidad ->> 'raiz_valida_desde')::timestamptz
       OR (p_capacidad ->> 'verificada_en')::timestamptz >=
          (p_capacidad ->> 'raiz_valida_hasta')::timestamptz
       OR (p_capacidad ->> 'emitida_en')::timestamptz <
          (p_capacidad ->> 'verificada_en')::timestamptz
       OR (p_capacidad ->> 'emitida_en')::timestamptz -
          (p_capacidad ->> 'verificada_en')::timestamptz >
          interval '5 seconds' THEN
        RETURN;
    END IF;

    -- La autoridad nominal se revalida y registra dentro de ESTA transaccion.
    -- La capacidad no sustituye RBAC/ABAC, sesion, motivo ni vigencia.
    IF vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
           p_decision_canonica, p_motivo_canonico
       ) IS NOT TRUE THEN
        RETURN;
    END IF;
    instante := clock_timestamp();
    IF instante >= (p_capacidad ->> 'expira_en')::timestamptz
       OR instante >= (p_capacidad ->> 'decision_valida_hasta')::timestamptz
       OR instante >= clave.valida_hasta THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'capacidad o decision caduco durante la revalidacion';
    END IF;

    INSERT INTO vec_autorizacion_atestada_v2.atestacion_decision_v2 (
        registro_ref, decision_ref, huella_decision_sha256,
        decision_canonica, huella_motivo_sha256, motivo_canonico,
        payload_vec_ad_2, huella_payload_vec_ad_2_sha256,
        sobre_cose_sign1, huella_sobre_cose_sign1_sha256,
        evidencia_verificacion_canonica,
        huella_evidencia_verificacion_sha256, suite,
        audiencia_despliegue, principal_id, accion, finalidad, sujeto_ref,
        recurso_ref, contexto_recurso_huella_sha256, correlacion_ref,
        decision_valida_hasta, verificada_en, revision_confianza,
        huella_configuracion_sha256, configuracion_publicada_en,
        configuracion_expira_en, raiz_clave_id, raiz_version,
        raiz_publica_spki, huella_raiz_spki_sha256, raiz_valida_desde,
        raiz_valida_hasta, registrada_en
    ) VALUES (
        p_capacidad ->> 'registro_ref', p_capacidad ->> 'decision_ref',
        p_capacidad ->> 'huella_decision_sha256', p_decision_canonica,
        p_capacidad ->> 'huella_motivo_sha256', p_motivo_canonico,
        p_payload_vec_ad_2,
        p_capacidad ->> 'huella_payload_vec_ad_2_sha256',
        p_sobre_cose_sign1,
        p_capacidad ->> 'huella_sobre_cose_sign1_sha256',
        p_evidencia_verificacion,
        p_capacidad ->> 'huella_evidencia_verificacion_sha256',
        p_capacidad ->> 'suite', p_capacidad ->> 'audiencia_despliegue',
        p_capacidad ->> 'principal_id', p_capacidad ->> 'accion',
        p_capacidad ->> 'finalidad', p_capacidad ->> 'sujeto_ref',
        p_capacidad ->> 'recurso_ref',
        p_capacidad ->> 'contexto_recurso_huella_sha256',
        p_capacidad ->> 'correlacion_ref',
        (p_capacidad ->> 'decision_valida_hasta')::timestamptz,
        (p_capacidad ->> 'verificada_en')::timestamptz,
        p_capacidad ->> 'revision_confianza',
        p_capacidad ->> 'huella_configuracion_sha256',
        (p_capacidad ->> 'configuracion_publicada_en')::timestamptz,
        (p_capacidad ->> 'configuracion_expira_en')::timestamptz,
        p_capacidad ->> 'raiz_clave_id',
        (p_capacidad ->> 'raiz_version')::numeric, p_raiz_publica_spki,
        p_capacidad ->> 'huella_raiz_spki_sha256',
        (p_capacidad ->> 'raiz_valida_desde')::timestamptz,
        (p_capacidad ->> 'raiz_valida_hasta')::timestamptz, instante
    );
    INSERT INTO vec_autorizacion_atestada_v2.consumo_capacidad_v2 (
        clave_id, clave_version, nonce, registro_ref,
        huella_capacidad_sha256, capacidad, emitida_en, expira_en,
        consumida_en
    ) VALUES (
        p_capacidad ->> 'clave_id',
        (p_capacidad ->> 'clave_version')::numeric,
        p_capacidad ->> 'nonce', p_capacidad ->> 'registro_ref',
        encode(sha256(convert_to(p_capacidad::text, 'UTF8')), 'hex'),
        p_capacidad, (p_capacidad ->> 'emitida_en')::timestamptz,
        (p_capacidad ->> 'expira_en')::timestamptz, instante
    );
    INSERT INTO vec_autorizacion_atestada_v2.consumo_decision_v2 (
        consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
        efecto_ref, huella_efecto_sha256, principal_id, accion, finalidad,
        sujeto_ref, recurso_ref, contexto_recurso_huella_sha256,
        correlacion_ref, consumida_en
    ) VALUES (
        p_capacidad ->> 'consumo_ref', p_capacidad ->> 'registro_ref',
        p_capacidad ->> 'decision_ref',
        p_capacidad ->> 'huella_decision_sha256',
        p_capacidad ->> 'efecto_ref',
        p_capacidad ->> 'huella_efecto_sha256',
        p_capacidad ->> 'principal_id', p_capacidad ->> 'accion',
        p_capacidad ->> 'finalidad', p_capacidad ->> 'sujeto_ref',
        p_capacidad ->> 'recurso_ref',
        p_capacidad ->> 'contexto_recurso_huella_sha256',
        p_capacidad ->> 'correlacion_ref', instante
    );

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT secuencia, huella_anterior
      FROM vec_autorizacion_atestada_v2.control_cadena_auditoria
     WHERE control_id = true FOR UPDATE;
    secuencia := secuencia + 1;
    referencia_auditoria := 'auditoria-atestada:' || encode(sha256(
        vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'consumo_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'decision_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'efecto_ref'
        )
    ), 'hex');
    huella_auditoria := encode(sha256(
        vec_autorizacion_atestada_v2.encuadrar_capacidad(secuencia::text) ||
        vec_autorizacion_atestada_v2.encuadrar_capacidad(
            referencia_auditoria
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'registro_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'decision_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'efecto_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'accion'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'finalidad'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            p_capacidad ->> 'correlacion_ref'
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            to_char(instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ) || vec_autorizacion_atestada_v2.encuadrar_capacidad(
            huella_anterior
        )
    ), 'hex');
    INSERT INTO vec_autorizacion_atestada_v2.auditoria_consumo_v2 (
        auditoria_ref, secuencia, consumo_ref, registro_ref, decision_ref,
        efecto_ref, accion, finalidad, correlacion_ref, ocurrida_en,
        huella_anterior_sha256, huella_registro_sha256
    ) VALUES (
        referencia_auditoria, secuencia, p_capacidad ->> 'consumo_ref',
        p_capacidad ->> 'registro_ref', p_capacidad ->> 'decision_ref',
        p_capacidad ->> 'efecto_ref', p_capacidad ->> 'accion',
        p_capacidad ->> 'finalidad', p_capacidad ->> 'correlacion_ref',
        instante, huella_anterior, huella_auditoria
    );
    UPDATE vec_autorizacion_atestada_v2.control_cadena_auditoria
       SET ultima_secuencia = secuencia,
           ultima_huella_sha256 = huella_auditoria
     WHERE control_id = true;

    RETURN QUERY SELECT p_capacidad ->> 'registro_ref',
        p_capacidad ->> 'consumo_ref', referencia_auditoria, instante;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
    p_decision_ref text,
    p_huella_decision_sha256 text,
    p_efecto_ref text,
    p_huella_efecto_sha256 text,
    p_nonce text
)
RETURNS TABLE (
    registro_ref text,
    consumo_ref text,
    auditoria_ref text,
    consumida_en timestamptz(6),
    huella_auditoria_sha256 text
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF vec_autorizacion_atestada_v2.identidad_runtime_valida(
           'vec_autorizacion_atestada_v2_consumidor', false
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_decision_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_efecto_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.huella_sha256_valida(
              p_huella_decision_sha256
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.huella_sha256_valida(
              p_huella_efecto_sha256
          ) IS NOT TRUE
       OR p_nonce !~ '^[0-9a-f]{64}$' THEN
        RETURN;
    END IF;
    RETURN QUERY
    SELECT consumo.registro_ref, consumo.consumo_ref,
           auditoria.auditoria_ref, consumo.consumida_en,
           auditoria.huella_registro_sha256
      FROM vec_autorizacion_atestada_v2.consumo_decision_v2 AS consumo
      JOIN vec_autorizacion_atestada_v2.consumo_capacidad_v2 AS capacidad
        ON capacidad.registro_ref = consumo.registro_ref
      JOIN vec_autorizacion_atestada_v2.auditoria_consumo_v2 AS auditoria
        ON auditoria.consumo_ref = consumo.consumo_ref
     WHERE consumo.decision_ref = p_decision_ref
       AND consumo.huella_decision_sha256 = p_huella_decision_sha256
       AND consumo.efecto_ref = p_efecto_ref
       AND consumo.huella_efecto_sha256 = p_huella_efecto_sha256
       AND capacidad.nonce = p_nonce;
END
$funcion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_autorizacion_atestada_v2
    FROM PUBLIC, vec_autorizacion_atestada_v2_emisor_capacidad,
         vec_autorizacion_atestada_v2_consumidor;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_autorizacion_atestada_v2
    FROM PUBLIC, vec_autorizacion_atestada_v2_emisor_capacidad,
         vec_autorizacion_atestada_v2_consumidor;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_autorizacion_atestada_v2
    FROM PUBLIC, vec_autorizacion_atestada_v2_emisor_capacidad,
         vec_autorizacion_atestada_v2_consumidor;

DO $cerrar_tipos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_autorizacion_atestada_v2'
           AND definicion.typelem = 0 AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I',
            tipo.nspname, tipo.typname,
            'vec_autorizacion_atestada_v2_emisor_capacidad',
            'vec_autorizacion_atestada_v2_consumidor'
        );
    END LOOP;
END
$cerrar_tipos$;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v2
    TO vec_autorizacion_atestada_v2_emisor_capacidad,
       vec_autorizacion_atestada_v2_consumidor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()
    TO vec_autorizacion_atestada_v2_emisor_capacidad;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) TO vec_autorizacion_atestada_v2_consumidor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    ) TO vec_autorizacion_atestada_v2_consumidor;

COMMIT;
