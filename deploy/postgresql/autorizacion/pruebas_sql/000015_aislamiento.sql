\set ON_ERROR_STOP on
DO $prueba$
DECLARE grupo text; firma text;
BEGIN
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
      'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex')
      <> '45c7708252254d24df761419fe80fbfd00749cb0f5b47e5221631fd70b5d3e3f'
 THEN RAISE EXCEPTION 'AUT-15 alteró el registrador interno'; END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid IN (
     'vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass,
     'vec_autorizacion.decision_denegada_contexto_actor_v3_externa'::regclass)
     AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
 THEN RAISE EXCEPTION 'AUT-15: RLS externa incompleta'; END IF;
 IF EXISTS (SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa)
    OR EXISTS (SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa)
 THEN RAISE EXCEPTION 'AUT-15: las tablas nuevas no comienzan vacías'; END IF;
 IF pg_catalog.strpos(pg_catalog.pg_get_functiondef(
      'vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure),
      'vec_autorizacion.revalidar_sesion_vinculo_v2(') <> 0
    OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(
      'vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure),
      'vec_autorizacion.revalidar_sesion_vinculo_v2(') <> 0
 THEN RAISE EXCEPTION 'AUT-15: uso de sesión compartida'; END IF;
 FOREACH grupo IN ARRAY ARRAY['vec_autorizacion_fuente_externa','vec_autorizacion_motivos_externos','vec_autorizacion_registro_externo'] LOOP
  IF pg_catalog.has_table_privilege(grupo,'vec_autorizacion.decision_concedida_contexto_actor_v3_externa','SELECT')
     OR pg_catalog.has_table_privilege(grupo,'vec_autorizacion.decision_denegada_contexto_actor_v3_externa','SELECT')
     OR pg_catalog.has_function_privilege(grupo,'vec_autorizacion.obtener_instantanea(text,text)','EXECUTE')
     OR pg_catalog.has_function_privilege(grupo,'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)','EXECUTE')
     OR pg_catalog.has_function_privilege(grupo,'vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)','EXECUTE')
     OR pg_catalog.has_function_privilege(grupo,'vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','EXECUTE')
     OR pg_catalog.has_function_privilege(grupo,'vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','EXECUTE')
  THEN RAISE EXCEPTION 'AUT-15: permiso general o tabla para %',grupo; END IF;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege('vec_autorizacion_fuente_externa','vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_motivos_externos','vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_registro_externo','vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','EXECUTE')
 THEN RAISE EXCEPTION 'AUT-15: fachada nominal ausente'; END IF;
END $prueba$;
