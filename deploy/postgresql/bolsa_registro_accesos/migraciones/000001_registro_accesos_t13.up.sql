-- Registro durable y consulta administrativa minimizada de accesos (T13).
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TIME ZONE 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_registro_accesos:migracion:v1', 0)
);
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_registro_accesos') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar T13';
    END IF;
END
$prevalidacion$;
CREATE SCHEMA vec_bolsa_registro_accesos
    AUTHORIZATION vec_bolsa_accesos_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_registro_accesos FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_accesos_propietario
    IN SCHEMA vec_bolsa_registro_accesos REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_accesos_propietario
    IN SCHEMA vec_bolsa_registro_accesos REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_accesos_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_accesos_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;
CREATE FUNCTION vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
    p_objeto jsonb, p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_objeto) = 'object'
       AND (
           SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
             FROM pg_catalog.jsonb_object_keys(p_objeto) AS claves(clave)
       ) = (
           SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
             FROM pg_catalog.unnest(p_claves) AS claves(clave)
       )
$funcion$;
CREATE FUNCTION vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(
    p_objeto jsonb, p_tipos jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_tipos) = 'object'
       AND vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
           p_objeto, ARRAY(
               SELECT clave
                 FROM pg_catalog.jsonb_object_keys(p_tipos) AS x(clave)
           )
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(p_tipos) AS x(clave, tipo)
            WHERE pg_catalog.jsonb_typeof(p_objeto -> clave)
                  IS DISTINCT FROM tipo
       )
$funcion$;
CREATE FUNCTION vec_bolsa_registro_accesos.huella_valor_filtro_v1(p_valor text)
RETURNS text
LANGUAGE sql
IMMUTABLE STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT encode(sha256(convert_to(p_valor, 'UTF8')), 'hex')
$funcion$;
CREATE FUNCTION vec_bolsa_registro_accesos.impedir_mutacion_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historial T13 inmutable: operacion rechazada';
END
$funcion$;
CREATE TABLE vec_bolsa_registro_accesos.politica_retencion (
    version bigint PRIMARY KEY CHECK (version > 0),
    politica_ref text NOT NULL UNIQUE
        CHECK (politica_ref ~ '^ret_[a-z0-9]{16,80}$'),
    retencion_dias integer NOT NULL CHECK (retencion_dias BETWEEN 30 AND 3650),
    bloqueo_legal boolean NOT NULL,
    bloqueada_hasta timestamptz,
    publicada_en timestamptz NOT NULL,
    huella_sha256 text NOT NULL UNIQUE
        CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'
               AND huella_sha256 <> repeat('0', 64)),
    CHECK (bloqueada_hasta IS NULL OR bloqueo_legal)
);
INSERT INTO vec_bolsa_registro_accesos.politica_retencion (
    version, politica_ref, retencion_dias, bloqueo_legal, bloqueada_hasta,
    publicada_en, huella_sha256
) VALUES (
    1, 'ret_inicialt13julio2026', 365, false, NULL,
    transaction_timestamp(),
    encode(sha256(convert_to(
        'vec.bolsa.registro-accesos.retencion.v1|1|365|false|',
        'UTF8'
    )), 'hex')
);
CREATE TABLE vec_bolsa_registro_accesos.politica_actual (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    version bigint NOT NULL REFERENCES
        vec_bolsa_registro_accesos.politica_retencion(version),
    actualizada_en timestamptz NOT NULL
);
INSERT INTO vec_bolsa_registro_accesos.politica_actual
    (singleton, version, actualizada_en)
VALUES (true, 1, transaction_timestamp());
CREATE TABLE vec_bolsa_registro_accesos.control_cadena (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultima_firma text NOT NULL
        CHECK (ultima_firma ~ '^[0-9a-f]{64}$'),
    actualizada_en timestamptz NOT NULL
);
INSERT INTO vec_bolsa_registro_accesos.control_cadena
    (singleton, ultima_secuencia, ultima_firma, actualizada_en)
