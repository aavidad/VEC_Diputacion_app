BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $precondicion_pgcrypto$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = extension.extnamespace
         WHERE extension.extname = 'pgcrypto'
           AND espacio.nspname = 'public'
    ) OR pg_catalog.to_regprocedure(
        'public.hmac(bytea,bytea,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta pgcrypto instalado en el esquema public';
    END IF;
END
$precondicion_pgcrypto$;

CREATE SCHEMA vec_ejecucion_documental_v4
    AUTHORIZATION vec_ejecucion_documental_v4_propietario;
REVOKE ALL ON SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario
    IN SCHEMA vec_ejecucion_documental_v4 REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario
    IN SCHEMA vec_ejecucion_documental_v4 REVOKE ALL ON SEQUENCES FROM PUBLIC;
-- Las revocaciones por esquema no pueden reducir los privilegios globales que
-- PostgreSQL concede por defecto a PUBLIC en funciones y tipos. Por eso estas
-- dos son globales para el rol propietario, que es exclusivo del modulo V4.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_ejecucion_documental_v4.texto_tecnico_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor ~ '^[!-~]+$'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.huella_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.instante_valido(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9]{4}-(0[1-9]|1[0-2])-([0-2][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]{1,6})?Z$'
       AND (p_valor::timestamptz)::text IS NOT NULL
$funcion$;

-- Encuadre estable compartido con el emisor Go: longitud UTF-8 decimal,
-- dos puntos, valor y LF. El orden de los 19 campos se fija en la funcion
-- siguiente; mac_sha256 nunca forma parte de su propia preimagen.
CREATE FUNCTION vec_ejecucion_documental_v4.encuadrar_capacidad(p_valor text)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT convert_to(
        octet_length(convert_to(p_valor, 'UTF8'))::text || ':' ||
        p_valor || E'\n',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.preimagen_capacidad(
    p_capacidad jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'esquema'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'clave_id'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'clave_version'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'emisor_id'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'audiencia'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'nonce'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'emitida_en'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'expira_en'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_metadatos_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_payload_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_sobre_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_evidencia_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_preimagen_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_decision_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_efecto_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'revision_confianza'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_configuracion_sha256'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'raiz_clave_id'
           ) ||
           vec_ejecucion_documental_v4.encuadrar_capacidad(
               p_capacidad ->> 'huella_raiz_sha256'
           )
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.bytea_igual_constante(
    p_esperado bytea,
    p_recibido bytea
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    indice integer;
    diferencia integer;
BEGIN
    diferencia := octet_length(p_esperado) # octet_length(p_recibido);
    FOR indice IN 0..greatest(
        octet_length(p_esperado), octet_length(p_recibido)
    ) - 1 LOOP
        diferencia := diferencia |
            (get_byte(p_esperado, indice % octet_length(p_esperado)) #
             get_byte(p_recibido, indice % octet_length(p_recibido)));
    END LOOP;
    RETURN diferencia = 0;
END
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.aplicacion_valida(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_documento IS NULL OR jsonb_typeof(p_documento) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_documento)) <> 25
       OR NOT (p_documento ?& ARRAY[
           'esquema', 'decision_ref', 'huella_plan_sha256', 'efecto_ref',
           'esquema_huella_decision', 'huella_decision_sha256',
           'perfil_activo_ref', 'contexto_actor_huella_sha256', 'accion',
           'recurso_ref', 'modulo_id', 'tipo_recurso', 'huella_recurso_sha256',
           'huella_ambitos_sha256', 'finalidad', 'correlacion_ref',
           'huella_campos_permitidos_sha256', 'huella_obligaciones_sha256',
           'huella_cumplimientos_sha256', 'verificada_en', 'vinculada_en',
           'solicitada_en', 'valida_hasta',
           'huella_solicitud_vinculada_sha256',
           'huella_solicitud_aplicacion_sha256'
       ]) THEN
        RETURN false;
    END IF;
    IF p_documento ->> 'esquema' <>
           'vec.documentos.autorizacion-ejecucion.solicitud-aplicacion.v4'
       OR p_documento ->> 'esquema_huella_decision' <>
           'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR p_documento ->> 'accion' <>
           'vec.documentos.ejecucion.ejecutar_plan_v4'
       OR p_documento ->> 'decision_ref' = p_documento ->> 'efecto_ref'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'decision_ref', 'efecto_ref', 'perfil_activo_ref', 'recurso_ref',
               'modulo_id', 'tipo_recurso', 'finalidad', 'correlacion_ref'
           ]) AS clave
           WHERE jsonb_typeof(p_documento -> clave) <> 'string'
              OR vec_ejecucion_documental_v4.texto_tecnico_valido(
                  p_documento ->> clave,
                  CASE WHEN clave IN ('modulo_id', 'tipo_recurso')
                       THEN 128 ELSE 512 END
              ) IS NOT TRUE
       ) OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_plan_sha256', 'huella_decision_sha256',
               'contexto_actor_huella_sha256', 'huella_recurso_sha256',
               'huella_ambitos_sha256', 'huella_campos_permitidos_sha256',
               'huella_obligaciones_sha256', 'huella_cumplimientos_sha256',
               'huella_solicitud_vinculada_sha256',
               'huella_solicitud_aplicacion_sha256'
           ]) AS clave
           WHERE jsonb_typeof(p_documento -> clave) <> 'string'
              OR vec_ejecucion_documental_v4.huella_sha256_valida(
                  p_documento ->> clave
              ) IS NOT TRUE
       ) OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'verificada_en', 'vinculada_en', 'solicitada_en', 'valida_hasta'
           ]) AS clave
           WHERE vec_ejecucion_documental_v4.instante_valido(
               p_documento ->> clave
           ) IS NOT TRUE
       ) THEN
        RETURN false;
    END IF;
    RETURN (p_documento ->> 'vinculada_en')::timestamptz >=
               (p_documento ->> 'verificada_en')::timestamptz
       AND (p_documento ->> 'solicitada_en')::timestamptz >=
               (p_documento ->> 'vinculada_en')::timestamptz
       AND (p_documento ->> 'solicitada_en')::timestamptz <
               (p_documento ->> 'valida_hasta')::timestamptz;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Registro de confianza versionado y administrado exclusivamente por DBA.
-- El runtime no recibe INSERT/UPDATE/DELETE/SELECT sobre estas tablas.
CREATE TABLE vec_ejecucion_documental_v4.configuracion_confianza_version (
    revision text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    estado text NOT NULL,
    revocada_en timestamptz(6),
    acto_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (revision, huella_configuracion_sha256),
    UNIQUE (huella_configuracion_sha256),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(revision, 128)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_configuracion_sha256
    )),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512)),
    CHECK (estado IN ('activa', 'revocada')),
    CHECK (expira_en > publicada_en AND expira_en <= publicada_en + interval '24 hours'),
    CHECK ((estado = 'activa' AND revocada_en IS NULL)
        OR (estado = 'revocada' AND revocada_en IS NOT NULL
            AND revocada_en >= publicada_en))
);

CREATE TABLE vec_ejecucion_documental_v4.configuracion_confianza_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    revision text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    FOREIGN KEY (revision, huella_configuracion_sha256)
        REFERENCES vec_ejecucion_documental_v4.configuracion_confianza_version(
            revision, huella_configuracion_sha256
        ),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512))
);

