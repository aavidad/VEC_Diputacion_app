\set ON_ERROR_STOP on
DO $rechazar_reversion$
BEGIN
 RAISE EXCEPTION 'AD3-43: reversión automática no admitida; conservar consumo, auditoría y gobierno' USING ERRCODE='55000';
END $rechazar_reversion$;
