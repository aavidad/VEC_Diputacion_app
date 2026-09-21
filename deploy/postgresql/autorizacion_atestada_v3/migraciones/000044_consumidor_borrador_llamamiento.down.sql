\set ON_ERROR_STOP on
DO $f$
BEGIN
 RAISE EXCEPTION 'AD3-44: reversión automática no admitida; conservar gobierno, consumo y auditoría' USING ERRCODE='55000';
END $f$;
