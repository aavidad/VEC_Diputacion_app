-- T17 B1: esquema durable, límites, cadena de historia y aislamiento.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_importacion_convoca') IS NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace AS esquema
            WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
              AND esquema.nspowner <> (
                  SELECT oid FROM pg_catalog.pg_roles
                   WHERE rolname =
                     'vec_bolsa_importacion_convoca_propietario'
              )
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS objeto
           JOIN pg_catalog.pg_namespace AS esquema
             ON esquema.oid = objeto.relnamespace
          WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS objeto
           JOIN pg_catalog.pg_namespace AS esquema
             ON esquema.oid = objeto.pronamespace
          WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar importacion Convoca';
    END IF;
END
$prevalidacion$;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_importacion_convoca_propietario
    IN SCHEMA vec_bolsa_importacion_convoca REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_importacion_convoca_propietario
    IN SCHEMA vec_bolsa_importacion_convoca REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

CREATE FUNCTION vec_bolsa_importacion_convoca.texto_opaco_valido(
    p_valor text, p_maximo integer
)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND pg_catalog.octet_length(p_valor) BETWEEN 3 AND p_maximo
       AND p_valor = pg_catalog.btrim(p_valor)
       AND p_valor ~ '^[a-z][a-z0-9_.:/-]+$'
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.huella_valida(p_valor text)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
    SELECT p_valor IS NOT NULL AND p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.codigo_gobernado_valido(
    p_valor text
)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
    SELECT p_valor IS NOT NULL
       AND pg_catalog.octet_length(p_valor) BETWEEN 3 AND 128
       AND p_valor ~ '^[a-z][a-z0-9_.:-]+$'
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.instante_microsegundo_valido(
    p_valor text
)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
DECLARE convertido timestamptz;
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    convertido := p_valor::timestamptz;
    RETURN pg_catalog.to_char(convertido AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') = p_valor;
EXCEPTION WHEN OTHERS THEN RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.acta_valida(p_acta jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
DECLARE
    leidas integer;
    aceptadas integer;
    rechazadas integer;
    incidencia jsonb;
    filas_incidencia integer;
BEGIN
    IF pg_catalog.jsonb_typeof(p_acta) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.array_agg(clave ORDER BY clave)
             FROM pg_catalog.jsonb_object_keys(p_acta) AS clave)
          IS DISTINCT FROM
          ARRAY[
              'acta_ref','actor_ref','esquema','fichero_custodiado_ref',
              'filas_aceptadas','filas_leidas','filas_rechazadas',
              'huella_fichero_sha256','importacion_ref','incidencias',
              'nombre_fichero','procedencia','registrada_en'
          ]::text[]
       OR pg_catalog.octet_length(p_acta::text) > 25165824
       OR pg_catalog.jsonb_typeof(p_acta->'acta_ref')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'actor_ref')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'esquema')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'fichero_custodiado_ref')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'huella_fichero_sha256')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'importacion_ref')
          IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_acta->'registrada_en')
          IS DISTINCT FROM 'string'
       OR vec_bolsa_importacion_convoca.huella_valida(
           p_acta->>'huella_fichero_sha256'
       ) IS NOT TRUE
       OR p_acta->>'acta_ref' IS DISTINCT FROM
          ('acta:importacion-convoca:'::text ||
           (p_acta->>'huella_fichero_sha256'))
       OR p_acta->>'importacion_ref' IS DISTINCT FROM
          ('importacion:convoca:'::text ||
           (p_acta->>'huella_fichero_sha256'))
       OR vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_acta->>'fichero_custodiado_ref', 512
       ) IS NOT TRUE
       OR p_acta->>'esquema' NOT IN (
           'convoca_resumen_persona_v1','convoca_detalle_merito_v1'
       )
       OR p_acta->>'esquema' IS NULL
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_acta->>'actor_ref'
       ) IS NOT TRUE
       OR pg_catalog.jsonb_typeof(p_acta->'nombre_fichero')
          IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_acta->>'nombre_fichero') NOT BETWEEN 5 AND 255
       OR p_acta->>'nombre_fichero' IS DISTINCT FROM
          pg_catalog.btrim(p_acta->>'nombre_fichero')
       OR pg_catalog.lower(pg_catalog.right(p_acta->>'nombre_fichero', 4))
          IS DISTINCT FROM '.xls'
       OR pg_catalog.strpos(p_acta->>'nombre_fichero', '/') > 0
       OR pg_catalog.strpos(p_acta->>'nombre_fichero', '\') > 0
       OR p_acta->>'nombre_fichero' ~ '[[:cntrl:]]'
       OR vec_bolsa_importacion_convoca.instante_microsegundo_valido(
           p_acta->>'registrada_en'
       ) IS NOT TRUE
       OR pg_catalog.jsonb_typeof(p_acta->'filas_leidas')
          IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(p_acta->'filas_aceptadas')
          IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(p_acta->'filas_rechazadas')
          IS DISTINCT FROM 'number'
       OR p_acta->>'filas_leidas' !~ '^(0|[1-9][0-9]{0,5})$'
       OR p_acta->>'filas_aceptadas' !~ '^(0|[1-9][0-9]{0,5})$'
       OR p_acta->>'filas_rechazadas' !~ '^(0|[1-9][0-9]{0,5})$'
       OR pg_catalog.jsonb_typeof(p_acta->'incidencias')
          IS DISTINCT FROM 'array'
       OR p_acta->'procedencia' IS DISTINCT FROM pg_catalog.jsonb_build_object(
           'esquema', 'vec.bolsa.importacion-convoca.procedencia.v1',
           'fuente', 'Convoca (exportacion enmascarada)',
           'autoridad', 'no_autoritativa',
           'habilita_actos_con_efectos', false,
           'requiere_confirmacion_registro', true,
           'uso_puntos_autobaremacion', 'historico_contraste'
       ) THEN
        RETURN false;
    END IF;
    leidas := (p_acta->>'filas_leidas')::integer;
    aceptadas := (p_acta->>'filas_aceptadas')::integer;
    rechazadas := (p_acta->>'filas_rechazadas')::integer;
    IF leidas < 0 OR aceptadas < 0 OR rechazadas < 0
       OR leidas > 100001 OR aceptadas + rechazadas <> leidas THEN
        RETURN false;
    END IF;
    FOR incidencia IN
        SELECT value FROM pg_catalog.jsonb_array_elements(p_acta->'incidencias')
    LOOP
        IF pg_catalog.jsonb_typeof(incidencia) IS DISTINCT FROM 'object'
           OR (SELECT pg_catalog.array_agg(clave ORDER BY clave)
                 FROM pg_catalog.jsonb_object_keys(incidencia) AS clave)
              IS DISTINCT FROM
              ARRAY['campo','codigo','fila']::text[]
           OR pg_catalog.jsonb_typeof(incidencia->'fila')
              IS DISTINCT FROM 'number'
           OR incidencia->>'fila' !~ '^[0-9]{1,6}$'
           OR (incidencia->>'fila')::integer < 2
           OR pg_catalog.jsonb_typeof(incidencia->'campo')
              IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(incidencia->>'campo') NOT BETWEEN 1 AND 120
           OR incidencia->>'campo' IS DISTINCT FROM
              pg_catalog.btrim(incidencia->>'campo')
           OR incidencia->>'campo' ~ '[[:cntrl:]]'
           OR pg_catalog.jsonb_typeof(incidencia->'codigo')
              IS DISTINCT FROM 'string'
           OR incidencia->>'codigo' !~ '^[a-z][a-z0-9_]{0,127}$' THEN
            RETURN false;
        END IF;
    END LOOP;
    SELECT pg_catalog.count(DISTINCT (value->>'fila')::integer)
      INTO filas_incidencia
      FROM pg_catalog.jsonb_array_elements(p_acta->'incidencias');
    RETURN (filas_incidencia = rechazadas) IS TRUE;
EXCEPTION WHEN OTHERS THEN RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.filas_protegidas_validas(
    p_filas jsonb, p_esperadas integer
)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
DECLARE
    fila jsonb;
    nonce bytea;
    cifrado bytea;
    derivacion bytea;
    atestacion bytea;
    numero_anterior integer := 1;
    total_bytes bigint := 0;
BEGIN
    IF pg_catalog.jsonb_typeof(p_filas) IS DISTINCT FROM 'array'
       OR p_esperadas IS NULL OR p_esperadas < 0
       OR p_esperadas > 100001
       OR pg_catalog.jsonb_array_length(p_filas) <> p_esperadas
       OR pg_catalog.octet_length(p_filas::text) > 75497472 THEN
        RETURN false;
    END IF;
    FOR fila IN SELECT value FROM pg_catalog.jsonb_array_elements(p_filas)
    LOOP
        IF pg_catalog.jsonb_typeof(fila) IS DISTINCT FROM 'object'
           OR (SELECT pg_catalog.array_agg(clave ORDER BY clave)
                 FROM pg_catalog.jsonb_object_keys(fila) AS clave)
              IS DISTINCT FROM
              ARRAY[
                  'atestacion_fila_hmac_sha256','clave_atestacion_ref',
                  'clave_derivacion_ref','clave_ref','contenido_cifrado_hex',
                  'derivacion_documento_hmac_sha256',
                  'esquema_proteccion',
                  'huella_contenido_cifrado_sha256','nonce_hex','numero'
              ]::text[]
           OR pg_catalog.jsonb_typeof(fila->'numero')
              IS DISTINCT FROM 'number'
           OR pg_catalog.jsonb_typeof(fila->'esquema_proteccion')
              IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(fila->'clave_ref')
              IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(fila->'clave_derivacion_ref')
              IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(fila->'clave_atestacion_ref')
              IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(
               fila->'huella_contenido_cifrado_sha256'
           ) IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(
               fila->'derivacion_documento_hmac_sha256'
           ) IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(
               fila->'atestacion_fila_hmac_sha256'
           ) IS DISTINCT FROM 'string'
           OR fila->>'numero' !~ '^[0-9]{1,6}$'
           OR (fila->>'numero')::integer <= numero_anterior
           OR fila->>'esquema_proteccion' IS DISTINCT FROM
              'vec.bolsa.importacion-convoca.proteccion-staging.v1'
           OR vec_bolsa_importacion_convoca.texto_opaco_valido(
               fila->>'clave_ref', 256
           ) IS NOT TRUE
           OR vec_bolsa_importacion_convoca.texto_opaco_valido(
               fila->>'clave_derivacion_ref', 256
           ) IS NOT TRUE
           OR vec_bolsa_importacion_convoca.texto_opaco_valido(
               fila->>'clave_atestacion_ref', 256
           ) IS NOT TRUE
           OR fila->>'clave_derivacion_ref' = fila->>'clave_ref'
           OR fila->>'clave_atestacion_ref' = fila->>'clave_ref'
           OR fila->>'clave_atestacion_ref' =
              fila->>'clave_derivacion_ref'
           OR vec_bolsa_importacion_convoca.huella_valida(
               fila->>'huella_contenido_cifrado_sha256'
           ) IS NOT TRUE
           OR pg_catalog.jsonb_typeof(fila->'nonce_hex')
              IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(fila->'contenido_cifrado_hex')
              IS DISTINCT FROM 'string'
           OR fila->>'derivacion_documento_hmac_sha256' !~ '^[0-9a-f]{64}$'
           OR fila->>'derivacion_documento_hmac_sha256' =
              pg_catalog.repeat('0', 64)
           OR fila->>'atestacion_fila_hmac_sha256' !~ '^[0-9a-f]{64}$'
           OR fila->>'atestacion_fila_hmac_sha256' =
              pg_catalog.repeat('0', 64) THEN
            RETURN false;
        END IF;
        nonce := pg_catalog.decode(fila->>'nonce_hex', 'hex');
        cifrado := pg_catalog.decode(fila->>'contenido_cifrado_hex', 'hex');
        derivacion := pg_catalog.decode(
            fila->>'derivacion_documento_hmac_sha256', 'hex'
        );
        atestacion := pg_catalog.decode(
            fila->>'atestacion_fila_hmac_sha256', 'hex'
        );
        IF pg_catalog.octet_length(nonce) NOT BETWEEN 12 AND 64
           OR pg_catalog.octet_length(cifrado) NOT BETWEEN 16 AND 131072
           OR pg_catalog.octet_length(derivacion) <> 32
           OR pg_catalog.octet_length(atestacion) <> 32
           OR pg_catalog.encode(pg_catalog.sha256(cifrado), 'hex') <>
              fila->>'huella_contenido_cifrado_sha256' THEN
            RETURN false;
        END IF;
        total_bytes := total_bytes + pg_catalog.octet_length(nonce)
                     + pg_catalog.octet_length(cifrado)
                     + pg_catalog.octet_length(derivacion)
                     + pg_catalog.octet_length(atestacion);
        IF total_bytes > 25165824 THEN RETURN false; END IF;
        numero_anterior := (fila->>'numero')::integer;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.huella_evento_estado(
    p_evento_ref text, p_importacion_ref text, p_secuencia bigint,
    p_huella_anterior text, p_tipo text, p_actor_ref text,
    p_estado_conciliacion text, p_estado_staging text,
    p_bloqueo_retencion boolean, p_registrada_en timestamptz
)
RETURNS text LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.concat_ws(pg_catalog.chr(31), p_evento_ref,
            p_importacion_ref, p_secuencia::text, p_huella_anterior, p_tipo,
            p_actor_ref, p_estado_conciliacion, p_estado_staging,
            p_bloqueo_retencion::text,
            pg_catalog.to_char(p_registrada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ), 'UTF8'
    )), 'hex')
