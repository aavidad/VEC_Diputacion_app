\set ON_ERROR_STOP on
-- Ensayo focal en clon sintético, después de CTX16, AD3-119 y U14.
-- No prueba SMTP: comprueba conservación, replay y auditoría del resultado.
-- Todo el material y los cambios de ensayo se revierten al final.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $login$ BEGIN
 IF to_regrole('vec_externo_avisos_usuarios') IS NULL THEN
  CREATE ROLE vec_externo_avisos_usuarios LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_usuarios_ejecutor_externo TO vec_externo_avisos_usuarios WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_externo_avisos_usuarios' AND rolcanlogin AND rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
   AND (rolvaliduntil IS NULL OR rolvaliduntil>clock_timestamp()))
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member='vec_externo_avisos_usuarios'::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_externo_avisos_usuarios'::regrole
   AND roleid='vec_usuarios_ejecutor_externo'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 THEN RAISE EXCEPTION 'U14 login de ensayo incompatible'; END IF;
END $login$;
CREATE TEMP TABLE u14_incierto_observacion(recibo text,token text);
GRANT USAGE ON SCHEMA pg_temp TO vec_externo_avisos_usuarios;
GRANT INSERT ON u14_incierto_observacion TO vec_externo_avisos_usuarios;
SET SESSION AUTHORIZATION vec_externo_avisos_usuarios;
DO $recorrido$
DECLARE material text; ack jsonb; reserva jsonb; confirmacion jsonb; replay jsonb; ref text; token text;
BEGIN
 material:='{"evento_ref":"evento:ensayo:u14-incierto","productor_ref":"productor:ensayo:u14","tipo_versionado":"vec.bolsa.aviso-llamamiento.v1","ocurrido_en":"2026-09-30T12:13:14.123456Z","correlacion_ref":"corr:ensayo:u14-incierto","destinatario_externo_ref":"can_'||repeat('a',24)||'","comunicacion_ref":"llamamiento:'||repeat('b',64)||'","plantilla_ref":"plantilla:ensayo","plantilla_version":"1","recurso_publico_ref":""}';
 ack:=vec_usuarios_correos_externo.aceptar_aviso_externo_v1(material);
 IF ack ? 'error' OR ack->>'auditoria_ref' IS NULL THEN RAISE EXCEPTION 'U14 aceptación de ensayo fallida'; END IF;
 ref:=ack#>>'{recibo,recibo_ref}';
 reserva:=vec_usuarios_correos_externo.reservar_aviso_externo_v1(ref);
 IF reserva->>'estado' IS DISTINCT FROM 'reservado' OR reserva->>'replay' IS DISTINCT FROM 'false' OR reserva->>'reserva_ref' IS NULL
 THEN RAISE EXCEPTION 'U14 reserva inicial fallida'; END IF;
 token:=reserva->>'reserva_ref';
 confirmacion:=vec_usuarios_correos_externo.confirmar_aviso_externo_v1(ref,token,'reservado_incierto');
 IF confirmacion->>'confirmado' IS DISTINCT FROM 'true' OR confirmacion->>'auditoria_ref' IS NULL
 THEN RAISE EXCEPTION 'U14 incertidumbre no confirmada'; END IF;
 replay:=vec_usuarios_correos_externo.confirmar_aviso_externo_v1(ref,token,'reservado_incierto');
 IF replay->>'confirmado' IS DISTINCT FROM 'true' OR replay->>'auditoria_ref' IS NULL OR replay->>'auditoria_ref'=confirmacion->>'auditoria_ref'
 THEN RAISE EXCEPTION 'U14 replay de confirmación no auditado'; END IF;
 replay:=vec_usuarios_correos_externo.confirmar_aviso_externo_v1(ref,token,'no_aceptado');
 IF replay->>'error' IS DISTINCT FROM 'conflicto' OR replay->>'auditoria_ref' IS NULL
 THEN RAISE EXCEPTION 'U14 incertidumbre sustituida por rechazo'; END IF;
 replay:=vec_usuarios_correos_externo.aceptar_aviso_externo_v1(material);
 IF replay#>>'{recibo,replay}' IS DISTINCT FROM 'true' OR replay#>>'{recibo,recibo_ref}' IS DISTINCT FROM ref
 OR replay#>>'{recibo,huella}' IS DISTINCT FROM ack#>>'{recibo,huella}' OR replay#>>'{recibo,aceptado_en}' IS DISTINCT FROM ack#>>'{recibo,aceptado_en}'
 THEN RAISE EXCEPTION 'U14 replay sustituyó recibo'; END IF;
 replay:=vec_usuarios_correos_externo.reservar_aviso_externo_v1(ref);
 IF replay->>'estado' IS DISTINCT FROM 'reservado_incierto' OR replay->>'replay' IS DISTINCT FROM 'true'
 OR replay#>>'{recibo,recibo_ref}' IS DISTINCT FROM ref OR replay#>>'{recibo,huella}' IS DISTINCT FROM ack#>>'{recibo,huella}'
 OR replay#>>'{recibo,aceptado_en}' IS DISTINCT FROM ack#>>'{recibo,aceptado_en}'
 OR replay ?| ARRAY['reserva_ref','evento','persona_ref','correo_ref','sobre']
 THEN RAISE EXCEPTION 'U14 replay perdió incertidumbre o entregó otra reserva'; END IF;
 INSERT INTO pg_temp.u14_incierto_observacion VALUES(ref,token);
END $recorrido$;
RESET SESSION AUTHORIZATION;
DO $durable$ DECLARE ref text; token text; BEGIN
 SELECT recibo,u14_incierto_observacion.token INTO STRICT ref,token FROM pg_temp.u14_incierto_observacion;
 IF (SELECT count(*) FROM vec_usuarios_correos_externo.avisos_reserva WHERE recibo_ref=ref
   AND reserva_sha256=encode(sha256(convert_to(token,'UTF8')),'hex'))<>1
 OR (SELECT count(*) FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=ref AND accion='reservado_incierto')<>1
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=ref AND accion IN('aceptado','no_aceptado','sin_destino'))
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa WHERE recibo_ref=ref
   AND accion='confirmar' AND resultado='reservado_incierto' AND version=3)<>1
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa WHERE recibo_ref=ref
   AND accion='confirmar' AND resultado='replay' AND version=3)<>1
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_tecnica_inbox_externa WHERE recibo_ref=ref
   AND accion='confirmar' AND resultado='denegado' AND version=3)<>1
 THEN RAISE EXCEPTION 'U14 reserva, historia o auditoría de incertidumbre alteradas'; END IF;
END $durable$;
ROLLBACK;
SELECT 'U14 resultado incierto y replay OK' AS resultado;
