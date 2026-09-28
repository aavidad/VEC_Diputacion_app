\set ON_ERROR_STOP on
-- DOWN prohibido: restaurar EXECUTE abriría las fachadas sin organización.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'AD3-99: DOWN denegado; conservar consumos' USING ERRCODE='55000';
END $denegado$;