$funcion$;

CREATE TABLE vec_bolsa_importacion_convoca.lote (
    importacion_ref text PRIMARY KEY,
    acta_ref text NOT NULL UNIQUE,
    huella_fichero_sha256 text NOT NULL UNIQUE,
    acta_canonica jsonb NOT NULL,
    huella_acta_sha256 text NOT NULL,
    huella_staging_sha256 text NOT NULL,
    huella_staging_semantica_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    estado_conciliacion text NOT NULL DEFAULT 'pendiente',
    estado_staging text NOT NULL DEFAULT 'disponible',
    politica_retencion_ref text NOT NULL,
    politica_retencion_version bigint NOT NULL,
    conservar_staging_hasta timestamptz(6) NOT NULL,
    bloqueo_retencion boolean NOT NULL DEFAULT false,
    version bigint NOT NULL DEFAULT 1,
    secuencia_historia bigint NOT NULL,
    cabeza_historia_sha256 text NOT NULL,
    CHECK (vec_bolsa_importacion_convoca.acta_valida(acta_canonica) IS TRUE),
    CHECK (acta_canonica->>'importacion_ref' IS NOT DISTINCT FROM importacion_ref),
    CHECK (acta_canonica->>'acta_ref' IS NOT DISTINCT FROM acta_ref),
    CHECK (acta_canonica->>'huella_fichero_sha256'
           IS NOT DISTINCT FROM huella_fichero_sha256),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_acta_sha256)),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_staging_sha256)),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(
        huella_staging_semantica_sha256
    )),
    CHECK (estado_conciliacion IN ('pendiente','confirmada','descartada')),
    CHECK (estado_staging IN ('disponible','expurgado')),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        politica_retencion_ref, 512
    )),
    CHECK (politica_retencion_version > 0 AND version > 0),
    CHECK (conservar_staging_hasta > registrada_en),
    CHECK (secuencia_historia > 0),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(cabeza_historia_sha256))
);

