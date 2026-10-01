\set ON_ERROR_STOP on
DO $no_down$ BEGIN
 RAISE EXCEPTION 'Personal24: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
