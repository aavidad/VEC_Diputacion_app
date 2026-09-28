\set ON_ERROR_STOP on
-- DOWN prohibido: restauraría una fachada con menor ámbito y alteraría consumos.
DO $denegado$ BEGIN RAISE EXCEPTION 'CT-137: DOWN denegado; conservar historia y autorización' USING ERRCODE='55000'; END $denegado$;
