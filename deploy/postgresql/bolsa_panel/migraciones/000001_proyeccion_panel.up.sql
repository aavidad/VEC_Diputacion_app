-- Proyeccion operativa versionada, autoridad criptografica preparada,
-- consumos idempotentes y auditoria encadenada del panel interno de bolsas.
BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_panel') IS NOT NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_panel_bolsa_v2(jsonb,bytea,bytea,text,text,text,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar el almacen del panel';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_panel AUTHORIZATION vec_bolsa_panel_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_panel FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    IN SCHEMA vec_bolsa_panel REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    IN SCHEMA vec_bolsa_panel REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_bolsa_panel.referencia_opaca_valida(
    p_valor text, p_prefijo text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[a-z]{3}_[a-z0-9]{16,80}$'
       AND left(p_valor, length(p_prefijo)) = p_prefijo
$funcion$;

CREATE FUNCTION vec_bolsa_panel.clave_catalogo_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[a-z][a-z0-9_.-]{1,79}$'
$funcion$;

CREATE FUNCTION vec_bolsa_panel.instante_utc_microsegundo_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    convertido timestamptz;
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    convertido := p_valor::timestamptz;
    RETURN isfinite(convertido)
       AND to_char(convertido AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') = p_valor;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.contador_valido(p_valor jsonb)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT jsonb_typeof(p_valor) = 'number'
       AND p_valor #>> '{}' ~ '^(0|[1-9][0-9]{0,9})$'
       AND (p_valor #>> '{}')::numeric BETWEEN 0 AND 1000000000
$funcion$;

CREATE FUNCTION vec_bolsa_panel.selector_valido(p_selector jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    clase text;
BEGIN
    IF jsonb_typeof(p_selector) <> 'object'
       OR jsonb_typeof(p_selector -> 'clase') <> 'string'
       OR jsonb_typeof(p_selector -> 'organizacion_ref') <> 'string'
       OR vec_bolsa_panel.referencia_opaca_valida(
              p_selector ->> 'organizacion_ref', 'org_'
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    clase := p_selector ->> 'clase';
    IF clase = 'organizacion' THEN
        RETURN (SELECT count(*) FROM jsonb_object_keys(p_selector)) = 2
           AND NOT (p_selector ? 'unidad_gestion_ref');
    END IF;
    IF clase = 'unidad_gestion' THEN
        RETURN (SELECT count(*) FROM jsonb_object_keys(p_selector)) = 3
           AND jsonb_typeof(p_selector -> 'unidad_gestion_ref') = 'string'
           AND vec_bolsa_panel.referencia_opaca_valida(
                  p_selector ->> 'unidad_gestion_ref', 'uni_'
               );
    END IF;
    RETURN false;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.indicadores_validos(p_valor jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    clave text;
BEGIN
    IF jsonb_typeof(p_valor) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_valor)) <> 12
       OR NOT (p_valor ?& ARRAY[
           'convocatorias_borrador', 'convocatorias_revision',
           'convocatorias_pendientes_firma', 'convocatorias_publicadas',
           'bolsas_activas', 'bolsas_suspendidas', 'bolsas_agotadas',
           'llamamientos_pendientes', 'llamamientos_en_curso',
           'llamamientos_vencen_hoy', 'documentos_pendientes_firma',
           'incidencias_abiertas'
       ]) THEN
        RETURN false;
    END IF;
    FOREACH clave IN ARRAY ARRAY[
        'convocatorias_borrador', 'convocatorias_revision',
        'convocatorias_pendientes_firma', 'convocatorias_publicadas',
        'bolsas_activas', 'bolsas_suspendidas', 'bolsas_agotadas',
        'llamamientos_pendientes', 'llamamientos_en_curso',
        'llamamientos_vencen_hoy', 'documentos_pendientes_firma',
        'incidencias_abiertas'
    ] LOOP
        IF vec_bolsa_panel.contador_valido(p_valor -> clave) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.convocatoria_resumen_valida(p_valor jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    numero_solicitudes numeric;
    numero_pendientes numeric;
BEGIN
    IF jsonb_typeof(p_valor) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_valor)) NOT IN (5, 6)
       OR NOT (p_valor ?& ARRAY[
           'convocatoria_ref', 'categoria_clave', 'estado_clave',
           'numero_solicitudes', 'numero_pendientes'
       ])
       OR ((SELECT count(*) FROM jsonb_object_keys(p_valor)) = 6
           AND NOT (p_valor ? 'plazo_cierra_en'))
       OR jsonb_typeof(p_valor -> 'convocatoria_ref') <> 'string'
       OR vec_bolsa_panel.referencia_opaca_valida(
              p_valor ->> 'convocatoria_ref', 'cnv_'
          ) IS NOT TRUE
       OR jsonb_typeof(p_valor -> 'categoria_clave') <> 'string'
       OR vec_bolsa_panel.clave_catalogo_valida(
              p_valor ->> 'categoria_clave'
          ) IS NOT TRUE
       OR jsonb_typeof(p_valor -> 'estado_clave') <> 'string'
       OR vec_bolsa_panel.clave_catalogo_valida(
              p_valor ->> 'estado_clave'
          ) IS NOT TRUE
       OR vec_bolsa_panel.contador_valido(
              p_valor -> 'numero_solicitudes'
          ) IS NOT TRUE
       OR vec_bolsa_panel.contador_valido(
              p_valor -> 'numero_pendientes'
          ) IS NOT TRUE
       OR (p_valor ? 'plazo_cierra_en' AND (
           jsonb_typeof(p_valor -> 'plazo_cierra_en') <> 'string'
           OR vec_bolsa_panel.instante_utc_microsegundo_valido(
                  p_valor ->> 'plazo_cierra_en'
              ) IS NOT TRUE
       )) THEN
        RETURN false;
    END IF;
    numero_solicitudes := (p_valor ->> 'numero_solicitudes')::numeric;
    numero_pendientes := (p_valor ->> 'numero_pendientes')::numeric;
    RETURN numero_pendientes <= numero_solicitudes;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.actuacion_pendiente_valida(p_valor jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF jsonb_typeof(p_valor) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_valor)) NOT IN (6, 7)
       OR NOT (p_valor ?& ARRAY[
           'actuacion_ref', 'recurso_ref', 'tipo_clave', 'estado_clave',
           'prioridad_clave', 'numero_elementos'
       ])
       OR ((SELECT count(*) FROM jsonb_object_keys(p_valor)) = 7
           AND NOT (p_valor ? 'fecha_limite'))
       OR jsonb_typeof(p_valor -> 'actuacion_ref') <> 'string'
       OR vec_bolsa_panel.referencia_opaca_valida(
              p_valor ->> 'actuacion_ref', 'act_'
          ) IS NOT TRUE
       OR jsonb_typeof(p_valor -> 'recurso_ref') <> 'string'
       OR (p_valor ->> 'recurso_ref') !~ '^[a-z]{3}_[a-z0-9]{16,80}$'
       OR jsonb_typeof(p_valor -> 'tipo_clave') <> 'string'
       OR vec_bolsa_panel.clave_catalogo_valida(
              p_valor ->> 'tipo_clave'
          ) IS NOT TRUE
       OR jsonb_typeof(p_valor -> 'estado_clave') <> 'string'
       OR vec_bolsa_panel.clave_catalogo_valida(
              p_valor ->> 'estado_clave'
          ) IS NOT TRUE
       OR jsonb_typeof(p_valor -> 'prioridad_clave') <> 'string'
       OR vec_bolsa_panel.clave_catalogo_valida(
              p_valor ->> 'prioridad_clave'
          ) IS NOT TRUE
       OR vec_bolsa_panel.contador_valido(
              p_valor -> 'numero_elementos'
          ) IS NOT TRUE
       OR (p_valor ? 'fecha_limite' AND (
           jsonb_typeof(p_valor -> 'fecha_limite') <> 'string'
           OR vec_bolsa_panel.instante_utc_microsegundo_valido(
                  p_valor ->> 'fecha_limite'
              ) IS NOT TRUE
       )) THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

CREATE TABLE vec_bolsa_panel.proyeccion_panel (
    clase_ambito text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text NOT NULL DEFAULT '',
    revision text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    indicadores jsonb NOT NULL,
    documento_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ),
    CONSTRAINT proyeccion_selector_exacto CHECK (
        vec_bolsa_panel.referencia_opaca_valida(
            organizacion_ref, 'org_'
        ) IS TRUE
        AND (
            (clase_ambito = 'organizacion' AND unidad_gestion_ref = '')
            OR
            (clase_ambito = 'unidad_gestion'
             AND vec_bolsa_panel.referencia_opaca_valida(
                 unidad_gestion_ref, 'uni_'
             ) IS TRUE)
        )
    ),
    CONSTRAINT proyeccion_revision_valida CHECK (
        vec_bolsa_panel.referencia_opaca_valida(revision, 'rev_') IS TRUE
    ),
    CONSTRAINT proyeccion_indicadores_validos CHECK (
        vec_bolsa_panel.indicadores_validos(indicadores) IS TRUE
    ),
    CONSTRAINT proyeccion_huella_valida CHECK (
        documento_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND documento_huella_sha256 <> repeat('0', 64)
    ),
    CONSTRAINT proyeccion_tiempos_validos CHECK (
        actualizada_en <= registrada_en
    )
);

CREATE TABLE vec_bolsa_panel.proyeccion_actual (
    clase_ambito text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text NOT NULL DEFAULT '',
    revision text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (clase_ambito, organizacion_ref, unidad_gestion_ref),
    FOREIGN KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ) REFERENCES vec_bolsa_panel.proyeccion_panel(
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    )
);

