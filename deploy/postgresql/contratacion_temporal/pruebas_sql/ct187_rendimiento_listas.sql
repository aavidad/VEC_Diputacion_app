\set ON_ERROR_STOP on
-- Ejecutar en dos copias iguales con CT184: una antes de CT187 y otra después.
-- La salida contiene solo huellas de página, totales y resumen; `diff` debe
-- ser vacío. Se usa el número anual existente para probar la unión nueva.
BEGIN READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
WITH datos AS (
    SELECT (SELECT organizacion_ref
              FROM vec_contratacion_temporal.publicacion_version_rrhh
             ORDER BY corte_global LIMIT 1) AS organizacion_ref,
           (SELECT centro_ref
              FROM vec_contratacion_temporal.publicacion_version_rrhh
             ORDER BY corte_global LIMIT 1) AS centro_ref,
           (SELECT numero_visible
              FROM vec_contratacion_temporal.numeracion_anual_asignada
             ORDER BY numero_visible LIMIT 1) AS numero_asignado,
           (SELECT ultimo_corte
              FROM vec_contratacion_temporal.control_publicacion_rrhh
             WHERE control) AS corte
), casos AS (
    SELECT 'organizacion'::text AS clase, organizacion_ref AS ambito_ref,
           ''::text AS texto, organizacion_ref, corte FROM datos
    UNION ALL
    SELECT 'centro', centro_ref, '', organizacion_ref, corte FROM datos
    UNION ALL
    SELECT 'organizacion', organizacion_ref, numero_asignado,
           organizacion_ref, corte FROM datos WHERE numero_asignado IS NOT NULL
)
SELECT casos.clase, CASE WHEN casos.texto = '' THEN 'todos' ELSE 'numero_anual' END,
       (SELECT pg_catalog.md5(pg_catalog.to_jsonb(t)::text)
          FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
              ROW(casos.organizacion_ref, casos.clase, casos.ambito_ref)::
                  vec_contratacion_temporal.alcance_consulta_rrhh_v1,
              ROW(casos.texto, '', '', 100, '')::
                  vec_contratacion_temporal.consulta_cuadro_rrhh_v1, '') t) AS totales_sha,
       (SELECT pg_catalog.md5(COALESCE(pg_catalog.string_agg(
                   pg_catalog.to_jsonb(r)::text, '|' ORDER BY
                   r.clase, r.estado_clave, r.fase_clave, r.fase_desde, r.urgente), ''))
          FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
              ROW(casos.organizacion_ref, casos.clase, casos.ambito_ref)::
                  vec_contratacion_temporal.alcance_consulta_rrhh_v1,
              ROW(casos.texto, '', '', 100, '')::
                  vec_contratacion_temporal.consulta_cuadro_rrhh_v1, '') r) AS resumen_sha,
       (SELECT pg_catalog.md5(COALESCE(pg_catalog.string_agg(
                   (m.resumenes[g.i]).expediente_ref || ':' ||
                   (m.resumenes[g.i]).numero_visible, '|' ORDER BY g.i), ''))
          FROM vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
              ROW(casos.organizacion_ref, casos.clase, casos.ambito_ref)::
                  vec_contratacion_temporal.alcance_consulta_rrhh_v1,
              ROW(casos.texto, '', '', 100, '')::
                  vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
              ROW(false, NULL, casos.corte, 0, NULL, NULL, NULL, NULL,
                  NULL, NULL, NULL)::
                  vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1) m
          CROSS JOIN LATERAL pg_catalog.generate_subscripts(m.resumenes, 1) g(i))
           AS pagina_sha
  FROM casos
 ORDER BY casos.clase, casos.texto;
SELECT 'peticiones' AS lectura,
       pg_catalog.md5(COALESCE(pg_catalog.jsonb_agg(x.fila ORDER BY
           x.registrada_en DESC, x.peticion_ref), '[]'::jsonb)::text) AS pagina_sha
  FROM (
    SELECT revision.peticion_ref, revision.registrada_en,
           CASE WHEN confirmacion.peticion_ref IS NOT NULL THEN
               pg_catalog.jsonb_build_object('peticion', revision.peticion,
                   'estado_entrega', 'confirmada', 'recibo_alta', confirmacion.recibo_alta)
           WHEN reserva.peticion_ref IS NOT NULL THEN
               pg_catalog.jsonb_build_object('peticion', revision.peticion,
                   'estado_entrega', 'preparada')
           ELSE pg_catalog.jsonb_build_object('peticion', revision.peticion,
                   'estado_entrega', 'pendiente') END AS fila
      FROM vec_contratacion_temporal.peticion_centro_revision revision
      LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_reserva reserva
        ON reserva.peticion_ref = revision.peticion_ref
      LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_confirmacion confirmacion
        ON confirmacion.peticion_ref = revision.peticion_ref
     WHERE revision.version = 2 AND revision.operacion = 'ratificar'
       AND revision.estado = 'ratificada'
     ORDER BY revision.registrada_en DESC, revision.peticion_ref LIMIT 50
  ) x;
