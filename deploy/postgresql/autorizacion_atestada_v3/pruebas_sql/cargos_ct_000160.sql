\set ON_ERROR_STOP on
-- Comprobación estructural, no sustituye recorrido ni carrera PostgreSQL.
BEGIN;
DO $contrato$
DECLARE n text; f oid;
BEGIN
 FOREACH n IN ARRAY ARRAY['cargo_ct_plan','cargo_ct_aprobacion','cargo_ct_consumo','cargo_ct_auditoria','cargo_ct_recibo'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class WHERE oid=pg_catalog.to_regclass('vec_autorizacion.'||n)
   AND relowner='vec_autorizacion_propietario'::regrole AND relrowsecurity AND relforcerowsecurity)
   OR pg_catalog.has_table_privilege('vec_autorizacion_cargos_ct_ejecutor','vec_autorizacion.'||n,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
   OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE((SELECT typacl FROM pg_catalog.pg_type WHERE oid=pg_catalog.to_regtype('vec_autorizacion.'||n)),pg_catalog.acldefault('T','vec_autorizacion_propietario'::regrole))) WHERE grantee=0)
  THEN RAISE EXCEPTION 'contrato AD160 tabla/ACL no exacto %',n; END IF;
 END LOOP;
 FOREACH n IN ARRAY ARRAY['preparar','aprobar','aplicar','recuperar'] LOOP
  f:=pg_catalog.to_regprocedure('vec_autorizacion.'||n||'_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)');
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef AND provolatile='v' AND proconfig @> ARRAY['search_path=pg_catalog'])
   OR NOT pg_catalog.has_function_privilege('vec_autorizacion_cargos_ct_ejecutor',f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode((SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f)) WHERE grantee=0 OR is_grantable AND grantee<>'vec_autorizacion_propietario'::regrole)
  THEN RAISE EXCEPTION 'contrato AD160 fachada no exacta %',n; END IF;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario','vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)','EXECUTE')
  OR pg_catalog.has_function_privilege('vec_autorizacion_cargos_ct_ejecutor','vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)','EXECUTE')
  OR pg_catalog.has_function_privilege('vec_autorizacion_cargos_ct_ejecutor','vec_autorizacion.operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)','EXECUTE')
  OR pg_catalog.has_table_privilege('vec_autorizacion_cargos_ct_ejecutor','vec_autorizacion.asignacion_perfil','SELECT,INSERT,UPDATE,DELETE')
  OR pg_catalog.has_table_privilege('vec_autorizacion_cargos_ct_ejecutor','vec_autorizacion.version_rol','SELECT,INSERT,UPDATE,DELETE')
 THEN RAISE EXCEPTION 'contrato AD160 acceso técnico amplio'; END IF;
END $contrato$;
ROLLBACK;
