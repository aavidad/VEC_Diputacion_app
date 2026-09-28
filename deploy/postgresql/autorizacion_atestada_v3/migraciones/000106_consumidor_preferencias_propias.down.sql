\set ON_ERROR_STOP on
-- AD3-106 puede tener decisiones, accesos y consumos durables. Esta revisión
-- no define reversión; para un cambio posterior se emite nueva migración.
DO $no_down$ BEGIN
 RAISE EXCEPTION 'AD3-106: DOWN no autorizado con historia potencial' USING ERRCODE='55000';
END $no_down$;
