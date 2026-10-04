\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
DO $vector$ DECLARE f oid;g oid;BEGIN
 g:='vec_admin_mantenimiento_lote_ejecutor'::regrole;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v6') OR EXISTS(SELECT 1 FROM vec_autorizacion.config_mantenimiento_lote_admin_v1) THEN RAISE EXCEPTION 'AUT45 vector: estructura_publico_datos';END IF;
 IF (SELECT rolcanlogin OR rolinherit OR rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls FROM pg_roles WHERE oid=g) THEN RAISE EXCEPTION 'AUT45 vector: grupo_no_minimo';END IF;
 FOREACH f IN ARRAY ARRAY[to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_lote_admin_v1(text,text)'),to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)')] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog']) OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND(x.grantee=0 OR x.is_grantable)) THEN RAISE EXCEPTION 'AUT45 vector: frontera_owner_acl';END IF;
 END LOOP;
 IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:administracion_perfiles:v5','','','','administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]','{}') IS NOT FALSE OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:administracion_perfiles:v7','','','','administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]','{}') IS NOT FALSE THEN RAISE EXCEPTION 'AUT45 vector: version_no_cerrada';END IF;
 IF NOT COALESCE((SELECT proconfig @> ARRAY['search_path=pg_catalog','TimeZone=UTC','row_security=on'] AND cardinality(proconfig)=3 FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)')),false) THEN RAISE EXCEPTION 'AUT45 vector: gate_usuarios_config_divergente';END IF;
 IF jsonb_array_length(vec_autorizacion.concesiones_lote_ordinario_admin_v1())<>1 OR vec_autorizacion.concesiones_lote_ordinario_admin_v1()#>'{0,campos_permitidos}' IS DISTINCT FROM '[]'::jsonb OR vec_autorizacion.concesiones_lote_ordinario_admin_v1()#>'{0,obligaciones}' IS DISTINCT FROM '["auditar"]'::jsonb THEN RAISE EXCEPTION 'AUT45 vector: catalogo_no_cerrado';END IF;
END $vector$;
ROLLBACK;
