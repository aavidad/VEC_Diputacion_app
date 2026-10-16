\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid:='vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)'::regprocedure;
 s text;
BEGIN
 s:=pg_get_functiondef(f);
 IF strpos(s,$q$p_orden->'campos_permitidos' ? 'evidencias'$q$)=0 OR
  strpos(s,$q$ELSE '{}'::jsonb END ORDER BY$q$)=0 OR
  strpos(s,$q$jsonb_build_object('evidencias',coalesce(nullif(p.hecho->'evidencias','null'::jsonb),'[]'::jsonb))$q$)=0 OR
  strpos(s,'registrar_lectura_cartera_meritos_v1')=0 OR
  strpos(s,'vec_meritos.persona_ref')=0 OR
  NOT has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE') OR
  EXISTS(SELECT 1 FROM aclexplode((SELECT proacl FROM pg_proc WHERE oid=f)) a
   WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
 THEN RAISE EXCEPTION 'RUM7: proyección o frontera divergente'; END IF;
END $test$;
ROLLBACK;
