-- Cierre transaccional de borradores con cifrado por sobre, procedencia y
-- acreditacion KMS. Esta migracion no convierte PostgreSQL en autoridad
-- criptografica: las firmas A/B se verifican en conectores independientes.
-- La base valida forma, compromisos, enlaces, CAS y atomicidad, y conserva
-- exclusivamente AAD no sensible, DEK envuelta, ciphertext y evidencias.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_convocatorias.confirmar_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.confirmar_borrador_interna_v1(jsonb,bytea,bytea,bytea)'
       ) IS NULL
       OR to_regclass(
           'vec_bolsa_convocatorias.cifrado_kms_borrador'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar confirmacion KMS de borradores';
    END IF;

    -- La frontera publica de 000003 nunca pudo confirmar. Si existen filas
    -- de negocio, proceden de una invocacion propietaria de laboratorio y no
    -- se inventa para ellas procedencia, AAD ni una DEK envuelta inexistentes.
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_convocatorias.borrador_convocatoria_version
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'hay borradores legados sin contrato KMS migrable';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_convocatorias_verificador_recibo'
           AND rolcanlogin IS FALSE
           AND rolinherit IS TRUE
           AND rolsuper IS FALSE
           AND rolcreatedb IS FALSE
           AND rolcreaterole IS FALSE
           AND rolreplication IS FALSE
           AND rolbypassrls IS FALSE
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS m
          JOIN pg_catalog.pg_roles AS r ON r.oid = m.member
         WHERE r.rolname = 'vec_bolsa_convocatorias_verificador_recibo'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol verificador de recibos inexistente o no endurecido';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_convocatorias.base64url_sin_relleno_valido(
    p_valor text, p_maximo_bytes integer
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    normalizado text;
    decodificado bytea;
BEGIN
    IF p_valor IS NULL OR p_maximo_bytes < 1
       OR octet_length(p_valor) NOT BETWEEN 2 AND 21846
       OR p_valor !~ '^[A-Za-z0-9_-]+$'
       OR length(p_valor) % 4 = 1 THEN
        RETURN false;
    END IF;
    normalizado := translate(p_valor, '-_', '+/') ||
        repeat('=', (4 - length(p_valor) % 4) % 4);
    BEGIN
        decodificado := decode(normalizado, 'base64');
    EXCEPTION WHEN OTHERS THEN
        RETURN false;
    END;
    RETURN octet_length(decodificado) BETWEEN 1 AND p_maximo_bytes
       AND rtrim(translate(
               replace(encode(decodificado, 'base64'), E'\n', ''),
               '+/', '-_'
           ), '=') = p_valor;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.procedencia_borrador_valida(
    p_procedencia jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_bolsa_convocatorias.objeto_json_exacto(
               p_procedencia, ARRAY[
                   'autoridad','esquema','migrable_produccion',
                   'perfil_ejecucion','proveedor_ref'
               ]
           ) IS TRUE
       AND p_procedencia ->> 'esquema' = 'vec.acto.procedencia.v1'
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_procedencia ->> 'perfil_ejecucion', 512
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_procedencia ->> 'proveedor_ref', 512
           ) IS TRUE
       AND jsonb_typeof(p_procedencia -> 'migrable_produccion') = 'boolean'
       AND CASE p_procedencia ->> 'autoridad'
           WHEN 'autoritativo' THEN
               (p_procedencia ->> 'migrable_produccion')::boolean
               AND p_procedencia ->> 'perfil_ejecucion' <> 'desarrollo'
           WHEN 'no_autoritativo' THEN
               NOT (p_procedencia ->> 'migrable_produccion')::boolean
               AND (
                   p_procedencia ->> 'perfil_ejecucion' <> 'desarrollo'
                   OR p_procedencia ->> 'proveedor_ref' =
                      'proveedor:seguridad:desarrollo:t21'
               )
           ELSE false
       END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.perfil_cifrado_borrador_valido(
    p_perfil jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_bolsa_convocatorias.objeto_json_exacto(
               p_perfil, ARRAY[
                   'algoritmo_aead','algoritmo_envoltura_clave',
                   'huella_contenido_sha256','referencia','version'
               ]
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_perfil ->> 'referencia', 512
           ) IS TRUE
       AND (p_perfil ->> 'version') ~ '^[1-9][0-9]{0,9}$'
       AND vec_bolsa_convocatorias.huella_sha256_valida(
               p_perfil ->> 'huella_contenido_sha256'
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_perfil ->> 'algoritmo_aead', 128
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_perfil ->> 'algoritmo_envoltura_clave', 128
           ) IS TRUE
       AND p_perfil ->> 'algoritmo_aead' <>
           p_perfil ->> 'algoritmo_envoltura_clave'
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.firma_evidencia_borrador_valida(
    p_firma jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_bolsa_convocatorias.objeto_json_exacto(
               p_firma, ARRAY[
                   'algoritmo_firma','firma_base64url_sin_relleno',
                   'huella_clave_publica_sha256',
                   'huella_preimagen_sha256','verificador_ref'
               ]
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_firma ->> 'algoritmo_firma', 128
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_firma ->> 'verificador_ref', 512
           ) IS TRUE
       AND vec_bolsa_convocatorias.huella_sha256_valida(
               p_firma ->> 'huella_clave_publica_sha256'
           ) IS TRUE
       AND vec_bolsa_convocatorias.huella_sha256_valida(
               p_firma ->> 'huella_preimagen_sha256'
           ) IS TRUE
       AND vec_bolsa_convocatorias.base64url_sin_relleno_valido(
               p_firma ->> 'firma_base64url_sin_relleno', 16384
           ) IS TRUE
$funcion$;

-- Punto unico de configuracion operativo, propiedad de base de datos. El
-- runtime no recibe EXECUTE ni puede alterarlo mediante GUC. Produccion debe
-- versionar/reemplazar esta funcion en una migracion propia; la fase A aplica
-- ademas limites duros que ninguna configuracion puede ampliar.
CREATE FUNCTION
vec_bolsa_convocatorias.presupuesto_confirmacion_kms_borrador_v1()
RETURNS TABLE (
    perfil_presupuesto text,
    espera_not_before_milisegundos integer,
    tolerancia_cierre_milisegundos integer
)
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT 'desarrollo:t20'::text, 1000::integer, 500::integer
$funcion$;

CREATE TABLE vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador (
    preparacion_ref text PRIMARY KEY,
    transaccion_bd bigint NOT NULL,
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision_diario bigint NOT NULL,
    cercado bigint NOT NULL,
    convocatoria_id text NOT NULL,
    secuencia_convocatoria bigint NOT NULL,
    revision_convocatoria bigint NOT NULL,
    transaccion_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    huella_auditoria_sha256 text NOT NULL,
    evento_outbox_ref text NOT NULL UNIQUE,
    huella_evento_outbox_sha256 text NOT NULL,
    confirmacion_solicitada_en timestamptz(6) NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    perfil_presupuesto text NOT NULL,
    espera_not_before_milisegundos integer NOT NULL,
    tolerancia_cierre_milisegundos integer NOT NULL,
    recibo_cuerpo jsonb NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    UNIQUE (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac
    ),
    CONSTRAINT preparacion_confirmacion_kms_integra CHECK (
        preparacion_ref ~ '^preparacion-kms-borrador-[0-9a-f]{64}$'
        AND transaccion_bd > 0 AND revision_diario > 1 AND cercado > 0
        AND secuencia_convocatoria > 0 AND revision_convocatoria > 0
        AND confirmacion_solicitada_en <= creada_en
        AND creada_en <= confirmada_en
        AND perfil_presupuesto = 'desarrollo:t20'
        AND espera_not_before_milisegundos BETWEEN 100 AND 5000
        AND tolerancia_cierre_milisegundos BETWEEN 100 AND 1000
        AND espera_not_before_milisegundos +
            tolerancia_cierre_milisegundos <= 10000
        AND confirmada_en - creada_en =
            espera_not_before_milisegundos * interval '1 millisecond'
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_auditoria_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_evento_outbox_sha256
        ) IS TRUE
    )
);

CREATE TABLE vec_bolsa_convocatorias.cifrado_kms_borrador (
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    revision bigint NOT NULL,
    transaccion_ref text NOT NULL UNIQUE,
    perfil_ref text NOT NULL,
    perfil_version bigint NOT NULL,
    huella_perfil_sha256 text NOT NULL,
    algoritmo_aead text NOT NULL,
    algoritmo_envoltura_clave text NOT NULL,
    aad_canonica bytea NOT NULL,
    huella_aad_sha256 text NOT NULL,
    material_clave_envuelto bytea NOT NULL,
    huella_envoltura_sha256 text NOT NULL,
    nonce bytea NOT NULL,
    contenido_cifrado bytea NOT NULL,
    huella_contenido_cifrado_sha256 text NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    politica jsonb NOT NULL,
    evidencia_perfil jsonb NOT NULL,
    atestacion_kms jsonb NOT NULL,
    procedencia jsonb NOT NULL,
    perfil_presupuesto text NOT NULL,
    espera_not_before_milisegundos integer NOT NULL,
    tolerancia_cierre_milisegundos integer NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (convocatoria_id, secuencia, revision),
    FOREIGN KEY (convocatoria_id, secuencia, revision)
        REFERENCES vec_bolsa_convocatorias.borrador_convocatoria_version(
            convocatoria_id, secuencia, revision
        ),
    CONSTRAINT cifrado_kms_borrador_integro CHECK (
        perfil_version BETWEEN 1 AND 4294967295
        AND octet_length(aad_canonica) BETWEEN 2 AND 32768
        AND encode(sha256(aad_canonica), 'hex') = huella_aad_sha256
        AND octet_length(material_clave_envuelto) BETWEEN 16 AND 65536
        AND octet_length(nonce) BETWEEN 8 AND 64
        AND octet_length(contenido_cifrado) BETWEEN 16 AND 16777216
        AND encode(sha256(contenido_cifrado), 'hex') =
            huella_contenido_cifrado_sha256
        AND perfil_presupuesto = 'desarrollo:t20'
        AND espera_not_before_milisegundos BETWEEN 100 AND 5000
        AND tolerancia_cierre_milisegundos BETWEEN 100 AND 1000
        AND espera_not_before_milisegundos +
            tolerancia_cierre_milisegundos <= 10000
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_perfil_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_envoltura_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_sobre_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.procedencia_borrador_valida(
            procedencia
        ) IS TRUE
    )
);

CREATE TABLE vec_bolsa_convocatorias.acreditacion_kms_borrador (
    acreditacion_ref text NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    transaccion_ref text NOT NULL UNIQUE,
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    revision bigint NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    evento_outbox_ref text NOT NULL UNIQUE,
    cuerpo_recibo_canonico bytea NOT NULL,
    huella_cuerpo_recibo_sha256 text NOT NULL,
    acreditacion jsonb NOT NULL,
    acreditacion_canonica bytea NOT NULL,
    huella_acreditacion_sha256 text NOT NULL UNIQUE,
    recibo_canonico bytea NOT NULL,
    huella_recibo_sha256 text NOT NULL UNIQUE,
    procedencia jsonb NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (acreditacion_ref),
    FOREIGN KEY (convocatoria_id, secuencia, revision)
        REFERENCES vec_bolsa_convocatorias.cifrado_kms_borrador(
            convocatoria_id, secuencia, revision
        ),
    FOREIGN KEY (auditoria_ref)
        REFERENCES vec_bolsa_convocatorias.auditoria_borrador(auditoria_ref),
    FOREIGN KEY (evento_outbox_ref)
        REFERENCES vec_bolsa_convocatorias.outbox_borrador(evento_ref),
    CONSTRAINT acreditacion_kms_borrador_integra CHECK (
        octet_length(cuerpo_recibo_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(cuerpo_recibo_canonico), 'hex') =
            huella_cuerpo_recibo_sha256
        AND octet_length(acreditacion_canonica) BETWEEN 2 AND 1048576
        AND encode(sha256(acreditacion_canonica), 'hex') =
            huella_acreditacion_sha256
        AND octet_length(recibo_canonico) BETWEEN 2 AND 2097152
        AND encode(sha256(recibo_canonico), 'hex') = huella_recibo_sha256
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_acreditacion_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.procedencia_borrador_valida(
            procedencia
        ) IS TRUE
    )
);

CREATE FUNCTION vec_bolsa_convocatorias.huella_envoltura_clave_borrador_v1(
    p_perfil jsonb, p_envoltura jsonb, p_material_envuelto bytea
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT encode(sha256(convert_to(
        '{"Esquema":"bolsa.convocatoria.borrador.clave-envuelta.v1"' ||
        ',"PerfilRef":' || to_jsonb(p_perfil ->> 'referencia')::text ||
        ',"PerfilHuella":' ||
            to_jsonb(p_perfil ->> 'huella_contenido_sha256')::text ||
        ',"Algoritmo":' ||
            to_jsonb(p_perfil ->> 'algoritmo_envoltura_clave')::text ||
        ',"ClaveRef":' ||
            to_jsonb(p_envoltura ->> 'clave_maestra_ref')::text ||
        ',"HuellaAAD":' || to_jsonb(p_envoltura ->> 'huella_aad')::text ||
        ',"PerfilVersion":' || (p_perfil ->> 'version') ||
        ',"VersionClave":' || (p_envoltura ->> 'version_clave') ||
        ',"Material":"' || replace(encode(p_material_envuelto, 'base64'),
                                      E'\n', '') || '"}',
        'UTF8'
    )), 'hex')
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.huella_sobre_aead_borrador_v1(
    p_perfil jsonb, p_sobre jsonb, p_nonce bytea, p_texto_cifrado bytea
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT encode(sha256(convert_to(
        '{"Esquema":"bolsa.convocatoria.borrador.sobre-aead.v1"' ||
        ',"PerfilRef":' || to_jsonb(p_perfil ->> 'referencia')::text ||
        ',"PerfilHuella":' ||
            to_jsonb(p_perfil ->> 'huella_contenido_sha256')::text ||
        ',"Algoritmo":' ||
            to_jsonb(p_perfil ->> 'algoritmo_aead')::text ||
        ',"HuellaAAD":' || to_jsonb(p_sobre ->> 'huella_aad')::text ||
        ',"PerfilVersion":' || (p_perfil ->> 'version') ||
        ',"Nonce":"' || replace(encode(p_nonce, 'base64'), E'\n', '') ||
        '","TextoCifrado":"' ||
            replace(encode(p_texto_cifrado, 'base64'), E'\n', '') || '"}',
        'UTF8'
    )), 'hex')
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.aad_canonica_borrador_v1(
    p_aad jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT convert_to(
        '{"esquema":' || to_jsonb(p_aad ->> 'esquema')::text ||
        ',"version_ref":' || to_jsonb(p_aad ->> 'version_ref')::text ||
        ',"version_revision":' || (p_aad ->> 'version_revision') ||
        ',"huella_version_sha256":' ||
            to_jsonb(p_aad ->> 'huella_version_sha256')::text ||
        ',"esquema_material":' ||
            to_jsonb(p_aad ->> 'esquema_material')::text ||
        ',"huella_material_sha256":' ||
            to_jsonb(p_aad ->> 'huella_material_sha256')::text ||
        ',"perfil_cifrado_ref":' ||
            to_jsonb(p_aad ->> 'perfil_cifrado_ref')::text ||
        ',"perfil_cifrado_version":' ||
            (p_aad ->> 'perfil_cifrado_version') ||
        ',"huella_perfil_cifrado_sha256":' ||
            to_jsonb(p_aad ->> 'huella_perfil_cifrado_sha256')::text ||
        ',"algoritmo_aead":' ||
            to_jsonb(p_aad ->> 'algoritmo_aead')::text ||
        ',"algoritmo_envoltura_clave":' ||
            to_jsonb(p_aad ->> 'algoritmo_envoltura_clave')::text ||
        ',"evidencia_perfil_ref":' ||
            to_jsonb(p_aad ->> 'evidencia_perfil_ref')::text ||
        ',"evidencia_perfil_version":' ||
            (p_aad ->> 'evidencia_perfil_version') ||
        ',"huella_evidencia_perfil_sha256":' ||
            to_jsonb(p_aad ->> 'huella_evidencia_perfil_sha256')::text ||
        ',"decision_politica_ref":' ||
            to_jsonb(p_aad ->> 'decision_politica_ref')::text ||
        ',"decision_politica_version":' ||
            (p_aad ->> 'decision_politica_version') ||
        ',"huella_decision_politica_sha256":' ||
            to_jsonb(p_aad ->> 'huella_decision_politica_sha256')::text ||
        ',"localizador_esquema":' || (p_aad ->> 'localizador_esquema') ||
        ',"localizador_dominio":' ||
            to_jsonb(p_aad ->> 'localizador_dominio')::text ||
        ',"localizador_clave_ref":' ||
            to_jsonb(p_aad ->> 'localizador_clave_ref')::text ||
        ',"localizador_generacion":' ||
            (p_aad ->> 'localizador_generacion') ||
        ',"localizador_hmac_sha256":' ||
            to_jsonb(p_aad ->> 'localizador_hmac_sha256')::text ||
        ',"huella_solicitud_esquema":' ||
            (p_aad ->> 'huella_solicitud_esquema') ||
        ',"huella_solicitud_dominio":' ||
            to_jsonb(p_aad ->> 'huella_solicitud_dominio')::text ||
        ',"huella_solicitud_clave_ref":' ||
            to_jsonb(p_aad ->> 'huella_solicitud_clave_ref')::text ||
        ',"huella_solicitud_generacion":' ||
            (p_aad ->> 'huella_solicitud_generacion') ||
        ',"huella_solicitud_hmac_sha256":' ||
            to_jsonb(p_aad ->> 'huella_solicitud_hmac_sha256')::text ||
        ',"revision_diario":' || (p_aad ->> 'revision_diario') ||
        ',"cercado_diario":' || (p_aad ->> 'cercado_diario') ||
        ',"arrendamiento_inicia_en":' ||
            to_jsonb(p_aad ->> 'arrendamiento_inicia_en')::text ||
        ',"arrendamiento_vence_en":' ||
            to_jsonb(p_aad ->> 'arrendamiento_vence_en')::text ||
        ',"atestacion_sellado_ref":' ||
            to_jsonb(p_aad ->> 'atestacion_sellado_ref')::text ||
        ',"atestacion_sellado_version":' ||
            (p_aad ->> 'atestacion_sellado_version') ||
        ',"huella_atestacion_sellado_sha256":' ||
            to_jsonb(p_aad ->> 'huella_atestacion_sellado_sha256')::text ||
        ',"token_consumo_sellado_ref":' ||
            to_jsonb(p_aad ->> 'token_consumo_sellado_ref')::text ||
        ',"huella_correlacion_sha256":' ||
            to_jsonb(p_aad ->> 'huella_correlacion_sha256')::text ||
        ',"procedencia_esquema":' ||
            to_jsonb(p_aad ->> 'procedencia_esquema')::text ||
        ',"perfil_ejecucion":' ||
            to_jsonb(p_aad ->> 'perfil_ejecucion')::text ||
        ',"autoridad_acto":' ||
            to_jsonb(p_aad ->> 'autoridad_acto')::text ||
        ',"proveedor_procedencia_ref":' ||
            to_jsonb(p_aad ->> 'proveedor_procedencia_ref')::text ||
        ',"migrable_produccion":' ||
            lower(p_aad ->> 'migrable_produccion') || '}',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
    p_instante text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT replace(
        regexp_replace(
            to_char(p_instante::timestamptz AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            '(\.[0-9]*?[1-9])0+Z$', '\1Z'
        ), '.000000Z', 'Z'
    )
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(
    p_recibo jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT convert_to(
        '{"Esquema":' || to_jsonb(p_recibo ->> 'esquema')::text ||
        ',"ReciboRef":' || to_jsonb(p_recibo ->> 'recibo_ref')::text ||
        ',"TransaccionRef":' ||
            to_jsonb(p_recibo ->> 'transaccion_ref')::text ||
        ',"Accion":' || to_jsonb(p_recibo ->> 'accion')::text ||
        ',"EstadoRef":' ||
            to_jsonb(p_recibo -> 'estado_principal' ->> 'referencia')::text ||
        ',"EstadoHuella":' || to_jsonb(
            p_recibo -> 'estado_principal' ->> 'huella_estado_sha256'
        )::text ||
        ',"EstadoRevision":' ||
            (p_recibo -> 'estado_principal' ->> 'revision') ||
        ',"Identidad":{"localizador":{"version_esquema":' ||
            (p_recibo -> 'identidad' -> 'localizador' ->> 'version_esquema') ||
        ',"dominio":' || to_jsonb(
            p_recibo -> 'identidad' -> 'localizador' ->> 'dominio'
        )::text ||
        ',"clave_ref":' || to_jsonb(
            p_recibo -> 'identidad' -> 'localizador' ->> 'clave_ref'
        )::text ||
        ',"generacion_clave":' || (p_recibo -> 'identidad' ->
            'localizador' ->> 'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_recibo -> 'identidad' ->
            'localizador' ->> 'hmac_sha256')::text ||
        '},"huella_solicitud":{"version_esquema":' ||
            (p_recibo -> 'identidad' -> 'huella_solicitud' ->>
             'version_esquema') ||
        ',"dominio":' || to_jsonb(p_recibo -> 'identidad' ->
            'huella_solicitud' ->> 'dominio')::text ||
        ',"clave_ref":' || to_jsonb(p_recibo -> 'identidad' ->
            'huella_solicitud' ->> 'clave_ref')::text ||
        ',"generacion_clave":' || (p_recibo -> 'identidad' ->
            'huella_solicitud' ->> 'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_recibo -> 'identidad' ->
            'huella_solicitud' ->> 'hmac_sha256')::text || '}},' ||
        '"Decision":{"EsquemaHuella":' || to_jsonb(
            p_recibo -> 'decision' ->> 'esquema_huella')::text ||
        ',"DecisionRef":' || to_jsonb(
            p_recibo -> 'decision' ->> 'decision_ref')::text ||
        ',"HuellaDecision":' || to_jsonb(
            p_recibo -> 'decision' ->> 'huella_decision_sha256')::text ||
        ',"Accion":' || to_jsonb(
            p_recibo -> 'decision' ->> 'accion')::text ||
        ',"RecursoRef":' || to_jsonb(
            p_recibo -> 'decision' ->> 'recurso_ref')::text ||
        ',"ModuloID":' || to_jsonb(
            p_recibo -> 'decision' ->> 'modulo_id')::text ||
        ',"TipoRecurso":' || to_jsonb(
            p_recibo -> 'decision' ->> 'tipo_recurso')::text ||
        ',"ContextoHuella":' || to_jsonb(p_recibo -> 'decision' ->>
            'contexto_recurso_huella_sha256')::text ||
        ',"Finalidad":' || to_jsonb(
            p_recibo -> 'decision' ->> 'finalidad')::text ||
        ',"AsignacionRef":' || to_jsonb(
            p_recibo -> 'decision' ->> 'asignacion_ref')::text ||
        ',"AsignacionHuella":' || to_jsonb(p_recibo -> 'decision' ->>
            'asignacion_huella_sha256')::text ||
        ',"VersionRolRef":' || to_jsonb(
            p_recibo -> 'decision' ->> 'version_rol_ref')::text ||
        ',"VersionRolHuella":' || to_jsonb(p_recibo -> 'decision' ->>
            'version_rol_huella_sha256')::text ||
        ',"ControlRolRef":' || to_jsonb(p_recibo -> 'decision' ->>
            'control_vigencia_version_rol_ref')::text ||
        ',"ControlRolHuella":' || to_jsonb(p_recibo -> 'decision' ->>
            'control_vigencia_version_rol_huella_sha256')::text ||
        ',"CatalogoHuella":' || to_jsonb(p_recibo -> 'decision' ->>
            'catalogo_politicas_huella_sha256')::text ||
        ',"ControlRolRevision":' || (p_recibo -> 'decision' ->>
            'control_vigencia_version_rol_revision') ||
        ',"RevisionCatalogo":' || (p_recibo -> 'decision' ->>
            'revision_catalogo_politicas') ||
        ',"EmitidaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'decision' ->> 'emitida_en'
            ))::text ||
        ',"VerificadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'decision' ->> 'verificada_en'
            ))::text ||
        ',"ValidaHasta":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'decision' ->> 'valida_hasta'
            ))::text ||
        ',"Atestacion":{"DecisionRef":' || to_jsonb(p_recibo ->
            'decision' -> 'atestacion_pdp' ->> 'decision_ref')::text ||
        ',"AtestacionRef":' || to_jsonb(p_recibo -> 'decision' ->
            'atestacion_pdp' ->> 'atestacion_ref')::text ||
        ',"Estado":' || to_jsonb(p_recibo -> 'decision' ->
            'atestacion_pdp' ->> 'estado')::text ||
        ',"Huella":' || to_jsonb(p_recibo -> 'decision' ->
            'atestacion_pdp' ->> 'huella_atestacion_sha256')::text ||
        ',"VerificadorRef":' || to_jsonb(p_recibo -> 'decision' ->
            'atestacion_pdp' ->> 'verificador_ref')::text ||
        ',"Version":' || (p_recibo -> 'decision' ->
            'atestacion_pdp' ->> 'version') ||
        ',"VerificadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'decision' -> 'atestacion_pdp' ->> 'verificada_en'
            ))::text || '}},' ||
        '"Sellado":{"Accion":' || to_jsonb(
            p_recibo -> 'sellado_motivo' ->> 'accion')::text ||
        ',"ConvocatoriaRef":' || to_jsonb(
            p_recibo -> 'sellado_motivo' ->> 'convocatoria_ref')::text ||
        ',"AtestacionRef":' || to_jsonb(
            p_recibo -> 'sellado_motivo' ->> 'atestacion_ref')::text ||
        ',"Estado":' || to_jsonb(
            p_recibo -> 'sellado_motivo' ->> 'estado_atestacion')::text ||
        ',"Huella":' || to_jsonb(p_recibo -> 'sellado_motivo' ->>
            'huella_atestacion_sha256')::text ||
        ',"TokenConsumoRef":' || to_jsonb(p_recibo -> 'sellado_motivo' ->>
            'token_consumo_ref')::text ||
        ',"MaterializadorRef":' || to_jsonb(p_recibo -> 'sellado_motivo' ->>
            'materializador_ref')::text ||
        ',"Version":' || (p_recibo -> 'sellado_motivo' ->>
            'version_atestacion') ||
        ',"EmitidaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'sellado_motivo' ->> 'atestacion_emitida_en'
            ))::text ||
        ',"ValidaHasta":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo -> 'sellado_motivo' ->> 'atestacion_valida_hasta'
            ))::text ||
        ',"HMAC":{"Dominio":' || to_jsonb(p_recibo -> 'sellado_motivo' ->
            'hmac' ->> 'dominio_criptografico')::text ||
        ',"ClaveRef":' || to_jsonb(p_recibo -> 'sellado_motivo' ->
            'hmac' ->> 'clave_hmac_ref')::text ||
        ',"Valor":' || to_jsonb(p_recibo -> 'sellado_motivo' ->
            'hmac' ->> 'valor_hmac_sha256')::text ||
        ',"Generacion":' || (p_recibo -> 'sellado_motivo' ->
            'hmac' ->> 'generacion_clave') || '}},' ||
        '"Revision":' || (p_recibo ->> 'revision_confirmada') ||
        ',"Cercado":' || (p_recibo ->> 'cercado_confirmado') ||
        ',"ArrendamientoIniciaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo ->> 'arrendamiento_inicia_en'
            ))::text ||
        ',"ArrendamientoVenceEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo ->> 'arrendamiento_vence_en'
            ))::text ||
        ',"AuditoriaRef":' || to_jsonb(p_recibo ->> 'auditoria_ref')::text ||
        ',"HuellaAuditoria":' ||
            to_jsonb(p_recibo ->> 'huella_auditoria_sha256')::text ||
        ',"EventoRef":' || to_jsonb(p_recibo ->> 'evento_outbox_ref')::text ||
        ',"HuellaEvento":' ||
            to_jsonb(p_recibo ->> 'huella_evento_outbox_sha256')::text ||
        ',"ConfirmadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_recibo ->> 'confirmada_en'
            ))::text ||
        ',"Procedencia":{"Esquema":' || to_jsonb(
            p_recibo -> 'procedencia' ->> 'esquema')::text ||
        ',"Perfil":' || to_jsonb(
            p_recibo -> 'procedencia' ->> 'perfil_ejecucion')::text ||
        ',"Autoridad":' || to_jsonb(
            p_recibo -> 'procedencia' ->> 'autoridad')::text ||
        ',"ProveedorRef":' || to_jsonb(
            p_recibo -> 'procedencia' ->> 'proveedor_ref')::text ||
        ',"Migrable":' || lower(
            p_recibo -> 'procedencia' ->> 'migrable_produccion'
        ) || '}}',
        'UTF8'
    )
