\set ON_ERROR_STOP on
-- Las consultas CT211 comparten el perfil nominal ya instalado de CT96 y
-- dejan historia de autorización indistinguible por número de migración.
-- La reversión de una instalación usada no puede demostrarse sin borrar esa
-- historia. La restauración operativa se hace desde el dump previo verificado.
DO $no_down$
BEGIN
 RAISE EXCEPTION 'CT211 DOWN no admitido con historia nominal posible'
  USING ERRCODE='55000';
END $no_down$;
