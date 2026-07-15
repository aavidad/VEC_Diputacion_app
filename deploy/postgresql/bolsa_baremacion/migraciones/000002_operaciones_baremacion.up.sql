BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

ALTER TABLE vec_bolsa_baremacion.version_baremacion
    ADD CONSTRAINT version_auditoria_exacta FOREIGN KEY (auditoria_ref)
        REFERENCES vec_bolsa_baremacion.auditoria(referencia),
    ADD CONSTRAINT version_evento_exacto FOREIGN KEY (evento_outbox_ref)
        REFERENCES vec_bolsa_baremacion.evento_outbox(referencia);

CREATE FUNCTION vec_bolsa_baremacion.instante_rfc3339nano(
    p_instante timestamptz
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    base text;
    fraccion text;
BEGIN
    base := to_char(p_instante AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS');
    fraccion := rtrim(
        to_char(p_instante AT TIME ZONE 'UTC', 'US'), '0'
    );
    IF fraccion = '' THEN
        RETURN base || 'Z';
    END IF;
    RETURN base || '.' || fraccion || 'Z';
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.unir_textos_nul(p_lista jsonb)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    valor text;
    resultado bytea := ''::bytea;
    primero boolean := true;
BEGIN
    IF jsonb_typeof(p_lista) <> 'array' THEN
        RETURN NULL;
    END IF;
    FOR valor IN SELECT value FROM jsonb_array_elements_text(p_lista) LOOP
        IF NOT primero THEN
            resultado := resultado || decode('00', 'hex');
        END IF;
        resultado := resultado || convert_to(valor, 'UTF8');
        primero := false;
    END LOOP;
    RETURN resultado;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.huella_canonica_bytes(p_partes bytea[])
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    parte bytea;
    contenido bytea := ''::bytea;
BEGIN
    FOREACH parte IN ARRAY p_partes LOOP
        IF parte IS NULL THEN
            RETURN NULL;
        END IF;
        contenido := contenido || int8send(octet_length(parte)::bigint)
            || parte;
    END LOOP;
    RETURN encode(sha256(contenido), 'hex');
END
$funcion$;

-- Esta funcion no verifica COSE. Exige una atestacion durable previamente
-- registrada por una frontera criptografica confiable, actualmente sin grant
-- ni funcion publica de alta. Ausencia, revocacion o caducidad deniegan.
CREATE FUNCTION vec_bolsa_baremacion.obtener_atestacion_actual_valida(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_instante timestamptz
)
RETURNS TABLE (atestacion_ref text, atestacion_version numeric)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    atestacion record;
BEGIN
    IF p_prueba IS NULL OR p_decision_canonica IS NULL
       OR p_instante IS NULL THEN
        RETURN;
    END IF;
    SELECT version.atestacion_ref, version.version,
           version.estado, version.estado_confianza,
           version.esquema_huella_decision,
           version.huella_decision_sha256, version.decision_canonica,
           version.verificada_en, version.raiz_valida_hasta,
           version.configuracion_expira_en
      INTO atestacion
      FROM vec_bolsa_baremacion.atestacion_pdp_actual AS actual
      JOIN vec_bolsa_baremacion.atestacion_pdp_version AS version
        ON version.decision_ref = actual.decision_ref
       AND version.atestacion_ref = actual.atestacion_ref
       AND version.version = actual.version
     WHERE actual.decision_ref = p_prueba ->> 'decision_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND OR atestacion.estado <> 'activa'
       OR atestacion.estado_confianza <> 'activa'
       OR atestacion.esquema_huella_decision IS DISTINCT FROM
          p_prueba ->> 'esquema_huella'
       OR atestacion.huella_decision_sha256 IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256'
       OR atestacion.decision_canonica IS DISTINCT FROM p_decision_canonica
       OR atestacion.verificada_en >
          (p_prueba ->> 'verificada_en')::timestamptz
       OR p_instante >= atestacion.raiz_valida_hasta
       OR p_instante >= atestacion.configuracion_expira_en THEN
        RETURN;
    END IF;
    atestacion_ref := atestacion.atestacion_ref;
    atestacion_version := atestacion.version;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.revalidar_operacion(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_accion text,
    p_clase_recurso text,
    p_recurso_ref text,
    p_campos_exactos jsonb,
    p_instante timestamptz
)
RETURNS TABLE (
    atestacion_ref text,
    atestacion_version numeric,
    decision_canonica jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_prueba IS NULL OR p_decision_canonica IS NULL
       OR p_recurso_canonico IS NULL OR p_accion IS NULL
       OR p_clase_recurso IS NULL OR p_recurso_ref IS NULL
       OR p_campos_exactos IS NULL OR p_instante IS NULL THEN
        RETURN;
    END IF;
    SELECT a.atestacion_ref, a.atestacion_version
      INTO atestacion_ref, atestacion_version
      FROM vec_bolsa_baremacion.obtener_atestacion_actual_valida(
          p_prueba, p_decision_canonica, p_instante
      ) AS a;
    IF NOT FOUND OR vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
        p_prueba, p_decision_canonica, p_recurso_canonico,
        p_accion, p_clase_recurso, p_recurso_ref, p_campos_exactos,
        p_instante
    ) IS NOT TRUE THEN
        RETURN;
    END IF;
    decision_canonica := convert_from(p_decision_canonica, 'UTF8')::jsonb;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.reservar_cambio(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    reserva_ref text,
    expira_en timestamptz,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    evento_outbox_ref text,
    huella_evento_outbox_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6);
    solicitada timestamptz(6);
    caducidad timestamptz(6);
    accion text;
    campos jsonb;
    version_esperada numeric(20, 0);
    huella_esperada text;
    autorizacion record;
    actual_reserva record;
    otra_reserva record;
    actual_baremacion record;
    uso record;
    version_confirmada record;
BEGIN
    resultado := 'rechazada';
    reserva_ref := '';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    huella_auditoria_sha256 := '';
    evento_outbox_ref := '';
    huella_evento_outbox_sha256 := '';

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 12
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'reserva_ref', 'huella_token_sha256',
           'ambito_idempotencia_sha256', 'clase',
           'baremacion_merito_ref', 'version_esperada',
           'huella_version_esperada_sha256', 'huella_solicitud_hmac',
           'huella_efecto_sha256', 'solicitada_en', 'expira_en'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.reserva-postgresql.v1'
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'reserva_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_token_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'ambito_idempotencia_sha256'
       ) IS NOT TRUE
       OR p_operacion ->> 'clase' NOT IN ('alta', 'incorporar_decision')
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_hmac_sha256_valida(
           p_operacion ->> 'huella_solicitud_hmac'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.instante_utc_valido(
           p_operacion ->> 'solicitada_en'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.instante_utc_valido(
           p_operacion ->> 'expira_en'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;

    IF p_operacion ->> 'clase' = 'alta' THEN
        accion := 'bolsa.baremacion.alta.reservar';
        campos := '["reserva.alta"]'::jsonb;
        IF p_operacion ->> 'version_esperada' <> '0'
           OR p_operacion ->> 'huella_version_esperada_sha256' <> '' THEN
            RETURN NEXT;
            RETURN;
        END IF;
        version_esperada := NULL;
        huella_esperada := NULL;
    ELSE
        accion := 'bolsa.baremacion.decision.reservar';
        campos := '["reserva.decision"]'::jsonb;
        IF (p_operacion ->> 'version_esperada') !~ '^[1-9][0-9]{0,19}$'
           OR vec_bolsa_baremacion.huella_sha256_valida(
               p_operacion ->> 'huella_version_esperada_sha256'
           ) IS NOT TRUE THEN
            RETURN NEXT;
            RETURN;
        END IF;
        BEGIN
            version_esperada :=
                (p_operacion ->> 'version_esperada')::numeric;
        EXCEPTION WHEN numeric_value_out_of_range THEN
            RETURN NEXT;
            RETURN;
        END;
        IF version_esperada > 18446744073709551615 THEN
            RETURN NEXT;
            RETURN;
        END IF;
        huella_esperada :=
            p_operacion ->> 'huella_version_esperada_sha256';
    END IF;

    solicitada := (p_operacion ->> 'solicitada_en')::timestamptz;
    caducidad := (p_operacion ->> 'expira_en')::timestamptz;
    instante := clock_timestamp();
    IF solicitada > instante OR instante >= caducidad
       OR caducidad <= solicitada
       OR caducidad > solicitada + interval '10 minutes' THEN
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          accion, 'baremacion', p_operacion ->> 'baremacion_merito_ref',
          campos, instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:idempotencia:' ||
            (p_operacion ->> 'ambito_idempotencia_sha256'), 0
    ));
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:agregado:' ||
            (p_operacion ->> 'baremacion_merito_ref'), 0
    ));

    SELECT actual.reserva_ref, actual.version, actual.estado,
           version.principal_ref, version.sujeto_ref,
           version.baremacion_merito_ref, version.clase,
           version.version_esperada, version.huella_version_esperada_sha256,
           version.huella_solicitud_hmac,
           version.huella_efecto_reserva_sha256,
           version.decision_reserva_ref,
           version.huella_decision_reserva_sha256,
           version.solicitada_en, version.expira_en,
           version.numero_version_confirmada
      INTO actual_reserva
      FROM vec_bolsa_baremacion.reserva_actual AS actual
      JOIN vec_bolsa_baremacion.reserva_version AS version
        ON version.ambito_idempotencia_sha256 =
           actual.ambito_idempotencia_sha256
       AND version.reserva_ref = actual.reserva_ref
       AND version.version = actual.version
     WHERE actual.ambito_idempotencia_sha256 =
           p_operacion ->> 'ambito_idempotencia_sha256'
     FOR UPDATE OF actual;
    IF FOUND THEN
        reserva_ref := actual_reserva.reserva_ref;
        expira_en := actual_reserva.expira_en;
        IF actual_reserva.principal_ref IS DISTINCT FROM
               autorizacion.decision_canonica ->> 'principal_id'
           OR actual_reserva.sujeto_ref IS DISTINCT FROM
               p_prueba ->> 'sujeto_ref'
           OR actual_reserva.baremacion_merito_ref IS DISTINCT FROM
               p_operacion ->> 'baremacion_merito_ref'
           OR actual_reserva.clase IS DISTINCT FROM p_operacion ->> 'clase'
           OR actual_reserva.version_esperada IS DISTINCT FROM version_esperada
           OR actual_reserva.huella_version_esperada_sha256 IS DISTINCT FROM
               huella_esperada
           OR actual_reserva.huella_solicitud_hmac IS DISTINCT FROM
               p_operacion ->> 'huella_solicitud_hmac'
           OR actual_reserva.huella_efecto_reserva_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR actual_reserva.decision_reserva_ref IS DISTINCT FROM
               p_prueba ->> 'decision_ref'
           OR actual_reserva.huella_decision_reserva_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR actual_reserva.solicitada_en IS DISTINCT FROM solicitada
           OR actual_reserva.expira_en IS DISTINCT FROM caducidad THEN
            resultado := 'idempotencia_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        IF actual_reserva.estado = 'confirmada' THEN
            SELECT version.numero::text, version.huella_estado_sha256,
                   version.agregado_canonico, version.confirmada_en,
                   version.auditoria_ref, auditoria.huella_registro_sha256,
                   version.evento_outbox_ref, evento.huella_registro_sha256
              INTO numero_version, huella_estado_sha256,
                   agregado_canonico, confirmada_en,
                   auditoria_ref, huella_auditoria_sha256,
                   evento_outbox_ref, huella_evento_outbox_sha256
              FROM vec_bolsa_baremacion.version_baremacion AS version
              JOIN vec_bolsa_baremacion.auditoria
                ON auditoria.referencia = version.auditoria_ref
              JOIN vec_bolsa_baremacion.evento_outbox AS evento
                ON evento.referencia = version.evento_outbox_ref
             WHERE version.baremacion_merito_ref =
                   actual_reserva.baremacion_merito_ref
               AND version.numero =
                   actual_reserva.numero_version_confirmada;
            IF NOT FOUND THEN
                resultado := 'evidencia_no_confiable';
            ELSE
                resultado := 'confirmada';
            END IF;
            RETURN NEXT;
            RETURN;
        ELSIF actual_reserva.estado = 'activa' AND instante < caducidad THEN
            resultado := 'en_curso';
            RETURN NEXT;
            RETURN;
        ELSIF actual_reserva.estado = 'activa' THEN
            INSERT INTO vec_bolsa_baremacion.reserva_version (
                reserva_ref, version, estado, ambito_idempotencia_sha256,
                principal_ref, sujeto_ref, vinculo_autenticacion_actor,
                baremacion_merito_ref, clase, version_esperada,
                huella_version_esperada_sha256, huella_solicitud_hmac,
                huella_efecto_reserva_sha256, decision_reserva_ref,
                huella_decision_reserva_sha256, solicitada_en, expira_en,
                registrada_en
            ) SELECT reserva_ref, version + 1, 'expirada',
                ambito_idempotencia_sha256, principal_ref, sujeto_ref,
                vinculo_autenticacion_actor, baremacion_merito_ref, clase,
                version_esperada, huella_version_esperada_sha256,
                huella_solicitud_hmac, huella_efecto_reserva_sha256,
                decision_reserva_ref, huella_decision_reserva_sha256,
                solicitada_en, expira_en, instante
              FROM vec_bolsa_baremacion.reserva_version
             WHERE reserva_ref = actual_reserva.reserva_ref
               AND version = actual_reserva.version;
            UPDATE vec_bolsa_baremacion.reserva_actual
               SET version = actual_reserva.version + 1,
                   estado = 'expirada'
             WHERE ambito_idempotencia_sha256 =
                   p_operacion ->> 'ambito_idempotencia_sha256'
               AND version = actual_reserva.version;
            resultado := 'idempotencia_reutilizada';
            RETURN NEXT;
            RETURN;
        ELSE
            resultado := 'idempotencia_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
    END IF;

    SELECT * INTO uso
      FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    IF FOUND THEN
        resultado := 'autorizacion_reutilizada';
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT actual.numero, actual.huella_estado_sha256,
           version.sujeto_ref
      INTO actual_baremacion
      FROM vec_bolsa_baremacion.baremacion_actual AS actual
      JOIN vec_bolsa_baremacion.version_baremacion AS version
        ON version.baremacion_merito_ref = actual.baremacion_merito_ref
       AND version.numero = actual.numero
     WHERE actual.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
     FOR UPDATE OF actual;
    IF p_operacion ->> 'clase' = 'alta' THEN
        IF FOUND THEN
            IF actual_baremacion.sujeto_ref IS DISTINCT FROM
               p_prueba ->> 'sujeto_ref' THEN
                resultado := 'no_encontrada';
            ELSE
                resultado := 'ya_existe';
            END IF;
            RETURN NEXT;
            RETURN;
        END IF;
    ELSE
        IF NOT FOUND OR actual_baremacion.sujeto_ref IS DISTINCT FROM
           p_prueba ->> 'sujeto_ref' THEN
            resultado := 'no_encontrada';
            RETURN NEXT;
            RETURN;
        END IF;
        IF actual_baremacion.numero IS DISTINCT FROM version_esperada
           OR actual_baremacion.huella_estado_sha256 IS DISTINCT FROM
              huella_esperada THEN
            resultado := 'conflicto_version';
            RETURN NEXT;
            RETURN;
        END IF;
    END IF;

    SELECT actual.reserva_ref, version.sujeto_ref
      INTO otra_reserva
      FROM vec_bolsa_baremacion.reserva_actual AS actual
      JOIN vec_bolsa_baremacion.reserva_version AS version
        ON version.ambito_idempotencia_sha256 =
           actual.ambito_idempotencia_sha256
       AND version.reserva_ref = actual.reserva_ref
       AND version.version = actual.version
     WHERE version.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
       AND actual.estado = 'activa' AND version.expira_en > instante
     LIMIT 1 FOR UPDATE OF actual;
    IF FOUND THEN
        IF otra_reserva.sujeto_ref IS DISTINCT FROM
           p_prueba ->> 'sujeto_ref' THEN
            resultado := 'no_encontrada';
        ELSE
            resultado := 'en_curso';
        END IF;
        RETURN NEXT;
        RETURN;
    END IF;

    INSERT INTO vec_bolsa_baremacion.reserva_version (
        reserva_ref, version, estado, ambito_idempotencia_sha256,
        principal_ref, sujeto_ref, vinculo_autenticacion_actor,
        baremacion_merito_ref, clase, version_esperada,
        huella_version_esperada_sha256, huella_solicitud_hmac,
        huella_efecto_reserva_sha256, decision_reserva_ref,
        huella_decision_reserva_sha256, solicitada_en, expira_en,
        registrada_en
    ) VALUES (
        p_operacion ->> 'reserva_ref', 1, 'activa',
        p_operacion ->> 'ambito_idempotencia_sha256',
        autorizacion.decision_canonica ->> 'principal_id',
        p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor',
        p_operacion ->> 'baremacion_merito_ref', p_operacion ->> 'clase',
        version_esperada, huella_esperada,
        p_operacion ->> 'huella_solicitud_hmac',
        p_operacion ->> 'huella_efecto_sha256',
        p_prueba ->> 'decision_ref',
        p_prueba ->> 'huella_decision_sha256', solicitada, caducidad,
        instante
    );
    INSERT INTO vec_bolsa_baremacion.reserva_actual (
        ambito_idempotencia_sha256, reserva_ref, version, estado
    ) VALUES (
        p_operacion ->> 'ambito_idempotencia_sha256',
        p_operacion ->> 'reserva_ref', 1, 'activa'
    );
    INSERT INTO vec_bolsa_baremacion.token_reserva (
        huella_token_sha256, reserva_ref, ambito_idempotencia_sha256,
        version_reserva, creada_en
    ) VALUES (
        p_operacion ->> 'huella_token_sha256',
        p_operacion ->> 'reserva_ref',
        p_operacion ->> 'ambito_idempotencia_sha256', 1, instante
    );
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'reserva',
        p_operacion ->> 'reserva_ref', autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );
    resultado := 'reservada';
    reserva_ref := p_operacion ->> 'reserva_ref';
    expira_en := caducidad;
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'colision';
        reserva_ref := '';
        expira_en := NULL;
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR check_violation
        OR foreign_key_violation OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        reserva_ref := '';
        expira_en := NULL;
        RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.confirmar_cambio(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_agregado_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    evento_outbox_ref text,
    huella_evento_outbox_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
#variable_conflict use_variable
DECLARE
    instante timestamptz(6);
    solicitada_confirmacion timestamptz(6);
    accion_autorizada text;
    campos jsonb;
    version_esperada numeric(20, 0);
    huella_esperada text;
    autorizacion record;
    agregado jsonb;
    reserva record;
    uso record;
    actual_baremacion record;
    uso_encontrado boolean := false;
    confirmada_anterior timestamptz(6);
    agregado_anterior jsonb;
    prefijo_decisiones jsonb;
    ultima_decision jsonb;
    version_anterior numeric(20, 0);
    version_nueva numeric(20, 0);
    huella_anterior text;
    secuencia_auditoria numeric(20, 0);
    secuencia_evento numeric(20, 0);
    huella_auditoria_anterior text;
    huella_evento_anterior text;
    accion_auditoria text;
    tipo_evento text;
    decision_tecnica_ref text := '';
    manifiesto_ref text := '';
    huella_manifiesto text := '';
    documento_firmado_ref text := '';
    evidencia_custodia_ref text := '';
    evidencia_retencion_ref text := '';
    huella_auditoria text;
    huella_evento text;
    campos_unidos bytea;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    huella_auditoria_sha256 := '';
    evento_outbox_ref := '';
    huella_evento_outbox_sha256 := '';

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 13
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'huella_token_sha256', 'clase',
           'version_esperada', 'huella_version_esperada_sha256',
           'huella_solicitud_hmac', 'huella_efecto_sha256',
           'huella_agregado_sha256', 'motivo_clave', 'motivo',
           'confirmada_en', 'auditoria_ref', 'evento_outbox_ref'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.confirmacion-postgresql.v1'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_token_sha256'
       ) IS NOT TRUE
       OR p_operacion ->> 'clase' NOT IN ('alta', 'incorporar_decision')
       OR vec_bolsa_baremacion.huella_hmac_sha256_valida(
           p_operacion ->> 'huella_solicitud_hmac'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_agregado_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'motivo_clave', 128
       ) IS NOT TRUE
       OR p_operacion ->> 'motivo' IS NULL
       OR octet_length(p_operacion ->> 'motivo') NOT BETWEEN 1 AND 8000
       OR p_operacion ->> 'motivo' <> btrim(p_operacion ->> 'motivo')
       OR vec_bolsa_baremacion.instante_utc_valido(
           p_operacion ->> 'confirmada_en'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'auditoria_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'evento_outbox_ref', 512
       ) IS NOT TRUE
       OR p_operacion ->> 'auditoria_ref' =
          p_operacion ->> 'evento_outbox_ref'
       OR p_agregado_canonico IS NULL
       OR octet_length(p_agregado_canonico) NOT BETWEEN 1 AND 33554432
       OR encode(sha256(p_agregado_canonico), 'hex') IS DISTINCT FROM
          p_operacion ->> 'huella_agregado_sha256' THEN
        RETURN NEXT;
        RETURN;
    END IF;

    BEGIN
        agregado := convert_from(p_agregado_canonico, 'UTF8')::jsonb;
        solicitada_confirmacion :=
            (p_operacion ->> 'confirmada_en')::timestamptz;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN NEXT;
            RETURN;
    END;
    instante := clock_timestamp();
    IF solicitada_confirmacion > instante
       OR jsonb_typeof(agregado) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(agregado)) <> 10
       OR NOT (agregado ?& ARRAY[
           'id', 'proceso_ref', 'solicitud_ref', 'sujeto_ref', 'criterio',
           'evidencias_iniciales', 'puntos_declarados', 'calculo_inicial',
           'creada_en', 'decisiones'
       ])
       OR vec_bolsa_baremacion.texto_opaco_valido(
           agregado ->> 'id', 512
       ) IS NOT TRUE
       OR agregado ->> 'sujeto_ref' IS DISTINCT FROM
          p_prueba ->> 'sujeto_ref'
       OR jsonb_typeof(agregado -> 'decisiones') <> 'array'
       OR jsonb_array_length(agregado -> 'decisiones') > 4096 THEN
        RETURN NEXT;
        RETURN;
    END IF;

    IF p_operacion ->> 'clase' = 'alta' THEN
        accion_autorizada := 'bolsa.baremacion.alta.confirmar';
        campos := '["baremacion","evidencia_transaccion"]'::jsonb;
        IF p_operacion ->> 'version_esperada' <> '0'
           OR p_operacion ->> 'huella_version_esperada_sha256' <> ''
           OR jsonb_array_length(agregado -> 'decisiones') <> 0 THEN
            RETURN NEXT;
            RETURN;
        END IF;
        version_esperada := NULL;
        huella_esperada := NULL;
    ELSE
        accion_autorizada := 'bolsa.baremacion.decision.confirmar';
        campos := '["baremacion","decision","evidencia_transaccion"]'::jsonb;
        IF (p_operacion ->> 'version_esperada') !~ '^[1-9][0-9]{0,19}$'
           OR vec_bolsa_baremacion.huella_sha256_valida(
               p_operacion ->> 'huella_version_esperada_sha256'
           ) IS NOT TRUE THEN
            RETURN NEXT;
            RETURN;
        END IF;
        version_esperada :=
            (p_operacion ->> 'version_esperada')::numeric;
        huella_esperada :=
            p_operacion ->> 'huella_version_esperada_sha256';
        IF version_esperada > 4096
           OR jsonb_array_length(agregado -> 'decisiones') IS DISTINCT FROM
              version_esperada::integer THEN
            RETURN NEXT;
            RETURN;
        END IF;
    END IF;

    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          accion_autorizada, 'baremacion', agregado ->> 'id',
          campos, instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:confirmacion:' ||
            (p_operacion ->> 'huella_token_sha256'), 0
    ));
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:agregado:' || (agregado ->> 'id'), 0
    ));

    SELECT actual.ambito_idempotencia_sha256, actual.reserva_ref,
           actual.version, actual.estado, version.principal_ref,
           version.sujeto_ref, version.vinculo_autenticacion_actor,
           version.baremacion_merito_ref, version.clase,
           version.version_esperada, version.huella_version_esperada_sha256,
           version.huella_solicitud_hmac, version.solicitada_en,
           version.expira_en, version.huella_confirmacion_sha256,
           version.numero_version_confirmada
      INTO reserva
      FROM vec_bolsa_baremacion.token_reserva AS token
      JOIN vec_bolsa_baremacion.reserva_actual AS actual
        ON actual.ambito_idempotencia_sha256 = token.ambito_idempotencia_sha256
       AND actual.reserva_ref = token.reserva_ref
      JOIN vec_bolsa_baremacion.reserva_version AS version
        ON version.ambito_idempotencia_sha256 =
           actual.ambito_idempotencia_sha256
       AND version.reserva_ref = actual.reserva_ref
       AND version.version = actual.version
     WHERE token.huella_token_sha256 =
           p_operacion ->> 'huella_token_sha256'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR reserva.principal_ref IS DISTINCT FROM
          autorizacion.decision_canonica ->> 'principal_id'
       OR reserva.sujeto_ref IS DISTINCT FROM p_prueba ->> 'sujeto_ref'
       OR reserva.vinculo_autenticacion_actor IS DISTINCT FROM
          autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
       OR reserva.baremacion_merito_ref IS DISTINCT FROM agregado ->> 'id'
       OR reserva.clase IS DISTINCT FROM p_operacion ->> 'clase'
       OR reserva.version_esperada IS DISTINCT FROM version_esperada
       OR reserva.huella_version_esperada_sha256 IS DISTINCT FROM
          huella_esperada
       OR reserva.huella_solicitud_hmac IS DISTINCT FROM
          p_operacion ->> 'huella_solicitud_hmac'
       OR solicitada_confirmacion < reserva.solicitada_en
       OR solicitada_confirmacion >= reserva.expira_en THEN
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT * INTO uso FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref' FOR SHARE;
    uso_encontrado := FOUND;
    IF reserva.estado = 'confirmada' THEN
        IF reserva.huella_confirmacion_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR NOT uso_encontrado
           OR uso.huella_decision_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR uso.tipo_efecto <> 'confirmacion' THEN
            resultado := 'idempotencia_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT version.numero::text, version.huella_estado_sha256,
               version.agregado_canonico, version.confirmada_en,
               version.auditoria_ref, auditoria.huella_registro_sha256,
               version.evento_outbox_ref, evento.huella_registro_sha256
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref, huella_auditoria_sha256,
               evento_outbox_ref, huella_evento_outbox_sha256
          FROM vec_bolsa_baremacion.version_baremacion AS version
          JOIN vec_bolsa_baremacion.auditoria
            ON auditoria.referencia = version.auditoria_ref
          JOIN vec_bolsa_baremacion.evento_outbox AS evento
            ON evento.referencia = version.evento_outbox_ref
         WHERE version.baremacion_merito_ref = reserva.baremacion_merito_ref
           AND version.numero = reserva.numero_version_confirmada;
        resultado := CASE WHEN FOUND THEN 'confirmada' ELSE
            'evidencia_no_confiable' END;
        RETURN NEXT;
        RETURN;
    END IF;
    IF reserva.estado <> 'activa' THEN
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;
    IF instante >= reserva.expira_en THEN
        INSERT INTO vec_bolsa_baremacion.reserva_version (
            reserva_ref, version, estado, ambito_idempotencia_sha256,
            principal_ref, sujeto_ref, vinculo_autenticacion_actor,
            baremacion_merito_ref, clase, version_esperada,
            huella_version_esperada_sha256, huella_solicitud_hmac,
            huella_efecto_reserva_sha256, decision_reserva_ref,
            huella_decision_reserva_sha256, solicitada_en, expira_en,
            registrada_en
        ) SELECT reserva_ref, version + 1, 'expirada',
            ambito_idempotencia_sha256, principal_ref, sujeto_ref,
            vinculo_autenticacion_actor, baremacion_merito_ref, clase,
            version_esperada, huella_version_esperada_sha256,
            huella_solicitud_hmac, huella_efecto_reserva_sha256,
            decision_reserva_ref, huella_decision_reserva_sha256,
            solicitada_en, expira_en, instante
          FROM vec_bolsa_baremacion.reserva_version
         WHERE reserva_ref = reserva.reserva_ref
           AND version = reserva.version;
        UPDATE vec_bolsa_baremacion.reserva_actual
           SET version = reserva.version + 1, estado = 'expirada'
         WHERE ambito_idempotencia_sha256 =
               reserva.ambito_idempotencia_sha256
           AND version = reserva.version AND estado = 'activa';
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS de expiracion perdido';
        END IF;
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;
    IF uso_encontrado THEN
        resultado := 'autorizacion_reutilizada';
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT actual.numero, actual.huella_estado_sha256,
           version.agregado, version.sujeto_ref, version.confirmada_en
      INTO actual_baremacion
      FROM vec_bolsa_baremacion.baremacion_actual AS actual
      JOIN vec_bolsa_baremacion.version_baremacion AS version
        ON version.baremacion_merito_ref = actual.baremacion_merito_ref
       AND version.numero = actual.numero
     WHERE actual.baremacion_merito_ref = agregado ->> 'id'
     FOR UPDATE OF actual;
    IF p_operacion ->> 'clase' = 'alta' THEN
        IF FOUND THEN
            resultado := CASE
                WHEN actual_baremacion.sujeto_ref IS DISTINCT FROM
                     p_prueba ->> 'sujeto_ref' THEN 'no_encontrada'
                ELSE 'ya_existe' END;
            RETURN NEXT;
            RETURN;
        END IF;
        version_anterior := 0;
        version_nueva := 1;
        huella_anterior := '';
        confirmada_anterior := reserva.solicitada_en;
        accion_auditoria := 'crear_baremacion';
        tipo_evento := 'bolsa.baremacion_creada.v1';
    ELSE
        IF NOT FOUND OR actual_baremacion.sujeto_ref IS DISTINCT FROM
           p_prueba ->> 'sujeto_ref' THEN
            resultado := 'no_encontrada';
            RETURN NEXT;
            RETURN;
        END IF;
        IF actual_baremacion.numero IS DISTINCT FROM version_esperada
           OR actual_baremacion.huella_estado_sha256 IS DISTINCT FROM
              huella_esperada THEN
            resultado := 'conflicto_version';
            RETURN NEXT;
            RETURN;
        END IF;
        agregado_anterior := actual_baremacion.agregado;
        confirmada_anterior := actual_baremacion.confirmada_en;
        IF (agregado - 'decisiones') IS DISTINCT FROM
               (agregado_anterior - 'decisiones')
           OR jsonb_array_length(agregado -> 'decisiones') < 1 THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT COALESCE(jsonb_agg(elemento ORDER BY orden), '[]'::jsonb)
          INTO prefijo_decisiones
          FROM jsonb_array_elements(agregado -> 'decisiones')
               WITH ORDINALITY AS d(elemento, orden)
         WHERE orden <= jsonb_array_length(
             agregado_anterior -> 'decisiones'
         );
        IF prefijo_decisiones IS DISTINCT FROM
           agregado_anterior -> 'decisiones' THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        ultima_decision := agregado -> 'decisiones'
            -> (jsonb_array_length(agregado -> 'decisiones') - 1);
        IF jsonb_typeof(ultima_decision) <> 'object'
           OR ultima_decision -> 'contenido' ->> 'baremacion_merito_ref'
              IS DISTINCT FROM agregado ->> 'id'
           OR ultima_decision -> 'contenido' ->> 'proceso_ref'
              IS DISTINCT FROM agregado ->> 'proceso_ref'
           OR ultima_decision -> 'contenido' ->> 'solicitud_ref'
              IS DISTINCT FROM agregado ->> 'solicitud_ref'
           OR ultima_decision -> 'contenido' ->> 'sujeto_ref'
              IS DISTINCT FROM agregado ->> 'sujeto_ref'
           OR (ultima_decision -> 'contenido'
               ->> 'version_anterior_baremacion')::numeric
              IS DISTINCT FROM version_esperada
           OR (ultima_decision -> 'contenido'
               ->> 'version_baremacion')::numeric
              IS DISTINCT FROM version_esperada + 1
           OR ultima_decision -> 'contenido' ->> 'decisor_ref'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'principal_id'
           OR ultima_decision -> 'contenido' ->> 'perfil_decisor_clave'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'perfil_activo_ref'
           OR ultima_decision -> 'contenido' ->> 'autorizacion_ref' =
              p_prueba ->> 'decision_ref'
           OR ultima_decision -> 'contenido' ->> 'finalidad_clave'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'finalidad'
           OR ultima_decision -> 'contenido' ->> 'correlacion_ref'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'correlacion_ref'
           OR vec_bolsa_baremacion.texto_opaco_valido(
               ultima_decision -> 'contenido' ->> 'id', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.huella_sha256_valida(
               ultima_decision ->> 'huella_sha256'
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.texto_opaco_valido(
               ultima_decision -> 'firma'
                   ->> 'documento_firmado_custodiado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.texto_opaco_valido(
               ultima_decision -> 'firma'
                   ->> 'evidencia_custodia_documento_firmado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.texto_opaco_valido(
               ultima_decision -> 'firma'
                   ->> 'evidencia_retencion_documento_firmado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.huella_sha256_valida(
               ultima_decision -> 'firma'
                   ->> 'huella_manifiesto_probatorio_sha256'
           ) IS NOT TRUE THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        version_anterior := version_esperada;
        version_nueva := version_esperada + 1;
        huella_anterior := huella_esperada;
        accion_auditoria := 'incorporar_decision_baremacion';
        tipo_evento := 'bolsa.decision_baremacion_incorporada.v1';
        decision_tecnica_ref := ultima_decision -> 'contenido' ->> 'id';
        manifiesto_ref := ultima_decision -> 'firma'
            ->> 'manifiesto_probatorio_ref';
        huella_manifiesto := ultima_decision -> 'firma'
            ->> 'huella_manifiesto_probatorio_sha256';
        documento_firmado_ref := ultima_decision -> 'firma'
            ->> 'documento_firmado_custodiado_ref';
        evidencia_custodia_ref := ultima_decision -> 'firma'
            ->> 'evidencia_custodia_documento_firmado_ref';
        evidencia_retencion_ref := ultima_decision -> 'firma'
            ->> 'evidencia_retencion_documento_firmado_ref';
    END IF;

    IF solicitada_confirmacion < confirmada_anterior THEN
        resultado := 'historial_no_anexable';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:cadenas-transaccion:v1', 0
    ));
    SELECT secuencia, huella_registro_sha256
      INTO secuencia_auditoria, huella_auditoria_anterior
      FROM vec_bolsa_baremacion.auditoria
     ORDER BY secuencia DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND THEN
        secuencia_auditoria := 1;
        huella_auditoria_anterior := '';
    ELSE
        secuencia_auditoria := secuencia_auditoria + 1;
    END IF;
    SELECT secuencia, huella_registro_sha256
      INTO secuencia_evento, huella_evento_anterior
      FROM vec_bolsa_baremacion.evento_outbox
     ORDER BY secuencia DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND THEN
        secuencia_evento := 1;
        huella_evento_anterior := '';
    ELSE
        secuencia_evento := secuencia_evento + 1;
    END IF;
    IF secuencia_auditoria IS DISTINCT FROM secuencia_evento
       OR secuencia_auditoria > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'cadenas de auditoria y outbox divergentes';
    END IF;

    confirmada_en := instante;
    auditoria_ref := p_operacion ->> 'auditoria_ref';
    evento_outbox_ref := p_operacion ->> 'evento_outbox_ref';
    campos_unidos := vec_bolsa_baremacion.unir_textos_nul(campos);
    huella_auditoria := vec_bolsa_baremacion.huella_canonica_bytes(ARRAY[
        convert_to(auditoria_ref, 'UTF8'),
        convert_to(secuencia_auditoria::text, 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'principal_id', 'UTF8'),
        convert_to(p_prueba ->> 'sujeto_ref', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'perfil_activo_ref', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'metodo_observado', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'garantia_observada', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'garantia_minima', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'autenticacion_ref', 'UTF8'),
        convert_to(p_prueba ->> 'decision_ref', 'UTF8'),
        convert_to(accion_autorizada, 'UTF8'), convert_to('baremacion', 'UTF8'),
        convert_to(agregado ->> 'id', 'UTF8'), campos_unidos,
        convert_to(autorizacion.decision_canonica ->> 'finalidad', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'correlacion_ref', 'UTF8'),
        convert_to('bolsa', 'UTF8'), convert_to(accion_auditoria, 'UTF8'),
        convert_to(p_operacion ->> 'clase', 'UTF8'),
        convert_to(agregado ->> 'proceso_ref', 'UTF8'),
        convert_to(agregado ->> 'solicitud_ref', 'UTF8'),
        convert_to(agregado ->> 'id', 'UTF8'),
        convert_to(decision_tecnica_ref, 'UTF8'),
        convert_to(manifiesto_ref, 'UTF8'), convert_to(huella_manifiesto, 'UTF8'),
        convert_to(documento_firmado_ref, 'UTF8'),
        convert_to(evidencia_custodia_ref, 'UTF8'),
        convert_to(evidencia_retencion_ref, 'UTF8'),
        convert_to(version_anterior::text, 'UTF8'),
        convert_to(version_nueva::text, 'UTF8'),
        convert_to(huella_anterior, 'UTF8'),
        convert_to(p_operacion ->> 'huella_agregado_sha256', 'UTF8'),
        convert_to(p_operacion ->> 'motivo_clave', 'UTF8'),
        convert_to(p_operacion ->> 'motivo', 'UTF8'),
        convert_to(p_operacion ->> 'huella_solicitud_hmac', 'UTF8'),
        convert_to('correcto', 'UTF8'),
        convert_to(p_operacion ->> 'confirmada_en', 'UTF8'),
        convert_to(vec_bolsa_baremacion.instante_rfc3339nano(instante), 'UTF8'),
        convert_to(huella_auditoria_anterior, 'UTF8')
    ]);
    huella_evento := vec_bolsa_baremacion.huella_canonica(ARRAY[
        evento_outbox_ref, secuencia_evento::text, tipo_evento, 'pendiente',
        'bolsa', agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'principal_id',
        version_nueva::text, p_operacion ->> 'huella_agregado_sha256',
        auditoria_ref, huella_auditoria,
        autorizacion.decision_canonica ->> 'correlacion_ref',
        vec_bolsa_baremacion.instante_rfc3339nano(instante),
        huella_evento_anterior
    ]);

    INSERT INTO vec_bolsa_baremacion.auditoria (
        referencia, secuencia, principal_ref, sujeto_ref,
        perfil_actor_clave, metodo_autenticacion, nivel_autenticacion,
        garantia_minima, autenticacion_ref, autorizacion_ref,
        accion_autorizada, clase_recurso_autorizada,
        recurso_autorizado_ref, campos_permitidos, finalidad_clave,
        correlacion_ref, modulo, accion, clase_cambio, proceso_ref,
        solicitud_ref, baremacion_merito_ref, decision_ref,
        manifiesto_probatorio_ref, huella_manifiesto_sha256,
        documento_firmado_custodiado_ref,
        evidencia_custodia_firmado_ref, evidencia_retencion_firmado_ref,
        version_anterior, version_nueva, huella_anterior_sha256,
        huella_nueva_sha256, motivo_clave, motivo, huella_solicitud_hmac,
        resultado, solicitada_confirmacion_en,
        solicitada_confirmacion_canonica, registrada_en,
        huella_anterior_auditoria_sha256, huella_registro_sha256
    ) VALUES (
        auditoria_ref, secuencia_auditoria,
        autorizacion.decision_canonica ->> 'principal_id',
        p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'perfil_activo_ref',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'metodo_observado',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'garantia_observada',
        autorizacion.decision_canonica ->> 'garantia_minima',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'autenticacion_ref',
        p_prueba ->> 'decision_ref', accion_autorizada, 'baremacion',
        agregado ->> 'id', campos,
        autorizacion.decision_canonica ->> 'finalidad',
        autorizacion.decision_canonica ->> 'correlacion_ref',
        'bolsa', accion_auditoria, p_operacion ->> 'clase',
        agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, version_anterior, version_nueva,
        huella_anterior, p_operacion ->> 'huella_agregado_sha256',
        p_operacion ->> 'motivo_clave', p_operacion ->> 'motivo',
        p_operacion ->> 'huella_solicitud_hmac', 'correcto',
        solicitada_confirmacion, p_operacion ->> 'confirmada_en',
        instante, huella_auditoria_anterior,
        huella_auditoria
    );
    INSERT INTO vec_bolsa_baremacion.evento_outbox (
        referencia, secuencia, tipo, estado, modulo, proceso_ref,
        solicitud_ref, baremacion_merito_ref, decision_ref,
        manifiesto_probatorio_ref, huella_manifiesto_sha256,
        documento_firmado_ref, evidencia_custodia_firmado_ref,
        evidencia_retencion_firmado_ref, sujeto_ref, principal_ref,
        version_nueva, huella_nueva_sha256, auditoria_ref,
        huella_auditoria_sha256, correlacion_ref, registrada_en,
        huella_evento_anterior_sha256, huella_registro_sha256
    ) VALUES (
        evento_outbox_ref, secuencia_evento, tipo_evento, 'pendiente',
        'bolsa', agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'principal_id', version_nueva,
        p_operacion ->> 'huella_agregado_sha256', auditoria_ref,
        huella_auditoria,
        autorizacion.decision_canonica ->> 'correlacion_ref',
        instante, huella_evento_anterior, huella_evento
    );
    INSERT INTO vec_bolsa_baremacion.version_baremacion (
        baremacion_merito_ref, numero, huella_estado_sha256,
        agregado_canonico, agregado, sujeto_ref, proceso_ref,
        solicitud_ref, confirmada_en, reserva_ref, auditoria_ref,
        evento_outbox_ref
    ) VALUES (
        agregado ->> 'id', version_nueva,
        p_operacion ->> 'huella_agregado_sha256', p_agregado_canonico,
        agregado, p_prueba ->> 'sujeto_ref', agregado ->> 'proceso_ref',
        agregado ->> 'solicitud_ref', instante, reserva.reserva_ref,
        auditoria_ref, evento_outbox_ref
    );
    IF version_nueva = 1 THEN
        INSERT INTO vec_bolsa_baremacion.baremacion_actual (
            baremacion_merito_ref, numero, huella_estado_sha256,
            actualizada_en
        ) VALUES (
            agregado ->> 'id', 1,
            p_operacion ->> 'huella_agregado_sha256', instante
        );
    ELSE
        UPDATE vec_bolsa_baremacion.baremacion_actual
           SET numero = version_nueva,
               huella_estado_sha256 =
                   p_operacion ->> 'huella_agregado_sha256',
               actualizada_en = instante
         WHERE baremacion_merito_ref = agregado ->> 'id'
           AND numero = version_anterior
           AND huella_estado_sha256 = huella_anterior;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'OCC de baremacion perdido';
        END IF;
    END IF;
    INSERT INTO vec_bolsa_baremacion.reserva_version (
        reserva_ref, version, estado, ambito_idempotencia_sha256,
        principal_ref, sujeto_ref, vinculo_autenticacion_actor,
        baremacion_merito_ref, clase, version_esperada,
        huella_version_esperada_sha256, huella_solicitud_hmac,
        huella_efecto_reserva_sha256, decision_reserva_ref,
        huella_decision_reserva_sha256, solicitada_en, expira_en,
        huella_confirmacion_sha256, numero_version_confirmada,
        registrada_en
    ) SELECT reserva_ref, version + 1, 'confirmada',
        ambito_idempotencia_sha256, principal_ref, sujeto_ref,
        vinculo_autenticacion_actor, baremacion_merito_ref, clase,
        version_esperada, huella_version_esperada_sha256,
        huella_solicitud_hmac, huella_efecto_reserva_sha256,
        decision_reserva_ref, huella_decision_reserva_sha256,
        solicitada_en, expira_en,
        p_operacion ->> 'huella_efecto_sha256', version_nueva, instante
      FROM vec_bolsa_baremacion.reserva_version
     WHERE reserva_ref = reserva.reserva_ref AND version = reserva.version;
    UPDATE vec_bolsa_baremacion.reserva_actual
       SET version = reserva.version + 1, estado = 'confirmada'
     WHERE ambito_idempotencia_sha256 =
           reserva.ambito_idempotencia_sha256
       AND version = reserva.version AND estado = 'activa';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de reserva perdido';
    END IF;
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'confirmacion',
        auditoria_ref, autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );

    resultado := 'confirmada';
    numero_version := version_nueva::text;
    huella_estado_sha256 := p_operacion ->> 'huella_agregado_sha256';
    agregado_canonico := p_agregado_canonico;
    huella_auditoria_sha256 := huella_auditoria;
    huella_evento_outbox_sha256 := huella_evento;
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'colision';
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR check_violation
        OR foreign_key_violation OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        RETURN NEXT;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_baremacion.obtener_atestacion_actual_valida(
    jsonb, bytea, timestamptz
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.revalidar_operacion(
    jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.reservar_cambio(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.confirmar_cambio(
    jsonb, jsonb, bytea, bytea, bytea
) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_baremacion
    TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.reservar_cambio(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.confirmar_cambio(
    jsonb, jsonb, bytea, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
COMMIT;
