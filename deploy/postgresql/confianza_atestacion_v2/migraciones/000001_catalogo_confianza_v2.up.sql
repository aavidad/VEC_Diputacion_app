-- Catalogo durable de material PUBLICO de confianza VEC-AD-2.
-- No contiene claves privadas, HMAC, capacidades ni credenciales runtime.
BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:migracion:v1', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace(
           'vec_confianza_atestacion_v2'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el catalogo de confianza V2 ya existe';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_confianza_atestacion_v2
    AUTHORIZATION vec_confianza_atestacion_v2_propietario;
REVOKE ALL ON SCHEMA vec_confianza_atestacion_v2 FROM PUBLIC;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    IN SCHEMA vec_confianza_atestacion_v2
    REVOKE ALL ON TABLES FROM PUBLIC,
        vec_confianza_atestacion_v2_lector_autoridad;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    IN SCHEMA vec_confianza_atestacion_v2
    REVOKE ALL ON SEQUENCES FROM PUBLIC,
        vec_confianza_atestacion_v2_lector_autoridad;
-- PostgreSQL concede EXECUTE/USAGE a PUBLIC globalmente por defecto. El rol
-- propietario es exclusivo de este modulo, por lo que se cierran globalmente.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC,
        vec_confianza_atestacion_v2_lector_autoridad;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    REVOKE ALL ON TYPES FROM PUBLIC,
        vec_confianza_atestacion_v2_lector_autoridad;

