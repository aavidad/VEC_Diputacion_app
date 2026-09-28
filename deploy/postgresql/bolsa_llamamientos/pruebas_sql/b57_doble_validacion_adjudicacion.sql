-- Prueba focal B57 sobre PostgreSQL 18.4 efímero con la cadena completa.
-- Los dobles AD3 son locales a ROLLBACK: prueban las guardas de B57, no la
-- criptografía V3, que se acredita con la cadena y sus pruebas propias.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone='UTC';
CREATE ROLE vec_b57_runtime_prueba NOLOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b57_runtime_prueba;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE AS $f$
 SELECT 'decision:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',repeat('a',64),repeat('b',64),
  'auditoria:'||gen_random_uuid(),clock_timestamp(),true
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_confirmacion_adjudicacion_oferta_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE AS $f$
 SELECT 'decision:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',repeat('a',64),repeat('b',64),
  'auditoria:'||gen_random_uuid(),clock_timestamp(),true
$f$;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:b57:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:b57:1','bolsa:b57:1',1,encode(sha256('{}'::bytea),'hex'),'inst:b57:1',1,repeat('c',64),'categoria:rpt:aux',
  'per_aaaaaaaaaaaaaaaaaaaaaaaa',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada VALUES
 ('inst:b57:1',1,1,'part:b57:1',1),('inst:b57:1',1,2,'part:b57:2',2),('inst:b57:1',1,3,'part:b57:3',3);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 SELECT p,'disponible',now()-interval '9 days',NULL,NULL,'alta','per_aaaaaaaaaaaaaaaaaaaaaaaa',now()-interval '9 days',
   'clave:'||p,'recibo:'||p FROM unnest(ARRAY['part:b57:1','part:b57:2','part:b57:3']) p;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa
 (politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
 VALUES('politica:b57:1','bolsa:b57:1',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Ejemplo',
 'per_aaaaaaaaaaaaaaaaaaaaaaaa',now()-interval '10 days',NULL,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version
 (bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,recibo_ref,
  publicada_en,decision_ref,auditoria_ref)
 VALUES('bolsa:b57:1',1,'{"plazo":{"unidad":"horas_naturales","cantidad":48,"computo":"continuo_utc","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo","requiere_segunda_validacion":true},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}',
 repeat('d',64),true,'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-politica-b57',0,'recibo:politica-ofertas:'||repeat('e',64),
 now()-interval '3 days','decision:politica:b57','auditoria:politica:b57');
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,
  huella_comando_sha256,decision_ref)
 VALUES('oferta:'||repeat('7',64),'recibo:oferta:'||repeat('7',64),'bolsa:b57:1',
 'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-oferta-b57',
 '{"categoria":"Auxiliar","centro":"Centro Norte","fecha_inicio":"2026-10-01","descripcion":"Cobertura temporal"}',
 '{"politica_version":1}',now()-interval '3 days',now()-interval '1 day',repeat('f',64),'decision:oferta:b57');
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,
  huella_comando_sha256,decision_ref)
 VALUES('oferta:'||repeat('6',64),'recibo:oferta:'||repeat('6',64),'bolsa:b57:1',
 'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-oferta-b57-legacy',
 '{"categoria":"Auxiliar","centro":"Centro Este","fecha_inicio":"2026-10-01","descripcion":"Sin disposición"}',
 '{"politica_version":1}',now()-interval '3 days',now()-interval '1 day',repeat('6',64),'decision:oferta:b57:legacy');
INSERT INTO vec_bolsa_llamamientos.capacidad_oferta
 VALUES('oferta:'||repeat('7',64),2,'unidad:rrhh','ambito:bolsa','recibo:oferta:'||repeat('7',64),now()-interval '3 days');
