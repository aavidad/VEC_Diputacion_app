\set ON_ERROR_STOP on
-- PostgreSQL 18, clon sintético desechable; la prueba completa se revierte.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
DO $login$ BEGIN
 IF to_regrole('vec_externo_avisos_usuarios') IS NULL THEN
  CREATE ROLE vec_externo_avisos_usuarios LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_usuarios_ejecutor_externo TO vec_externo_avisos_usuarios WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_externo_avisos_usuarios' AND rolcanlogin AND rolinherit
    AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member='vec_externo_avisos_usuarios'::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_externo_avisos_usuarios'::regrole
    AND roleid='vec_usuarios_ejecutor_externo'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 THEN RAISE EXCEPTION 'login sintético de ensayo incompatible'; END IF;
END $login$;
CREATE ROLE ad3_119_ajeno LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_ejecutor_externo TO ad3_119_ajeno WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE ad3_119_otro NOLOGIN;
CREATE TABLE vec_usuarios_correos_externo.ad3_119_efecto_prueba (id integer PRIMARY KEY);
ALTER TABLE vec_usuarios_correos_externo.ad3_119_efecto_prueba OWNER TO vec_usuarios_correos_externo_propietario;
CREATE FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper(
 accion text DEFAULT 'aceptar',resultado text DEFAULT 'aceptado',recibo text DEFAULT 'aviso_recibo:11111111111111111111111111111111',version bigint DEFAULT 1,
 huella text DEFAULT repeat('1',64)
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RETURN vec_autorizacion_atestada_v3.registrar_auditoria_inbox_externo_v1(
   accion,'productor:prueba','evento:prueba',recibo,'correlacion:prueba',repeat('0',64),huella,version,resultado);
END $f$;
ALTER FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper(text,text,text,bigint,text) OWNER TO vec_usuarios_correos_externo_propietario;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper(text,text,text,bigint,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_usuarios_correos_externo TO ad3_119_ajeno;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper(text,text,text,bigint,text) TO vec_externo_avisos_usuarios,ad3_119_ajeno;
CREATE FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper_efecto()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 INSERT INTO vec_usuarios_correos_externo.ad3_119_efecto_prueba VALUES(1);
 RETURN vec_usuarios_correos_externo.ad3_119_wrapper();
END $f$;
ALTER FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper_efecto() OWNER TO vec_usuarios_correos_externo_propietario;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper_efecto() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_externo.ad3_119_wrapper_efecto() TO vec_externo_avisos_usuarios;
CREATE TEMP TABLE ad3_119_preimagen AS SELECT secuencia,cabeza_sha256 FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_inbox_externa;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $positivo$ DECLARE r text; BEGIN
 r:=vec_usuarios_correos_externo.ad3_119_wrapper();
 IF r !~ '^auditoria_tecnica_externa:[0-9a-f]{32}$' THEN RAISE EXCEPTION 'ref inválida'; END IF;
 PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('aceptar','replay');
 PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('reservar','reservado','aviso_recibo:11111111111111111111111111111111',2);
 PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('confirmar','sin_destino','aviso_recibo:11111111111111111111111111111111',3);
 PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('aceptar','denegado',NULL,0,repeat('0',64));
END $positivo$;
DO $negativos$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_auditoria_inbox_externo_v1('aceptar','p','e','aviso_recibo:11111111111111111111111111111111','c',repeat('0',64),repeat('1',64),1,'aceptado');
  RAISE EXCEPTION 'runtime llamó writer directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('otra','aceptado');
  RAISE EXCEPTION 'acción libre admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('aceptar','sin_destino');
  RAISE EXCEPTION 'resultado incompatible admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('aceptar','aceptado',NULL,0);
  RAISE EXCEPTION 'éxito sin recurso admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios_correos_externo.ad3_119_wrapper('aceptar','aceptado','aviso_recibo:11111111111111111111111111111111',1,'texto personal');
  RAISE EXCEPTION 'huella libre admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa;
  RAISE EXCEPTION 'runtime leyó auditoría';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $negativos$;
RESET SESSION AUTHORIZATION;
DO $cadena$
DECLARE a record; h text; cabeza text; s bigint; n bigint:=0;
BEGIN
 SELECT secuencia,cabeza_sha256 INTO s,cabeza FROM ad3_119_preimagen;
 FOR a IN SELECT * FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa WHERE secuencia>s ORDER BY secuencia LOOP
  s:=s+1; n:=n+1;
  h:=encode(sha256(convert_to(jsonb_build_object('tipo','vec.auditoria.inbox-externo.v1','auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
    'actor_tecnico',a.actor_tecnico,'perfil_tecnico',a.perfil_tecnico,'accion',a.accion,
    'productor_ref',a.productor_ref,'evento_ref',a.evento_ref,'recibo_ref',a.recibo_ref,'recurso_ref',a.recurso_ref,
    'correlacion_ref',a.correlacion_ref,'antes_sha256',a.antes_sha256,'despues_sha256',a.despues_sha256,
    'version',a.version,'resultado',a.resultado,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'anterior_sha256',a.anterior_sha256)::text,'UTF8')),'hex');
  IF a.secuencia<>s OR a.anterior_sha256<>cabeza OR a.huella_sha256<>h OR a.actor_tecnico<>'vec_externo_avisos_usuarios' THEN RAISE EXCEPTION 'cadena o actor alterados'; END IF;
  cabeza:=h;
 END LOOP;
 IF n<>5 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_inbox_externa WHERE secuencia=s AND cabeza_sha256=cabeza) THEN RAISE EXCEPTION 'efectos o cabeza incorrectos'; END IF;
END $cadena$;
-- El LOGIN con TEMP intenta conservar una membresía/estado falsos. El
-- escritor debe consultar exclusivamente los catálogos auténticos, incluso
-- después de que el atacante conceda SELECT sobre sus tablas temporales.
DO $temp$ BEGIN
 EXECUTE format('GRANT TEMP ON DATABASE %I TO vec_externo_avisos_usuarios',current_database());
END $temp$;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
CREATE TEMP TABLE pg_roles AS SELECT * FROM pg_catalog.pg_roles;
CREATE TEMP TABLE pg_auth_members AS SELECT * FROM pg_catalog.pg_auth_members;
GRANT SELECT ON pg_temp.pg_roles,pg_temp.pg_auth_members TO vec_autorizacion_atestada_v3_propietario;
RESET SESSION AUTHORIZATION;
-- Membresía y vigencia se revalidan, sin confiar en una sesión ya abierta.
GRANT ad3_119_otro TO vec_externo_avisos_usuarios;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $multigrupo$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper(); RAISE EXCEPTION 'multigrupo admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $multigrupo$;
RESET SESSION AUTHORIZATION;
REVOKE ad3_119_otro FROM vec_externo_avisos_usuarios;
REVOKE vec_usuarios_ejecutor_externo FROM vec_externo_avisos_usuarios;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $revocado$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper(); RAISE EXCEPTION 'revocado admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $revocado$;
RESET SESSION AUTHORIZATION;
GRANT vec_usuarios_ejecutor_externo TO vec_externo_avisos_usuarios WITH INHERIT TRUE, SET TRUE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $setrole$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper(); RAISE EXCEPTION 'SET role admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $setrole$;
RESET SESSION AUTHORIZATION;
GRANT vec_usuarios_ejecutor_externo TO vec_externo_avisos_usuarios WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
ALTER ROLE vec_externo_avisos_usuarios VALID UNTIL '2000-01-01';
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $expirado$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper(); RAISE EXCEPTION 'expirado admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $expirado$;
RESET SESSION AUTHORIZATION;
ALTER ROLE vec_externo_avisos_usuarios VALID UNTIL 'infinity';
SET SESSION AUTHORIZATION ad3_119_ajeno;
DO $ajeno$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper(); RAISE EXCEPTION 'ajeno admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ajeno$;
RESET SESSION AUTHORIZATION;
-- El propietario funcional no dispone de INSERT/SELECT ni puede truncar.
SET LOCAL ROLE vec_usuarios_correos_externo_propietario;
DO $acl$ BEGIN
 BEGIN PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa; RAISE EXCEPTION 'funcional leyó auditoría'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN TRUNCATE vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa; RAISE EXCEPTION 'funcional truncó auditoría'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $inmutable$ BEGIN
 BEGIN UPDATE vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa SET resultado='replay'; RAISE EXCEPTION 'UPDATE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa; RAISE EXCEPTION 'DELETE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN TRUNCATE vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa; RAISE EXCEPTION 'TRUNCATE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $inmutable$;
RESET ROLE;
CREATE FUNCTION vec_autorizacion_atestada_v3.ad3_119_fallo_prueba()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$ BEGIN RAISE EXCEPTION 'fallo sintético auditoría' USING ERRCODE='55000'; END $f$;
CREATE TRIGGER ad3_119_fallo BEFORE INSERT ON vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.ad3_119_fallo_prueba();
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $atomico$ BEGIN
 BEGIN PERFORM vec_usuarios_correos_externo.ad3_119_wrapper_efecto(); RAISE EXCEPTION 'fallo auditoría ignorado'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $atomico$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.ad3_119_efecto_prueba)
 OR (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_inbox_externa)<>(SELECT secuencia+5 FROM ad3_119_preimagen)
 THEN RAISE EXCEPTION 'auditoría fallida dejó efecto parcial'; END IF;
END $sin_efecto$;
ROLLBACK;
SELECT 'AD3-119 pruebas adversariales OK' AS resultado;
