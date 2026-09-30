\set ON_ERROR_STOP on
-- Sólo clon desechable PG18 después de B62. ROLLBACK restaura todas las
-- funciones. Los dobles de reserva, CTX y consumo sirven para ejercitar el
-- wrapper; NO acreditan criptografía ni el recorrido productivo V3.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 OR to_regrole('vec_externo_avisos_bolsa') IS NOT NULL
 THEN RAISE EXCEPTION 'B62: ensayo requiere clon DBA con LOGIN nominal libre'; END IF;
END $pre$;
CREATE ROLE vec_externo_avisos_bolsa LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_avisos_externos_consumidor TO vec_externo_avisos_bolsa WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
CREATE TEMP TABLE b62_fixture AS
 SELECT c.bolsa_ref,v.participacion_ref,v.candidato_ref,
 (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion) AS contactos_previo,
 'llamamiento:'||repeat('b',64) AS llamamiento,
 'recibo:llamamiento:'||repeat('b',64) AS recibo,
 '2026-09-30T00:01:02.123456Z'::timestamptz AS emitido
 FROM vec_bolsa_llamamientos.vinculo_candidato v
 JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=v.acta_ref
 ORDER BY c.confirmada_en DESC,v.participacion_ref LIMIT 1;
GRANT SELECT ON TABLE pg_temp.b62_fixture TO vec_bolsa_llamamientos_propietario,vec_contexto_actor_v1_propietario,vec_autorizacion_atestada_v3_propietario;
DO $fixture$ BEGIN IF (SELECT count(*) FROM pg_temp.b62_fixture)<>1 THEN RAISE EXCEPTION 'B62: falta vínculo sintético en clon'; END IF; END $fixture$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.es_candidato_externo_avisos_v1(p_candidato text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT EXISTS(SELECT 1 FROM pg_temp.b62_fixture WHERE candidato_ref=p_candidato)
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:b62:doble',(convert_from(p_decision,'UTF8')::jsonb)->>'recurso_ref',repeat('a',64),repeat('a',64),'auditoria:b62:doble',clock_timestamp(),true
$f$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v1(p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,p_huella_finalizacion bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE anterior record; repetida boolean:=false; BEGIN
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN repetida:=true; ELSE
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,repeat('a',64),p_huella_finalizacion,'emision_reservada',p_emitido,'decision:b62:'||p_llamamiento);
 END IF;
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,'bolsa_ref',p_bolsa,'emitido_en',p_emitido,'estado','emision_reservada_resultado_pendiente'),repetida;
END $f$;
CREATE FUNCTION pg_temp.b62_evento() RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('evento_ref','evento_aviso:'||encode(sha256(convert_to(llamamiento||chr(31)||participacion_ref,'UTF8')),'hex'),
 'productor_ref','productor:bolsa:prueba','tipo_versionado','vec.bolsa.aviso-llamamiento.v1','ocurrido_en','2026-09-30T00:01:02.123456Z',
 'correlacion_ref','corr_'||repeat('a',32),'destinatario_externo_ref',candidato_ref,'comunicacion_ref',llamamiento,
 'plantilla_ref','bolsa-llamamiento-v1','plantilla_version','bolsa-llamamiento-v1','recurso_publico_ref','') FROM pg_temp.b62_fixture
$f$;
CREATE FUNCTION pg_temp.b62_llamar(p_eventos jsonb,p_actor text DEFAULT 'per_AAAAAAAAAAAAAAAAAAAAAA') RETURNS jsonb LANGUAGE sql AS $f$
 SELECT r.emision FROM pg_temp.b62_fixture f CROSS JOIN LATERAL vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1(
 f.llamamiento,f.recibo,f.bolsa_ref,p_actor,'b62:clave:ensayo',jsonb_build_array(f.participacion_ref),jsonb_build_object('plantilla_version','bolsa-llamamiento-v1'),f.emitido,decode(repeat('a',64),'hex'),
 ''::bytea,convert_to(jsonb_build_object('principal_id','per_AAAAAAAAAAAAAAAAAAAAAA','accion','llamamiento.emitir.v1','tipo_recurso','bolsa_constituida','recurso_ref',f.bolsa_ref)::text,'UTF8'),
 ''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea,p_eventos) r