INSERT INTO vec_bolsa_llamamientos.disposicion_oferta
 VALUES('oferta:'||repeat('7',64),'part:b57:1',now()-interval '2 days','clave-disp-b57-1','recibo:disposicion:'||repeat('1',64)),
       ('oferta:'||repeat('7',64),'part:b57:2',now()-interval '2 days','clave-disp-b57-2','recibo:disposicion:'||repeat('2',64)),
       ('oferta:'||repeat('7',64),'part:b57:3',now()-interval '2 days','clave-disp-b57-3','recibo:disposicion:'||repeat('3',64));
SET LOCAL session_replication_role=origin;

DO $prueba$
DECLARE
 o text:='oferta:'||repeat('7',64); bolsa text:='bolsa:b57:1';
 actor_a text:='per_aaaaaaaaaaaaaaaaaaaaaaaa'; actor_b text:='per_bbbbbbbbbbbbbbbbbbbbbbbb';
 actor_c text:='per_cccccccccccccccccccccccc';
 cap bytea:=convert_to('{"efecto_ref":"bolsa:b57:1"}','UTF8');
 dec_a bytea:=convert_to('{"principal_id":"per_aaaaaaaaaaaaaaaaaaaaaaaa","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:b57:1","tipo_recurso":"bolsa_constituida"}','UTF8');
 prep text; rec1 text; rec2 text; x record; y record; n integer; contexto_confirmacion text;
 publicada timestamptz(6); vence timestamptz(6); plazo jsonb; datos jsonb;
 material text; contexto_huella text; decision_publicar bytea;
