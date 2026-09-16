\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_bolsa_registro_accesos:migracion:000004', 0));
LOCK TABLE vec_bolsa_registro_accesos.registro_acceso
    IN SHARE ROW EXCLUSIVE MODE;

DO $historia$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
         WHERE action = 'administracion.autenticacion'
    ) THEN
        RAISE EXCEPTION 'T13/4 DOWN rechazado: existe historia de autenticacion'
            USING ERRCODE = '55000';
    END IF;
END
$historia$;

REVOKE ALL ON FUNCTION
    vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)
    FROM PUBLIC, vec_administracion_ejecutor;
DROP FUNCTION
    vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)
    RESTRICT;

CREATE FUNCTION vec_bolsa_registro_accesos.registrar_interno_v1(p_entrada jsonb)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    ahora timestamptz := transaction_timestamp();
    control record;
    politica record;
    existente record;
    secuencia_nueva bigint;
    entrada_huella text;
    canonico bytea;
    firma_nueva text;
    referencia_nueva text;
    ocurrido timestamptz;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR octet_length(p_entrada::text) > 131072
       OR NOT vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
           p_entrada, ARRAY[
               'actor_id', 'actor_profile', 'actor_roles',
               'represented_subject_id', 'auth_method', 'auth_assurance',
               'authorization_ref', 'purpose', 'action', 'module_id',
               'subject_ref', 'object_version', 'expediente_ref',
               'document_ref', 'rule_ref', 'reason', 'result',
               'before_hash', 'after_hash', 'correlation_ref',
               'metadata', 'occurred_at'
           ]
       )
       OR p_entrada ->> 'actor_id' !~
          '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR p_entrada ->> 'actor_id' =
          regexp_replace(
              p_entrada ->> 'actor_id', '[0-9a-f]{64}$', repeat('0', 64)
          )
       OR jsonb_typeof(p_entrada -> 'actor_roles') <> 'array'
       OR jsonb_array_length(p_entrada -> 'actor_roles') > 16
       OR jsonb_typeof(p_entrada -> 'metadata') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_entrada -> 'metadata')) > 16
       OR p_entrada ->> 'result' NOT IN ('permitido', 'denegado', 'error')
       OR p_entrada ->> 'correlation_ref' = ''
       OR p_entrada ->> 'occurred_at' !~ 'Z$'
       OR (p_entrada ->> 'object_version') !~ '^(0|[1-9][0-9]{0,15})$'
       OR EXISTS (
           SELECT 1
             FROM jsonb_array_elements_text(p_entrada -> 'actor_roles') AS x(rol)
            WHERE rol = '' OR rol ~ '[*?[:space:][:cntrl:]]'
       )
       OR (
           SELECT COALESCE(
               jsonb_agg(rol ORDER BY rol COLLATE "C"), '[]'::jsonb
           )
             FROM jsonb_array_elements_text(p_entrada -> 'actor_roles') AS x(rol)
       ) IS DISTINCT FROM p_entrada -> 'actor_roles'
       OR (
           SELECT count(*) <> count(DISTINCT rol)
             FROM jsonb_array_elements_text(p_entrada -> 'actor_roles') AS x(rol)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada T13 invalida';
    END IF;
    ocurrido := (p_entrada ->> 'occurred_at')::timestamptz;
    entrada_huella := encode(sha256(convert_to(p_entrada::text, 'UTF8')), 'hex');
    SELECT * INTO control
      FROM vec_bolsa_registro_accesos.control_cadena
     WHERE singleton
     FOR UPDATE;
    SELECT secuencia, entrada_huella_sha256 INTO existente
      FROM vec_bolsa_registro_accesos.registro_acceso
     WHERE correlation_ref = p_entrada ->> 'correlation_ref';
    IF FOUND THEN
        IF existente.entrada_huella_sha256 <> entrada_huella THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'colision de correlacion T13';
        END IF;
        RETURN vec_bolsa_registro_accesos.auditoria_json_v1(
            existente.secuencia
        );
    END IF;
    IF ocurrido < ahora - interval '5 minutes'
       OR ocurrido > ahora + interval '5 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'instante de acceso T13 fuera de ventana';
    END IF;
    SELECT p.* INTO politica
      FROM vec_bolsa_registro_accesos.politica_actual AS a
      JOIN vec_bolsa_registro_accesos.politica_retencion AS p
        ON p.version = a.version
     WHERE a.singleton;
    secuencia_nueva := control.ultima_secuencia + 1;
    referencia_nueva := 'acc_' || substr(encode(sha256(convert_to(
        (p_entrada ->> 'correlation_ref') || '|' || entrada_huella,
        'UTF8'
    )), 'hex'), 1, 40);
    canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.registro-accesos.cadena.v1',
        'secuencia', secuencia_nueva, 'registro_ref', referencia_nueva,
        'entrada', p_entrada, 'politica_version', politica.version,
        'retener_hasta', to_char(
            (ocurrido + make_interval(days => politica.retencion_dias))
                AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    )::text, 'UTF8');
    firma_nueva := encode(sha256(
        decode(control.ultima_firma, 'hex') || canonico
    ), 'hex');
    INSERT INTO vec_bolsa_registro_accesos.registro_acceso (
        secuencia, registro_ref, actor_id, actor_profile, actor_roles,
        represented_subject_id, auth_method, auth_assurance,
        authorization_ref, purpose, action, module_id, subject_ref,
        object_version, expediente_ref, document_ref, rule_ref, reason,
        result, before_hash, after_hash, correlation_ref, metadata,
        occurred_at, politica_version, retener_hasta, bloqueada_hasta,
        entrada_huella_sha256, registro_canonico, firma_anterior, firma,
        registrada_en
    ) VALUES (
        secuencia_nueva, referencia_nueva, p_entrada ->> 'actor_id',
        p_entrada ->> 'actor_profile', p_entrada -> 'actor_roles',
        p_entrada ->> 'represented_subject_id', p_entrada ->> 'auth_method',
        p_entrada ->> 'auth_assurance', p_entrada ->> 'authorization_ref',
        p_entrada ->> 'purpose', p_entrada ->> 'action',
        p_entrada ->> 'module_id', p_entrada ->> 'subject_ref',
        (p_entrada ->> 'object_version')::bigint,
        p_entrada ->> 'expediente_ref', p_entrada ->> 'document_ref',
        p_entrada ->> 'rule_ref', p_entrada ->> 'reason',
        p_entrada ->> 'result', p_entrada ->> 'before_hash',
        p_entrada ->> 'after_hash', p_entrada ->> 'correlation_ref',
        p_entrada -> 'metadata', ocurrido, politica.version,
        ocurrido + make_interval(days => politica.retencion_dias),
        politica.bloqueada_hasta, entrada_huella, canonico,
        control.ultima_firma, firma_nueva, ahora
    );
    UPDATE vec_bolsa_registro_accesos.control_cadena
       SET ultima_secuencia = secuencia_nueva,
           ultima_firma = firma_nueva, actualizada_en = ahora
     WHERE singleton;
    RETURN vec_bolsa_registro_accesos.auditoria_json_v1(secuencia_nueva);
END
$funcion$;
ALTER TABLE vec_bolsa_registro_accesos.registro_acceso
    DROP CONSTRAINT registro_acceso_actor_id_check;
ALTER TABLE vec_bolsa_registro_accesos.registro_acceso
    ADD CONSTRAINT registro_acceso_actor_id_check CHECK (
        actor_id ~
            '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
        AND split_part(actor_id, ':', 3) <> repeat('0', 64)
    );
-- Se conserva USAGE sobre el esquema: puede pertenecer a T13/2, T13/3 u otra
-- fachada posterior y no concede acceso a tablas ni funciones por sí solo.
COMMIT;
