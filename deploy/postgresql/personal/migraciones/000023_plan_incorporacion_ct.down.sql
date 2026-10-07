\set ON_ERROR_STOP on
-- El plan reserva claves originales y puede enlazar efectos de B2.
-- La retirada necesita conservar esa historia por un procedimiento específico.
DO $no_down$ BEGIN
 RAISE EXCEPTION 'Personal23: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