CREATE TABLE vec_bolsa_importacion_convoca.fila_staging (
    importacion_ref text NOT NULL
        REFERENCES vec_bolsa_importacion_convoca.lote(importacion_ref),
    numero integer NOT NULL,
    esquema_proteccion text NOT NULL,
    clave_ref text NOT NULL,
    nonce bytea NOT NULL,
    contenido_cifrado bytea NOT NULL,
    huella_contenido_cifrado_sha256 text NOT NULL,
    derivacion_documento_hmac_sha256 bytea NOT NULL,
    clave_derivacion_ref text NOT NULL,
    clave_atestacion_ref text NOT NULL,
    atestacion_fila_hmac_sha256 bytea NOT NULL,
    PRIMARY KEY (importacion_ref, numero),
    CHECK (numero >= 2),
    CHECK (esquema_proteccion =
        'vec.bolsa.importacion-convoca.proteccion-staging.v1'),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(clave_ref, 256)),
    CHECK (pg_catalog.octet_length(nonce) BETWEEN 12 AND 64),
    CHECK (pg_catalog.octet_length(contenido_cifrado) BETWEEN 16 AND 131072),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(
        huella_contenido_cifrado_sha256
    )),
    CHECK (pg_catalog.encode(pg_catalog.sha256(contenido_cifrado), 'hex') =
        huella_contenido_cifrado_sha256),
    CHECK (pg_catalog.octet_length(derivacion_documento_hmac_sha256) = 32),
    CHECK (derivacion_documento_hmac_sha256 <>
        pg_catalog.decode(pg_catalog.repeat('00', 32), 'hex')),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        clave_derivacion_ref, 256
    )),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        clave_atestacion_ref, 256
    )),
    CHECK (clave_atestacion_ref <> clave_ref),
    CHECK (clave_derivacion_ref <> clave_ref),
    CHECK (clave_atestacion_ref <> clave_derivacion_ref),
    CHECK (pg_catalog.octet_length(atestacion_fila_hmac_sha256) = 32),
    CHECK (atestacion_fila_hmac_sha256 <>
        pg_catalog.decode(pg_catalog.repeat('00', 32), 'hex'))
);

