\set ON_ERROR_STOP on
-- PostgreSQL 18, clon sintético desechable; la prueba completa se revierte.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='30s';
DO $grupo$ BEGIN
 IF to_regrole('vec_bolsa_avisos_externos_consumidor') IS NULL THEN
  CREATE ROLE vec_bolsa_avisos_externos_consumidor NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 END IF;
END $grupo$;
DO $login$ BEGIN
 IF to_regrole('vec_externo_avisos_bolsa') IS NULL THEN
  CREATE ROLE vec_externo_avisos_bolsa LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_avisos_externos_consumidor TO vec_externo_avisos_bolsa WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_externo_avisos_bolsa' AND rolcanlogin AND rolinherit
    AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR (SELECT count(*) FROM pg_auth_members WHERE member='vec_externo_avisos_bolsa'::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_externo_avisos_bolsa'::regrole
    AND roleid='vec_bolsa_avisos_externos_consumidor'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 THEN RAISE EXCEPTION 'login sintético de ensayo incompatible'; END IF;
END $login$;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_avisos_externos_consumidor;
CREATE ROLE ad3_120_ajeno LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_avisos_externos_consumidor TO ad3_120_ajeno WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE ad3_120_otro NOLOGIN;
CREATE TABLE vec_bolsa_llamamientos.ad3_120_efecto_prueba (id integer PRIMARY KEY);
ALTER TABLE vec_bolsa_llamamientos.ad3_120_efecto_prueba OWNER TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper(
 accion text DEFAULT 'aceptar',resultado text DEFAULT 'aceptado',recibo text DEFAULT 'aviso_recibo:11111111111111111111111111111111',version bigint DEFAULT 1,
 huella text DEFAULT repeat('1',64),productor text DEFAULT 'productor:prueba',evento text DEFAULT 'evento_aviso:'||repeat('1',64)
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(
   accion,productor,evento,recibo,'correlacion:prueba',repeat('0',64),huella,version,resultado);
END $f$;
ALTER FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper(text,text,text,bigint,text,text,text) OWNER TO vec_bolsa_llamamientos_propietario;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper(text,text,text,bigint,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO ad3_120_ajeno;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper(text,text,text,bigint,text,text,text) TO vec_externo_avisos_bolsa,ad3_120_ajeno;
CREATE FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper_efecto()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 INSERT INTO vec_bolsa_llamamientos.ad3_120_efecto_prueba VALUES(1);
 RETURN vec_bolsa_llamamientos.ad3_120_wrapper();
END $f$;
ALTER FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper_efecto() OWNER TO vec_bolsa_llamamientos_propietario;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper_efecto() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ad3_120_wrapper_efecto() TO vec_externo_avisos_bolsa;
CREATE TEMP TABLE ad3_120_preimagen AS SELECT secuencia,cabeza_sha256 FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
-- El LOGIN puede crear tablas temporales con nombres de catálogo y conceder
-- lectura al owner AD3. Su contenido aparenta una identidad siempre vigente
-- y una sola membresía; las negativas posteriores deben consultar el catálogo real.
CREATE TEMP TABLE pg_roles AS
 SELECT oid,rolname,rolcanlogin,rolinherit,rolsuper,rolcreatedb,rolcreaterole,
        rolreplication,rolbypassrls,'infinity'::timestamptz AS rolvaliduntil
 FROM pg_catalog.pg_roles
 WHERE rolname IN('vec_externo_avisos_bolsa','vec_bolsa_avisos_externos_consumidor');
CREATE TEMP TABLE pg_auth_members AS
 SELECT member,roleid,inherit_option,set_option,admin_option
 FROM pg_catalog.pg_auth_members
 WHERE member='vec_externo_avisos_bolsa'::regrole;
GRANT SELECT ON TABLE pg_temp.pg_roles,pg_temp.pg_auth_members TO vec_autorizacion_atestada_v3_propietario;
DO $positivo$ DECLARE r text; BEGIN
 r:=vec_bolsa_llamamientos.ad3_120_wrapper();
 IF r !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$' THEN RAISE EXCEPTION 'ref inválida'; END IF;
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('aceptar','replay');
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','extraido',NULL,1);
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','sin_registro',NULL,0,repeat('0',64),NULL,NULL);
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('resultado','aceptado');
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('resultado','no_aceptado');
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('resultado','sin_destino');
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('resultado','reservado_incierto');
 PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('aceptar','denegado',NULL,0,repeat('0',64),NULL,NULL);
END $positivo$;
DO $negativos$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1('aceptar','p','evento_aviso:'||repeat('1',64),'aviso_recibo:11111111111111111111111111111111','c',repeat('0',64),repeat('1',64),1,'aceptado');
  RAISE EXCEPTION 'runtime llamó writer directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('otra','aceptado');
  RAISE EXCEPTION 'acción libre admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('aceptar','sin_destino');
  RAISE EXCEPTION 'resultado incompatible admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('aceptar','aceptado',NULL,0);
  RAISE EXCEPTION 'éxito sin recurso admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('aceptar','aceptado','aviso_recibo:11111111111111111111111111111111',1,'texto personal');
  RAISE EXCEPTION 'huella libre admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna;
  RAISE EXCEPTION 'runtime leyó auditoría';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','sin_registro',NULL,1);
  RAISE EXCEPTION 'vacío con evento admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','extraido',NULL,1,repeat('1',64),NULL,'evento_aviso:'||repeat('1',64));
  RAISE EXCEPTION 'par productor/evento incompleto admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('resultado','reservado_incierto',NULL,1);
  RAISE EXCEPTION 'resultado sin recibo real admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','extraido',NULL,1,repeat('1',64),'persona@correo.invalid');
  RAISE EXCEPTION 'correo en productor admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.ad3_120_wrapper('extraer','extraido',NULL,1,repeat('1',64),'productor:prueba','texto_personal');
  RAISE EXCEPTION 'evento libre admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $negativos$;
RESET SESSION AUTHORIZATION;
DO $cadena$
DECLARE a record; h text; cabeza text; s bigint; n bigint:=0;
BEGIN
 SELECT secuencia,cabeza_sha256 INTO s,cabeza FROM ad3_120_preimagen;
 FOR a IN SELECT * FROM vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna WHERE secuencia>s ORDER BY secuencia LOOP
  s:=s+1; n:=n+1;
  h:=encode(sha256(convert_to(jsonb_build_object('tipo','vec.auditoria.outbox-interno.v1','auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
    'actor_tecnico',a.actor_tecnico,'perfil_tecnico',a.perfil_tecnico,'accion',a.accion,
    'productor_ref',a.productor_ref,'evento_ref',a.evento_ref,'recibo_ref',a.recibo_ref,'recurso_ref',a.recurso_ref,
    'correlacion_ref',a.correlacion_ref,'antes_sha256',a.antes_sha256,'despues_sha256',a.despues_sha256,
    'version',a.version,'resultado',a.resultado,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'anterior_sha256',a.anterior_sha256)::text,'UTF8')),'hex');
  IF a.secuencia<>s OR a.anterior_sha256<>cabeza OR a.huella_sha256<>h OR a.actor_tecnico<>'vec_externo_avisos_bolsa' THEN RAISE EXCEPTION 'cadena o actor alterados'; END IF;
  cabeza:=h;
 END LOOP;
 IF n<>9 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna WHERE secuencia=s AND cabeza_sha256=cabeza) THEN RAISE EXCEPTION 'efectos o cabeza incorrectos'; END IF;