CREATE FUNCTION vec_confianza_atestacion_v2.texto_tecnico_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor ~ '^[!-~]+$'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.huella_sha256_valida(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.audiencia_despliegue_valida(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    partes text[];
    parte text;
BEGIN
    IF vec_confianza_atestacion_v2.texto_tecnico_valido(
           p_valor, 512
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    partes := string_to_array(p_valor, '/');
    IF cardinality(partes) <> 4 OR partes[1] <> 'vec-diputacion' THEN
        RETURN false;
    END IF;
    FOREACH parte IN ARRAY partes[2:4] LOOP
        IF parte IN ('', '.', '..')
           OR vec_confianza_atestacion_v2.texto_tecnico_valido(
                  parte, 256
              ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.clave_spki_ed25519_valida(
    p_clave bytea
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    -- SubjectPublicKeyInfo DER canonico de Ed25519: RFC 8410, sin parametros.
    SELECT octet_length(p_clave) = 44
       AND substring(p_clave FROM 1 FOR 12) =
           decode('302a300506032b6570032100', 'hex')
       AND substring(p_clave FROM 13 FOR 32) <>
           decode(repeat('00', 32), 'hex')
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.instante_go_valido(
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_instante IS NOT NULL
       AND isfinite(p_instante)
       AND extract(
           year FROM p_instante AT TIME ZONE 'UTC'
       ) BETWEEN 1 AND 9999
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.instante_rfc3339nano(
    p_instante timestamptz
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    base text;
    microsegundos text;
BEGIN
    IF vec_confianza_atestacion_v2.instante_go_valido(
           p_instante
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22008',
            MESSAGE = 'instante no finito';
    END IF;
    base := to_char(
        p_instante AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS'
    );
    microsegundos := regexp_replace(
        to_char(p_instante AT TIME ZONE 'UTC', 'US'),
        '0+$',
        ''
    );
    RETURN base || CASE
        WHEN microsegundos = '' THEN ''
        ELSE '.' || microsegundos
    END || 'Z';
END
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.encuadrar_huella(
    p_valor text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.int8send(
               octet_length(convert_to(p_valor, 'UTF8'))::bigint
           ) || convert_to(p_valor, 'UTF8')
$funcion$;

-- Todos los cambios de gobierno y las lecturas autoritativas comparten este
-- candado. Evita instantaneas partidas; no pretende sobrevivir a restaurar una
-- copia antigua completa de la base, que requiere controles operacionales.
CREATE FUNCTION vec_confianza_atestacion_v2.proteger_historia_fila()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'la historia de confianza V2 es inmutable';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.rechazar_truncado()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'TRUNCATE de confianza V2 rechazado';
END
$funcion$;

CREATE TABLE vec_confianza_atestacion_v2.acto_gobierno (
    acto_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    clase text NOT NULL,
    emitido_en timestamptz(6) NOT NULL,
    documento_huella_sha256 text NOT NULL,
    registrado_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (acto_ref, clase),
    CHECK (secuencia BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_confianza_atestacion_v2.texto_tecnico_valido(
        acto_ref, 512
    )),
    CHECK (clase IN (
        'publicacion_configuracion',
        'publicacion_raiz',
        'activacion_configuracion',
        'activacion_raiz',
        'revocacion_configuracion',
        'revocacion_raiz'
    )),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(emitido_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrado_en)),
    CHECK (vec_confianza_atestacion_v2.huella_sha256_valida(
        documento_huella_sha256
    ))
);

CREATE TABLE vec_confianza_atestacion_v2.configuracion_confianza_version (
    revision text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    huella_configuracion_sha256 text NOT NULL UNIQUE,
    publicada_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    numero_raices smallint NOT NULL,
    acto_ref text NOT NULL,
    acto_clase text GENERATED ALWAYS AS (
        'publicacion_configuracion'::text
    ) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (secuencia BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_confianza_atestacion_v2.texto_tecnico_valido(
        revision, 128
    )),
    CHECK (vec_confianza_atestacion_v2.huella_sha256_valida(
        huella_configuracion_sha256
    )),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(publicada_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(expira_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en)),
    CHECK (expira_en > publicada_en),
    CHECK (expira_en <= publicada_en + interval '24 hours'),
    CHECK (numero_raices BETWEEN 1 AND 64)
);

CREATE TABLE vec_confianza_atestacion_v2.raiz_confianza_version (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    algoritmo_cose text NOT NULL DEFAULT 'EdDSA',
    suite text NOT NULL DEFAULT 'VEC-AD-2-COSE-EDDSA-1',
    audiencia_despliegue text NOT NULL,
    clave_publica_spki bytea NOT NULL,
    huella_clave_spki_sha256 text NOT NULL UNIQUE,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    acto_clase text GENERATED ALWAYS AS ('publicacion_raiz'::text) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (version BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_confianza_atestacion_v2.texto_tecnico_valido(
        clave_id, 512
    )),
    CHECK (algoritmo_cose = 'EdDSA'),
    CHECK (suite = 'VEC-AD-2-COSE-EDDSA-1'),
    CHECK (vec_confianza_atestacion_v2.audiencia_despliegue_valida(
        audiencia_despliegue
    )),
    CHECK (vec_confianza_atestacion_v2.clave_spki_ed25519_valida(
        clave_publica_spki
    )),
    CHECK (
        encode(sha256(clave_publica_spki), 'hex') =
            huella_clave_spki_sha256
    ),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(valida_desde)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(valida_hasta)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en)),
    CHECK (valida_hasta > valida_desde)
);

-- Una configuracion referencia versiones ya publicadas. Reutilizar una raiz
-- vigente entre revisiones no duplica material; la unicidad global de huella
-- impide resucitarla con otro clave_id.
CREATE TABLE vec_confianza_atestacion_v2.configuracion_raiz (
    configuracion_revision text NOT NULL,
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    acto_ref text NOT NULL,
    acto_clase text GENERATED ALWAYS AS (
        'publicacion_configuracion'::text
    ) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (configuracion_revision, clave_id),
    UNIQUE (configuracion_revision, clave_id, version),
    FOREIGN KEY (configuracion_revision)
        REFERENCES vec_confianza_atestacion_v2.configuracion_confianza_version(
            revision
        ),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version
        ),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en))
);

CREATE TABLE vec_confianza_atestacion_v2.revocacion_configuracion (
    revision text PRIMARY KEY,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    acto_clase text GENERATED ALWAYS AS (
        'revocacion_configuracion'::text
    ) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (revision)
        REFERENCES vec_confianza_atestacion_v2.configuracion_confianza_version(
            revision
        ),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(revocada_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en)),
    CHECK (vec_confianza_atestacion_v2.texto_tecnico_valido(
        motivo_catalogado_ref, 512
    ))
);

CREATE TABLE vec_confianza_atestacion_v2.revocacion_raiz (
    clave_id text NOT NULL,
    version numeric(20, 0) NOT NULL,
    revocada_en timestamptz(6) NOT NULL,
    motivo_catalogado_ref text NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    acto_clase text GENERATED ALWAYS AS ('revocacion_raiz'::text) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, version),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version
        ),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(revocada_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en)),
    CHECK (vec_confianza_atestacion_v2.texto_tecnico_valido(
        motivo_catalogado_ref, 512
    ))
);

