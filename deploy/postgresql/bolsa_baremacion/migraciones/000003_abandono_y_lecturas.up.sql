-- Cierra el puerto durable de BaremacionMerito: abandono con tombstone y
-- lecturas sensibles. Toda lectura revalida la autorizacion viva y consume la
-- decision exacta; el reintento solo recupera la misma version inmutable.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_baremacion.abandonar_reserva(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    accion text;
    campos jsonb;
    autorizacion record;
    reserva record;
    uso record;
BEGIN
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 5
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'huella_token_sha256', 'clase',
           'baremacion_merito_ref', 'huella_efecto_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.abandono-postgresql.v1'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_token_sha256'
       ) IS NOT TRUE
       OR p_operacion ->> 'clase' NOT IN ('alta', 'incorporar_decision')
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN 'rechazada';
    END IF;

    IF p_operacion ->> 'clase' = 'alta' THEN
        accion := 'bolsa.baremacion.alta.abandonar';
        campos := '["reserva.alta"]'::jsonb;
    ELSE
        accion := 'bolsa.baremacion.decision.abandonar';
        campos := '["reserva.decision"]'::jsonb;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          accion, 'baremacion', p_operacion ->> 'baremacion_merito_ref',
          campos, instante
      );
    IF NOT FOUND THEN
        RETURN 'autorizacion_obsoleta';
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:abandono:' ||
            (p_operacion ->> 'huella_token_sha256'), 0
    ));
    SELECT actual.ambito_idempotencia_sha256, actual.reserva_ref,
           actual.version, actual.estado, version.principal_ref,
           version.sujeto_ref, version.vinculo_autenticacion_actor,
           version.baremacion_merito_ref, version.clase,
           version.solicitada_en, version.expira_en
      INTO reserva
      FROM vec_bolsa_baremacion.token_reserva AS token
      JOIN vec_bolsa_baremacion.reserva_actual AS actual
        ON actual.ambito_idempotencia_sha256 =
           token.ambito_idempotencia_sha256
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
       OR reserva.baremacion_merito_ref IS DISTINCT FROM
          p_operacion ->> 'baremacion_merito_ref'
       OR reserva.clase IS DISTINCT FROM p_operacion ->> 'clase' THEN
        RETURN 'reserva_invalida';
    END IF;

    SELECT * INTO uso
      FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    IF reserva.estado = 'abandonada' THEN
        IF FOUND
           AND uso.huella_decision_sha256 IS NOT DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           AND uso.huella_efecto_sha256 IS NOT DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           AND uso.tipo_efecto = 'abandono'
           AND uso.resultado_ref = reserva.reserva_ref THEN
            RETURN 'abandonada';
        END IF;
        RETURN 'idempotencia_reutilizada';
    END IF;
    IF FOUND THEN
        RETURN 'autorizacion_reutilizada';
    END IF;
    IF reserva.estado <> 'activa' THEN
        RETURN 'reserva_invalida';
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
        RETURN 'reserva_invalida';
    END IF;

    INSERT INTO vec_bolsa_baremacion.reserva_version (
        reserva_ref, version, estado, ambito_idempotencia_sha256,
        principal_ref, sujeto_ref, vinculo_autenticacion_actor,
        baremacion_merito_ref, clase, version_esperada,
        huella_version_esperada_sha256, huella_solicitud_hmac,
        huella_efecto_reserva_sha256, decision_reserva_ref,
        huella_decision_reserva_sha256, solicitada_en, expira_en,
        registrada_en
    ) SELECT reserva_ref, version + 1, 'abandonada',
        ambito_idempotencia_sha256, principal_ref, sujeto_ref,
        vinculo_autenticacion_actor, baremacion_merito_ref, clase,
        version_esperada, huella_version_esperada_sha256,
        huella_solicitud_hmac, huella_efecto_reserva_sha256,
        decision_reserva_ref, huella_decision_reserva_sha256,
        solicitada_en, expira_en, instante
      FROM vec_bolsa_baremacion.reserva_version
     WHERE reserva_ref = reserva.reserva_ref AND version = reserva.version;
    UPDATE vec_bolsa_baremacion.reserva_actual
       SET version = reserva.version + 1, estado = 'abandonada'
     WHERE ambito_idempotencia_sha256 = reserva.ambito_idempotencia_sha256
       AND version = reserva.version AND estado = 'activa';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de abandono perdido';
    END IF;
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'abandono',
        reserva.reserva_ref, autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );
    RETURN 'abandonada';
