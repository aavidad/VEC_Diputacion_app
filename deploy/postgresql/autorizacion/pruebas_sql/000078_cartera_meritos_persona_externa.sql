\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;cuerpo text;rama_denegada text;antes bigint;despues bigint;
BEGIN
 IF to_regrole('vec_meritos_cartera_externo') IS NULL
 OR to_regrole('vec_autorizacion_fuente_meritos_externa') IS NULL
 OR to_regrole('vec_autorizacion_registro_meritos_externo') IS NULL
 OR to_regrole('vec_autorizacion_motivos_meritos_externo') IS NULL
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_cartera_externo'
  AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_fuente_meritos_externa'
  AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname IN(
  'vec_autorizacion_registro_meritos_externo','vec_autorizacion_motivos_meritos_externo')
  AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
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
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: registrador V3 abierto o ausente' USING ERRCODE='55000';END IF;
 SELECT p.prosrc INTO cuerpo FROM pg_proc p WHERE p.oid=f;
 rama_denegada:=substring(cuerpo FROM 'IF registro[.]concedida IS FALSE THEN(.*?)END IF;');
 IF rama_denegada IS NULL
 OR strpos(rama_denegada,'RETURN QUERY SELECT false,registro.codigo,registro.decision_huella_sha256,')=0
 OR strpos(rama_denegada,'registro.registrada_en,NULL::timestamptz;')=0
 OR strpos(rama_denegada,'RETURN;')=0
 OR strpos(rama_denegada,'revalidar')>0
 OR strpos(rama_denegada,'emitir')>0
 OR strpos(rama_denegada,'consumir')>0
 OR strpos(cuerpo,'IF registro.concedida IS FALSE THEN')>=strpos(cuerpo,
  'viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(')
 OR strpos(cuerpo,'IF viva IS NULL THEN RETURN;END IF;')=0
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prosecdef
  AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 THEN RAISE EXCEPTION 'AUT78: denegación sin fila durable o revalidación abierta' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)');
 SELECT p.prosrc INTO cuerpo FROM pg_proc p WHERE p.oid=f;
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR strpos(cuerpo,'vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE')=0
 OR strpos(cuerpo,'d->>''accion'' IS DISTINCT FROM ''meritos.cartera.retirar_propio''')=0
 OR strpos(cuerpo,'d->>''tipo_recurso'' IS DISTINCT FROM ''hecho''')=0
 OR strpos(cuerpo,'d->>''finalidad'' IS DISTINCT FROM ''gestion_cartera_propia''')=0
 OR strpos(cuerpo,'vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(')=0
 OR strpos(cuerpo,'r.rol_id=''persona_meritos_cartera''')=0
 OR strpos(cuerpo,'IF d->>''concedida''=''true'' THEN')=0
 OR strpos(cuerpo,'d->''campos_permitidos'' IS NOT DISTINCT FROM ''["disponibilidad","recibo"]''::jsonb')=0
 OR strpos(cuerpo,'d->''obligaciones'' IS NOT DISTINCT FROM ''["auditar"]''::jsonb')=0
 OR strpos(cuerpo,'RETURN d->>''concedida''=''false''')=0
 OR strpos(cuerpo,'d->''campos_permitidos'' <@ ''["disponibilidad","recibo"]''::jsonb')=0
 OR strpos(cuerpo,'d->''obligaciones'' <@ ''["auditar"]''::jsonb')=0
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prosecdef
  AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 THEN RAISE EXCEPTION 'AUT78: contrato positivo o negativo divergente' USING ERRCODE='55000';END IF;
 -- La ausencia de sesión nominal no debe registrar ninguna decisión,
 -- aunque el material pretenda declarar una denegación.
 SELECT count(*) INTO antes FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa;
 BEGIN
  PERFORM vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
   convert_to('{"concedida":false,"campos_permitidos":[],"obligaciones":[]}','UTF8'),
   convert_to('{}','UTF8'),1,1);
  RAISE EXCEPTION 'AUT78: denegación registrada sin sesión nominal' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 SELECT count(*) INTO despues FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa;
 IF despues<>antes
 THEN RAISE EXCEPTION 'AUT78: denegación sin autoridad produjo un asiento' USING ERRCODE='55000';END IF;
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
