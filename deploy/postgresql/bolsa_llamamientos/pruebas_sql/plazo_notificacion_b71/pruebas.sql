\set ON_ERROR_STOP on
-- Ensayo transaccional en clon sintetico. El doble AD3 solo sustituye el
-- consumidor criptografico; B71, ACL, trigger, tablas y recibos son reales.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:b71:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
        repeat('a',64),repeat('b',64),'auditoria:b71:'||gen_random_uuid(),clock_timestamp(),true
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:b71:politica:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
        repeat('a',64),repeat('b',64),'auditoria:b71:politica:'||gen_random_uuid(),clock_timestamp(),true
$f$;
DO $f$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)','EXECUTE')
  OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)','EXECUTE')
 THEN RAISE EXCEPTION 'B71: ACL ejecutor incorrecta'; END IF;
END $f$;
CREATE SCHEMA prueba_b71 AUTHORIZATION vec_bolsa_llamamientos_propietario;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE prueba_b71.estado(bolsa text, notificada timestamptz, vence timestamptz, primera jsonb, recibo text);
DO $f$
DECLARE b text; n timestamptz; v timestamptz; p jsonb; h text;
BEGIN
 SELECT bolsa_ref INTO b FROM vec_bolsa_llamamientos.constitucion ORDER BY bolsa_ref LIMIT 1;
 IF b IS NULL THEN RAISE EXCEPTION 'B71: falta bolsa sintetica del clon'; END IF;
 n:=date_trunc('second',clock_timestamp())-interval '59 minutes 55 seconds';
 v:=n+interval '1 hour';
 p:=jsonb_build_object('plazo',jsonb_build_object('unidad','horas_naturales','cantidad',1,
     'computo','continuo_utc','municipio_sede','18087','inicio','notificacion'),
     'adjudicacion',jsonb_build_object('criterio','orden_vigente','elegibilidad','disposicion_en_plazo'),
     'no_cubierta',jsonb_build_object('accion','llamamiento_directo','condicion','sin_disposiciones_elegibles'));
 -- La política sin inicio ya no se puede publicar como versión nueva.
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(b,0,
   jsonb_set(p,'{plazo}',(p->'plazo') - 'inicio'::text),'per_actoractoractoractoractor',
   'clave-politica-sin-inicio-b71','recibo:politica-ofertas:'||repeat('0',64),
   convert_to(jsonb_build_object('efecto_ref',b)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id','per_actoractoractoractoractor',
     'accion','bolsa.politica_ofertas.publicar','modulo_id','bolsa','tipo_recurso','bolsa_constituida',
     'finalidad','gobierno_politica_ofertas_bolsa','recurso_ref',b)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B71: política sin inicio aceptada';
 EXCEPTION WHEN sqlstate '22023' THEN NULL; END;
 PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(b,0,p,
  'per_actoractoractoractoractor','clave-politica-b71','recibo:politica-ofertas:'||repeat('1',64),
  convert_to(jsonb_build_object('efecto_ref',b)::text,'UTF8'),
  convert_to(jsonb_build_object('principal_id','per_actoractoractoractoractor',
    'accion','bolsa.politica_ofertas.publicar','modulo_id','bolsa','tipo_recurso','bolsa_constituida',
    'finalidad','gobierno_politica_ofertas_bolsa','recurso_ref',b)::text,'UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_outbox WHERE bolsa_ref=b)<>1 THEN
  RAISE EXCEPTION 'B71: política sin outbox'; END IF;
 INSERT INTO prueba_b71.estado(bolsa,notificada,vence) VALUES(b,n,v);
END $f$;
-- Una versión anterior de cuatro claves conserva su replay exacto.
DO $f$
DECLARE b text; p jsonb; h text; r record;
BEGIN
 SELECT bolsa_ref INTO b FROM vec_bolsa_llamamientos.constitucion ORDER BY bolsa_ref OFFSET 1 LIMIT 1;
 IF b IS NULL THEN RAISE EXCEPTION 'B71: falta segunda bolsa sintetica'; END IF;
 p:=jsonb_build_object('plazo',jsonb_build_object('unidad','horas_naturales','cantidad',1,
    'computo','continuo_utc','municipio_sede','18087'),
    'adjudicacion',jsonb_build_object('criterio','orden_vigente','elegibilidad','disposicion_en_plazo'),
    'no_cubierta',jsonb_build_object('accion','llamamiento_directo','condicion','sin_disposiciones_elegibles'));
 h:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,
  recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(b,1,p,h,true,'per_actoractoractoractoractor','clave-legada-b71',0,
  'recibo:politica-ofertas:'||repeat('9',64),clock_timestamp(),'decision:politica:b71:legada','auditoria:politica:b71:legada');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(b,0,p,
  'per_actoractoractoractoractor','clave-legada-b71','recibo:politica-ofertas:'||repeat('9',64),
  convert_to(jsonb_build_object('efecto_ref',b)::text,'UTF8'),
  convert_to(jsonb_build_object('principal_id','per_actoractoractoractoractor',
    'accion','bolsa.politica_ofertas.publicar','modulo_id','bolsa','tipo_recurso','bolsa_constituida',
    'finalidad','gobierno_politica_ofertas_bolsa','recurso_ref',b)::text,'UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT r.reutilizada OR r.politica->>'recibo_ref' IS DISTINCT FROM 'recibo:politica-ofertas:'||repeat('9',64)
 THEN RAISE EXCEPTION 'B71: replay histórico distinto'; END IF;
END $f$;
CREATE FUNCTION prueba_b71.publicar(p_clave text,p_notificada timestamptz)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE e record; plazo jsonb; datos jsonb; publicada timestamptz:=clock_timestamp(); vence timestamptz;
 v_huella text; v_material text; v_contexto text; v_decision bytea; v_sufijo text;
BEGIN
 SELECT * INTO STRICT e FROM prueba_b71.estado;
 vence:=p_notificada+interval '1 hour';
 v_huella:=encode(sha256(convert_to((SELECT politica::text FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref=e.bolsa AND version=1),'UTF8')),'hex');
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:'||e.bolsa||':1','huella_catalogo',v_huella,
  'unidad','horas_naturales','cantidad',1,'computo','continuo_utc','municipio_sede','18087',
  'ultimo_dia',((vence-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text,
  'ejemplo',true,'politica_version',1,'calendarios',jsonb_build_array('calendario:utc-continuo:v1'),
  'apertura_en',to_char(p_notificada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vence_en',to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'notificacion',jsonb_build_object('notificada_en',to_char(p_notificada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'referencia_correo','correo:b71:1','huella_correo_sha256',repeat('c',64),'fuente','correo_externo_declarado_rrhh'));
 datos:='{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-03","descripcion":"Sustitución temporal"}'::jsonb;
 v_material:=encode(sha256(convert_to(array_to_string(ARRAY[
  e.bolsa,to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  plazo->>'regla_ref',plazo->>'huella_catalogo',plazo->>'unidad',plazo->>'cantidad',
  plazo->>'computo',plazo->>'municipio_sede',plazo->>'ultimo_dia',plazo->>'politica_version',
  'calendario:utc-continuo:v1',plazo->>'apertura_en',plazo->>'vence_en','1',
  plazo#>>'{notificacion,notificada_en}',plazo#>>'{notificacion,referencia_correo}',
  plazo#>>'{notificacion,huella_correo_sha256}',plazo#>>'{notificacion,fuente}'],chr(31)),'UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||v_material||'"}}','UTF8')),'hex');
 v_decision:=convert_to(jsonb_build_object('principal_id','per_actoractoractoractoractor','accion','llamamiento.emitir.v1',
  'modulo_id','bolsa','tipo_recurso','bolsa_constituida','finalidad','gestion_llamamientos_bolsa',
  'recurso_ref',e.bolsa,'contexto_recurso_huella_sha256',v_contexto)::text,'UTF8');
 v_sufijo:=encode(sha256(convert_to(p_clave,'UTF8')),'hex');
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.publicar_oferta_v4(
  'oferta:'||v_sufijo,'recibo:oferta:'||v_sufijo,e.bolsa,'per_actoractoractoractoractor',p_clave,
  datos,plazo,publicada,vence,convert_to(jsonb_build_object('efecto_ref',e.bolsa,'nonce',gen_random_uuid())::text,'UTF8'),
  v_decision,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa',1);
END $f$;
DO $f$
DECLARE e record; r record;
BEGIN
 SELECT * INTO STRICT e FROM prueba_b71.estado;
 SELECT * INTO STRICT r FROM prueba_b71.publicar('clave-oferta-b71',e.notificada);
 IF r.reutilizada OR r.oferta->>'recibo_ref' IS NULL THEN RAISE EXCEPTION 'B71: publicacion sin recibo'; END IF;
 UPDATE prueba_b71.estado SET primera=r.oferta,recibo=r.oferta->>'recibo_ref';
END $f$;
SELECT pg_sleep(6);
DO $f$
DECLARE e record; r record; n integer;
BEGIN
 SELECT * INTO STRICT e FROM prueba_b71.estado;
 SELECT * INTO STRICT r FROM prueba_b71.publicar('clave-oferta-b71',e.notificada);
 IF NOT r.reutilizada OR r.oferta->>'recibo_ref' IS DISTINCT FROM e.recibo
 THEN RAISE EXCEPTION 'B71: replay vencido distinto'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia='clave-oferta-b71';
 IF n<>1 THEN RAISE EXCEPTION 'B71: replay duplicado: %',n; END IF;
 BEGIN PERFORM prueba_b71.publicar('clave-oferta-b71',e.notificada+interval '1 second');
  RAISE EXCEPTION 'B71: notificacion cambiada aceptada';
 EXCEPTION WHEN sqlstate 'VBO01' THEN NULL; END;
 BEGIN PERFORM prueba_b71.publicar('clave-vencida-b71',e.notificada);
  RAISE EXCEPTION 'B71: vencida aceptada';
 EXCEPTION WHEN sqlstate '22023' THEN NULL; END;
 BEGIN PERFORM prueba_b71.publicar('clave-futura-b71',clock_timestamp()+interval '1 hour');
  RAISE EXCEPTION 'B71: futura aceptada';
 EXCEPTION WHEN sqlstate '22023' THEN NULL; END;
 RAISE NOTICE 'B71: publicacion, replay vencido, plazos rechazados, recibo e historia OK';
END $f$;
-- El catálogo también puede publicar días hábiles; el trigger exige la
-- notificación sin cambiar la validación de calendarios de la aplicación.
DO $f$
DECLARE e record; p jsonb; h text; v timestamptz; plazo jsonb; v_referencia text;
BEGIN
 SELECT * INTO STRICT e FROM prueba_b71.estado;
 p:=jsonb_build_object('plazo',jsonb_build_object('unidad','dias_habiles','cantidad',2,
   'computo','administrativo','municipio_sede','18087','inicio','notificacion'),
   'adjudicacion',jsonb_build_object('criterio','orden_vigente','elegibilidad','disposicion_en_plazo'),
   'no_cubierta',jsonb_build_object('accion','llamamiento_directo','condicion','sin_disposiciones_elegibles'));
 h:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,
  recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(e.bolsa,2,p,h,true,'per_actoractoractoractoractor','clave-politica-b71-v2',1,
  'recibo:politica-ofertas:'||repeat('2',64),clock_timestamp(),'decision:politica:b71:2','auditoria:politica:b71:2');
 v:=(((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date+3)::timestamp AT TIME ZONE 'Europe/Madrid');
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:'||e.bolsa||':2','huella_catalogo',h,
  'unidad','dias_habiles','cantidad',2,'computo','administrativo','municipio_sede','18087',
  'ultimo_dia',((v-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text,
  'ejemplo',true,'politica_version',2,'calendarios',jsonb_build_array('cal:synthetic'),
  'notificacion',jsonb_build_object('notificada_en',to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'referencia_correo','correo:b71:2','huella_correo_sha256',repeat('d',64),'fuente','correo_externo_declarado_rrhh'));
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
  oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
  publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES('oferta:'||repeat('e',64),'recibo:oferta:'||repeat('e',64),e.bolsa,
  'per_actoractoractoractoractor','clave-dias-b71','{}'::jsonb,plazo,
  clock_timestamp(),v,repeat('e',64),'decision:oferta:b71:dias');
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
   oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
   publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
  VALUES('oferta:'||repeat('f',64),'recibo:oferta:'||repeat('f',64),e.bolsa,
   'per_actoractoractoractoractor','clave-dias-sin-notif-b71','{}'::jsonb,plazo-'notificacion',
   clock_timestamp(),v,repeat('f',64),'decision:oferta:b71:sin-notif');
  RAISE EXCEPTION 'B71: días sin notificación aceptados';
 EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 -- El acceso directo a SQL tampoco conserva identificadores personales.
 FOREACH v_referencia IN ARRAY ARRAY['correo:dni-12345678Z','correo:X1234567L','correo:pasaporte:extracto-1'] LOOP
  BEGIN
   INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
    oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
    publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
   VALUES('oferta:'||repeat('f',64),'recibo:oferta:'||repeat('f',64),e.bolsa,
    'per_actoractoractoractoractor','clave-dias-identidad-b71','{}'::jsonb,
    jsonb_set(plazo,'{notificacion,referencia_correo}',to_jsonb(v_referencia)),
    clock_timestamp(),v,repeat('f',64),'decision:oferta:b71:identidad');
   RAISE EXCEPTION 'B71: referencia con documento personal aceptada';
  EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 END LOOP;
 RAISE NOTICE 'B71: días configurables y notificación obligatoria OK';
END $f$;
ROLLBACK;