CREATE TABLE vec_ejecucion_documental_v4.raiz_confianza_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    revision_configuracion text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    algoritmo_cose text NOT NULL,
    suite text NOT NULL,
    audiencia_cose text NOT NULL,
    audiencia_despliegue text NOT NULL,
    clave_publica_spki bytea NOT NULL,
    huella_clave_sha256 text NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    estado text NOT NULL,
    revocada_en timestamptz(6),
    acto_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    UNIQUE (clave_id, version, revision_configuracion,
            huella_configuracion_sha256),
    FOREIGN KEY (revision_configuracion, huella_configuracion_sha256)
        REFERENCES vec_ejecucion_documental_v4.configuracion_confianza_version(
            revision, huella_configuracion_sha256
        ),
    CHECK (version BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(clave_id, 512)),
    CHECK (algoritmo_cose = 'EdDSA' AND suite = 'VEC-AD-COSE-EDDSA-1'
        AND audiencia_cose = 'atestacion_autorizacion_pdp'),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(
        audiencia_despliegue, 512
    )),
    CHECK (octet_length(clave_publica_spki) BETWEEN 32 AND 4096),
    CHECK (encode(sha256(clave_publica_spki), 'hex') = huella_clave_sha256),
    CHECK (valida_hasta > valida_desde),
    CHECK (estado IN ('activa', 'revocada')),
    CHECK ((estado = 'activa' AND revocada_en IS NULL)
        OR (estado = 'revocada' AND revocada_en IS NOT NULL
            AND revocada_en >= valida_desde)),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512))
);

CREATE TABLE vec_ejecucion_documental_v4.raiz_confianza_actual (
    clave_id text PRIMARY KEY,
    version numeric(20, 0) NOT NULL,
    revision_configuracion text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    FOREIGN KEY (
        clave_id, version, revision_configuracion, huella_configuracion_sha256
    ) REFERENCES vec_ejecucion_documental_v4.raiz_confianza_version(
        clave_id, version, revision_configuracion, huella_configuracion_sha256
    ),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512))
);

-- Claves simetricas de capacidad separadas de la credencial que ejecuta.
-- Una version revocada puede repetir la huella de la version activa a la que
-- cierra; el puntero monotono impide volver despues a esa huella como activa.
CREATE TABLE vec_ejecucion_documental_v4.clave_capacidad_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL UNIQUE,
    secreto_hmac bytea NOT NULL,
    huella_secreto_sha256 text NOT NULL,
    emisor_id text NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    estado text NOT NULL,
    revocada_en timestamptz(6),
    acto_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    CHECK (version BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(clave_id, 512)),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(emisor_id, 512)),
    CHECK (octet_length(secreto_hmac) BETWEEN 32 AND 128),
    CHECK (encode(sha256(secreto_hmac), 'hex') = huella_secreto_sha256),
    CHECK (valida_hasta > valida_desde),
    CHECK (estado IN ('activa', 'revocada')),
    CHECK ((estado = 'activa' AND revocada_en IS NULL)
        OR (estado = 'revocada' AND revocada_en IS NOT NULL
            AND revocada_en >= valida_desde)),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512))
);

CREATE TABLE vec_ejecucion_documental_v4.clave_capacidad_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_ejecucion_documental_v4.clave_capacidad_version(
            clave_id, version
        ),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(acto_ref, 512))
);

CREATE TABLE vec_ejecucion_documental_v4.atestacion_pdp (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL,
    huella_plan_sha256 text NOT NULL,
    efecto_ref text NOT NULL,
    huella_solicitud_vinculada_sha256 text NOT NULL,
    aplicacion_registro jsonb NOT NULL,
    huella_preimagen_sha256 text NOT NULL,
    preimagen_recurso bytea NOT NULL,
    formato_vec_ad_version integer NOT NULL,
    suite text NOT NULL,
    clave_id text NOT NULL,
    version_raiz numeric(20, 0) NOT NULL,
    audiencia_despliegue text NOT NULL,
    algoritmo_cose text NOT NULL,
    audiencia_cose text NOT NULL,
    huella_clave_sha256 text NOT NULL,
    huella_payload_sha256 text NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    raiz_valida_desde timestamptz(6) NOT NULL,
    raiz_valida_hasta timestamptz(6) NOT NULL,
    revision_confianza text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    configuracion_publicada_en timestamptz(6) NOT NULL,
    configuracion_expira_en timestamptz(6) NOT NULL,
    huella_evidencia_sha256 text NOT NULL UNIQUE,
    payload_vec_ad_1 bytea NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    evidencia_canonica bytea NOT NULL,
    decision_canonica bytea NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (decision_ref, huella_evidencia_sha256, huella_decision_sha256,
            huella_plan_sha256, efecto_ref,
            huella_solicitud_vinculada_sha256),
    FOREIGN KEY (
        clave_id, version_raiz, revision_confianza,
        huella_configuracion_sha256
    ) REFERENCES vec_ejecucion_documental_v4.raiz_confianza_version(
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256
    ),
    CHECK (vec_ejecucion_documental_v4.aplicacion_valida(aplicacion_registro)),
    CHECK (formato_vec_ad_version = 1 AND suite = 'VEC-AD-COSE-EDDSA-1'
        AND algoritmo_cose = 'EdDSA'
        AND audiencia_cose = 'atestacion_autorizacion_pdp'),
    CHECK (encode(sha256(payload_vec_ad_1), 'hex') = huella_payload_sha256
        AND encode(sha256(sobre_cose_sign1), 'hex') = huella_sobre_sha256
        AND encode(sha256(evidencia_canonica), 'hex') = huella_evidencia_sha256
        AND encode(sha256(preimagen_recurso), 'hex') = huella_preimagen_sha256
        AND encode(sha256(decision_canonica), 'hex') = huella_decision_sha256),
    CHECK (octet_length(payload_vec_ad_1) BETWEEN 1 AND 524288
        AND octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384
        AND octet_length(evidencia_canonica) BETWEEN 1 AND 2097152
        AND octet_length(preimagen_recurso) BETWEEN 1 AND 2097152
        AND octet_length(decision_canonica) BETWEEN 128 AND 524288)
);

CREATE TABLE vec_ejecucion_documental_v4.orden_generacion_documental (
    orden_ref text PRIMARY KEY,
    estado text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL UNIQUE,
    huella_plan_sha256 text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_aplicacion_sha256 text NOT NULL,
    huella_orden_sha256 text NOT NULL UNIQUE,
    correlacion_ref text NOT NULL,
    solicitada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref) REFERENCES
        vec_ejecucion_documental_v4.atestacion_pdp(decision_ref),
    CHECK (estado = 'pendiente_generacion'),
    CHECK (orden_ref = efecto_ref),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(orden_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(correlacion_ref, 512)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_plan_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_decision_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_aplicacion_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_orden_sha256))
);

CREATE TABLE vec_ejecucion_documental_v4.consumo_decision_atomico (
    decision_ref text PRIMARY KEY REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(decision_ref),
    efecto_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(efecto_ref),
    orden_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(orden_ref),
    huella_decision_sha256 text NOT NULL,
    huella_aplicacion_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_decision_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(
            huella_aplicacion_sha256
        ))
);

