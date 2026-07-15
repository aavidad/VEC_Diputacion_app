-- Persistencia durable de la unidad atomica BaremacionMerito. Se instala
-- despues de la frontera vec_autorizacion/revalidar_decision_bolsa_baremacion_v1.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE SCHEMA vec_bolsa_baremacion
    AUTHORIZATION vec_bolsa_baremacion_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_baremacion FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_baremacion_propietario
    IN SCHEMA vec_bolsa_baremacion REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_baremacion_propietario
    IN SCHEMA vec_bolsa_baremacion REVOKE ALL ON SEQUENCES FROM PUBLIC;
-- Los privilegios globales predeterminados de PostgreSQL conceden EXECUTE en
-- funciones y USAGE en tipos a PUBLIC. Un REVOKE limitado al esquema no puede
-- reducir ese origen global, por lo que ambos defaults se cierran para el rol
-- propietario exclusivo del modulo.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_baremacion_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_baremacion_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_bolsa_baremacion.texto_opaco_valido(
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
       AND p_valor !~ '[[:space:][:cntrl:]]'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.huella_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.huella_hmac_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^hmac-sha256:[a-z0-9][a-z0-9._-]*:[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.instante_utc_valido(p_valor text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,9})?Z$' THEN
        RETURN false;
    END IF;
    PERFORM p_valor::timestamptz;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Misma codificacion que transaccion.HuellaCanonica: uint64 big-endian de la
-- longitud UTF-8, seguido de los bytes de cada parte.
CREATE FUNCTION vec_bolsa_baremacion.huella_canonica(p_partes text[])
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    parte text;
    contenido bytea := ''::bytea;
BEGIN
    FOREACH parte IN ARRAY p_partes LOOP
        IF parte IS NULL THEN
            RETURN NULL;
        END IF;
        contenido := contenido || int8send(octet_length(parte)::bigint)
            || convert_to(parte, 'UTF8');
    END LOOP;
    RETURN encode(sha256(contenido), 'hex');
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

CREATE TABLE vec_bolsa_baremacion.atestacion_pdp_version (
    atestacion_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    decision_ref text NOT NULL,
    esquema_huella_decision text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    decision_canonica bytea NOT NULL,
    suite text NOT NULL,
    algoritmo_cose text NOT NULL,
    audiencia_cose text NOT NULL,
    clave_id text NOT NULL,
    audiencia_despliegue text NOT NULL,
    estado_confianza text NOT NULL,
    huella_clave_sha256 text NOT NULL,
    payload_vec_ad_1 bytea NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    evidencia_canonica bytea NOT NULL,
    huella_payload_sha256 text NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    huella_evidencia_sha256 text NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    raiz_valida_desde timestamptz(6) NOT NULL,
    raiz_valida_hasta timestamptz(6) NOT NULL,
    revision_confianza text NOT NULL,
    huella_configuracion_sha256 text NOT NULL,
    configuracion_publicada_en timestamptz(6) NOT NULL,
    configuracion_expira_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version, estado),
    UNIQUE (decision_ref, huella_decision_sha256, atestacion_ref, version),
    CONSTRAINT atestacion_revision_valida CHECK (
        version BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT atestacion_estado_cerrado CHECK (
        estado IN ('activa', 'revocada')
    ),
    CONSTRAINT atestacion_identidad_valida CHECK (
        vec_bolsa_baremacion.texto_opaco_valido(atestacion_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(decision_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(clave_id, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(audiencia_despliegue, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(revision_confianza, 128)
        AND vec_bolsa_baremacion.texto_opaco_valido(acto_ref, 512)
        AND esquema_huella_decision =
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
        AND suite = 'VEC-AD-COSE-EDDSA-1'
        AND algoritmo_cose = 'EdDSA'
        AND audiencia_cose = 'atestacion_autorizacion_pdp'
        AND estado_confianza IN ('activa', 'revocada')
        AND estado = estado_confianza
    ),
    CONSTRAINT atestacion_huellas_validas CHECK (
        vec_bolsa_baremacion.huella_sha256_valida(huella_decision_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_clave_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_payload_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_sobre_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_evidencia_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_configuracion_sha256
        )
        AND encode(sha256(decision_canonica), 'hex') = huella_decision_sha256
        AND encode(sha256(payload_vec_ad_1), 'hex') = huella_payload_sha256
        AND encode(sha256(sobre_cose_sign1), 'hex') = huella_sobre_sha256
        AND encode(sha256(evidencia_canonica), 'hex') = huella_evidencia_sha256
    ),
    CONSTRAINT atestacion_tamanos_acotados CHECK (
        octet_length(decision_canonica) BETWEEN 1 AND 1048576
        AND octet_length(payload_vec_ad_1) BETWEEN 1 AND 524288
        AND octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384
        AND octet_length(evidencia_canonica) BETWEEN 1 AND 2097152
    ),
    CONSTRAINT atestacion_ventanas_validas CHECK (
        verificada_en >= raiz_valida_desde
        AND verificada_en < raiz_valida_hasta
        AND verificada_en >= configuracion_publicada_en
        AND verificada_en < configuracion_expira_en
        AND registrada_en >= verificada_en
    )
);

CREATE TABLE vec_bolsa_baremacion.atestacion_pdp_actual (
    decision_ref text PRIMARY KEY,
    atestacion_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    FOREIGN KEY (decision_ref, atestacion_ref, version, estado)
        REFERENCES vec_bolsa_baremacion.atestacion_pdp_version(
            decision_ref, atestacion_ref, version, estado
        ),
    CONSTRAINT atestacion_actual_estado CHECK (
        estado IN ('activa', 'revocada')
    ),
    CONSTRAINT atestacion_actual_acto CHECK (
        vec_bolsa_baremacion.texto_opaco_valido(acto_ref, 512)
    )
);

CREATE TABLE vec_bolsa_baremacion.uso_decision (
    decision_ref text PRIMARY KEY,
    esquema_huella_decision text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    tipo_efecto text NOT NULL,
    resultado_ref text NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version numeric(20, 0) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    CONSTRAINT uso_perfil_cerrado CHECK (
        esquema_huella_decision =
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
        AND tipo_efecto IN (
            'reserva', 'confirmacion', 'abandono',
            'lectura_vigente', 'lectura_version', 'lectura_evidencia'
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_efecto_sha256)
        AND vec_bolsa_baremacion.texto_opaco_valido(resultado_ref, 512)
    ),
    FOREIGN KEY (
        decision_ref, huella_decision_sha256,
        atestacion_ref, atestacion_version
    ) REFERENCES vec_bolsa_baremacion.atestacion_pdp_version(
        decision_ref, huella_decision_sha256, atestacion_ref, version
    )
);

CREATE TABLE vec_bolsa_baremacion.reserva_version (
    reserva_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    ambito_idempotencia_sha256 text NOT NULL,
    principal_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    vinculo_autenticacion_actor jsonb NOT NULL,
    baremacion_merito_ref text NOT NULL,
    clase text NOT NULL,
    version_esperada numeric(20, 0),
    huella_version_esperada_sha256 text,
    huella_solicitud_hmac text NOT NULL,
    huella_efecto_reserva_sha256 text NOT NULL,
    decision_reserva_ref text NOT NULL,
    huella_decision_reserva_sha256 text NOT NULL,
    solicitada_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    huella_confirmacion_sha256 text,
    numero_version_confirmada numeric(20, 0),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (reserva_ref, version),
    UNIQUE (ambito_idempotencia_sha256, reserva_ref, version),
    UNIQUE (ambito_idempotencia_sha256, reserva_ref, version, estado),
    CONSTRAINT reserva_version_rango CHECK (
        version BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT reserva_estado_cerrado CHECK (
        estado IN (
            'activa', 'confirmada', 'abandonada', 'expirada', 'invalidada'
        )
    ),
    CONSTRAINT reserva_identidad CHECK (
        vec_bolsa_baremacion.texto_opaco_valido(reserva_ref, 512)
        AND vec_bolsa_baremacion.huella_sha256_valida(
            ambito_idempotencia_sha256
        )
        AND vec_bolsa_baremacion.texto_opaco_valido(principal_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(sujeto_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(
            baremacion_merito_ref, 512
        )
        AND clase IN ('alta', 'incorporar_decision')
        AND vec_bolsa_baremacion.huella_hmac_sha256_valida(
            huella_solicitud_hmac
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_efecto_reserva_sha256
        )
        AND vec_bolsa_baremacion.texto_opaco_valido(
            decision_reserva_ref, 512
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_decision_reserva_sha256
        )
        AND jsonb_typeof(vinculo_autenticacion_actor) = 'object'
    ),
    CONSTRAINT reserva_occ_coherente CHECK (
        (clase = 'alta' AND version_esperada IS NULL
         AND huella_version_esperada_sha256 IS NULL)
        OR
        (clase = 'incorporar_decision'
         AND version_esperada BETWEEN 1 AND 18446744073709551615
         AND vec_bolsa_baremacion.huella_sha256_valida(
             huella_version_esperada_sha256
         ))
    ),
    CONSTRAINT reserva_ventana CHECK (
        expira_en > solicitada_en
        AND expira_en <= solicitada_en + interval '10 minutes'
        AND registrada_en >= solicitada_en
    ),
    CONSTRAINT reserva_resultado_coherente CHECK (
        (estado = 'confirmada'
         AND vec_bolsa_baremacion.huella_sha256_valida(
             huella_confirmacion_sha256
         )
         AND numero_version_confirmada BETWEEN 1 AND 18446744073709551615)
        OR
        (estado <> 'confirmada' AND huella_confirmacion_sha256 IS NULL
         AND numero_version_confirmada IS NULL)
    )
);

CREATE TABLE vec_bolsa_baremacion.reserva_actual (
    ambito_idempotencia_sha256 text PRIMARY KEY,
    reserva_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    FOREIGN KEY (
        ambito_idempotencia_sha256, reserva_ref, version, estado
    )
        REFERENCES vec_bolsa_baremacion.reserva_version(
            ambito_idempotencia_sha256, reserva_ref, version, estado
        )
);

CREATE TABLE vec_bolsa_baremacion.token_reserva (
    huella_token_sha256 text PRIMARY KEY,
    reserva_ref text NOT NULL UNIQUE,
    ambito_idempotencia_sha256 text NOT NULL UNIQUE,
    version_reserva numeric(20, 0) NOT NULL DEFAULT 1 CHECK (
        version_reserva = 1
    ),
    creada_en timestamptz(6) NOT NULL,
    CONSTRAINT token_solo_huella CHECK (
        vec_bolsa_baremacion.huella_sha256_valida(huella_token_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(
            ambito_idempotencia_sha256
        )
        AND vec_bolsa_baremacion.texto_opaco_valido(reserva_ref, 512)
    ),
    FOREIGN KEY (
        ambito_idempotencia_sha256, reserva_ref, version_reserva
    )
        REFERENCES vec_bolsa_baremacion.reserva_version(
            ambito_idempotencia_sha256, reserva_ref, version
        )
) ;

CREATE TABLE vec_bolsa_baremacion.version_baremacion (
    baremacion_merito_ref text NOT NULL,
    numero numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    agregado_canonico bytea NOT NULL,
    agregado jsonb NOT NULL,
    sujeto_ref text NOT NULL,
    proceso_ref text NOT NULL,
    solicitud_ref text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    evento_outbox_ref text NOT NULL UNIQUE,
    PRIMARY KEY (baremacion_merito_ref, numero),
    UNIQUE (baremacion_merito_ref, numero, huella_estado_sha256),
    CONSTRAINT version_baremacion_rango CHECK (
        numero BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT version_baremacion_huella CHECK (
        vec_bolsa_baremacion.huella_sha256_valida(huella_estado_sha256)
        AND encode(sha256(agregado_canonico), 'hex') = huella_estado_sha256
        AND agregado = convert_from(agregado_canonico, 'UTF8')::jsonb
        AND octet_length(agregado_canonico) BETWEEN 1 AND 33554432
    ),
    CONSTRAINT version_baremacion_identidad CHECK (
        vec_bolsa_baremacion.texto_opaco_valido(
            baremacion_merito_ref, 512
        )
        AND vec_bolsa_baremacion.texto_opaco_valido(sujeto_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(proceso_ref, 512)
        AND vec_bolsa_baremacion.texto_opaco_valido(solicitud_ref, 512)
        AND agregado ->> 'id' = baremacion_merito_ref
        AND agregado ->> 'sujeto_ref' = sujeto_ref
        AND agregado ->> 'proceso_ref' = proceso_ref
        AND agregado ->> 'solicitud_ref' = solicitud_ref
        AND jsonb_typeof(agregado -> 'decisiones') = 'array'
        AND jsonb_array_length(agregado -> 'decisiones') + 1 = numero
    )
);

CREATE TABLE vec_bolsa_baremacion.baremacion_actual (
    baremacion_merito_ref text PRIMARY KEY,
    numero numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (baremacion_merito_ref, numero, huella_estado_sha256)
        REFERENCES vec_bolsa_baremacion.version_baremacion(
            baremacion_merito_ref, numero, huella_estado_sha256
        )
);

CREATE TABLE vec_bolsa_baremacion.auditoria (
    referencia text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    principal_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    perfil_actor_clave text NOT NULL,
    metodo_autenticacion text NOT NULL,
    nivel_autenticacion text NOT NULL,
    garantia_minima text NOT NULL,
    autenticacion_ref text NOT NULL,
    autorizacion_ref text NOT NULL,
    accion_autorizada text NOT NULL,
    clase_recurso_autorizada text NOT NULL,
    recurso_autorizado_ref text NOT NULL,
    campos_permitidos jsonb NOT NULL,
    finalidad_clave text NOT NULL,
    correlacion_ref text NOT NULL,
    modulo text NOT NULL,
    accion text NOT NULL,
    clase_cambio text NOT NULL,
    proceso_ref text NOT NULL,
    solicitud_ref text NOT NULL,
    baremacion_merito_ref text NOT NULL,
    decision_ref text NOT NULL,
    manifiesto_probatorio_ref text NOT NULL,
    huella_manifiesto_sha256 text NOT NULL,
    documento_firmado_custodiado_ref text NOT NULL,
    evidencia_custodia_firmado_ref text NOT NULL,
    evidencia_retencion_firmado_ref text NOT NULL,
    version_anterior numeric(20, 0) NOT NULL,
    version_nueva numeric(20, 0) NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_nueva_sha256 text NOT NULL,
    motivo_clave text NOT NULL,
    motivo text NOT NULL,
    huella_solicitud_hmac text NOT NULL,
    resultado text NOT NULL,
    solicitada_confirmacion_en timestamptz(6) NOT NULL,
    -- PostgreSQL conserva microsegundos; esta preimagen preserva los
    -- nanosegundos que forman parte de la solicitud sellada y de la cadena.
    solicitada_confirmacion_canonica text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_anterior_auditoria_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CONSTRAINT auditoria_perfil_cerrado CHECK (
        secuencia BETWEEN 1 AND 18446744073709551615
        AND modulo = 'bolsa' AND resultado = 'correcto'
        AND accion IN ('crear_baremacion', 'incorporar_decision_baremacion')
        AND clase_cambio IN ('alta', 'incorporar_decision')
        AND jsonb_typeof(campos_permitidos) = 'array'
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_nueva_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_registro_sha256)
        AND vec_bolsa_baremacion.huella_hmac_sha256_valida(
            huella_solicitud_hmac
        )
        AND registrada_en >= solicitada_confirmacion_en
        AND vec_bolsa_baremacion.instante_utc_valido(
            solicitada_confirmacion_canonica
        )
        AND solicitada_confirmacion_en =
            solicitada_confirmacion_canonica::timestamptz
        AND version_nueva = version_anterior + 1
    )
);

CREATE TABLE vec_bolsa_baremacion.evento_outbox (
    referencia text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo text NOT NULL,
    estado text NOT NULL,
    modulo text NOT NULL,
    proceso_ref text NOT NULL,
    solicitud_ref text NOT NULL,
    baremacion_merito_ref text NOT NULL,
    decision_ref text NOT NULL,
    manifiesto_probatorio_ref text NOT NULL,
    huella_manifiesto_sha256 text NOT NULL,
    documento_firmado_ref text NOT NULL,
    evidencia_custodia_firmado_ref text NOT NULL,
    evidencia_retencion_firmado_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    principal_ref text NOT NULL,
    version_nueva numeric(20, 0) NOT NULL,
    huella_nueva_sha256 text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE
        REFERENCES vec_bolsa_baremacion.auditoria(referencia),
    huella_auditoria_sha256 text NOT NULL,
    correlacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_evento_anterior_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CONSTRAINT outbox_perfil_cerrado CHECK (
        secuencia BETWEEN 1 AND 18446744073709551615
        AND tipo IN (
            'bolsa.baremacion_creada.v1',
            'bolsa.decision_baremacion_incorporada.v1'
        )
        AND estado = 'pendiente' AND modulo = 'bolsa'
        AND version_nueva BETWEEN 1 AND 18446744073709551615
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_nueva_sha256)
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_auditoria_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_registro_sha256)
    )
);

CREATE INDEX reserva_actual_baremacion
    ON vec_bolsa_baremacion.reserva_version(
        baremacion_merito_ref, estado, expira_en
    );
CREATE INDEX version_baremacion_sujeto
    ON vec_bolsa_baremacion.version_baremacion(sujeto_ref, confirmada_en);
CREATE INDEX outbox_pendiente_secuencia
    ON vec_bolsa_baremacion.evento_outbox(estado, secuencia);

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_atestacion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.decision_ref IS DISTINCT FROM OLD.decision_ref
       OR NEW.atestacion_ref IS DISTINCT FROM OLD.atestacion_ref
       OR NEW.version IS DISTINCT FROM OLD.version + 1
       OR NEW.actualizada_en <= OLD.actualizada_en
       OR OLD.estado = 'revocada' THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de atestacion PDP invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_reserva_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.ambito_idempotencia_sha256 IS DISTINCT FROM
           OLD.ambito_idempotencia_sha256
       OR NEW.reserva_ref IS DISTINCT FROM OLD.reserva_ref
       OR NEW.version IS DISTINCT FROM OLD.version + 1
       OR OLD.estado <> 'activa'
       OR NEW.estado NOT IN ('confirmada', 'abandonada', 'expirada', 'invalidada') THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de reserva invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_baremacion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.baremacion_merito_ref IS DISTINCT FROM OLD.baremacion_merito_ref
       OR NEW.numero IS DISTINCT FROM OLD.numero + 1
       OR NEW.actualizada_en <= OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de version de baremacion invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER atestacion_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.atestacion_pdp_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_atestacion_actual();
CREATE TRIGGER reserva_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.reserva_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_reserva_actual();
CREATE TRIGGER baremacion_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.baremacion_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_baremacion_actual();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_pdp_version', 'uso_decision', 'reserva_version',
        'token_reserva', 'version_baremacion', 'auditoria', 'evento_outbox'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_baremacion.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_baremacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_pdp_actual', 'reserva_actual', 'baremacion_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_bolsa_baremacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
END
$protecciones$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_pdp_version', 'atestacion_pdp_actual', 'uso_decision',
        'reserva_version', 'reserva_actual', 'token_reserva',
        'version_baremacion', 'baremacion_actual', 'auditoria', 'evento_outbox'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_baremacion.%I FOR ALL TO vec_bolsa_baremacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_baremacion_propietario',
            'vec_bolsa_baremacion_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_baremacion FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_baremacion FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_baremacion FROM PUBLIC;
-- El default de TYPES tampoco alcanza los tipos fila implicitos. La guarda DDL
-- instalada por roles_up protege los futuros; este cierre explicito comprueba
-- y refuerza el estado de todos los tipos creados por esta migracion.
DO $cerrar_tipos_existentes$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I, %I',
            tipo.nspname,
            tipo.typname,
            'vec_bolsa_baremacion_ejecutor',
            'vec_bolsa_baremacion_lector_outbox',
            'vec_bolsa_baremacion_registrador_atestacion'
        );
    END LOOP;
END
$cerrar_tipos_existentes$;

COMMIT;
