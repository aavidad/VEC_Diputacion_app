\set ON_ERROR_STOP on
-- Vector estructural tras instalar CA36 en el clon autorizado. Sin DML.
BEGIN;
DO $v$
DECLARE grupo oid:=to_regrole('vec_contexto_actor_v1_admin_contexto');
BEGIN
 IF grupo IS NULL OR to_regclass('vec_contexto_actor_v1.enlace_contexto_admin_v1') IS NULL
  OR NOT has_function_privilege(grupo,
   'vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()','EXECUTE')
  OR NOT has_function_privilege(grupo,
   'vec_contexto_actor_v1.registrar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)','EXECUTE')
  OR NOT has_function_privilege(grupo,
   'vec_contexto_actor_v1.recuperar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text,jsonb)','EXECUTE')
  OR NOT has_function_privilege(grupo,
   'vec_contexto_actor_v1.reconciliar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)','EXECUTE')
  OR has_function_privilege(grupo,
   'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])','EXECUTE')
  OR has_function_privilege(grupo,
   'vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])','EXECUTE')
  OR has_table_privilege(grupo,'vec_contexto_actor_v1.enlace_contexto_admin_v1',
   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
  OR has_any_column_privilege(grupo,'vec_contexto_actor_v1.enlace_contexto_admin_v1',
   'SELECT,INSERT,UPDATE,REFERENCES')
 THEN RAISE EXCEPTION 'CA36: runtime ADMIN no segregado' USING ERRCODE='55000'; END IF;
 IF (SELECT NOT c.relrowsecurity OR NOT c.relforcerowsecurity
   FROM pg_class c WHERE c.oid='vec_contexto_actor_v1.enlace_contexto_admin_v1'::regclass)
 THEN RAISE EXCEPTION 'CA36: enlace sin RLS forzada' USING ERRCODE='55000'; END IF;
END $v$;
ROLLBACK;
