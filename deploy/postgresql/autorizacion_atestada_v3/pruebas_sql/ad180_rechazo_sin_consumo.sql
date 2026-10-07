\set ON_ERROR_STOP on
-- PREPARADA, NO EJECUTADA. Sólo clon sintético tras cerrar/ensayar AD180.
-- Verifica cierre por entrada nula antes del núcleo: no es prueba positiva
-- de V3, perfil, cortes o concurrencia. Esos casos requieren fixture nominal
-- real emitido por las autoridades, sin fabricar capacidades JSON.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $rechazo$
BEGIN
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'ad180.prueba: entrada nula admitida' USING ERRCODE='P0001';
 EXCEPTION WHEN insufficient_privilege THEN
  -- 42501 del contrato antes de leer/registrar la atestación común.
  NULL;
 END;
END $rechazo$;
ROLLBACK;
