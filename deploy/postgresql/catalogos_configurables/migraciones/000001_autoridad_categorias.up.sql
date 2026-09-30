-- Autoridad durable común de categorías. Requiere roles_up.sql.
-- El núcleo V3 consumirá autorización y auditoría central antes de exponer estas
-- funciones a una cuenta de aplicación; esta migración no concede tal acceso.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000001', 0));
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regnamespace('vec_catalogos_configurables') IS NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
                   WHERE nspname = 'vec_catalogos_configurables'
                     AND nspowner <> 'vec_catalogos_configurables_propietario'::regrole)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n
                   ON n.oid = c.relnamespace WHERE n.nspname = 'vec_catalogos_configurables')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p JOIN pg_catalog.pg_namespace AS n
                   ON n.oid = p.pronamespace WHERE n.nspname = 'vec_catalogos_configurables')
       OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL THEN
        RAISE EXCEPTION 'catalogos configurables 000001: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $pre$;
-- En funciones y tipos el PUBLIC por defecto es global; el REVOKE por esquema
-- no lo retira. El propietario solo posee este esquema.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario REVOKE ALL ON TYPES FROM PUBLIC;

CREATE TABLE vec_catalogos_configurables.publicacion (
    catalogo_id text NOT NULL CHECK (catalogo_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
    version integer NOT NULL CHECK (version > 0),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    documento_canonico text NOT NULL CHECK (pg_catalog.octet_length(documento_canonico) BETWEEN 2 AND 16777216),
    preimagenes_control jsonb NOT NULL CHECK (pg_catalog.jsonb_typeof(preimagenes_control) = 'object'),
    preimagenes_huella_sha256 text NOT NULL CHECK (preimagenes_huella_sha256 ~ '^[0-9a-f]{64}$'),
    aprobacion_a_ref text NOT NULL CHECK (pg_catalog.octet_length(aprobacion_a_ref) BETWEEN 3 AND 160),
    aprobacion_b_ref text NOT NULL CHECK (pg_catalog.octet_length(aprobacion_b_ref) BETWEEN 3 AND 160),
    actor_ref text NOT NULL CHECK (pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 160),
    decision_ref text NOT NULL CHECK (pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 160),
    recibo_ref text NOT NULL UNIQUE CHECK (pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
    publicada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    PRIMARY KEY (catalogo_id, version),
    UNIQUE (catalogo_id, version, huella_sha256),
    CHECK (aprobacion_a_ref <> aprobacion_b_ref),
    CHECK (pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento_canonico, 'UTF8')), 'hex') = huella_sha256),
    CHECK (pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(preimagenes_control::text, 'UTF8')), 'hex') = preimagenes_huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.entrada_publicada (
    catalogo_id text NOT NULL,
    version integer NOT NULL,
    huella_sha256 text NOT NULL,
    categoria_id text NOT NULL CHECK (categoria_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
    etiqueta text NOT NULL CHECK (pg_catalog.octet_length(etiqueta) BETWEEN 1 AND 2048),
    definicion jsonb NOT NULL CHECK (pg_catalog.jsonb_typeof(definicion) = 'object'),
    PRIMARY KEY (catalogo_id, version, categoria_id),
    FOREIGN KEY (catalogo_id, version, huella_sha256)
        REFERENCES vec_catalogos_configurables.publicacion (catalogo_id, version, huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.categoria_control (
    categoria_id text PRIMARY KEY,
    catalogo_id text NOT NULL,
    version integer NOT NULL,
    huella_sha256 text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    estado text NOT NULL CHECK (estado IN ('habilitada', 'deshabilitada', 'tombstone')),
    cobertura_verificada boolean NOT NULL DEFAULT false,
    cobertura_ref text,
    total_historico_declarado bigint,
    actualizada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    FOREIGN KEY (catalogo_id, version, categoria_id)
        REFERENCES vec_catalogos_configurables.entrada_publicada (catalogo_id, version, categoria_id),
    CHECK ((NOT cobertura_verificada AND cobertura_ref IS NULL AND total_historico_declarado IS NULL)
        OR (cobertura_verificada AND pg_catalog.octet_length(cobertura_ref) BETWEEN 3 AND 160
            AND total_historico_declarado >= 0))
);
CREATE TABLE vec_catalogos_configurables.uso (
    consumidor text NOT NULL CHECK (consumidor ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
    uso_ref text NOT NULL CHECK (pg_catalog.octet_length(uso_ref) BETWEEN 3 AND 160),
    categoria_id text NOT NULL REFERENCES vec_catalogos_configurables.categoria_control (categoria_id),
    catalogo_id text NOT NULL,
    version integer NOT NULL,
    huella_sha256 text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('reservado', 'confirmado', 'cancelado')),
    revision bigint NOT NULL CHECK (revision BETWEEN 1 AND 2),
    reserva_recibo_ref text NOT NULL UNIQUE,
    terminal_recibo_ref text UNIQUE,
    reservado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    terminal_en timestamptz(6),
    PRIMARY KEY (consumidor, uso_ref),
    FOREIGN KEY (catalogo_id, version, huella_sha256)
        REFERENCES vec_catalogos_configurables.publicacion (catalogo_id, version, huella_sha256),
    FOREIGN KEY (catalogo_id, version, categoria_id)
        REFERENCES vec_catalogos_configurables.entrada_publicada (catalogo_id, version, categoria_id),
    CHECK ((estado = 'reservado' AND revision = 1 AND terminal_recibo_ref IS NULL AND terminal_en IS NULL)
        OR (estado IN ('confirmado', 'cancelado') AND revision = 2 AND terminal_recibo_ref IS NOT NULL AND terminal_en IS NOT NULL))
);
CREATE INDEX uso_reservado_categoria_idx ON vec_catalogos_configurables.uso (categoria_id)
    WHERE estado = 'reservado';
CREATE TABLE vec_catalogos_configurables.historia (
    historia_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    categoria_id text NOT NULL,
    accion text NOT NULL CHECK (accion IN ('publicar', 'reservar', 'confirmar', 'cancelar', 'deshabilitar', 'cobertura', 'tombstone')),
    revision bigint NOT NULL CHECK (revision > 0),
    consumidor text,
    uso_ref text,
    recibo_ref text NOT NULL UNIQUE CHECK (pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 320),
    decision_ref text NOT NULL CHECK (pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 160),
    actor_ref text NOT NULL CHECK (pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 160),
    -- ReferenciaEntradaCatalogo.Referencia() del motivo V3 ya acreditado.
    motivo_ref text NOT NULL CHECK (pg_catalog.octet_length(motivo_ref) BETWEEN 3 AND 320
        AND motivo_ref ~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$'),
    estado_anterior text,
    estado_posterior text NOT NULL,
    cobertura_ref text,
    total_historico_declarado bigint,
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    CHECK ((accion = 'cobertura' AND cobertura_ref IS NOT NULL AND total_historico_declarado >= 0)
         OR (accion <> 'cobertura' AND cobertura_ref IS NULL AND total_historico_declarado IS NULL)),
    CHECK (((accion = 'publicar' AND estado_posterior IN ('habilitada', 'deshabilitada')
               AND (estado_anterior IS NULL OR estado_anterior = estado_posterior))
        OR (accion = 'reservar' AND estado_anterior IS NULL AND estado_posterior = 'reservado')
        OR (accion = 'confirmar' AND estado_anterior = 'reservado' AND estado_posterior = 'confirmado')
        OR (accion = 'cancelar' AND estado_anterior = 'reservado' AND estado_posterior = 'cancelado')
        OR (accion = 'deshabilitar' AND estado_anterior = 'habilitada' AND estado_posterior = 'deshabilitada')
        OR (accion = 'cobertura' AND estado_anterior = 'deshabilitada' AND estado_posterior = 'deshabilitada')
        OR (accion = 'tombstone' AND estado_anterior = 'deshabilitada' AND estado_posterior = 'tombstone')) IS TRUE)
);

-- Ninguna tabla tiene políticas de aplicación. Solo el propietario NOLOGIN
-- ejecuta las funciones definidoras. FORCE RLS sigue activo en todos los hechos.
ALTER TABLE vec_catalogos_configurables.publicacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.publicacion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.entrada_publicada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.entrada_publicada FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.categoria_control ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.categoria_control FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.uso ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.uso FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.historia FORCE ROW LEVEL SECURITY;
CREATE POLICY publicacion_propietario ON vec_catalogos_configurables.publicacion TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
CREATE POLICY entrada_propietario ON vec_catalogos_configurables.entrada_publicada TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
CREATE POLICY control_propietario ON vec_catalogos_configurables.categoria_control TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
CREATE POLICY uso_propietario ON vec_catalogos_configurables.uso TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
CREATE POLICY historia_propietario ON vec_catalogos_configurables.historia TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON ALL TABLES IN SCHEMA vec_catalogos_configurables FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_catalogos_configurables FROM PUBLIC;
REVOKE ALL ON TYPE vec_catalogos_configurables.publicacion,
    vec_catalogos_configurables.entrada_publicada,
    vec_catalogos_configurables.categoria_control,
    vec_catalogos_configurables.uso,
    vec_catalogos_configurables.historia FROM PUBLIC;

CREATE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    RAISE EXCEPTION 'publicacion o historia inmutable' USING ERRCODE = '55000';
END $f$;
CREATE TRIGGER publicacion_inmutable BEFORE UPDATE OR DELETE ON vec_catalogos_configurables.publicacion
    FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER entrada_inmutable BEFORE UPDATE OR DELETE ON vec_catalogos_configurables.entrada_publicada
    FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_catalogos_configurables.historia
    FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER control_no_borrar BEFORE DELETE ON vec_catalogos_configurables.categoria_control
    FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER uso_no_borrar BEFORE DELETE ON vec_catalogos_configurables.uso
    FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER publicacion_no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.publicacion
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER entrada_no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.entrada_publicada
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER control_no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.categoria_control
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER uso_no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.uso
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.historia
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();

-- El documento exacto es el JSON canónico de Go; se conserva como bytes UTF-8
-- y se coteja su SHA-256. Nunca se reserializa JSONB para reconstruir la huella.
CREATE FUNCTION vec_catalogos_configurables.publicar(
    p_catalogo_id text, p_version integer, p_huella text, p_documento text,
    p_preimagenes jsonb, p_preimagenes_huella text,
    p_aprobacion_a text, p_aprobacion_b text, p_actor text, p_decision text, p_recibo text,
    p_motivo_ref text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET lock_timeout = '5s' SET statement_timeout = '30s' AS $f$
DECLARE
    contenido jsonb;
    item jsonb;
    clave text;
    etiqueta text;
    total integer := 0;
    anterior vec_catalogos_configurables.publicacion%ROWTYPE;
    control_anterior vec_catalogos_configurables.categoria_control%ROWTYPE;
    preimagenes_restantes jsonb;
BEGIN
    IF p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_version IS NULL OR p_version < 1
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_documento IS NULL OR pg_catalog.octet_length(p_documento) > 16777216
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_documento, 'UTF8')), 'hex') <> p_huella
       OR p_preimagenes IS NULL OR pg_catalog.jsonb_typeof(p_preimagenes) <> 'object'
       OR pg_catalog.octet_length(p_preimagenes::text) > 1048576
       OR p_preimagenes_huella IS NULL OR p_preimagenes_huella !~ '^[0-9a-f]{64}$'
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_preimagenes::text, 'UTF8')), 'hex') <> p_preimagenes_huella
       OR p_aprobacion_a IS NULL OR pg_catalog.octet_length(p_aprobacion_a) NOT BETWEEN 3 AND 160
       OR p_aprobacion_b IS NULL OR pg_catalog.octet_length(p_aprobacion_b) NOT BETWEEN 3 AND 160
       OR p_aprobacion_a = p_aprobacion_b
       OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 3 AND 160
       OR p_recibo IS NULL OR pg_catalog.octet_length(p_recibo) NOT BETWEEN 3 AND 160
       OR p_motivo_ref IS NULL OR pg_catalog.octet_length(p_motivo_ref) NOT BETWEEN 3 AND 320
       OR p_motivo_ref !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
        RAISE EXCEPTION 'publicacion invalida' USING ERRCODE = '22023';
    END IF;
    contenido := p_documento::jsonb;
    IF pg_catalog.jsonb_typeof(contenido) <> 'object'
       OR contenido->>'id' IS DISTINCT FROM p_catalogo_id
       OR (contenido->>'version') IS DISTINCT FROM p_version::text
       OR contenido->>'estado' IS DISTINCT FROM 'publicado'
       OR pg_catalog.jsonb_typeof(contenido->'entradas') IS DISTINCT FROM 'array'
       OR pg_catalog.jsonb_array_length(contenido->'entradas') NOT BETWEEN 1 AND 10000 THEN
        RAISE EXCEPTION 'documento de publicacion incompatible' USING ERRCODE = '22023';
    END IF;
    preimagenes_restantes := p_preimagenes;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:' || p_catalogo_id, 0));
    SELECT * INTO anterior FROM vec_catalogos_configurables.publicacion
     WHERE catalogo_id = p_catalogo_id AND version = p_version;
    IF FOUND THEN
        IF anterior.huella_sha256 = p_huella AND anterior.documento_canonico = p_documento
           AND anterior.preimagenes_control = p_preimagenes
           AND anterior.preimagenes_huella_sha256 = p_preimagenes_huella
           AND anterior.aprobacion_a_ref = p_aprobacion_a AND anterior.aprobacion_b_ref = p_aprobacion_b
           AND anterior.recibo_ref = p_recibo THEN
            RETURN p_recibo;
        END IF;
        RAISE EXCEPTION 'version de publicacion en conflicto' USING ERRCODE = '23505';
    END IF;
    IF (p_version > 1 AND NOT EXISTS (
            SELECT 1 FROM vec_catalogos_configurables.publicacion
             WHERE catalogo_id = p_catalogo_id AND version = p_version - 1)) THEN
        RAISE EXCEPTION 'version de publicacion en conflicto' USING ERRCODE = '23505';
    END IF;
    INSERT INTO vec_catalogos_configurables.publicacion
        (catalogo_id, version, huella_sha256, documento_canonico,
         preimagenes_control, preimagenes_huella_sha256, aprobacion_a_ref,
         aprobacion_b_ref, actor_ref, decision_ref, recibo_ref)
    VALUES (p_catalogo_id, p_version, p_huella, p_documento,
            p_preimagenes, p_preimagenes_huella, p_aprobacion_a,
            p_aprobacion_b, p_actor, p_decision, p_recibo);
    FOR item IN SELECT value FROM pg_catalog.jsonb_array_elements(contenido->'entradas') LOOP
        clave := item->>'clave';
        etiqueta := item->>'etiqueta';
        IF pg_catalog.jsonb_typeof(item) <> 'object'
           OR clave IS NULL OR clave !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR etiqueta IS NULL OR pg_catalog.octet_length(etiqueta) NOT BETWEEN 1 AND 2048 THEN
            RAISE EXCEPTION 'entrada publicada invalida' USING ERRCODE = '22023';
        END IF;
        SELECT * INTO control_anterior FROM vec_catalogos_configurables.categoria_control
         WHERE categoria_id = clave FOR UPDATE;
        IF FOUND THEN
            IF control_anterior.catalogo_id <> p_catalogo_id
               OR control_anterior.estado = 'tombstone'
               OR p_preimagenes->clave IS DISTINCT FROM pg_catalog.jsonb_build_object(
                    'version', control_anterior.version,
                    'huella_sha256', control_anterior.huella_sha256,
                    'revision', control_anterior.revision,
                    'estado', control_anterior.estado) THEN
                RAISE EXCEPTION 'preimagen de categoria obsoleta' USING ERRCODE = '40001';
            END IF;
            preimagenes_restantes := preimagenes_restantes - clave;
        ELSIF p_preimagenes ? clave THEN
            RAISE EXCEPTION 'preimagen de categoria inexistente' USING ERRCODE = '22023';
        END IF;
        INSERT INTO vec_catalogos_configurables.entrada_publicada
            (catalogo_id, version, huella_sha256, categoria_id, etiqueta, definicion)
        VALUES (p_catalogo_id, p_version, p_huella, clave, etiqueta, item);
        IF control_anterior.categoria_id IS NULL THEN
            INSERT INTO vec_catalogos_configurables.categoria_control
                (categoria_id, catalogo_id, version, huella_sha256, revision, estado)
            VALUES (clave, p_catalogo_id, p_version, p_huella, 1, 'habilitada');
        ELSE
            UPDATE vec_catalogos_configurables.categoria_control
               SET version = p_version, huella_sha256 = p_huella, revision = revision + 1,
                   cobertura_verificada = false, cobertura_ref = NULL,
                   total_historico_declarado = NULL,
                   actualizada_en = pg_catalog.clock_timestamp()
             WHERE categoria_id = clave AND catalogo_id = p_catalogo_id
               AND version = control_anterior.version
               AND huella_sha256 = control_anterior.huella_sha256
               AND revision = control_anterior.revision
               AND estado = control_anterior.estado;
            IF NOT FOUND THEN
                RAISE EXCEPTION 'CAS de categoria fallido' USING ERRCODE = '40001';
            END IF;
        END IF;
        INSERT INTO vec_catalogos_configurables.historia
            (categoria_id, accion, revision, recibo_ref, decision_ref, actor_ref,
             motivo_ref, estado_anterior, estado_posterior)
        SELECT clave, 'publicar', revision, p_recibo || ':' || clave, p_decision, p_actor,
               p_motivo_ref, control_anterior.estado, estado
          FROM vec_catalogos_configurables.categoria_control WHERE categoria_id = clave;
        total := total + 1;
    END LOOP;
    IF total <> pg_catalog.jsonb_array_length(contenido->'entradas') THEN
        RAISE EXCEPTION 'publicacion incompleta' USING ERRCODE = '55000';
    END IF;
    IF preimagenes_restantes <> '{}'::jsonb THEN
        RAISE EXCEPTION 'preimagen sin categoria publicada' USING ERRCODE = '22023';
    END IF;
    RETURN p_recibo;