CREATE TABLE vec_bolsa_importacion_convoca.conciliacion (
    conciliacion_ref text PRIMARY KEY,
    importacion_ref text NOT NULL UNIQUE
        REFERENCES vec_bolsa_importacion_convoca.lote(importacion_ref),
    registro_corporativo_ref text NOT NULL,
    resultado text NOT NULL,
    actor_ref text NOT NULL,
    motivo_codigo text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_conciliacion_sha256 text NOT NULL,
    CHECK (resultado IN ('confirmada','descartada')),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        conciliacion_ref, 512
    )),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        registro_corporativo_ref, 512
    )),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref)),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(motivo_codigo)),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(
        huella_conciliacion_sha256
    ))
);

CREATE TABLE vec_bolsa_importacion_convoca.decision_retencion (
    decision_ref text PRIMARY KEY,
    importacion_ref text NOT NULL
        REFERENCES vec_bolsa_importacion_convoca.lote(importacion_ref),
    actor_ref text NOT NULL,
    motivo_codigo text NOT NULL,
    bloqueado boolean NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_decision_sha256 text NOT NULL,
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(decision_ref, 512)),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref)),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(motivo_codigo)),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_decision_sha256))
);

CREATE TABLE vec_bolsa_importacion_convoca.ejecucion_retencion (
    ejecucion_ref text PRIMARY KEY,
    actor_ref text NOT NULL,
    politica_retencion_ref text NOT NULL,
    politica_retencion_version bigint NOT NULL,
    limite integer NOT NULL,
    lotes integer NOT NULL,
    filas integer NOT NULL,
    ejecutada_en timestamptz(6) NOT NULL,
    huella_ejecucion_sha256 text NOT NULL,
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(ejecucion_ref, 512)),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref)),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        politica_retencion_ref, 512
    )),
    CHECK (politica_retencion_version > 0),
    CHECK (limite BETWEEN 1 AND 1000 AND lotes BETWEEN 0 AND limite AND filas >= 0),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_ejecucion_sha256))
);