VALUES (true, 0, repeat('0', 64), transaction_timestamp());
CREATE TABLE vec_bolsa_registro_accesos.registro_acceso (
    secuencia bigint PRIMARY KEY CHECK (secuencia > 0),
    registro_ref text NOT NULL UNIQUE CHECK (registro_ref ~ '^acc_[0-9a-f]{40}$'),
    actor_id text NOT NULL
        CHECK (actor_id ~
               '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
               AND split_part(actor_id, ':', 3) <> repeat('0', 64)),
    actor_profile text NOT NULL,
    actor_roles jsonb NOT NULL CHECK (jsonb_typeof(actor_roles) = 'array'),
    represented_subject_id text NOT NULL,
    auth_method text NOT NULL,
    auth_assurance text NOT NULL,
    authorization_ref text NOT NULL,
    purpose text NOT NULL,
    action text NOT NULL,
    module_id text NOT NULL,
    subject_ref text NOT NULL,
    object_version bigint NOT NULL CHECK (object_version >= 0),
    expediente_ref text NOT NULL,
    document_ref text NOT NULL,
    rule_ref text NOT NULL,
    reason text NOT NULL,
    result text NOT NULL CHECK (result IN ('permitido', 'denegado', 'error')),
    before_hash text NOT NULL,
    after_hash text NOT NULL,
    correlation_ref text NOT NULL UNIQUE,
    metadata jsonb NOT NULL CHECK (jsonb_typeof(metadata) = 'object'),
    occurred_at timestamptz NOT NULL,
    politica_version bigint NOT NULL REFERENCES
        vec_bolsa_registro_accesos.politica_retencion(version),
    retener_hasta timestamptz NOT NULL,
    bloqueada_hasta timestamptz,
    entrada_huella_sha256 text NOT NULL,
    registro_canonico bytea NOT NULL,
    firma_anterior text NOT NULL,
    firma text NOT NULL UNIQUE,
    registrada_en timestamptz NOT NULL,
    CHECK (entrada_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (firma_anterior ~ '^[0-9a-f]{64}$'),
    CHECK (firma ~ '^[0-9a-f]{64}$'),
    CHECK (firma = encode(sha256(
        decode(firma_anterior, 'hex') || registro_canonico
    ), 'hex'))
);
CREATE INDEX registro_acceso_actor_tiempo_idx
    ON vec_bolsa_registro_accesos.registro_acceso
       (actor_id, occurred_at DESC, secuencia DESC);
CREATE INDEX registro_acceso_recurso_tiempo_idx
    ON vec_bolsa_registro_accesos.registro_acceso
       (subject_ref, occurred_at DESC, secuencia DESC);
CREATE INDEX registro_acceso_expediente_tiempo_idx
    ON vec_bolsa_registro_accesos.registro_acceso
       (expediente_ref, occurred_at DESC, secuencia DESC)
    WHERE expediente_ref <> '';
-- Recibo de efecto, no copia ni autoridad de la decisión. Impide ejecutar dos
-- lecturas (potencialmente distintas) con una sola decisión/traza.
CREATE TABLE vec_bolsa_registro_accesos.consumo_efecto_consulta (
    decision_ref text PRIMARY KEY,
    correlacion_ref text NOT NULL UNIQUE,
    auditoria_secuencia bigint NOT NULL UNIQUE REFERENCES
        vec_bolsa_registro_accesos.registro_acceso(secuencia),
    recurso_ref text NOT NULL,
    finalidad text NOT NULL,
    consumida_en timestamptz NOT NULL
);
DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'politica_retencion', 'politica_actual', 'control_cadena',
        'registro_acceso', 'consumo_efecto_consulta'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_registro_accesos.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_registro_accesos.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY propietario_unico ON vec_bolsa_registro_accesos.%I '
            'TO vec_bolsa_accesos_propietario USING (true) WITH CHECK (true)',
            tabla
        );
    END LOOP;
END
$rls$;
DO $inmutabilidad$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'politica_retencion', 'registro_acceso', 'consumo_efecto_consulta'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER impedir_cambio BEFORE UPDATE OR DELETE ON '
            'vec_bolsa_registro_accesos.%I FOR EACH ROW EXECUTE FUNCTION '
            'vec_bolsa_registro_accesos.impedir_mutacion_v1()', tabla
        );
        EXECUTE format(
            'CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON '
            'vec_bolsa_registro_accesos.%I FOR EACH STATEMENT EXECUTE FUNCTION '
            'vec_bolsa_registro_accesos.impedir_mutacion_v1()', tabla
        );
    END LOOP;
