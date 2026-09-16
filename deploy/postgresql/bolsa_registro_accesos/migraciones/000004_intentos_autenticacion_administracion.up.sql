\set ON_ERROR_STOP on
-- T13/4: intentos de autenticación ADMIN en la autoridad, cadena y retención
-- existentes. No almacena material de certificado, sujeto, red ni error bruto.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_bolsa_registro_accesos:migracion:000004', 0));
LOCK TABLE vec_bolsa_registro_accesos.registro_acceso
    IN SHARE ROW EXCLUSIVE MODE;

DO $dependencias$
DECLARE
    propietario oid := 'vec_bolsa_accesos_propietario'::regrole;
    ejecutor oid := 'vec_administracion_ejecutor'::regrole;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_namespace
         WHERE nspname = 'vec_bolsa_registro_accesos'
           AND nspowner = propietario
    ) OR to_regprocedure(
        'vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)'
    ) IS NOT NULL
      OR NOT EXISTS (
          SELECT 1 FROM pg_proc
           WHERE oid = to_regprocedure(
               'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)'
           ) AND proowner = propietario AND NOT prosecdef
             AND proconfig = ARRAY['search_path=pg_catalog']
      ) OR NOT EXISTS (
          SELECT 1 FROM pg_roles
           WHERE oid = ejecutor AND NOT rolcanlogin AND NOT rolsuper
             AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole
             AND NOT rolreplication AND rolinherit
      ) OR EXISTS (
          SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
           WHERE actor_id = ''
      ) THEN
        RAISE EXCEPTION 'T13/4: dependencias incompatibles'
            USING ERRCODE = '55000';
    END IF;
END
$dependencias$;

ALTER TABLE vec_bolsa_registro_accesos.registro_acceso
    DROP CONSTRAINT registro_acceso_actor_id_check;
