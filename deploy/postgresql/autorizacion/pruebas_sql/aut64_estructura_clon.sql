\set ON_ERROR_STOP on
-- AUT64: sonda de estructura en clon PostgreSQL 18 desechable.
-- Ejecutar tras los UP AUT59, AUT62, AD227, AUT63 y AUT64, sin publicar Rol9.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $verificar$
DECLARE f oid; g oid; c jsonb;
BEGIN
 IF to_regclass('vec_autorizacion.config_mantenimiento_version_bolsa_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion.registro_mantenimiento_version_bolsa_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_version_bolsa_admin_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'AUT64: estructura ausente'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v9')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.config_mantenimiento_version_bolsa_admin_v1)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.registro_mantenimiento_version_bolsa_admin_v1)
 THEN RAISE EXCEPTION 'AUT64: publicación/configuración al instalar'; END IF;
 c:=vec_autorizacion.concesiones_version_bolsa_admin_v1();
 IF c IS DISTINCT FROM vec_autorizacion.concesiones_version_rol_bolsa_v1()
 OR jsonb_array_length(c)<>2
 OR NOT (c @> '[{"accion":"administracion.perfiles.version_bolsa.proponer","modulo_id":"administracion","tipo_recurso":"definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"administracion.perfiles.version_bolsa.aprobar","modulo_id":"administracion","tipo_recurso":"propuesta_definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}]'::jsonb)
 THEN RAISE EXCEPTION 'AUT64: concesiones nominales divergentes'; END IF;
 SELECT oid INTO g FROM pg_roles WHERE rolname='vec_admin_mantenimiento_version_bolsa_ejecutor';
 f:=to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_version_bolsa_admin_v1(text,text)');
 IF g IS NULL OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=g AND (rolcanlogin OR rolinherit OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid=g)
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=g AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>g AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'AUT64: ACL fachada divergente'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname LIKE '%version_bolsa_admin_v1' AND p.prosecdef AND NOT ('search_path=pg_catalog, pg_temp'=ANY(p.proconfig)))
 THEN RAISE EXCEPTION 'AUT64: search_path SECURITY DEFINER divergente'; END IF;
END $verificar$;
ROLLBACK;
SELECT 'AUT64-ESTRUCTURA-OK';
