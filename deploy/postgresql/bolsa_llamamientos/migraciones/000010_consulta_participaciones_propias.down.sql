\set ON_ERROR_STOP on
DO $rechazar_reversion$
BEGIN
 RAISE EXCEPTION 'Bolsa-10: reversión automática no admitida; conservar consumos y auditoría V3' USING ERRCODE='55000';
END $rechazar_reversion$;
