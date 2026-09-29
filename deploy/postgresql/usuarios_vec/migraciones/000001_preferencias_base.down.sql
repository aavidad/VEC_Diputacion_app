\set ON_ERROR_STOP on
-- La definición publicada y las referencias no se eliminan por DOWN.
DO $no_down$ BEGIN
 RAISE EXCEPTION 'Usuarios 000001: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
