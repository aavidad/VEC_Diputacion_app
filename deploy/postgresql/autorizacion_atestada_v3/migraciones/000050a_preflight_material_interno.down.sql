\set ON_ERROR_STOP on
-- La sonda pertenece al corte con historia AD3. No se revierte sobre datos conservados.
DO $f$ BEGIN
    RAISE EXCEPTION 'AD3-50a: DOWN no admitido con historia' USING ERRCODE = '55000';
END $f$;