-- Los punteros son historia append-only. La fila de orden mayor es la actual;
-- los triggers impiden que orden, secuencia o version retrocedan.
CREATE TABLE vec_confianza_atestacion_v2.puntero_raiz_actual (
    clave_id text NOT NULL,
    orden numeric(20, 0) NOT NULL,
    version numeric(20, 0) NOT NULL,
    establecida_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    acto_clase text GENERATED ALWAYS AS ('activacion_raiz'::text) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (clave_id, orden),
    UNIQUE (clave_id, version),
    FOREIGN KEY (clave_id, version)
        REFERENCES vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version
        ),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (orden BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(establecida_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en))
);

CREATE TABLE vec_confianza_atestacion_v2.puntero_configuracion_actual (
    orden numeric(20, 0) PRIMARY KEY,
    revision text NOT NULL UNIQUE,
    establecida_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    acto_clase text GENERATED ALWAYS AS (
        'activacion_configuracion'::text
    ) STORED,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (revision)
        REFERENCES vec_confianza_atestacion_v2.configuracion_confianza_version(
            revision
        ),
    FOREIGN KEY (acto_ref, acto_clase)
        REFERENCES vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, clase
        ),
    CHECK (orden BETWEEN 1 AND 18446744073709551615),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(establecida_en)),
    CHECK (vec_confianza_atestacion_v2.instante_go_valido(registrada_en))
);

-- RLS de lista positiva: solo el propietario exacto puede gobernar aun cuando
-- una futura concesion accidental otorgase privilegios de tabla.
DO $activar_rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'acto_gobierno',
        'configuracion_confianza_version',
        'raiz_confianza_version',
        'configuracion_raiz',
        'revocacion_configuracion',
        'revocacion_raiz',
        'puntero_raiz_actual',
        'puntero_configuracion_actual'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_confianza_atestacion_v2.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_confianza_atestacion_v2.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY propietario_exacto ON vec_confianza_atestacion_v2.%I TO vec_confianza_atestacion_v2_propietario USING (current_user = ''vec_confianza_atestacion_v2_propietario'') WITH CHECK (current_user = ''vec_confianza_atestacion_v2_propietario'')',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER a00_proteger_historia BEFORE INSERT OR UPDATE OR DELETE ON vec_confianza_atestacion_v2.%I FOR EACH ROW EXECUTE FUNCTION vec_confianza_atestacion_v2.proteger_historia_fila()',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER a01_rechazar_truncado BEFORE TRUNCATE ON vec_confianza_atestacion_v2.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_confianza_atestacion_v2.rechazar_truncado()',
            tabla
        );
    END LOOP;
END
$activar_rls$;

