\set ON_ERROR_STOP on
-- CT-133 DOWN únicamente en ensayo vacío. Con historia publicada, la lectura
-- documental sigue siendo parte de la recuperación y no se retira.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000133',0));
DO $vacia$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1)
 THEN RAISE EXCEPTION 'CT-133: DOWN denegado con historia o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $vacia$;
DROP FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1();
COMMIT;
