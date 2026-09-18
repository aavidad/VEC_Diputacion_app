-- CT-000007 (Bolsa): constitución de una bolsa a partir de un acta de importación
-- Convoca confirmada por RRHH. Persiste los canónicos de BolsaConstituida e
-- InstantaneaOrdenBolsa (mismas huellas e invariantes que el almacén
-- autoritativo de 000001, que no tiene por qué estar instalado), el vínculo
-- acta → bolsa (idempotente por acta) y orden → fila del acta. Sin datos
-- personales: los nombres se recuperan del staging protegido al leer.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000007', 0));

CREATE SCHEMA IF NOT EXISTS vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.constitucion_texto_valido(p_valor text, p_maximo integer)
RETURNS boolean LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[[:space:][:cntrl:]]'
       AND strpos(p_valor, '*') = 0
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.constitucion_texto_valido(text, integer) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'la constitucion de bolsa es inmutable';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion() FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.bolsa_constituida (
    bolsa_ref text NOT NULL,
    version bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    bolsa_canonica bytea NOT NULL,
    categoria_ref text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    estado text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (bolsa_ref, version, huella_bolsa_sha256),
    CHECK (version > 0),
    CHECK (estado IN ('vigente', 'suspendida', 'extinguida')),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(bolsa_ref, 512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(categoria_ref, 512)),
    CHECK (huella_bolsa_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (octet_length(bolsa_canonica) BETWEEN 2 AND 8388608),
    CHECK (encode(sha256(bolsa_canonica), 'hex') = huella_bolsa_sha256),
    CHECK (vigente_hasta IS NULL OR vigente_desde < vigente_hasta)
);

CREATE TABLE vec_bolsa_llamamientos.instantanea_orden_bolsa (
    instantanea_ref text NOT NULL,
    version bigint NOT NULL,
    huella_instantanea_sha256 text NOT NULL,
    instantanea_canonica bytea NOT NULL,
    bolsa_ref text NOT NULL,
    version_bolsa bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    total_participaciones bigint NOT NULL,
    referida_en timestamptz(6) NOT NULL,
    generada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (instantanea_ref, version, huella_instantanea_sha256),
    FOREIGN KEY (bolsa_ref, version_bolsa, huella_bolsa_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref, version, huella_bolsa_sha256),
    CHECK (version > 0 AND total_participaciones > 0),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(instantanea_ref, 512)),
    CHECK (huella_instantanea_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (octet_length(instantanea_canonica) BETWEEN 2 AND 33554432),
    CHECK (encode(sha256(instantanea_canonica), 'hex') = huella_instantanea_sha256),
    CHECK (referida_en <= generada_en)
);

CREATE TABLE vec_bolsa_llamamientos.constitucion (
    acta_ref text PRIMARY KEY,
    bolsa_ref text NOT NULL,
    version_bolsa bigint NOT NULL,
    huella_bolsa_sha256 text NOT NULL,
    instantanea_ref text NOT NULL,
    version_instantanea bigint NOT NULL,
    huella_instantanea_sha256 text NOT NULL,
    categoria_ref text NOT NULL,
    actor_ref text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (bolsa_ref, version_bolsa, huella_bolsa_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref, version, huella_bolsa_sha256),
    FOREIGN KEY (instantanea_ref, version_instantanea, huella_instantanea_sha256)
        REFERENCES vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref, version, huella_instantanea_sha256),
    UNIQUE (bolsa_ref, version_bolsa),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(acta_ref, 512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(categoria_ref, 512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(actor_ref, 512)),
    CHECK (confirmada_en <= registrada_en)
);

CREATE TABLE vec_bolsa_llamamientos.constitucion_entrada (
    instantanea_ref text NOT NULL,
    version_instantanea bigint NOT NULL,
    orden bigint NOT NULL,
    participacion_ref text NOT NULL,
    fila_numero integer NOT NULL,
    PRIMARY KEY (instantanea_ref, version_instantanea, orden),
    UNIQUE (instantanea_ref, version_instantanea, participacion_ref),
    UNIQUE (instantanea_ref, version_instantanea, fila_numero),
    CHECK (orden > 0 AND fila_numero > 0),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(participacion_ref, 512))
);