CREATE TABLE vec_bolsa_panel.convocatoria_resumen (
    clase_ambito text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text NOT NULL DEFAULT '',
    revision text NOT NULL,
    convocatoria_ref text NOT NULL,
    ordinal smallint NOT NULL,
    documento jsonb NOT NULL,
    PRIMARY KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
        convocatoria_ref
    ),
    UNIQUE (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision, ordinal
    ),
    FOREIGN KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ) REFERENCES vec_bolsa_panel.proyeccion_panel(
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ),
    CONSTRAINT convocatoria_ordinal_acotado CHECK (ordinal BETWEEN 1 AND 40),
    CONSTRAINT convocatoria_documento_valido CHECK (
        vec_bolsa_panel.convocatoria_resumen_valida(documento) IS TRUE
        AND documento ->> 'convocatoria_ref' = convocatoria_ref
    )
);

CREATE TABLE vec_bolsa_panel.actuacion_pendiente (
    clase_ambito text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text NOT NULL DEFAULT '',
    revision text NOT NULL,
    actuacion_ref text NOT NULL,
    ordinal smallint NOT NULL,
    documento jsonb NOT NULL,
    PRIMARY KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
        actuacion_ref
    ),
    UNIQUE (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision, ordinal
    ),
    FOREIGN KEY (
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ) REFERENCES vec_bolsa_panel.proyeccion_panel(
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision
    ),
    CONSTRAINT actuacion_ordinal_acotado CHECK (ordinal BETWEEN 1 AND 80),
    CONSTRAINT actuacion_documento_valido CHECK (
        vec_bolsa_panel.actuacion_pendiente_valida(documento) IS TRUE
        AND documento ->> 'actuacion_ref' = actuacion_ref
    )
);

