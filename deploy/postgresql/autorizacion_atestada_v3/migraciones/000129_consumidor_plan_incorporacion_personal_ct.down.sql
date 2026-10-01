\set ON_ERROR_STOP on
-- No retirar un consumidor que pudo producir historia durable.
DO $no_down$ BEGIN
 RAISE EXCEPTION 'AD3-129: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
