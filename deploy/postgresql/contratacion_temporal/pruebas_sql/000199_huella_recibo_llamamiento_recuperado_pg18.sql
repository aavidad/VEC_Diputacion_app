\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $test$
DECLARE artefacto jsonb; anterior text[]; recuperado text[]; legado text[];
BEGIN
 SELECT artefacto_canonico::jsonb INTO artefacto
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE situacion='confirmada' AND artefacto_canonico IS NOT NULL
 ORDER BY clave_idempotencia LIMIT 1;
 IF artefacto IS NULL OR artefacto #> '{recibo,llamamiento_recuperado}' IS NOT NULL THEN
  RAISE EXCEPTION 'CT199 prueba: falta artefacto histórico sin marca';
 END IF;
 anterior := vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(artefacto);
 recuperado := vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
  jsonb_set(artefacto,'{recibo,llamamiento_recuperado}','true'::jsonb,true));
 legado := vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
  artefacto #- '{recibo,llamamiento_recuperado}');
 IF array_length(anterior,1) IS DISTINCT FROM 2 OR anterior[1] !~ '^[0-9a-f]{64}$'
    OR anterior[2] !~ '^[0-9a-f]{64}$' OR anterior IS DISTINCT FROM legado
    OR recuperado[1] IS DISTINCT FROM anterior[1]
    OR recuperado[2] IS NOT DISTINCT FROM anterior[2] THEN
  RAISE EXCEPTION 'CT199 prueba: petición alterada, legado cambiado o respuesta recuperada sin marca';
 END IF;
END
$test$;
ROLLBACK;
