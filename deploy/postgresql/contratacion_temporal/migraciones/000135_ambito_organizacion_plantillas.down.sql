\set ON_ERROR_STOP on
-- DOWN prohibido: la columna puede custodiar recibos y la definición
-- anterior reabriría decisiones sin ámbito. La recuperación es forward.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'CT-135: DOWN denegado; conservar historia y autorización' USING ERRCODE='55000';
END $denegado$;
