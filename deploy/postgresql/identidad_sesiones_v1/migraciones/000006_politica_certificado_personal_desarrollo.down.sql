-- La politica puede sustentar historia; no revertir automaticamente.
DO $$ BEGIN RAISE EXCEPTION 'politica temporal con historia: DOWN prohibido' USING ERRCODE='55000'; END $$;