$funcion$;

-- Reproduce, sin depender del orden de jsonb, la preimagen exacta que el
-- nucleo Go usa para HuellaAcreditacionSHA256. El hash final queda fuera de
-- su propia preimagen y por tanto no introduce un ciclo.
CREATE FUNCTION
vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
    p_acreditacion jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT convert_to(
        '{"Esquema":' || to_jsonb(p_acreditacion ->> 'esquema')::text ||
        ',"AcreditacionRef":' ||
            to_jsonb(p_acreditacion ->> 'acreditacion_ref')::text ||
        ',"Estado":' || to_jsonb(p_acreditacion ->> 'estado')::text ||
        ',"AtestacionRef":' ||
            to_jsonb(p_acreditacion ->> 'atestacion_ref')::text ||
        ',"PerfilRef":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'referencia')::text ||
        ',"HuellaPerfilSHA256":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'huella_contenido_sha256')::text ||
        ',"AlgoritmoAEAD":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'algoritmo_aead')::text ||
        ',"AlgoritmoEnvoltura":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'algoritmo_envoltura_clave')::text ||
        ',"ClaveRef":' ||
            to_jsonb(p_acreditacion ->> 'clave_maestra_ref')::text ||
        ',"HuellaAAD":' ||
            to_jsonb(p_acreditacion ->> 'huella_aad')::text ||
        ',"HuellaEnvoltura":' || to_jsonb(
            p_acreditacion ->> 'huella_envoltura_sha256')::text ||
        ',"HuellaSobre":' ||
            to_jsonb(p_acreditacion ->> 'huella_sobre_sha256')::text ||
        ',"ComprobacionRef":' ||
            to_jsonb(p_acreditacion ->> 'comprobacion_kms_ref')::text ||
        ',"HuellaComprobacion":' || to_jsonb(
            p_acreditacion ->> 'huella_comprobacion_kms_sha256')::text ||
        ',"TransaccionRef":' ||
            to_jsonb(p_acreditacion ->> 'transaccion_ref')::text ||
        ',"ReciboRef":' ||
            to_jsonb(p_acreditacion ->> 'recibo_ref')::text ||
        ',"HuellaCuerpoRecibo":' || to_jsonb(
            p_acreditacion ->> 'huella_cuerpo_recibo_sha256')::text ||
        ',"VerificadorRef":' ||
            to_jsonb(p_acreditacion ->> 'verificador_ref')::text ||
        ',"ProcedenciaEsquema":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'esquema')::text ||
        ',"PerfilEjecucion":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'perfil_ejecucion')::text ||
        ',"Autoridad":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'autoridad')::text ||
        ',"ProveedorProcedencia":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'proveedor_ref')::text ||
        ',"AlgoritmoFirmaAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'algoritmo_firma')::text ||
        ',"VerificadorAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'verificador_ref')::text ||
        ',"HuellaClaveAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'huella_clave_publica_sha256')::text ||
        ',"HuellaPreimagenAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'huella_preimagen_sha256')::text ||
        ',"FirmaAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'firma_base64url_sin_relleno')::text ||
        ',"AlgoritmoFirmaRevalidacion":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'algoritmo_firma')::text ||
        ',"VerificadorRevalidacion":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'verificador_ref')::text ||
        ',"HuellaClaveRevalidacion":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'huella_clave_publica_sha256')::text ||
        ',"HuellaPreimagenRevalidacion":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'huella_preimagen_sha256')::text ||
        ',"FirmaRevalidacion":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'firma_base64url_sin_relleno')::text ||
        ',"VersionAcreditacion":' ||
            (p_acreditacion ->> 'version_acreditacion') ||
        ',"VersionAtestacion":' ||
            (p_acreditacion ->> 'version_atestacion') ||
        ',"PerfilVersion":' ||
            (p_acreditacion -> 'perfil' ->> 'version') ||
        ',"VersionClave":' || (p_acreditacion ->> 'version_clave') ||
        ',"RevisionReserva":' ||
            (p_acreditacion ->> 'revision_reserva') ||
        ',"RevisionConfirmada":' ||
            (p_acreditacion ->> 'revision_confirmada') ||
        ',"Cercado":' || (p_acreditacion ->> 'cercado') ||
        ',"MigrableProduccion":' || lower(
            p_acreditacion -> 'procedencia' ->> 'migrable_produccion') ||
        ',"Identidad":{"localizador":{"version_esquema":' ||
            (p_acreditacion -> 'identidad_primaria' -> 'localizador' ->>
                'version_esquema') ||
        ',"dominio":' || to_jsonb(p_acreditacion -> 'identidad_primaria' ->
            'localizador' ->> 'dominio')::text ||
        ',"clave_ref":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'clave_ref')::text ||
        ',"generacion_clave":' || (p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'hmac_sha256')::text ||
        '},"huella_solicitud":{"version_esquema":' ||
            (p_acreditacion -> 'identidad_primaria' ->
                'huella_solicitud' ->> 'version_esquema') ||
        ',"dominio":' || to_jsonb(p_acreditacion -> 'identidad_primaria' ->
            'huella_solicitud' ->> 'dominio')::text ||
        ',"clave_ref":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->> 'clave_ref')::text ||
        ',"generacion_clave":' || (p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->>
                'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->>
                'hmac_sha256')::text || '}},' ||
        '"ArrendamientoIniciaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'arrendamiento_inicia_en'))::text ||
        ',"ArrendamientoVenceEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'arrendamiento_vence_en'))::text ||
        ',"AtestacionEmitidaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'atestacion_emitida_en'))::text ||
        ',"AtestacionValidaHasta":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'atestacion_valida_hasta'))::text ||
        ',"ConfirmacionSolicitadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'confirmacion_solicitada_en'))::text ||
        ',"RevalidacionSolicitadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'revalidacion_solicitada_en'))::text ||
        ',"RevalidadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'revalidada_en'))::text ||
        ',"ConfirmadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'confirmada_en'))::text || '}',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION
vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
    p_acreditacion jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT convert_to(
        '{"Esquema":"bolsa.convocatoria.borrador.atestacion-kms.v1"' ||
        ',"AtestacionRef":' ||
            to_jsonb(p_acreditacion ->> 'atestacion_ref')::text ||
        ',"Estado":"vigente"' ||
        ',"PerfilRef":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'referencia')::text ||
        ',"HuellaPerfil":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'huella_contenido_sha256')::text ||
        ',"AlgoritmoAEAD":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'algoritmo_aead')::text ||
        ',"AlgoritmoEnvoltura":' || to_jsonb(
            p_acreditacion -> 'perfil' ->> 'algoritmo_envoltura_clave')::text ||
        ',"ClaveRef":' ||
            to_jsonb(p_acreditacion ->> 'clave_maestra_ref')::text ||
        ',"HuellaAAD":' ||
            to_jsonb(p_acreditacion ->> 'huella_aad')::text ||
        ',"HuellaEnvoltura":' || to_jsonb(
            p_acreditacion ->> 'huella_envoltura_sha256')::text ||
        ',"HuellaSobre":' ||
            to_jsonb(p_acreditacion ->> 'huella_sobre_sha256')::text ||
        ',"VerificadorRef":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'verificador_ref')::text ||
        ',"AlgoritmoFirma":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'algoritmo_firma')::text ||
        ',"HuellaClavePublica":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'huella_clave_publica_sha256')::text ||
        ',"ProcedenciaEsquema":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'esquema')::text ||
        ',"PerfilEjecucion":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'perfil_ejecucion')::text ||
        ',"Autoridad":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'autoridad')::text ||
        ',"ProveedorProcedenciaRef":' || to_jsonb(
            p_acreditacion -> 'procedencia' ->> 'proveedor_ref')::text ||
        ',"VersionAtestacion":' ||
            (p_acreditacion ->> 'version_atestacion') ||
        ',"PerfilVersion":' ||
            (p_acreditacion -> 'perfil' ->> 'version') ||
        ',"VersionClave":' || (p_acreditacion ->> 'version_clave') ||
        ',"MigrableProduccion":' || lower(
            p_acreditacion -> 'procedencia' ->> 'migrable_produccion') ||
        ',"EmitidaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'atestacion_emitida_en'))::text ||
        ',"ValidaHasta":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'atestacion_valida_hasta'))::text || '}',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.firma_base64url_borrador_v1(
    p_firma text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT decode(
        translate(p_firma, '-_', '+/') ||
            repeat('=', (4 - length(p_firma) % 4) % 4),
        'base64'
    )
$funcion$;

CREATE FUNCTION
vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
    p_acreditacion jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT convert_to(
        '{"Esquema":"bolsa.convocatoria.borrador.revalidacion-kms.v1"' ||
        ',"AtestacionRef":' ||
            to_jsonb(p_acreditacion ->> 'atestacion_ref')::text ||
        ',"Estado":"autorizada"' ||
        ',"HuellaAAD":' ||
            to_jsonb(p_acreditacion ->> 'huella_aad')::text ||
        ',"HuellaCuerpoRecibo":' || to_jsonb(
            p_acreditacion ->> 'huella_cuerpo_recibo_sha256')::text ||
        ',"ComprobacionRef":' ||
            to_jsonb(p_acreditacion ->> 'comprobacion_kms_ref')::text ||
        ',"HuellaComprobacion":' || to_jsonb(
            p_acreditacion ->> 'huella_comprobacion_kms_sha256')::text ||
        ',"AlgoritmoFirma":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'algoritmo_firma')::text ||
        ',"VerificadorRef":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'verificador_ref')::text ||
        ',"HuellaClavePublica":' || to_jsonb(
            p_acreditacion -> 'firma_revalidacion_kms' ->>
                'huella_clave_publica_sha256')::text ||
        ',"AlgoritmoAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'algoritmo_firma')::text ||
        ',"VerificadorAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'verificador_ref')::text ||
        ',"HuellaClaveAtestacion":' || to_jsonb(
            p_acreditacion -> 'firma_atestacion_kms' ->>
                'huella_clave_publica_sha256')::text ||
        ',"HuellaPreimagenAtestacion":' || to_jsonb(encode(sha256(
            vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
                p_acreditacion)), 'hex'))::text ||
        ',"HuellaFirmaAtestacion":' || to_jsonb(encode(sha256(
            vec_bolsa_convocatorias.firma_base64url_borrador_v1(
                p_acreditacion -> 'firma_atestacion_kms' ->>
                    'firma_base64url_sin_relleno')), 'hex'))::text ||
        ',"VersionAtestacion":' ||
            (p_acreditacion ->> 'version_atestacion') ||
        ',"Revision":' || (p_acreditacion ->> 'revision_reserva') ||
        ',"Cercado":' || (p_acreditacion ->> 'cercado') ||
        ',"Identidad":{"localizador":{"version_esquema":' ||
            (p_acreditacion -> 'identidad_primaria' -> 'localizador' ->>
                'version_esquema') ||
        ',"dominio":' || to_jsonb(p_acreditacion -> 'identidad_primaria' ->
            'localizador' ->> 'dominio')::text ||
        ',"clave_ref":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'clave_ref')::text ||
        ',"generacion_clave":' || (p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'localizador' ->> 'hmac_sha256')::text ||
        '},"huella_solicitud":{"version_esquema":' ||
            (p_acreditacion -> 'identidad_primaria' ->
                'huella_solicitud' ->> 'version_esquema') ||
        ',"dominio":' || to_jsonb(p_acreditacion -> 'identidad_primaria' ->
            'huella_solicitud' ->> 'dominio')::text ||
        ',"clave_ref":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->> 'clave_ref')::text ||
        ',"generacion_clave":' || (p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->>
                'generacion_clave') ||
        ',"valor_hmac_sha256":' || to_jsonb(p_acreditacion ->
            'identidad_primaria' -> 'huella_solicitud' ->>
                'hmac_sha256')::text || '}},' ||
        '"ArrendamientoVenceEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'arrendamiento_vence_en'))::text ||
        ',"ConfirmacionSolicitadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'confirmacion_solicitada_en'))::text ||
        ',"RevalidacionSolicitadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'revalidacion_solicitada_en'))::text ||
        ',"ComprobadaEn":' || to_jsonb(
            vec_bolsa_convocatorias.instante_rfc3339nano_borrador_v1(
                p_acreditacion ->> 'revalidada_en'))::text || '}',
        'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.evidencia_cifrado_kms_borrador_valida(
    p_evidencia jsonb,
    p_confirmacion jsonb,
    p_material_canonico bytea,
    p_aad_canonica bytea,
    p_material_envuelto bytea,
    p_nonce bytea,
    p_texto_cifrado bytea
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    perfil jsonb := p_evidencia -> 'perfil';
    politica jsonb := p_evidencia -> 'politica';
    evidencia_perfil jsonb := p_evidencia -> 'evidencia_perfil';
    procedencia jsonb := p_evidencia -> 'procedencia';
    aad jsonb := p_evidencia -> 'aad';
    aad_contenido jsonb;
    envoltura jsonb := p_evidencia -> 'envoltura_clave';
    sobre jsonb := p_evidencia -> 'sobre';
    atestacion jsonb := p_evidencia -> 'atestacion_kms';
    identidad jsonb := p_confirmacion -> 'identidad';
    sellado jsonb := p_confirmacion -> 'sellado_motivo';
    material jsonb;
    huella_material text := encode(sha256(p_material_canonico), 'hex');
