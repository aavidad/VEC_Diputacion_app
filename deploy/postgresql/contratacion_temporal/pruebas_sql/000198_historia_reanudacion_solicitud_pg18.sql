\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $test$
DECLARE funcion oid:=to_regprocedure('vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 total integer; divergentes integer;
BEGIN
 IF funcion IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=funcion
   AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
   AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
   OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',funcion,'EXECUTE')
   OR has_function_privilege('public',funcion,'EXECUTE') THEN
  RAISE EXCEPTION 'CT198 prueba: autoridad SQL de recuperación divergente';
 END IF;
 SELECT count(*),count(*) FILTER (WHERE
     e.situacion NOT IN ('confirmada','indeterminada','propietaria')
     OR h.fencing_nuevo <> (o.carga_json->>'fencing_version')::bigint
     OR o.carga_json->>'clave_idempotencia' IS DISTINCT FROM h.clave_idempotencia::text
     OR o.creada_en IS DISTINCT FROM h.reanudada_en)
 INTO total,divergentes
 FROM vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento h
 JOIN vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento o USING(auditoria_ref)
 JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e USING(clave_idempotencia)
 WHERE o.tipo='seleccion.solicitud_llamamiento.reanudada';
 IF divergentes<>0 THEN
  RAISE EXCEPTION 'CT198 prueba: historia/outbox de solicitud actual=%/% esperado=0_divergencias',total,divergentes;
 END IF;
END
$test$;
ROLLBACK;
