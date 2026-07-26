-- Proyección minimizada de los motivos funcionales de cobertura. No es el
-- catálogo maestro: solo materializa, de forma síncrona, coordenadas publicadas
-- por el gobierno VEC. Sin un maestro PostgreSQL gobernado que invoque estas
-- funciones en su misma transacción, Contratación Temporal debe permanecer no
-- preparada y fallar cerrado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:motivo-cobertura-v1:000002', 0
    )
);

DO $prevalidacion$
DECLARE
    v_rol text;
BEGIN
    IF pg_catalog.to_regnamespace('vec_autorizacion') IS NULL
       OR pg_catalog.pg_get_userbyid(
           (SELECT nspowner FROM pg_catalog.pg_namespace
             WHERE nspname = 'vec_autorizacion')
       ) <> 'vec_autorizacion_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta la autoridad VEC esperada';
    END IF;
    FOREACH v_rol IN ARRAY ARRAY[
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion_motivos_evaluador',
        'vec_contratacion_temporal_propietario'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = v_rol AND NOT rolcanlogin
               AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb
               AND NOT rolreplication AND NOT rolbypassrls
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'falta un rol cerrado requerido',
                DETAIL = v_rol;
        END IF;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_class c
        JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'vec_autorizacion'
          AND c.relname LIKE 'motivo_cobertura_v1_%'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'vec_autorizacion'
          AND (
              p.proname LIKE 'motivo_cobertura_v1_%'
              OR p.proname IN (
                  'publicar_motivos_cobertura_v1',
                  'retirar_motivos_cobertura_v1',
                  'resolver_motivo_cobertura_historico_v1',
                  'resolver_motivo_cobertura_actual_v1'
              )
          )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar motivos de cobertura';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_autorizacion.motivo_cobertura_v1_evento_origen (
    secuencia_origen bigint PRIMARY KEY,
    evento_origen_ref text NOT NULL UNIQUE,
    tipo_evento text NOT NULL,
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    huella_evento_sha256 text NOT NULL,
    registrado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    registrado_txid bigint NOT NULL DEFAULT pg_catalog.txid_current(),
    CONSTRAINT motivo_cobertura_v1_evento_coordenadas
        UNIQUE (secuencia_origen, evento_origen_ref),
    CONSTRAINT motivo_cobertura_v1_evento_secuencia
        CHECK (secuencia_origen BETWEEN 1 AND 9007199254740991),
    CONSTRAINT motivo_cobertura_v1_evento_ref
        CHECK (evento_origen_ref ~ '^evento_[0-9a-f]{32}$'),
    CONSTRAINT motivo_cobertura_v1_evento_tipo
        CHECK (tipo_evento IN ('publicacion', 'retirada')),
    CONSTRAINT motivo_cobertura_v1_evento_catalogo
        CHECK (catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CONSTRAINT motivo_cobertura_v1_evento_version
        CHECK (catalogo_version > 0),
    CONSTRAINT motivo_cobertura_v1_evento_huella
        CHECK (huella_evento_sha256 ~ '^[0-9a-f]{64}$'
               AND huella_evento_sha256 <> pg_catalog.repeat('0', 64)),
    CONSTRAINT motivo_cobertura_v1_evento_instante
        CHECK (pg_catalog.isfinite(registrado_en)
               AND pg_catalog.date_part(
                   'year', (registrado_en AT TIME ZONE 'UTC')
               ) BETWEEN 1 AND 9999)
);

CREATE TABLE vec_autorizacion.motivo_cobertura_v1_catalogo_publicado (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    modulo_id text NOT NULL,
    catalogo_huella_publicada_sha256 text NOT NULL,
    publicado_en timestamptz(6) NOT NULL,
    evento_origen_ref text NOT NULL UNIQUE,
    secuencia_origen bigint NOT NULL UNIQUE,
    PRIMARY KEY (catalogo_id, catalogo_version),
    CONSTRAINT motivo_cobertura_v1_catalogo_id
        CHECK (catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CONSTRAINT motivo_cobertura_v1_catalogo_version
        CHECK (catalogo_version > 0),
    CONSTRAINT motivo_cobertura_v1_catalogo_modulo
        CHECK (modulo_id = 'contratacion_temporal'),
    CONSTRAINT motivo_cobertura_v1_catalogo_huella
        CHECK (catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
               AND catalogo_huella_publicada_sha256 <>
                   pg_catalog.repeat('0', 64)),
    CONSTRAINT motivo_cobertura_v1_catalogo_instante
        CHECK (pg_catalog.isfinite(publicado_en)
               AND pg_catalog.date_part(
                   'year', (publicado_en AT TIME ZONE 'UTC')
               ) BETWEEN 1 AND 9999),
    CONSTRAINT motivo_cobertura_v1_catalogo_evento_fk
        FOREIGN KEY (secuencia_origen, evento_origen_ref)
        REFERENCES vec_autorizacion.motivo_cobertura_v1_evento_origen
            (secuencia_origen, evento_origen_ref)
);

CREATE TABLE vec_autorizacion.motivo_cobertura_v1_entrada (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    entrada_clave text NOT NULL,
    clave_i18n text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    PRIMARY KEY (catalogo_id, catalogo_version, entrada_clave),
    CONSTRAINT motivo_cobertura_v1_entrada_catalogo_fk
        FOREIGN KEY (catalogo_id, catalogo_version)
        REFERENCES vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
            (catalogo_id, catalogo_version),
    CONSTRAINT motivo_cobertura_v1_entrada_clave
        CHECK (entrada_clave ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CONSTRAINT motivo_cobertura_v1_entrada_i18n
        CHECK (clave_i18n ~ '^[a-z][a-z0-9._-]{1,79}$'),
    CONSTRAINT motivo_cobertura_v1_entrada_instantes
        CHECK (pg_catalog.isfinite(vigente_desde)
               AND pg_catalog.date_part(
                   'year', (vigente_desde AT TIME ZONE 'UTC')
               ) BETWEEN 1 AND 9999
               AND (
                   vigente_hasta IS NULL
                   OR (
                       pg_catalog.isfinite(vigente_hasta)
                       AND pg_catalog.date_part(
                           'year', (vigente_hasta AT TIME ZONE 'UTC')
                       ) BETWEEN 1 AND 9999
                   )
               )),
    CONSTRAINT motivo_cobertura_v1_entrada_intervalo
        CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);

CREATE TABLE vec_autorizacion.motivo_cobertura_v1_retirada (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    catalogo_huella_retirada_sha256 text NOT NULL,
    retirado_en timestamptz(6) NOT NULL,
    evento_origen_ref text NOT NULL UNIQUE,
    secuencia_origen bigint NOT NULL UNIQUE,
    PRIMARY KEY (catalogo_id, catalogo_version),
    CONSTRAINT motivo_cobertura_v1_retirada_catalogo_fk
        FOREIGN KEY (catalogo_id, catalogo_version)
        REFERENCES vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
            (catalogo_id, catalogo_version),
    CONSTRAINT motivo_cobertura_v1_retirada_huella
        CHECK (catalogo_huella_retirada_sha256 ~ '^[0-9a-f]{64}$'
               AND catalogo_huella_retirada_sha256 <>
                   pg_catalog.repeat('0', 64)),
    CONSTRAINT motivo_cobertura_v1_retirada_instante
        CHECK (pg_catalog.isfinite(retirado_en)
               AND pg_catalog.date_part(
                   'year', (retirado_en AT TIME ZONE 'UTC')
               ) BETWEEN 1 AND 9999),
    CONSTRAINT motivo_cobertura_v1_retirada_evento_fk
        FOREIGN KEY (secuencia_origen, evento_origen_ref)
        REFERENCES vec_autorizacion.motivo_cobertura_v1_evento_origen
            (secuencia_origen, evento_origen_ref)
);

CREATE TABLE vec_autorizacion.motivo_cobertura_v1_checkpoint_origen (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL
        CHECK (ultima_secuencia BETWEEN 0 AND 9007199254740991),
    ultimo_evento_ref text,
    ultima_huella_evento_sha256 text,
    actualizado_en timestamptz(6) NOT NULL,
    CONSTRAINT motivo_cobertura_v1_checkpoint_completo CHECK (
        (
            ultima_secuencia = 0
            AND ultimo_evento_ref IS NULL
            AND ultima_huella_evento_sha256 IS NULL
        ) OR (
            ultima_secuencia > 0
            AND ultimo_evento_ref ~ '^evento_[0-9a-f]{32}$'
            AND ultima_huella_evento_sha256 ~ '^[0-9a-f]{64}$'
            AND ultima_huella_evento_sha256 <>
                pg_catalog.repeat('0', 64)
        )
    ),
    CONSTRAINT motivo_cobertura_v1_checkpoint_instante
        CHECK (pg_catalog.isfinite(actualizado_en)
               AND pg_catalog.date_part(
                   'year', (actualizado_en AT TIME ZONE 'UTC')
               ) BETWEEN 1 AND 9999)
);

INSERT INTO vec_autorizacion.motivo_cobertura_v1_checkpoint_origen (
    control_id, ultima_secuencia, actualizado_en
) VALUES (true, 0, pg_catalog.clock_timestamp());

CREATE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'la proyeccion de motivos de cobertura es inmutable';
END;
$funcion$;

CREATE FUNCTION vec_autorizacion.motivo_cobertura_v1_validar_checkpoint()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.control_id IS DISTINCT FROM OLD.control_id
       OR NEW.ultima_secuencia IS DISTINCT FROM OLD.ultima_secuencia + 1
       OR NEW.ultimo_evento_ref IS NULL
       OR NEW.ultima_huella_evento_sha256 IS NULL
       OR NEW.actualizado_en < OLD.actualizado_en
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_cobertura_v1_evento_origen e
            WHERE e.secuencia_origen = NEW.ultima_secuencia
              AND e.evento_origen_ref = NEW.ultimo_evento_ref
              AND e.huella_evento_sha256 =
                  NEW.ultima_huella_evento_sha256
              AND e.registrado_txid = pg_catalog.txid_current()
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance no atomico del checkpoint de cobertura';
    END IF;
    RETURN NEW;
END;
$funcion$;

DO $triggers$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'motivo_cobertura_v1_evento_origen',
        'motivo_cobertura_v1_catalogo_publicado',
        'motivo_cobertura_v1_entrada',
        'motivo_cobertura_v1_retirada'
    ] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_mutacion BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_truncado BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable()',
            v_tabla
        );
    END LOOP;
END
$triggers$;

CREATE TRIGGER validar_avance_checkpoint
    BEFORE UPDATE
    ON vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
    FOR EACH ROW
    EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_validar_checkpoint();
CREATE TRIGGER bloquear_insercion_checkpoint
    BEFORE INSERT
    ON vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
    FOR EACH ROW
    EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable();
CREATE TRIGGER bloquear_borrado_checkpoint
    BEFORE DELETE
    ON vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
    FOR EACH ROW
    EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable();
CREATE TRIGGER bloquear_truncado_checkpoint
    BEFORE TRUNCATE
    ON vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
    FOR EACH STATEMENT
    EXECUTE FUNCTION vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable();

CREATE FUNCTION vec_autorizacion.motivo_cobertura_v1_instante_canonico(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_instante timestamptz;
BEGIN
    IF p_valor IS NULL
       OR p_valor !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    v_instante := p_valor::timestamptz;
    RETURN pg_catalog.isfinite(v_instante)
       AND pg_catalog.date_part(
           'year', (v_instante AT TIME ZONE 'UTC')
       ) BETWEEN 1 AND 9999
       AND pg_catalog.to_char(
           v_instante AT TIME ZONE 'UTC',
           'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
       ) = p_valor;
EXCEPTION
    WHEN datetime_field_overflow OR invalid_datetime_format THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.motivo_cobertura_v1_entradas_validas(
    p_entradas jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_entrada jsonb;
    v_desde timestamptz;
    v_hasta timestamptz;
BEGIN
    IF p_entradas IS NULL
       OR pg_catalog.jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
       OR pg_catalog.pg_column_size(p_entradas) > 16 * 1024 * 1024
       OR pg_catalog.jsonb_array_length(p_entradas) NOT BETWEEN 1 AND 10000 THEN
        RETURN false;
    END IF;
    FOR v_entrada IN
        SELECT value FROM pg_catalog.jsonb_array_elements(p_entradas)
    LOOP
        IF pg_catalog.jsonb_typeof(v_entrada) IS DISTINCT FROM 'object'
           OR (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.jsonb_object_keys(v_entrada)
           ) <> 4
           OR NOT (v_entrada ?& ARRAY[
               'clave', 'clave_i18n', 'vigente_desde', 'vigente_hasta'
           ])
           OR pg_catalog.jsonb_typeof(v_entrada -> 'clave') <>
              'string'
           OR v_entrada ->> 'clave' !~ '^[a-z][a-z0-9._-]{0,127}$'
           OR pg_catalog.jsonb_typeof(v_entrada -> 'clave_i18n') <>
              'string'
           OR v_entrada ->> 'clave_i18n' !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR pg_catalog.jsonb_typeof(v_entrada -> 'vigente_desde') <>
              'string'
           OR vec_autorizacion.motivo_cobertura_v1_instante_canonico(
               v_entrada ->> 'vigente_desde'
           ) IS NOT TRUE
           OR (
               pg_catalog.jsonb_typeof(v_entrada -> 'vigente_hasta') <>
                   'null'
               AND (
                   pg_catalog.jsonb_typeof(
                       v_entrada -> 'vigente_hasta'
                   ) <> 'string'
                   OR vec_autorizacion.motivo_cobertura_v1_instante_canonico(
                       v_entrada ->> 'vigente_hasta'
                   ) IS NOT TRUE
               )
           ) THEN
            RETURN false;
        END IF;
        v_desde := (v_entrada ->> 'vigente_desde')::timestamptz;
        v_hasta := CASE
            WHEN pg_catalog.jsonb_typeof(
                v_entrada -> 'vigente_hasta'
            ) = 'null' THEN NULL
            ELSE (v_entrada ->> 'vigente_hasta')::timestamptz
        END;
        IF v_hasta IS NOT NULL AND v_hasta <= v_desde THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN NOT EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(p_entradas) e
         GROUP BY e ->> 'clave'
        HAVING pg_catalog.count(*) > 1
    );
END
$funcion$;

CREATE FUNCTION vec_autorizacion.publicar_motivos_cobertura_v1(
    p_evento_origen_ref text,
    p_secuencia_origen bigint,
    p_huella_evento_sha256 text,
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_modulo_id text,
    p_publicado_en timestamptz,
    p_entradas jsonb
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_checkpoint record;
    v_evento record;
    v_catalogo record;
    v_coincidencias integer;
    v_entradas jsonb;
    v_existentes jsonb;
BEGIN
    IF (p_evento_origen_ref ~ '^evento_[0-9a-f]{32}$') IS NOT TRUE
       OR p_secuencia_origen IS NULL
       OR p_secuencia_origen NOT BETWEEN 1 AND 9007199254740991
       OR (p_huella_evento_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_huella_evento_sha256 = pg_catalog.repeat('0', 64)
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL OR p_catalogo_version < 1
       OR (
           p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
       ) IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 =
          pg_catalog.repeat('0', 64)
       OR p_modulo_id IS DISTINCT FROM 'contratacion_temporal'
       OR p_publicado_en IS NULL
       OR pg_catalog.isfinite(p_publicado_en) IS NOT TRUE
       OR pg_catalog.date_trunc('microseconds', p_publicado_en) <>
          p_publicado_en
       OR pg_catalog.date_part(
           'year', (p_publicado_en AT TIME ZONE 'UTC')
       ) NOT BETWEEN 1 AND 9999
       OR p_publicado_en > pg_catalog.clock_timestamp()
       OR vec_autorizacion.motivo_cobertura_v1_entradas_validas(
           p_entradas
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    SELECT pg_catalog.jsonb_agg(e ORDER BY e ->> 'clave')
      INTO v_entradas
      FROM pg_catalog.jsonb_array_elements(p_entradas) e;
    SELECT ultima_secuencia
      INTO STRICT v_checkpoint
      FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
     WHERE control_id
     FOR UPDATE;
    SELECT pg_catalog.count(*)
      INTO v_coincidencias
      FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
     WHERE secuencia_origen = p_secuencia_origen
        OR evento_origen_ref = p_evento_origen_ref;
    IF v_coincidencias > 0 THEN
        IF v_coincidencias <> 1 THEN RETURN false; END IF;
        SELECT * INTO STRICT v_evento
          FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
         WHERE secuencia_origen = p_secuencia_origen
            OR evento_origen_ref = p_evento_origen_ref;
        SELECT * INTO v_catalogo
          FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        SELECT pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_object(
                'clave', entrada_clave,
                'clave_i18n', clave_i18n,
                'vigente_desde', pg_catalog.to_char(
                    vigente_desde AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                ),
                'vigente_hasta', CASE
                    WHEN vigente_hasta IS NULL THEN 'null'::jsonb
                    ELSE pg_catalog.to_jsonb(pg_catalog.to_char(
                        vigente_hasta AT TIME ZONE 'UTC',
                        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                    ))
                END
            ) ORDER BY entrada_clave
        ) INTO v_existentes
          FROM vec_autorizacion.motivo_cobertura_v1_entrada
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        RETURN v_evento.tipo_evento = 'publicacion'
           AND v_evento.secuencia_origen = p_secuencia_origen
           AND v_evento.evento_origen_ref = p_evento_origen_ref
           AND v_evento.catalogo_id = p_catalogo_id
           AND v_evento.catalogo_version = p_catalogo_version
           AND v_evento.huella_evento_sha256 =
               p_huella_evento_sha256
           AND v_catalogo.catalogo_id IS NOT NULL
           AND v_catalogo.modulo_id = p_modulo_id
           AND v_catalogo.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND v_catalogo.publicado_en = p_publicado_en
           AND v_catalogo.evento_origen_ref = p_evento_origen_ref
           AND v_catalogo.secuencia_origen = p_secuencia_origen
           AND v_existentes IS NOT DISTINCT FROM v_entradas;
    END IF;
    IF p_secuencia_origen <> v_checkpoint.ultima_secuencia + 1
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
            WHERE catalogo_id = p_catalogo_id
              AND catalogo_version = p_catalogo_version
       )
       OR (
           p_catalogo_version > 1
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
                WHERE catalogo_id = p_catalogo_id
                  AND catalogo_version = p_catalogo_version - 1
                  AND publicado_en <= p_publicado_en
           )
       ) THEN
        RETURN false;
    END IF;
    INSERT INTO vec_autorizacion.motivo_cobertura_v1_evento_origen (
        secuencia_origen, evento_origen_ref, tipo_evento, catalogo_id,
        catalogo_version, huella_evento_sha256
    ) VALUES (
        p_secuencia_origen, p_evento_origen_ref, 'publicacion',
        p_catalogo_id, p_catalogo_version, p_huella_evento_sha256
    );
    INSERT INTO vec_autorizacion.motivo_cobertura_v1_catalogo_publicado (
        catalogo_id, catalogo_version, modulo_id,
        catalogo_huella_publicada_sha256, publicado_en,
        evento_origen_ref, secuencia_origen
    ) VALUES (
        p_catalogo_id, p_catalogo_version, p_modulo_id,
        p_catalogo_huella_publicada_sha256, p_publicado_en,
        p_evento_origen_ref, p_secuencia_origen
    );
    INSERT INTO vec_autorizacion.motivo_cobertura_v1_entrada (
        catalogo_id, catalogo_version, entrada_clave, clave_i18n,
        vigente_desde, vigente_hasta
    )
    SELECT
        p_catalogo_id, p_catalogo_version, e ->> 'clave',
        e ->> 'clave_i18n',
        (e ->> 'vigente_desde')::timestamptz,
        CASE
            WHEN pg_catalog.jsonb_typeof(e -> 'vigente_hasta') = 'null'
                THEN NULL
            ELSE (e ->> 'vigente_hasta')::timestamptz
        END
      FROM pg_catalog.jsonb_array_elements(p_entradas) e;
    UPDATE vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
       SET ultima_secuencia = p_secuencia_origen,
           ultimo_evento_ref = p_evento_origen_ref,
           ultima_huella_evento_sha256 = p_huella_evento_sha256,
           actualizado_en = pg_catalog.clock_timestamp()
     WHERE control_id;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.retirar_motivos_cobertura_v1(
    p_evento_origen_ref text,
    p_secuencia_origen bigint,
    p_huella_evento_sha256 text,
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_catalogo_huella_retirada_sha256 text,
    p_modulo_id text,
    p_retirado_en timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_checkpoint record;
    v_evento record;
    v_catalogo record;
    v_retirada record;
    v_coincidencias integer;
BEGIN
    IF (p_evento_origen_ref ~ '^evento_[0-9a-f]{32}$') IS NOT TRUE
       OR p_secuencia_origen IS NULL
       OR p_secuencia_origen NOT BETWEEN 1 AND 9007199254740991
       OR (p_huella_evento_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_huella_evento_sha256 = pg_catalog.repeat('0', 64)
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL OR p_catalogo_version < 1
       OR (
           p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
       ) IS NOT TRUE
       OR (
           p_catalogo_huella_retirada_sha256 ~ '^[0-9a-f]{64}$'
       ) IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 =
          pg_catalog.repeat('0', 64)
       OR p_catalogo_huella_retirada_sha256 =
          pg_catalog.repeat('0', 64)
       OR p_catalogo_huella_retirada_sha256 =
          p_catalogo_huella_publicada_sha256
       OR p_modulo_id IS DISTINCT FROM 'contratacion_temporal'
       OR p_retirado_en IS NULL
       OR pg_catalog.isfinite(p_retirado_en) IS NOT TRUE
       OR pg_catalog.date_trunc('microseconds', p_retirado_en) <>
          p_retirado_en
       OR pg_catalog.date_part(
           'year', (p_retirado_en AT TIME ZONE 'UTC')
       ) NOT BETWEEN 1 AND 9999
       OR p_retirado_en > pg_catalog.clock_timestamp() THEN
        RETURN false;
    END IF;
    SELECT ultima_secuencia
      INTO STRICT v_checkpoint
      FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
     WHERE control_id
     FOR UPDATE;
    SELECT pg_catalog.count(*)
      INTO v_coincidencias
      FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
     WHERE secuencia_origen = p_secuencia_origen
        OR evento_origen_ref = p_evento_origen_ref;
    IF v_coincidencias > 0 THEN
        IF v_coincidencias <> 1 THEN RETURN false; END IF;
        SELECT * INTO STRICT v_evento
          FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
         WHERE secuencia_origen = p_secuencia_origen
            OR evento_origen_ref = p_evento_origen_ref;
        SELECT * INTO v_catalogo
          FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        SELECT * INTO v_retirada
          FROM vec_autorizacion.motivo_cobertura_v1_retirada
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        RETURN v_evento.tipo_evento = 'retirada'
           AND v_evento.secuencia_origen = p_secuencia_origen
           AND v_evento.evento_origen_ref = p_evento_origen_ref
           AND v_evento.catalogo_id = p_catalogo_id
           AND v_evento.catalogo_version = p_catalogo_version
           AND v_evento.huella_evento_sha256 =
               p_huella_evento_sha256
           AND v_catalogo.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND v_catalogo.modulo_id = p_modulo_id
           AND v_retirada.catalogo_id IS NOT NULL
           AND v_retirada.catalogo_huella_retirada_sha256 =
               p_catalogo_huella_retirada_sha256
           AND v_retirada.retirado_en = p_retirado_en
           AND v_retirada.evento_origen_ref = p_evento_origen_ref
           AND v_retirada.secuencia_origen = p_secuencia_origen;
    END IF;
    SELECT * INTO v_catalogo
      FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
     WHERE catalogo_id = p_catalogo_id
       AND catalogo_version = p_catalogo_version;
    IF NOT FOUND
       OR v_catalogo.catalogo_huella_publicada_sha256 <>
          p_catalogo_huella_publicada_sha256
       OR v_catalogo.modulo_id <> p_modulo_id
       OR p_retirado_en < v_catalogo.publicado_en
       OR p_secuencia_origen <> v_checkpoint.ultima_secuencia + 1
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_cobertura_v1_retirada
            WHERE catalogo_id = p_catalogo_id
              AND catalogo_version = p_catalogo_version
       ) THEN
        RETURN false;
    END IF;
    INSERT INTO vec_autorizacion.motivo_cobertura_v1_evento_origen (
        secuencia_origen, evento_origen_ref, tipo_evento, catalogo_id,
        catalogo_version, huella_evento_sha256
    ) VALUES (
        p_secuencia_origen, p_evento_origen_ref, 'retirada',
        p_catalogo_id, p_catalogo_version, p_huella_evento_sha256
    );
    INSERT INTO vec_autorizacion.motivo_cobertura_v1_retirada (
        catalogo_id, catalogo_version,
        catalogo_huella_retirada_sha256, retirado_en,
        evento_origen_ref, secuencia_origen
    ) VALUES (
        p_catalogo_id, p_catalogo_version,
        p_catalogo_huella_retirada_sha256, p_retirado_en,
        p_evento_origen_ref, p_secuencia_origen
    );
    UPDATE vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
       SET ultima_secuencia = p_secuencia_origen,
           ultimo_evento_ref = p_evento_origen_ref,
           ultima_huella_evento_sha256 = p_huella_evento_sha256,
           actualizado_en = pg_catalog.clock_timestamp()
     WHERE control_id;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'motivo_cobertura_v1_evento_origen',
        'motivo_cobertura_v1_catalogo_publicado',
        'motivo_cobertura_v1_entrada',
        'motivo_cobertura_v1_retirada',
        'motivo_cobertura_v1_checkpoint_origen'
    ] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            v_tabla, 'vec_autorizacion_propietario',
            'vec_autorizacion_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON TABLE
    vec_autorizacion.motivo_cobertura_v1_evento_origen,
    vec_autorizacion.motivo_cobertura_v1_catalogo_publicado,
    vec_autorizacion.motivo_cobertura_v1_entrada,
    vec_autorizacion.motivo_cobertura_v1_retirada,
    vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
FROM PUBLIC, vec_autorizacion_motivos_proyector,
     vec_autorizacion_motivos_evaluador,
     vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
    vec_autorizacion.motivo_cobertura_v1_bloquear_inmutable(),
    vec_autorizacion.motivo_cobertura_v1_validar_checkpoint(),
    vec_autorizacion.motivo_cobertura_v1_instante_canonico(text),
    vec_autorizacion.motivo_cobertura_v1_entradas_validas(jsonb),
    vec_autorizacion.publicar_motivos_cobertura_v1(
        text, bigint, text, text, integer, text, text, timestamptz, jsonb
    ),
    vec_autorizacion.retirar_motivos_cobertura_v1(
        text, bigint, text, text, integer, text, text, text, timestamptz
    )
FROM PUBLIC, vec_autorizacion_motivos_proyector,
     vec_autorizacion_motivos_evaluador,
     vec_contratacion_temporal_propietario;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_motivos_proyector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, timestamptz, jsonb
) TO vec_autorizacion_motivos_proyector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.retirar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, text, timestamptz
) TO vec_autorizacion_motivos_proyector;

COMMENT ON FUNCTION vec_autorizacion.publicar_motivos_cobertura_v1(
    text, bigint, text, text, integer, text, text, timestamptz, jsonb
) IS
    'Proyecta una publicación ya gobernada; no crea ni sustituye al catálogo maestro PostgreSQL.';

COMMIT;