EXCEPTION
    WHEN unique_violation THEN
        RETURN 'colision';
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR check_violation
        OR foreign_key_violation OR no_data_found OR too_many_rows THEN
        RETURN 'rechazada';
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.obtener_version_vigente(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    autorizacion record;
    uso record;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 3
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'baremacion_merito_ref', 'huella_efecto_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.lectura-vigente-postgresql.v1'
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.vigente.consultar', 'baremacion',
          p_operacion ->> 'baremacion_merito_ref', '["baremacion"]'::jsonb,
          instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:uso-lectura:' ||
            (p_prueba ->> 'decision_ref'), 0
    ));
    SELECT * INTO uso FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref' FOR SHARE;
    IF FOUND THEN
        IF uso.huella_decision_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR uso.tipo_efecto <> 'lectura_vigente' THEN
            resultado := 'autorizacion_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT version.numero::text, version.huella_estado_sha256,
               version.agregado_canonico, version.confirmada_en,
               version.auditoria_ref
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref
          FROM vec_bolsa_baremacion.version_baremacion AS version
         WHERE version.auditoria_ref = uso.resultado_ref
           AND version.baremacion_merito_ref =
               p_operacion ->> 'baremacion_merito_ref'
           AND version.sujeto_ref = p_prueba ->> 'sujeto_ref';
        resultado := CASE WHEN FOUND THEN 'obtenida'
            ELSE 'evidencia_no_confiable' END;
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT version.numero::text, version.huella_estado_sha256,
           version.agregado_canonico, version.confirmada_en,
           version.auditoria_ref
      INTO numero_version, huella_estado_sha256, agregado_canonico,
           confirmada_en, auditoria_ref
      FROM vec_bolsa_baremacion.baremacion_actual AS actual
      JOIN vec_bolsa_baremacion.version_baremacion AS version
        ON version.baremacion_merito_ref = actual.baremacion_merito_ref
       AND version.numero = actual.numero
     WHERE actual.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
       AND version.sujeto_ref = p_prueba ->> 'sujeto_ref'
     FOR SHARE OF actual;
    IF NOT FOUND THEN
        resultado := 'no_encontrada';
        RETURN NEXT;
        RETURN;
    END IF;
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'lectura_vigente',
        auditoria_ref, autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );
    resultado := 'obtenida';
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

CREATE FUNCTION vec_bolsa_baremacion.obtener_version(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    numero numeric(20, 0);
    autorizacion record;
    uso record;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 4
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'baremacion_merito_ref', 'numero_version',
           'huella_efecto_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.lectura-version-postgresql.v1'
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR (p_operacion ->> 'numero_version') !~ '^[1-9][0-9]{0,19}$'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    numero := (p_operacion ->> 'numero_version')::numeric;
    IF numero > 18446744073709551615 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.version.consultar', 'baremacion',
          p_operacion ->> 'baremacion_merito_ref', '["baremacion"]'::jsonb,
          instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:uso-lectura:' ||
            (p_prueba ->> 'decision_ref'), 0
    ));
    SELECT * INTO uso FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref' FOR SHARE;
    IF FOUND THEN
        IF uso.huella_decision_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR uso.tipo_efecto <> 'lectura_version' THEN
            resultado := 'autorizacion_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT version.numero::text, version.huella_estado_sha256,
               version.agregado_canonico, version.confirmada_en,
               version.auditoria_ref
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref
          FROM vec_bolsa_baremacion.version_baremacion AS version
         WHERE version.auditoria_ref = uso.resultado_ref
           AND version.baremacion_merito_ref =
               p_operacion ->> 'baremacion_merito_ref'
           AND version.numero = numero
           AND version.sujeto_ref = p_prueba ->> 'sujeto_ref';
        resultado := CASE WHEN FOUND THEN 'obtenida'
            ELSE 'evidencia_no_confiable' END;
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT version.numero::text, version.huella_estado_sha256,
           version.agregado_canonico, version.confirmada_en,
           version.auditoria_ref
      INTO numero_version, huella_estado_sha256, agregado_canonico,
           confirmada_en, auditoria_ref
      FROM vec_bolsa_baremacion.version_baremacion AS version
     WHERE version.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
       AND version.numero = numero
       AND version.sujeto_ref = p_prueba ->> 'sujeto_ref'
     FOR SHARE;
    IF NOT FOUND THEN
        resultado := 'no_encontrada';
        RETURN NEXT;
        RETURN;
    END IF;
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'lectura_version',
        auditoria_ref, autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );
    resultado := 'obtenida';
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

