\set ON_ERROR_STOP on
-- 500 marcas de urgencia de proyección, solo en el banco desechable.
-- No son nuevos expedientes integrales ni un análisis RRHH real.
SELECT ultimo_corte::bigint AS ct_seed_corte
FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control \gset
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
DO $guard$
BEGIN
  IF (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref LIKE 'expediente:perf:%' AND version=1) <> 5000
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref LIKE 'expediente:perf:%' AND version=2) THEN
    RAISE EXCEPTION 'clave=ct_urgencias esperado=v1:5000,v2:0 observado=v1:%,v2:%', (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref LIKE 'expediente:perf:%' AND version=1), (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref LIKE 'expediente:perf:%' AND version=2);
  END IF;
END $guard$;
WITH filas AS (
  SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.publicacion_version_rrhh,
    to_jsonb(p) || jsonb_build_object(
      'version',2,
      'corte_global',:ct_seed_corte + right(p.expediente_ref,7)::integer / 10,
      'actualizado_en',date_trunc('microseconds',now()),
      'registrada_en',date_trunc('microseconds',now() + interval '1 second')
    ))).* FROM vec_contratacion_temporal.publicacion_version_rrhh p
  WHERE p.expediente_ref LIKE 'expediente:perf:%'
    AND p.version=1 AND right(p.expediente_ref,7)::integer % 10 = 0
)
INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh SELECT * FROM filas;
WITH filas AS (
  SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.fase_entrada_publicacion_rrhh,
    to_jsonb(e) || '{"version":2}'::jsonb)).*
  FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh e
  WHERE e.expediente_ref LIKE 'expediente:perf:%'
    AND e.version=1 AND right(e.expediente_ref,7)::integer % 10 = 0
)
INSERT INTO vec_contratacion_temporal.fase_entrada_publicacion_rrhh SELECT * FROM filas;
INSERT INTO vec_contratacion_temporal.urgencia_expediente_analisis
 (expediente_ref,version,recibo_ref,motivo,registrada_en)
SELECT p.expediente_ref,2,
       'recibo:rendimiento:urgencia:' || right(p.expediente_ref,7),
       'Marca sintética para filtro de rendimiento',
       date_trunc('microseconds',now())
FROM vec_contratacion_temporal.publicacion_version_rrhh p
WHERE p.expediente_ref LIKE 'expediente:perf:%' AND p.version=2;
UPDATE vec_contratacion_temporal.control_publicacion_rrhh
SET ultimo_corte = :ct_seed_corte + 500 WHERE control;
COMMIT;
ANALYZE vec_contratacion_temporal.publicacion_version_rrhh;
ANALYZE vec_contratacion_temporal.fase_entrada_publicacion_rrhh;
ANALYZE vec_contratacion_temporal.urgencia_expediente_analisis;
