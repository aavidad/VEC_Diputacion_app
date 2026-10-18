\set ON_ERROR_STOP on
-- F1-69: la continuación conserva la selección original del expediente.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000238',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_class c
   WHERE c.oid=to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6')
     AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
 OR to_regclass('vec_contratacion_temporal.ejecucion_seleccion_oferta_unica_expediente_ct220') IS NULL
 OR to_regclass('vec_contratacion_temporal.seleccion_confirmada_unica_expediente_ct238') IS NOT NULL
 THEN RAISE EXCEPTION 'CT238: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
-- Impide nuevas escrituras entre la comprobación y la creación del índice.
LOCK TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 IN SHARE MODE;
DO $datos$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
   WHERE situacion='confirmada' AND (
     coalesce(solicitud_json->>'organizacion_ref','')='' OR
     coalesce(solicitud_json->>'expediente_ref','')=''))
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
   WHERE situacion='confirmada'
   GROUP BY solicitud_json->>'organizacion_ref',solicitud_json->>'expediente_ref'
   HAVING count(*)>1)
 THEN RAISE EXCEPTION 'CT238: selecciones confirmadas requieren comprobación'
   USING ERRCODE='55000'; END IF;
END $datos$;
-- El mismo índice sirve al predicado de la consulta CT231 por ambos campos JSON.
CREATE UNIQUE INDEX seleccion_confirmada_unica_expediente_ct238
 ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 ((solicitud_json->>'organizacion_ref'),(solicitud_json->>'expediente_ref'))
 WHERE situacion='confirmada';
COMMIT;