-- Opcional: psql -v MEDIR_CT187=1 ejecuta 30 muestras calientes por lectura.
-- La medición es SQL interna; no atribuye latencia HTTP/autorización/auditoría.
\if :{?MEDIR_CT187}
DO $medida$
DECLARE
    v_organizacion text;
    v_corte numeric;
    v_inicio timestamptz;
    v_muestras double precision[];
    v_p50 double precision;
    v_p95 double precision;
    v_operacion text;
BEGIN
    SELECT organizacion_ref INTO STRICT v_organizacion
      FROM vec_contratacion_temporal.publicacion_version_rrhh
     ORDER BY corte_global LIMIT 1;
    SELECT ultimo_corte INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control;
    FOREACH v_operacion IN ARRAY ARRAY['pagina', 'totales', 'resumen', 'peticiones'] LOOP
        v_muestras := '{}';
        FOR i IN 1..31 LOOP
            v_inicio := pg_catalog.clock_timestamp();
            CASE v_operacion
            WHEN 'pagina' THEN
                PERFORM * FROM vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
                    ROW(v_organizacion, 'organizacion', v_organizacion)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
                    ROW('', '', '', 100, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
                    ROW(false, NULL, v_corte, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL)::
                        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1);
            WHEN 'totales' THEN
                PERFORM * FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
                    ROW(v_organizacion, 'organizacion', v_organizacion)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
                    ROW('', '', '', 100, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1, '');
            WHEN 'resumen' THEN
                PERFORM * FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
                    ROW(v_organizacion, 'organizacion', v_organizacion)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
                    ROW('', '', '', 100, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1, '');
            ELSE
                PERFORM COALESCE(pg_catalog.jsonb_agg(x.fila ORDER BY
                    x.registrada_en DESC, x.peticion_ref), '[]'::jsonb)
                  FROM (
                    SELECT revision.peticion_ref, revision.registrada_en,
                           CASE WHEN confirmacion.peticion_ref IS NOT NULL THEN
                               pg_catalog.jsonb_build_object('peticion', revision.peticion,
                                   'estado_entrega', 'confirmada', 'recibo_alta', confirmacion.recibo_alta)
                           WHEN reserva.peticion_ref IS NOT NULL THEN
                               pg_catalog.jsonb_build_object('peticion', revision.peticion,
                                   'estado_entrega', 'preparada')
                           ELSE pg_catalog.jsonb_build_object('peticion', revision.peticion,
                                   'estado_entrega', 'pendiente') END AS fila
                      FROM vec_contratacion_temporal.peticion_centro_revision revision
                      LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_reserva reserva
                        ON reserva.peticion_ref = revision.peticion_ref
                      LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_confirmacion confirmacion
                        ON confirmacion.peticion_ref = revision.peticion_ref
                     WHERE revision.version = 2 AND revision.operacion = 'ratificar'
                       AND revision.estado = 'ratificada'
                     ORDER BY revision.registrada_en DESC, revision.peticion_ref LIMIT 50
                  ) x;
            END CASE;
            IF i > 1 THEN
                v_muestras := pg_catalog.array_append(v_muestras,
                    extract(epoch FROM pg_catalog.clock_timestamp() - v_inicio) * 1000);
            END IF;
        END LOOP;
        SELECT pg_catalog.percentile_cont(0.5) WITHIN GROUP (ORDER BY x),
               pg_catalog.percentile_cont(0.95) WITHIN GROUP (ORDER BY x)
          INTO v_p50, v_p95 FROM pg_catalog.unnest(v_muestras) x;
        RAISE NOTICE '%: n=% p50=% ms p95=% ms', v_operacion,
            pg_catalog.cardinality(v_muestras),
            pg_catalog.round(v_p50::numeric, 2), pg_catalog.round(v_p95::numeric, 2);
    END LOOP;
END
$medida$;
\endif
ROLLBACK;
