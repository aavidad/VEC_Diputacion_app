\set ON_ERROR_STOP on
-- Solo clon desechable. Comprueba SQL, ACL, aislamiento y los núcleos privados.
-- La preparación sintética del acta no acredita constitución ni consumo V3.
-- El recorrido con material criptográfico real se prueba desde la composición.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
-- Se prepara exclusivamente el vínculo sintético de esta prueba; se reponen
-- los disparadores antes de probar las tablas y funciones nuevas de B63.
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:b63:sintetica',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:b63',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES('inst:b63:sintetica',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:b63:sintetica',1,encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:b63:sintetica','bolsa:b63:sintetica',1,encode(sha256('{}'::bytea),'hex'),'inst:b63:sintetica',1,encode(sha256('{}'::bytea),'hex'),'categoria:rpt:b63','per_b63_operador_sintetico_00001',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES('inst:b63:sintetica',1,1,'part:b63:sintetica',1);
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato VALUES('part:b63:sintetica','can_b63_candidato_sintetico_0001','acta:b63:sintetica','inst:b63:sintetica',1,now()-interval '10 days');
SET LOCAL session_replication_role=origin;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $gobierno$
DECLARE d jsonb; x bytea; h text; r record;
BEGIN
 x:=convert_to('{"persona_ref":"per_b63_persona_sintetica_00001","perfil_activo_ref":"prf_b63_perfil_sintetico_00001","contexto_actor_ref":"vca_b63_contexto_sintetico_0001"}','UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1('can_b63_candidato_sintetico_0001','part:b63:sintetica',x);
  RAISE EXCEPTION 'B63: actuación sin clasificación positiva';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 d:=jsonb_build_object('participacion_ref','part:b63:sintetica','bolsa_ref','bolsa:b63:sintetica','candidato_ref','can_b63_candidato_sintetico_0001',
  'persona_ref','per_b63_persona_sintetica_00001','perfil_ref','prf_b63_perfil_sintetico_00001','contexto_actor_ref','vca_b63_contexto_sintetico_0001',
  'estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z',
  'fuente_ref','prc_b63_fuente_sintetica_000001','fuente_version',1,'fuente_huella_sha256',repeat('a',64));
 h:=vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(d,1);
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(d,0,NULL,repeat('f',64));
  RAISE EXCEPTION 'B63: huella de gobierno no aprobada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(d,0,NULL,h);
 IF r.version<>1 OR r.huella_sha256<>h THEN RAISE EXCEPTION 'B63: recibo de provisión divergente'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(d,0,NULL,h);
  RAISE EXCEPTION 'B63: gobierno CAS antiguo';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1('can_b63_candidato_sintetico_0001','part:b63:sintetica',x);
 BEGIN
  PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1('can_b63_candidato_sintetico_0001','part:b63:sintetica',convert_to('{"persona_ref":"per_b63_persona_sintetica_00001","perfil_activo_ref":"prf_b63_ajeno_sintetico_000001","contexto_actor_ref":"vca_b63_contexto_sintetico_0001"}','UTF8'));
  RAISE EXCEPTION 'B63: perfil ajeno usó clasificación';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $gobierno$;
DO $nucleos$
DECLARE b constant text:='bolsa:b63:sintetica'; p constant text:='part:b63:sintetica'; c constant text:='can_b63_candidato_sintetico_0001';
 ahora timestamptz:=clock_timestamp(); l text:='llamamiento:'||repeat('b',64); o text:='oferta:'||repeat('c',64);
 r record; antes jsonb; despues jsonb; t text;
BEGIN
 SELECT jsonb_build_array((SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_portal_candidato),(SELECT count(*) FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento),(SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta),(SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta_candidato),(SELECT count(*) FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion)) INTO antes;
 PERFORM vec_bolsa_llamamientos.registrar_solicitud_portal_externa_interna_v1('solicitud-portal:'||repeat('d',64),'recibo:solicitud-portal:'||repeat('d',64),c,b,p,'pausa',ahora+interval '1 day',ahora+interval '2 days',ARRAY['disponible'],'regla:b63:sintetica','clave-b63-solicitud',ahora,'decision:b63:solicitud');
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_solicitud_portal_externa_interna_v1('solicitud-portal:'||repeat('e',64),'recibo:solicitud-portal:'||repeat('e',64),c,b,p,'pausa',ahora+interval '1 day',ahora+interval '2 days',ARRAY['disponible'],'regla:b63:sintetica','clave-b63-solicitud-2',ahora,'decision:b63:solicitud:2');
  RAISE EXCEPTION 'B63: segunda solicitud pendiente';
 EXCEPTION WHEN SQLSTATE 'VBP02' THEN NULL; END;
 -- Antecedentes propios de Bolsa; el foco prueba que B63 solo añade hechos
 -- a sus cinco tablas externas, con las FK reales y disparadores activados.
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES(l,'recibo:llamamiento:'||repeat('b',64),b,'per_b63_operador_sintetico_00001','clave-b63-llamamiento',jsonb_build_array(p),'{}'::jsonb,repeat('a',64),decode(repeat('ab',32),'hex'),'emision_reservada',ahora-interval '1 minute','decision:b63:llamamiento');
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref)
 VALUES('contacto:b63:sintetico',b,p,l,'telefono',ahora-interval '30 seconds','per_b63_operador_sintetico_00001','contactado','Foco sintético B63','clave-b63-contacto','recibo:b63:contacto');
 PERFORM vec_bolsa_llamamientos.registrar_respuesta_portal_externa_interna_v1('respuesta-portal:'||repeat('f',64),'recibo:respuesta-portal:'||repeat('f',64),c,b,p,'acepta',NULL,NULL,NULL,'propuesta_rrhh',ahora-interval '30 seconds',ahora+interval '1 day',ARRAY['contactado'],'regla:b63:sintetica','clave-b63-respuesta',ahora,'decision:b63:respuesta');
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_externo_v1(p,ahora+interval '1 second',ARRAY['contactado'])) THEN RAISE EXCEPTION 'B63: ignoró la respuesta externa'; END IF;
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(o,'recibo:oferta:'||repeat('c',64),b,'per_b63_operador_sintetico_00001','clave-b63-oferta','{}'::jsonb,'{}'::jsonb,ahora-interval '1 hour',ahora+interval '1 day',repeat('a',64),'decision:b63:oferta');
 PERFORM vec_bolsa_llamamientos.registrar_disposicion_oferta_externa_interna_v1(o,'recibo:disposicion:'||repeat('1',64),c,p,'clave-b63-disposicion',ahora,'decision:b63:disposicion');
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES(p,1,'kms:b63:sintetico',decode(repeat('01',12),'hex'),decode(repeat('02',32),'hex'),'Foco sintético B63','per_b63_operador_sintetico_00001',ahora-interval '1 hour','clave-b63-datos','recibo:b63:datos');
 PERFORM vec_bolsa_llamamientos.registrar_confirmacion_contacto_externa_interna_v1(c,b,p,1,'clave-b63-confirmacion','recibo:confirmacion-contacto:'||repeat('2',64),ahora,'decision:b63:confirmacion');
 SELECT jsonb_build_array((SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_portal_candidato),(SELECT count(*) FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento),(SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta),(SELECT count(*) FROM vec_bolsa_llamamientos.disposicion_oferta_candidato),(SELECT count(*) FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion)) INTO despues;
 IF antes IS DISTINCT FROM despues THEN RAISE EXCEPTION 'B63: escribió las cinco historias compartidas'; END IF;
 FOREACH t IN ARRAY ARRAY['solicitud_portal_candidato_externa','respuesta_portal_llamamiento_externa','disposicion_oferta_externa','disposicion_oferta_candidato_externa','confirmacion_contacto_participacion_externa'] LOOP
  EXECUTE format('SELECT count(*) FROM vec_bolsa_llamamientos.%I',t) INTO r;
  IF r.count<>1 THEN RAISE EXCEPTION 'B63: duplicó/perdió actuación %',t; END IF;
  BEGIN
   EXECUTE format('DELETE FROM vec_bolsa_llamamientos.%I',t);
   RAISE EXCEPTION 'B63: historia mutable %',t;
  EXCEPTION WHEN insufficient_privilege OR object_not_in_prerequisite_state THEN NULL; END;
 END LOOP;
