\set ON_ERROR_STOP on
-- La historia publicada no se retira: una corrección es una versión nueva.
-- Esta retirada falla siempre con 55000 y no modifica nada.
BEGIN;
DO $$ BEGIN
  RAISE EXCEPTION 'Calendarios 000002: la historia publicada no se retira; publique una versión sucesora' USING ERRCODE='55000';
END $$;
COMMIT;
