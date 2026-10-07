\set ON_ERROR_STOP on
-- AD213 no revierte automáticamente la fachada: puede haber consumos y
-- auditoría nominal. AD214 pertenece a la autoridad común y no se toca aquí.
DO $rechazar_reversion$
BEGIN
 RAISE EXCEPTION 'AD213: reversión automática no admitida; conservar consumos y auditoría nominal' USING ERRCODE='55000';
END $rechazar_reversion$;
