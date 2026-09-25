\set ON_ERROR_STOP on
DO $f$ BEGIN
 RAISE EXCEPTION 'AUT14: DOWN no admitido con asignaciones/decisiones durables'
   USING ERRCODE='55000';
END $f$;