END $nucleos$;
-- Conservación tras revocar la proyección gobernada; no se reescriben recibos.
DO $revocacion$
DECLARE d jsonb; h text; x bytea;
BEGIN
 SELECT documento,huella_sha256 INTO STRICT d,h FROM vec_bolsa_llamamientos.contexto_participacion_externa_versiones WHERE participacion_ref='part:b63:sintetica' AND version=1;
 d:=jsonb_set(d,'{estado}','"revocado"'::jsonb);
 PERFORM vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(d,1,h,vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(d,2));
 x:=convert_to('{"persona_ref":"per_b63_persona_sintetica_00001","perfil_activo_ref":"prf_b63_perfil_sintetico_00001","contexto_actor_ref":"vca_b63_contexto_sintetico_0001"}','UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1('can_b63_candidato_sintetico_0001','part:b63:sintetica',x);
  RAISE EXCEPTION 'B63: contexto revocado positivo';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $revocacion$;
RESET ROLE;
DO $acl$
DECLARE f regprocedure; rol oid:='vec_bolsa_llamamientos_portal_externo'::regrole; n integer;
BEGIN
 SELECT count(*) INTO n FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.pronamespace='vec_bolsa_llamamientos'::regnamespace AND a.grantee=rol;
 IF n<>11 THEN RAISE EXCEPTION 'B63: número de fachadas nominales'; END IF;
 IF has_function_privilege(rol,'vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege(rol,'vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(jsonb,bigint,text,text)','EXECUTE')
    OR has_function_privilege(rol,'vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1(text,text,text)','EXECUTE')
    OR has_table_privilege(rol,'vec_bolsa_llamamientos.solicitud_portal_candidato_externa','SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN RAISE EXCEPTION 'B63: privilegio exterior fuera del contrato'; END IF;
END $acl$;
-- El migrador coloca tipos y catálogos falsos en su propio espacio temporal;
-- las fachadas nuevas deben resolver siempre los objetos de pg_catalog.
CREATE TEMP TABLE sombra_b63(unico boolean);
CREATE DOMAIN pg_temp.jsonb AS text;
CREATE TEMP TABLE pg_roles(rolname text,rolcanlogin boolean,rolinherit boolean,rolsuper boolean,rolcreatedb boolean,rolcreaterole boolean,rolreplication boolean,rolbypassrls boolean,rolconfig text[]);
DO $login$ BEGIN
 IF to_regrole('vec_externo_bolsa_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_bolsa_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_llamamientos_portal_externo TO vec_externo_bolsa_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
 END IF;
END $login$;
SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $frontera$
DECLARE x pg_catalog.bytea:=pg_catalog.convert_to('{"persona_ref":"per_b63_persona_sintetica_00001","perfil_activo_ref":"prf_b63_perfil_sintetico_00001","contexto_actor_ref":"vca_b63_contexto_sintetico_0001","vinculos":[{"tipo":"candidato","estado":"activo","referencia":"can_b63_candidato_sintetico_0001"}]}','UTF8');
BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.manifestar_disposicion_oferta_externo_v1('oferta:'||pg_catalog.repeat('c',64),'recibo:disposicion:'||pg_catalog.repeat('4',64),'can_b63_candidato_sintetico_0001','clave-b63-frontera',pg_catalog.clock_timestamp(),
   pg_catalog.convert_to('{"efecto_ref":"oferta:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","operacion":"bolsa.participaciones_propias.manifestar_disposicion"}','UTF8'),
   pg_catalog.convert_to('{"recurso_ref":"oferta:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","accion":"bolsa.participaciones_propias.manifestar_disposicion","tipo_recurso":"oferta_bolsa"}','UTF8'),
   '\x00'::pg_catalog.bytea,x,1,1,'\x00'::pg_catalog.bytea,'\x00'::pg_catalog.bytea,'\x00'::pg_catalog.bytea,'\x00'::pg_catalog.bytea);
  RAISE EXCEPTION 'B63: fachada aceptó proyección revocada/material inexistente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_portal_candidato_externo_v1('can_b63_candidato_sintetico_0001',pg_catalog.clock_timestamp(),ARRAY['contactado']);
  RAISE EXCEPTION 'B63: lector auxiliar sin marca exterior';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 PERFORM pg_catalog.set_config('vec_bolsa_llamamientos.marca_consumo_portal_externo','consulta'||pg_catalog.chr(31)||'can_b63_candidato_sintetico_0001'||pg_catalog.chr(31)||'falso'||pg_catalog.chr(31)||pg_catalog.repeat('0',64),true);
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_portal_candidato_externo_v1('can_b63_candidato_sintetico_0001',pg_catalog.clock_timestamp(),ARRAY['contactado']);
  RAISE EXCEPTION 'B63: marca exterior forjada válida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1('consulta','can_b63_candidato_sintetico_0001','falso');
  RAISE EXCEPTION 'B63: runtime pudo firmar su propia marca';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $frontera$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo B63-NUCLEOS-ACL-OK
