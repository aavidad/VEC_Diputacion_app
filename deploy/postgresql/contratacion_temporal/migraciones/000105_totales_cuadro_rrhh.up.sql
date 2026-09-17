\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000105', 0
    )
);

-- El contador obtiene el corte después de v1. En una continuación el
-- token ya está consumido, pero su familia conserva el corte ya validado.
CREATE FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_cursor text
)
RETURNS TABLE(
    total_filtrado numeric,
    en_tramitacion numeric,
    con_incidencia numeric,
    en_llamamiento numeric
)
LANGUAGE plpgsql
STABLE
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
    v_corte_global numeric;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_cursor IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'totales de cuadro RRHH no disponibles';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_consulta);
    IF p_cursor = '' THEN
        SELECT ultimo_corte INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
    ELSE
        SELECT familia.corte_global INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
          JOIN vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
            USING (familia_ref)
         WHERE cursor.token_huella_sha256 = pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(p_cursor, 'UTF8')), 'hex'
         );
    END IF;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    -- Mantener estos predicados alineados con materializar_cuadro_rrhh_v1
    -- (CT44): ésta cuenta todo el corte, aquélla ordena y pagina.
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.organizacion_ref,
               publicada.numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(ultima.numero_visible, pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.estado_clave = '' OR ultima.estado_clave COLLATE "C" = p_consulta.estado_clave COLLATE "C")
           AND (p_consulta.fase_clave = '' OR ultima.fase_clave COLLATE "C" = p_consulta.fase_clave COLLATE "C")
    )
    SELECT pg_catalog.count(*)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'en_curso' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'incidencia' = estado_clave)::numeric,
           pg_catalog.count(*) FILTER (WHERE 'llamamiento' = fase_clave)::numeric
      FROM filtradas;
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1, text
) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
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
    con_incidencia numeric, en_llamamiento numeric
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
    v_v1 record;
    v_totales record;
BEGIN
    SELECT * INTO STRICT v_v1
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    SELECT * INTO STRICT v_totales
      FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
          p_alcance, p_consulta, p_consulta.cursor
      );
    IF v_totales.total_filtrado < 0
       OR v_totales.en_tramitacion < 0
       OR v_totales.con_incidencia < 0
       OR v_totales.en_llamamiento < 0
       OR v_totales.en_tramitacion > v_totales.total_filtrado
       OR v_totales.con_incidencia > v_totales.total_filtrado
       OR v_totales.en_llamamiento > v_totales.total_filtrado
    THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'totales de cuadro RRHH no disponibles';
    END IF;
    RETURN QUERY SELECT v_v1.contenido_canonico, v_v1.cursor_siguiente,
        v_v1.esquema, v_v1.acceso_ref, v_v1.secuencia,
        v_v1.anterior_sha256, v_v1.huella_sha256,
        v_v1.vinculo_identidad_huella_sha256,
        v_v1.alcance_huella_sha256, v_v1.registrada_en,
        v_v1.auditoria_vec_ref, v_v1.auditoria_vec_huella_sha256,
        v_v1.consumo_vec_huella_sha256, v_v1.contenido_huella_sha256,
        v_v1.resultado_huella_sha256, v_v1.cursor_huella_sha256,
        v_v1.generada_en, v_v1.expediente_ref, v_v1.version_expediente,
        v_v1.total, v_v1.recibo_sello_sha256, v_totales.total_filtrado,
        v_totales.en_tramitacion, v_totales.con_incidencia,
        v_totales.en_llamamiento;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
