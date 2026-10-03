\set ON_ERROR_STOP on
-- Sólo clon sintético, tras la cadena causal cerrada AD175 -> Personal32.
-- No cambia roles, datos ni configuración. Ejecutar por el canal de verificación.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $acl$
DECLARE
 f oid:=to_regprocedure('vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_exportacion_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 tabla oid:='vec_personal.recibo_ficha_propia_empleado'::regclass;
 rol text;
BEGIN
 IF f IS NULL OR consumidor IS NULL THEN RAISE EXCEPTION 'prueba: falta exportación nominal'; END IF;
 IF NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',consumidor,'EXECUTE')
    OR has_function_privilege('vec_personal_registrador_frontera',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid IN (f,consumidor) AND a.grantee=0) THEN
  RAISE EXCEPTION 'prueba: ACL de exportación abierta o incompleta'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_personal_ejecutor','vec_personal_registrador_frontera'] LOOP
  IF has_table_privilege(rol,tabla,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
     OR has_column_privilege(rol,tabla,'vigente_en','SELECT,INSERT,UPDATE')
     OR has_column_privilege(rol,tabla,'conocido_en','SELECT,INSERT,UPDATE') THEN
   RAISE EXCEPTION 'prueba: corte accesible por rol runtime'; END IF;
 END LOOP;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=tabla AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner='vec_personal_propietario'::regrole)
    OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=tabla AND NOT t.tgisinternal
      AND t.tgname IN ('historia_inmutable','no_truncar') AND t.tgenabled='O')<>2 THEN
  RAISE EXCEPTION 'prueba: recibo perdió su protección'; END IF;
END $acl$;
ROLLBACK;
