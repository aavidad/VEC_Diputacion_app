\set ON_ERROR_STOP on
-- Las fachadas son consumidas por operaciones con recibos e historia.
DO $no_down$ BEGIN
 RAISE EXCEPTION 'Usuarios 000002: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