CREATE TABLE vec_ejecucion_documental_v4.consumo_capacidad (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    nonce text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    huella_capacidad_sha256 text NOT NULL UNIQUE,
    capacidad jsonb NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    PRIMARY KEY (clave_id, version, nonce),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_ejecucion_documental_v4.clave_capacidad_version(
            clave_id, version
        ),
    FOREIGN KEY (decision_ref)
        REFERENCES vec_ejecucion_documental_v4.consumo_decision_atomico(
            decision_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    CHECK (nonce ~ '^[0-9a-f]{64}$'),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_capacidad_sha256
    )),
    CHECK (expira_en > emitida_en AND expira_en <= emitida_en + interval '15 seconds'),
    CHECK (consumida_en >= emitida_en AND consumida_en < expira_en)
);

CREATE TABLE vec_ejecucion_documental_v4.control_cadena_auditoria (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia numeric(20, 0) NOT NULL,
    ultima_huella_sha256 text NOT NULL,
    CHECK (ultima_secuencia BETWEEN 0 AND 18446744073709551615),
    CHECK ((ultima_secuencia = 0 AND ultima_huella_sha256 = repeat('0', 64))
        OR (ultima_secuencia > 0 AND
            vec_ejecucion_documental_v4.huella_sha256_valida(
                ultima_huella_sha256
            )))
);
INSERT INTO vec_ejecucion_documental_v4.control_cadena_auditoria
    (control_id, ultima_secuencia, ultima_huella_sha256)
VALUES (true, 0, repeat('0', 64));

CREATE TABLE vec_ejecucion_documental_v4.auditoria (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(decision_ref),
    orden_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(orden_ref),
    contexto_actor_huella_sha256 text NOT NULL,
    accion text NOT NULL,
    resultado text NOT NULL,
    correlacion_ref text NOT NULL,
    ocurrida_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(auditoria_ref, 512)),
    CHECK (accion = 'crear_orden_generacion_documental'
        AND resultado = 'pendiente_generacion'),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(
        contexto_actor_huella_sha256
    ) AND vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_anterior_sha256
    ) AND vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_registro_sha256
    ))
);

CREATE TABLE vec_ejecucion_documental_v4.evento_outbox (
    evento_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo text NOT NULL,
    estado text NOT NULL,
    decision_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(decision_ref),
    orden_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(orden_ref),
    auditoria_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.auditoria(auditoria_ref),
    huella_auditoria_sha256 text NOT NULL,
    correlacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (tipo = 'orden_generacion_documental_creada' AND estado = 'pendiente'),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(evento_ref, 512)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_auditoria_sha256
    ) AND vec_ejecucion_documental_v4.huella_sha256_valida(
        huella_registro_sha256
    ))
);

CREATE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.validar_actual_configuracion()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    anterior record;
    nueva record;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el puntero de configuracion no se puede eliminar';
    END IF;
    SELECT publicada_en, estado, huella_configuracion_sha256
      INTO nueva
      FROM vec_ejecucion_documental_v4.configuracion_confianza_version
     WHERE revision = NEW.revision
       AND huella_configuracion_sha256 = NEW.huella_configuracion_sha256
     FOR SHARE;
    IF NOT FOUND OR NEW.actualizada_en < nueva.publicada_en THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destino de configuracion actual invalido';
    END IF;
    IF nueva.estado = 'activa' AND EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.configuracion_confianza_version
         WHERE revision = NEW.revision
           AND estado = 'revocada'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'resurreccion de revision de configuracion rechazada';
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF NEW.control_id IS DISTINCT FROM OLD.control_id
           OR NEW.actualizada_en <= OLD.actualizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'transicion de configuracion no monotona';
        END IF;
        SELECT publicada_en, estado, huella_configuracion_sha256
          INTO anterior
          FROM vec_ejecucion_documental_v4.configuracion_confianza_version
         WHERE revision = OLD.revision
           AND huella_configuracion_sha256 = OLD.huella_configuracion_sha256
         FOR SHARE;
        IF NOT FOUND OR nueva.publicada_en <= anterior.publicada_en THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'retroceso de configuracion rechazado';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.validar_actual_raiz()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    anterior record;
    nueva record;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el puntero de raiz no se puede eliminar';
    END IF;
    SELECT version, valida_desde, estado, huella_clave_sha256
      INTO nueva
      FROM vec_ejecucion_documental_v4.raiz_confianza_version
     WHERE clave_id = NEW.clave_id AND version = NEW.version
       AND revision_configuracion = NEW.revision_configuracion
       AND huella_configuracion_sha256 = NEW.huella_configuracion_sha256
     FOR SHARE;
    IF NOT FOUND OR NEW.actualizada_en < nueva.valida_desde THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destino de raiz actual invalido';
    END IF;
    IF nueva.estado = 'activa' AND EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.raiz_confianza_version
         WHERE huella_clave_sha256 = nueva.huella_clave_sha256
           AND estado = 'revocada'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'resurreccion de raiz rechazada';
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF NEW.clave_id IS DISTINCT FROM OLD.clave_id
           OR NEW.version <= OLD.version
           OR NEW.actualizada_en <= OLD.actualizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'transicion de raiz no monotona';
        END IF;
        SELECT version, valida_desde, estado, huella_clave_sha256
          INTO anterior
          FROM vec_ejecucion_documental_v4.raiz_confianza_version
         WHERE clave_id = OLD.clave_id AND version = OLD.version
           AND revision_configuracion = OLD.revision_configuracion
           AND huella_configuracion_sha256 = OLD.huella_configuracion_sha256
         FOR SHARE;
        IF NOT FOUND OR nueva.valida_desde < anterior.valida_desde THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'retroceso temporal de raiz rechazado';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_ejecucion_documental_v4.validar_actual_clave_capacidad()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    anterior record;
    nueva record;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el puntero de capacidad no se puede eliminar';
    END IF;
    SELECT version, valida_desde, estado, huella_secreto_sha256
      INTO nueva
      FROM vec_ejecucion_documental_v4.clave_capacidad_version
     WHERE clave_id = NEW.clave_id AND version = NEW.version
     FOR SHARE;
    IF NOT FOUND OR NEW.actualizada_en < nueva.valida_desde THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'destino de clave de capacidad invalido';
    END IF;
    IF nueva.estado = 'activa' AND EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.clave_capacidad_version
         WHERE huella_secreto_sha256 = nueva.huella_secreto_sha256
           AND estado = 'revocada'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'resurreccion de clave de capacidad rechazada';
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF NEW.control_id IS DISTINCT FROM OLD.control_id
           OR NEW.version <= OLD.version
           OR NEW.actualizada_en <= OLD.actualizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'transicion de clave de capacidad no monotona';
        END IF;
        SELECT version, valida_desde, estado, huella_secreto_sha256
          INTO anterior
          FROM vec_ejecucion_documental_v4.clave_capacidad_version
         WHERE clave_id = OLD.clave_id AND version = OLD.version
         FOR SHARE;
        IF NOT FOUND OR nueva.valida_desde < anterior.valida_desde THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'retroceso temporal de clave de capacidad rechazado';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER configuracion_confianza_actual_transicion
BEFORE INSERT OR UPDATE OR DELETE
ON vec_ejecucion_documental_v4.configuracion_confianza_actual
FOR EACH ROW EXECUTE FUNCTION
    vec_ejecucion_documental_v4.validar_actual_configuracion();
