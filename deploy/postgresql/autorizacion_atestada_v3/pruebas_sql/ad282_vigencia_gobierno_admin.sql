\set ON_ERROR_STOP on
-- Consulta posterior a AD282: funciones exactas y fronteras de ejecución.
DO $test$
DECLARE f regprocedure; esperado text; actual text;
BEGIN
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'7a2c53d075e5c45a7455f2e8175498b13e8c82e120e08fcd9cdc79194d3c73a3'),
  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'8330b0ad391ce73d62fc3f2781b2ada7094d1e61d4b340afca6ba68dd769639f')
 ) AS v(f,h) LOOP
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
  IF actual IS DISTINCT FROM esperado THEN
   RAISE EXCEPTION 'AD282 prueba: función distinta %',f;
  END IF;
  IF (SELECT p.proowner::regrole::text='vec_autorizacion_atestada_v3_propietario'
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']
      AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM true THEN
   RAISE EXCEPTION 'AD282 prueba: frontera de función distinta %',f;
  END IF;
 END LOOP;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid
  WHERE c.oid='vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1'::regclass
  AND pg_get_constraintdef(k.oid)='CHECK ((entorno = ''desarrollo''::text))')
 OR NOT EXISTS (SELECT 1 FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid
  WHERE c.oid='vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1'::regclass
  AND pg_get_constraintdef(k.oid)='CHECK ((entorno = ''desarrollo''::text))') THEN
  RAISE EXCEPTION 'AD282 prueba: entorno nominal distinto';
 END IF;
END $test$;