END
$inmutabilidad$;
CREATE FUNCTION vec_bolsa_registro_accesos.auditoria_json_v1(p_secuencia bigint)
RETURNS jsonb
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT jsonb_build_object(
        'id', r.registro_ref, 'seq', r.secuencia,
        'integrity_algorithm', 'sha256-chain-v1',
        'prev_signature', r.firma_anterior, 'signature', r.firma,
        'actor_id', r.actor_id, 'actor_profile', r.actor_profile,
        'actor_roles', r.actor_roles,
        'represented_subject_id', r.represented_subject_id,
        'auth_method', r.auth_method, 'auth_assurance', r.auth_assurance,
        'authorization_ref', r.authorization_ref, 'purpose', r.purpose,
        'action', r.action, 'module_id', r.module_id,
        'subject_ref', r.subject_ref, 'object_version', r.object_version,
        'expediente_ref', r.expediente_ref, 'document_ref', r.document_ref,
        'rule_ref', r.rule_ref, 'reason', r.reason, 'result', r.result,
        'before_hash', r.before_hash, 'after_hash', r.after_hash,
        'correlation_ref', r.correlation_ref, 'metadata', r.metadata,
        'occurred_at', to_char(
            r.occurred_at AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    )
      FROM vec_bolsa_registro_accesos.registro_acceso AS r
     WHERE r.secuencia = p_secuencia
$funcion$;
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
CREATE FUNCTION vec_bolsa_registro_accesos.registrar_acceso_v1(p_entrada jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF NOT pg_has_role(
        session_user, 'vec_bolsa_accesos_registrador', 'member'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'registro T13 no autorizado';
    END IF;
    RETURN vec_bolsa_registro_accesos.registrar_interno_v1(p_entrada);
END
$funcion$;
CREATE FUNCTION vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(
    p_solicitud jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    filtro jsonb;
    auditoria jsonb;
    autorizacion jsonb;
    canonica bytea;
    recurso_canonico bytea;
    recurso jsonb;
    audit_confirmada jsonb;
    limite integer;
    version_objeto bigint;
    desde_inclusive timestamptz;
    hasta_exclusive timestamptz;
    cursor_secuencia bigint;
    corte_secuencia bigint;
    filas jsonb;
    siguiente text := '';
    ahora timestamptz := transaction_timestamp();
BEGIN
    IF p_solicitud IS NULL
       OR NOT pg_has_role(
        session_user, 'vec_bolsa_accesos_consultor', 'member'
       )
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR octet_length(p_solicitud::text) > 1048576
       OR NOT vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(
           p_solicitud, '{
               "version":"number", "filtro":"object",
               "auditoria":"object", "autorizacion":"object"
           }'::jsonb
       )
       OR p_solicitud ->> 'version' <> '1' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta T13 denegada';
    END IF;
    filtro := p_solicitud -> 'filtro';
    auditoria := p_solicitud -> 'auditoria';
    autorizacion := p_solicitud -> 'autorizacion';
    IF NOT vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(
           filtro, '{
               "version":"number", "actor_seudonimizado":"string",
               "module_id":"string", "accion":"string",
               "finalidad_acceso":"string", "recurso_ref":"string",
               "expediente_ref":"string", "resultado":"string",
               "desde_inclusive":"string", "hasta_exclusive":"string",
               "version_objeto":"number", "limite":"number",
               "cursor":"string", "finalidad_consulta":"string"
           }'::jsonb
       )
       OR filtro ->> 'version' <> '1'
       OR (filtro ->> 'limite') !~ '^[1-9][0-9]{0,2}$'
       OR (filtro ->> 'version_objeto') !~ '^(0|[1-9][0-9]{0,15})$'
       OR filtro ->> 'desde_inclusive' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR filtro ->> 'hasta_exclusive' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR (filtro ->> 'actor_seudonimizado' = ''
           AND filtro ->> 'recurso_ref' = ''
           AND filtro ->> 'expediente_ref' = '')
       OR filtro ->> 'actor_seudonimizado' !~
          '^(|hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64})$'
       OR (
           filtro ->> 'actor_seudonimizado' <> ''
           AND split_part(
               filtro ->> 'actor_seudonimizado', ':', 3
           ) = repeat('0', 64)
       )
       OR filtro ->> 'resultado' !~ '^(|permitido|denegado|error)$'
       OR filtro ->> 'cursor' !~ '^(|cursor:v1:[0-9a-f]{64})$'
       OR filtro ->> 'module_id' !~ '^[^*?[:space:][:cntrl:]]{0,128}$'
       OR filtro ->> 'accion' !~ '^[^*?[:space:][:cntrl:]]{0,160}$'
       OR filtro ->> 'finalidad_acceso' !~
          '^[^*?[:space:][:cntrl:]]{0,128}$'
       OR octet_length(filtro ->> 'recurso_ref') > 512
       OR filtro ->> 'recurso_ref' !~ '^[^*?[:space:][:cntrl:]]*$'
       OR octet_length(filtro ->> 'expediente_ref') > 512
       OR filtro ->> 'expediente_ref' !~ '^[^*?[:space:][:cntrl:]]*$'
       OR filtro ->> 'finalidad_consulta' !~
          '^[^*?[:space:][:cntrl:]]{1,128}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'filtro T13 invalido';
    END IF;
    BEGIN
        limite := (filtro ->> 'limite')::integer;
        version_objeto := (filtro ->> 'version_objeto')::bigint;
        desde_inclusive := (filtro ->> 'desde_inclusive')::timestamptz;
        hasta_exclusive := (filtro ->> 'hasta_exclusive')::timestamptz;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
            OR datetime_field_overflow THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'filtro T13 invalido';
    END;
    IF limite > 100 OR version_objeto > 9007199254740991
       OR desde_inclusive >= hasta_exclusive
       OR hasta_exclusive - desde_inclusive > interval '31 days'
       OR to_char(
              desde_inclusive AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
          ) IS DISTINCT FROM filtro ->> 'desde_inclusive'
       OR to_char(
              hasta_exclusive AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
          ) IS DISTINCT FROM filtro ->> 'hasta_exclusive' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'filtro T13 invalido';
    END IF;
    IF NOT vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(
           auditoria, '{
               "actor_id":"string", "actor_profile":"string",
               "actor_roles":"array", "represented_subject_id":"string",
               "auth_method":"string", "auth_assurance":"string",
               "authorization_ref":"string", "purpose":"string",
               "action":"string", "module_id":"string",
               "subject_ref":"string", "object_version":"number",
               "expediente_ref":"string", "document_ref":"string",
               "rule_ref":"string", "reason":"string", "result":"string",
               "before_hash":"string", "after_hash":"string",
               "correlation_ref":"string", "metadata":"object",
               "occurred_at":"string"
           }'::jsonb
       )
       OR auditoria ->> 'actor_id' !~
          '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR split_part(auditoria ->> 'actor_id', ':', 3) = repeat('0', 64)
       OR auditoria ->> 'actor_profile' !~
          '^[^*?[:space:][:cntrl:]]{1,160}$'
       OR jsonb_array_length(auditoria -> 'actor_roles') = 0
       OR EXISTS (
           SELECT 1 FROM jsonb_array_elements(
               auditoria -> 'actor_roles'
           ) AS rol
           WHERE jsonb_typeof(rol) <> 'string'
       )
       OR auditoria ->> 'represented_subject_id' <> ''
       OR auditoria ->> 'auth_method' NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR auditoria ->> 'auth_assurance' <> 'alto'
       OR auditoria ->> 'authorization_ref' !~
          '^[^*[:space:][:cntrl:]]{1,160}$'
       OR auditoria ->> 'purpose'
          IS DISTINCT FROM filtro ->> 'finalidad_consulta'
       OR auditoria ->> 'action' <> 'bolsa.registro_accesos.consultar'
       OR auditoria ->> 'module_id' <> 'vec.module.bolsa'
       OR auditoria ->> 'subject_ref' !~
          '^consulta-accesos:sha256:[0-9a-f]{64}$'
       OR auditoria ->> 'object_version' <> '1'
       OR auditoria ->> 'expediente_ref' <> ''
       OR auditoria ->> 'document_ref' <> ''
       OR auditoria ->> 'rule_ref' <> ''
       OR auditoria ->> 'reason' <> ''
       OR auditoria ->> 'result' <> 'permitido'
       OR auditoria ->> 'before_hash' <> ''
       OR auditoria ->> 'after_hash' <> ''
       OR auditoria ->> 'correlation_ref' !~ '^correlacion_[0-9a-f]{32}$'
       OR auditoria -> 'metadata' IS DISTINCT FROM '{}'::jsonb
       OR auditoria ->> 'occurred_at' !~ 'Z$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de consulta T13 invalida';
    END IF;
    IF NOT vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
           autorizacion,
           ARRAY['prueba', 'decision_canonica', 'recurso_canonico']
       )
       OR NOT vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
           autorizacion -> 'prueba', ARRAY[
               'esquema_huella', 'decision_ref',
               'huella_decision_sha256', 'verificada_en', 'principal_ref'
           ]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'prueba PDP T13 malformada';
    END IF;
    BEGIN
        canonica := decode(autorizacion ->> 'decision_canonica', 'base64');
        recurso_canonico :=
            decode(autorizacion ->> 'recurso_canonico', 'base64');
        recurso := convert_from(recurso_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
            OR character_not_in_repertoire THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'prueba PDP T13 no decodificable';
    END;
    IF recurso -> 'ambitos' ->> 'version_esquema'
          IS DISTINCT FROM filtro ->> 'version'
       OR recurso -> 'atributos' ->> 'actor_operador_seudonimizado'
          IS DISTINCT FROM auditoria ->> 'actor_id'
       OR recurso -> 'atributos' ->> 'filtro_version_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'version')
       OR recurso -> 'atributos' ->> 'filtro_actor_seudonimizado_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'actor_seudonimizado')
       OR recurso -> 'atributos' ->> 'filtro_modulo_id_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'module_id')
       OR recurso -> 'atributos' ->> 'filtro_accion_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'accion')
       OR recurso -> 'atributos' ->> 'filtro_finalidad_acceso_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'finalidad_acceso')
       OR recurso -> 'atributos' ->> 'filtro_recurso_ref_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'recurso_ref')
       OR recurso -> 'atributos' ->> 'filtro_expediente_ref_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'expediente_ref')
       OR recurso -> 'atributos' ->> 'filtro_resultado_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'resultado')
       OR recurso -> 'atributos' ->> 'filtro_desde_inclusive_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'desde_inclusive')
       OR recurso -> 'atributos' ->> 'filtro_hasta_exclusive_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'hasta_exclusive')
       OR recurso -> 'atributos' ->> 'filtro_version_objeto_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'version_objeto')
       OR recurso -> 'atributos' ->> 'filtro_limite_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'limite')
       OR recurso -> 'atributos' ->> 'filtro_cursor_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'cursor')
       OR recurso -> 'atributos' ->> 'filtro_finalidad_consulta_sha256' IS DISTINCT FROM vec_bolsa_registro_accesos.huella_valor_filtro_v1(filtro ->> 'finalidad_consulta')
       OR recurso -> 'atributos' ->> 'finalidad_consulta'
          IS DISTINCT FROM filtro ->> 'finalidad_consulta'
       OR autorizacion -> 'prueba' ->> 'decision_ref'
          IS DISTINCT FROM auditoria ->> 'authorization_ref'
       OR vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
           autorizacion -> 'prueba', canonica, recurso_canonico,
           auditoria ->> 'correlation_ref', auditoria ->> 'subject_ref',
           filtro ->> 'finalidad_consulta', '[
               "accion", "actor_seudonimizado", "expediente_ref",
               "finalidad", "modulo_id", "ocurrido_en", "recurso_ref",
               "resultado", "version_esquema", "version_objeto"
           ]'::jsonb, auditoria ->> 'actor_id',
           auditoria ->> 'auth_method', auditoria ->> 'auth_assurance'
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'decision VEC durable T13 no revalidada';
    END IF;
    audit_confirmada :=
        vec_bolsa_registro_accesos.registrar_interno_v1(auditoria);
    corte_secuencia := (audit_confirmada ->> 'seq')::bigint;
    BEGIN
        INSERT INTO vec_bolsa_registro_accesos.consumo_efecto_consulta (
            decision_ref, correlacion_ref, auditoria_secuencia,
            recurso_ref, finalidad, consumida_en
        ) VALUES (
            autorizacion -> 'prueba' ->> 'decision_ref',
            auditoria ->> 'correlation_ref',
            (audit_confirmada ->> 'seq')::bigint,
            auditoria ->> 'subject_ref',
            filtro ->> 'finalidad_consulta', clock_timestamp()
        );
    EXCEPTION WHEN unique_violation THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'decision T13 ya consumida; solicite nueva autorizacion';
    END;
    cursor_secuencia := NULL;
    IF filtro ->> 'cursor' <> '' THEN
        SELECT secuencia INTO cursor_secuencia
          FROM vec_bolsa_registro_accesos.registro_acceso
         WHERE firma = substr(filtro ->> 'cursor', 11);
        IF cursor_secuencia IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'cursor T13 desconocido';
        END IF;
    END IF;
    WITH candidatas AS MATERIALIZED (
        SELECT r.*
          FROM vec_bolsa_registro_accesos.registro_acceso AS r
         WHERE r.occurred_at >= desde_inclusive
           AND r.occurred_at < hasta_exclusive
           -- La lectura usa la frontera anterior a su propio asiento. En
           -- páginas siguientes, el cursor reduce aún más esa frontera y
           -- excluye cualquier evento incorporado entre páginas.
           AND r.secuencia < corte_secuencia
           AND (cursor_secuencia IS NULL OR r.secuencia < cursor_secuencia)
           AND (filtro ->> 'actor_seudonimizado' = ''
                OR r.actor_id = filtro ->> 'actor_seudonimizado')
           AND (filtro ->> 'module_id' = ''
                OR r.module_id = filtro ->> 'module_id')
           AND (filtro ->> 'accion' = ''
                OR r.action = filtro ->> 'accion')
           AND (filtro ->> 'finalidad_acceso' = ''
                OR r.purpose = filtro ->> 'finalidad_acceso')
           AND (filtro ->> 'recurso_ref' = ''
                OR r.subject_ref = filtro ->> 'recurso_ref')
           AND (filtro ->> 'expediente_ref' = ''
                OR r.expediente_ref = filtro ->> 'expediente_ref')
           AND (filtro ->> 'resultado' = ''
                OR r.result = filtro ->> 'resultado')
           AND (version_objeto = 0 OR r.object_version = version_objeto)
         ORDER BY r.secuencia DESC
         LIMIT limite + 1
    ), visibles AS MATERIALIZED (
        SELECT * FROM candidatas ORDER BY secuencia DESC LIMIT limite
    )
    SELECT COALESCE(jsonb_agg(jsonb_build_object(
               'registro_ref', registro_ref,
               'actor_seudonimizado', actor_id, 'modulo_id', module_id,
               'accion', action, 'finalidad', purpose,
               'recurso_ref', subject_ref, 'expediente_ref', expediente_ref,
               'resultado', result, 'version_objeto', object_version,
               'ocurrido_en', to_char(
                   occurred_at AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
               ), 'version_esquema', 1
           ) ORDER BY secuencia DESC), '[]'::jsonb),
           CASE WHEN (SELECT count(*) FROM candidatas) > limite
                THEN 'cursor:v1:' || (
                    SELECT firma FROM visibles ORDER BY secuencia LIMIT 1
                )
                ELSE '' END
      INTO filas, siguiente
      FROM visibles;
    RETURN jsonb_build_object(
        'auditoria', audit_confirmada,
        'registros', filas,
        'siguiente', siguiente
    );
