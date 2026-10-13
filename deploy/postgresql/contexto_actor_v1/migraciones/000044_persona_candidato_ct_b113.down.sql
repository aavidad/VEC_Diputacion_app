\set ON_ERROR_STOP on
-- La guarda compartida de CTX18 puede haber acreditado lecturas CT; su
-- reversión no puede distinguir historia posterior sin borrar datos.
DO $no_down$
BEGIN
 RAISE EXCEPTION 'CA44 DOWN no admitido con historia nominal posible'
  USING ERRCODE='55000';
END $no_down$;
