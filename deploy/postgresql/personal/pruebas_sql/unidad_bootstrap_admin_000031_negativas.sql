\set ON_ERROR_STOP on
-- Sólo clon sintético. No crea una fuente positiva y todo queda en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $acl$
DECLARE f oid:=to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)');
BEGIN
 IF has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
 OR has_function_privilege('vec_contexto_actor_v1_propietario',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'ACL de unidad expuesta o AUT sin puerto'; END IF;
 IF has_table_privilege('vec_personal_ejecutor','vec_personal.control_unidad_bootstrap_admin_v1','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('vec_autorizacion_propietario','vec_personal.control_unidad_bootstrap_admin_v1','SELECT,INSERT,UPDATE,DELETE')
 OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_personal.org_nodo_historia'::regclass AND tgname='barrera_unidad_bootstrap_admin_v1' AND tgtype=6 AND tgenabled='O' AND tgfoid=to_regprocedure('vec_personal.avanzar_barrera_unidad_bootstrap_admin_v1()'))
 THEN RAISE EXCEPTION 'barrera expuesta o trigger distinto de BEFORE INSERT STATEMENT'; END IF;
END $acl$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $cerradas$
BEGIN
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1('org_no_acreditada','{"dimension":"unidad_ref","valores":["unidad_no_acreditada"],"fuente":{"referencia":"fuente_no_acreditada","version":1,"huella_sha256":"1111111111111111111111111111111111111111111111111111111111111111"}}'::jsonb,clock_timestamp()+interval '1 hour');
  RAISE EXCEPTION 'fuente inexistente admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1('org_no_acreditada','{"dimension":"unidad_ref","valores":["*"],"fuente":{"referencia":"fuente_no_acreditada","version":1,"huella_sha256":"1111111111111111111111111111111111111111111111111111111111111111"}}'::jsonb,clock_timestamp()+interval '1 hour');
  RAISE EXCEPTION 'comodín admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1('org_no_acreditada','{"dimension":"unidad_ref","valores":["unidad_no_acreditada"],"fuente":{"referencia":"fuente_no_acreditada","version":0,"huella_sha256":"1111111111111111111111111111111111111111111111111111111111111111"}}'::jsonb,clock_timestamp()+interval '1 hour');
  RAISE EXCEPTION 'versión cero admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $cerradas$;
RESET ROLE;
-- La ausencia del singleton no abre una vía sin coordinación.
DELETE FROM vec_personal.control_unidad_bootstrap_admin_v1;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $sin_guard$
BEGIN
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1('org_no_acreditada','{"dimension":"unidad_ref","valores":["unidad_no_acreditada"],"fuente":{"referencia":"fuente_no_acreditada","version":1,"huella_sha256":"1111111111111111111111111111111111111111111111111111111111111111"}}'::jsonb,clock_timestamp()+interval '1 hour');
  RAISE EXCEPTION 'singleton ausente admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $sin_guard$;
RESET ROLE;
ROLLBACK;
