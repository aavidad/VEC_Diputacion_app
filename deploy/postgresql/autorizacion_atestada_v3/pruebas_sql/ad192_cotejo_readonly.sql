\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL READ COMMITTED READ ONLY;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
DO $v$ DECLARE f oid;BEGIN
 FOREACH f IN ARRAY ARRAY[to_regprocedure('vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(jsonb)'),to_regprocedure('vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(jsonb)')] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND provolatile='s') OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND(x.grantee=0 OR x.is_grantable)) THEN RAISE EXCEPTION 'AD192 vector: cotejo_no_lectura_privada';END IF;
 END LOOP;
 IF has_function_privilege('vec_contexto_actor_v1_propietario','vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(jsonb)','EXECUTE') OR has_function_privilege('vec_identidad_sesiones_v1_propietario','vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(jsonb)','EXECUTE') THEN RAISE EXCEPTION 'AD192 vector: cotejo_autoridades_cruzadas';END IF;
END $v$;
ROLLBACK;
