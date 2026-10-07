\set ON_ERROR_STOP on
-- CT-000184: resumen de la portada de RRHH sin descargar el cuadro.
-- La portada contaba en el navegador las filas de la primera página (100) del
-- cuadro: con más expedientes los recuentos eran parciales y la portada
-- descargaba filas que no muestra. La fachada v5 del cuadro añade, sobre la
-- misma consulta atestada (misma capacidad V3, mismo consumo y misma
-- auditoría de acceso que la v4), dos agregados de todo el corte filtrado:
--   * recuentos por estado y fase, y
--   * para los expedientes en trámite (ni completados ni cancelados),
--     grupos (fase, entrada en fase, urgente) con su número, sin
--     referencias de expediente, para que la aplicación resuelva el plazo de
--     cada grupo con el catálogo de reglas (como ya hace por fila).
-- Ninguna fila nueva sale del módulo: solo números y claves de catálogo.
-- La página de expedientes de la v4 no cambia y sigue saliendo.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000184', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000184 ya instalada';
    END IF;
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.fase_entrada_publicacion_rrhh'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.urgencia_expediente_analisis'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para CT-000184: faltan CT-000110, CT-000125 o CT-000142';
    END IF;
END
$prevalidacion$;

-- Mismo corte y mismos predicados que contar_totales_cuadro_rrhh_v1 (y, por
-- ella, que materializar_cuadro_rrhh_v1): ésta agrega en lugar de contar.
-- Solo la invoca la fachada del propietario.
CREATE FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_cursor text
)
RETURNS TABLE(
    clase text,
    estado_clave text,
    fase_clave text,
    fase_desde timestamptz,
    urgente boolean,
    numero numeric
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
            MESSAGE = 'resumen de cuadro RRHH no disponible';
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
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.version,
               publicada.organizacion_ref,
               vec_contratacion_temporal.numero_visible_vigente_v1(
                   publicada.expediente_ref, publicada.numero_visible
               ) AS numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.expediente_ref, ultima.version,
               ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(ultima.numero_visible, pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.estado_clave = '' OR ultima.estado_clave COLLATE "C" = p_consulta.estado_clave COLLATE "C")
           AND (p_consulta.fase_clave = '' OR ultima.fase_clave COLLATE "C" = p_consulta.fase_clave COLLATE "C")
    )
    SELECT 'estado_fase'::text, filtrada.estado_clave, filtrada.fase_clave,
           NULL::timestamptz, NULL::boolean, pg_catalog.count(*)::numeric
      FROM filtradas filtrada
     GROUP BY filtrada.estado_clave, filtrada.fase_clave
    UNION ALL
    -- Un expediente en trámite sin fecha de entrada en fase aparece con
    -- fase_desde nula: la fachada lo rechaza en lugar de omitirlo.
    SELECT 'plazo'::text, NULL::text, filtrada.fase_clave, entrada.fase_desde,
           EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.urgencia_expediente_analisis u
                WHERE u.expediente_ref = filtrada.expediente_ref
                  AND u.version <= filtrada.version
           ),
           pg_catalog.count(*)::numeric
      FROM filtradas filtrada
      LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
        ON entrada.expediente_ref = filtrada.expediente_ref
       AND entrada.version = filtrada.version
     WHERE filtrada.estado_clave NOT IN ('completado', 'cancelado')
     GROUP BY filtrada.fase_clave, entrada.fase_desde, 5;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
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
    urgente_expedientes boolean[],
    recuento_estados text[], recuento_fases text[], recuento_numeros numeric[],
    plazo_fases text[], plazo_desde timestamptz[], plazo_urgentes boolean[],
    plazo_numeros numeric[]
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
    v_v4 record;
    v_estados text[];
    v_fases text[];
    v_numeros numeric[];
    v_total_recuento numeric;
    v_en_tramite numeric;
    v_plazo_fases text[];
    v_plazo_desde timestamptz[];
    v_plazo_urgentes boolean[];
    v_plazo_numeros numeric[];
    v_total_plazos numeric;
    v_sin_fase integer;
BEGIN
    -- Autorización, consumo, auditoría de acceso y página: los de la v4.
    SELECT * INTO STRICT v_v4
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    WITH resumen AS MATERIALIZED (
        SELECT *
          FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
              p_alcance, p_consulta, p_consulta.cursor
          )
    ), recuento AS (
        SELECT * FROM resumen WHERE clase = 'estado_fase'
    ), plazo AS (
        SELECT * FROM resumen WHERE clase = 'plazo'
    )
    SELECT (SELECT COALESCE(pg_catalog.array_agg(r.estado_clave ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM recuento r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM recuento r
             WHERE r.estado_clave NOT IN ('completado', 'cancelado')),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_desde ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.urgente ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM plazo r),
           (SELECT pg_catalog.count(*)::integer FROM plazo r WHERE r.fase_desde IS NULL)
      INTO v_estados, v_fases, v_numeros, v_total_recuento, v_en_tramite,
           v_plazo_fases, v_plazo_desde, v_plazo_urgentes, v_plazo_numeros,
           v_total_plazos, v_sin_fase;
    -- Coherencia con los totales de la v4 (mismo corte y filtros): el
    -- recuento suma el total filtrado y los grupos de plazo suman los
    -- expedientes en trámite, todos con su entrada en fase. Si no cuadra, no
    -- se publica nada.
    IF v_total_recuento <> v_v4.total_filtrado
       OR v_total_plazos <> v_en_tramite
       OR v_sin_fase <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resumen de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY SELECT v_v4.contenido_canonico, v_v4.cursor_siguiente,
        v_v4.esquema, v_v4.acceso_ref, v_v4.secuencia,
        v_v4.anterior_sha256, v_v4.huella_sha256,
        v_v4.vinculo_identidad_huella_sha256,
        v_v4.alcance_huella_sha256, v_v4.registrada_en,
        v_v4.auditoria_vec_ref, v_v4.auditoria_vec_huella_sha256,
        v_v4.consumo_vec_huella_sha256, v_v4.contenido_huella_sha256,
        v_v4.resultado_huella_sha256, v_v4.cursor_huella_sha256,
        v_v4.generada_en, v_v4.expediente_ref, v_v4.version_expediente,
        v_v4.total, v_v4.recibo_sello_sha256, v_v4.total_filtrado,
        v_v4.en_tramitacion, v_v4.con_incidencia, v_v4.en_llamamiento,
        v_v4.fase_desde_expedientes, v_v4.fase_desde_instantes,
        v_v4.urgente_expedientes,
        v_estados, v_fases, v_numeros,
        v_plazo_fases, v_plazo_desde, v_plazo_urgentes, v_plazo_numeros;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1, text
) FROM PUBLIC;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMENT ON FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1, text
) IS
'Solo del propietario: agregados de la portada RRHH con el mismo corte y los mismos predicados que contar_totales_cuadro_rrhh_v1, sin referencias de expediente.';
COMMENT ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) IS
'Fachada v5 del cuadro RRHH: la v4 (misma autorización, consumo y auditoría) más el recuento por estado y fase y los grupos de plazo de los expedientes en trámite de todo el corte filtrado, sin referencias de expediente.';

COMMIT;
