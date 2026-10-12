\set ON_ERROR_STOP on
-- Misma forma legacy que las 170 instantáneas originales; sin reglas nuevas.
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
DO $guard$
BEGIN
  IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.fase_regla_instantanea_v1
             WHERE expediente_ref LIKE 'expediente:perf:%') THEN
    RAISE EXCEPTION 'clave=ct_instantanea_previa esperado=false observado=true';
  END IF;
END $guard$;
INSERT INTO vec_contratacion_temporal.fase_regla_instantanea_v1
 (expediente_ref,version,fase_clave,fase_desde,estado)
SELECT expediente_ref,version,fase_clave,fase_desde,'legado_sin_instantanea'
FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh
WHERE expediente_ref LIKE 'expediente:perf:%';
COMMIT;
ANALYZE vec_contratacion_temporal.fase_regla_instantanea_v1;
