\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $estructura$
DECLARE f oid; p text; a text;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_admin_perfiles_ejecutor',f,'EXECUTE') IS TRUE
 OR EXISTS(SELECT 1 FROM pg_proc q CROSS JOIN LATERAL aclexplode(coalesce(q.proacl,acldefault('f',q.proowner))) x WHERE q.oid=f AND x.grantee=0)
 THEN RAISE EXCEPTION 'AD236: ACL fachada insegura'; END IF;
 SELECT prosrc INTO p FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF (length(p)-length(replace(p,'actos_admin_perfiles','')))/length('actos_admin_perfiles')<>4
 OR strpos(p,'login_actos_admin_perfiles_valido_v1() IS TRUE')=0
 OR strpos(p,'p_perfil_mutacion IS DISTINCT FROM ''actos_admin_perfiles''')=0
 OR strpos(p,'administracion.perfiles.otorgar')=0
 OR strpos(p,'administracion.perfiles.revocar')=0
 OR strpos(p,'administracion.perfiles.proponer')=0
 OR strpos(p,'administracion.perfiles.aprobar')=0
 OR strpos(p,'administracion.perfiles.rechazar')=0
 THEN RAISE EXCEPTION 'AD236: política nominal incompleta'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO a FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated;
 IF a IS NULL OR strpos(a,'vec_autorizacion.administracion_perfiles.ordinario.v1')=0
 OR strpos(a,'vec_autorizacion.administracion_perfiles.propuesta.v1')=0
 OR strpos(a,'vec_autorizacion.administracion_perfiles.cierre.v1')=0
 THEN RAISE EXCEPTION 'AD236: audiencias ausentes'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
   WHERE audiencia_consumo IN ('vec_autorizacion.administracion_perfiles.ordinario.v1','vec_autorizacion.administracion_perfiles.propuesta.v1','vec_autorizacion.administracion_perfiles.cierre.v1'))
 THEN RAISE EXCEPTION 'AD236: creó claves sin gobierno'; END IF;
END $estructura$;
CREATE FUNCTION pg_temp.login_admin() RETURNS boolean LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT vec_autorizacion_atestada_v3.login_actos_admin_perfiles_valido_v1() $f$;
CREATE FUNCTION pg_temp.consumir_basura() RETURNS text LANGUAGE plpgsql SECURITY DEFINER AS $f$
BEGIN
 PERFORM * FROM vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(
  convert_to('{}','UTF8'),convert_to('{}','UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RETURN 'consumido';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;
END $f$;
CREATE ROLE prueba_ad236_admin LOGIN;
GRANT vec_admin_perfiles_ejecutor TO prueba_ad236_admin WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_ad236_ajeno LOGIN;
CREATE ROLE prueba_ad236_doble LOGIN;
GRANT vec_admin_perfiles_ejecutor TO prueba_ad236_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_admin_usuarios_lector TO prueba_ad236_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION prueba_ad236_admin;
SELECT pg_temp.login_admin() AS l_admin \gset
SELECT pg_temp.consumir_basura() AS c_admin \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad236_ajeno;
SELECT pg_temp.login_admin() AS l_ajeno \gset
SELECT pg_temp.consumir_basura() AS c_ajeno \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad236_doble;
SELECT pg_temp.login_admin() AS l_doble \gset
RESET SESSION AUTHORIZATION;
CREATE FUNCTION pg_temp.comprobar_resultado(l_admin boolean,l_ajeno boolean,l_doble boolean,c_admin text,c_ajeno text)
RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 IF NOT l_admin OR l_ajeno OR l_doble OR c_admin<>'42501' OR c_ajeno<>'42501'
 THEN RAISE EXCEPTION 'AD236: LOGIN o material inválido admitido'; END IF;
 RETURN 'AD236-OK';
END $f$;
SELECT pg_temp.comprobar_resultado(:'l_admin'::boolean,:'l_ajeno'::boolean,:'l_doble'::boolean,:'c_admin',:'c_ajeno');
ROLLBACK;