CREATE TRIGGER configuracion_confianza_actual_no_truncar
BEFORE TRUNCATE ON vec_ejecucion_documental_v4.configuracion_confianza_actual
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_ejecucion_documental_v4.rechazar_mutacion_inmutable();

CREATE TRIGGER raiz_confianza_actual_transicion
BEFORE INSERT OR UPDATE OR DELETE
ON vec_ejecucion_documental_v4.raiz_confianza_actual
FOR EACH ROW EXECUTE FUNCTION
    vec_ejecucion_documental_v4.validar_actual_raiz();
CREATE TRIGGER raiz_confianza_actual_no_truncar
BEFORE TRUNCATE ON vec_ejecucion_documental_v4.raiz_confianza_actual
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_ejecucion_documental_v4.rechazar_mutacion_inmutable();

CREATE TRIGGER clave_capacidad_actual_transicion
BEFORE INSERT OR UPDATE OR DELETE
ON vec_ejecucion_documental_v4.clave_capacidad_actual
FOR EACH ROW EXECUTE FUNCTION
    vec_ejecucion_documental_v4.validar_actual_clave_capacidad();
CREATE TRIGGER clave_capacidad_actual_no_truncar
BEFORE TRUNCATE ON vec_ejecucion_documental_v4.clave_capacidad_actual
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_ejecucion_documental_v4.rechazar_mutacion_inmutable();

DO $triggers$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'configuracion_confianza_version', 'raiz_confianza_version',
        'clave_capacidad_version',
        'atestacion_pdp', 'orden_generacion_documental',
        'consumo_decision_atomico', 'consumo_capacidad',
        'auditoria', 'evento_outbox'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_ejecucion_documental_v4.%I FOR EACH ROW EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_ejecucion_documental_v4.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
END
$triggers$;

DO $rls$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'configuracion_confianza_version', 'configuracion_confianza_actual',
        'raiz_confianza_version', 'raiz_confianza_actual',
        'clave_capacidad_version', 'clave_capacidad_actual', 'atestacion_pdp',
        'orden_generacion_documental', 'consumo_decision_atomico',
        'consumo_capacidad', 'control_cadena_auditoria',
        'auditoria', 'evento_outbox'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_ejecucion_documental_v4.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_ejecucion_documental_v4.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_ejecucion_documental_v4.%I FOR ALL TO vec_ejecucion_documental_v4_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_ejecucion_documental_v4_propietario',
            'vec_ejecucion_documental_v4_propietario'
        );
    END LOOP;
END
$rls$;

