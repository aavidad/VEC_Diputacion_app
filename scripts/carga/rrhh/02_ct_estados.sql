\set ON_ERROR_STOP on
-- Exclusivamente la copia PostgreSQL desechable del banco. Proyecciones sintéticas,
-- sin nuevos expedientes integrales ni recibos de actuaciones.
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
DO $guard$
BEGIN
  IF (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref LIKE 'expediente:perf:%') <> 5000 THEN
    RAISE EXCEPTION 'clave=ct_semilla esperado=5000 observado=%', (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref LIKE 'expediente:perf:%');
  END IF;
END $guard$;
WITH catalogo AS (
  SELECT array_agg(DISTINCT centro_ref ORDER BY centro_ref) AS centros,
         array_agg(DISTINCT categoria_ref ORDER BY categoria_ref) AS categorias
  FROM vec_contratacion_temporal.publicacion_version_rrhh
  WHERE expediente_ref NOT LIKE 'expediente:perf:%'
), filas AS (
  SELECT p.expediente_ref,
         right(p.expediente_ref, 7)::integer AS n
  FROM vec_contratacion_temporal.publicacion_version_rrhh p
  WHERE p.expediente_ref LIKE 'expediente:perf:%'
)
UPDATE vec_contratacion_temporal.publicacion_version_rrhh p
SET fase_clave = CASE f.n % 6
    WHEN 0 THEN 'solicitud'
    WHEN 1 THEN 'asignacion_unidad'
    WHEN 2 THEN 'fiscalizacion'
    WHEN 3 THEN 'informe_juridico'
    WHEN 4 THEN 'subsanacion_unidad'
    ELSE 'nombramiento' END,
    estado_clave = CASE WHEN f.n % 6 = 4 THEN 'incidencia'
                        WHEN f.n % 11 = 0 THEN 'espera_externa'
                        ELSE 'en_curso' END,
    centro_ref = c.centros[1 + (f.n % array_length(c.centros, 1))],
    categoria_ref = c.categorias[1 + (f.n % array_length(c.categorias, 1))]
FROM filas f CROSS JOIN catalogo c
WHERE p.expediente_ref = f.expediente_ref AND p.version = 1;
UPDATE vec_contratacion_temporal.fase_entrada_publicacion_rrhh e
SET fase_clave = p.fase_clave,
    fase_desde = date_trunc('microseconds', now() - ((right(e.expediente_ref, 7)::integer % 120) + 1) * interval '1 day')
FROM vec_contratacion_temporal.publicacion_version_rrhh p
WHERE e.expediente_ref = p.expediente_ref AND e.version = p.version
  AND e.expediente_ref LIKE 'expediente:perf:%';
COMMIT;
ANALYZE vec_contratacion_temporal.publicacion_version_rrhh;
ANALYZE vec_contratacion_temporal.fase_entrada_publicacion_rrhh;