-- Esta tabla no posee escritor runtime. Una migracion futura debera conectar
-- el verificador COSE aislado antes de conceder EXECUTE a la consulta.
CREATE TABLE vec_bolsa_panel.atestacion_autorizacion_version (
    decision_ref text NOT NULL,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    evidencia_canonica bytea NOT NULL,
    huella_evidencia_sha256 text NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    clave_id text NOT NULL,
    revision_confianza text NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version, estado),
    CONSTRAINT atestacion_version_positiva CHECK (version > 0),
    CONSTRAINT atestacion_estado_cerrado CHECK (
        estado IN ('activa', 'revocada')
    ),
    CONSTRAINT atestacion_referencias_validas CHECK (
        octet_length(decision_ref) BETWEEN 1 AND 512
        AND decision_ref !~ '[*[:space:][:cntrl:]]'
        AND vec_bolsa_panel.referencia_opaca_valida(
            atestacion_ref, 'ate_'
        ) IS TRUE
        AND octet_length(clave_id) BETWEEN 1 AND 512
        AND clave_id !~ '[*[:space:][:cntrl:]]'
        AND octet_length(revision_confianza) BETWEEN 1 AND 128
        AND revision_confianza !~ '[*[:space:][:cntrl:]]'
    ),
    CONSTRAINT atestacion_huellas_validas CHECK (
        huella_decision_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_evidencia_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_sobre_sha256 ~ '^[0-9a-f]{64}$'
        AND encode(sha256(evidencia_canonica), 'hex') =
            huella_evidencia_sha256
        AND encode(sha256(sobre_cose_sign1), 'hex') = huella_sobre_sha256
    ),
    CONSTRAINT atestacion_tamanos_acotados CHECK (
        octet_length(evidencia_canonica) BETWEEN 1 AND 2097152
        AND octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384
    ),
    CONSTRAINT atestacion_ventana_valida CHECK (
        valida_desde <= verificada_en AND verificada_en < valida_hasta
        AND registrada_en >= verificada_en
    )
);

