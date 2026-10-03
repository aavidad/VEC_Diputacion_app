\set ON_ERROR_STOP on
-- Comprobación estructural sin altas. Ejecutar sólo sobre un clon desechable
-- después de AD166 y Personal28; la transacción no conserva efectos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f oid; p record; n text;
BEGIN
 f:=to_regprocedure('vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'Personal28: lector ausente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_personal_propietario'::regrole OR NOT p.prosecdef
  OR p.provolatile<>'v' OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee NOT IN (p.proowner,'vec_autorizacion_propietario'::regrole))
  OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE') THEN
  RAISE EXCEPTION 'Personal28: ACL del lector divergente'; END IF;
 f:=to_regprocedure('vec_personal.publicar_cargo_competencial_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p,
    LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee=0) THEN
  RAISE EXCEPTION 'Personal28: fachada de publicación divergente'; END IF;
 FOREACH n IN ARRAY ARRAY['cargo_competencial_historia','cargo_competencial_actual',
  'enlace_cargo_competencial_historia','enlace_cargo_competencial_actual','recibo_publicacion_cargo_competencial'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_personal.'||n)
    AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
   OR has_table_privilege('vec_personal_ejecutor','vec_personal.'||n,'SELECT') THEN
   RAISE EXCEPTION 'Personal28: tabla expuesta %',n; END IF;
 END LOOP;
END $test$;
ROLLBACK;