BEGIN
 SET LOCAL ROLE vec_b57_runtime_prueba;
 dec_a:=convert_to((convert_from(dec_a,'UTF8')::jsonb||jsonb_build_object(
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{}}','UTF8')),'hex')))::text,'UTF8');
 SELECT * INTO x FROM vec_bolsa_llamamientos.resolver_oferta_v2('oferta:'||repeat('6',64),
  'recibo:resolucion-oferta:'||encode(sha256(convert_to('oferta:'||repeat('6',64)||chr(31)||'1'||chr(31)||'clave-b57-legacy-directo','UTF8')),'hex'),
  bolsa,NULL,actor_a,'clave-b57-legacy-directo',1,cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF x.oferta->>'estado'<>'llamamiento_directo' THEN
  RAISE EXCEPTION 'B57: oferta legacy sin cubrir perdió llamamiento directo'; END IF;
 publicada:=clock_timestamp();vence:=publicada+interval '48 hours';
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:bolsa:b57:1:1','huella_catalogo',repeat('d',64),
  'unidad','horas_naturales','cantidad',48,'computo','continuo_utc','municipio_sede','18087',
  'ultimo_dia',((vence-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text,
  'ejemplo',true,'calendarios',jsonb_build_array('calendario:utc-continuo:v1'),'politica_version',1,
  'apertura_en',to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vence_en',to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 datos:='{"categoria":"Auxiliar","centro":"Centro Sur","fecha_inicio":"2026-10-01","descripcion":"Cobertura de dos plazas","numero_plazas":2}'::jsonb;
 material:=encode(sha256(convert_to(array_to_string(ARRAY[bolsa,
  to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  plazo->>'regla_ref',plazo->>'huella_catalogo',plazo->>'unidad',plazo->>'cantidad',plazo->>'computo',
  plazo->>'municipio_sede',plazo->>'ultimo_dia',plazo->>'politica_version','calendario:utc-continuo:v1',
  plazo->>'apertura_en',plazo->>'vence_en','1'],chr(31)),'UTF8')),'hex');
 contexto_huella:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||material||'"}}','UTF8')),'hex');
 decision_publicar:=convert_to((convert_from(dec_a,'UTF8')::jsonb||
   jsonb_build_object('contexto_recurso_huella_sha256',contexto_huella))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v3('oferta:'||repeat('8',64),
   'recibo:oferta:'||repeat('8',64),bolsa,actor_a,'clave-oferta-b57-2',datos,plazo,
   publicada,vence,cap,decision_publicar,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
  RAISE EXCEPTION 'B57: autorización de una plaza permitió publicar dos';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 material:=encode(sha256(convert_to(array_to_string(ARRAY[bolsa,
  to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  plazo->>'regla_ref',plazo->>'huella_catalogo',plazo->>'unidad',plazo->>'cantidad',plazo->>'computo',
  plazo->>'municipio_sede',plazo->>'ultimo_dia',plazo->>'politica_version','calendario:utc-continuo:v1',
  plazo->>'apertura_en',plazo->>'vence_en','2'],chr(31)),'UTF8')),'hex');
 contexto_huella:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||material||'"}}','UTF8')),'hex');
 decision_publicar:=convert_to((convert_from(dec_a,'UTF8')::jsonb||
   jsonb_build_object('contexto_recurso_huella_sha256',contexto_huella))::text,'UTF8');
 SELECT * INTO x FROM vec_bolsa_llamamientos.publicar_oferta_v3('oferta:'||repeat('8',64),
   'recibo:oferta:'||repeat('8',64),bolsa,actor_a,'clave-oferta-b57-2',datos,plazo,
   publicada,vence,cap,decision_publicar,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF x.reutilizada OR (x.oferta->'datos'->>'numero_plazas')::integer<>2 THEN
  RAISE EXCEPTION 'B57: capacidad de dos plazas no quedó duradera'; END IF;
 -- La ruta B28 no es invocable desde SQL por el LOGIN runtime.
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v1(o,'recibo:resolucion-oferta:'||repeat('9',64),bolsa,
   'part:b57:1',actor_a,'clave-resolucion-vieja',cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B57: bypass B28 permitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 rec1:='recibo:resolucion-oferta:'||encode(sha256(convert_to(o||chr(31)||'1'||chr(31)||'clave-b57-plaza-uno','UTF8')),'hex');
 rec2:='recibo:resolucion-oferta:'||encode(sha256(convert_to(o||chr(31)||'2'||chr(31)||'clave-b57-plaza-dos','UTF8')),'hex');
 IF rec1=rec2 THEN RAISE EXCEPTION 'B57: recibos de plazas colisionan'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec1,bolsa,'part:b57:1',actor_a,'clave-b57-plaza-uno',1,
   cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:otra','ambito:bolsa');
  RAISE EXCEPTION 'B57: ámbito de otra unidad aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec1,bolsa,'part:b57:2',actor_a,'clave-b57-plaza-uno',1,
   cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
  RAISE EXCEPTION 'B57: se aceptó participante fuera de orden';
 EXCEPTION WHEN SQLSTATE 'VBO04' THEN NULL; END;
 SELECT * INTO x FROM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec1,bolsa,'part:b57:1',actor_a,
  'clave-b57-plaza-uno',1,cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF x.reutilizada OR x.oferta->>'estado'<>'pendiente_segunda_validacion'
    OR (x.oferta->'preparacion'->>'numero_de_plaza')::int<>1 THEN
  RAISE EXCEPTION 'B57: primera preparación no durable: %',x.oferta; END IF;
 prep:=x.oferta->'preparacion'->>'recibo_ref';
 contexto_confirmacion:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"bolsa_ref":"'||bolsa||
  '","numero_de_plaza":"1","oferta_ref":"'||o||'"}}','UTF8')),'hex');
 -- Otra concesión viva del mismo actor no sustituye a otra persona.
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(o,bolsa,1,prep,actor_a,'clave-confirmar-uno',
   convert_to(jsonb_build_object('efecto_ref',prep)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id',actor_a,'accion','bolsa.oferta.adjudicacion.confirmar','modulo_id','bolsa',
    'finalidad','confirmar_adjudicacion_oferta','recurso_ref',prep,'tipo_recurso','preparacion_adjudicacion_oferta','contexto_recurso_huella_sha256',contexto_confirmacion)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B57: una persona confirmó sus dos actos';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(o,bolsa,1,prep,actor_b,'clave-confirmar-uno',
   convert_to(jsonb_build_object('efecto_ref',prep)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id',actor_b,'accion','bolsa.oferta.adjudicacion.confirmar','modulo_id','bolsa',
    'finalidad','confirmar_adjudicacion_oferta','recurso_ref',prep,'tipo_recurso','preparacion_adjudicacion_oferta',
    'contexto_recurso_huella_sha256',repeat('0',64))::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B57: ámbito de confirmación ajeno aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO x FROM vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(o,bolsa,1,prep,actor_b,'clave-confirmar-uno',
   convert_to(jsonb_build_object('efecto_ref',prep)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id',actor_b,'accion','bolsa.oferta.adjudicacion.confirmar','modulo_id','bolsa',
    'finalidad','confirmar_adjudicacion_oferta','recurso_ref',prep,'tipo_recurso','preparacion_adjudicacion_oferta','contexto_recurso_huella_sha256',contexto_confirmacion)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF x.reutilizada OR jsonb_array_length(x.oferta->'adjudicaciones')<>1
    OR x.oferta->'adjudicaciones'->0->>'participacion_ref'<>'part:b57:1' THEN
  RAISE EXCEPTION 'B57: primera plaza incorrecta: %',x.oferta; END IF;
 SELECT * INTO y FROM vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(o,bolsa,1,prep,actor_b,'clave-confirmar-uno',
   convert_to(jsonb_build_object('efecto_ref',prep)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id',actor_b,'accion','bolsa.oferta.adjudicacion.confirmar','modulo_id','bolsa',
    'finalidad','confirmar_adjudicacion_oferta','recurso_ref',prep,'tipo_recurso','preparacion_adjudicacion_oferta','contexto_recurso_huella_sha256',contexto_confirmacion)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT y.reutilizada OR y.oferta->'adjudicaciones'->0->>'recibo_ref' IS DISTINCT FROM
    x.oferta->'adjudicaciones'->0->>'recibo_ref' THEN RAISE EXCEPTION 'B57: replay creó otro recibo'; END IF;
 SELECT * INTO y FROM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec1,bolsa,'part:b57:1',actor_a,
  'clave-b57-plaza-uno',1,cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF NOT y.reutilizada OR y.oferta->>'recibo_preparacion_replay' IS DISTINCT FROM prep THEN
  RAISE EXCEPTION 'B57: replay de preparación perdió su recibo'; END IF;
 SELECT * INTO x FROM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec2,bolsa,'part:b57:2',actor_a,
  'clave-b57-plaza-dos',2,cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF x.reutilizada OR (x.oferta->'preparacion'->>'numero_de_plaza')::int<>2 THEN
  RAISE EXCEPTION 'B57: segunda plaza no tomó siguiente orden'; END IF;
 prep:=x.oferta->'preparacion'->>'recibo_ref';
 contexto_confirmacion:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"bolsa_ref":"'||bolsa||
  '","numero_de_plaza":"2","oferta_ref":"'||o||'"}}','UTF8')),'hex');
 SELECT * INTO x FROM vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(o,bolsa,2,prep,actor_c,'clave-confirmar-dos',
   convert_to(jsonb_build_object('efecto_ref',prep)::text,'UTF8'),
   convert_to(jsonb_build_object('principal_id',actor_c,'accion','bolsa.oferta.adjudicacion.confirmar','modulo_id','bolsa',
    'finalidad','confirmar_adjudicacion_oferta','recurso_ref',prep,'tipo_recurso','preparacion_adjudicacion_oferta','contexto_recurso_huella_sha256',contexto_confirmacion)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF x.oferta->>'estado'<>'adjudicada' OR jsonb_array_length(x.oferta->'adjudicaciones')<>2 THEN
  RAISE EXCEPTION 'B57: oferta de dos plazas no cerró'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_oferta_v2(o,rec2,bolsa,'part:b57:3',actor_a,'clave-b57-plaza-dos',3,
   cap,dec_a,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
  RAISE EXCEPTION 'B57: tercera adjudicación excedió capacidad';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 RESET ROLE;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.adjudicacion_oferta WHERE oferta_ref=o;
 IF n<>2 OR (SELECT count(*) FROM vec_bolsa_llamamientos.evento_adjudicacion_oferta WHERE oferta_ref=o)<>4 THEN
  RAISE EXCEPTION 'B57: historia o outbox incompletos'; END IF;
END $prueba$;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version
 (bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,recibo_ref,
  publicada_en,decision_ref,auditoria_ref)
 VALUES('bolsa:b57:1',2,'{"plazo":{"unidad":"horas_naturales","cantidad":48,"computo":"continuo_utc","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo","requiere_segunda_validacion":false},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}',
 repeat('f',64),true,'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-politica-b57-false',1,'recibo:politica-ofertas:'||repeat('f',64),
 now()-interval '3 days','decision:politica:b57:false','auditoria:politica:b57:false');
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,
  huella_comando_sha256,decision_ref)
 VALUES('oferta:'||repeat('9',64),'recibo:oferta:'||repeat('9',64),'bolsa:b57:1',
 'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-oferta-b57-false',
 '{"categoria":"Auxiliar","centro":"Centro Oeste","fecha_inicio":"2026-10-01","descripcion":"Una plaza"}',
 '{"politica_version":2}',now()-interval '3 days',now()-interval '1 day',repeat('9',64),'decision:oferta:b57:false');
INSERT INTO vec_bolsa_llamamientos.capacidad_oferta
 VALUES('oferta:'||repeat('9',64),1,'unidad:rrhh','ambito:bolsa','recibo:oferta:'||repeat('9',64),now()-interval '3 days');
INSERT INTO vec_bolsa_llamamientos.disposicion_oferta
 VALUES('oferta:'||repeat('9',64),'part:b57:1',now()-interval '2 days','clave-disp-b57-false','recibo:disposicion:'||repeat('9',64));
SET LOCAL session_replication_role=origin;
DO $sin_segunda$
DECLARE o text:='oferta:'||repeat('9',64); bolsa text:='bolsa:b57:1'; x record;
 cap bytea:=convert_to('{"efecto_ref":"bolsa:b57:1"}','UTF8');
 dec bytea:=convert_to('{"principal_id":"per_aaaaaaaaaaaaaaaaaaaaaaaa","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:b57:1","tipo_recurso":"bolsa_constituida"}','UTF8');
 clave text:='clave-b57-sin-segunda'; recibo text; politica_false jsonb;
BEGIN
 dec:=convert_to((convert_from(dec,'UTF8')::jsonb||jsonb_build_object(
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{}}','UTF8')),'hex')))::text,'UTF8');
 SELECT politica INTO politica_false FROM vec_bolsa_llamamientos.politica_ofertas_version
 WHERE bolsa_ref=bolsa AND version=2;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(bolsa,2,politica_false,
   'per_aaaaaaaaaaaaaaaaaaaaaaaa','clave-politica-prohibida',
   'recibo:politica-ofertas:'||repeat('4',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B57: se publicó nueva política de adjudicación sin segunda persona';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 recibo:='recibo:resolucion-oferta:'||encode(sha256(convert_to(o||chr(31)||'1'||chr(31)||clave,'UTF8')),'hex');
 SET LOCAL ROLE vec_b57_runtime_prueba;
 SELECT * INTO x FROM vec_bolsa_llamamientos.resolver_oferta_v2(o,recibo,bolsa,'part:b57:1',
  'per_aaaaaaaaaaaaaaaaaaaaaaaa',clave,1,cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF x.reutilizada OR x.oferta->>'estado'<>'pendiente_segunda_validacion' OR jsonb_array_length(x.oferta->'adjudicaciones')<>0 THEN
  RAISE EXCEPTION 'B57: política false eludió la segunda persona'; END IF;
 RESET ROLE;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.adjudicacion_oferta WHERE oferta_ref=o) THEN
  RAISE EXCEPTION 'B57: false produjo adjudicación sin segunda persona'; END IF;
END $sin_segunda$;
SELECT 'OK B57 doble validacion por plaza' AS resultado;
ROLLBACK;
