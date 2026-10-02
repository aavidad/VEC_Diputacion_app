-- Sonda focal para clon efímero con AUT24+CA23 e IS12; no sustituye un replay nominal.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $aislamiento$
DECLARE n integer; cuerpo text; continuidad integer; operacion integer; lector integer;
BEGIN
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1('cta_fixture_ca23_inexistente_aaaa');
 IF n<>0 THEN RAISE EXCEPTION 'CA23: lector SERIALIZABLE habilitado en READ COMMITTED'; END IF;
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1('cta_fixture_ca23_inexistente_aaaa');
 IF n<>0 THEN RAISE EXCEPTION 'CA23: reconciliación habilita cuenta sin asignación'; END IF;
 SELECT prosrc INTO STRICT cuerpo FROM pg_catalog.pg_proc WHERE oid='vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'::regprocedure;
 continuidad:=pg_catalog.strpos(cuerpo,'vec:admin:continuidad:v1');
 operacion:=pg_catalog.strpos(cuerpo,'vec_contexto_actor_v1:operacion:v2:');
 lector:=pg_catalog.strpos(cuerpo,'SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1');
 IF continuidad=0 OR operacion<=continuidad OR lector<=operacion THEN RAISE EXCEPTION 'CA23: orden de cerrojos de reconciliación incompatible'; END IF;
 SELECT prosrc INTO STRICT cuerpo FROM pg_catalog.pg_proc WHERE oid='vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'::regprocedure;
 continuidad:=pg_catalog.strpos(cuerpo,'vec:admin:continuidad:v1');
 operacion:=pg_catalog.strpos(cuerpo,'vec_contexto_actor_v1:operacion:v2:');
 lector:=pg_catalog.strpos(cuerpo,'SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1');
 IF continuidad=0 OR operacion<=continuidad OR lector<=operacion THEN RAISE EXCEPTION 'CA23: orden de cerrojos de resolución incompatible'; END IF;
 IF pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_admin_perfiles','vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1(text)','EXECUTE') THEN RAISE EXCEPTION 'CA23: lector privado accesible al runtime'; END IF;
END $aislamiento$;
RESET ROLE;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $sin_reconciliacion$ DECLARE n integer; BEGIN
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1('cta_fixture_ca23_inexistente_aaaa');
 IF n<>0 THEN RAISE EXCEPTION 'CA23: lector READ COMMITTED habilitado en SERIALIZABLE'; END IF;
END $sin_reconciliacion$;
RESET ROLE;
ROLLBACK;
-- El listado propio público tiene su conexión READ COMMITTED independiente.
BEGIN ISOLATION LEVEL READ COMMITTED;
CREATE ROLE vec_fixture_is12_lista LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_perfiles TO vec_fixture_is12_lista WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_fixture_is12_lista;
DO $lista_sin_identidad$ DECLARE n integer; BEGIN
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.listar_perfiles_admin_v1('desarrollo','admin.test.invalid','fixture_is12',repeat('0',64),repeat('0',64),clock_timestamp(),clock_timestamp(),clock_timestamp()+interval '1 minute',clock_timestamp()+interval '1 minute');
 IF n<>0 THEN RAISE EXCEPTION 'IS12: listado sin identidad nominal'; END IF;
END $lista_sin_identidad$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