DO $proteccion$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['bolsa_constituida', 'instantanea_orden_bolsa', 'constitucion', 'constitucion_entrada'] LOOP
        EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format(
            -- TO explícito: una política para PUBLIC (polroles={0}) la rechaza la
            -- acreditación de los runtimes de RRHH aunque su predicado no conceda nada.
            'CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_llamamientos_propietario', 'vec_bolsa_llamamientos_propietario'
        );
        EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC', tabla);
        EXECUTE format(
            'CREATE TRIGGER negar_mutacion BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',
            tabla
        );
    END LOOP;
END
$proteccion$;

-- Constituye la bolsa. Idempotente por acta: una segunda llamada con la misma
-- acta devuelve la constitución existente sin escribir nada. Las huellas de
-- los canónicos las calcula la propia función y las comprueban las tablas.
CREATE FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
    p_acta_ref text,
    p_actor_ref text,
    p_categoria_ref text,
    p_bolsa_ref text,
    p_version_bolsa bigint,
    p_bolsa_canonica bytea,
    p_vigente_desde timestamptz,
    p_instantanea_ref text,
    p_version_instantanea bigint,
    p_instantanea_canonica bytea,
    p_referida_en timestamptz,
    p_generada_en timestamptz,
    p_entradas jsonb,
    p_confirmada_en timestamptz
)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
SET statement_timeout = '30s'
AS $funcion$
DECLARE
    v_existente vec_bolsa_llamamientos.constitucion%ROWTYPE;
    v_huella_bolsa text;
    v_huella_instantanea text;
    v_total bigint;
    v_entrada jsonb;
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
BEGIN
    IF NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_acta_ref, 512)
       OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_actor_ref, 512)
       OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_categoria_ref, 512)
       OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_bolsa_ref, 512)
       OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_instantanea_ref, 512)
       OR p_version_bolsa IS NULL OR p_version_bolsa < 1
       OR p_version_instantanea IS NULL OR p_version_instantanea < 1
       OR p_bolsa_canonica IS NULL OR octet_length(p_bolsa_canonica) < 2
       OR p_instantanea_canonica IS NULL OR octet_length(p_instantanea_canonica) < 2
       OR p_vigente_desde IS NULL OR p_referida_en IS NULL OR p_generada_en IS NULL
       OR p_confirmada_en IS NULL OR p_referida_en > p_generada_en
       OR p_entradas IS NULL OR jsonb_typeof(p_entradas) <> 'array'
       OR jsonb_array_length(p_entradas) < 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'constitucion de bolsa invalida';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:constitucion:' || p_acta_ref, 0));
    SELECT * INTO v_existente FROM vec_bolsa_llamamientos.constitucion WHERE acta_ref = p_acta_ref;
    IF FOUND THEN
        IF v_existente.bolsa_ref <> p_bolsa_ref OR v_existente.categoria_ref <> p_categoria_ref THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'acta ya constituida en otra bolsa';
        END IF;
        RETURN jsonb_build_object(
            'reutilizada', true, 'acta_ref', v_existente.acta_ref,
            'bolsa_ref', v_existente.bolsa_ref, 'version_bolsa', v_existente.version_bolsa,
            'instantanea_ref', v_existente.instantanea_ref,
            'version_instantanea', v_existente.version_instantanea,
            'confirmada_en', to_char(v_existente.confirmada_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        );
    END IF;
    v_huella_bolsa := encode(sha256(p_bolsa_canonica), 'hex');
    v_huella_instantanea := encode(sha256(p_instantanea_canonica), 'hex');
    v_total := jsonb_array_length(p_entradas);
    INSERT INTO vec_bolsa_llamamientos.bolsa_constituida (
        bolsa_ref, version, huella_bolsa_sha256, bolsa_canonica, categoria_ref,
        vigente_desde, vigente_hasta, estado, registrada_en
    ) VALUES (
        p_bolsa_ref, p_version_bolsa, v_huella_bolsa, p_bolsa_canonica, p_categoria_ref,
        p_vigente_desde, NULL, 'vigente', v_ahora
    );
    INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa (
        instantanea_ref, version, huella_instantanea_sha256, instantanea_canonica,
        bolsa_ref, version_bolsa, huella_bolsa_sha256, total_participaciones,
        referida_en, generada_en, registrada_en
    ) VALUES (
        p_instantanea_ref, p_version_instantanea, v_huella_instantanea, p_instantanea_canonica,
        p_bolsa_ref, p_version_bolsa, v_huella_bolsa, v_total,
        p_referida_en, p_generada_en, v_ahora
    );
    FOR v_entrada IN SELECT value FROM jsonb_array_elements(p_entradas) LOOP
        INSERT INTO vec_bolsa_llamamientos.constitucion_entrada (
            instantanea_ref, version_instantanea, orden, participacion_ref, fila_numero
        ) VALUES (
            p_instantanea_ref, p_version_instantanea,
            (v_entrada ->> 'orden')::bigint, v_entrada ->> 'participacion_ref',
            (v_entrada ->> 'fila_numero')::integer
        );
    END LOOP;
    INSERT INTO vec_bolsa_llamamientos.constitucion (
        acta_ref, bolsa_ref, version_bolsa, huella_bolsa_sha256,
        instantanea_ref, version_instantanea, huella_instantanea_sha256,
        categoria_ref, actor_ref, confirmada_en, registrada_en
    ) VALUES (
        p_acta_ref, p_bolsa_ref, p_version_bolsa, v_huella_bolsa,
        p_instantanea_ref, p_version_instantanea, v_huella_instantanea,
        p_categoria_ref, p_actor_ref, date_trunc('microseconds', p_confirmada_en), v_ahora
    );
    RETURN jsonb_build_object(
        'reutilizada', false, 'acta_ref', p_acta_ref,
        'bolsa_ref', p_bolsa_ref, 'version_bolsa', p_version_bolsa,
        'instantanea_ref', p_instantanea_ref, 'version_instantanea', p_version_instantanea,
        'confirmada_en', to_char(date_trunc('microseconds', p_confirmada_en) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
END
$funcion$;

-- Última constitución de cada categoría (la más reciente sustituye a las
-- anteriores de la misma categoría: B9).
CREATE FUNCTION vec_bolsa_llamamientos.listar_constituciones_v1()
RETURNS TABLE(
    acta_ref text, bolsa_ref text, version_bolsa bigint, categoria_ref text,
    vigente_desde timestamptz, vigente_hasta timestamptz, estado text,
    instantanea_ref text, version_instantanea bigint, instantanea_canonica bytea,
    total_participaciones bigint, confirmada_en timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT DISTINCT ON (c.categoria_ref)
           c.acta_ref, c.bolsa_ref, c.version_bolsa, c.categoria_ref,
           b.vigente_desde, b.vigente_hasta, b.estado,
           c.instantanea_ref, c.version_instantanea, i.instantanea_canonica,
           i.total_participaciones, c.confirmada_en
      FROM vec_bolsa_llamamientos.constitucion c
      JOIN vec_bolsa_llamamientos.bolsa_constituida b
        ON b.bolsa_ref = c.bolsa_ref AND b.version = c.version_bolsa
       AND b.huella_bolsa_sha256 = c.huella_bolsa_sha256
      JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
        ON i.instantanea_ref = c.instantanea_ref AND i.version = c.version_instantanea
       AND i.huella_instantanea_sha256 = c.huella_instantanea_sha256
     ORDER BY c.categoria_ref, c.confirmada_en DESC, c.registrada_en DESC;
$funcion$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_entradas_constitucion_v1(
    p_instantanea_ref text, p_version_instantanea bigint
)
RETURNS TABLE(orden bigint, participacion_ref text, fila_numero integer)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT e.orden, e.participacion_ref, e.fila_numero
      FROM vec_bolsa_llamamientos.constitucion_entrada e
     WHERE e.instantanea_ref = p_instantanea_ref AND e.version_instantanea = p_version_instantanea
     ORDER BY e.orden;
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
    text, text, text, text, bigint, bytea, timestamptz, text, bigint, bytea, timestamptz, timestamptz, jsonb, timestamptz
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_constituciones_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_entradas_constitucion_v1(text, bigint) FROM PUBLIC;
-- Perfil de desarrollo: el ejecutor constituye y lee. En producción la
-- constitución pasa a un rol propio de RRHH; las lecturas quedan en el ejecutor.
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
    text, text, text, text, bigint, bytea, timestamptz, text, bigint, bytea, timestamptz, timestamptz, jsonb, timestamptz
) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_constituciones_v1() TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_entradas_constitucion_v1(text, bigint) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
