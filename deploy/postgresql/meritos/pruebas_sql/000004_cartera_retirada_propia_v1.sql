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
 OR to_regclass('vec_meritos.denegacion_frontera_cartera_v1') IS NULL
 OR to_regrole('vec_meritos_frontera_externo') IS NULL
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_frontera_externo'
  AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole
   OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'M4: falta estructura' USING ERRCODE='55000'; END IF;
 FOREACH t IN ARRAY ARRAY['disponibilidad_evento_v1','retirada_operacion_v1',
  'retirada_acceso_v1','retirada_outbox_v1','retirada_resultado_v1','cartera_acceso_v1'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_meritos.'||t)
   AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_trigger g WHERE g.tgrelid=to_regclass('vec_meritos.'||t)
   AND g.tgname='historia_inmutable' AND g.tgenabled='O')
  THEN RAISE EXCEPTION 'M4: tabla no inmutable o sin RLS %',t USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_class c
   WHERE c.oid='vec_meritos.denegacion_frontera_cartera_v1'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_trigger t
   WHERE t.tgrelid='vec_meritos.denegacion_frontera_cartera_v1'::regclass
   AND t.tgname='frontera_inmutable' AND t.tgenabled='O')
 THEN RAISE EXCEPTION 'M4: frontera mutable o sin RLS' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'M4: ACL de lectura divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_meritos.retirar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'M4: ACL de retirada divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_meritos.registrar_denegacion_frontera_cartera_v1(text,text,text,text,text)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_frontera_externo',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'M4: ACL de frontera divergente' USING ERRCODE='55000';END IF;
 BEGIN
  PERFORM vec_meritos.registrar_denegacion_frontera_cartera_v1(
   'corr_no_disponible','autenticacion_requerida',
   'api.meritos.cartera.ruta_exacta','/api/vec/meritos/cartera',NULL);
  RAISE EXCEPTION 'M4: emisor no nominal aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
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