CREATE TABLE vec_bolsa_importacion_convoca.historia_estado (
    evento_ref text PRIMARY KEY,
    importacion_ref text NOT NULL
        REFERENCES vec_bolsa_importacion_convoca.lote(importacion_ref),
    secuencia bigint NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    tipo text NOT NULL,
    actor_ref text NOT NULL,
    estado_conciliacion text NOT NULL,
    estado_staging text NOT NULL,
    bloqueo_retencion boolean NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_evento_sha256 text NOT NULL,
    UNIQUE (importacion_ref, secuencia),
    CHECK (pg_catalog.octet_length(evento_ref) BETWEEN 3 AND 1600),
    CHECK (secuencia > 0),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_anterior_sha256)),
    CHECK (tipo IN ('importacion','conciliacion','bloqueo_retencion','expurgo')),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref)),
    CHECK (estado_conciliacion IN ('pendiente','confirmada','descartada')),
    CHECK (estado_staging IN ('disponible','expurgado')),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_evento_sha256))
);

CREATE TABLE vec_bolsa_importacion_convoca.outbox (
    evento_ref text PRIMARY KEY,
    importacion_ref text NOT NULL
        REFERENCES vec_bolsa_importacion_convoca.lote(importacion_ref),
    tipo text NOT NULL,
    payload jsonb NOT NULL,
    ocurrida_en timestamptz(6) NOT NULL,
    huella_evento_sha256 text NOT NULL,
    CHECK (tipo = 'importacion_convoca_confirmada_v1'),
    CHECK (pg_catalog.jsonb_typeof(payload) = 'object'),
    CHECK (pg_catalog.octet_length(payload::text) <= 16384),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(huella_evento_sha256))
);

