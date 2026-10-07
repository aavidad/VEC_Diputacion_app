\set ON_ERROR_STOP on
-- Sólo clon: configuración y LOGIN sintéticos, sin personas/control fabricados.
-- Prueba la frontera técnica; no acredita un bootstrap permitido.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
SELECT to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)')::oid AS bootstrap_oid,
       to_regprocedure('vec_autorizacion.canon_bootstrap_central_admin_v3(jsonb,text)')::oid AS canon_oid,
       to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)')::oid AS auditoria_oid,
       to_regclass('vec_autorizacion.config_bootstrap_central_admin_v3')::oid AS configuracion_oid
\gset
CREATE ROLE vec_prueba_bootstrap37 LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_admin_bootstrap_central_v3_ejecutor TO vec_prueba_bootstrap37 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
DO $sin_config$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.preflight_bootstrap_central_admin_v3();
  RAISE EXCEPTION 'AUT37 prueba: admitió operador sin configuración';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $sin_config$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.config_bootstrap_central_admin_v3(
 login_nombre,proceso,plan_sha256,preimagen_sha256,aprobacion_ref,aprobacion_sha256,
 fuente_reparto,fuente_identidad,fuente_ca_admin,vigente_desde,vigente_hasta)
VALUES('vec_prueba_bootstrap37','bootstrap-sintetico',repeat('1',64),repeat('2',64),
 'aprobacion:fixture:bootstrap37',repeat('3',64),
 jsonb_build_object('referencia','fixture:reparto','version',1,'huella_sha256',repeat('4',64)),
 jsonb_build_object('referencia','fixture:identidad','version',1,'huella_sha256',repeat('5',64)),
 jsonb_build_object('referencia','fixture:ca','version',1,'huella_sha256',repeat('6',64)),
 clock_timestamp()-interval '1 second',clock_timestamp()+interval '1 minute');
RESET ROLE;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
SELECT 1/(CASE WHEN NOT has_function_privilege(current_user,:bootstrap_oid::oid,'EXECUTE')
 OR has_function_privilege(current_user,:canon_oid::oid,'EXECUTE')
 OR has_function_privilege(current_user,:auditoria_oid::oid,'EXECUTE')
 OR has_table_privilege(current_user,:configuracion_oid::oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 THEN 0 ELSE 1 END) AS privilegios_minimos;
DO $frontera$
BEGIN
 IF vec_autorizacion.preflight_bootstrap_central_admin_v3() IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT37 prueba: rechazó LOGIN técnico mínimo'; END IF;
 BEGIN
  PERFORM vec_autorizacion.registrar_bootstrap_central_admin_v3('{}',repeat('1',64));
  RAISE EXCEPTION 'AUT37 prueba: admitió plan sin ABI/fuentes';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $frontera$;
RESET SESSION AUTHORIZATION;
-- Mismo objeto permitido, privilegio distinto: no debe pasar por pg_shdepend.
GRANT CREATE ON SCHEMA vec_autorizacion TO vec_admin_bootstrap_central_v3_ejecutor;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
DO $schema_create$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.preflight_bootstrap_central_admin_v3();
  RAISE EXCEPTION 'AUT37 prueba: admitió CREATE sobre esquema permitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $schema_create$;
RESET SESSION AUTHORIZATION;
REVOKE CREATE ON SCHEMA vec_autorizacion FROM vec_admin_bootstrap_central_v3_ejecutor;
DO $db_temporal$
BEGIN
 EXECUTE format('GRANT TEMP ON DATABASE %I TO vec_admin_bootstrap_central_v3_ejecutor',current_database());
END $db_temporal$;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
DO $db_temp$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.preflight_bootstrap_central_admin_v3();
  RAISE EXCEPTION 'AUT37 prueba: admitió TEMP sobre base permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $db_temp$;
RESET SESSION AUTHORIZATION;
DO $db_retirar$
BEGIN
 EXECUTE format('REVOKE TEMP ON DATABASE %I FROM vec_admin_bootstrap_central_v3_ejecutor',current_database());
END $db_retirar$;
GRANT EXECUTE ON FUNCTION vec_autorizacion.preflight_bootstrap_central_admin_v3()
 TO vec_admin_bootstrap_central_v3_ejecutor WITH GRANT OPTION;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
DO $exec_grant_option$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.preflight_bootstrap_central_admin_v3();
  RAISE EXCEPTION 'AUT37 prueba: admitió GRANT OPTION en función permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $exec_grant_option$;
RESET SESSION AUTHORIZATION;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion.preflight_bootstrap_central_admin_v3()
 FROM vec_admin_bootstrap_central_v3_ejecutor;
-- Una capacidad adicional al grupo invalida su frontera, incluso si es lectura.
GRANT EXECUTE ON FUNCTION vec_autorizacion.canon_bootstrap_central_admin_v3(jsonb,text)
 TO vec_admin_bootstrap_central_v3_ejecutor;
SET SESSION AUTHORIZATION vec_prueba_bootstrap37;
DO $acl_ampliada$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.preflight_bootstrap_central_admin_v3();
  RAISE EXCEPTION 'AUT37 prueba: admitió ACL ampliada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl_ampliada$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
