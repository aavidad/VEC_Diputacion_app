DO $no_down$ BEGIN
    RAISE EXCEPTION 'ContextoActor 000010: DOWN no autorizado con historia potencial' USING ERRCODE = '55000';
END $no_down$;