CREATE TABLE vec_bolsa_importacion_convoca.politica_retencion (
    politica_retencion_ref text NOT NULL,
    politica_retencion_version bigint NOT NULL,
    duracion_segundos bigint NOT NULL,
    secuencia_publicacion bigint NOT NULL UNIQUE,
    huella_anterior_sha256 text NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    actor_ref text NOT NULL,
    huella_publicacion_sha256 text NOT NULL,
    PRIMARY KEY (politica_retencion_ref, politica_retencion_version),
    CHECK (vec_bolsa_importacion_convoca.texto_opaco_valido(
        politica_retencion_ref, 512
    )),
    CHECK (politica_retencion_version > 0),
    CHECK (duracion_segundos BETWEEN 3600 AND 3153600000),
    CHECK (secuencia_publicacion > 0),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(
        huella_anterior_sha256
    )),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref)),
    CHECK (vec_bolsa_importacion_convoca.huella_valida(
        huella_publicacion_sha256
    ))
);

CREATE TABLE vec_bolsa_importacion_convoca.politica_retencion_actual (
    ambito text PRIMARY KEY CHECK (ambito = 'staging_convoca'),
    politica_retencion_ref text NOT NULL,
    politica_retencion_version bigint NOT NULL,
    secuencia_publicacion bigint NOT NULL UNIQUE,
    actualizada_en timestamptz(6) NOT NULL,
    actor_ref text NOT NULL,
    FOREIGN KEY (politica_retencion_ref, politica_retencion_version)
        REFERENCES vec_bolsa_importacion_convoca.politica_retencion(
            politica_retencion_ref, politica_retencion_version
        ),
    CHECK (secuencia_publicacion > 0),
    CHECK (vec_bolsa_importacion_convoca.codigo_gobernado_valido(actor_ref))
);