END
$funcion$;
CREATE FUNCTION vec_bolsa_registro_accesos.publicar_politica_retencion_v1(
    p_politica jsonb
)
RETURNS bigint
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    actual bigint;
    nueva bigint;
    huella text;
BEGIN
    IF NOT pg_has_role(
        session_user, 'vec_bolsa_accesos_gobernador', 'member'
    ) OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR NOT vec_bolsa_registro_accesos.objeto_claves_exactas_v1(
           p_politica, ARRAY[
               'politica_ref', 'retencion_dias', 'bloqueo_legal',
               'bloqueada_hasta'
           ]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'gobierno de retencion T13 denegado';
    END IF;
    SELECT version INTO actual
      FROM vec_bolsa_registro_accesos.politica_actual
     WHERE singleton FOR UPDATE;
    nueva := actual + 1;
    huella := encode(sha256(convert_to(p_politica::text, 'UTF8')), 'hex');
    INSERT INTO vec_bolsa_registro_accesos.politica_retencion (
        version, politica_ref, retencion_dias, bloqueo_legal,
        bloqueada_hasta, publicada_en, huella_sha256
    ) VALUES (
        nueva, p_politica ->> 'politica_ref',
        (p_politica ->> 'retencion_dias')::integer,
        (p_politica ->> 'bloqueo_legal')::boolean,
        NULLIF(p_politica ->> 'bloqueada_hasta', '')::timestamptz,
        transaction_timestamp(), huella
    );
    UPDATE vec_bolsa_registro_accesos.politica_actual
       SET version = nueva, actualizada_en = transaction_timestamp()
     WHERE singleton;
    RETURN nueva;
END
$funcion$;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_registro_accesos FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_registro_accesos FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_registro_accesos FROM PUBLIC;
REVOKE ALL ON SCHEMA vec_bolsa_registro_accesos FROM PUBLIC;
-- Este corte solo expone la consulta administrativa, que revalida una
-- decisión VEC durable y registra su propio acceso en la misma transacción.
-- El append genérico y el gobierno de retención permanecen privados al
-- propietario hasta disponer de wrappers VEC ligados al efecto gobernado.
GRANT USAGE ON SCHEMA vec_bolsa_registro_accesos
    TO vec_bolsa_accesos_consultor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(jsonb)
    TO vec_bolsa_accesos_consultor;
COMMIT;
