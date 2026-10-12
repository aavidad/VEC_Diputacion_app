\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;
BEGIN
 IF to_regrole('vec_meritos_cartera_externo') IS NULL
 OR to_regrole('vec_autorizacion_fuente_meritos_externa') IS NULL
 OR to_regrole('vec_autorizacion_registro_meritos_externo') IS NULL
 OR to_regrole('vec_autorizacion_motivos_meritos_externo') IS NULL
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_cartera_externo'
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_fuente_meritos_externa'
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname IN(
  'vec_autorizacion_registro_meritos_externo','vec_autorizacion_motivos_meritos_externo')
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'AUT78: grupo técnico inseguro' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: ACL divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_fuente_meritos_externa',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: ACL de fuente divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: ACL de sesión divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
 OR vec_autorizacion.validar_motivo_retirada_meritos_actual_v1('{}'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT78: motivo libre o ACL divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(text)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: lector de motivos abierto o ausente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_registro_meritos_externo',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: registrador V3 abierto o ausente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.resolver_motivo_meritos_externo_v1(text,integer,text,text,timestamptz)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_motivos_meritos_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: motivo V3 abierto o ausente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: revalidador V3 abierto o ausente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR has_function_privilege('vec_autorizacion_registro_meritos_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: primitiva V3 de registro expuesta' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: primitiva V3 de revalidación expuesta' USING ERRCODE='55000';END IF;
 IF vec_autorizacion.cotejar_cartera_meritos_actual_v1('{}'::jsonb,'{}'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT78: concede orden vacía' USING ERRCODE='55000'; END IF;
END $test$;
ROLLBACK;