CREATE FUNCTION vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable()
RETURNS trigger LANGUAGE plpgsql
SET search_path = pg_catalog AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia de importacion Convoca inmutable';
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.validar_mutacion_lote()
RETURNS trigger LANGUAGE plpgsql
SET search_path = pg_catalog AS $funcion$
DECLARE cambios integer;
BEGIN
    IF TG_OP = 'DELETE' OR NEW.importacion_ref <> OLD.importacion_ref
       OR NEW.acta_ref <> OLD.acta_ref
       OR NEW.huella_fichero_sha256 <> OLD.huella_fichero_sha256
       OR NEW.acta_canonica <> OLD.acta_canonica
       OR NEW.huella_acta_sha256 <> OLD.huella_acta_sha256
       OR NEW.huella_staging_sha256 <> OLD.huella_staging_sha256
       OR NEW.huella_staging_semantica_sha256 <>
          OLD.huella_staging_semantica_sha256
       OR NEW.registrada_en <> OLD.registrada_en
       OR NEW.politica_retencion_ref <> OLD.politica_retencion_ref
       OR NEW.politica_retencion_version <> OLD.politica_retencion_version
       OR NEW.conservar_staging_hasta <> OLD.conservar_staging_hasta
       OR NEW.version <> OLD.version + 1
       OR NEW.secuencia_historia <> OLD.secuencia_historia + 1
       OR vec_bolsa_importacion_convoca.huella_valida(
           NEW.cabeza_historia_sha256
       ) IS NOT TRUE
       OR NEW.cabeza_historia_sha256 = OLD.cabeza_historia_sha256 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'mutacion de lote Convoca no permitida';
    END IF;
    cambios := (NEW.estado_conciliacion <> OLD.estado_conciliacion)::integer
             + (NEW.estado_staging <> OLD.estado_staging)::integer
             + (NEW.bloqueo_retencion <> OLD.bloqueo_retencion)::integer;
    IF cambios <> 1
       OR (NEW.estado_conciliacion <> OLD.estado_conciliacion AND
           (OLD.estado_conciliacion <> 'pendiente' OR
            NEW.estado_conciliacion NOT IN ('confirmada','descartada')))
       OR (NEW.estado_staging <> OLD.estado_staging AND
           (OLD.estado_staging <> 'disponible' OR
            NEW.estado_staging <> 'expurgado' OR NEW.bloqueo_retencion))
       OR (NEW.bloqueo_retencion <> OLD.bloqueo_retencion AND
           OLD.estado_staging <> 'disponible') THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'transicion de lote Convoca no permitida';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER lote_transicion_cerrada
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.lote
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.validar_mutacion_lote();
CREATE TRIGGER fila_staging_sin_actualizacion
BEFORE UPDATE ON vec_bolsa_importacion_convoca.fila_staging
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER conciliacion_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.conciliacion
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER decision_retencion_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.decision_retencion
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER ejecucion_retencion_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.ejecucion_retencion
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER historia_estado_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.historia_estado
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER outbox_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.outbox
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER politica_retencion_inmutable
BEFORE UPDATE OR DELETE ON vec_bolsa_importacion_convoca.politica_retencion
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();

CREATE FUNCTION vec_bolsa_importacion_convoca.validar_puntero_politica()
RETURNS trigger LANGUAGE plpgsql
SET search_path = pg_catalog AS $funcion$
BEGIN
    IF TG_OP = 'DELETE' OR NEW.ambito <> OLD.ambito
       OR NEW.secuencia_publicacion <= OLD.secuencia_publicacion
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'transicion de politica Convoca no permitida';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER puntero_politica_cerrado
BEFORE UPDATE OR DELETE
ON vec_bolsa_importacion_convoca.politica_retencion_actual
FOR EACH ROW EXECUTE FUNCTION
    vec_bolsa_importacion_convoca.validar_puntero_politica();

CREATE TRIGGER lote_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.lote FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER staging_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.fila_staging FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER conciliacion_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.conciliacion FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER decision_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.decision_retencion FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER ejecucion_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.ejecucion_retencion FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER historia_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.historia_estado FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER outbox_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.outbox FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER politica_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.politica_retencion
FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();
CREATE TRIGGER puntero_politica_sin_truncate BEFORE TRUNCATE
ON vec_bolsa_importacion_convoca.politica_retencion_actual
FOR EACH STATEMENT EXECUTE FUNCTION
vec_bolsa_importacion_convoca.rechazar_mutacion_inmutable();

