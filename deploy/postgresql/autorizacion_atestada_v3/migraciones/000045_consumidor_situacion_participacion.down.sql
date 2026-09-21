\set ON_ERROR_STOP on
-- AD3-45 amplía el mismo núcleo compartido y puede tener consumos B2. La
-- reversión coherente conserva AD3-32/43/44 y rechaza retirar historia viva.
DO $f$ BEGIN
 RAISE EXCEPTION 'AD3-45: DOWN no admitido; conservar AD3-32/43/44, gobierno, consumo y auditoría B2' USING ERRCODE='55000';
END $f$;
