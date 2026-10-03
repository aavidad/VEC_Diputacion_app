\set ON_ERROR_STOP on
-- Prueba focal de postimagen. Ejecutar tras Personal28 y Personal29 en una
-- base desechable; no instala ni revierte migraciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid; propietario oid; aut oid;
BEGIN
 SELECT to_regprocedure('vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)')::oid
  INTO f;
 SELECT oid INTO propietario FROM pg_roles WHERE rolname='vec_personal_propietario';
 SELECT oid INTO aut FROM pg_roles WHERE rolname='vec_autorizacion_propietario';
 IF f IS NULL OR propietario IS NULL OR aut IS NULL THEN
  RAISE EXCEPTION 'Personal29: función o roles ausentes'; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE oid IN (propietario,aut) AND rolcanlogin) THEN
  RAISE EXCEPTION 'Personal29: propietario ejecutable como LOGIN'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario
  AND prosecdef AND provolatile='v' AND pronargs=5
  AND 'search_path=pg_catalog, pg_temp'=ANY(proconfig)
  AND 'lock_timeout=2s'=ANY(proconfig)) THEN
  RAISE EXCEPTION 'Personal29: dueño o barreras de función incorrectos'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p,
  LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) acl
  WHERE p.oid=f AND acl.privilege_type='EXECUTE'
   AND (acl.grantee=0 OR acl.grantee<>aut AND acl.grantee<>propietario)) THEN
  RAISE EXCEPTION 'Personal29: EXECUTE fuera de propietarios'; END IF;
 IF NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE') THEN
  RAISE EXCEPTION 'Personal29: ACL efectiva incompatible'; END IF;
 BEGIN
  PERFORM vec_personal.resolver_fuente_cargo_ocupante_ct_v1(NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'Personal29: contexto vacío aceptado';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT IN ('cargo_nominal_resolucion_denegada',
    'permission denied for function resolver_fuente_cargo_ocupante_ct_v1') THEN
   RAISE; END IF;
 END;
END $test$;
ROLLBACK;