CREATE FUNCTION vec_bolsa_importacion_convoca.historia_integra(
    p_importacion_ref text
)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    evento vec_bolsa_importacion_convoca.historia_estado%ROWTYPE;
    anterior text := pg_catalog.repeat('0', 64);
    secuencia bigint := 0;
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
BEGIN
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref;
    IF NOT FOUND THEN RETURN false; END IF;
    FOR evento IN
        SELECT * FROM vec_bolsa_importacion_convoca.historia_estado
         WHERE importacion_ref = p_importacion_ref ORDER BY secuencia
    LOOP
        secuencia := secuencia + 1;
        IF evento.secuencia <> secuencia
           OR evento.huella_anterior_sha256 <> anterior
           OR evento.huella_evento_sha256 <>
              vec_bolsa_importacion_convoca.huella_evento_estado(
                  evento.evento_ref, evento.importacion_ref, evento.secuencia,
                  evento.huella_anterior_sha256, evento.tipo, evento.actor_ref,
                  evento.estado_conciliacion, evento.estado_staging,
                  evento.bloqueo_retencion, evento.registrada_en
              ) THEN
            RETURN false;
        END IF;
        anterior := evento.huella_evento_sha256;
    END LOOP;
    RETURN secuencia = actual.secuencia_historia
       AND anterior = actual.cabeza_historia_sha256;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.lote_integro(
    p_importacion_ref text
)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
    filas integer;
    huella text;
    huella_semantica text;
BEGIN
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref;
    IF NOT FOUND
       OR vec_bolsa_importacion_convoca.acta_valida(actual.acta_canonica)
          IS NOT TRUE
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           actual.acta_canonica::text, 'UTF8'
       )), 'hex') <> actual.huella_acta_sha256
       OR vec_bolsa_importacion_convoca.historia_integra(
           p_importacion_ref
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    SELECT pg_catalog.count(*),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   pg_catalog.concat_ws(pg_catalog.chr(30), numero::text,
                       esquema_proteccion, clave_ref, clave_derivacion_ref,
                       clave_atestacion_ref,
                       pg_catalog.encode(nonce, 'hex'),
                       pg_catalog.encode(contenido_cifrado, 'hex'),
                       huella_contenido_cifrado_sha256,
                       pg_catalog.encode(
                           derivacion_documento_hmac_sha256, 'hex'
                       ),
                       pg_catalog.encode(
                           atestacion_fila_hmac_sha256, 'hex'
                       )
                   ),
                   ',' ORDER BY numero
               ), ''), 'UTF8'
           )), 'hex'),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   numero::text || ':' || clave_atestacion_ref || ':' ||
                   pg_catalog.encode(atestacion_fila_hmac_sha256, 'hex'),
                   ',' ORDER BY numero
               ), ''), 'UTF8'
           )), 'hex')
      INTO filas, huella, huella_semantica
      FROM vec_bolsa_importacion_convoca.fila_staging
     WHERE importacion_ref = p_importacion_ref;
    IF actual.estado_staging = 'disponible' THEN
        RETURN (
               filas = (actual.acta_canonica->>'filas_aceptadas')::integer
           AND huella = actual.huella_staging_sha256
           AND huella_semantica =
               actual.huella_staging_semantica_sha256
        ) IS TRUE;
    END IF;
    RETURN (actual.estado_staging = 'expurgado' AND filas = 0) IS TRUE;
END
$funcion$;

ALTER TABLE vec_bolsa_importacion_convoca.lote ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.lote FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    FORCE ROW LEVEL SECURITY;

CREATE POLICY propietario_lote ON vec_bolsa_importacion_convoca.lote
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_staging ON vec_bolsa_importacion_convoca.fila_staging
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_conciliacion ON vec_bolsa_importacion_convoca.conciliacion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_decision ON vec_bolsa_importacion_convoca.decision_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_ejecucion ON vec_bolsa_importacion_convoca.ejecucion_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_historia ON vec_bolsa_importacion_convoca.historia_estado
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_outbox ON vec_bolsa_importacion_convoca.outbox
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_politica
ON vec_bolsa_importacion_convoca.politica_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_puntero_politica
ON vec_bolsa_importacion_convoca.politica_retencion_actual
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
COMMIT;
