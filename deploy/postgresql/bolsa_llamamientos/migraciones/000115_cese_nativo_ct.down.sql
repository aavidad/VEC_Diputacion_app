\set ON_ERROR_STOP on
-- B115 crea contratos, situaciones, restricciones y recibos inmutables.
-- Su reversión exige preservar antes la historia nativa y sus consumidores.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'B115: DOWN prohibido; conservar contratos y ceses nativos'
  USING ERRCODE='55000';
END $denegado$;