CREATE FUNCTION vec_confianza_atestacion_v2.validar_acto_monotono()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    ultima numeric(20, 0);
BEGIN
    SELECT max(secuencia) INTO ultima
      FROM vec_confianza_atestacion_v2.acto_gobierno;
    IF ultima IS NOT NULL AND NEW.secuencia <= ultima THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'secuencia de acto no monotona';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_acto_monotono
    BEFORE INSERT ON vec_confianza_atestacion_v2.acto_gobierno
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_acto_monotono();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_configuracion_monotona()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    ultima record;
BEGIN
    SELECT secuencia, publicada_en INTO ultima
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version
     ORDER BY secuencia DESC
     LIMIT 1
     FOR SHARE;
    IF ultima.secuencia IS NOT NULL AND (
        NEW.secuencia <= ultima.secuencia
        OR NEW.publicada_en <= ultima.publicada_en
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'secuencia o publicacion de configuracion no monotona';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_configuracion_monotona
    BEFORE INSERT
    ON vec_confianza_atestacion_v2.configuracion_confianza_version
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_configuracion_monotona();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_raiz_monotona()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    ultima record;
BEGIN
    SELECT version, valida_desde INTO ultima
      FROM vec_confianza_atestacion_v2.raiz_confianza_version
     WHERE clave_id = NEW.clave_id
     ORDER BY version DESC
     LIMIT 1
     FOR SHARE;
    IF ultima.version IS NOT NULL AND (
        NEW.version <= ultima.version
        OR NEW.valida_desde <= ultima.valida_desde
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'version o inicio de vigencia de raiz no monotono';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_raiz_monotona
    BEFORE INSERT ON vec_confianza_atestacion_v2.raiz_confianza_version
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_raiz_monotona();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_miembro_configuracion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    acto_publicacion text;
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_confianza_atestacion_v2.puntero_configuracion_actual
         WHERE revision = NEW.configuracion_revision
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no se puede ampliar una configuracion ya activada';
    END IF;
    SELECT acto_ref INTO STRICT acto_publicacion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version
     WHERE revision = NEW.configuracion_revision
     FOR SHARE;
    IF NEW.acto_ref <> acto_publicacion THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'miembro desligado del acto de publicacion';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_miembro_configuracion
    BEFORE INSERT ON vec_confianza_atestacion_v2.configuracion_raiz
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_miembro_configuracion();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_revocacion_configuracion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    publicada timestamptz(6);
BEGIN
    SELECT publicada_en INTO STRICT publicada
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version
     WHERE revision = NEW.revision
     FOR SHARE;
    IF NEW.revocada_en < publicada THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'revocacion de configuracion anterior a su publicacion';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_revocacion_configuracion
    BEFORE INSERT ON vec_confianza_atestacion_v2.revocacion_configuracion
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_revocacion_configuracion();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_revocacion_raiz()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    desde timestamptz(6);
BEGIN
    SELECT valida_desde INTO STRICT desde
      FROM vec_confianza_atestacion_v2.raiz_confianza_version
     WHERE clave_id = NEW.clave_id AND version = NEW.version
     FOR SHARE;
    IF NEW.revocada_en < desde THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'revocacion de raiz anterior a su vigencia';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_revocacion_raiz
    BEFORE INSERT ON vec_confianza_atestacion_v2.revocacion_raiz
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_revocacion_raiz();

CREATE FUNCTION vec_confianza_atestacion_v2.calcular_huella_configuracion(
    p_revision text
)
RETURNS text
LANGUAGE plpgsql
STABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    configuracion record;
    raiz record;
    preimagen bytea := ''::bytea;
    revocada_texto text;
    estado_texto text;
BEGIN
    SELECT revision, publicada_en, expira_en
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version
     WHERE revision = p_revision;

    preimagen :=
        vec_confianza_atestacion_v2.encuadrar_huella(
            'vec.configuracion-confianza-atestacion-autorizacion.v2'
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            configuracion.revision
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(
                configuracion.publicada_en
            )
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(
                configuracion.expira_en
            )
        );

    FOR raiz IN
        SELECT version.clave_id, version.huella_clave_spki_sha256,
               version.audiencia_despliegue, version.valida_desde,
               version.valida_hasta, revocacion.revocada_en
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS miembro
          JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
            ON version.clave_id = miembro.clave_id
           AND version.version = miembro.version
          LEFT JOIN vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
            ON revocacion.clave_id = version.clave_id
           AND revocacion.version = version.version
           AND revocacion.revocada_en <= configuracion.publicada_en
         WHERE miembro.configuracion_revision = p_revision
         ORDER BY version.clave_id COLLATE "C"
    LOOP
        IF raiz.revocada_en IS NULL THEN
            estado_texto := 'activa';
            revocada_texto := '';
        ELSE
            estado_texto := 'revocada';
            revocada_texto :=
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.revocada_en
                );
        END IF;
        preimagen := preimagen ||
            vec_confianza_atestacion_v2.encuadrar_huella(raiz.clave_id) ||
            vec_confianza_atestacion_v2.encuadrar_huella('EdDSA') ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.huella_clave_spki_sha256
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                'VEC-AD-2-COSE-EDDSA-1'
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.audiencia_despliegue
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(estado_texto) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_desde
                )
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_hasta
                )
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(revocada_texto);
    END LOOP;
    RETURN encode(sha256(preimagen), 'hex');