BEGIN
    IF p_material_canonico IS NULL
       OR octet_length(p_material_canonico) NOT BETWEEN 2 AND 1048576
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_confirmacion, ARRAY[
               'cercado','envoltura_cifrado','esquema','identidad',
               'proyeccion_ligera','revision','sellado_motivo','solicitada_en'
           ]
       ) IS NOT TRUE
       OR p_confirmacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.convocatoria.confirmacion-borrador.v2'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           identidad
       ) IS NOT TRUE
       OR (p_confirmacion ->> 'revision' ~ '^[1-9][0-9]{0,18}$') IS NOT TRUE
       OR (p_confirmacion ->> 'cercado' ~ '^[1-9][0-9]{0,18}$') IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_confirmacion ->> 'solicitada_en'
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    BEGIN
        material := convert_from(p_material_canonico, 'UTF8')::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN false;
    END;
    IF jsonb_typeof(material) IS DISTINCT FROM 'object'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           material ->> 'esquema', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_evidencia, ARRAY[
               'aad','atestacion_kms','envoltura_clave','esquema',
               'evidencia_perfil','perfil','politica','procedencia','sobre'
           ]
       ) IS NOT TRUE
       OR p_evidencia ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.convocatoria.cifrado-kms-persistencia.v1'
       OR vec_bolsa_convocatorias.perfil_cifrado_borrador_valido(perfil)
          IS NOT TRUE
       OR vec_bolsa_convocatorias.procedencia_borrador_valida(procedencia)
          IS NOT TRUE
       OR p_aad_canonica IS NULL
       OR octet_length(p_aad_canonica) NOT BETWEEN 2 AND 32768
       OR p_material_envuelto IS NULL
       OR octet_length(p_material_envuelto) NOT BETWEEN 16 AND 65536
       OR p_nonce IS NULL OR octet_length(p_nonce) NOT BETWEEN 8 AND 64
       OR p_texto_cifrado IS NULL
       OR octet_length(p_texto_cifrado) NOT BETWEEN 16 AND 16777216
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           aad, ARRAY['esquema','huella_sha256']
       ) IS NOT TRUE
       OR aad ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.aad.v1'
       OR encode(sha256(p_aad_canonica), 'hex') IS DISTINCT FROM
          aad ->> 'huella_sha256'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           envoltura, ARRAY[
               'clave_maestra_ref','esquema','huella_aad',
               'huella_envoltura_sha256','version_clave'
           ]
       ) IS NOT TRUE
       OR envoltura ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.clave-envuelta.v1'
       OR (envoltura ->> 'version_clave' ~ '^[1-9][0-9]{0,9}$')
          IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           envoltura ->> 'clave_maestra_ref', 512
       ) IS NOT TRUE
       OR envoltura ->> 'huella_aad' IS DISTINCT FROM
          aad ->> 'huella_sha256'
       OR envoltura ->> 'huella_envoltura_sha256' IS DISTINCT FROM
          vec_bolsa_convocatorias.huella_envoltura_clave_borrador_v1(
              perfil, envoltura, p_material_envuelto
          )
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           sobre, ARRAY['esquema','huella_aad','huella_sobre_sha256']
       ) IS NOT TRUE
       OR sobre ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.sobre-aead.v1'
       OR sobre ->> 'huella_aad' IS DISTINCT FROM aad ->> 'huella_sha256'
       OR sobre ->> 'huella_sobre_sha256' IS DISTINCT FROM
          vec_bolsa_convocatorias.huella_sobre_aead_borrador_v1(
              perfil, sobre, p_nonce, p_texto_cifrado
          ) THEN
        RETURN false;
    END IF;

    BEGIN
        aad_contenido := convert_from(p_aad_canonica, 'UTF8')::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN false;
    END;
    IF vec_bolsa_convocatorias.objeto_json_exacto(
           aad_contenido, ARRAY[
               'algoritmo_aead','algoritmo_envoltura_clave',
               'arrendamiento_inicia_en','arrendamiento_vence_en',
               'atestacion_sellado_ref','atestacion_sellado_version',
               'autoridad_acto','cercado_diario','decision_politica_ref',
               'decision_politica_version','esquema','esquema_material',
               'evidencia_perfil_ref','evidencia_perfil_version',
               'huella_atestacion_sellado_sha256',
               'huella_correlacion_sha256',
               'huella_decision_politica_sha256',
               'huella_evidencia_perfil_sha256',
               'huella_material_sha256','huella_perfil_cifrado_sha256',
               'huella_solicitud_clave_ref','huella_solicitud_dominio',
               'huella_solicitud_esquema','huella_solicitud_generacion',
               'huella_solicitud_hmac_sha256','huella_version_sha256',
               'localizador_clave_ref','localizador_dominio',
               'localizador_esquema','localizador_generacion',
               'localizador_hmac_sha256','migrable_produccion',
               'perfil_cifrado_ref','perfil_cifrado_version',
               'perfil_ejecucion','procedencia_esquema',
               'proveedor_procedencia_ref',
               'revision_diario','token_consumo_sellado_ref',
               'version_ref','version_revision'
           ]
       ) IS NOT TRUE
       OR p_aad_canonica IS DISTINCT FROM
          vec_bolsa_convocatorias.aad_canonica_borrador_v1(aad_contenido)
       OR aad_contenido ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.aad.v1'
       OR aad_contenido ->> 'version_ref' IS DISTINCT FROM
          p_confirmacion -> 'proyeccion_ligera' ->> 'referencia'
       OR aad_contenido ->> 'version_revision' IS DISTINCT FROM
          p_confirmacion -> 'proyeccion_ligera' ->> 'revision'
       OR aad_contenido ->> 'huella_version_sha256' IS DISTINCT FROM
          p_confirmacion -> 'proyeccion_ligera' ->> 'huella_estado_sha256'
       OR aad_contenido ->> 'esquema_material' IS DISTINCT FROM
          material ->> 'esquema'
       OR aad_contenido ->> 'huella_material_sha256' IS DISTINCT FROM
          huella_material
       OR aad_contenido ->> 'perfil_cifrado_ref' IS DISTINCT FROM
          perfil ->> 'referencia'
       OR aad_contenido ->> 'perfil_cifrado_version' IS DISTINCT FROM
          perfil ->> 'version'
       OR aad_contenido ->> 'huella_perfil_cifrado_sha256' IS DISTINCT FROM
          perfil ->> 'huella_contenido_sha256'
       OR aad_contenido ->> 'algoritmo_aead' IS DISTINCT FROM
          perfil ->> 'algoritmo_aead'
       OR aad_contenido ->> 'algoritmo_envoltura_clave' IS DISTINCT FROM
          perfil ->> 'algoritmo_envoltura_clave'
       OR aad_contenido ->> 'evidencia_perfil_ref' IS DISTINCT FROM
          evidencia_perfil ->> 'evidencia_ref'
       OR aad_contenido ->> 'evidencia_perfil_version' IS DISTINCT FROM
          evidencia_perfil ->> 'version_evidencia'
       OR aad_contenido ->> 'huella_evidencia_perfil_sha256'
          IS DISTINCT FROM evidencia_perfil ->> 'huella_evidencia_sha256'
       OR aad_contenido ->> 'decision_politica_ref' IS DISTINCT FROM
          politica ->> 'decision_politica_ref'
       OR aad_contenido ->> 'decision_politica_version' IS DISTINCT FROM
          politica ->> 'version_decision_politica'
       OR aad_contenido ->> 'huella_decision_politica_sha256'
          IS DISTINCT FROM politica ->> 'huella_decision_sha256'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
          aad_contenido ->> 'huella_correlacion_sha256'
       ) IS NOT TRUE
       OR aad_contenido ->> 'localizador_esquema' IS DISTINCT FROM
          identidad -> 'localizador' ->> 'version_esquema'
       OR aad_contenido ->> 'localizador_dominio' IS DISTINCT FROM
          identidad -> 'localizador' ->> 'dominio'
       OR aad_contenido ->> 'localizador_clave_ref' IS DISTINCT FROM
          identidad -> 'localizador' ->> 'clave_ref'
       OR aad_contenido ->> 'localizador_generacion' IS DISTINCT FROM
          identidad -> 'localizador' ->> 'generacion_clave'
       OR aad_contenido ->> 'localizador_hmac_sha256' IS DISTINCT FROM
          identidad -> 'localizador' ->> 'hmac_sha256'
       OR aad_contenido ->> 'huella_solicitud_esquema' IS DISTINCT FROM
          identidad -> 'huella_solicitud' ->> 'version_esquema'
       OR aad_contenido ->> 'huella_solicitud_dominio' IS DISTINCT FROM
          identidad -> 'huella_solicitud' ->> 'dominio'
       OR aad_contenido ->> 'huella_solicitud_clave_ref' IS DISTINCT FROM
          identidad -> 'huella_solicitud' ->> 'clave_ref'
       OR aad_contenido ->> 'huella_solicitud_generacion' IS DISTINCT FROM
          identidad -> 'huella_solicitud' ->> 'generacion_clave'
       OR aad_contenido ->> 'huella_solicitud_hmac_sha256' IS DISTINCT FROM
          identidad -> 'huella_solicitud' ->> 'hmac_sha256'
       OR aad_contenido ->> 'revision_diario' IS DISTINCT FROM
          p_confirmacion ->> 'revision'
       OR aad_contenido ->> 'cercado_diario' IS DISTINCT FROM
          p_confirmacion ->> 'cercado'
       OR aad_contenido ->> 'arrendamiento_inicia_en' IS DISTINCT FROM
          politica ->> 'arrendamiento_inicia_en'
       OR aad_contenido ->> 'arrendamiento_vence_en' IS DISTINCT FROM
          politica ->> 'arrendamiento_vence_en'
       OR aad_contenido ->> 'atestacion_sellado_ref' IS DISTINCT FROM
          sellado ->> 'atestacion_ref'
       OR aad_contenido ->> 'atestacion_sellado_version' IS DISTINCT FROM
          sellado ->> 'version_atestacion'
       OR aad_contenido ->> 'huella_atestacion_sellado_sha256'
          IS DISTINCT FROM
          sellado ->> 'huella_atestacion_sha256'
       OR aad_contenido ->> 'token_consumo_sellado_ref' IS DISTINCT FROM
          sellado ->> 'token_consumo_ref'
       OR aad_contenido ->> 'procedencia_esquema' IS DISTINCT FROM
          procedencia ->> 'esquema'
       OR aad_contenido ->> 'perfil_ejecucion' IS DISTINCT FROM
          procedencia ->> 'perfil_ejecucion'
       OR aad_contenido ->> 'autoridad_acto' IS DISTINCT FROM
          procedencia ->> 'autoridad'
       OR aad_contenido ->> 'proveedor_procedencia_ref' IS DISTINCT FROM
          procedencia ->> 'proveedor_ref'
       OR (aad_contenido ->> 'migrable_produccion')::boolean
          IS DISTINCT FROM
          (procedencia ->> 'migrable_produccion')::boolean THEN
        RETURN false;
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           politica, ARRAY[
               'accion','arrendamiento_inicia_en','arrendamiento_vence_en',
               'autoridad_ref','catalogo_ref','cercado',
               'decision_politica_ref','emitida_en','esquema','estado',
               'huella_catalogo_sha256','huella_decision_sha256',
               'huella_material_sha256','identidad_primaria','perfil',
               'revision','revision_catalogo','solicitada_en','valida_hasta',
               'verificada_en','version_decision_politica'
           ]
       ) IS NOT TRUE
       OR politica ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.politica-cifrado.v1'
       OR politica ->> 'estado' IS DISTINCT FROM 'vigente'
       OR politica -> 'perfil' IS DISTINCT FROM perfil
       OR politica -> 'identidad_primaria' IS DISTINCT FROM identidad
       OR politica ->> 'accion' IS DISTINCT FROM
          p_confirmacion -> 'sellado_motivo' ->> 'accion'
       OR politica ->> 'huella_material_sha256' IS DISTINCT FROM
          huella_material
       OR politica ->> 'revision' IS DISTINCT FROM
          p_confirmacion ->> 'revision'
       OR politica ->> 'cercado' IS DISTINCT FROM
          p_confirmacion ->> 'cercado'
       OR (politica ->> 'version_decision_politica' ~
           '^[1-9][0-9]{0,9}$') IS NOT TRUE
       OR (politica ->> 'revision_catalogo' ~
           '^[1-9][0-9]{0,18}$') IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           politica ->> 'decision_politica_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           politica ->> 'catalogo_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           politica ->> 'autoridad_ref', 512
       ) IS NOT TRUE
       OR politica ->> 'autoridad_ref' IS NOT DISTINCT FROM
          politica ->> 'decision_politica_ref'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           politica ->> 'huella_decision_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           politica ->> 'huella_catalogo_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'arrendamiento_inicia_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'arrendamiento_vence_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'solicitada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'emitida_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'verificada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           politica ->> 'valida_hasta'
       ) IS NOT TRUE
       OR (politica ->> 'arrendamiento_inicia_en')::timestamptz >=
          (politica ->> 'arrendamiento_vence_en')::timestamptz
       OR (politica ->> 'verificada_en')::timestamptz <
          (politica ->> 'emitida_en')::timestamptz
       OR (politica ->> 'verificada_en')::timestamptz <
          (politica ->> 'arrendamiento_inicia_en')::timestamptz
       OR (politica ->> 'solicitada_en')::timestamptz <
          (politica ->> 'verificada_en')::timestamptz
       OR (politica ->> 'solicitada_en')::timestamptz >=
          (politica ->> 'valida_hasta')::timestamptz
       OR (politica ->> 'valida_hasta')::timestamptz >
          (politica ->> 'arrendamiento_vence_en')::timestamptz THEN
        RETURN false;
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           evidencia_perfil, ARRAY[
               'accion','arrendamiento_inicia_en','arrendamiento_vence_en',
               'catalogo_ref','cercado','decision_politica_ref',
               'emitida_en','esquema','estado','evidencia_ref',
               'huella_catalogo_sha256','huella_decision_politica_sha256',
               'huella_evidencia_sha256','huella_material_sha256',
               'identidad_primaria','perfil','revision','revision_catalogo',
               'solicitud_resolucion_en','valida_hasta','verificada_en',
               'verificador_ref','version_decision_politica',
               'version_evidencia'
           ]
       ) IS NOT TRUE
       OR evidencia_perfil ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.evidencia-perfil-cifrado.v1'
       OR evidencia_perfil ->> 'estado' IS DISTINCT FROM 'vigente'
       OR evidencia_perfil -> 'perfil' IS DISTINCT FROM perfil
       OR evidencia_perfil -> 'identidad_primaria' IS DISTINCT FROM identidad
       OR evidencia_perfil ->> 'accion' IS DISTINCT FROM
          politica ->> 'accion'
       OR evidencia_perfil ->> 'huella_material_sha256' IS DISTINCT FROM
          huella_material
       OR evidencia_perfil ->> 'revision' IS DISTINCT FROM
          p_confirmacion ->> 'revision'
       OR evidencia_perfil ->> 'cercado' IS DISTINCT FROM
          p_confirmacion ->> 'cercado'
       OR evidencia_perfil ->> 'arrendamiento_inicia_en' IS DISTINCT FROM
          politica ->> 'arrendamiento_inicia_en'
       OR evidencia_perfil ->> 'arrendamiento_vence_en' IS DISTINCT FROM
          politica ->> 'arrendamiento_vence_en'
       OR evidencia_perfil ->> 'catalogo_ref' IS DISTINCT FROM
          politica ->> 'catalogo_ref'
       OR evidencia_perfil ->> 'revision_catalogo' IS DISTINCT FROM
          politica ->> 'revision_catalogo'
       OR evidencia_perfil ->> 'huella_catalogo_sha256' IS DISTINCT FROM
          politica ->> 'huella_catalogo_sha256'
       OR evidencia_perfil ->> 'decision_politica_ref' IS DISTINCT FROM
          politica ->> 'decision_politica_ref'
       OR evidencia_perfil ->> 'version_decision_politica' IS DISTINCT FROM
          politica ->> 'version_decision_politica'
       OR evidencia_perfil ->> 'huella_decision_politica_sha256'
          IS DISTINCT FROM politica ->> 'huella_decision_sha256'
       OR (evidencia_perfil ->> 'version_evidencia' ~
           '^[1-9][0-9]{0,9}$') IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           evidencia_perfil ->> 'evidencia_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           evidencia_perfil ->> 'verificador_ref', 512
       ) IS NOT TRUE
       OR evidencia_perfil ->> 'evidencia_ref' IS NOT DISTINCT FROM
          evidencia_perfil ->> 'verificador_ref'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
          evidencia_perfil ->> 'huella_evidencia_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'arrendamiento_inicia_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'arrendamiento_vence_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'solicitud_resolucion_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'emitida_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'verificada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           evidencia_perfil ->> 'valida_hasta'
       ) IS NOT TRUE
       OR (evidencia_perfil ->> 'verificada_en')::timestamptz <
          (evidencia_perfil ->> 'emitida_en')::timestamptz
       OR (evidencia_perfil ->> 'verificada_en')::timestamptz <
          (evidencia_perfil ->> 'arrendamiento_inicia_en')::timestamptz
       OR (evidencia_perfil ->> 'solicitud_resolucion_en')::timestamptz <
          (evidencia_perfil ->> 'verificada_en')::timestamptz
       OR (evidencia_perfil ->> 'solicitud_resolucion_en')::timestamptz >=
          (evidencia_perfil ->> 'valida_hasta')::timestamptz
       OR (evidencia_perfil ->> 'valida_hasta')::timestamptz <=
          (evidencia_perfil ->> 'verificada_en')::timestamptz
       OR (evidencia_perfil ->> 'valida_hasta')::timestamptz >
          (evidencia_perfil ->> 'arrendamiento_vence_en')::timestamptz
       OR (evidencia_perfil ->> 'verificada_en')::timestamptz <
          (politica ->> 'verificada_en')::timestamptz
       OR (evidencia_perfil ->> 'valida_hasta')::timestamptz >
          (politica ->> 'valida_hasta')::timestamptz THEN
        RETURN false;
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           atestacion, ARRAY[
               'atestacion_ref','clave_maestra_ref','emitida_en','esquema',
               'estado','firma','huella_aad','huella_envoltura_sha256',
               'huella_sobre_sha256','perfil','procedencia','valida_hasta',
               'verificador_ref','version_atestacion','version_clave'
           ]
       ) IS NOT TRUE
       OR atestacion ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.atestacion-kms.v1'
       OR atestacion ->> 'estado' IS DISTINCT FROM 'vigente'
       OR atestacion -> 'perfil' IS DISTINCT FROM perfil
       OR atestacion -> 'procedencia' IS DISTINCT FROM procedencia
       OR atestacion ->> 'clave_maestra_ref' IS DISTINCT FROM
          envoltura ->> 'clave_maestra_ref'
       OR atestacion ->> 'version_clave' IS DISTINCT FROM
          envoltura ->> 'version_clave'
       OR atestacion ->> 'huella_aad' IS DISTINCT FROM
          aad ->> 'huella_sha256'
       OR atestacion ->> 'huella_envoltura_sha256' IS DISTINCT FROM
          envoltura ->> 'huella_envoltura_sha256'
       OR atestacion ->> 'huella_sobre_sha256' IS DISTINCT FROM
          sobre ->> 'huella_sobre_sha256'
       OR atestacion ->> 'verificador_ref' IS DISTINCT FROM
          atestacion -> 'firma' ->> 'verificador_ref'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
          atestacion ->> 'atestacion_ref', 512
       ) IS NOT TRUE
       OR (atestacion ->> 'version_atestacion' ~
           '^[1-9][0-9]{0,9}$') IS NOT TRUE
       OR atestacion ->> 'atestacion_ref' IS NOT DISTINCT FROM
          atestacion ->> 'verificador_ref'
       OR vec_bolsa_convocatorias.firma_evidencia_borrador_valida(
          atestacion -> 'firma'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
          atestacion ->> 'emitida_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
          atestacion ->> 'valida_hasta'
       ) IS NOT TRUE
       OR (atestacion ->> 'emitida_en')::timestamptz >=
          (atestacion ->> 'valida_hasta')::timestamptz
       OR (atestacion ->> 'valida_hasta')::timestamptz -
          (atestacion ->> 'emitida_en')::timestamptz > interval '10 minutes'
       OR (atestacion ->> 'valida_hasta')::timestamptz >
          (politica ->> 'arrendamiento_vence_en')::timestamptz THEN
        RETURN false;
    END IF;

    RETURN true;
