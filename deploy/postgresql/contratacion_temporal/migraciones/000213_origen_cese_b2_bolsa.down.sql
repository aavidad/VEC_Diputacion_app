\set ON_ERROR_STOP on
-- CT213 altera la proyección de orígenes y puede tener ceses nativos
-- publicados. La reversión requiere una operación histórica específica.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'CT213: DOWN prohibido; conservar recibos y orígenes publicados'
  USING ERRCODE='55000';
END $denegado$;
