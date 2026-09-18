\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000008', 0)
);

-- Vínculo durable entre la persona candidata y su participación (C21, corte 1).
-- `candidato_ref` es la referencia opaca `can_*` que Bolsa deriva con clave
-- (HMAC) de la identidad enmascarada del acta; la frontera de identidad
-- deriva la misma referencia al acreditar a la persona, sin que el documento
-- se guarde nunca aquí. Una participación pertenece a un solo candidato y un
-- candidato tiene una sola participación por instantánea. Solo inserción.
CREATE TABLE vec_bolsa_llamamientos.vinculo_candidato (
    participacion_ref text PRIMARY KEY,
    candidato_ref text NOT NULL,
    acta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
    instantanea_ref text NOT NULL,
    version_instantanea bigint NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (candidato_ref, instantanea_ref, version_instantanea),
    FOREIGN KEY (instantanea_ref, version_instantanea, participacion_ref)
        REFERENCES vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref, version_instantanea, participacion_ref),
    CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(participacion_ref, 512))
);
CREATE INDEX vinculo_candidato_por_candidato
    ON vec_bolsa_llamamientos.vinculo_candidato (candidato_ref);

ALTER TABLE vec_bolsa_llamamientos.vinculo_candidato ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.vinculo_candidato FORCE ROW LEVEL SECURITY;
-- TO explícito: una política para PUBLIC la rechaza la acreditación de los runtimes.
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.vinculo_candidato
    TO vec_bolsa_llamamientos_propietario
    USING (current_user = 'vec_bolsa_llamamientos_propietario')
    WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.vinculo_candidato FROM PUBLIC;
CREATE TRIGGER negar_mutacion BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.vinculo_candidato
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Registra los vínculos de un acta ya constituida. Idempotente: un vínculo ya
-- registrado con el mismo candidato no escribe nada; con otro candidato, o si
-- la participación no pertenece a la instantánea del acta, falla (23505 /
-- 23503). Devuelve {"nuevos": n, "existentes": m}.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(
    p_acta_ref text,
    p_vinculos jsonb,
    p_registrada_en timestamptz
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
SET statement_timeout = '10s'
AS $funcion$
DECLARE
    v_instantanea_ref text;
    v_version_instantanea bigint;
    v_vinculo jsonb;
    v_candidato text;
    v_participacion text;
    v_existente text;
    v_nuevos integer := 0;
    v_existentes integer := 0;
BEGIN
    IF CURRENT_USER <> 'vec_bolsa_llamamientos_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'vinculos de candidato no disponibles';
    END IF;
    IF p_acta_ref IS NULL OR p_vinculos IS NULL OR pg_catalog.jsonb_typeof(p_vinculos) <> 'array'
       OR pg_catalog.jsonb_array_length(p_vinculos) = 0 OR pg_catalog.jsonb_array_length(p_vinculos) > 100000
       OR p_registrada_en IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'vinculos de candidato invalidos';
    END IF;
    SELECT c.instantanea_ref, c.version_instantanea INTO v_instantanea_ref, v_version_instantanea
      FROM vec_bolsa_llamamientos.constitucion c
     WHERE c.acta_ref = p_acta_ref
       FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '23503', MESSAGE = 'acta no constituida';
    END IF;
    FOR v_vinculo IN SELECT * FROM pg_catalog.jsonb_array_elements(p_vinculos) LOOP
        v_candidato := v_vinculo ->> 'candidato_ref';
        v_participacion := v_vinculo ->> 'participacion_ref';
        IF v_candidato IS NULL OR v_participacion IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'vinculo de candidato incompleto';
        END IF;
        SELECT vc.candidato_ref INTO v_existente
          FROM vec_bolsa_llamamientos.vinculo_candidato vc
         WHERE vc.participacion_ref = v_participacion;
        IF FOUND THEN
            IF v_existente <> v_candidato THEN
                RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'participacion vinculada a otro candidato';
            END IF;
            v_existentes := v_existentes + 1;
            CONTINUE;
        END IF;
        INSERT INTO vec_bolsa_llamamientos.vinculo_candidato (
            participacion_ref, candidato_ref, acta_ref, instantanea_ref, version_instantanea, registrada_en
        ) VALUES (v_participacion, v_candidato, p_acta_ref, v_instantanea_ref, v_version_instantanea, p_registrada_en);
        v_nuevos := v_nuevos + 1;
    END LOOP;
    RETURN pg_catalog.jsonb_build_object('nuevos', v_nuevos, 'existentes', v_existentes);
END
$funcion$;

-- Participaciones de una persona candidata en todas sus constituciones, la
-- más reciente primero. Sin datos personales: solo referencias, orden y
-- metadatos de la bolsa.
CREATE FUNCTION vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref text)
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
           b.vigente_desde, b.vigente_hasta, b.estado,
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
     WHERE vc.candidato_ref = p_candidato_ref
     ORDER BY c.confirmada_en DESC, c.categoria_ref;
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text, jsonb, timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text, jsonb, timestamptz) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
