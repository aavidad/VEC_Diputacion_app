\set ON_ERROR_STOP on
-- Solo publicaciones sintéticas de este banco. La aplicación exige
-- creado_en <= fase_desde <= actualizado_en para cada versión de la lista.
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
UPDATE vec_contratacion_temporal.publicacion_version_rrhh p
SET creado_en = e.fase_desde
FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh e
WHERE p.expediente_ref = e.expediente_ref AND p.version = e.version
  AND p.expediente_ref LIKE 'expediente:perf:%'
  AND e.fase_desde < p.creado_en;
DO $guard$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM vec_contratacion_temporal.publicacion_version_rrhh p
    JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh e USING(expediente_ref,version)
    WHERE p.expediente_ref LIKE 'expediente:perf:%'
      AND (e.fase_desde < p.creado_en OR e.fase_desde > p.actualizado_en)
  ) THEN
    RAISE EXCEPTION 'clave=ct_cronologia_incoherente esperado=false observado=true';
  END IF;
END $guard$;
COMMIT;
ANALYZE vec_contratacion_temporal.publicacion_version_rrhh;
