\set ON_ERROR_STOP on
-- Ejecutar sólo en clon sintético, después AD171/CA31/IS14. Todo en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='10s';
CREATE ROLE vec_prueba_is14_admin LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_preperfil TO vec_prueba_is14_admin WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
INSERT INTO vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1(identidad_login,proceso,entorno,host_admin,audiencia,vigente_hasta)
VALUES('vec_prueba_is14_admin','admin-selector-test','desarrollo','admin.test.invalid','admin.selector.test',clock_timestamp()+interval '1 hour');
RESET ROLE;
SET SESSION AUTHORIZATION vec_prueba_is14_admin;
DO $frontera$
DECLARE ok boolean;
BEGIN
 SELECT acreditada INTO STRICT ok FROM vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1();
 IF ok IS DISTINCT FROM true THEN RAISE EXCEPTION 'LOGIN mínimo no acreditado'; END IF;
 IF has_function_privilege(current_user,'vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz)','EXECUTE')
 OR has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)','EXECUTE')
 OR has_function_privilege(current_user,'vec_contexto_actor_v1.seleccionar_admin_preperfil_propietaria_v1(text,text,text,text,numeric,text)','EXECUTE')
 OR has_table_privilege(current_user,'vec_identidad_sesiones_v1.observacion_admin_preperfil_v1','SELECT,INSERT,UPDATE,DELETE')
 THEN RAISE EXCEPTION 'LOGIN puede fabricar evidencia o cambiar selección sin fachada'; END IF;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1('desarrollo','admin.ajeno.invalid','admin.selector.test',repeat('1',64),repeat('2',64),clock_timestamp(),clock_timestamp(),clock_timestamp()+interval '1 hour',clock_timestamp()+interval '1 hour','evento_'||repeat('a',32),'correlacion_'||repeat('b',32));
  RAISE EXCEPTION 'host ajeno admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1('desarrollo','admin.test.invalid','admin.selector.test',repeat('1',64),repeat('2',64),clock_timestamp(),clock_timestamp(),clock_timestamp()+interval '1 hour',clock_timestamp()+interval '1 hour','evento_'||repeat('a',32),'correlacion_'||repeat('b',32));
  RAISE EXCEPTION 'identidad no acreditada admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $frontera$;
RESET SESSION AUTHORIZATION;
-- Una ampliación posterior del rol técnico invalida la acreditación.
GRANT SELECT ON TABLE vec_identidad_sesiones_v1.observacion_admin_preperfil_v1 TO vec_identidad_sesiones_v1_admin_preperfil;
SET SESSION AUTHORIZATION vec_prueba_is14_admin;
DO $acl_veneno$
BEGIN
 BEGIN
  PERFORM vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1();
  RAISE EXCEPTION 'ACL ampliada admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl_veneno$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
