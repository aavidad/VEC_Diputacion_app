\set ON_ERROR_STOP on
DO $f$ BEGIN
 RAISE EXCEPTION 'AD3-47: DOWN no admitido; conservar AD3-46 y los consumos/auditoría B4' USING ERRCODE='55000';
END $f$;
