SET ROLE cs06l_propietario;
CREATE OR REPLACE FUNCTION cs06l_sintetico.admite_codigo(codigo text)
RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT
AS $$ SELECT codigo = 'sintetico_v2' $$;
RESET ROLE;
