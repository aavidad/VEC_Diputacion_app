\set ON_ERROR_STOP on
-- Sólo definición nueva y denegación sin autoridad; no publica Rol5.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $pruebas$
DECLARE cfg record;f record;grupo oid;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v5')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.config_mantenimiento_perfil_fijo_admin_v1)
 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=estructura actual=efecto_o_config_previo esperado=instalacion_sin_publicacion'; END IF;
 SELECT relowner,relrowsecurity,relforcerowsecurity INTO STRICT cfg FROM pg_class WHERE oid='vec_autorizacion.config_mantenimiento_perfil_fijo_admin_v1'::regclass;
 IF cfg.relowner<>'vec_autorizacion_propietario'::regrole OR NOT cfg.relrowsecurity OR NOT cfg.relforcerowsecurity
 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=configuracion actual=ACL_divergente esperado=owner_RLS_FORCE'; END IF;
 SELECT oid INTO STRICT grupo FROM pg_roles WHERE rolname='vec_admin_mantenimiento_fijo_ejecutor' AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls;
 FOR f IN SELECT p.* FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND (p.proname LIKE '%mantenimiento%fijo%v1' OR p.proname LIKE '%aut42' OR p.proname='validar_administrador_denominacion_persona_v1' OR p.proname IN('concesiones_denominacion_persona_admin_v1','concesiones_usuarios_admin_v1')) LOOP
  IF f.proowner<>'vec_autorizacion_propietario'::regrole OR EXISTS(SELECT 1 FROM aclexplode(COALESCE(f.proacl,acldefault('f',f.proowner))) x WHERE x.grantee=0 AND x.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=helper_ACL actual=divergente esperado=owner_sin_PUBLIC'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE x.grantee=grupo AND p.oid<>to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_admin_v1(text,text)'))
 OR NOT has_function_privilege(grupo,'vec_autorizacion.mantener_version_perfil_fijo_admin_v1(text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=LOGIN_grupo actual=divergente esperado=solo_wrapper'; END IF;
 IF encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(text,text,text,text,text,text,text,text,jsonb,jsonb)')),'UTF8')),'hex') IS DISTINCT FROM 'd88a2c5f24fc6f359e3230a2ed71f1f84272902d1dc268f401e97f5ce4f02ba5'
 OR encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(text,text,text)')),'UTF8')),'hex') IS DISTINCT FROM 'a84934dc423940c5e0e435ca4fca5ee430cfb8c8db02210d1868ac14d2ef2a6d'
 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=legado4 actual=fuente_alterada esperado=literal_AUT33'; END IF;
 IF jsonb_array_length(vec_autorizacion.concesiones_mantenimiento_fijo_admin_v1())<>4 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=catalogo_destino actual=divergente esperado=cuatro_concesiones_cerradas'; END IF;
 IF vec_autorizacion.validar_administrador_denominacion_persona_v1('{}','{}')
 THEN RAISE EXCEPTION 'AUT42 prueba: PARO clave=gate actual=permitido esperado=denegado_sin_autoridad'; END IF;
END $pruebas$;
ROLLBACK;
