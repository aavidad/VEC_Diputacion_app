\set ON_ERROR_STOP on
-- DOWN prohibido: restauraría EXECUTE de una fachada con menor ámbito.
DO $denegado$ BEGIN RAISE EXCEPTION 'AD3-100: DOWN denegado; conservar consumos' USING ERRCODE='55000'; END $denegado$;