END $cadena$;
-- Membresía y vigencia se revalidan, sin confiar en una sesión ya abierta.
GRANT ad3_120_otro TO vec_externo_avisos_bolsa;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $multigrupo$ BEGIN
 IF (SELECT count(*) FROM pg_temp.pg_auth_members WHERE member='vec_externo_avisos_bolsa'::regrole)<>1
 THEN RAISE EXCEPTION 'fixture temporal no oculta membresía adicional'; END IF;
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper(); RAISE EXCEPTION 'multigrupo admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $multigrupo$;
RESET SESSION AUTHORIZATION;
REVOKE ad3_120_otro FROM vec_externo_avisos_bolsa;
REVOKE vec_bolsa_avisos_externos_consumidor FROM vec_externo_avisos_bolsa;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $revocado$ BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper(); RAISE EXCEPTION 'revocado admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $revocado$;
RESET SESSION AUTHORIZATION;
GRANT vec_bolsa_avisos_externos_consumidor TO vec_externo_avisos_bolsa WITH INHERIT TRUE, SET TRUE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $setrole$ BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper(); RAISE EXCEPTION 'SET role admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $setrole$;
RESET SESSION AUTHORIZATION;
GRANT vec_bolsa_avisos_externos_consumidor TO vec_externo_avisos_bolsa WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
ALTER ROLE vec_externo_avisos_bolsa VALID UNTIL '2000-01-01';
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $expirado$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_temp.pg_roles WHERE rolname=session_user AND rolvaliduntil>clock_timestamp())
 THEN RAISE EXCEPTION 'fixture temporal no oculta caducidad'; END IF;
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper(); RAISE EXCEPTION 'expirado admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $expirado$;
RESET SESSION AUTHORIZATION;
ALTER ROLE vec_externo_avisos_bolsa VALID UNTIL 'infinity';
SET SESSION AUTHORIZATION ad3_120_ajeno;
DO $ajeno$ BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper(); RAISE EXCEPTION 'ajeno admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ajeno$;
RESET SESSION AUTHORIZATION;
-- El propietario funcional no dispone de INSERT/SELECT ni puede truncar.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $acl$ BEGIN
 BEGIN PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna; RAISE EXCEPTION 'funcional leyó auditoría'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN TRUNCATE vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna; RAISE EXCEPTION 'funcional truncó auditoría'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $inmutable$ BEGIN
 BEGIN UPDATE vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna SET resultado='replay'; RAISE EXCEPTION 'UPDATE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna; RAISE EXCEPTION 'DELETE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN TRUNCATE vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna; RAISE EXCEPTION 'TRUNCATE admitido'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $inmutable$;
RESET ROLE;
CREATE FUNCTION vec_autorizacion_atestada_v3.ad3_120_fallo_prueba()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$ BEGIN RAISE EXCEPTION 'fallo sintético auditoría' USING ERRCODE='55000'; END $f$;
CREATE TRIGGER ad3_120_fallo BEFORE INSERT ON vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.ad3_120_fallo_prueba();
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $atomico$ BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.ad3_120_wrapper_efecto(); RAISE EXCEPTION 'fallo auditoría ignorado'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $atomico$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.ad3_120_efecto_prueba)
 OR (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_tecnica_outbox_interna)<>(SELECT secuencia+9 FROM ad3_120_preimagen)
 THEN RAISE EXCEPTION 'auditoría fallida dejó efecto parcial'; END IF;
END $sin_efecto$;
ROLLBACK;
SELECT 'AD3-120 pruebas adversariales OK' AS resultado;
