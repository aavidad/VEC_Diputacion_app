\set ON_ERROR_STOP on
-- Ejecutar únicamente después del UP real en el clon sintético efímero PG18.
-- Esta puerta acredita estructura/ACL/denegación; el positivo V3 completo se
-- ejecuta con el generador central real de AD143, sin sustituir sus funciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $contratos$
DECLARE t text;f record;u oid:=to_regrole('vec_administracion_copias_ejecutor');
BEGIN
 IF u IS NULL THEN RAISE EXCEPTION 'test_cs08_runtime_ausente'; END IF;
 FOREACH t IN ARRAY ARRAY['propuesta','revision','control_destino','orden','auditoria','outbox'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_administracion_copias' AND c.relname=t AND c.relowner='vec_administracion_copias_propietario'::regrole
     AND c.relrowsecurity AND c.relforcerowsecurity) THEN RAISE EXCEPTION 'test_cs08_rls_ausente'; END IF;
  IF has_table_privilege(u,'vec_administracion_copias.'||t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
   RAISE EXCEPTION 'test_cs08_acceso_directo_runtime'; END IF;
  IF EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=to_regclass('vec_administracion_copias.'||t)
    AND p.polroles IS DISTINCT FROM ARRAY['vec_administracion_copias_propietario'::regrole::oid]) THEN
   RAISE EXCEPTION 'test_cs08_politica_publica'; END IF;
 END LOOP;
 FOR f IN SELECT p.* FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_administracion_copias' LOOP
  IF EXISTS(SELECT 1 FROM aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a WHERE a.grantee=0) THEN
   RAISE EXCEPTION 'test_cs08_funcion_publica'; END IF;
  IF NOT f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[] THEN RAISE EXCEPTION 'test_cs08_search_path'; END IF;
  IF has_function_privilege(u,f.oid,'EXECUTE') IS DISTINCT FROM
    (f.proname IN('registrar_propuesta_v1','registrar_revision_v1','comprometer_orden_v1','leer_orden_comprometida_v1')) THEN
   RAISE EXCEPTION 'test_cs08_funcion_runtime_incorrecta'; END IF;
 END LOOP;
 IF NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',
   'vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz)','EXECUTE')
 OR has_function_privilege(u,'vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz)','EXECUTE') THEN
  RAISE EXCEPTION 'test_cs08_fachada_doble_control_acl'; END IF;
END $contratos$;
SET LOCAL ROLE vec_administracion_copias_propietario;
DO $sin_doble_control$
BEGIN
 BEGIN
  PERFORM vec_administracion_copias.acreditar_doble_control_orden_v1('orden:sintetica:ausente',repeat('a',64),'persona:sintetica',now(),now()+interval '1 second');
  RAISE EXCEPTION 'test_cs08_orden_sin_aprobaciones_aceptada';
 EXCEPTION WHEN no_data_found OR insufficient_privilege THEN NULL;
 END;
END $sin_doble_control$;
ROLLBACK;
SELECT 'CS08_CONTRATOS_ACL_OK';
