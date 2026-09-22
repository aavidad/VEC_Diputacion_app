\set ON_ERROR_STOP on
DO $f$ BEGIN
 RAISE EXCEPTION 'AD3-46: DOWN no admitido; conservar AD3-45 y los consumos/auditoría B3' USING ERRCODE='55000';
END $f$;
