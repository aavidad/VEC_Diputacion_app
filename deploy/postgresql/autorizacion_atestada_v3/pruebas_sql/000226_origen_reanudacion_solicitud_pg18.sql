\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $test$
DECLARE previo record; nuevo record;
BEGIN
 SELECT login_nombre,audiencia_consumo,proceso,canal_permitido INTO STRICT previo
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE operacion='contratacion_temporal.llamamiento.reanudar_orden';
 SELECT login_nombre,audiencia_consumo,proceso,canal_permitido INTO STRICT nuevo
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE operacion='contratacion_temporal.llamamiento.reanudar_solicitud';
 IF row(previo.login_nombre,previo.audiencia_consumo,previo.proceso,previo.canal_permitido)
    IS DISTINCT FROM row(nuevo.login_nombre,nuevo.audiencia_consumo,nuevo.proceso,nuevo.canal_permitido)
    OR nuevo.audiencia_consumo IS DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
    OR nuevo.canal_permitido IS DISTINCT FROM 'interna_corporativa' THEN
  RAISE EXCEPTION 'AD226 prueba: origen nuevo diverge del anterior';
 END IF;
END
$test$;
ROLLBACK;