ALTER TABLE vec_bolsa_registro_accesos.registro_acceso
    ADD CONSTRAINT registro_acceso_actor_id_check CHECK (
        (
            actor_id ~
                '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
            AND split_part(actor_id, ':', 3) <> repeat('0', 64)
        ) OR (
            actor_id = ''
            AND action = 'administracion.autenticacion'
            AND module_id = 'vec.module.administracion'
            AND metadata ->> 'actor_estado' = 'no_acreditado'
            AND metadata ->> 'superficie' = 'administracion_privilegiada'
        )
    );

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
       OR NOT (
           (
               p_entrada ->> 'actor_id' ~
                   '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
               AND p_entrada ->> 'actor_id' <>
                   regexp_replace(
                       p_entrada ->> 'actor_id', '[0-9a-f]{64}$', repeat('0', 64)
                   )
           ) OR (
               p_entrada ->> 'actor_id' = ''
               AND p_entrada ->> 'action' = 'administracion.autenticacion'
               AND p_entrada ->> 'module_id' = 'vec.module.administracion'
               AND vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
                   p_entrada -> 'metadata',
                   ARRAY['actor_estado', 'intento_ref', 'superficie']
               )
               AND p_entrada -> 'metadata' ->> 'actor_estado' = 'no_acreditado'
               AND p_entrada -> 'metadata' ->> 'intento_ref' ~
                   '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
               AND split_part(
                   p_entrada -> 'metadata' ->> 'intento_ref', ':', 3
               ) <> repeat('0', 64)
               AND p_entrada -> 'metadata' ->> 'superficie' =
                   'administracion_privilegiada'
           )
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
-- La frontera entrega exactamente el contrato de dominio V1 ya validado. La
-- función vuelve a cerrarlo en SQL y lo traduce al formato canónico T13. El rol
-- runtime solo recibe esta capacidad nominal; registrar_interno_v1 y
-- registrar_acceso_v1 continúan cerradas.
CREATE FUNCTION vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(
    p_intento jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s' SET row_security = on
AS $funcion$
DECLARE
    actor_estado text;
    hmac_actor text;
    motivo text;
    resultado text;
    entrada_t13 jsonb;
BEGIN
    IF NOT pg_has_role(
           session_user, 'vec_administracion_ejecutor', 'MEMBER'
       ) OR pg_has_role(
           session_user, 'vec_administracion_propietario', 'MEMBER'
       ) OR pg_has_role(
           session_user, 'vec_administracion_migrador', 'MEMBER'
       ) OR pg_has_role(
           session_user, 'vec_bolsa_accesos_propietario', 'MEMBER'
       ) OR pg_has_role(
           session_user, 'vec_bolsa_accesos_migrador', 'MEMBER'
       ) OR current_setting('transaction_isolation') <> 'serializable'
         OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'T13/4: registro de intento no autorizado'
            USING ERRCODE = '42501';
    END IF;
    IF p_intento IS NULL OR octet_length(p_intento::text) > 4096
       OR vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(
           p_intento, '{
               "intento_ref":"string","correlacion_ref":"string",
               "instante_utc":"string","superficie":"string",
               "resultado":"string","motivo":"string",
               "actor_estado":"string","hmac_actor":"string"
           }'::jsonb
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'T13/4: intento incompleto' USING ERRCODE = '22023';
    END IF;

    actor_estado := p_intento ->> 'actor_estado';
    hmac_actor := p_intento ->> 'hmac_actor';
    motivo := p_intento ->> 'motivo';
    resultado := p_intento ->> 'resultado';
    IF p_intento ->> 'intento_ref' !~
           '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR split_part(p_intento ->> 'intento_ref', ':', 3) = repeat('0', 64)
       OR p_intento ->> 'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
       OR p_intento ->> 'correlacion_ref' =
           'correlacion_' || repeat('0', 32)
       OR p_intento ->> 'instante_utc' !~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$'
       OR p_intento ->> 'superficie' <> 'administracion_privilegiada'
       OR actor_estado NOT IN ('acreditado', 'no_acreditado')
       OR resultado NOT IN ('permitido', 'denegado', 'error')
       OR motivo NOT IN (
           'permiso_concedido', 'certificado_ausente',
           'certificado_no_verificado', 'certificado_revocado',
           'sesion_revocada', 'permiso_denegado', 'canal_no_valido',
           'error_infraestructura'
       ) OR (
           actor_estado = 'acreditado' AND (
               hmac_actor !~
                   '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
               OR split_part(hmac_actor, ':', 3) = repeat('0', 64)
           )
       ) OR (actor_estado = 'no_acreditado' AND hmac_actor <> '')
       OR NOT (
           (resultado = 'permitido' AND motivo = 'permiso_concedido'
                AND actor_estado = 'acreditado')
           OR (resultado = 'denegado'
                AND motivo IN (
                    'certificado_ausente', 'certificado_no_verificado',
                    'certificado_revocado', 'canal_no_valido'
                ) AND actor_estado = 'no_acreditado')
           OR (resultado = 'denegado'
                AND motivo IN ('sesion_revocada', 'permiso_denegado')
                AND actor_estado = 'acreditado')
           OR (resultado = 'error' AND motivo = 'error_infraestructura')
       ) THEN
        RAISE EXCEPTION 'T13/4: intento invalido' USING ERRCODE = '22023';
    END IF;

    entrada_t13 := jsonb_build_object(
        'actor_id', hmac_actor,
        'actor_profile', '',
        'actor_roles', '[]'::jsonb,
        'represented_subject_id', '',
        'auth_method', '',
        'auth_assurance', '',
        'authorization_ref', '',
        'purpose', 'administrar_integraciones',
        'action', 'administracion.autenticacion',
        'module_id', 'vec.module.administracion',
        'subject_ref', 'superficie:administracion_privilegiada',
        'object_version', 1,
        'expediente_ref', '',
        'document_ref', '',
        'rule_ref', '',
        'reason', motivo,
        'result', resultado,
        'before_hash', '',
        'after_hash', '',
        'correlation_ref', p_intento ->> 'correlacion_ref',
        'metadata', jsonb_build_object(
            'actor_estado', actor_estado,
            'intento_ref', p_intento ->> 'intento_ref',
            'superficie', p_intento ->> 'superficie'
        ),
        'occurred_at', p_intento ->> 'instante_utc'
    );
    RETURN vec_bolsa_registro_accesos.registrar_interno_v1(entrada_t13);
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_registro_accesos
    TO vec_administracion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)
    TO vec_administracion_ejecutor;

DO $acl_final$
DECLARE
    funcion oid := 'vec_bolsa_registro_accesos.registrar_intento_autenticacion_administracion_v1(jsonb)'::regprocedure;
    propietario oid := 'vec_bolsa_accesos_propietario'::regrole;
    ejecutor oid := 'vec_administracion_ejecutor'::regrole;
BEGIN
    IF NOT COALESCE((
        SELECT count(*) = 2 AND count(DISTINCT acl.grantee) = 2
               AND bool_and(
                   acl.grantee IN (propietario, ejecutor)
                   AND acl.grantor = propietario
                   AND acl.privilege_type = 'EXECUTE'
                   AND NOT acl.is_grantable
               )
          FROM pg_proc AS procedimiento
          CROSS JOIN LATERAL aclexplode(
              coalesce(
                  procedimiento.proacl,
                  acldefault('f', procedimiento.proowner)
              )
          ) AS acl
         WHERE procedimiento.oid = funcion
    ), false)
       OR has_function_privilege(
           'vec_administracion_ejecutor',
           'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_administracion_ejecutor',
           'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)',
           'EXECUTE'
       ) OR has_table_privilege(
           'vec_administracion_ejecutor',
           'vec_bolsa_registro_accesos.registro_acceso',
           'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
       ) THEN
        RAISE EXCEPTION 'T13/4: ACL final incompatible'
            USING ERRCODE = '55000';
    END IF;
END
$acl_final$;
COMMIT;
