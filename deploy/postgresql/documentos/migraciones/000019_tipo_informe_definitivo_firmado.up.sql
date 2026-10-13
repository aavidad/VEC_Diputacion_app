\set ON_ERROR_STOP on
-- D19: el informe firmado conserva su propio tipo; no se etiqueta como resolución.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_documentos:migracion:000019',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_documentos.tipo_reservado_firmado
  WHERE tipo_ref='ref:f0074a505ef2a693dfd5bf2195228c47a7ce2a1435d4acf6a6b9f50e89e5663c '
   AND tipo='contratacion_temporal.resolucion_firmada.v1' AND modulo_id='contratacion_temporal') THEN
  RAISE EXCEPTION 'D19 clave=tipo_reservado_firmado esperado=resolucion_firmada.v1 actual=ausente_o_divergente' USING ERRCODE='55000';
 END IF;
END $pre$;
INSERT INTO vec_documentos.tipo_reservado_firmado(tipo_ref,tipo,modulo_id)
VALUES('ref:956a159e958e8881eaf47da9f84663748108e8aaff2108d59fe7bc8c1202f99a','contratacion_temporal.informe_definitivo_firmado.v1','contratacion_temporal');
COMMIT;