CREATE TABLE vec_bolsa_panel.atestacion_autorizacion_actual (
    decision_ref text PRIMARY KEY,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref, atestacion_ref, version, estado)
        REFERENCES vec_bolsa_panel.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version, estado
        ),
    CONSTRAINT atestacion_actual_estado CHECK (
        estado IN ('activa', 'revocada')
    )
);

CREATE TABLE vec_bolsa_panel.auditoria (
    auditoria_ref text PRIMARY KEY,
    secuencia bigint NOT NULL UNIQUE,
    lectura_ref text NOT NULL UNIQUE,
    registro_canonico bytea NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_auditoria_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_secuencia_positiva CHECK (secuencia > 0),
    CONSTRAINT auditoria_referencias_validas CHECK (
        vec_bolsa_panel.referencia_opaca_valida(
            auditoria_ref, 'aud_'
        ) IS TRUE
        AND vec_bolsa_panel.referencia_opaca_valida(
            lectura_ref, 'lec_'
        ) IS TRUE
    ),
    CONSTRAINT auditoria_huellas_validas CHECK (
        huella_anterior_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_auditoria_sha256 ~ '^[0-9a-f]{64}$'
        AND octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(
            decode(huella_anterior_sha256, 'hex') || registro_canonico
        ), 'hex') = huella_auditoria_sha256
    )
);

CREATE TABLE vec_bolsa_panel.auditoria_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultima_huella_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_actual_huella_valida CHECK (
        ultima_huella_sha256 ~ '^[0-9a-f]{64}$'
    )
);

INSERT INTO vec_bolsa_panel.auditoria_actual(
    control_id, ultima_secuencia, ultima_huella_sha256, actualizada_en
) VALUES (true, 0, repeat('0', 64), statement_timestamp());

CREATE TABLE vec_bolsa_panel.consulta_confirmada (
    decision_ref text PRIMARY KEY,
    correlacion_ref text NOT NULL UNIQUE,
    clase_ambito text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text NOT NULL DEFAULT '',
    revision text NOT NULL,
    operacion_huella_sha256 text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_motivo_sha256 text NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    lectura_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    auditoria_secuencia bigint NOT NULL UNIQUE,
    panel_canonico bytea NOT NULL,
    panel_huella_sha256 text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (clase_ambito, organizacion_ref, unidad_gestion_ref, revision)
        REFERENCES vec_bolsa_panel.proyeccion_panel(
            clase_ambito, organizacion_ref, unidad_gestion_ref, revision
        ),
    FOREIGN KEY (decision_ref, atestacion_ref, atestacion_version)
        REFERENCES vec_bolsa_panel.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version
        ),
    FOREIGN KEY (auditoria_ref)
        REFERENCES vec_bolsa_panel.auditoria(auditoria_ref),
    CONSTRAINT consulta_selector_exacto CHECK (
        vec_bolsa_panel.referencia_opaca_valida(
            organizacion_ref, 'org_'
        ) IS TRUE
        AND (
            (clase_ambito = 'organizacion' AND unidad_gestion_ref = '')
            OR
            (clase_ambito = 'unidad_gestion'
             AND vec_bolsa_panel.referencia_opaca_valida(
                 unidad_gestion_ref, 'uni_'
             ) IS TRUE)
        )
    ),
    CONSTRAINT consulta_referencias_validas CHECK (
        correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
        AND vec_bolsa_panel.referencia_opaca_valida(
            revision, 'rev_'
        ) IS TRUE
        AND vec_bolsa_panel.referencia_opaca_valida(
            lectura_ref, 'lec_'
        ) IS TRUE
        AND vec_bolsa_panel.referencia_opaca_valida(
            auditoria_ref, 'aud_'
        ) IS TRUE
    ),
    CONSTRAINT consulta_huellas_validas CHECK (
        operacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_decision_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_motivo_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_atestacion_sha256 ~ '^[0-9a-f]{64}$'
        AND panel_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND octet_length(panel_canonico) BETWEEN 2 AND 2097152
        AND encode(sha256(panel_canonico), 'hex') = panel_huella_sha256
    )
);

