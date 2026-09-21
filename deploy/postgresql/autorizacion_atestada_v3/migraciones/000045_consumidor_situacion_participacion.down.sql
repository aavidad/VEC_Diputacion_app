\set ON_ERROR_STOP on
DO $f$ BEGIN
 RAISE EXCEPTION 'AD3-45: reversión automática no admitida; conservar gobierno, consumo y auditoría B2' USING ERRCODE='55000';
END $f$;
