\set ON_ERROR_STOP on
-- Como AD3-49: el núcleo modificado no se revierte con historia de consumos.
DO $f$ BEGIN RAISE EXCEPTION 'AD3-51: DOWN no admitido con historia' USING ERRCODE='55000'; END $f$;
