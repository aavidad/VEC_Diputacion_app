\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000107', 0
    )
);

-- Estadísticas de contratación por periodo (C18). Agregados sin datos
-- personales sobre las versiones publicadas para RRHH hasta el último corte
-- global, con el mismo alcance (organización / centro / unidad) que el cuadro.
-- Cada expediente cuenta una sola vez por concepto, en el periodo de su
-- primera versión en esa situación; las incidencias cuentan cada entrada en
-- el estado `incidencia`. Los periodos se cortan en Europe/Madrid.
CREATE FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_periodo text,
    p_desde date,
    p_hasta date
)
RETURNS TABLE(
    corte_global numeric,
    inicio date,
    altas numeric,
    llamamientos numeric,
    formalizaciones numeric,
    cierres numeric,
    incidencias numeric
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
    v_unidad text;
    v_paso interval;
    v_dias integer;
    v_inicio date;
    v_fin date;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'estadísticas RRHH no disponibles';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    IF p_periodo IS NULL OR p_desde IS NULL OR p_hasta IS NULL
       OR p_periodo NOT IN ('anual', 'mensual', 'semanal')
       OR p_desde > p_hasta
       OR p_desde < DATE '2000-01-01' OR p_hasta > DATE '2100-12-31' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta de estadísticas RRHH inválida';
    END IF;
    CASE p_periodo
        WHEN 'anual' THEN v_unidad := 'year'; v_paso := INTERVAL '1 year'; v_dias := 365;
        WHEN 'mensual' THEN v_unidad := 'month'; v_paso := INTERVAL '1 month'; v_dias := 28;
        ELSE v_unidad := 'week'; v_paso := INTERVAL '1 week'; v_dias := 7;
    END CASE;
    v_inicio := pg_catalog.date_trunc(v_unidad, p_desde::timestamp)::date;
    v_fin := pg_catalog.date_trunc(v_unidad, p_hasta::timestamp)::date;
    IF (v_fin - v_inicio) / v_dias > 400 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'intervalo de estadísticas RRHH demasiado amplio';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    -- Mantener el predicado de alcance alineado con
    -- contar_totales_cuadro_rrhh_v1 (000105).
    RETURN QUERY
    WITH versiones AS MATERIALIZED (
        SELECT publicada.expediente_ref, publicada.version,
               publicada.fase_clave, publicada.estado_clave,
               publicada.creado_en, publicada.actualizado_en,
               pg_catalog.lag(publicada.estado_clave) OVER (
                   PARTITION BY publicada.expediente_ref COLLATE "C"
                   ORDER BY publicada.version
               ) AS estado_anterior
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
           AND publicada.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND publicada.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND publicada.unidad_ref IS NOT NULL AND publicada.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
    ), hitos AS MATERIALIZED (
        SELECT 'alta' AS concepto, pg_catalog.min(v.creado_en) AS instante
          FROM versiones v GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'llamamiento', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE 'llamamiento' = v.fase_clave GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'formalizacion', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE 'nombramiento' = v.fase_clave GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'cierre', pg_catalog.min(v.actualizado_en)
          FROM versiones v WHERE v.estado_clave = ANY (ARRAY['completado', 'cancelado']) GROUP BY v.expediente_ref
        UNION ALL
        SELECT 'incidencia', v.actualizado_en
          FROM versiones v
         WHERE 'incidencia' = v.estado_clave
           AND (v.estado_anterior IS NULL OR v.estado_anterior <> 'incidencia')
    ), periodos AS (
        SELECT pg_catalog.date_trunc(v_unidad, serie)::date AS inicio
          FROM pg_catalog.generate_series(v_inicio::timestamp, v_fin::timestamp, v_paso) AS serie
    ), agrupados AS (
        SELECT pg_catalog.date_trunc(v_unidad, h.instante AT TIME ZONE 'Europe/Madrid')::date AS inicio,
               h.concepto, pg_catalog.count(*) AS cuenta
          FROM hitos h
         WHERE h.instante IS NOT NULL
         GROUP BY 1, 2
    )
    SELECT v_corte_global,
           p.inicio,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'alta'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'llamamiento'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'formalizacion'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'cierre'), 0)::numeric,
           COALESCE(pg_catalog.sum(a.cuenta) FILTER (WHERE a.concepto = 'incidencia'), 0)::numeric
      FROM periodos p
      LEFT JOIN agrupados a ON a.inicio = p.inicio
     GROUP BY p.inicio
     ORDER BY p.inicio;
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1, text, date, date
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1, text, date, date
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1, text, date, date
) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
