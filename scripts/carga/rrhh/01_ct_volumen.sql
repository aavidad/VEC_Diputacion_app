\set ON_ERROR_STOP on
-- Semilla CT187: toma el corte máximo de la copia restaurada.
SELECT coalesce(max(corte_global),0)::bigint AS base_ct_corte
FROM vec_contratacion_temporal.publicacion_version_rrhh \gset
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
WITH semilla AS (
  SELECT p.* FROM vec_contratacion_temporal.publicacion_version_rrhh p
  WHERE p.version = 1 AND p.fase_clave = 'solicitud' LIMIT 1
), filas AS (
  SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.publicacion_version_rrhh,
      to_jsonb(s) || jsonb_build_object(
      'expediente_ref', 'expediente:perf:' || lpad(g::text, 7, '0'),
      'numero_visible', '2026/CT-' || lpad(g::text, 7, '0'),
      'corte_global', :base_ct_corte + g,
      'actualizado_en', date_trunc('microseconds', now() - g * interval '1 millisecond'),
      'registrada_en', date_trunc('microseconds', now() + interval '1 second')
  ))).* FROM semilla s CROSS JOIN generate_series(1, 5000) g
)
INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh SELECT * FROM filas;
WITH semilla AS (
  SELECT f.* FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh f
  JOIN vec_contratacion_temporal.publicacion_version_rrhh p USING (expediente_ref, version)
  WHERE p.version = 1 AND p.fase_clave = 'solicitud' LIMIT 1
), filas AS (
  SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.fase_entrada_publicacion_rrhh,
      to_jsonb(s) || jsonb_build_object('expediente_ref', 'expediente:perf:' || lpad(g::text, 7, '0')))).*
  FROM semilla s CROSS JOIN generate_series(1, 5000) g
)
INSERT INTO vec_contratacion_temporal.fase_entrada_publicacion_rrhh SELECT * FROM filas;
UPDATE vec_contratacion_temporal.control_publicacion_rrhh
SET ultimo_corte = :base_ct_corte + 5000 WHERE control;
WITH semilla AS (
  SELECT r.* FROM vec_contratacion_temporal.peticion_centro_revision r WHERE version = 2 LIMIT 1
), filas AS (
  SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.peticion_centro_revision,
      to_jsonb(s) || jsonb_build_object(
      'peticion_ref', 'peticion:perf:' || lpad(g::text, 7, '0'),
      'peticion', s.peticion || jsonb_build_object('referencia', 'peticion:perf:' || lpad(g::text, 7, '0')),
      'recibo_ref', 'recibo:perf:' || lpad(g::text, 7, '0'),
      'recibo_json', s.recibo_json || jsonb_build_object('peticion_ref', 'peticion:perf:' || lpad(g::text, 7, '0'), 'recibo_ref', 'recibo:perf:' || lpad(g::text, 7, '0')),
      'clave_idempotencia', md5('clave:perf:' || g::text)::uuid,
      'auditoria_ref', 'auditoria:perf:' || lpad(g::text, 7, '0'),
      'decision_ref', 'decision:perf:' || lpad(g::text, 7, '0'),
      'consumo_huella_sha256', encode(sha256(convert_to('perf:' || g::text, 'UTF8')), 'hex'),
      'registrada_en', date_trunc('microseconds', now() - g * interval '1 millisecond')
  ))).* FROM semilla s CROSS JOIN generate_series(1, 5000) g
)
INSERT INTO vec_contratacion_temporal.peticion_centro_revision SELECT * FROM filas;
COMMIT;
ANALYZE vec_contratacion_temporal.publicacion_version_rrhh;
ANALYZE vec_contratacion_temporal.fase_entrada_publicacion_rrhh;
ANALYZE vec_contratacion_temporal.peticion_centro_revision;
