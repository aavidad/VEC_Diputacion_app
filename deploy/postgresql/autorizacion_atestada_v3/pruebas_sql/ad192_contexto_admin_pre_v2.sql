\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
DO $v$ DECLARE f oid;BEGIN
 FOREACH f IN ARRAY ARRAY[to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb)'),to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb)')] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog']) OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND(x.grantee=0 OR x.is_grantable)) THEN RAISE EXCEPTION 'AD192 vector: owner_acl_divergentes';END IF;
 END LOOP;
 IF NOT has_function_privilege('vec_identidad_sesiones_v1_propietario','vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb)','EXECUTE') OR has_function_privilege('vec_identidad_sesiones_v1_propietario','vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb)','EXECUTE') OR NOT has_function_privilege('vec_contexto_actor_v1_propietario','vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb)','EXECUTE') OR has_function_privilege('vec_contexto_actor_v1_propietario','vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb)','EXECUTE') THEN RAISE EXCEPTION 'AD192 vector: autoridades_cruzadas';END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1) THEN RAISE EXCEPTION 'AD192 vector: configuracion_favorable_sembrada';END IF;
 IF jsonb_array_length(vec_autorizacion_atestada_v3.motivos_contexto_admin_pre_v2_v1())<>12 THEN RAISE EXCEPTION 'AD192 vector: catalogo_divergente';END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1('{}');
  RAISE EXCEPTION 'AD192 vector: invocacion_sin_LOGIN_admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $v$;
ROLLBACK;
