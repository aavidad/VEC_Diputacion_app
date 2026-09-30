\set ON_ERROR_STOP on
-- PG18 desechable tras #178 y AUT17/18/AD3-118. No instala correctivas.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='60s';
CREATE ROLE prueba_usuarios_tipos LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_registro_externo TO prueba_usuarios_tipos WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
DO $temp$ BEGIN EXECUTE pg_catalog.format('GRANT TEMP ON DATABASE %I TO prueba_usuarios_tipos',pg_catalog.current_database()); END $temp$;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_tipos;
CREATE TEMP SEQUENCE contador_tipos MINVALUE 0 START 1;
SELECT pg_catalog.setval('pg_temp.contador_tipos'::pg_catalog.regclass,0,true);
CREATE FUNCTION pg_temp.sonda_tipo() RETURNS pg_catalog.bool LANGUAGE plpgsql AS $sonda$
BEGIN
 IF current_user IN('vec_autorizacion_propietario','vec_autorizacion_atestada_v3_propietario') THEN
  PERFORM pg_catalog.nextval('pg_temp.contador_tipos'::pg_catalog.regclass);
  RAISE EXCEPTION 'USUARIOS-TIPOS: CHECK temporal ejecutado por propietario' USING ERRCODE='PT021';
 END IF;
 RETURN true;
END $sonda$;
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.oid AS pg_catalog.oid CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.regrole AS pg_catalog.regrole CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK(pg_temp.sonda_tipo());
GRANT USAGE,SELECT ON SEQUENCE pg_temp.contador_tipos TO vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION pg_temp.sonda_tipo() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pg_temp.sonda_tipo() TO vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario;
RESET SESSION AUTHORIZATION;
DO $no_temp$ BEGIN EXECUTE pg_catalog.format('REVOKE TEMP ON DATABASE %I FROM prueba_usuarios_tipos',pg_catalog.current_database()); END $no_temp$;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text) TO prueba_usuarios_tipos;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_tipos;
DO $antes$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.publicar_rol_usuarios_externo_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 EXCEPTION WHEN OTHERS THEN NULL; END;
 IF (SELECT last_value FROM pg_temp.contador_tipos)=0 THEN RAISE EXCEPTION 'USUARIOS-TIPOS: falta control vulnerable'; END IF;
 PERFORM pg_catalog.setval('pg_temp.contador_tipos'::pg_catalog.regclass,0,true);
END $antes$;
RESET SESSION AUTHORIZATION;
REVOKE EXECUTE ON FUNCTION vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text) FROM prueba_usuarios_tipos;
-- USUARIOS-CORRECTIVOS-AQUI
CREATE TEMP TABLE llamadas_usuarios(firma pg_catalog.text,consulta pg_catalog.text);
INSERT INTO pg_temp.llamadas_usuarios VALUES
  ('vec_autorizacion.obtener_instantanea_usuarios_externo_v1(text,text)','SELECT vec_autorizacion.obtener_instantanea_usuarios_externo_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()','SELECT vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()'),
  ('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)','SELECT vec_autorizacion.publicar_asignacion_usuarios_externo_v1(NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','SELECT vec_autorizacion.publicar_rol_usuarios_externo_v1(NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.numeric,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_decision_usuarios_externo_v3(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.resolver_motivo_usuarios_externo_v1(text,integer,text,text,timestamp with time zone)','SELECT vec_autorizacion.resolver_motivo_usuarios_externo_v1(NULL::pg_catalog.text,NULL::pg_catalog.int4,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','SELECT vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2(NULL::pg_catalog.jsonb,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz)'),
  ('vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb)','SELECT vec_autorizacion.rol_usuarios_externo_acotado_v1(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)');
DO $permisos_fixture$
DECLARE f pg_catalog.text;
BEGIN
 FOR f IN SELECT firma FROM pg_temp.llamadas_usuarios LOOP
  EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO prueba_usuarios_tipos',f);
 END LOOP;
END $permisos_fixture$;
GRANT SELECT ON TABLE pg_temp.llamadas_usuarios TO prueba_usuarios_tipos;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_tipos;
DO $despues$
DECLARE consulta pg_catalog.text;
BEGIN
 FOR consulta IN SELECT l.consulta FROM pg_temp.llamadas_usuarios l LOOP
  BEGIN EXECUTE consulta;
  EXCEPTION WHEN SQLSTATE 'PT021' THEN RAISE;
   WHEN OTHERS THEN NULL;
  END;
 END LOOP;
 IF (SELECT last_value FROM pg_temp.contador_tipos)<>0 THEN RAISE EXCEPTION 'USUARIOS-TIPOS: propietario alcanzó dominio temporal'; END IF;
END $despues$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo USUARIOS-TIPOS-CIERRE-OK
