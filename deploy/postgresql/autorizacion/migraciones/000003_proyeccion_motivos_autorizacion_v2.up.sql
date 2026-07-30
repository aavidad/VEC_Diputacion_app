BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:migracion:motivos_v2:000003', 0)
);

DO $prevalidacion$
DECLARE
    rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_autorizacion'
           AND nspowner = 'vec_autorizacion_propietario'::regrole
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000003 requiere el esquema V1 con su propietario exacto';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion_motivos_evaluador'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_roles
             WHERE rolname = rol
               AND NOT rolcanlogin
               AND NOT rolsuper
               AND NOT rolcreaterole
               AND NOT rolcreatedb
               AND NOT rolinherit
               AND NOT rolreplication
               AND NOT rolbypassrls
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = '000003 requiere roles V2 NOLOGIN con atributos cerrados';
        END IF;
        IF pg_catalog.pg_has_role(
            rol,
            'vec_autorizacion_propietario',
            'MEMBER'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'un rol V2 no puede pertenecer al propietario';
        END IF;
    END LOOP;
END
$prevalidacion$;

SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.motivo_v2_evento_origen (
    secuencia_origen bigint PRIMARY KEY,
    evento_origen_ref text NOT NULL UNIQUE,
    tipo_evento text NOT NULL,
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    huella_evento_sha256 text NOT NULL,
    registrado_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    registrado_txid bigint NOT NULL DEFAULT txid_current(),
    CONSTRAINT motivo_v2_evento_coordenadas_unicas
        UNIQUE (secuencia_origen, evento_origen_ref),
    CONSTRAINT motivo_v2_evento_secuencia_positiva
        CHECK (secuencia_origen > 0),
    CONSTRAINT motivo_v2_evento_ref_opaca
        CHECK (evento_origen_ref ~ '^evento_[0-9a-f]{32}$'),
    CONSTRAINT motivo_v2_evento_tipo_cerrado
        CHECK (tipo_evento IN ('publicacion', 'retirada')),
    CONSTRAINT motivo_v2_evento_catalogo_canonico
        CHECK (catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CONSTRAINT motivo_v2_evento_version_positiva
        CHECK (catalogo_version > 0),
    CONSTRAINT motivo_v2_evento_huella_valida
        CHECK (huella_evento_sha256 ~ '^[0-9a-f]{64}$'
               AND huella_evento_sha256 <> repeat('0', 64)),
    CONSTRAINT motivo_v2_evento_registro_finito
        CHECK (isfinite(registrado_en)
               AND extract(year FROM (registrado_en AT TIME ZONE 'UTC'))
                   BETWEEN 1 AND 9999)
);

CREATE TABLE vec_autorizacion.motivo_v2_catalogo_publicado (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    catalogo_huella_publicada_sha256 text NOT NULL,
    publicado_en timestamptz(6) NOT NULL,
    evento_origen_ref text NOT NULL UNIQUE,
    secuencia_origen bigint NOT NULL UNIQUE,
    PRIMARY KEY (catalogo_id, catalogo_version),
    CONSTRAINT motivo_v2_catalogo_id_canonico
        CHECK (catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$'),
    CONSTRAINT motivo_v2_catalogo_version_positiva
        CHECK (catalogo_version > 0),
    CONSTRAINT motivo_v2_catalogo_huella_publicada_valida
        CHECK (catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
               AND catalogo_huella_publicada_sha256 <> repeat('0', 64)),
    CONSTRAINT motivo_v2_catalogo_publicacion_finita
        CHECK (isfinite(publicado_en)
               AND extract(year FROM (publicado_en AT TIME ZONE 'UTC'))
                   BETWEEN 1 AND 9999),
    CONSTRAINT motivo_v2_catalogo_evento_fk
        FOREIGN KEY (secuencia_origen, evento_origen_ref)
        REFERENCES vec_autorizacion.motivo_v2_evento_origen
            (secuencia_origen, evento_origen_ref)
);

CREATE TABLE vec_autorizacion.motivo_v2_entrada (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    entrada_clave text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    PRIMARY KEY (catalogo_id, catalogo_version, entrada_clave),
    CONSTRAINT motivo_v2_entrada_catalogo_fk
        FOREIGN KEY (catalogo_id, catalogo_version)
        REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado
            (catalogo_id, catalogo_version),
    CONSTRAINT motivo_v2_entrada_clave_opaca
        CHECK (entrada_clave ~ '^motivo_[0-9a-f]{32}$'),
    CONSTRAINT motivo_v2_entrada_instantes_finitos
        CHECK (isfinite(vigente_desde)
               AND extract(year FROM (vigente_desde AT TIME ZONE 'UTC'))
                   BETWEEN 1 AND 9999
               AND (vigente_hasta IS NULL
                    OR (isfinite(vigente_hasta)
                        AND extract(year FROM (
                            vigente_hasta AT TIME ZONE 'UTC'
                        )) BETWEEN 1 AND 9999))),
    CONSTRAINT motivo_v2_entrada_intervalo_valido
        CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);

CREATE TABLE vec_autorizacion.motivo_v2_retirada (
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    catalogo_huella_retirada_sha256 text NOT NULL,
    retirado_en timestamptz(6) NOT NULL,
    evento_origen_ref text NOT NULL UNIQUE,
    secuencia_origen bigint NOT NULL UNIQUE,
    PRIMARY KEY (catalogo_id, catalogo_version),
    CONSTRAINT motivo_v2_retirada_catalogo_fk
        FOREIGN KEY (catalogo_id, catalogo_version)
        REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado
            (catalogo_id, catalogo_version),
    CONSTRAINT motivo_v2_retirada_huella_valida
        CHECK (catalogo_huella_retirada_sha256 ~ '^[0-9a-f]{64}$'
               AND catalogo_huella_retirada_sha256 <> repeat('0', 64)),
    CONSTRAINT motivo_v2_retirada_instante_finito
        CHECK (isfinite(retirado_en)
               AND extract(year FROM (retirado_en AT TIME ZONE 'UTC'))
                   BETWEEN 1 AND 9999),
    CONSTRAINT motivo_v2_retirada_evento_fk
        FOREIGN KEY (secuencia_origen, evento_origen_ref)
        REFERENCES vec_autorizacion.motivo_v2_evento_origen
            (secuencia_origen, evento_origen_ref)
);

CREATE TABLE vec_autorizacion.motivo_v2_checkpoint_origen (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultimo_evento_ref text,
    ultima_huella_evento_sha256 text,
    actualizado_en timestamptz(6) NOT NULL,
    CONSTRAINT motivo_v2_checkpoint_evento_completo CHECK (
        (ultima_secuencia = 0
         AND ultimo_evento_ref IS NULL
         AND ultima_huella_evento_sha256 IS NULL)
        OR
        (ultima_secuencia > 0
         AND ultimo_evento_ref ~ '^evento_[0-9a-f]{32}$'
         AND ultima_huella_evento_sha256 ~ '^[0-9a-f]{64}$'
         AND ultima_huella_evento_sha256 <> repeat('0', 64))
    ),
    CONSTRAINT motivo_v2_checkpoint_actualizacion_finita
        CHECK (isfinite(actualizado_en)
               AND extract(year FROM (actualizado_en AT TIME ZONE 'UTC'))
                   BETWEEN 1 AND 9999)
);

INSERT INTO vec_autorizacion.motivo_v2_checkpoint_origen (
    control_id,
    ultima_secuencia,
    actualizado_en
) VALUES (true, 0, clock_timestamp());

CREATE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'la proyeccion historica de motivos V2 es inmutable';
END
$funcion$;

DO $bloque$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'motivo_v2_evento_origen',
        'motivo_v2_catalogo_publicado',
        'motivo_v2_entrada',
        'motivo_v2_retirada'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER bloquear_mutacion_fila BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable()',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER bloquear_truncado BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable()',
            tabla
        );
    END LOOP;
END
$bloque$;

CREATE FUNCTION vec_autorizacion.motivo_v2_validar_avance_checkpoint()
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
              FROM vec_autorizacion.motivo_v2_evento_origen AS evento
             WHERE evento.secuencia_origen = NEW.ultima_secuencia
               AND evento.evento_origen_ref = NEW.ultimo_evento_ref
               AND evento.huella_evento_sha256 = NEW.ultima_huella_evento_sha256
               AND evento.registrado_txid = txid_current()
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance no atomico o no monotono del checkpoint de motivos V2';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_avance_checkpoint
    BEFORE UPDATE ON vec_autorizacion.motivo_v2_checkpoint_origen
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.motivo_v2_validar_avance_checkpoint();
CREATE TRIGGER bloquear_insercion_checkpoint
    BEFORE INSERT ON vec_autorizacion.motivo_v2_checkpoint_origen
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable();
CREATE TRIGGER bloquear_borrado_checkpoint
    BEFORE DELETE ON vec_autorizacion.motivo_v2_checkpoint_origen
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable();
CREATE TRIGGER bloquear_truncado_checkpoint
    BEFORE TRUNCATE ON vec_autorizacion.motivo_v2_checkpoint_origen
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable();

CREATE FUNCTION vec_autorizacion.motivo_v2_instante_canonico_valido(p_valor text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz;
BEGIN
    IF p_valor IS NULL
       OR p_valor !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    instante := p_valor::timestamptz;
    IF isfinite(instante) IS NOT TRUE
       OR extract(year FROM (instante AT TIME ZONE 'UTC')) NOT BETWEEN 1 AND 9999 THEN
        RETURN false;
    END IF;
    -- PostgreSQL normaliza entradas como 24:00:00 o segundos 60. El
    -- round-trip UTC exacto rechaza esas formas aunque el cast las acepte.
    RETURN to_char(
        instante AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    ) = p_valor;
EXCEPTION WHEN datetime_field_overflow OR invalid_datetime_format THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.motivo_v2_entradas_validas(p_entradas jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    entrada jsonb;
    desde timestamptz;
    hasta timestamptz;
    numero_claves integer;
BEGIN
    -- El techo se comprueba antes de enumerar objetos. Coincide con el maximo
    -- del catalogo de dominio y limita trabajo/memoria en esta frontera
    -- SECURITY DEFINER incluso con JSON adversario.
    IF p_entradas IS NULL
       OR jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
       OR pg_column_size(p_entradas) > 16 * 1024 * 1024 THEN
        RETURN false;
    END IF;
    IF jsonb_array_length(p_entradas) NOT BETWEEN 1 AND 10000 THEN
        RETURN false;
    END IF;

    FOR entrada IN SELECT value FROM jsonb_array_elements(p_entradas) LOOP
        IF jsonb_typeof(entrada) IS DISTINCT FROM 'object' THEN
            RETURN false;
        END IF;
        SELECT count(*) INTO numero_claves FROM jsonb_object_keys(entrada);
        IF numero_claves <> 3
           OR jsonb_typeof(entrada -> 'clave') IS DISTINCT FROM 'string'
           OR (entrada ->> 'clave') !~ '^motivo_[0-9a-f]{32}$'
           OR jsonb_typeof(entrada -> 'vigente_desde') IS DISTINCT FROM 'string'
           OR vec_autorizacion.motivo_v2_instante_canonico_valido(
                entrada ->> 'vigente_desde'
              ) IS NOT TRUE
           OR (
                jsonb_typeof(entrada -> 'vigente_hasta') IS DISTINCT FROM 'null'
                AND (
                    jsonb_typeof(entrada -> 'vigente_hasta') IS DISTINCT FROM 'string'
                    OR vec_autorizacion.motivo_v2_instante_canonico_valido(
                        entrada ->> 'vigente_hasta'
                    ) IS NOT TRUE
                )
           ) THEN
            RETURN false;
        END IF;

        desde := (entrada ->> 'vigente_desde')::timestamptz;
        hasta := CASE
            WHEN jsonb_typeof(entrada -> 'vigente_hasta') = 'null' THEN NULL
            ELSE (entrada ->> 'vigente_hasta')::timestamptz
        END;
        IF hasta IS NOT NULL AND hasta <= desde THEN
            RETURN false;
        END IF;
    END LOOP;

    -- GROUP BY usa agregacion hash/ordenada acotada; evita el array creciente
    -- y la busqueda lineal O(n²) para catalogos de hasta 10.000 entradas.
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements(p_entradas) AS elemento
         GROUP BY elemento ->> 'clave'
        HAVING count(*) > 1
    ) THEN
        RETURN false;
    END IF;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.publicar_motivos_autorizacion_v2(
    p_evento_origen_ref text,
    p_secuencia_origen bigint,
    p_huella_evento_sha256 text,
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
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
    checkpoint record;
    evento_existente record;
    cabecera_existente record;
    entradas_canonicas jsonb;
    entradas_existentes jsonb;
BEGIN
    IF (p_evento_origen_ref ~ '^evento_[0-9a-f]{32}$') IS NOT TRUE
       OR p_secuencia_origen IS NULL
       OR p_secuencia_origen < 1
       OR (p_huella_evento_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_huella_evento_sha256 = repeat('0', 64)
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL
       OR p_catalogo_version < 1
       OR (p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 = repeat('0', 64) THEN
        RETURN false;
    END IF;
    IF p_publicado_en IS NULL OR isfinite(p_publicado_en) IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF extract(year FROM (p_publicado_en AT TIME ZONE 'UTC'))
           NOT BETWEEN 1 AND 9999
       OR p_publicado_en > clock_timestamp() THEN
        RETURN false;
    END IF;
    IF vec_autorizacion.motivo_v2_entradas_validas(p_entradas) IS NOT TRUE THEN
        RETURN false;
    END IF;

    SELECT jsonb_agg(entrada ORDER BY entrada ->> 'clave')
      INTO entradas_canonicas
      FROM jsonb_array_elements(p_entradas) AS entrada;

    SELECT ultima_secuencia
      INTO STRICT checkpoint
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR UPDATE;

    SELECT *
      INTO evento_existente
      FROM vec_autorizacion.motivo_v2_evento_origen
     WHERE secuencia_origen = p_secuencia_origen
        OR evento_origen_ref = p_evento_origen_ref;
    IF FOUND THEN
        IF evento_existente.secuencia_origen IS DISTINCT FROM p_secuencia_origen
           OR evento_existente.evento_origen_ref IS DISTINCT FROM p_evento_origen_ref
           OR evento_existente.tipo_evento IS DISTINCT FROM 'publicacion'
           OR evento_existente.catalogo_id IS DISTINCT FROM p_catalogo_id
           OR evento_existente.catalogo_version IS DISTINCT FROM p_catalogo_version
           OR evento_existente.huella_evento_sha256 IS DISTINCT FROM p_huella_evento_sha256 THEN
            RETURN false;
        END IF;

        SELECT * INTO cabecera_existente
          FROM vec_autorizacion.motivo_v2_catalogo_publicado
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        SELECT jsonb_agg(
                   jsonb_build_object(
                       'clave', entrada_clave,
                       'vigente_desde', to_char(
                           vigente_desde AT TIME ZONE 'UTC',
                           'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                       ),
                       'vigente_hasta', CASE
                           WHEN vigente_hasta IS NULL THEN 'null'::jsonb
                           ELSE to_jsonb(to_char(
                               vigente_hasta AT TIME ZONE 'UTC',
                               'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                           ))
                       END
                   ) ORDER BY entrada_clave
               )
          INTO entradas_existentes
          FROM vec_autorizacion.motivo_v2_entrada
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        RETURN cabecera_existente.catalogo_id IS NOT NULL
           AND cabecera_existente.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND cabecera_existente.publicado_en = p_publicado_en
           AND cabecera_existente.evento_origen_ref = p_evento_origen_ref
           AND cabecera_existente.secuencia_origen = p_secuencia_origen
           AND entradas_existentes IS NOT DISTINCT FROM entradas_canonicas;
    END IF;

    IF p_secuencia_origen IS DISTINCT FROM checkpoint.ultima_secuencia + 1
       OR EXISTS (
            SELECT 1
              FROM vec_autorizacion.motivo_v2_catalogo_publicado
             WHERE catalogo_id = p_catalogo_id
               AND catalogo_version = p_catalogo_version
       )
       OR (
            p_catalogo_version > 1
            AND NOT EXISTS (
                SELECT 1
                  FROM vec_autorizacion.motivo_v2_catalogo_publicado
                 WHERE catalogo_id = p_catalogo_id
                   AND catalogo_version = p_catalogo_version - 1
                   AND publicado_en <= p_publicado_en
            )
       ) THEN
        RETURN false;
    END IF;

    INSERT INTO vec_autorizacion.motivo_v2_evento_origen (
        secuencia_origen, evento_origen_ref, tipo_evento, catalogo_id,
        catalogo_version, huella_evento_sha256
    ) VALUES (
        p_secuencia_origen, p_evento_origen_ref, 'publicacion', p_catalogo_id,
        p_catalogo_version, p_huella_evento_sha256
    );
    INSERT INTO vec_autorizacion.motivo_v2_catalogo_publicado (
        catalogo_id, catalogo_version, catalogo_huella_publicada_sha256,
        publicado_en, evento_origen_ref, secuencia_origen
    ) VALUES (
        p_catalogo_id, p_catalogo_version,
        p_catalogo_huella_publicada_sha256, p_publicado_en,
        p_evento_origen_ref, p_secuencia_origen
    );
    INSERT INTO vec_autorizacion.motivo_v2_entrada (
        catalogo_id, catalogo_version, entrada_clave,
        vigente_desde, vigente_hasta
    )
    SELECT
        p_catalogo_id,
        p_catalogo_version,
        entrada ->> 'clave',
        (entrada ->> 'vigente_desde')::timestamptz,
        CASE
            WHEN jsonb_typeof(entrada -> 'vigente_hasta') = 'null' THEN NULL
            ELSE (entrada ->> 'vigente_hasta')::timestamptz
        END
      FROM jsonb_array_elements(p_entradas) AS entrada;
    UPDATE vec_autorizacion.motivo_v2_checkpoint_origen
       SET ultima_secuencia = p_secuencia_origen,
           ultimo_evento_ref = p_evento_origen_ref,
           ultima_huella_evento_sha256 = p_huella_evento_sha256,
           actualizado_en = clock_timestamp()
     WHERE control_id = true;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.retirar_motivos_autorizacion_v2(
    p_evento_origen_ref text,
    p_secuencia_origen bigint,
    p_huella_evento_sha256 text,
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_catalogo_huella_retirada_sha256 text,
    p_retirado_en timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    checkpoint record;
    evento_existente record;
    catalogo_existente record;
    retirada_existente record;
BEGIN
    IF (p_evento_origen_ref ~ '^evento_[0-9a-f]{32}$') IS NOT TRUE
       OR p_secuencia_origen IS NULL
       OR p_secuencia_origen < 1
       OR (p_huella_evento_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_huella_evento_sha256 = repeat('0', 64)
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL
       OR p_catalogo_version < 1
       OR (p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 = repeat('0', 64)
       OR (p_catalogo_huella_retirada_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_retirada_sha256 = repeat('0', 64)
       OR p_catalogo_huella_retirada_sha256 =
          p_catalogo_huella_publicada_sha256 THEN
        RETURN false;
    END IF;
    IF p_retirado_en IS NULL OR isfinite(p_retirado_en) IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF extract(year FROM (p_retirado_en AT TIME ZONE 'UTC'))
           NOT BETWEEN 1 AND 9999
       OR p_retirado_en > clock_timestamp() THEN
        RETURN false;
    END IF;

    SELECT ultima_secuencia
      INTO STRICT checkpoint
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR UPDATE;

    SELECT *
      INTO evento_existente
      FROM vec_autorizacion.motivo_v2_evento_origen
     WHERE secuencia_origen = p_secuencia_origen
        OR evento_origen_ref = p_evento_origen_ref;
    IF FOUND THEN
        IF evento_existente.secuencia_origen IS DISTINCT FROM p_secuencia_origen
           OR evento_existente.evento_origen_ref IS DISTINCT FROM p_evento_origen_ref
           OR evento_existente.tipo_evento IS DISTINCT FROM 'retirada'
           OR evento_existente.catalogo_id IS DISTINCT FROM p_catalogo_id
           OR evento_existente.catalogo_version IS DISTINCT FROM p_catalogo_version
           OR evento_existente.huella_evento_sha256 IS DISTINCT FROM p_huella_evento_sha256 THEN
            RETURN false;
        END IF;
        SELECT * INTO retirada_existente
          FROM vec_autorizacion.motivo_v2_retirada
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        SELECT * INTO catalogo_existente
          FROM vec_autorizacion.motivo_v2_catalogo_publicado
         WHERE catalogo_id = p_catalogo_id
           AND catalogo_version = p_catalogo_version;
        RETURN retirada_existente.catalogo_id IS NOT NULL
           AND catalogo_existente.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND retirada_existente.catalogo_huella_retirada_sha256 =
               p_catalogo_huella_retirada_sha256
           AND retirada_existente.retirado_en = p_retirado_en
           AND retirada_existente.evento_origen_ref = p_evento_origen_ref
           AND retirada_existente.secuencia_origen = p_secuencia_origen;
    END IF;

    SELECT * INTO catalogo_existente
      FROM vec_autorizacion.motivo_v2_catalogo_publicado
     WHERE catalogo_id = p_catalogo_id
       AND catalogo_version = p_catalogo_version;
    IF NOT FOUND
       OR catalogo_existente.catalogo_huella_publicada_sha256 IS DISTINCT FROM
          p_catalogo_huella_publicada_sha256
       OR p_retirado_en < catalogo_existente.publicado_en
       OR p_secuencia_origen IS DISTINCT FROM checkpoint.ultima_secuencia + 1
       OR EXISTS (
            SELECT 1 FROM vec_autorizacion.motivo_v2_retirada
             WHERE catalogo_id = p_catalogo_id
               AND catalogo_version = p_catalogo_version
       ) THEN
        RETURN false;
    END IF;

    INSERT INTO vec_autorizacion.motivo_v2_evento_origen (
        secuencia_origen, evento_origen_ref, tipo_evento, catalogo_id,
        catalogo_version, huella_evento_sha256
    ) VALUES (
        p_secuencia_origen, p_evento_origen_ref, 'retirada', p_catalogo_id,
        p_catalogo_version, p_huella_evento_sha256
    );
    INSERT INTO vec_autorizacion.motivo_v2_retirada (
        catalogo_id, catalogo_version, catalogo_huella_retirada_sha256,
        retirado_en, evento_origen_ref, secuencia_origen
    ) VALUES (
        p_catalogo_id, p_catalogo_version,
        p_catalogo_huella_retirada_sha256, p_retirado_en,
        p_evento_origen_ref, p_secuencia_origen
    );
    UPDATE vec_autorizacion.motivo_v2_checkpoint_origen
       SET ultima_secuencia = p_secuencia_origen,
           ultimo_evento_ref = p_evento_origen_ref,
           ultima_huella_evento_sha256 = p_huella_evento_sha256,
           actualizado_en = clock_timestamp()
     WHERE control_id = true;
    RETURN true;
END
$funcion$;

-- Consulta probatoria al instante evaluado por el PDP. No bloquea retiradas;
-- es la unica funcion concedida al evaluador historico.
CREATE FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_entrada_clave text,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL
       OR p_catalogo_version < 1
       OR (p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 = repeat('0', 64)
       OR (p_entrada_clave ~ '^motivo_[0-9a-f]{32}$') IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF p_instante IS NULL OR isfinite(p_instante) IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF extract(year FROM (p_instante AT TIME ZONE 'UTC')) NOT BETWEEN 1 AND 9999
       OR p_instante > statement_timestamp() THEN
        RETURN false;
    END IF;

    RETURN EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_v2_catalogo_publicado AS catalogo
          JOIN vec_autorizacion.motivo_v2_entrada AS entrada
            ON entrada.catalogo_id = catalogo.catalogo_id
           AND entrada.catalogo_version = catalogo.catalogo_version
         WHERE catalogo.catalogo_id = p_catalogo_id
           AND catalogo.catalogo_version = p_catalogo_version
           AND catalogo.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND entrada.entrada_clave = p_entrada_clave
           AND catalogo.publicado_en <= p_instante
           AND entrada.vigente_desde <= p_instante
           AND (entrada.vigente_hasta IS NULL
                OR p_instante < entrada.vigente_hasta)
           AND NOT EXISTS (
                SELECT 1
                  FROM vec_autorizacion.motivo_v2_retirada AS retirada
                 WHERE retirada.catalogo_id = catalogo.catalogo_id
                   AND retirada.catalogo_version = catalogo.catalogo_version
                   AND retirada.retirado_en <= p_instante
           )
    );
END
$funcion$;

-- Helper privado para futuras funciones definidoras de registro/efecto. El
-- bloqueo compartido se conserva hasta el COMMIT: una retirada concurrente,
-- que necesita FOR UPDATE, no puede adelantarse al efecto. Nunca se sustituye
-- el reloj de base por la fecha emitida por una decision o por un cliente.
CREATE FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_entrada_clave text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante_actual timestamptz;
BEGIN
    IF (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL
       OR p_catalogo_version < 1
       OR (p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 = repeat('0', 64)
       OR (p_entrada_clave ~ '^motivo_[0-9a-f]{32}$') IS NOT TRUE THEN
        RETURN false;
    END IF;
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    instante_actual := clock_timestamp();
    RETURN EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_v2_catalogo_publicado AS catalogo
          JOIN vec_autorizacion.motivo_v2_entrada AS entrada
            ON entrada.catalogo_id = catalogo.catalogo_id
           AND entrada.catalogo_version = catalogo.catalogo_version
         WHERE catalogo.catalogo_id = p_catalogo_id
           AND catalogo.catalogo_version = p_catalogo_version
           AND catalogo.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND entrada.entrada_clave = p_entrada_clave
           AND catalogo.publicado_en <= instante_actual
           AND entrada.vigente_desde <= instante_actual
           AND (entrada.vigente_hasta IS NULL
                OR instante_actual < entrada.vigente_hasta)
           AND NOT EXISTS (
                SELECT 1
                  FROM vec_autorizacion.motivo_v2_retirada AS retirada
                 WHERE retirada.catalogo_id = catalogo.catalogo_id
                   AND retirada.catalogo_version = catalogo.catalogo_version
           )
    );
END
$funcion$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'motivo_v2_evento_origen',
        'motivo_v2_catalogo_publicado',
        'motivo_v2_entrada',
        'motivo_v2_retirada',
        'motivo_v2_checkpoint_origen'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla,
            'vec_autorizacion_propietario',
            'vec_autorizacion_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_autorizacion
    FROM PUBLIC, vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
-- Los tipos de fila implícitos requieren una revocación propia.
REVOKE ALL ON TYPE
    vec_autorizacion.motivo_v2_evento_origen, vec_autorizacion.motivo_v2_catalogo_publicado,
    vec_autorizacion.motivo_v2_entrada, vec_autorizacion.motivo_v2_retirada,
    vec_autorizacion.motivo_v2_checkpoint_origen FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable()
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.motivo_v2_validar_avance_checkpoint()
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.motivo_v2_instante_canonico_valido(text)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.motivo_v2_entradas_validas(jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.publicar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, timestamptz, jsonb
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.retirar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, text, timestamptz
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
    text, integer, text, text, timestamptz
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
    text, integer, text, text
) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_motivos_proyector,
       vec_autorizacion_motivos_evaluador;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, timestamptz, jsonb
) TO vec_autorizacion_motivos_proyector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.retirar_motivos_autorizacion_v2(
    text, bigint, text, text, integer, text, text, timestamptz
) TO vec_autorizacion_motivos_proyector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
    text, integer, text, text, timestamptz
) TO vec_autorizacion_motivos_evaluador;

COMMIT;