CREATE FUNCTION vec_bolsa_panel.validar_avance_proyeccion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.clase_ambito IS DISTINCT FROM OLD.clase_ambito
       OR NEW.organizacion_ref IS DISTINCT FROM OLD.organizacion_ref
       OR NEW.unidad_gestion_ref IS DISTINCT FROM OLD.unidad_gestion_ref
       OR NEW.revision IS NOT DISTINCT FROM OLD.revision
       OR NEW.actualizada_en <= OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de proyeccion no monotono';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.validar_avance_atestacion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.decision_ref IS DISTINCT FROM OLD.decision_ref
       OR NEW.version <> OLD.version + 1
       OR NEW.actualizada_en <= OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de atestacion invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_panel.validar_avance_auditoria_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.control_id IS DISTINCT FROM OLD.control_id
       OR NEW.ultima_secuencia <> OLD.ultima_secuencia + 1
       OR NEW.actualizada_en < OLD.actualizada_en
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_panel.auditoria AS registro
            WHERE registro.secuencia = NEW.ultima_secuencia
              AND registro.huella_anterior_sha256 = OLD.ultima_huella_sha256
              AND registro.huella_auditoria_sha256 = NEW.ultima_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de cadena de auditoria invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER proyeccion_actual_avance
    BEFORE UPDATE ON vec_bolsa_panel.proyeccion_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_panel.validar_avance_proyeccion_actual();
CREATE TRIGGER atestacion_actual_avance
    BEFORE UPDATE ON vec_bolsa_panel.atestacion_autorizacion_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_panel.validar_avance_atestacion_actual();
CREATE TRIGGER auditoria_actual_avance
    BEFORE UPDATE ON vec_bolsa_panel.auditoria_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_panel.validar_avance_auditoria_actual();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'proyeccion_panel', 'convocatoria_resumen', 'actuacion_pendiente',
        'atestacion_autorizacion_version', 'auditoria',
        'consulta_confirmada'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_panel.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_panel.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_panel.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_panel.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'proyeccion_actual', 'atestacion_autorizacion_actual',
        'auditoria_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_bolsa_panel.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_panel.rechazar_mutacion_inmutable()',
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
        'proyeccion_panel', 'proyeccion_actual', 'convocatoria_resumen',
        'actuacion_pendiente', 'atestacion_autorizacion_version',
        'atestacion_autorizacion_actual', 'auditoria', 'auditoria_actual',
        'consulta_confirmada'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_panel.%I ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_panel.%I FORCE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_panel.%I FOR ALL TO vec_bolsa_panel_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_panel_propietario',
            'vec_bolsa_panel_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_panel FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_panel FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_panel FROM PUBLIC;

DO $cerrar_tipos_implicitos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_bolsa_panel'
           AND definicion.typelem = 0 AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I, %I',
            tipo.nspname, tipo.typname,
            'vec_bolsa_panel_proyector',
            'vec_bolsa_panel_ejecutor_consulta',
            'vec_bolsa_panel_registrador_atestacion'
        );
    END LOOP;
END
$cerrar_tipos_implicitos$;
COMMIT;
