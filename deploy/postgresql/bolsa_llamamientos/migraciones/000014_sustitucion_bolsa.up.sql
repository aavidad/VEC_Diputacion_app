\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000014', 0));

-- B9 (manual de bolsas de Mérida §2; ficha de adaptación a CONVOCA, fila B9): la
-- bolsa constituida con el listado definitivo entra en vigor y sustituye a las
-- anteriores de su misma categoría. Las filas de bolsa_constituida son inmutables,
-- así que la sustitución se registra en una tabla append-only y las lecturas
-- derivan de ella vigente_hasta y estado ('extinguida') de la bolsa sustituida.
CREATE TABLE vec_bolsa_llamamientos.sustitucion_bolsa (
    bolsa_ref_sustituida text NOT NULL,
    version_sustituida bigint NOT NULL,
    huella_sustituida_sha256 text NOT NULL,
    bolsa_ref_nueva text NOT NULL,
    version_nueva bigint NOT NULL,
    huella_nueva_sha256 text NOT NULL,
    categoria_ref text NOT NULL,
    acta_ref_nueva text NOT NULL,
    sustituida_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (bolsa_ref_sustituida, version_sustituida, huella_sustituida_sha256),
    FOREIGN KEY (bolsa_ref_sustituida, version_sustituida, huella_sustituida_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_constituida (bolsa_ref, version, huella_bolsa_sha256),
    FOREIGN KEY (bolsa_ref_nueva, version_nueva, huella_nueva_sha256)
        REFERENCES vec_bolsa_llamamientos.bolsa_constituida (bolsa_ref, version, huella_bolsa_sha256),
    FOREIGN KEY (acta_ref_nueva) REFERENCES vec_bolsa_llamamientos.constitucion (acta_ref),
    CHECK (bolsa_ref_sustituida <> bolsa_ref_nueva OR version_sustituida <> version_nueva),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(categoria_ref, 512))
);
CREATE INDEX sustitucion_bolsa_nueva ON vec_bolsa_llamamientos.sustitucion_bolsa (bolsa_ref_nueva, version_nueva);
ALTER TABLE vec_bolsa_llamamientos.sustitucion_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.sustitucion_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY sustitucion_bolsa_solo_propietario ON vec_bolsa_llamamientos.sustitucion_bolsa TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.sustitucion_bolsa FROM PUBLIC;
CREATE TRIGGER sustitucion_bolsa_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.sustitucion_bolsa FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Registra que la constitución p_acta_ref sustituye a toda constitución anterior
-- de su categoría todavía vigente. Una constitución confirmada antes que otra ya
-- registrada no sustituye a la posterior: la más reciente manda. Devuelve las
-- bolsas sustituidas para el recibo.
CREATE FUNCTION vec_bolsa_llamamientos.sustituir_bolsas_anteriores_v1(p_acta_ref text)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_nueva vec_bolsa_llamamientos.constitucion%ROWTYPE;
    v_anterior record;
    v_sustituidas jsonb := '[]'::jsonb;
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
BEGIN
    SELECT * INTO STRICT v_nueva FROM vec_bolsa_llamamientos.constitucion WHERE acta_ref = p_acta_ref;
    FOR v_anterior IN
        SELECT c.bolsa_ref, c.version_bolsa, c.huella_bolsa_sha256
          FROM vec_bolsa_llamamientos.constitucion c
         WHERE c.categoria_ref = v_nueva.categoria_ref
           AND c.acta_ref <> v_nueva.acta_ref
           AND (c.confirmada_en, c.registrada_en) < (v_nueva.confirmada_en, v_nueva.registrada_en)
           AND NOT EXISTS (
               SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
                WHERE s.bolsa_ref_sustituida = c.bolsa_ref
                  AND s.version_sustituida = c.version_bolsa
                  AND s.huella_sustituida_sha256 = c.huella_bolsa_sha256)
         ORDER BY c.confirmada_en, c.registrada_en
    LOOP
        INSERT INTO vec_bolsa_llamamientos.sustitucion_bolsa (
            bolsa_ref_sustituida, version_sustituida, huella_sustituida_sha256,
            bolsa_ref_nueva, version_nueva, huella_nueva_sha256,
            categoria_ref, acta_ref_nueva, sustituida_en, registrada_en
        ) VALUES (
            v_anterior.bolsa_ref, v_anterior.version_bolsa, v_anterior.huella_bolsa_sha256,
            v_nueva.bolsa_ref, v_nueva.version_bolsa, v_nueva.huella_bolsa_sha256,
            v_nueva.categoria_ref, v_nueva.acta_ref, v_nueva.confirmada_en, v_ahora
        );
        v_sustituidas := v_sustituidas || jsonb_build_object(
            'bolsa_ref', v_anterior.bolsa_ref, 'version_bolsa', v_anterior.version_bolsa);
    END LOOP;
    RETURN v_sustituidas;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.sustituir_bolsas_anteriores_v1(text) FROM PUBLIC;

