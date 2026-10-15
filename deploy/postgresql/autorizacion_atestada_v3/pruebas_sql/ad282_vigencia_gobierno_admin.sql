\set ON_ERROR_STOP on
-- Consulta posterior a AD282: funciones exactas y fronteras de ejecución.
DO $test$
DECLARE f regprocedure; esperado text; actual text;
BEGIN
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'31fa5026f10d19ce278d4bd50a956bc963117f7fcbed93043c793a21f516837d'),
  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'b6a18055d19c5b12c25bdf7d01cce72be6dfabb14e29d4e4311ef82f17da36b8')
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
