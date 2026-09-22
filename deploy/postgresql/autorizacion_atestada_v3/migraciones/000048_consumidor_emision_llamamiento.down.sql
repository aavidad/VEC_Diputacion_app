\set ON_ERROR_STOP on
DO $f$ BEGIN
 RAISE EXCEPTION 'AD3-48: DOWN no admitido; conservar concesiones, consumos y auditoría B7' USING ERRCODE='55000';
END $f$;