-- constituir_bolsa_v1 conserva su contrato (misma firma, mismos errores) y añade
-- la sustitución en la misma transacción y el campo 'sustituye_a' en el recibo.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
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
    v_sustituidas jsonb;
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
    -- La sustitución se decide por categoría: se serializan las constituciones de
    -- la misma categoría para que dos actas simultáneas no se sustituyan a medias.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:categoria:' || p_categoria_ref, 0));
    SELECT * INTO v_existente FROM vec_bolsa_llamamientos.constitucion WHERE acta_ref = p_acta_ref;
    IF FOUND THEN
        IF v_existente.bolsa_ref <> p_bolsa_ref OR v_existente.categoria_ref <> p_categoria_ref THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'acta ya constituida en otra bolsa';
        END IF;
        SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref', s.bolsa_ref_sustituida, 'version_bolsa', s.version_sustituida) ORDER BY s.sustituida_en, s.registrada_en), '[]'::jsonb)
          INTO v_sustituidas
          FROM vec_bolsa_llamamientos.sustitucion_bolsa s
         WHERE s.acta_ref_nueva = v_existente.acta_ref;
        RETURN jsonb_build_object(
            'reutilizada', true, 'acta_ref', v_existente.acta_ref,
            'bolsa_ref', v_existente.bolsa_ref, 'version_bolsa', v_existente.version_bolsa,
            'instantanea_ref', v_existente.instantanea_ref,
            'version_instantanea', v_existente.version_instantanea,
            'confirmada_en', to_char(v_existente.confirmada_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'sustituye_a', v_sustituidas
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
    v_sustituidas := vec_bolsa_llamamientos.sustituir_bolsas_anteriores_v1(p_acta_ref);
    RETURN jsonb_build_object(
        'reutilizada', false, 'acta_ref', p_acta_ref,
        'bolsa_ref', p_bolsa_ref, 'version_bolsa', p_version_bolsa,
        'instantanea_ref', p_instantanea_ref, 'version_instantanea', p_version_instantanea,
        'confirmada_en', to_char(date_trunc('microseconds', p_confirmada_en) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'sustituye_a', v_sustituidas
    );
END
$funcion$;

-- Las lecturas derivan vigencia y estado de la sustitución registrada.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_constituciones_v1()
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
           b.vigente_desde,
           coalesce(s.sustituida_en, b.vigente_hasta),
           CASE WHEN s.bolsa_ref_sustituida IS NOT NULL THEN 'extinguida' ELSE b.estado END,
           c.instantanea_ref, c.version_instantanea, i.instantanea_canonica,
           i.total_participaciones, c.confirmada_en
      FROM vec_bolsa_llamamientos.constitucion c
      JOIN vec_bolsa_llamamientos.bolsa_constituida b
        ON b.bolsa_ref = c.bolsa_ref AND b.version = c.version_bolsa
       AND b.huella_bolsa_sha256 = c.huella_bolsa_sha256
      JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
        ON i.instantanea_ref = c.instantanea_ref AND i.version = c.version_instantanea
       AND i.huella_instantanea_sha256 = c.huella_instantanea_sha256
      LEFT JOIN vec_bolsa_llamamientos.sustitucion_bolsa s
        ON s.bolsa_ref_sustituida = b.bolsa_ref AND s.version_sustituida = b.version
       AND s.huella_sustituida_sha256 = b.huella_bolsa_sha256
     ORDER BY c.categoria_ref, c.confirmada_en DESC, c.registrada_en DESC;
$funcion$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref text)
RETURNS TABLE(
    participacion_ref text, acta_ref text, bolsa_ref text, version_bolsa bigint, categoria_ref text,
    vigente_desde timestamptz, vigente_hasta timestamptz, estado text,
    instantanea_ref text, version_instantanea bigint, orden bigint,
    total_participaciones bigint, confirmada_en timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT vc.participacion_ref, c.acta_ref, c.bolsa_ref, c.version_bolsa, c.categoria_ref,
           b.vigente_desde,
           coalesce(s.sustituida_en, b.vigente_hasta),
           CASE WHEN s.bolsa_ref_sustituida IS NOT NULL THEN 'extinguida' ELSE b.estado END,
           c.instantanea_ref, c.version_instantanea, e.orden,
           i.total_participaciones, c.confirmada_en
      FROM vec_bolsa_llamamientos.vinculo_candidato vc
      JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref = vc.acta_ref
      JOIN vec_bolsa_llamamientos.constitucion_entrada e
        ON e.instantanea_ref = vc.instantanea_ref AND e.version_instantanea = vc.version_instantanea
       AND e.participacion_ref = vc.participacion_ref
      JOIN vec_bolsa_llamamientos.bolsa_constituida b
        ON b.bolsa_ref = c.bolsa_ref AND b.version = c.version_bolsa
       AND b.huella_bolsa_sha256 = c.huella_bolsa_sha256
      JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
        ON i.instantanea_ref = c.instantanea_ref AND i.version = c.version_instantanea
       AND i.huella_instantanea_sha256 = c.huella_instantanea_sha256
      LEFT JOIN vec_bolsa_llamamientos.sustitucion_bolsa s
        ON s.bolsa_ref_sustituida = b.bolsa_ref AND s.version_sustituida = b.version
       AND s.huella_sustituida_sha256 = b.huella_bolsa_sha256
     WHERE vc.candidato_ref = p_candidato_ref
     ORDER BY c.confirmada_en DESC, c.categoria_ref;
$funcion$;

-- Historia ya constituida: cada categoría con más de una constitución queda
-- con la última vigente y las anteriores sustituidas por ella.
DO $relleno$
DECLARE v_acta text;
BEGIN
    FOR v_acta IN
        SELECT DISTINCT ON (c.categoria_ref) c.acta_ref
          FROM vec_bolsa_llamamientos.constitucion c
         ORDER BY c.categoria_ref, c.confirmada_en DESC, c.registrada_en DESC
    LOOP
        PERFORM vec_bolsa_llamamientos.sustituir_bolsas_anteriores_v1(v_acta);
    END LOOP;
END
$relleno$;
COMMIT;
