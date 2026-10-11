\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE f oid;h text;antes bigint;despues bigint;codigo text;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(bytea,bytea,jsonb)');
 IF f IS NULL OR to_regprocedure('vec_autorizacion.cotejar_lectura_respuestas_llamamiento_actual_v1(jsonb,text,text)') IS NULL
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',
  'vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE') IS TRUE
 OR has_function_privilege('public',f,'EXECUTE') IS TRUE
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'lock_timeout=2s','statement_timeout=10s'])
 THEN RAISE EXCEPTION 'AD237: ACL o definidor de lectura incorrectos'; END IF;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO h
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF h IS DISTINCT FROM 'af22c823fde6cf5df7722908c465c68e2b912ff260e34ee9152d609e38969b2a'
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_lectura_respuestas_llamamiento_formato_v1' AND c.convalidated)
 THEN RAISE EXCEPTION 'AD237: familia auditora divergente'; END IF;
 SELECT count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF vec_autorizacion_atestada_v3.login_lectura_respuestas_llamamiento_valido_v1() IS NOT FALSE
 OR vec_autorizacion.cotejar_lectura_respuestas_llamamiento_actual_v1(
    '{}'::jsonb,'per_'||repeat('a',22),'prf_'||repeat('a',22)) IS NOT FALSE
 THEN RAISE EXCEPTION 'AD237: autoridad sintética sin concesión admitida'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(NULL,NULL,NULL);
  RAISE EXCEPTION 'AD237: lectura sin vínculo admitida';
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  IF codigo<>'42501' THEN RAISE; END IF;
 END;
 SELECT count(*) INTO despues FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF despues<>antes THEN RAISE EXCEPTION 'AD237: lectura denegada alteró auditoría'; END IF;
END $prueba$;
ROLLBACK;
