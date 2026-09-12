\set ON_ERROR_STOP on
-- AD3-33 CANDIDATA. No hay reversión automática de gobierno/consumos.
-- Conserva la audiencia, su fachada y cualquier historial de configuración relacionado.
DO $rechazar_reversion$
BEGIN
    RAISE EXCEPTION 'AD3-33: reversión automática no admitida; conservar gobierno e historial'
        USING ERRCODE='55000';
END $rechazar_reversion$;