CREATE FUNCTION vec_bolsa_baremacion.obtener_evidencia_transaccion(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_documento jsonb,
    evento_documento jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    numero numeric(20, 0);
    autorizacion record;
    uso record;
    uso_encontrado boolean := false;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 6
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'baremacion_merito_ref', 'numero_version',
           'auditoria_ref', 'evento_outbox_ref', 'huella_efecto_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.lectura-evidencia-postgresql.v1'
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR (p_operacion ->> 'numero_version') !~ '^[1-9][0-9]{0,19}$'
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'auditoria_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'evento_outbox_ref', 512
       ) IS NOT TRUE
       OR p_operacion ->> 'auditoria_ref' =
          p_operacion ->> 'evento_outbox_ref'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    numero := (p_operacion ->> 'numero_version')::numeric;
    IF numero > 18446744073709551615 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.transaccion.consultar', 'transaccion',
          p_operacion ->> 'auditoria_ref',
          '["auditoria","evento_outbox","evidencia_transaccion"]'::jsonb,
          instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:uso-lectura:' ||
            (p_prueba ->> 'decision_ref'), 0
    ));
    SELECT * INTO uso FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref' FOR SHARE;
    uso_encontrado := FOUND;
    IF uso_encontrado AND (
       uso.huella_decision_sha256 IS DISTINCT FROM
           p_prueba ->> 'huella_decision_sha256'
       OR uso.huella_efecto_sha256 IS DISTINCT FROM
           p_operacion ->> 'huella_efecto_sha256'
       OR uso.tipo_efecto <> 'lectura_evidencia'
       OR uso.resultado_ref IS DISTINCT FROM
          p_operacion ->> 'auditoria_ref') THEN
        resultado := 'autorizacion_reutilizada';
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT version.numero::text, version.huella_estado_sha256,
           version.agregado_canonico, version.confirmada_en,
           to_jsonb(auditoria), to_jsonb(evento)
      INTO numero_version, huella_estado_sha256, agregado_canonico,
           confirmada_en, auditoria_documento, evento_documento
      FROM vec_bolsa_baremacion.version_baremacion AS version
      JOIN vec_bolsa_baremacion.auditoria
        ON auditoria.referencia = version.auditoria_ref
      JOIN vec_bolsa_baremacion.evento_outbox AS evento
        ON evento.referencia = version.evento_outbox_ref
       AND evento.auditoria_ref = auditoria.referencia
       AND evento.huella_auditoria_sha256 =
           auditoria.huella_registro_sha256
     WHERE version.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
       AND version.numero = numero
       AND version.auditoria_ref = p_operacion ->> 'auditoria_ref'
       AND version.evento_outbox_ref =
           p_operacion ->> 'evento_outbox_ref'
       AND version.sujeto_ref = p_prueba ->> 'sujeto_ref'
       AND auditoria.sujeto_ref = version.sujeto_ref
       AND auditoria.baremacion_merito_ref = version.baremacion_merito_ref
       AND auditoria.version_nueva = version.numero
       AND auditoria.huella_nueva_sha256 = version.huella_estado_sha256
       AND auditoria.registrada_en = version.confirmada_en
       AND evento.sujeto_ref = version.sujeto_ref
       AND evento.baremacion_merito_ref = version.baremacion_merito_ref
       AND evento.version_nueva = version.numero
       AND evento.huella_nueva_sha256 = version.huella_estado_sha256
       AND evento.registrada_en = version.confirmada_en
     FOR SHARE OF version, auditoria, evento;
    IF NOT FOUND THEN
        resultado := 'no_encontrada';
        RETURN NEXT;
        RETURN;
    END IF;
    IF NOT uso_encontrado THEN
        INSERT INTO vec_bolsa_baremacion.uso_decision (
            decision_ref, esquema_huella_decision,
            huella_decision_sha256, huella_efecto_sha256, tipo_efecto,
            resultado_ref, atestacion_ref, atestacion_version, consumida_en
        ) VALUES (
            p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
            p_prueba ->> 'huella_decision_sha256',
            p_operacion ->> 'huella_efecto_sha256', 'lectura_evidencia',
            p_operacion ->> 'auditoria_ref', autorizacion.atestacion_ref,
            autorizacion.atestacion_version, instante
        );
    END IF;
    resultado := 'obtenida';
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

REVOKE ALL ON FUNCTION vec_bolsa_baremacion.abandonar_reserva(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.obtener_version_vigente(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.obtener_version(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.obtener_evidencia_transaccion(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.abandonar_reserva(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version_vigente(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_evidencia_transaccion(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
COMMIT;