END $f$;

-- Una identidad de uso solo puede tener una reserva. El recibo de replay es
-- idéntico; una cancelación no reabre la identidad.
CREATE FUNCTION vec_catalogos_configurables.reservar(
    p_consumidor text, p_uso_ref text, p_categoria_id text, p_catalogo_id text,
    p_version integer, p_huella text, p_actor text, p_decision text, p_recibo text,
    p_motivo_ref text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET lock_timeout = '5s' SET statement_timeout = '30s' AS $f$
DECLARE
    c vec_catalogos_configurables.categoria_control%ROWTYPE;
    anterior vec_catalogos_configurables.uso%ROWTYPE;
BEGIN
    IF p_consumidor IS NULL OR p_consumidor !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_uso_ref IS NULL OR pg_catalog.octet_length(p_uso_ref) NOT BETWEEN 3 AND 160
       OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 3 AND 160
       OR p_recibo IS NULL OR pg_catalog.octet_length(p_recibo) NOT BETWEEN 3 AND 160
       OR p_motivo_ref IS NULL OR pg_catalog.octet_length(p_motivo_ref) NOT BETWEEN 3 AND 320
       OR p_motivo_ref !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
        RAISE EXCEPTION 'reserva invalida' USING ERRCODE = '22023';
    END IF;
    SELECT * INTO c FROM vec_catalogos_configurables.categoria_control
     WHERE categoria_id = p_categoria_id FOR UPDATE;
    SELECT * INTO anterior FROM vec_catalogos_configurables.uso
     WHERE consumidor = p_consumidor AND uso_ref = p_uso_ref FOR UPDATE;
    IF FOUND THEN
        IF anterior.categoria_id = p_categoria_id AND anterior.catalogo_id = p_catalogo_id
           AND anterior.version = p_version AND anterior.huella_sha256 = p_huella
           AND anterior.reserva_recibo_ref = p_recibo THEN
            RETURN anterior.reserva_recibo_ref;
        END IF;
        RAISE EXCEPTION 'identidad de uso ya ocupada' USING ERRCODE = '23505';
    END IF;
    IF c.categoria_id IS NULL OR c.estado <> 'habilitada' OR c.catalogo_id IS DISTINCT FROM p_catalogo_id
       OR c.version IS DISTINCT FROM p_version OR c.huella_sha256 IS DISTINCT FROM p_huella THEN
        RAISE EXCEPTION 'categoria no reservable' USING ERRCODE = '55000';
    END IF;
    INSERT INTO vec_catalogos_configurables.uso
        (consumidor, uso_ref, categoria_id, catalogo_id, version, huella_sha256,
         estado, revision, reserva_recibo_ref)
    VALUES (p_consumidor, p_uso_ref, p_categoria_id, p_catalogo_id, p_version, p_huella,
            'reservado', 1, p_recibo);
    INSERT INTO vec_catalogos_configurables.historia
        (categoria_id, accion, revision, consumidor, uso_ref, recibo_ref, decision_ref, actor_ref,
         motivo_ref, estado_anterior, estado_posterior)
    VALUES (p_categoria_id, 'reservar', 1, p_consumidor, p_uso_ref, p_recibo, p_decision, p_actor,
            p_motivo_ref, NULL, 'reservado');
    RETURN p_recibo;
END $f$;

CREATE FUNCTION vec_catalogos_configurables.terminar_uso(
    p_consumidor text, p_uso_ref text, p_reserva_recibo text, p_estado text,
    p_actor text, p_decision text, p_recibo text, p_motivo_ref text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET lock_timeout = '5s' SET statement_timeout = '30s' AS $f$
DECLARE
    u vec_catalogos_configurables.uso%ROWTYPE;
BEGIN
    IF p_estado IS NULL OR p_estado NOT IN ('confirmado', 'cancelado')
       OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 3 AND 160
       OR p_recibo IS NULL OR pg_catalog.octet_length(p_recibo) NOT BETWEEN 3 AND 160
       OR p_motivo_ref IS NULL OR pg_catalog.octet_length(p_motivo_ref) NOT BETWEEN 3 AND 320
       OR p_motivo_ref !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
        RAISE EXCEPTION 'terminal de uso invalido' USING ERRCODE = '22023';
    END IF;
    SELECT * INTO u FROM vec_catalogos_configurables.uso
     WHERE consumidor = p_consumidor AND uso_ref = p_uso_ref;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'reserva de uso ausente' USING ERRCODE = '55000';
    END IF;
    PERFORM 1 FROM vec_catalogos_configurables.categoria_control
     WHERE categoria_id = u.categoria_id FOR UPDATE;
    SELECT * INTO u FROM vec_catalogos_configurables.uso
     WHERE consumidor = p_consumidor AND uso_ref = p_uso_ref FOR UPDATE;
    IF NOT FOUND OR u.reserva_recibo_ref IS DISTINCT FROM p_reserva_recibo THEN
        RAISE EXCEPTION 'reserva de uso ausente' USING ERRCODE = '55000';
    END IF;
    IF u.estado = p_estado AND u.terminal_recibo_ref = p_recibo THEN
        RETURN p_recibo;
    END IF;
    IF u.estado <> 'reservado' THEN
        RAISE EXCEPTION 'uso ya terminal' USING ERRCODE = '23505';
    END IF;
    UPDATE vec_catalogos_configurables.uso SET
        estado = p_estado, revision = 2, terminal_recibo_ref = p_recibo,
        terminal_en = pg_catalog.clock_timestamp()
    WHERE consumidor = p_consumidor AND uso_ref = p_uso_ref AND revision = 1 AND estado = 'reservado';
    IF NOT FOUND THEN RAISE EXCEPTION 'CAS de uso fallido' USING ERRCODE = '40001'; END IF;
    INSERT INTO vec_catalogos_configurables.historia
        (categoria_id, accion, revision, consumidor, uso_ref, recibo_ref, decision_ref, actor_ref,
         motivo_ref, estado_anterior, estado_posterior)
    VALUES (u.categoria_id, CASE WHEN p_estado = 'confirmado' THEN 'confirmar' ELSE 'cancelar' END,
            2, p_consumidor, p_uso_ref, p_recibo, p_decision, p_actor,
            p_motivo_ref, 'reservado', p_estado);
    RETURN p_recibo;
END $f$;

CREATE FUNCTION vec_catalogos_configurables.cambiar_proyeccion(
    p_categoria_id text, p_revision bigint, p_accion text, p_cobertura_ref text,
    p_total_historico bigint, p_actor text, p_decision text, p_recibo text,
    p_motivo_ref text
) RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET lock_timeout = '5s' SET statement_timeout = '30s' AS $f$
DECLARE
    c vec_catalogos_configurables.categoria_control%ROWTYPE;
    nuevos bigint;
BEGIN
    IF p_accion IS NULL OR p_accion NOT IN ('deshabilitar', 'cobertura', 'tombstone')
       OR (p_accion <> 'cobertura' AND (p_cobertura_ref IS NOT NULL OR p_total_historico IS NOT NULL))
       OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 3 AND 160
       OR p_recibo IS NULL OR pg_catalog.octet_length(p_recibo) NOT BETWEEN 3 AND 160
       OR p_motivo_ref IS NULL OR pg_catalog.octet_length(p_motivo_ref) NOT BETWEEN 3 AND 320
       OR p_motivo_ref !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
        RAISE EXCEPTION 'cambio de proyeccion invalido' USING ERRCODE = '22023';
    END IF;
    SELECT * INTO c FROM vec_catalogos_configurables.categoria_control
     WHERE categoria_id = p_categoria_id FOR UPDATE;
    IF c.categoria_id IS NOT NULL
       AND EXISTS (SELECT 1 FROM vec_catalogos_configurables.historia
                    WHERE categoria_id = p_categoria_id AND accion = p_accion
                      AND revision = p_revision + 1 AND recibo_ref = p_recibo
                      AND cobertura_ref IS NOT DISTINCT FROM p_cobertura_ref
                      AND total_historico_declarado IS NOT DISTINCT FROM p_total_historico) THEN
        RETURN p_revision + 1;
    END IF;
    IF c.categoria_id IS NULL OR c.revision IS DISTINCT FROM p_revision OR c.estado = 'tombstone' THEN
        RAISE EXCEPTION 'revision de categoria en conflicto' USING ERRCODE = '40001';
    END IF;
    IF p_accion = 'deshabilitar' THEN
        IF c.estado <> 'habilitada' THEN RAISE EXCEPTION 'categoria ya deshabilitada' USING ERRCODE = '55000'; END IF;
        UPDATE vec_catalogos_configurables.categoria_control
           SET estado = 'deshabilitada', revision = revision + 1,
               actualizada_en = pg_catalog.clock_timestamp()
         WHERE categoria_id = p_categoria_id AND revision = p_revision;
    ELSIF p_accion = 'cobertura' THEN
        IF c.estado <> 'deshabilitada' OR c.cobertura_verificada OR p_cobertura_ref IS NULL
           OR pg_catalog.octet_length(p_cobertura_ref) NOT BETWEEN 3 AND 160
           OR p_total_historico IS NULL OR p_total_historico < 0
           OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso
                       WHERE categoria_id = p_categoria_id AND estado = 'reservado')
           OR p_total_historico < COALESCE(
                (SELECT pg_catalog.max(h.total_historico_declarado)
                   FROM vec_catalogos_configurables.historia h
                  WHERE h.categoria_id = p_categoria_id AND h.accion = 'cobertura'), 0)
           OR p_total_historico < (SELECT count(*) FROM vec_catalogos_configurables.uso
                                    WHERE categoria_id = p_categoria_id AND estado = 'confirmado') THEN
            RAISE EXCEPTION 'cobertura historica no acreditada' USING ERRCODE = '55000';
        END IF;
        UPDATE vec_catalogos_configurables.categoria_control
           SET cobertura_verificada = true, cobertura_ref = p_cobertura_ref,
               total_historico_declarado = p_total_historico, revision = revision + 1,
               actualizada_en = pg_catalog.clock_timestamp()
         WHERE categoria_id = p_categoria_id AND revision = p_revision;
    ELSE
        IF c.estado <> 'deshabilitada' OR NOT c.cobertura_verificada
           OR c.total_historico_declarado IS DISTINCT FROM 0
           OR c.total_historico_declarado < (SELECT count(*) FROM vec_catalogos_configurables.uso
                                               WHERE categoria_id = p_categoria_id AND estado = 'confirmado')
           OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso
                       WHERE categoria_id = p_categoria_id AND estado = 'reservado')
           OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso
                       WHERE categoria_id = p_categoria_id AND estado IN ('reservado', 'confirmado'))
           OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.historia h
                       WHERE h.categoria_id = p_categoria_id AND h.accion = 'cobertura'
                         AND h.total_historico_declarado > 0) THEN
            RAISE EXCEPTION 'proyeccion con reservas o sin cobertura' USING ERRCODE = '55000';
        END IF;
        UPDATE vec_catalogos_configurables.categoria_control
           SET estado = 'tombstone', revision = revision + 1,
               actualizada_en = pg_catalog.clock_timestamp()
         WHERE categoria_id = p_categoria_id AND revision = p_revision;
    END IF;
    IF NOT FOUND THEN RAISE EXCEPTION 'CAS de categoria fallido' USING ERRCODE = '40001'; END IF;
    nuevos := p_revision + 1;
    INSERT INTO vec_catalogos_configurables.historia
        (categoria_id, accion, revision, recibo_ref, decision_ref, actor_ref,
         motivo_ref, estado_anterior, estado_posterior,
         cobertura_ref, total_historico_declarado)
    VALUES (p_categoria_id, p_accion, nuevos, p_recibo, p_decision, p_actor,
            p_motivo_ref, c.estado,
            CASE p_accion WHEN 'deshabilitar' THEN 'deshabilitada'
                          WHEN 'tombstone' THEN 'tombstone' ELSE c.estado END,
            p_cobertura_ref, p_total_historico);
    RETURN nuevos;
END $f$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_catalogos_configurables FROM PUBLIC;
-- Solo el propietario NOLOGIN de AD3 podrá llamar al core desde sus wrappers
-- SECURITY DEFINER, después de consumir la decisión V3 en la misma transacción.
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text),
    vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text),
    vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text),
    vec_catalogos_configurables.cambiar_proyeccion(text,bigint,text,text,bigint,text,text,text,text)
    TO vec_autorizacion_atestada_v3_propietario;
DO $acl$
DECLARE f regprocedure;
BEGIN
    FOREACH f IN ARRAY ARRAY[
      'vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)'::regprocedure,
      'vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text)'::regprocedure,
      'vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text)'::regprocedure,
      'vec_catalogos_configurables.cambiar_proyeccion(text,bigint,text,text,bigint,text,text,text,text)'::regprocedure
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
             AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef
             AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=5s','statement_timeout=30s'])
           OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
               WHERE p.oid=f AND (a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable
                 OR a.grantee NOT IN (p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
            RAISE EXCEPTION 'catalogos configurables 000001: ACL o configuracion incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
COMMIT;
