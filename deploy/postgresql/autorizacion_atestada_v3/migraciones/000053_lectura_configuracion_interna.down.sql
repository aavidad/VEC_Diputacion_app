\set ON_ERROR_STOP on
DO $f$ BEGIN
    RAISE EXCEPTION 'AD3-53: DOWN no admitido con historia' USING ERRCODE = '55000';
END $f$;