$f$;
DO $atomicidad$ DECLARE original jsonb:=pg_temp.b62_evento(); salida jsonb; n integer:=0; BEGIN
 BEGIN PERFORM pg_temp.b62_llamar(jsonb_build_array(original||jsonb_build_object('plantilla_ref','plantilla:ajena')));
 EXCEPTION WHEN invalid_parameter_value THEN n:=n+1; END;
 BEGIN PERFORM pg_temp.b62_llamar(jsonb_build_array(original||jsonb_build_object('direccion','prueba@example.invalid')));
 EXCEPTION WHEN invalid_parameter_value THEN n:=n+1; END;
 BEGIN PERFORM pg_temp.b62_llamar('[]'::jsonb);
 EXCEPTION WHEN SQLSTATE 'VBE01' THEN n:=n+1; END;
 IF n<>3 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido l JOIN pg_temp.b62_fixture f ON f.llamamiento=l.llamamiento_ref)
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox)
 THEN RAISE EXCEPTION 'B62: efecto parcial o material ajeno aceptado'; END IF;
 salida:=pg_temp.b62_llamar(jsonb_build_array(original));
 IF jsonb_array_length(salida->'avisos_externos')<>1 OR jsonb_array_length(salida->'contactos')<>1 OR salida#>>'{contactos,0,resultado}' IS DISTINCT FROM 'aviso_pendiente' OR salida#>>'{contactos,0,recibo_ref}' IS DISTINCT FROM salida#>>'{avisos_externos,0,recibo_outbox_ref}' OR (SELECT count(*) FROM vec_bolsa_llamamientos.aviso_externo_outbox)<>1
 THEN RAISE EXCEPTION 'B62: evento no atomico'; END IF;
 salida:=pg_temp.b62_llamar(jsonb_build_array(original));
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.aviso_externo_outbox)<>1 THEN RAISE EXCEPTION 'B62: replay duplicado'; END IF;
 BEGIN PERFORM pg_temp.b62_llamar(jsonb_build_array(original||jsonb_build_object('correlacion_ref','corr_'||repeat('b',32))));
 EXCEPTION WHEN SQLSTATE 'VBE01' THEN n:=n+1; END;
 BEGIN PERFORM pg_temp.b62_llamar(jsonb_build_array(original),'per_BBBBBBBBBBBBBBBBBBBBBB');
 EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 IF n<>5 THEN RAISE EXCEPTION 'B62: replay huella/actor aceptado'; END IF;
END $atomicidad$;

-- El pendiente tiene un recibo propio real y no escribe contacto común.
DO $contactos$ DECLARE f record; BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b62_fixture;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion)<>f.contactos_previo
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox o WHERE o.llamamiento_ref=f.llamamiento AND o.recibo_outbox_ref ~ '^recibo_outbox:[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'B62: contacto común escrito o recibo ficticio'; END IF;
END $contactos$;

-- El fallo de la autoridad común revierte un ACK que habría sido válido.
CREATE FUNCTION pg_temp.b62_ack_prueba() RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE o record; BEGIN
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.aviso_externo_outbox LIMIT 1;
 PERFORM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(o.productor_ref,o.evento_ref,o.huella_sha256,'aviso_recibo:'||repeat('a',32));
END $f$;
ALTER FUNCTION pg_temp.b62_ack_prueba() OWNER TO vec_bolsa_llamamientos_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text) FROM vec_bolsa_llamamientos_propietario;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $fallo_auditoria$ DECLARE denegada boolean:=false; BEGIN
 BEGIN PERFORM pg_temp.b62_ack_prueba(); EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B62: efecto aceptado sin auditoría'; END IF;
END $fallo_auditoria$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$ BEGIN IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_aceptacion)
 THEN RAISE EXCEPTION 'B62: ACK quedó persistido tras fallo auditoría'; END IF; END $sin_efecto$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text) TO vec_bolsa_llamamientos_propietario;

SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
-- Si no se fija pg_temp al final, estos catálogos privados pueden ocultar
-- nuevas membresías al SECURITY DEFINER. Sólo se copian LOGIN/grupo de prueba.
CREATE TEMP TABLE pg_roles AS SELECT * FROM pg_catalog.pg_roles WHERE rolname IN('vec_externo_avisos_bolsa','vec_bolsa_avisos_externos_consumidor');
CREATE TEMP TABLE pg_auth_members AS SELECT * FROM pg_catalog.pg_auth_members WHERE member=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='vec_externo_avisos_bolsa');
GRANT SELECT ON pg_temp.pg_roles,pg_temp.pg_auth_members TO vec_bolsa_llamamientos_propietario;
DO $consumidor$ DECLARE e record; ack record; r record; vacio record;
BEGIN
 SELECT * INTO STRICT e FROM vec_bolsa_llamamientos.tirar_avisos_externos_v1(1);
 IF (SELECT count(*) FROM jsonb_object_keys(e.evento))<>10 OR e.evento ?| ARRAY['direccion','actor_ref','bolsa_ref','participacion_ref','cuerpo','asunto']
 OR e.auditoria_ref !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$' OR e.error_codigo IS NOT NULL
 THEN RAISE EXCEPTION 'B62: proyección excesiva o sin auditoría'; END IF;
 SELECT * INTO STRICT ack FROM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',repeat('0',64),'aviso_recibo:'||repeat('a',32));
 IF ack.aceptada OR ack.error_codigo IS DISTINCT FROM 'VBE01' OR ack.auditoria_ref !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'B62: ACK divergente sin rechazo durable'; END IF;
 SELECT * INTO STRICT vacio FROM vec_bolsa_llamamientos.tirar_avisos_externos_v1(101);
 IF vacio.evento IS NOT NULL OR vacio.error_codigo IS DISTINCT FROM '22023' OR vacio.auditoria_ref !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'B62: límite inválido no auditado'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32),'reservado_incierto');
 IF NOT r.registrada OR r.error_codigo IS NOT NULL OR r.auditoria_ref !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$' THEN RAISE EXCEPTION 'B62: resultado incierto no registrado'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32),'reservado_incierto');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32),'aceptado');
 IF NOT r.registrada OR r.error_codigo IS NOT NULL THEN RAISE EXCEPTION 'B62: finalización incierta bloqueada'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32),'reservado_incierto');
 IF NOT r.registrada OR r.error_codigo IS NOT NULL THEN RAISE EXCEPTION 'B62: replay histórico bloqueado'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32),'no_aceptado');
 IF r.registrada OR r.error_codigo IS DISTINCT FROM 'VBE01' THEN RAISE EXCEPTION 'B62: resultado terminal reescrito'; END IF;
 SELECT * INTO STRICT ack FROM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32));
 IF NOT ack.aceptada OR ack.error_codigo IS NOT NULL THEN RAISE EXCEPTION 'B62: ACK válido bloqueado'; END IF;
 SELECT * INTO STRICT ack FROM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('a',32));
 SELECT * INTO STRICT ack FROM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(e.evento->>'productor_ref',e.evento->>'evento_ref',e.huella,'aviso_recibo:'||repeat('b',32));
 IF ack.aceptada OR ack.error_codigo IS DISTINCT FROM 'VBE01' THEN RAISE EXCEPTION 'B62: aceptación original reescrita'; END IF;
 SELECT * INTO STRICT vacio FROM vec_bolsa_llamamientos.tirar_avisos_externos_v1(100);
 IF vacio.evento IS NOT NULL OR vacio.huella IS NOT NULL OR vacio.error_codigo IS NOT NULL OR vacio.auditoria_ref !~ '^auditoria_tecnica_interna:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'B62: lote vacío sin prueba real'; END IF;
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox; RAISE EXCEPTION 'B62: tabla visible al consumidor';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $consumidor$;
RESET SESSION AUTHORIZATION;
-- Una concesión ajena invalida inmediatamente al mismo LOGIN.
GRANT vec_bolsa_llamamientos_ejecutor TO vec_externo_avisos_bolsa WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET SESSION AUTHORIZATION vec_externo_avisos_bolsa;
DO $acl_extra$ DECLARE denegada boolean:=false; BEGIN
 BEGIN PERFORM vec_bolsa_llamamientos.tirar_avisos_externos_v1(1);
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B62: consumidor con permisos internos aceptado'; END IF;
END $acl_extra$;
RESET SESSION AUTHORIZATION;
REVOKE vec_bolsa_llamamientos_ejecutor FROM vec_externo_avisos_bolsa;
DO $inmutabilidad$ DECLARE n integer:=0; BEGIN
 BEGIN UPDATE vec_bolsa_llamamientos.aviso_externo_outbox SET canon='otro'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.aviso_externo_aceptacion; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN TRUNCATE vec_bolsa_llamamientos.aviso_externo_aceptacion; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 IF n<>3 OR (SELECT count(*) FROM vec_bolsa_llamamientos.aviso_externo_outbox)<>1 OR (SELECT count(*) FROM vec_bolsa_llamamientos.aviso_externo_aceptacion)<>1
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion) IS DISTINCT FROM (SELECT contactos_previo FROM pg_temp.b62_fixture) OR (SELECT count(*) FROM vec_bolsa_llamamientos.aviso_externo_resultado)<>2 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_tecnica_outbox_interna WHERE actor_tecnico='vec_externo_avisos_bolsa' AND resultado='denegado')<3
 THEN RAISE EXCEPTION 'B62: historia mutada o contacto común escrito'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_portal_externo','vec_bolsa_llamamientos.tirar_avisos_externos_v1(integer)','EXECUTE')
 OR has_type_privilege('public','vec_bolsa_llamamientos.aviso_externo_outbox','USAGE')
 THEN RAISE EXCEPTION 'B62: concesion excesiva'; END IF;
END $inmutabilidad$;
ROLLBACK;
