\set ON_ERROR_STOP on
-- CT-000125: urgencia declarada por RRHH al analizar el expediente (duda 63).
-- La marca y su motivo viven en una tabla de solo adición enlazada con el
-- recibo del análisis que la declaró; el análisis publicado, su huella y el
-- canon del cuadro no cambian. Solo se escribe en la misma transacción que
-- confirma el análisis (la autorización ya se consumió en ella) y una
-- repetición con el mismo motivo devuelve «repetida».
-- La fachada v4 del cuadro añade, alineada con fase_desde_expedientes de la
-- v3, si cada expediente consta como urgente en su versión publicada. El
-- plazo que corresponda lo resuelve la aplicación con el catálogo de reglas.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000125', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.urgencia_expediente_analisis'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_urgencia_analisis_v1(text,text)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000125 ya instalada';
    END IF;
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.reserva_operacion_analisis'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.confirmacion_operacion_analisis'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para CT-000125: falta CT-000110 o el análisis O3';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.urgencia_expediente_analisis (
    expediente_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    motivo text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version),
    FOREIGN KEY (expediente_ref, version)
        REFERENCES vec_contratacion_temporal.expediente_version_integral (
            expediente_ref, version
        ),
    FOREIGN KEY (recibo_ref)
        REFERENCES vec_contratacion_temporal.reserva_operacion_analisis (
            recibo_ref
        ),
    CHECK (version BETWEEN 2 AND 9007199254740991::numeric),
    CHECK (pg_catalog.char_length(motivo) BETWEEN 1 AND 1000
       AND motivo = pg_catalog.btrim(motivo, E' \t\n\r')
       AND motivo !~ '[\x01-\x08\x0b\x0c\x0e-\x1f\x7f-\x9f]'),
    CHECK (registrada_en = pg_catalog.date_trunc('microseconds', registrada_en))
);

COMMENT ON TABLE vec_contratacion_temporal.urgencia_expediente_analisis IS
'Urgencia declarada por RRHH en un análisis confirmado (versión resultante y recibo). Solo adición: una vez declarada se mantiene.';

CREATE FUNCTION vec_contratacion_temporal.registrar_urgencia_analisis_v1(
    p_recibo_ref text,
    p_motivo text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_analisis record;
    v_existente record;
BEGIN
    IF p_recibo_ref IS NULL OR p_motivo IS NULL
       OR pg_catalog.char_length(p_motivo) NOT BETWEEN 1 AND 1000
       OR p_motivo <> pg_catalog.btrim(p_motivo, E' \t\n\r')
       OR p_motivo ~ '[\x01-\x08\x0b\x0c\x0e-\x1f\x7f-\x9f]' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'urgencia del análisis no válida';
    END IF;
    SELECT reserva.expediente_ref,
           reserva.version_expediente + 1 AS version,
           pg_catalog.age(confirmacion.xmin) AS antiguedad
      INTO v_analisis
      FROM vec_contratacion_temporal.reserva_operacion_analisis reserva
      JOIN vec_contratacion_temporal.confirmacion_operacion_analisis confirmacion
        USING (ambito_raiz_hmac)
     WHERE reserva.recibo_ref = p_recibo_ref;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'urgencia sin análisis confirmado';
    END IF;
    SELECT urgencia.expediente_ref, urgencia.version, urgencia.motivo
      INTO v_existente
      FROM vec_contratacion_temporal.urgencia_expediente_analisis urgencia
     WHERE urgencia.recibo_ref = p_recibo_ref;
    IF FOUND THEN
        IF v_existente.motivo = p_motivo
           AND v_existente.expediente_ref = v_analisis.expediente_ref
           AND v_existente.version = v_analisis.version THEN
            RETURN 'repetida';
        END IF;
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'el análisis ya declaró otra urgencia';
    END IF;
    -- Solo la transacción que confirmó el análisis (o una de sus
    -- subtransacciones) puede declarar su urgencia: la edad de la fila de
    -- confirmación respecto a la transacción actual es 0 o negativa.
    IF v_analisis.antiguedad > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'urgencia fuera de la confirmación del análisis';
    END IF;
    INSERT INTO vec_contratacion_temporal.urgencia_expediente_analisis (
        expediente_ref, version, recibo_ref, motivo, registrada_en
    ) VALUES (
        v_analisis.expediente_ref, v_analisis.version, p_recibo_ref, p_motivo,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN 'registrada';
END
$funcion$;

CREATE TRIGGER urgencia_expediente_analisis_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.urgencia_expediente_analisis
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER urgencia_expediente_analisis_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.urgencia_expediente_analisis
FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.urgencia_expediente_analisis
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.urgencia_expediente_analisis
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.urgencia_expediente_analisis
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.urgencia_expediente_analisis
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_capacidad_canonica bytea, p_decision_canonica bytea,
    p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea, p_raiz_publica_spki bytea
)
RETURNS TABLE(
    contenido_canonico bytea, cursor_siguiente text, esquema text,
    acceso_ref text, secuencia numeric, anterior_sha256 text,
    huella_sha256 text, vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text, registrada_en timestamptz,
    auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
    resultado_huella_sha256 text, cursor_huella_sha256 text,
    generada_en timestamptz, expediente_ref text,
    version_expediente numeric, total smallint, recibo_sello_sha256 text,
    total_filtrado numeric, en_tramitacion numeric,
    con_incidencia numeric, en_llamamiento numeric,
    fase_desde_expedientes text[], fase_desde_instantes timestamptz[],
    urgente_expedientes boolean[]
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_v3 record;
    v_urgentes boolean[];
    v_filas integer;
BEGIN
    SELECT * INTO STRICT v_v3
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    SELECT COALESCE(pg_catalog.array_agg(EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.urgencia_expediente_analisis u
                WHERE u.expediente_ref = leido.expediente_ref
                  AND u.version <= leido.version
           ) ORDER BY leido.orden), '{}'),
           pg_catalog.count(*)::integer
      INTO v_urgentes, v_filas
      FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(
               v_v3.contenido_canonico
           ) leido;
    IF v_filas <> v_v3.total
       OR pg_catalog.cardinality(v_urgentes)
          <> pg_catalog.cardinality(v_v3.fase_desde_expedientes) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'urgencia de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY SELECT v_v3.contenido_canonico, v_v3.cursor_siguiente,
        v_v3.esquema, v_v3.acceso_ref, v_v3.secuencia,
        v_v3.anterior_sha256, v_v3.huella_sha256,
        v_v3.vinculo_identidad_huella_sha256,
        v_v3.alcance_huella_sha256, v_v3.registrada_en,
        v_v3.auditoria_vec_ref, v_v3.auditoria_vec_huella_sha256,
        v_v3.consumo_vec_huella_sha256, v_v3.contenido_huella_sha256,
        v_v3.resultado_huella_sha256, v_v3.cursor_huella_sha256,
        v_v3.generada_en, v_v3.expediente_ref, v_v3.version_expediente,
        v_v3.total, v_v3.recibo_sello_sha256, v_v3.total_filtrado,
        v_v3.en_tramitacion, v_v3.con_incidencia, v_v3.en_llamamiento,
        v_v3.fase_desde_expedientes, v_v3.fase_desde_instantes, v_urgentes;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.registrar_urgencia_analisis_v1(text, text)
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_urgencia_analisis_v1(text, text)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_urgencia_analisis_v1(text, text)
TO vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
