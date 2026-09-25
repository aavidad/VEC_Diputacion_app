\set ON_ERROR_STOP on
-- Calendarios 000003 no tiene vuelta atrás: restaurar el truncado silencioso no aporta nada.
DO $$ BEGIN RAISE EXCEPTION 'Calendarios 000003: sin retirada' USING ERRCODE='55000'; END $$;