END
$funcion$;

CREATE FUNCTION vec_confianza_atestacion_v2.validar_puntero_raiz()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    anterior record;
    raiz record;
BEGIN
    SELECT orden, version, establecida_en INTO anterior
      FROM vec_confianza_atestacion_v2.puntero_raiz_actual
     WHERE clave_id = NEW.clave_id
     ORDER BY orden DESC
     LIMIT 1
     FOR SHARE;
    IF FOUND AND (
        NEW.orden <= anterior.orden
        OR NEW.version <= anterior.version
        OR NEW.establecida_en <= anterior.establecida_en
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retroceso de puntero de raiz rechazado';
    END IF;

    SELECT valida_desde, valida_hasta INTO STRICT raiz
      FROM vec_confianza_atestacion_v2.raiz_confianza_version
     WHERE clave_id = NEW.clave_id AND version = NEW.version
     FOR SHARE;
    IF NEW.establecida_en > clock_timestamp()
       OR NEW.establecida_en < raiz.valida_desde
       OR NEW.establecida_en >= raiz.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_confianza_atestacion_v2.revocacion_raiz
            WHERE clave_id = NEW.clave_id
              AND version = NEW.version
              AND revocada_en <= NEW.establecida_en
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'puntero de raiz fuera de vigencia o revocado';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_puntero_raiz
    BEFORE INSERT ON vec_confianza_atestacion_v2.puntero_raiz_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_puntero_raiz();

CREATE FUNCTION vec_confianza_atestacion_v2.validar_puntero_configuracion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    anterior record;
    configuracion record;
    miembros bigint;
    activas_vigentes bigint;
BEGIN
    SELECT puntero.orden, version.secuencia, puntero.establecida_en
      INTO anterior
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
      JOIN vec_confianza_atestacion_v2.configuracion_confianza_version AS version
        ON version.revision = puntero.revision
     ORDER BY puntero.orden DESC
     LIMIT 1
     FOR SHARE OF puntero, version;

    SELECT secuencia, huella_configuracion_sha256, publicada_en,
           expira_en, numero_raices
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version
     WHERE revision = NEW.revision
     FOR SHARE;

    IF anterior.orden IS NOT NULL AND (
        NEW.orden <= anterior.orden
        OR configuracion.secuencia <= anterior.secuencia
        OR NEW.establecida_en <= anterior.establecida_en
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retroceso de configuracion actual rechazado';
    END IF;
    IF NEW.establecida_en > clock_timestamp()
       OR NEW.establecida_en < configuracion.publicada_en
       OR NEW.establecida_en >= configuracion.expira_en
       OR EXISTS (
           SELECT 1
             FROM vec_confianza_atestacion_v2.revocacion_configuracion
            WHERE revision = NEW.revision
              AND revocada_en <= NEW.establecida_en
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'configuracion fuera de vigencia o revocada';
    END IF;

    SELECT count(*) INTO miembros
      FROM vec_confianza_atestacion_v2.configuracion_raiz
     WHERE configuracion_revision = NEW.revision;
    IF miembros <> configuracion.numero_raices
       OR miembros NOT BETWEEN 1 AND 64
       OR EXISTS (
           SELECT 1
             FROM vec_confianza_atestacion_v2.configuracion_raiz AS miembro
            WHERE miembro.configuracion_revision = NEW.revision
              AND NOT EXISTS (
                  SELECT 1
                    FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS puntero
                   WHERE puntero.clave_id = miembro.clave_id
                     AND puntero.version = miembro.version
                     AND puntero.orden = (
                         SELECT max(actual.orden)
                           FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS actual
                          WHERE actual.clave_id = miembro.clave_id
                     )
              )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'configuracion incompleta o desligada de raices actuales';
    END IF;
    SELECT count(*) INTO activas_vigentes
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS miembro
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS raiz
        ON raiz.clave_id = miembro.clave_id
       AND raiz.version = miembro.version
      LEFT JOIN vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
        ON revocacion.clave_id = raiz.clave_id
       AND revocacion.version = raiz.version
       AND revocacion.revocada_en <= NEW.establecida_en
     WHERE miembro.configuracion_revision = NEW.revision
       AND revocacion.clave_id IS NULL
       AND NEW.establecida_en >= raiz.valida_desde
       AND NEW.establecida_en < raiz.valida_hasta;
    IF activas_vigentes < 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'configuracion sin raiz activa y vigente';
    END IF;
    IF vec_confianza_atestacion_v2.calcular_huella_configuracion(
           NEW.revision
       ) IS DISTINCT FROM configuracion.huella_configuracion_sha256 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'huella de configuracion VEC-AD-2 incoherente';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER b10_validar_puntero_configuracion
    BEFORE INSERT
    ON vec_confianza_atestacion_v2.puntero_configuracion_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2.validar_puntero_configuracion();

-- Unica superficie runtime. Devuelve exactamente las 16 columnas consumidas
-- por el cargador Go. Las raices revocadas NO se filtran: forman parte de la
-- configuracion exacta; una revocacion coherente, efectiva no despues de la
-- publicacion, puede aparecer ya incorporada en su huella. Una revocacion
-- posterior obliga a publicar otra revision antes de reconstruir confianza.
CREATE FUNCTION vec_confianza_atestacion_v2.obtener_confianza_actual()
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
    audiencia_despliegue text,
    clave_publica_spki bytea,
    huella_clave_spki_sha256 text,
    raiz_valida_desde timestamptz(6),
    raiz_valida_hasta timestamptz(6),
    raiz_estado text,
    raiz_revocada_en timestamptz(6)
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    oid_sesion oid;
    oid_lector oid;
    atributos record;
    puntero_config record;
    config record;
    miembro record;
    puntero_version numeric(20, 0);
    numero_miembros bigint;
    raices_activas bigint;
    instante timestamptz(6);
BEGIN
    SELECT oid, rolcanlogin, rolsuper, rolcreatedb, rolcreaterole,
           rolreplication, rolbypassrls
      INTO atributos
      FROM pg_catalog.pg_roles
     WHERE rolname = session_user;
    SELECT oid INTO oid_sesion
      FROM pg_catalog.pg_roles
     WHERE rolname = session_user;
    SELECT oid INTO oid_lector
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_confianza_atestacion_v2_lector_autoridad';
    IF atributos IS NULL OR oid_sesion IS NULL OR oid_lector IS NULL
       OR atributos.rolcanlogin IS NOT TRUE
       OR atributos.rolsuper OR atributos.rolcreatedb OR atributos.rolcreaterole
       OR atributos.rolreplication OR atributos.rolbypassrls
       OR current_user <> 'vec_confianza_atestacion_v2_propietario'
       OR NOT pg_catalog.pg_has_role(
           session_user,
           'vec_confianza_atestacion_v2_lector_autoridad',
           'MEMBER'
       )
       OR (SELECT count(*)
             FROM pg_catalog.pg_auth_members
            WHERE member = oid_sesion) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members
            WHERE member = oid_sesion AND roleid = oid_lector
              AND admin_option IS FALSE
              AND inherit_option IS TRUE
              AND set_option IS TRUE
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad de lectura de confianza V2 rechazada';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    -- El reloj se captura despues de cualquier espera por gobierno. Una
    -- configuracion que caduque mientras se adquiere el lock nunca se acepta.
    instante := clock_timestamp();

    SELECT puntero.orden, puntero.revision INTO puntero_config
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
     WHERE puntero.establecida_en <= instante
     ORDER BY puntero.orden DESC
     LIMIT 1
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN;
    END IF;
    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en, version.numero_raices
      INTO STRICT config
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
     WHERE version.revision = puntero_config.revision
     FOR SHARE;
    IF instante < config.publicada_en OR instante >= config.expira_en THEN
        RETURN;
    END IF;
    PERFORM 1
      FROM vec_confianza_atestacion_v2.revocacion_configuracion
     WHERE revocacion_configuracion.revision = config.revision
       AND revocacion_configuracion.revocada_en <= instante
     FOR SHARE;
    IF FOUND THEN
        RETURN;
    END IF;

    SELECT count(*) INTO numero_miembros
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
     WHERE enlace.configuracion_revision = config.revision;
    IF numero_miembros <> config.numero_raices
       OR numero_miembros NOT BETWEEN 1 AND 64
       OR vec_confianza_atestacion_v2.calcular_huella_configuracion(
              config.revision
          ) IS DISTINCT FROM config.huella_configuracion_sha256 THEN
        RETURN;
    END IF;

    FOR miembro IN
        SELECT enlace.clave_id, enlace.version
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
         WHERE enlace.configuracion_revision = config.revision
         ORDER BY enlace.clave_id COLLATE "C"
         FOR SHARE
    LOOP
        SELECT puntero.version INTO puntero_version
          FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS puntero
         WHERE puntero.clave_id = miembro.clave_id
           AND puntero.establecida_en <= instante
         ORDER BY puntero.orden DESC
         LIMIT 1
         FOR SHARE;
        IF NOT FOUND OR puntero_version <> miembro.version THEN
            RETURN;
        END IF;
        PERFORM 1
          FROM vec_confianza_atestacion_v2.raiz_confianza_version AS raiz
         WHERE raiz.clave_id = miembro.clave_id
           AND raiz.version = miembro.version
         FOR SHARE;
        IF NOT FOUND THEN
            RETURN;
        END IF;
        PERFORM 1
          FROM vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
         WHERE revocacion.clave_id = miembro.clave_id
           AND revocacion.version = miembro.version
         FOR SHARE;
    END LOOP;

    SELECT count(*) INTO raices_activas
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS raiz
        ON raiz.clave_id = enlace.clave_id
       AND raiz.version = enlace.version
      LEFT JOIN vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
        ON revocacion.clave_id = raiz.clave_id
       AND revocacion.version = raiz.version
       AND revocacion.revocada_en <= instante
     WHERE enlace.configuracion_revision = config.revision
       AND revocacion.clave_id IS NULL
       AND instante >= raiz.valida_desde
       AND instante < raiz.valida_hasta;
    IF raices_activas < 1 THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT config.revision,
           config.huella_configuracion_sha256,
           config.publicada_en,
           config.expira_en,
           'activa'::text,
           NULL::timestamptz(6),
           raiz.clave_id,
           raiz.algoritmo_cose,
           raiz.suite,
           raiz.audiencia_despliegue,
           raiz.clave_publica_spki,
           raiz.huella_clave_spki_sha256,
           raiz.valida_desde,
           raiz.valida_hasta,
           CASE WHEN revocacion.clave_id IS NULL
                THEN 'activa'::text ELSE 'revocada'::text END,
           revocacion.revocada_en
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS raiz
        ON raiz.clave_id = enlace.clave_id
       AND raiz.version = enlace.version
      LEFT JOIN vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
        ON revocacion.clave_id = raiz.clave_id
       AND revocacion.version = raiz.version
       AND revocacion.revocada_en <= instante
     WHERE enlace.configuracion_revision = config.revision
     ORDER BY raiz.clave_id COLLATE "C";
END
$funcion$;

-- Cierre explicito de todos los objetos actuales. La guarda DDL mantiene
-- cerrados los tipos fila que aparezcan en futuras migraciones.
REVOKE ALL ON ALL TABLES IN SCHEMA vec_confianza_atestacion_v2
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_confianza_atestacion_v2_migrador;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_confianza_atestacion_v2
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_confianza_atestacion_v2_migrador;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_confianza_atestacion_v2
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_confianza_atestacion_v2_migrador;

DO $cerrar_tipos_actuales$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_confianza_atestacion_v2'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I',
            tipo.nspname,
            tipo.typname,
            'vec_confianza_atestacion_v2_lector_autoridad',
            'vec_confianza_atestacion_v2_migrador'
        );
    END LOOP;
END
$cerrar_tipos_actuales$;

GRANT USAGE ON SCHEMA vec_confianza_atestacion_v2
    TO vec_confianza_atestacion_v2_lector_autoridad;
GRANT EXECUTE ON FUNCTION
    vec_confianza_atestacion_v2.obtener_confianza_actual()
    TO vec_confianza_atestacion_v2_lector_autoridad;

COMMIT;
