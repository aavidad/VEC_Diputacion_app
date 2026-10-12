\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;t text;
BEGIN
 IF to_regrole('vec_meritos_cartera_externo') IS NULL
 OR to_regclass('vec_meritos.disponibilidad_evento_v1') IS NULL
 OR to_regclass('vec_meritos.retirada_operacion_v1') IS NULL
 OR to_regclass('vec_meritos.retirada_acceso_v1') IS NULL
 OR to_regclass('vec_meritos.retirada_outbox_v1') IS NULL
 OR to_regclass('vec_meritos.retirada_resultado_v1') IS NULL
 OR to_regclass('vec_meritos.cartera_acceso_v1') IS NULL
 THEN RAISE EXCEPTION 'M4: falta estructura' USING ERRCODE='55000'; END IF;
 FOREACH t IN ARRAY ARRAY['disponibilidad_evento_v1','retirada_operacion_v1',
  'retirada_acceso_v1','retirada_outbox_v1','retirada_resultado_v1','cartera_acceso_v1'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_meritos.'||t)
   AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_trigger g WHERE g.tgrelid=to_regclass('vec_meritos.'||t)
   AND g.tgname='historia_inmutable' AND g.tgenabled='O')
  THEN RAISE EXCEPTION 'M4: tabla no inmutable o sin RLS %',t USING ERRCODE='55000'; END IF;
 END LOOP;
 f:=to_regprocedure('vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'M4: ACL de lectura divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_meritos.retirar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'M4: ACL de retirada divergente' USING ERRCODE='55000'; END IF;
 -- Ni un superusuario con una orden sin sesión nominal puede leer o retirar.
 BEGIN
  PERFORM vec_meritos.listar_cartera_propia_v1(NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'M4: lectura concedida sin autoridad' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_meritos.retirar_hecho_propio_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'M4: retirada concedida sin autoridad' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $test$;
ROLLBACK;
