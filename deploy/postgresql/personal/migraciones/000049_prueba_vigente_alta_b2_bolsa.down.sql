\set ON_ERROR_STOP on
-- P49 puede acreditar altas ya registradas en Bolsa; conservar ese vínculo.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'P49: DOWN prohibido con historia nativa posible'
  USING ERRCODE='55000';
END $denegado$;
