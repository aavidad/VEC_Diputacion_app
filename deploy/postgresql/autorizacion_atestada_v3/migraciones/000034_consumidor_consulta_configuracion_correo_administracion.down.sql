\set ON_ERROR_STOP on
-- AD3-34 CANDIDATA. Mismo cierre que AD3-33: ninguna reversión automática
-- puede retirar la audiencia o el consumidor con gobierno/historia conservados.
-- No se consulta ni elimina historia ADMIN/T13 desde la autoridad AD3.
DO $rechazar_reversion$
BEGIN
    RAISE EXCEPTION 'AD3-34: reversión automática no admitida; conservar gobierno e historial'
        USING ERRCODE='55000';
END $rechazar_reversion$;
