\set ON_ERROR_STOP on
-- AD3-32 CANDIDATA. No hay reversión automática de gobierno/consumos.
-- Conserva la audiencia, su fachada y cualquier historial CT88 relacionado.
DO $rechazar_reversion$
BEGIN
    RAISE EXCEPTION 'AD3-32: reversión automática no admitida; conservar gobierno e historial'
        USING ERRCODE='55000';
END $rechazar_reversion$;
