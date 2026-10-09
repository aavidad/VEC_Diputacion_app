\set ON_ERROR_STOP on
-- Sonda focal sobre copia desechable PostgreSQL 18 con AUT64 instalado.
-- AUT67 puede faltar: AUT69 instala su estructura y deniega el efecto.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $verificar$
DECLARE f oid;g oid;rol64 oid;firma text;funciones text[]:=ARRAY[
 'vec_autorizacion.concesiones_version_bolsa_post_inscripcion_admin_v1()',
 'vec_autorizacion.exigir_operador_version_bolsa_post_inscripcion_admin_v1()',
 'vec_autorizacion.documento_asignacion_destino_version_bolsa_post_inscripcion_v1(jsonb,jsonb,text)',
 'vec_autorizacion.exigir_procedencia_inscripcion_bolsa_post_v1(jsonb,text)',
 'vec_autorizacion.preimagen_version_bolsa_post_inscripcion_admin_v1(jsonb)',
 'vec_autorizacion.aplicar_mantenimiento_version_bolsa_post_inscripcion_admin_v1(text,text)',
 'vec_autorizacion.mantener_version_bolsa_post_inscripcion_admin_v1(text,text)'];
BEGIN
 IF to_regclass('vec_autorizacion.config_version_bolsa_post_inscripcion_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion.registro_version_bolsa_post_inscripcion_admin_v1') IS NULL
 THEN RAISE EXCEPTION 'AUT69: estructura ausente'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v10')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.config_version_bolsa_post_inscripcion_admin_v1)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.registro_version_bolsa_post_inscripcion_admin_v1)
 THEN RAISE EXCEPTION 'AUT69: efecto o configuración al instalar'; END IF;
 IF vec_autorizacion.concesiones_version_bolsa_post_inscripcion_admin_v1() IS DISTINCT FROM vec_autorizacion.concesiones_version_bolsa_admin_v1()
 OR jsonb_array_length(vec_autorizacion.concesiones_version_bolsa_post_inscripcion_admin_v1())<>2
 THEN RAISE EXCEPTION 'AUT69: concesiones divergentes de AUT64'; END IF;
 SELECT oid INTO g FROM pg_roles WHERE rolname='vec_admin_mantenimiento_bolsa_post_inscripcion_ejecutor';
 SELECT oid INTO rol64 FROM pg_roles WHERE rolname='vec_admin_mantenimiento_version_bolsa_ejecutor';
 f:=to_regprocedure('vec_autorizacion.mantener_version_bolsa_post_inscripcion_admin_v1(text,text)');
 IF g IS NULL OR f IS NULL OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=g AND (rolcanlogin OR rolinherit OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid=g)
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=g AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>g AND a.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=rol64)
 THEN RAISE EXCEPTION 'AUT69: ACL fachada divergente'; END IF;
 FOREACH firma IN ARRAY funciones LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prosecdef AND NOT ('search_path=pg_catalog, pg_temp'=ANY(p.proconfig)))
  THEN RAISE EXCEPTION 'AUT69: función o search_path ausente %',firma; END IF;
 END LOOP;
END $verificar$;
ROLLBACK;
SELECT 'AUT69-ESTRUCTURA-OK';

-- Ejecutar este bloque únicamente cuando AUT67 aún no esté instalada.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $cerrado$
BEGIN
 IF to_regclass('vec_autorizacion.registro_mantenimiento_version_inscripcion_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT69: prueba sin AUT67 sobre base con AUT67'; END IF;
 BEGIN
  PERFORM vec_autorizacion.exigir_procedencia_inscripcion_bolsa_post_v1('{}'::jsonb,repeat('0',64));
  RAISE EXCEPTION 'AUT69: AUT67 ausente fue admitida';
 EXCEPTION WHEN SQLSTATE '40001' THEN
  NULL;
 END;
END $cerrado$;
ROLLBACK;
SELECT 'AUT69-AUX-SIN-AUT67-PARO-40001';