CREATE FUNCTION vec_ejecucion_documental_v4.ejecutar_plan_atestado(
    p_metadatos bytea,
    p_payload bytea,
    p_sobre bytea,
    p_evidencia bytea,
    p_preimagen bytea,
    p_decision_canonica bytea,
    p_efecto bytea,
    p_capacidad jsonb
)
RETURNS TABLE (
    resultado text,
    orden_ref text,
    estado_orden text,
    auditoria_ref text,
    evento_outbox_ref text,
    registrada_en timestamptz(6)
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    metadatos jsonb;
    efecto jsonb;
    aplicacion jsonb;
    configuracion record;
    raiz record;
    clave_capacidad record;
    atestacion_existente record;
    consumo_existente record;
    instante timestamptz(6);
    base_orden bytea;
    huella_orden text;
    auditoria_esperada text;
    evento_esperado text;
    secuencia numeric(20, 0);
    huella_anterior text;
    huella_auditoria text;
    huella_evento text;
BEGIN
    resultado := 'rechazada'; orden_ref := ''; estado_orden := '';
    auditoria_ref := ''; evento_outbox_ref := ''; registrada_en := NULL;
    -- Antes de interpretar un solo byte JSON de los artefactos se autentican
    -- sus huellas exactas y la capacidad de vida corta que las liga.
    IF p_capacidad IS NULL OR jsonb_typeof(p_capacidad) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_capacidad)) <> 20
       OR NOT (p_capacidad ?& ARRAY[
           'esquema', 'clave_id', 'clave_version', 'emisor_id', 'audiencia',
           'nonce', 'emitida_en', 'expira_en', 'huella_metadatos_sha256',
           'huella_payload_sha256', 'huella_sobre_sha256',
           'huella_evidencia_sha256', 'huella_preimagen_sha256',
           'huella_decision_sha256', 'huella_efecto_sha256',
           'revision_confianza', 'huella_configuracion_sha256',
           'raiz_clave_id', 'huella_raiz_sha256', 'mac_sha256'
       ]) OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'esquema', 'clave_id', 'emisor_id', 'audiencia', 'nonce',
               'emitida_en', 'expira_en', 'huella_metadatos_sha256',
               'huella_payload_sha256', 'huella_sobre_sha256',
               'huella_evidencia_sha256', 'huella_preimagen_sha256',
               'huella_decision_sha256', 'huella_efecto_sha256',
               'revision_confianza', 'huella_configuracion_sha256',
               'raiz_clave_id', 'huella_raiz_sha256', 'mac_sha256'
           ]) AS clave
           WHERE jsonb_typeof(p_capacidad -> clave) <> 'string'
       ) OR jsonb_typeof(p_capacidad -> 'clave_version') <> 'number'
       OR p_capacidad ->> 'esquema' <>
          'vec.documentos.capacidad-ejecucion.v4'
       OR p_capacidad ->> 'audiencia' <>
          'vec_ejecucion_documental_v4.ejecutar_plan_atestado'
       OR p_capacidad ->> 'nonce' !~ '^[0-9a-f]{64}$'
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
              p_capacidad ->> 'clave_id', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
              p_capacidad ->> 'emisor_id', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
              p_capacidad ->> 'revision_confianza', 128
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
              p_capacidad ->> 'raiz_clave_id', 512
          ) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_metadatos_sha256', 'huella_payload_sha256',
               'huella_sobre_sha256', 'huella_evidencia_sha256',
               'huella_preimagen_sha256', 'huella_decision_sha256',
               'huella_efecto_sha256', 'huella_configuracion_sha256',
               'huella_raiz_sha256', 'mac_sha256'
           ]) AS clave
           WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_capacidad ->> clave
           ) IS NOT TRUE
       ) OR vec_ejecucion_documental_v4.instante_valido(
              p_capacidad ->> 'emitida_en'
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.instante_valido(
              p_capacidad ->> 'expira_en'
          ) IS NOT TRUE
       OR (p_capacidad ->> 'clave_version')::numeric <> trunc(
              (p_capacidad ->> 'clave_version')::numeric
          )
       OR (p_capacidad ->> 'clave_version')::numeric NOT BETWEEN
              1 AND 18446744073709551615
       OR octet_length(p_metadatos) NOT BETWEEN 2 AND 524288
       OR octet_length(p_payload) NOT BETWEEN 1 AND 524288
       OR octet_length(p_sobre) NOT BETWEEN 16 AND 528384
       OR octet_length(p_evidencia) NOT BETWEEN 1 AND 2097152
       OR octet_length(p_preimagen) NOT BETWEEN 1 AND 2097152
       OR octet_length(p_decision_canonica) NOT BETWEEN 128 AND 524288
       OR octet_length(p_efecto) NOT BETWEEN 2 AND 65536
       OR encode(sha256(p_metadatos), 'hex') <>
          p_capacidad ->> 'huella_metadatos_sha256'
       OR encode(sha256(p_payload), 'hex') <>
          p_capacidad ->> 'huella_payload_sha256'
       OR encode(sha256(p_sobre), 'hex') <>
          p_capacidad ->> 'huella_sobre_sha256'
       OR encode(sha256(p_evidencia), 'hex') <>
          p_capacidad ->> 'huella_evidencia_sha256'
       OR encode(sha256(p_preimagen), 'hex') <>
          p_capacidad ->> 'huella_preimagen_sha256'
       OR encode(sha256(p_decision_canonica), 'hex') <>
          p_capacidad ->> 'huella_decision_sha256'
       OR encode(sha256(p_efecto), 'hex') <>
          p_capacidad ->> 'huella_efecto_sha256' THEN
        RETURN NEXT; RETURN;
    END IF;

    instante := clock_timestamp();
    IF (p_capacidad ->> 'expira_en')::timestamptz <=
           (p_capacidad ->> 'emitida_en')::timestamptz
       OR (p_capacidad ->> 'expira_en')::timestamptz >
          (p_capacidad ->> 'emitida_en')::timestamptz + interval '15 seconds'
       OR instante < (p_capacidad ->> 'emitida_en')::timestamptz
       OR instante >= (p_capacidad ->> 'expira_en')::timestamptz THEN
        RETURN NEXT; RETURN;
    END IF;

    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en, version.estado,
           version.revocada_en
      INTO configuracion
      FROM vec_ejecucion_documental_v4.configuracion_confianza_actual AS actual
      JOIN vec_ejecucion_documental_v4.configuracion_confianza_version AS version
        ON version.revision = actual.revision
       AND version.huella_configuracion_sha256 =
           actual.huella_configuracion_sha256
     WHERE actual.control_id = true
     FOR SHARE OF actual, version;
    IF NOT FOUND OR configuracion.estado <> 'activa'
       OR configuracion.revocada_en IS NOT NULL
       OR instante < configuracion.publicada_en
       OR instante >= configuracion.expira_en
       OR configuracion.revision IS DISTINCT FROM
          p_capacidad ->> 'revision_confianza'
       OR configuracion.huella_configuracion_sha256 IS DISTINCT FROM
          p_capacidad ->> 'huella_configuracion_sha256' THEN
        RETURN NEXT; RETURN;
    END IF;

    SELECT version.clave_id, version.version, version.revision_configuracion,
           version.huella_configuracion_sha256, version.algoritmo_cose,
           version.suite, version.audiencia_cose, version.audiencia_despliegue,
           version.huella_clave_sha256, version.valida_desde,
           version.valida_hasta, version.estado, version.revocada_en
      INTO raiz
      FROM vec_ejecucion_documental_v4.raiz_confianza_actual AS actual
      JOIN vec_ejecucion_documental_v4.raiz_confianza_version AS version
        ON version.clave_id = actual.clave_id
       AND version.version = actual.version
       AND version.revision_configuracion = actual.revision_configuracion
       AND version.huella_configuracion_sha256 =
           actual.huella_configuracion_sha256
     WHERE actual.clave_id = p_capacidad ->> 'raiz_clave_id'
     FOR SHARE OF actual, version;
    IF NOT FOUND OR raiz.estado <> 'activa' OR raiz.revocada_en IS NOT NULL
       OR instante < raiz.valida_desde OR instante >= raiz.valida_hasta
       OR raiz.revision_configuracion IS DISTINCT FROM configuracion.revision
       OR raiz.huella_configuracion_sha256 IS DISTINCT FROM
          configuracion.huella_configuracion_sha256
       OR raiz.huella_clave_sha256 IS DISTINCT FROM
          p_capacidad ->> 'huella_raiz_sha256' THEN
        RETURN NEXT; RETURN;
    END IF;

    SELECT version.clave_id, version.version, version.secreto_hmac,
           version.huella_secreto_sha256, version.emisor_id,
           version.valida_desde, version.valida_hasta, version.estado,
           version.revocada_en
      INTO clave_capacidad
      FROM vec_ejecucion_documental_v4.clave_capacidad_actual AS actual
      JOIN vec_ejecucion_documental_v4.clave_capacidad_version AS version
        ON version.clave_id = actual.clave_id
       AND version.version = actual.version
     WHERE actual.control_id = true
     FOR SHARE OF actual, version;
    IF NOT FOUND OR clave_capacidad.estado <> 'activa'
       OR clave_capacidad.revocada_en IS NOT NULL
       OR instante < clave_capacidad.valida_desde
       OR instante >= clave_capacidad.valida_hasta
       OR clave_capacidad.clave_id IS DISTINCT FROM
          p_capacidad ->> 'clave_id'
       OR clave_capacidad.version IS DISTINCT FROM
          (p_capacidad ->> 'clave_version')::numeric
       OR clave_capacidad.emisor_id IS DISTINCT FROM
          p_capacidad ->> 'emisor_id'
       OR (p_capacidad ->> 'emitida_en')::timestamptz <
          clave_capacidad.valida_desde
       OR (p_capacidad ->> 'expira_en')::timestamptz >
          clave_capacidad.valida_hasta
       OR vec_ejecucion_documental_v4.bytea_igual_constante(
              public.hmac(
                  vec_ejecucion_documental_v4.preimagen_capacidad(p_capacidad),
                  clave_capacidad.secreto_hmac,
                  'sha256'
              ),
              decode(p_capacidad ->> 'mac_sha256', 'hex')
          ) IS NOT TRUE THEN
        RETURN NEXT; RETURN;
    END IF;

    metadatos := convert_from(p_metadatos, 'UTF8')::jsonb;
    efecto := convert_from(p_efecto, 'UTF8')::jsonb;
    IF metadatos IS NULL OR jsonb_typeof(metadatos) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(metadatos)) <> 20
       OR NOT (metadatos ?& ARRAY[
           'aplicacion', 'huella_preimagen_sha256', 'formato_vec_ad_version',
           'suite', 'clave_id', 'audiencia_despliegue', 'algoritmo_cose',
           'audiencia_cose', 'estado_confianza', 'huella_clave_sha256',
           'huella_payload_sha256', 'huella_sobre_sha256', 'verificada_en',
           'raiz_valida_desde', 'raiz_valida_hasta', 'revision_confianza',
           'huella_configuracion_sha256', 'configuracion_publicada_en',
           'configuracion_expira_en', 'huella_evidencia_sha256'
       ]) OR efecto IS NULL OR jsonb_typeof(efecto) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(efecto)) <> 13
       OR NOT (efecto ?& ARRAY[
           'esquema', 'orden_ref', 'estado', 'decision_ref', 'efecto_ref',
           'huella_plan_sha256', 'huella_decision_sha256',
           'huella_aplicacion_sha256', 'huella_orden_sha256', 'auditoria_ref',
           'evento_outbox_ref', 'correlacion_ref', 'solicitada_en'
       ]) THEN
        RETURN NEXT; RETURN;
    END IF;
    aplicacion := metadatos -> 'aplicacion';
    IF vec_ejecucion_documental_v4.aplicacion_valida(aplicacion) IS NOT TRUE
       OR (metadatos ->> 'formato_vec_ad_version')::integer <> 1
       OR metadatos ->> 'suite' <> 'VEC-AD-COSE-EDDSA-1'
       OR metadatos ->> 'algoritmo_cose' <> 'EdDSA'
       OR metadatos ->> 'audiencia_cose' <> 'atestacion_autorizacion_pdp'
       OR metadatos ->> 'estado_confianza' <> 'activa'
       OR encode(sha256(p_payload), 'hex') <>
          metadatos ->> 'huella_payload_sha256'
       OR encode(sha256(p_sobre), 'hex') <>
          metadatos ->> 'huella_sobre_sha256'
       OR encode(sha256(p_evidencia), 'hex') <>
          metadatos ->> 'huella_evidencia_sha256'
       OR encode(sha256(p_preimagen), 'hex') <>
          metadatos ->> 'huella_preimagen_sha256'
       OR encode(sha256(p_decision_canonica), 'hex') <>
          aplicacion ->> 'huella_decision_sha256'
       OR configuracion.revision IS DISTINCT FROM
          metadatos ->> 'revision_confianza'
       OR configuracion.huella_configuracion_sha256 IS DISTINCT FROM
          metadatos ->> 'huella_configuracion_sha256'
       OR configuracion.publicada_en IS DISTINCT FROM
          (metadatos ->> 'configuracion_publicada_en')::timestamptz
       OR configuracion.expira_en IS DISTINCT FROM
          (metadatos ->> 'configuracion_expira_en')::timestamptz
       OR raiz.clave_id IS DISTINCT FROM metadatos ->> 'clave_id'
       OR raiz.algoritmo_cose IS DISTINCT FROM metadatos ->> 'algoritmo_cose'
       OR raiz.suite IS DISTINCT FROM metadatos ->> 'suite'
       OR raiz.audiencia_cose IS DISTINCT FROM metadatos ->> 'audiencia_cose'
       OR raiz.audiencia_despliegue IS DISTINCT FROM
          metadatos ->> 'audiencia_despliegue'
       OR raiz.huella_clave_sha256 IS DISTINCT FROM
          metadatos ->> 'huella_clave_sha256'
       OR raiz.valida_desde IS DISTINCT FROM
          (metadatos ->> 'raiz_valida_desde')::timestamptz
       OR raiz.valida_hasta IS DISTINCT FROM
          (metadatos ->> 'raiz_valida_hasta')::timestamptz
       OR (metadatos ->> 'verificada_en')::timestamptz < raiz.valida_desde
       OR (metadatos ->> 'verificada_en')::timestamptz >= raiz.valida_hasta
       OR instante >= (aplicacion ->> 'valida_hasta')::timestamptz THEN
        RETURN NEXT; RETURN;
    END IF;

    IF vec_autorizacion.revalidar_decision_ejecucion_documental_v4(
        aplicacion, p_decision_canonica
    ) IS NOT TRUE THEN
        RETURN NEXT; RETURN;
    END IF;

    IF efecto ->> 'esquema' <> 'vec.documentos.orden-generacion.v4'
       OR efecto ->> 'estado' <> 'pendiente_generacion'
       OR efecto ->> 'orden_ref' <> aplicacion ->> 'efecto_ref'
       OR efecto ->> 'efecto_ref' <> aplicacion ->> 'efecto_ref'
       OR efecto ->> 'decision_ref' <> aplicacion ->> 'decision_ref'
       OR efecto ->> 'huella_plan_sha256' <>
          aplicacion ->> 'huella_plan_sha256'
       OR efecto ->> 'huella_decision_sha256' <>
          aplicacion ->> 'huella_decision_sha256'
       OR efecto ->> 'huella_aplicacion_sha256' <>
          aplicacion ->> 'huella_solicitud_aplicacion_sha256'
       OR efecto ->> 'correlacion_ref' <> aplicacion ->> 'correlacion_ref'
       OR (efecto ->> 'solicitada_en')::timestamptz IS DISTINCT FROM
          (aplicacion ->> 'solicitada_en')::timestamptz THEN
        RETURN NEXT; RETURN;
    END IF;
    base_orden := convert_to(aplicacion ->> 'decision_ref', 'UTF8') ||
        decode('00', 'hex') || convert_to(aplicacion ->> 'efecto_ref', 'UTF8') ||
        decode('00', 'hex') || convert_to(aplicacion ->> 'huella_plan_sha256', 'UTF8') ||
        decode('00', 'hex') || convert_to(aplicacion ->> 'huella_decision_sha256', 'UTF8') ||
        decode('00', 'hex') || convert_to(
            aplicacion ->> 'huella_solicitud_aplicacion_sha256', 'UTF8'
        );
    huella_orden := encode(sha256(
        convert_to('vec.documentos.orden-generacion.v4', 'UTF8') ||
        decode('00', 'hex') || base_orden
    ), 'hex');
    auditoria_esperada := 'auditoria:documental:v4:' || encode(sha256(
        convert_to('auditoria', 'UTF8') || decode('00', 'hex') || base_orden
    ), 'hex');
    evento_esperado := 'evento:documental:v4:' || encode(sha256(
        convert_to('outbox', 'UTF8') || decode('00', 'hex') || base_orden
    ), 'hex');
    IF efecto ->> 'huella_orden_sha256' <> huella_orden
       OR efecto ->> 'auditoria_ref' <> auditoria_esperada
       OR efecto ->> 'evento_outbox_ref' <> evento_esperado THEN
        RETURN NEXT; RETURN;
    END IF;

    -- La revalidacion de autorizacion puede haber esperado bloqueos ajenos.
    -- Se vuelve a tomar el reloj justo antes del consumo y del efecto para que
    -- una capacidad que caduco durante esa espera no autorice nada.
    instante := clock_timestamp();
    IF instante >= (p_capacidad ->> 'expira_en')::timestamptz
       OR instante >= clave_capacidad.valida_hasta
       OR instante >= configuracion.expira_en
       OR instante >= raiz.valida_hasta
       OR instante >= (aplicacion ->> 'valida_hasta')::timestamptz THEN
        RETURN NEXT; RETURN;
    END IF;

    SELECT * INTO consumo_existente
      FROM vec_ejecucion_documental_v4.consumo_decision_atomico
     WHERE decision_ref = aplicacion ->> 'decision_ref'
        OR efecto_ref = aplicacion ->> 'efecto_ref'
     FOR SHARE;
    IF FOUND THEN
        resultado := 'duplicada'; RETURN NEXT; RETURN;
    END IF;

    INSERT INTO vec_ejecucion_documental_v4.atestacion_pdp (
        decision_ref, huella_decision_sha256, huella_plan_sha256, efecto_ref,
        huella_solicitud_vinculada_sha256, aplicacion_registro,
        huella_preimagen_sha256, preimagen_recurso, formato_vec_ad_version,
        suite, clave_id, version_raiz, audiencia_despliegue, algoritmo_cose,
        audiencia_cose, huella_clave_sha256, huella_payload_sha256,
        huella_sobre_sha256, verificada_en, raiz_valida_desde,
        raiz_valida_hasta, revision_confianza,
        huella_configuracion_sha256, configuracion_publicada_en,
        configuracion_expira_en, huella_evidencia_sha256, payload_vec_ad_1,
        sobre_cose_sign1, evidencia_canonica, decision_canonica, registrada_en
    ) VALUES (
        aplicacion ->> 'decision_ref', aplicacion ->> 'huella_decision_sha256',
        aplicacion ->> 'huella_plan_sha256', aplicacion ->> 'efecto_ref',
        aplicacion ->> 'huella_solicitud_vinculada_sha256', aplicacion,
        metadatos ->> 'huella_preimagen_sha256', p_preimagen,
        (metadatos ->> 'formato_vec_ad_version')::integer,
        metadatos ->> 'suite', metadatos ->> 'clave_id', raiz.version,
        metadatos ->> 'audiencia_despliegue',
        metadatos ->> 'algoritmo_cose', metadatos ->> 'audiencia_cose',
        metadatos ->> 'huella_clave_sha256',
        metadatos ->> 'huella_payload_sha256',
        metadatos ->> 'huella_sobre_sha256',
        (metadatos ->> 'verificada_en')::timestamptz,
        (metadatos ->> 'raiz_valida_desde')::timestamptz,
        (metadatos ->> 'raiz_valida_hasta')::timestamptz,
        metadatos ->> 'revision_confianza',
        metadatos ->> 'huella_configuracion_sha256',
        (metadatos ->> 'configuracion_publicada_en')::timestamptz,
        (metadatos ->> 'configuracion_expira_en')::timestamptz,
        metadatos ->> 'huella_evidencia_sha256', p_payload, p_sobre,
        p_evidencia, p_decision_canonica, instante
    ) ON CONFLICT (decision_ref) DO NOTHING;
    SELECT * INTO atestacion_existente
      FROM vec_ejecucion_documental_v4.atestacion_pdp
     WHERE decision_ref = aplicacion ->> 'decision_ref'
     FOR SHARE;
    IF NOT FOUND OR atestacion_existente.aplicacion_registro IS DISTINCT FROM aplicacion
       OR atestacion_existente.decision_canonica IS DISTINCT FROM p_decision_canonica
       OR atestacion_existente.sobre_cose_sign1 IS DISTINCT FROM p_sobre
       OR atestacion_existente.evidencia_canonica IS DISTINCT FROM p_evidencia THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'atestacion preexistente incompatible';
    END IF;

    INSERT INTO vec_ejecucion_documental_v4.consumo_capacidad (
        clave_id, version, nonce, decision_ref, huella_capacidad_sha256,
        capacidad, emitida_en, expira_en, consumida_en
    ) VALUES (
        p_capacidad ->> 'clave_id',
        (p_capacidad ->> 'clave_version')::numeric,
        p_capacidad ->> 'nonce', aplicacion ->> 'decision_ref',
        encode(sha256(convert_to(p_capacidad::text, 'UTF8')), 'hex'),
        p_capacidad, (p_capacidad ->> 'emitida_en')::timestamptz,
        (p_capacidad ->> 'expira_en')::timestamptz, instante
    );

    INSERT INTO vec_ejecucion_documental_v4.orden_generacion_documental (
        orden_ref, estado, decision_ref, efecto_ref, huella_plan_sha256,
        huella_decision_sha256, huella_aplicacion_sha256,
        huella_orden_sha256, correlacion_ref, solicitada_en, registrada_en
    ) VALUES (
        efecto ->> 'orden_ref', efecto ->> 'estado',
        efecto ->> 'decision_ref', efecto ->> 'efecto_ref',
        efecto ->> 'huella_plan_sha256',
        efecto ->> 'huella_decision_sha256',
        efecto ->> 'huella_aplicacion_sha256',
        efecto ->> 'huella_orden_sha256', efecto ->> 'correlacion_ref',
        (efecto ->> 'solicitada_en')::timestamptz, instante
    );
    INSERT INTO vec_ejecucion_documental_v4.consumo_decision_atomico (
        decision_ref, efecto_ref, orden_ref, huella_decision_sha256,
        huella_aplicacion_sha256, consumida_en
    ) VALUES (
        aplicacion ->> 'decision_ref', aplicacion ->> 'efecto_ref',
        efecto ->> 'orden_ref', aplicacion ->> 'huella_decision_sha256',
        aplicacion ->> 'huella_solicitud_aplicacion_sha256', instante
    );

    SELECT ultima_secuencia + 1, ultima_huella_sha256
      INTO secuencia, huella_anterior
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria
     WHERE control_id = true FOR UPDATE;
    IF NOT FOUND OR secuencia > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'cadena de auditoria no disponible';
    END IF;
    huella_auditoria := encode(sha256(convert_to(concat_ws(E'\n',
        auditoria_esperada, secuencia::text,
        aplicacion ->> 'decision_ref', efecto ->> 'orden_ref',
        aplicacion ->> 'contexto_actor_huella_sha256',
        'crear_orden_generacion_documental', 'pendiente_generacion',
        aplicacion ->> 'correlacion_ref',
        to_char(instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        huella_anterior
    ), 'UTF8')), 'hex');
    huella_evento := encode(sha256(convert_to(concat_ws(E'\n',
        evento_esperado, secuencia::text,
        'orden_generacion_documental_creada', 'pendiente',
        aplicacion ->> 'decision_ref', efecto ->> 'orden_ref',
        auditoria_esperada, huella_auditoria,
        aplicacion ->> 'correlacion_ref',
        to_char(instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    ), 'UTF8')), 'hex');
    INSERT INTO vec_ejecucion_documental_v4.auditoria (
        auditoria_ref, secuencia, decision_ref, orden_ref,
        contexto_actor_huella_sha256, accion, resultado, correlacion_ref,
        ocurrida_en, huella_anterior_sha256, huella_registro_sha256
    ) VALUES (
        auditoria_esperada, secuencia, aplicacion ->> 'decision_ref',
        efecto ->> 'orden_ref', aplicacion ->> 'contexto_actor_huella_sha256',
        'crear_orden_generacion_documental', 'pendiente_generacion',
        aplicacion ->> 'correlacion_ref', instante, huella_anterior,
        huella_auditoria
    );
    INSERT INTO vec_ejecucion_documental_v4.evento_outbox (
        evento_ref, secuencia, tipo, estado, decision_ref, orden_ref,
        auditoria_ref, huella_auditoria_sha256, correlacion_ref,
        registrada_en, huella_registro_sha256
    ) VALUES (
        evento_esperado, secuencia, 'orden_generacion_documental_creada',
        'pendiente', aplicacion ->> 'decision_ref', efecto ->> 'orden_ref',
        auditoria_esperada, huella_auditoria,
        aplicacion ->> 'correlacion_ref', instante, huella_evento
    );
    UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
       SET ultima_secuencia = secuencia,
           ultima_huella_sha256 = huella_auditoria
     WHERE control_id = true AND ultima_secuencia = secuencia - 1
       AND ultima_huella_sha256 = huella_anterior;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de auditoria perdido';
    END IF;

    resultado := 'ejecutada'; orden_ref := efecto ->> 'orden_ref';
    estado_orden := 'pendiente_generacion';
    auditoria_ref := auditoria_esperada;
    evento_outbox_ref := evento_esperado; registrada_en := instante;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception OR invalid_text_representation OR datetime_field_overflow
        OR character_not_in_repertoire OR check_violation
        OR foreign_key_violation OR unique_violation THEN
        resultado := 'rechazada'; orden_ref := ''; estado_orden := '';
        auditoria_ref := ''; evento_outbox_ref := ''; registrada_en := NULL;
        RETURN NEXT;
END
$funcion$;

-- Lectura de arranque de solo datos publicos de confianza. El llamador no
-- propone claves ni revisiones: la funcion resuelve y bloquea los punteros
-- actuales gobernados por DBA. Una rotacion posterior provoca denegacion en
-- ejecutar_plan_atestado hasta que el proceso recargue la nueva lista.
CREATE FUNCTION vec_ejecucion_documental_v4.obtener_confianza_actual()
RETURNS TABLE (
    revision text,
    huella_configuracion_sha256 text,
    configuracion_publicada_en timestamptz(6),
    configuracion_expira_en timestamptz(6),
    configuracion_estado text,
    configuracion_revocada_en timestamptz(6),
    clave_id text,
    algoritmo_cose text,
    suite text,
    audiencia_cose text,
    audiencia_despliegue text,
    clave_publica_spki bytea,
    huella_clave_sha256 text,
    raiz_valida_desde timestamptz(6),
    raiz_valida_hasta timestamptz(6),
    raiz_estado text,
    raiz_revocada_en timestamptz(6)
)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT configuracion.revision,
           configuracion.huella_configuracion_sha256,
           configuracion.publicada_en, configuracion.expira_en,
           configuracion.estado, configuracion.revocada_en,
           raiz.clave_id, raiz.algoritmo_cose, raiz.suite,
           raiz.audiencia_cose, raiz.audiencia_despliegue,
           raiz.clave_publica_spki, raiz.huella_clave_sha256,
           raiz.valida_desde, raiz.valida_hasta, raiz.estado, raiz.revocada_en
      FROM vec_ejecucion_documental_v4.configuracion_confianza_actual AS actual_config
      JOIN vec_ejecucion_documental_v4.configuracion_confianza_version AS configuracion
        ON configuracion.revision = actual_config.revision
       AND configuracion.huella_configuracion_sha256 =
           actual_config.huella_configuracion_sha256
      JOIN vec_ejecucion_documental_v4.raiz_confianza_actual AS actual_raiz
        ON actual_raiz.revision_configuracion = configuracion.revision
       AND actual_raiz.huella_configuracion_sha256 =
           configuracion.huella_configuracion_sha256
      JOIN vec_ejecucion_documental_v4.raiz_confianza_version AS raiz
        ON raiz.clave_id = actual_raiz.clave_id
       AND raiz.version = actual_raiz.version
       AND raiz.revision_configuracion = actual_raiz.revision_configuracion
       AND raiz.huella_configuracion_sha256 =
           actual_raiz.huella_configuracion_sha256
     WHERE actual_config.control_id = true
       AND configuracion.estado = 'activa'
       AND configuracion.revocada_en IS NULL
       AND raiz.estado = 'activa'
       AND raiz.revocada_en IS NULL
     ORDER BY raiz.clave_id
     FOR SHARE OF actual_config, configuracion, actual_raiz, raiz
$funcion$;

-- Unica salida del secreto HMAC. Debe usarse con una cuenta LOGIN dedicada
-- que no sea miembro del rol ejecutor. Se bloquean en una sola instantanea la
-- configuracion, cada raiz y la clave de capacidad actuales.
CREATE FUNCTION vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()
RETURNS TABLE (
    revision text,
    huella_configuracion_sha256 text,
    configuracion_publicada_en timestamptz(6),
    configuracion_expira_en timestamptz(6),
    configuracion_estado text,
    configuracion_revocada_en timestamptz(6),
    clave_id text,
    algoritmo_cose text,
    suite text,
    audiencia_cose text,
    audiencia_despliegue text,
    clave_publica_spki bytea,
    huella_clave_sha256 text,
    raiz_valida_desde timestamptz(6),
    raiz_valida_hasta timestamptz(6),
    raiz_estado text,
    raiz_revocada_en timestamptz(6),
    capacidad_clave_id text,
    capacidad_clave_version numeric(20, 0),
    capacidad_secreto bytea,
    capacidad_emisor_id text,
    capacidad_valida_desde timestamptz(6),
    capacidad_valida_hasta timestamptz(6),
    capacidad_estado text,
    capacidad_revocada_en timestamptz(6)
)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT configuracion.revision,
           configuracion.huella_configuracion_sha256,
           configuracion.publicada_en, configuracion.expira_en,
           configuracion.estado, configuracion.revocada_en,
           raiz.clave_id, raiz.algoritmo_cose, raiz.suite,
           raiz.audiencia_cose, raiz.audiencia_despliegue,
           raiz.clave_publica_spki, raiz.huella_clave_sha256,
           raiz.valida_desde, raiz.valida_hasta, raiz.estado, raiz.revocada_en,
           clave.clave_id, clave.version, clave.secreto_hmac,
           clave.emisor_id, clave.valida_desde, clave.valida_hasta,
           clave.estado, clave.revocada_en
      FROM vec_ejecucion_documental_v4.configuracion_confianza_actual AS actual_config
      JOIN vec_ejecucion_documental_v4.configuracion_confianza_version AS configuracion
        ON configuracion.revision = actual_config.revision
       AND configuracion.huella_configuracion_sha256 =
           actual_config.huella_configuracion_sha256
      JOIN vec_ejecucion_documental_v4.raiz_confianza_actual AS actual_raiz
        ON actual_raiz.revision_configuracion = configuracion.revision
       AND actual_raiz.huella_configuracion_sha256 =
           configuracion.huella_configuracion_sha256
      JOIN vec_ejecucion_documental_v4.raiz_confianza_version AS raiz
        ON raiz.clave_id = actual_raiz.clave_id
       AND raiz.version = actual_raiz.version
       AND raiz.revision_configuracion = actual_raiz.revision_configuracion
       AND raiz.huella_configuracion_sha256 =
           actual_raiz.huella_configuracion_sha256
      JOIN vec_ejecucion_documental_v4.clave_capacidad_actual AS actual_clave
        ON actual_clave.control_id = true
      JOIN vec_ejecucion_documental_v4.clave_capacidad_version AS clave
        ON clave.clave_id = actual_clave.clave_id
       AND clave.version = actual_clave.version
     WHERE actual_config.control_id = true
       AND configuracion.estado = 'activa'
       AND configuracion.revocada_en IS NULL
       AND raiz.estado = 'activa'
       AND raiz.revocada_en IS NULL
       AND clave.estado = 'activa'
       AND clave.revocada_en IS NULL
     ORDER BY raiz.clave_id
     FOR SHARE OF actual_config, configuracion, actual_raiz, raiz,
                  actual_clave, clave
$funcion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
-- ALTER DEFAULT PRIVILEGES ON TYPES no se aplica a los tipos fila creados de
-- forma implicita por CREATE TABLE. Se cierran expresamente los tipos base; la
-- ACL efectiva de sus arrays queda ligada a la del elemento en PostgreSQL.
DO $cerrar_tipos_fila$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC',
            tipo.nspname,
            tipo.typname
        );
    END LOOP;
END
$cerrar_tipos_fila$;
GRANT USAGE ON SCHEMA vec_ejecucion_documental_v4
    TO vec_ejecucion_documental_v4_emisor_capacidad;
GRANT USAGE ON SCHEMA vec_ejecucion_documental_v4
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.ejecutar_plan_atestado(
    bytea, bytea, bytea, bytea, bytea, bytea, bytea, jsonb
) TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION
    vec_ejecucion_documental_v4.obtener_confianza_actual()
    TO vec_ejecucion_documental_v4_emisor_capacidad;
GRANT EXECUTE ON FUNCTION
    vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()
    TO vec_ejecucion_documental_v4_emisor_capacidad;
COMMIT;
