\set ON_ERROR_STOP on
-- La operación CT155 modificada comparte contrato y recibos con la historia
-- anterior. No hay reversión segura tras una consulta, plan u origen nativo.
DO $no_down$
BEGIN
 RAISE EXCEPTION 'CT212 DOWN no admitido con historia nominal posible'
  USING ERRCODE='55000';
END $no_down$;