END
$funcion$;

DROP FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    jsonb, jsonb, bytea, bytea, bytea, bytea, bytea
);

CREATE FUNCTION vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
    p_confirmacion jsonb,
    p_prueba jsonb,
    p_evidencia_cifrado jsonb,
    p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_aad_canonica bytea,
    p_material_clave_envuelto bytea,
    p_nonce bytea,
    p_texto_cifrado bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb,
    requiere_revalidacion_kms boolean, preparacion_ref text,
    recibo_cuerpo jsonb, cuerpo_recibo_canonico bytea,
    huella_cuerpo_recibo_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_evidencia jsonb := p_evidencia_cifrado;
    v_perfil jsonb := v_evidencia -> 'perfil';
    v_envoltura jsonb := v_evidencia -> 'envoltura_clave';
    v_atestacion jsonb := v_evidencia -> 'atestacion_kms';
    v_confirmacion_legada jsonb;
    v_envoltura_legada jsonb;
    v_resultado record;
    v_identidad_primaria jsonb;
    v_preparacion_ref text;
    v_recibo_cuerpo jsonb;
    v_transaccion_bd bigint;
    v_presupuesto record;
    v_preparada_en timestamptz;
    v_ahora timestamptz;
BEGIN
    SELECT p.* INTO STRICT v_presupuesto
      FROM vec_bolsa_convocatorias.presupuesto_confirmacion_kms_borrador_v1()
           AS p;
    IF v_presupuesto.perfil_presupuesto IS DISTINCT FROM 'desarrollo:t20'
       OR v_presupuesto.espera_not_before_milisegundos IS NULL
       OR v_presupuesto.espera_not_before_milisegundos NOT BETWEEN 100 AND 5000
       OR v_presupuesto.tolerancia_cierre_milisegundos IS NULL
       OR v_presupuesto.tolerancia_cierre_milisegundos NOT BETWEEN 100 AND 1000
       OR v_presupuesto.espera_not_before_milisegundos +
          v_presupuesto.tolerancia_cierre_milisegundos > 10000
       OR current_setting('transaction_isolation') IS DISTINCT FROM 'serializable'
       OR vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.evidencia_cifrado_kms_borrador_valida(
           p_evidencia_cifrado, p_confirmacion, p_material_canonico,
           p_aad_canonica, p_material_clave_envuelto, p_nonce,
           p_texto_cifrado
       ) IS NOT TRUE
       OR v_perfil ->> 'algoritmo_aead' IS DISTINCT FROM 'A256GCM'
       OR v_perfil ->> 'algoritmo_envoltura_clave' IS DISTINCT FROM 'A256KW'
       OR octet_length(p_texto_cifrado) < 16
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           p_confirmacion -> 'sellado_motivo' ->> 'accion',
           'version_convocatoria_gobernada',
           p_confirmacion -> 'proyeccion_ligera' ->> 'referencia',
           'gobierno_convocatorias',
           '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
           clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmacion KMS de borrador no revalidada';
    END IF;

    -- El nucleo legado se usa solo como motor CAS/atomicidad. Su envoltura se
    -- deriva de los bytes modernos ya validados; ningun metadato criptografico
    -- elegido por el llamador atraviesa esta traduccion.
    v_envoltura_legada := jsonb_build_object(
        'algoritmo', 'A256GCM',
        'clave_cifrado_ref', v_envoltura ->> 'clave_maestra_ref',
        'generacion_clave', (v_envoltura ->> 'version_clave')::bigint,
        'nonce_hex', encode(p_nonce, 'hex'),
        'etiqueta_autenticacion_hex', encode(substring(
            p_texto_cifrado FROM octet_length(p_texto_cifrado) - 15 FOR 16
        ), 'hex'),
        'atestacion_cifrado_ref', v_atestacion ->> 'atestacion_ref',
        'huella_atestacion_cifrado_sha256',
            v_atestacion -> 'firma' ->> 'huella_preimagen_sha256',
        'huella_sobre_cifrado_sha256',
            encode(sha256(p_texto_cifrado), 'hex')
    );
    v_confirmacion_legada := jsonb_set(
        p_confirmacion, '{envoltura_cifrado}', v_envoltura_legada, false
    );

    SELECT c.* INTO STRICT v_resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          v_confirmacion_legada, p_material_canonico,
          p_version_canonica, p_texto_cifrado
      ) AS c;

    IF v_resultado.resultado = 'idempotencia_reutilizada' THEN
        IF v_resultado.estado_diario = 'confirmado'
           AND (
               v_resultado.recibo IS NULL
               OR v_resultado.recibo -> 'procedencia' IS NULL
               OR v_resultado.recibo -> 'acreditacion_kms' IS NULL
               OR NOT EXISTS (
                   SELECT 1
                     FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
                    WHERE a.recibo_ref = v_resultado.recibo ->> 'recibo_ref'
                      AND a.huella_recibo_sha256 = encode(
                          sha256(convert_to(v_resultado.recibo::text, 'UTF8')),
                          'hex'
                      )
               )
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'replay confirmado sin acreditacion KMS durable';
        END IF;
        RETURN QUERY SELECT
            v_resultado.resultado::text,
            v_resultado.estado_diario::text,
            v_resultado.revision_diario::bigint,
            v_resultado.cercado::bigint,
            v_resultado.transaccion_ref::text,
            v_resultado.accion::text,
            v_resultado.estado_principal_ref::text,
            v_resultado.estado_principal_revision::bigint,
            v_resultado.estado_principal_huella_sha256::text,
            v_resultado.auditoria_ref::text,
            v_resultado.huella_auditoria_sha256::text,
            v_resultado.evento_outbox_ref::text,
            v_resultado.huella_evento_outbox_sha256::text,
            v_resultado.confirmada_en::timestamptz,
            v_resultado.recibo::jsonb, false, NULL::text, NULL::jsonb,
            NULL::bytea, NULL::text;
        RETURN;
    END IF;

    IF v_resultado.resultado <> 'confirmada' THEN
        RETURN QUERY SELECT
            v_resultado.resultado::text,
            v_resultado.estado_diario::text,
            v_resultado.revision_diario::bigint,
            v_resultado.cercado::bigint,
            v_resultado.transaccion_ref::text,
            v_resultado.accion::text,
            v_resultado.estado_principal_ref::text,
            v_resultado.estado_principal_revision::bigint,
            v_resultado.estado_principal_huella_sha256::text,
            v_resultado.auditoria_ref::text,
            v_resultado.huella_auditoria_sha256::text,
            v_resultado.evento_outbox_ref::text,
            v_resultado.huella_evento_outbox_sha256::text,
            v_resultado.confirmada_en::timestamptz,
            v_resultado.recibo::jsonb, false, NULL::text, NULL::jsonb,
            NULL::bytea, NULL::text;
        RETURN;
    END IF;

    SELECT jsonb_build_object(
               'localizador', jsonb_build_object(
                   'version_esquema', h.localizador_esquema_version,
                   'dominio', 'localizador',
                   'clave_ref', h.localizador_clave_ref,
                   'generacion_clave', h.localizador_generacion_clave,
                   'hmac_sha256', encode(h.localizador_hmac, 'hex')
               ),
               'huella_solicitud', jsonb_build_object(
                   'version_esquema', h.huella_esquema_version,
                   'dominio', 'huella_solicitud',
                   'clave_ref', h.huella_clave_ref,
                   'generacion_clave', h.huella_generacion_clave,
                   'hmac_sha256', encode(h.huella_hmac, 'hex')
               )
           )
      INTO STRICT v_identidad_primaria
      FROM vec_bolsa_convocatorias.diario_borrador_version h
     WHERE h.transaccion_ref = v_resultado.transaccion_ref
       AND h.estado = 'confirmado'
       AND h.revision = v_resultado.revision_diario;

    IF v_identidad_primaria IS DISTINCT FROM p_confirmacion -> 'identidad'
       OR v_evidencia -> 'politica' -> 'identidad_primaria'
          IS DISTINCT FROM v_identidad_primaria
       OR v_evidencia -> 'evidencia_perfil' -> 'identidad_primaria'
          IS DISTINCT FROM v_identidad_primaria THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'identidad primaria KMS no coincide con el diario';
    END IF;

    v_transaccion_bd := txid_current();
    v_preparada_en := date_trunc('microseconds', clock_timestamp());
    -- PostgreSQL fija el not-before. El adaptador puede esperar fuera de SQL,
    -- respetando context.Context, pero no puede elegir ni ampliar el plazo.
    v_ahora := v_preparada_en +
        v_presupuesto.espera_not_before_milisegundos *
        interval '1 millisecond';
    IF v_ahora >=
          (p_evidencia_cifrado -> 'politica' ->>
           'arrendamiento_vence_en')::timestamptz
       OR v_ahora >= (v_atestacion ->> 'valida_hasta')::timestamptz
       OR v_ahora >=
          (p_evidencia_cifrado -> 'politica' ->>
           'valida_hasta')::timestamptz
       OR v_ahora >=
          (p_evidencia_cifrado -> 'evidencia_perfil' ->>
           'valida_hasta')::timestamptz
       OR v_ahora >=
          (p_confirmacion -> 'sellado_motivo' ->>
           'atestacion_valida_hasta')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '57014',
            MESSAGE = 'presupuesto KMS excede la vigencia del cierre';
    END IF;
    v_preparacion_ref := 'preparacion-kms-borrador-' || encode(sha256(
        convert_to(
            v_resultado.transaccion_ref || ':' ||
            v_resultado.recibo ->> 'recibo_ref' || ':' ||
            v_transaccion_bd::text || ':' ||
            v_evidencia -> 'aad' ->> 'huella_sha256' || ':' ||
            v_evidencia -> 'envoltura_clave' ->>
                'huella_envoltura_sha256' || ':' ||
            v_evidencia -> 'sobre' ->> 'huella_sobre_sha256',
            'UTF8'
        )
    ), 'hex');
    v_recibo_cuerpo := v_resultado.recibo || jsonb_build_object(
        'identidad', v_identidad_primaria,
        'procedencia', v_evidencia -> 'procedencia',
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );

    INSERT INTO vec_bolsa_convocatorias.cifrado_kms_borrador(
        convocatoria_id, secuencia, revision, transaccion_ref,
        perfil_ref, perfil_version, huella_perfil_sha256,
        algoritmo_aead, algoritmo_envoltura_clave,
        aad_canonica, huella_aad_sha256, material_clave_envuelto,
        huella_envoltura_sha256, nonce, contenido_cifrado,
        huella_contenido_cifrado_sha256, huella_sobre_sha256,
        politica, evidencia_perfil, atestacion_kms, procedencia,
        perfil_presupuesto, espera_not_before_milisegundos,
        tolerancia_cierre_milisegundos, registrada_en
    ) VALUES (
        p_confirmacion -> 'proyeccion_ligera' ->> 'convocatoria_id',
        (p_confirmacion -> 'proyeccion_ligera' ->> 'secuencia')::bigint,
        (p_confirmacion -> 'proyeccion_ligera' ->> 'revision')::bigint,
        v_resultado.transaccion_ref,
        v_perfil ->> 'referencia', (v_perfil ->> 'version')::bigint,
        v_perfil ->> 'huella_contenido_sha256',
        v_perfil ->> 'algoritmo_aead',
        v_perfil ->> 'algoritmo_envoltura_clave',
        p_aad_canonica, v_evidencia -> 'aad' ->> 'huella_sha256',
        p_material_clave_envuelto,
        v_envoltura ->> 'huella_envoltura_sha256', p_nonce,
        p_texto_cifrado, encode(sha256(p_texto_cifrado), 'hex'),
        v_evidencia -> 'sobre' ->> 'huella_sobre_sha256',
        v_evidencia -> 'politica', v_evidencia -> 'evidencia_perfil',
        v_atestacion, v_evidencia -> 'procedencia',
        v_presupuesto.perfil_presupuesto,
        v_presupuesto.espera_not_before_milisegundos,
        v_presupuesto.tolerancia_cierre_milisegundos,
        v_preparada_en
    );

    INSERT INTO vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador(
        preparacion_ref, transaccion_bd,
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac,
        revision_diario, cercado, convocatoria_id,
        secuencia_convocatoria, revision_convocatoria,
        transaccion_ref, recibo_ref, auditoria_ref,
        huella_auditoria_sha256, evento_outbox_ref,
        huella_evento_outbox_sha256, confirmacion_solicitada_en,
        confirmada_en, perfil_presupuesto,
        espera_not_before_milisegundos, tolerancia_cierre_milisegundos,
        recibo_cuerpo, creada_en
    )
    SELECT
        v_preparacion_ref, v_transaccion_bd,
        h.localizador_esquema_version, h.localizador_clave_ref,
        h.localizador_generacion_clave, h.localizador_hmac,
        h.revision, h.cercado,
        p_confirmacion -> 'proyeccion_ligera' ->> 'convocatoria_id',
        (p_confirmacion -> 'proyeccion_ligera' ->> 'secuencia')::bigint,
        (p_confirmacion -> 'proyeccion_ligera' ->> 'revision')::bigint,
        v_resultado.transaccion_ref, v_resultado.recibo ->> 'recibo_ref',
        v_resultado.auditoria_ref, v_resultado.huella_auditoria_sha256,
        v_resultado.evento_outbox_ref,
        v_resultado.huella_evento_outbox_sha256,
        (p_confirmacion ->> 'solicitada_en')::timestamptz,
        v_ahora, v_presupuesto.perfil_presupuesto,
        v_presupuesto.espera_not_before_milisegundos,
        v_presupuesto.tolerancia_cierre_milisegundos,
        v_recibo_cuerpo, v_preparada_en
      FROM vec_bolsa_convocatorias.diario_borrador_version h
     WHERE h.transaccion_ref = v_resultado.transaccion_ref
       AND h.estado = 'confirmado'
       AND h.revision = v_resultado.revision_diario;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'preparacion KMS perdio el cierre del diario';
    END IF;

    RETURN QUERY SELECT
        'preparada'::text, 'en_curso'::text,
        v_resultado.revision_diario::bigint,
        v_resultado.cercado::bigint,
        v_resultado.transaccion_ref::text,
        v_resultado.accion::text,
        v_resultado.estado_principal_ref::text,
        v_resultado.estado_principal_revision::bigint,
        v_resultado.estado_principal_huella_sha256::text,
        v_resultado.auditoria_ref::text,
        v_resultado.huella_auditoria_sha256::text,
        v_resultado.evento_outbox_ref::text,
        v_resultado.huella_evento_outbox_sha256::text,
        v_ahora::timestamptz,
        NULL::jsonb, true, v_preparacion_ref, v_recibo_cuerpo,
        vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(
            v_recibo_cuerpo
        ),
        encode(sha256(
            vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(
                v_recibo_cuerpo
            )
        ), 'hex');
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    p_preparacion_ref text,
    p_acreditacion_kms jsonb,
    p_cuerpo_recibo_canonico bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    p vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador%ROWTYPE;
    c vec_bolsa_convocatorias.cifrado_kms_borrador%ROWTYPE;
    h vec_bolsa_convocatorias.diario_borrador_version%ROWTYPE;
    a jsonb := p_acreditacion_kms;
    firma_a jsonb := p_acreditacion_kms -> 'firma_atestacion_kms';
    firma_b jsonb := p_acreditacion_kms -> 'firma_revalidacion_kms';
    cuerpo_esperado bytea;
    v_acreditacion_canonica bytea;
    v_preimagen_atestacion bytea;
    v_preimagen_revalidacion bytea;
    recibo_final jsonb;
    v_recibo_canonico bytea;
    v_huella_recibo text;
    ahora timestamptz;
BEGIN
    IF current_setting('transaction_isolation') IS DISTINCT FROM 'serializable'
       OR vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE
       OR p_preparacion_ref IS NULL
       OR p_acreditacion_kms IS NULL
       OR p_cuerpo_recibo_canonico IS NULL
       OR octet_length(p_cuerpo_recibo_canonico) NOT BETWEEN 2 AND 1048576
       OR p_preparacion_ref !~ '^preparacion-kms-borrador-[0-9a-f]{64}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre KMS de borrador no autorizado';
    END IF;

    SELECT x.* INTO STRICT p
      FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador x
     WHERE x.preparacion_ref = p_preparacion_ref
     FOR UPDATE;
    IF p.transaccion_bd IS DISTINCT FROM txid_current() THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'fase KMS B fuera de la transaccion preparada';
    END IF;

    SELECT x.* INTO STRICT c
      FROM vec_bolsa_convocatorias.cifrado_kms_borrador x
     WHERE x.transaccion_ref = p.transaccion_ref;
    SELECT x.* INTO STRICT h
      FROM vec_bolsa_convocatorias.diario_borrador_version x
     WHERE x.localizador_esquema_version =
               p.localizador_esquema_version
       AND x.localizador_clave_ref = p.localizador_clave_ref
       AND x.localizador_generacion_clave =
               p.localizador_generacion_clave
       AND x.localizador_hmac = p.localizador_hmac
       AND x.revision = p.revision_diario
     FOR UPDATE;

    cuerpo_esperado :=
        vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(
            p.recibo_cuerpo
        );
    ahora := date_trunc('microseconds', clock_timestamp());
    IF p_cuerpo_recibo_canonico IS DISTINCT FROM cuerpo_esperado
       OR encode(sha256(p_cuerpo_recibo_canonico), 'hex') IS DISTINCT FROM
          a ->> 'huella_cuerpo_recibo_sha256'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           a, ARRAY[
               'acreditacion_ref','arrendamiento_inicia_en',
               'arrendamiento_vence_en','atestacion_emitida_en',
               'atestacion_ref','atestacion_valida_hasta',
               'cercado','clave_maestra_ref','comprobacion_kms_ref',
               'confirmacion_solicitada_en','confirmada_en','esquema',
               'estado','firma_atestacion_kms','firma_revalidacion_kms',
               'huella_acreditacion_sha256','huella_aad',
               'huella_comprobacion_kms_sha256',
               'huella_cuerpo_recibo_sha256','huella_envoltura_sha256',
               'huella_sobre_sha256','identidad_primaria','perfil',
               'procedencia','recibo_ref','revalidacion_solicitada_en',
               'revalidada_en','revision_confirmada','revision_reserva',
               'transaccion_ref','verificador_ref','version_acreditacion',
               'version_atestacion','version_clave'
           ]
       ) IS NOT TRUE
       OR a ->> 'esquema' IS DISTINCT FROM
          'bolsa.convocatoria.borrador.acreditacion-kms-confirmacion.v1'
       OR a ->> 'estado' IS DISTINCT FROM 'confirmada'
       OR (a ->> 'version_acreditacion' ~ '^[1-9][0-9]{0,9}$')
          IS NOT TRUE
       OR a ->> 'atestacion_ref' IS DISTINCT FROM
          c.atestacion_kms ->> 'atestacion_ref'
       OR a ->> 'version_atestacion' IS DISTINCT FROM
          c.atestacion_kms ->> 'version_atestacion'
       OR a -> 'perfil' IS DISTINCT FROM c.atestacion_kms -> 'perfil'
       OR a ->> 'clave_maestra_ref' IS DISTINCT FROM
          c.atestacion_kms ->> 'clave_maestra_ref'
       OR a ->> 'version_clave' IS DISTINCT FROM
          c.atestacion_kms ->> 'version_clave'
       OR a ->> 'huella_aad' IS DISTINCT FROM c.huella_aad_sha256
       OR a ->> 'huella_envoltura_sha256' IS DISTINCT FROM
          c.huella_envoltura_sha256
       OR a ->> 'huella_sobre_sha256' IS DISTINCT FROM
          c.huella_sobre_sha256
       OR a -> 'procedencia' IS DISTINCT FROM c.procedencia
       OR a -> 'procedencia' IS DISTINCT FROM p.recibo_cuerpo -> 'procedencia'
       OR firma_a IS DISTINCT FROM c.atestacion_kms -> 'firma'
       OR vec_bolsa_convocatorias.firma_evidencia_borrador_valida(firma_a)
          IS NOT TRUE
       OR vec_bolsa_convocatorias.firma_evidencia_borrador_valida(firma_b)
          IS NOT TRUE
       OR firma_a ->> 'verificador_ref' IS NOT DISTINCT FROM
          firma_b ->> 'verificador_ref'
       OR firma_a ->> 'huella_clave_publica_sha256' IS NOT DISTINCT FROM
          firma_b ->> 'huella_clave_publica_sha256'
       OR a -> 'identidad_primaria' IS DISTINCT FROM
          p.recibo_cuerpo -> 'identidad'
       OR a ->> 'revision_reserva' IS DISTINCT FROM
          c.politica ->> 'revision'
       OR a ->> 'revision_confirmada' IS DISTINCT FROM
          p.revision_diario::text
       OR a ->> 'cercado' IS DISTINCT FROM p.cercado::text
       OR a ->> 'arrendamiento_inicia_en' IS DISTINCT FROM
          p.recibo_cuerpo ->> 'arrendamiento_inicia_en'
       OR a ->> 'arrendamiento_vence_en' IS DISTINCT FROM
          p.recibo_cuerpo ->> 'arrendamiento_vence_en'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
          a ->> 'comprobacion_kms_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
          a ->> 'huella_comprobacion_kms_sha256'
       ) IS NOT TRUE
       OR a ->> 'transaccion_ref' IS DISTINCT FROM p.transaccion_ref
       OR a ->> 'recibo_ref' IS DISTINCT FROM p.recibo_ref
       OR a ->> 'atestacion_emitida_en' IS DISTINCT FROM
          c.atestacion_kms ->> 'emitida_en'
       OR a ->> 'atestacion_valida_hasta' IS DISTINCT FROM
          c.atestacion_kms ->> 'valida_hasta'
       OR a ->> 'confirmacion_solicitada_en' IS DISTINCT FROM
          to_char(p.confirmacion_solicitada_en AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       OR a ->> 'confirmada_en' IS DISTINCT FROM
          to_char(p.confirmada_en AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       OR vec_bolsa_convocatorias.texto_opaco_valido(
          a ->> 'acreditacion_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
          a ->> 'verificador_ref', 512
       ) IS NOT TRUE
       OR a ->> 'verificador_ref' IN (
          a ->> 'acreditacion_ref', a ->> 'atestacion_ref',
          a ->> 'transaccion_ref', a ->> 'recibo_ref',
          firma_a ->> 'verificador_ref', firma_b ->> 'verificador_ref'
       )
       OR vec_bolsa_convocatorias.huella_sha256_valida(
          a ->> 'huella_acreditacion_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
          a ->> 'revalidacion_solicitada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
          a ->> 'revalidada_en'
       ) IS NOT TRUE
       OR (a ->> 'revalidacion_solicitada_en')::timestamptz <
          p.confirmacion_solicitada_en
       OR (a ->> 'revalidada_en')::timestamptz <
          (a ->> 'revalidacion_solicitada_en')::timestamptz
       OR (a ->> 'revalidada_en')::timestamptz <
          (c.atestacion_kms ->> 'emitida_en')::timestamptz
       OR (a ->> 'revalidada_en')::timestamptz >=
          (c.atestacion_kms ->> 'valida_hasta')::timestamptz
       OR p.confirmada_en < (a ->> 'revalidada_en')::timestamptz
       OR p.perfil_presupuesto IS DISTINCT FROM c.perfil_presupuesto
       OR p.espera_not_before_milisegundos IS DISTINCT FROM
          c.espera_not_before_milisegundos
       OR p.tolerancia_cierre_milisegundos IS DISTINCT FROM
          c.tolerancia_cierre_milisegundos
       OR p.confirmada_en > ahora
       OR ahora - p.confirmada_en >
          p.tolerancia_cierre_milisegundos * interval '1 millisecond'
       OR ahora >= h.arrendamiento_vence_en
       OR ahora >= (c.atestacion_kms ->> 'valida_hasta')::timestamptz
       OR ahora >= (c.politica ->> 'valida_hasta')::timestamptz
       OR ahora >= (c.evidencia_perfil ->> 'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'acreditacion KMS B no coincide con la preparacion';
    END IF;

    v_preimagen_atestacion :=
        vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(a);
    v_preimagen_revalidacion :=
        vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(a);
    IF v_preimagen_atestacion IS NULL
       OR v_preimagen_revalidacion IS NULL
       OR encode(sha256(v_preimagen_atestacion), 'hex') IS DISTINCT FROM
          firma_a ->> 'huella_preimagen_sha256'
       OR encode(sha256(v_preimagen_revalidacion), 'hex') IS DISTINCT FROM
          firma_b ->> 'huella_preimagen_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preimagenes de firmas KMS no coinciden';
    END IF;

    v_acreditacion_canonica :=
        vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(a);
    IF v_acreditacion_canonica IS NULL
       OR octet_length(v_acreditacion_canonica) NOT BETWEEN 2 AND 1048576
       OR encode(sha256(v_acreditacion_canonica), 'hex') IS DISTINCT FROM
          a ->> 'huella_acreditacion_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preimagen canonica KMS B no coincide con la acreditacion';
    END IF;

    recibo_final := p.recibo_cuerpo ||
        jsonb_build_object('acreditacion_kms', a);
    v_recibo_canonico := convert_to(recibo_final::text, 'UTF8');
    v_huella_recibo := encode(sha256(v_recibo_canonico), 'hex');

    INSERT INTO vec_bolsa_convocatorias.acreditacion_kms_borrador(
        acreditacion_ref, recibo_ref, transaccion_ref,
        convocatoria_id, secuencia, revision, auditoria_ref,
        evento_outbox_ref, cuerpo_recibo_canonico,
        huella_cuerpo_recibo_sha256, acreditacion,
        acreditacion_canonica, huella_acreditacion_sha256,
        recibo_canonico, huella_recibo_sha256, procedencia, registrada_en
    ) VALUES (
        a ->> 'acreditacion_ref', p.recibo_ref, p.transaccion_ref,
        p.convocatoria_id, p.secuencia_convocatoria,
        p.revision_convocatoria, p.auditoria_ref, p.evento_outbox_ref,
        p_cuerpo_recibo_canonico, a ->> 'huella_cuerpo_recibo_sha256',
        a, v_acreditacion_canonica, a ->> 'huella_acreditacion_sha256',
        v_recibo_canonico, v_huella_recibo, c.procedencia, ahora
    );

    UPDATE vec_bolsa_convocatorias.diario_borrador_version
       SET confirmada_en = p.confirmada_en,
           recibo_canonico = v_recibo_canonico,
           huella_recibo_sha256 = v_huella_recibo
     WHERE localizador_esquema_version = p.localizador_esquema_version
       AND localizador_clave_ref = p.localizador_clave_ref
       AND localizador_generacion_clave = p.localizador_generacion_clave
       AND localizador_hmac = p.localizador_hmac
       AND revision = p.revision_diario
       AND estado = 'confirmado'
       AND transaccion_ref = p.transaccion_ref;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS perdido al acreditar el recibo KMS';
    END IF;

    DELETE FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
     WHERE preparacion_ref = p.preparacion_ref
       AND transaccion_bd = txid_current();
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'preparacion KMS no consumida';
    END IF;

    RETURN QUERY SELECT
        'confirmada'::text, 'confirmado'::text,
        p.revision_diario, p.cercado, p.transaccion_ref,
        h.accion, h.estado_principal_ref, h.estado_principal_revision,
        h.estado_principal_huella_sha256, p.auditoria_ref,
        p.huella_auditoria_sha256, p.evento_outbox_ref,
        p.huella_evento_outbox_sha256, p.confirmada_en, recibo_final;
EXCEPTION WHEN NO_DATA_FOUND THEN
    RAISE EXCEPTION USING ERRCODE = 'P0002',
        MESSAGE = 'preparacion KMS inexistente o ya consumida';
END
$funcion$;

-- Autoridad de lectura separada de las fases A/B. No se amplia el helper de
-- 000003: una credencial de verificacion debe tener exactamente esta unica
-- membresia y no puede heredar capacidad de reserva, confirmacion o registro.
CREATE FUNCTION
vec_bolsa_convocatorias.identidad_runtime_verificador_recibo_valida()
RETURNS boolean
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    identidad record;
    objetivo record;
    numero_membresias bigint;
    membresia_exacta boolean;
BEGIN
    SELECT oid, rolcanlogin, rolinherit, rolsuper, rolcreatedb,
           rolcreaterole, rolreplication, rolbypassrls
      INTO identidad
      FROM pg_catalog.pg_roles
     WHERE rolname = session_user;
    SELECT oid, rolcanlogin, rolinherit, rolsuper, rolcreatedb,
           rolcreaterole, rolreplication, rolbypassrls
      INTO objetivo
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_bolsa_convocatorias_verificador_recibo';
    SELECT count(*), COALESCE(bool_and(
               m.roleid = objetivo.oid
               AND m.admin_option IS FALSE
               AND m.inherit_option IS TRUE
               AND m.set_option IS TRUE
           ), false)
      INTO numero_membresias, membresia_exacta
      FROM pg_catalog.pg_auth_members AS m
     WHERE m.member = identidad.oid;
    RETURN session_user IS DISTINCT FROM current_user
       AND identidad IS NOT NULL AND identidad.rolcanlogin
       AND identidad.rolinherit
       AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
       AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
       AND NOT identidad.rolbypassrls
       AND objetivo IS NOT NULL AND NOT objetivo.rolcanlogin
       AND objetivo.rolinherit
       AND NOT objetivo.rolsuper AND NOT objetivo.rolcreatedb
       AND NOT objetivo.rolcreaterole AND NOT objetivo.rolreplication
       AND NOT objetivo.rolbypassrls
       AND numero_membresias = 1 AND membresia_exacta
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = objetivo.oid
       );
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Relectura posterior al COMMIT. Devuelve las preimagenes exactas para que un
-- verificador criptografico independiente valide las firmas A/B; PostgreSQL
-- solo demuestra que los mismos bytes y enlaces permanecen durables.
CREATE FUNCTION vec_bolsa_convocatorias.verificar_recibo_borrador_v1(
    p_recibo_ref text,
    p_transaccion_ref text,
    p_huella_recibo_sha256 text
)
RETURNS TABLE (
    recibo_ref text,
    transaccion_ref text,
    convocatoria_id text,
    secuencia bigint,
    revision bigint,
    recibo jsonb,
    recibo_canonico bytea,
    huella_recibo_sha256 text,
    cuerpo_recibo_canonico bytea,
    huella_cuerpo_recibo_sha256 text,
    acreditacion jsonb,
    acreditacion_canonica bytea,
    huella_acreditacion_sha256 text,
    preimagen_atestacion_kms bytea,
    preimagen_revalidacion_kms bytea,
    verificada_en timestamptz
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    a vec_bolsa_convocatorias.acreditacion_kms_borrador%ROWTYPE;
    c vec_bolsa_convocatorias.cifrado_kms_borrador%ROWTYPE;
    h vec_bolsa_convocatorias.diario_borrador_version%ROWTYPE;
    b vec_bolsa_convocatorias.borrador_convocatoria_version%ROWTYPE;
    au vec_bolsa_convocatorias.auditoria_borrador%ROWTYPE;
    o vec_bolsa_convocatorias.outbox_borrador%ROWTYPE;
    v_recibo jsonb;
    v_cuerpo bytea;
    v_acreditacion bytea;
    v_preimagen_a bytea;
    v_preimagen_b bytea;
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_verificador_recibo_valida()
          IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'verificacion durable de recibo no autorizada';
    END IF;
    IF vec_bolsa_convocatorias.texto_opaco_valido(p_recibo_ref, 512)
          IS NOT TRUE
       OR p_recibo_ref !~ '^recibo-borrador-[0-9a-f]{64}$'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
              p_transaccion_ref, 512
          ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
              p_huella_recibo_sha256
          ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'identidad de recibo no canonica';
    END IF;

    SELECT x.* INTO STRICT a
      FROM vec_bolsa_convocatorias.acreditacion_kms_borrador AS x
     WHERE x.recibo_ref = p_recibo_ref
       AND x.transaccion_ref = p_transaccion_ref
       AND x.huella_recibo_sha256 = p_huella_recibo_sha256;
    SELECT x.* INTO STRICT c
      FROM vec_bolsa_convocatorias.cifrado_kms_borrador AS x
     WHERE x.convocatoria_id = a.convocatoria_id
       AND x.secuencia = a.secuencia
       AND x.revision = a.revision
       AND x.transaccion_ref = a.transaccion_ref;
    SELECT x.* INTO STRICT h
      FROM vec_bolsa_convocatorias.diario_borrador_version AS x
     WHERE x.recibo_ref = a.recibo_ref
       AND x.transaccion_ref = a.transaccion_ref
       AND x.estado = 'confirmado';
    SELECT x.* INTO STRICT b
      FROM vec_bolsa_convocatorias.borrador_convocatoria_version AS x
     WHERE x.convocatoria_id = a.convocatoria_id
       AND x.secuencia = a.secuencia
       AND x.revision = a.revision;
    SELECT x.* INTO STRICT au
      FROM vec_bolsa_convocatorias.auditoria_borrador AS x
     WHERE x.auditoria_ref = a.auditoria_ref
       AND x.transaccion_ref = a.transaccion_ref;
    SELECT x.* INTO STRICT o
      FROM vec_bolsa_convocatorias.outbox_borrador AS x
     WHERE x.evento_ref = a.evento_outbox_ref
       AND x.transaccion_ref = a.transaccion_ref
       AND x.auditoria_ref = a.auditoria_ref
       AND x.convocatoria_id = a.convocatoria_id
       AND x.secuencia_convocatoria = a.secuencia
       AND x.revision = a.revision;

    BEGIN
        v_recibo := convert_from(a.recibo_canonico, 'UTF8')::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'recibo durable no es JSON canonico';
    END;
    v_cuerpo :=
        vec_bolsa_convocatorias.cuerpo_recibo_canonico_borrador_v1(v_recibo);
    v_acreditacion :=
        vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
            a.acreditacion
        );
    v_preimagen_a :=
        vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
            a.acreditacion
        );
    v_preimagen_b :=
        vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
            a.acreditacion
        );

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_recibo, ARRAY[
               'acreditacion_kms','accion','arrendamiento_inicia_en',
               'arrendamiento_vence_en','auditoria_ref','cercado_confirmado',
               'confirmada_en','decision','esquema','estado_principal',
               'evento_outbox_ref','huella_auditoria_sha256',
               'huella_evento_outbox_sha256','identidad','procedencia',
               'recibo_ref','revision_confirmada','sellado_motivo',
               'transaccion_ref'
           ]
       ) IS NOT TRUE
       OR convert_to(v_recibo::text, 'UTF8') IS DISTINCT FROM a.recibo_canonico
       OR encode(sha256(a.recibo_canonico), 'hex') IS DISTINCT FROM
          a.huella_recibo_sha256
       OR h.recibo_canonico IS DISTINCT FROM a.recibo_canonico
       OR h.huella_recibo_sha256 IS DISTINCT FROM a.huella_recibo_sha256
       OR v_recibo ->> 'recibo_ref' IS DISTINCT FROM a.recibo_ref
       OR v_recibo ->> 'transaccion_ref' IS DISTINCT FROM a.transaccion_ref
       OR v_recibo -> 'acreditacion_kms' IS DISTINCT FROM a.acreditacion
       OR v_recibo -> 'procedencia' IS DISTINCT FROM a.procedencia
       OR a.acreditacion -> 'procedencia' IS DISTINCT FROM a.procedencia
       OR c.procedencia IS DISTINCT FROM a.procedencia
       OR v_cuerpo IS NULL
       OR v_cuerpo IS DISTINCT FROM a.cuerpo_recibo_canonico
       OR encode(sha256(v_cuerpo), 'hex') IS DISTINCT FROM
          a.huella_cuerpo_recibo_sha256
       OR a.acreditacion ->> 'huella_cuerpo_recibo_sha256' IS DISTINCT FROM
          a.huella_cuerpo_recibo_sha256
       OR v_acreditacion IS NULL
       OR v_acreditacion IS DISTINCT FROM a.acreditacion_canonica
       OR encode(sha256(v_acreditacion), 'hex') IS DISTINCT FROM
          a.huella_acreditacion_sha256
       OR a.acreditacion ->> 'huella_acreditacion_sha256' IS DISTINCT FROM
          a.huella_acreditacion_sha256
       OR encode(sha256(v_preimagen_a), 'hex') IS DISTINCT FROM
          a.acreditacion -> 'firma_atestacion_kms' ->>
              'huella_preimagen_sha256'
       OR encode(sha256(v_preimagen_b), 'hex') IS DISTINCT FROM
          a.acreditacion -> 'firma_revalidacion_kms' ->>
              'huella_preimagen_sha256'
       OR a.acreditacion ->> 'recibo_ref' IS DISTINCT FROM a.recibo_ref
       OR a.acreditacion ->> 'transaccion_ref' IS DISTINCT FROM
          a.transaccion_ref
       OR a.acreditacion ->> 'huella_aad' IS DISTINCT FROM
          c.huella_aad_sha256
       OR a.acreditacion ->> 'huella_envoltura_sha256' IS DISTINCT FROM
          c.huella_envoltura_sha256
       OR a.acreditacion ->> 'huella_sobre_sha256' IS DISTINCT FROM
          c.huella_sobre_sha256
       OR b.sobre_cifrado IS DISTINCT FROM c.contenido_cifrado
       OR b.nonce IS DISTINCT FROM c.nonce
       OR h.auditoria_ref IS DISTINCT FROM au.auditoria_ref
       OR h.huella_auditoria_sha256 IS DISTINCT FROM
          au.huella_auditoria_sha256
       OR h.evento_outbox_ref IS DISTINCT FROM o.evento_ref
       OR h.huella_evento_outbox_sha256 IS DISTINCT FROM
          o.huella_evento_sha256
       OR v_recibo ->> 'auditoria_ref' IS DISTINCT FROM au.auditoria_ref
       OR v_recibo ->> 'huella_auditoria_sha256' IS DISTINCT FROM
          au.huella_auditoria_sha256
       OR v_recibo ->> 'evento_outbox_ref' IS DISTINCT FROM o.evento_ref
       OR v_recibo ->> 'huella_evento_outbox_sha256' IS DISTINCT FROM
          o.huella_evento_sha256
       OR v_recibo ->> 'confirmada_en' IS DISTINCT FROM
          to_char(h.confirmada_en AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       OR a.acreditacion ->> 'confirmada_en' IS DISTINCT FROM
          v_recibo ->> 'confirmada_en'
       OR h.confirmada_en - c.registrada_en IS DISTINCT FROM
          c.espera_not_before_milisegundos * interval '1 millisecond'
       OR a.registrada_en < h.confirmada_en
       OR a.registrada_en - h.confirmada_en >
          c.tolerancia_cierre_milisegundos * interval '1 millisecond'
       OR EXISTS (
           SELECT 1
             FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
                  AS p
            WHERE p.transaccion_ref = a.transaccion_ref
               OR p.recibo_ref = a.recibo_ref
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'recibo durable no supera la relectura exacta';
    END IF;

    RETURN QUERY SELECT
        a.recibo_ref, a.transaccion_ref, a.convocatoria_id,
        a.secuencia, a.revision, v_recibo, a.recibo_canonico,
        a.huella_recibo_sha256, a.cuerpo_recibo_canonico,
        a.huella_cuerpo_recibo_sha256, a.acreditacion,
        a.acreditacion_canonica, a.huella_acreditacion_sha256,
        v_preimagen_a, v_preimagen_b,
        date_trunc('microseconds', statement_timestamp());
EXCEPTION WHEN NO_DATA_FOUND THEN
    RAISE EXCEPTION USING ERRCODE = 'P0002',
        MESSAGE = 'recibo durable inexistente o no coincidente';
END
$funcion$;

CREATE FUNCTION
vec_bolsa_convocatorias.validar_actualizacion_acreditacion_diario_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    preparacion record;
    acreditacion record;
    recibo jsonb;
BEGIN
    IF TG_OP <> 'UPDATE'
       OR current_user <> 'vec_bolsa_convocatorias_propietario'
       OR OLD.estado <> 'confirmado' OR NEW.estado <> 'confirmado'
       OR (to_jsonb(NEW) - ARRAY[
               'confirmada_en','recibo_canonico','huella_recibo_sha256'
           ]) IS DISTINCT FROM
          (to_jsonb(OLD) - ARRAY[
               'confirmada_en','recibo_canonico','huella_recibo_sha256'
           ])
       OR NEW.confirmada_en < OLD.confirmada_en
       OR NEW.recibo_canonico IS NULL
       OR encode(sha256(NEW.recibo_canonico), 'hex') <>
          NEW.huella_recibo_sha256 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia de diario KMS inmutable';
    END IF;
    SELECT p.* INTO STRICT preparacion
      FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador p
     WHERE p.transaccion_ref = NEW.transaccion_ref
       AND p.recibo_ref = NEW.recibo_ref
       AND p.revision_diario = NEW.revision
       AND p.transaccion_bd = txid_current();
    SELECT a.* INTO STRICT acreditacion
      FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
     WHERE a.transaccion_ref = NEW.transaccion_ref
       AND a.recibo_ref = NEW.recibo_ref
       AND a.recibo_canonico = NEW.recibo_canonico
       AND a.huella_recibo_sha256 = NEW.huella_recibo_sha256;
    recibo := convert_from(NEW.recibo_canonico, 'UTF8')::jsonb;
    IF NEW.confirmada_en <> preparacion.confirmada_en
       OR recibo -> 'procedencia' IS DISTINCT FROM acreditacion.procedencia
       OR recibo -> 'acreditacion_kms' IS DISTINCT FROM
          acreditacion.acreditacion THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'recibo KMS no ligado a su acreditacion';
    END IF;
    RETURN NEW;
EXCEPTION WHEN NO_DATA_FOUND THEN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'actualizacion de diario sin preparacion KMS activa';
END
$funcion$;

DROP TRIGGER diario_borrador_version_inmutable
    ON vec_bolsa_convocatorias.diario_borrador_version;
CREATE TRIGGER diario_borrador_version_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_convocatorias.diario_borrador_version
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_actualizacion_acreditacion_diario_v1();

CREATE FUNCTION vec_bolsa_convocatorias.validar_consumo_preparacion_kms_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF OLD.transaccion_bd <> txid_current()
       OR NOT EXISTS (
           SELECT 1
             FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
            WHERE a.transaccion_ref = OLD.transaccion_ref
              AND a.recibo_ref = OLD.recibo_ref
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'preparacion KMS no consumible';
    END IF;
    RETURN OLD;
END
$funcion$;

CREATE TRIGGER preparacion_confirmacion_kms_consumo
    BEFORE DELETE
    ON vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_consumo_preparacion_kms_v1();

CREATE FUNCTION vec_bolsa_convocatorias.exigir_cierre_preparacion_kms_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador p
         WHERE p.preparacion_ref = NEW.preparacion_ref
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'transaccion con preparacion KMS sin fase B';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE CONSTRAINT TRIGGER preparacion_confirmacion_kms_debe_cerrarse
    AFTER INSERT
    ON vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.exigir_cierre_preparacion_kms_v1();

CREATE FUNCTION vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    encontrada boolean;
BEGIN
    CASE TG_TABLE_NAME
    WHEN 'borrador_convocatoria_version' THEN
        SELECT EXISTS (
            SELECT 1
              FROM vec_bolsa_convocatorias.cifrado_kms_borrador c
              JOIN vec_bolsa_convocatorias.acreditacion_kms_borrador a
                ON a.convocatoria_id = c.convocatoria_id
               AND a.secuencia = c.secuencia
               AND a.revision = c.revision
             WHERE a.convocatoria_id = NEW.convocatoria_id
               AND a.secuencia = NEW.secuencia
               AND a.revision = NEW.revision
               -- La tabla base de 000003 conserva el ciphertext legado. El
               -- compañero moderno debe comprometer exactamente esos bytes y
               -- el mismo nonce; no basta una acreditacion con igual PK.
               AND c.contenido_cifrado = NEW.sobre_cifrado
               AND c.nonce = NEW.nonce
               AND c.huella_contenido_cifrado_sha256 =
                   encode(sha256(NEW.sobre_cifrado), 'hex')
        ) INTO encontrada;
    WHEN 'auditoria_borrador' THEN
        SELECT EXISTS (
            SELECT 1
              FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
             WHERE a.auditoria_ref = NEW.auditoria_ref
               AND a.transaccion_ref = NEW.transaccion_ref
        ) INTO encontrada;
    WHEN 'outbox_borrador' THEN
        SELECT EXISTS (
            SELECT 1
              FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
             WHERE a.evento_outbox_ref = NEW.evento_ref
               AND a.transaccion_ref = NEW.transaccion_ref
        ) INTO encontrada;
    WHEN 'diario_borrador_version' THEN
        IF NEW.estado <> 'confirmado' THEN
            RETURN NULL;
        END IF;
        SELECT EXISTS (
            SELECT 1
              FROM vec_bolsa_convocatorias.acreditacion_kms_borrador a
             WHERE a.recibo_ref = NEW.recibo_ref
               AND a.transaccion_ref = NEW.transaccion_ref
               AND a.recibo_canonico = (
                   SELECT h.recibo_canonico
                     FROM vec_bolsa_convocatorias.diario_borrador_version h
                    WHERE h.localizador_esquema_version =
                              NEW.localizador_esquema_version
                      AND h.localizador_clave_ref = NEW.localizador_clave_ref
                      AND h.localizador_generacion_clave =
                              NEW.localizador_generacion_clave
                      AND h.localizador_hmac = NEW.localizador_hmac
                      AND h.revision = NEW.revision
               )
        ) INTO encontrada;
    ELSE
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'trigger KMS instalado en tabla no prevista';
    END CASE;
    IF encontrada IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'efecto de borrador sin acreditacion KMS atomica';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE CONSTRAINT TRIGGER borrador_version_exige_acreditacion_kms
    AFTER INSERT ON vec_bolsa_convocatorias.borrador_convocatoria_version
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1();
CREATE CONSTRAINT TRIGGER auditoria_borrador_exige_acreditacion_kms
    AFTER INSERT ON vec_bolsa_convocatorias.auditoria_borrador
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1();
CREATE CONSTRAINT TRIGGER outbox_borrador_exige_acreditacion_kms
    AFTER INSERT ON vec_bolsa_convocatorias.outbox_borrador
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1();
CREATE CONSTRAINT TRIGGER diario_borrador_exige_acreditacion_kms
    AFTER INSERT ON vec_bolsa_convocatorias.diario_borrador_version
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.exigir_acreditacion_kms_durable_v1();

CREATE TRIGGER cifrado_kms_borrador_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_convocatorias.cifrado_kms_borrador
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
CREATE TRIGGER cifrado_kms_borrador_no_truncar
    BEFORE TRUNCATE ON vec_bolsa_convocatorias.cifrado_kms_borrador
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
CREATE TRIGGER acreditacion_kms_borrador_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_convocatorias.acreditacion_kms_borrador
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
CREATE TRIGGER acreditacion_kms_borrador_no_truncar
    BEFORE TRUNCATE ON vec_bolsa_convocatorias.acreditacion_kms_borrador
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_convocatorias.rechazar_mutacion_inmutable();

COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

ALTER TABLE vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador
    FOR ALL TO vec_bolsa_convocatorias_propietario
    USING (current_user = 'vec_bolsa_convocatorias_propietario')
    WITH CHECK (current_user = 'vec_bolsa_convocatorias_propietario');

ALTER TABLE vec_bolsa_convocatorias.cifrado_kms_borrador
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_convocatorias.cifrado_kms_borrador
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_bolsa_convocatorias.cifrado_kms_borrador
    FOR ALL TO vec_bolsa_convocatorias_propietario
    USING (current_user = 'vec_bolsa_convocatorias_propietario')
    WITH CHECK (current_user = 'vec_bolsa_convocatorias_propietario');

ALTER TABLE vec_bolsa_convocatorias.acreditacion_kms_borrador
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_convocatorias.acreditacion_kms_borrador
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_bolsa_convocatorias.acreditacion_kms_borrador
    FOR ALL TO vec_bolsa_convocatorias_propietario
    USING (current_user = 'vec_bolsa_convocatorias_propietario')
    WITH CHECK (current_user = 'vec_bolsa_convocatorias_propietario');

REVOKE ALL ON TABLE
    vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador,
    vec_bolsa_convocatorias.cifrado_kms_borrador,
    vec_bolsa_convocatorias.acreditacion_kms_borrador
    FROM PUBLIC,
         vec_bolsa_convocatorias_ejecutor_consulta,
         vec_bolsa_convocatorias_proyector_gobierno,
         vec_bolsa_convocatorias_registrador_atestacion,
         vec_bolsa_convocatorias_verificador_recibo;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;

DO $cerrar_funciones_runtime$
DECLARE
    rol text;
    funcion record;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion',
        'vec_bolsa_convocatorias_verificador_recibo'
    ] LOOP
        FOR funcion IN
            SELECT p.oid::regprocedure AS firma
              FROM pg_catalog.pg_proc p
              JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND p.proname IN (
                   'base64url_sin_relleno_valido',
                   'procedencia_borrador_valida',
                   'perfil_cifrado_borrador_valido',
                   'firma_evidencia_borrador_valida',
                   'presupuesto_confirmacion_kms_borrador_v1',
                   'huella_envoltura_clave_borrador_v1',
                   'huella_sobre_aead_borrador_v1',
                   'aad_canonica_borrador_v1',
                   'instante_rfc3339nano_borrador_v1',
                   'cuerpo_recibo_canonico_borrador_v1',
                   'acreditacion_kms_canonica_borrador_v1',
                   'atestacion_kms_preimagen_borrador_v1',
                   'firma_base64url_borrador_v1',
                   'revalidacion_kms_preimagen_borrador_v1',
                   'evidencia_cifrado_kms_borrador_valida',
                   'preparar_confirmacion_borrador_v1',
                   'confirmar_borrador_v1',
                   'identidad_runtime_verificador_recibo_valida',
                   'verificar_recibo_borrador_v1',
                   'validar_actualizacion_acreditacion_diario_v1',
                   'validar_consumo_preparacion_kms_v1',
                   'exigir_cierre_preparacion_kms_v1',
                   'exigir_acreditacion_kms_durable_v1'
               )
        LOOP
            EXECUTE format(
                'REVOKE ALL ON FUNCTION %s FROM %I', funcion.firma, rol
            );
        END LOOP;
    END LOOP;
END
$cerrar_funciones_runtime$;

GRANT USAGE ON SCHEMA vec_bolsa_convocatorias
    TO vec_bolsa_convocatorias_proyector_gobierno,
       vec_bolsa_convocatorias_verificador_recibo;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ),
    vec_bolsa_convocatorias.confirmar_borrador_v1(
        text,jsonb,bytea
    ) TO vec_bolsa_convocatorias_proyector_gobierno;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text)
    TO vec_bolsa_convocatorias_verificador_recibo;

COMMENT ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) IS
    'Fase A transaccional: relee PDP/diario/CAS, fija desde PostgreSQL un not-before de desarrollo de 1000 ms, persiste solo AAD, DEK envuelta y ciphertext tentativos, y devuelve el cuerpo canonico que KMS B debe firmar. Un COMMIT sin la fase B aborta por trigger diferido.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.confirmar_borrador_v1(
        text,jsonb,bytea
    ) IS
    'Fase B en la misma transaccion SERIALIZABLE: valida txid, cuerpo, enlaces y huellas; persiste acreditacion, recibo, auditoria y outbox atomicos. La criptografia de firmas se verifica fuera de SQL con credencial independiente.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text) IS
    'Relectura exacta posterior al COMMIT para una credencial exclusiva sin capacidad A/B. Devuelve recibo, acreditacion y preimagenes; las firmas se validan en el verificador criptografico independiente.';

COMMIT;
