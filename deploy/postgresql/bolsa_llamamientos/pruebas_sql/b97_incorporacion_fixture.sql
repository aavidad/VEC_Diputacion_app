\set ON_ERROR_STOP on
-- B97: escenario sintético sobre PostgreSQL 18 desechable con B96/BC9/CC11,
-- CA38/CTX18 y AD229 instaladas. Los dobles transaccionales aíslan la lógica
-- Bolsa; este fichero NO acredita criptografía V3 ni identidad real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='30s';

-- Fuente histórica, política, identidad y consumo V3 con contrato exacto.
-- ROLLBACK final restaura íntegramente definiciones y datos.
CREATE OR REPLACE FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
 p_convocatoria_ref text,p_categoria_ref text,p_version_sha256 text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE s vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
BEGIN
 SELECT * INTO STRICT s FROM vec_bolsa_llamamientos.solicitud_inscripcion
 WHERE convocatoria_ref=p_convocatoria_ref AND categoria_ref=p_categoria_ref
  AND version_sha256=p_version_sha256 LIMIT 1;
 RETURN jsonb_build_object('convocatoria_ref',s.convocatoria_ref,
  'convocatoria_id',s.convocatoria_id,'secuencia',s.secuencia,
  'version_sha256',s.version_sha256,'categoria_ref',s.categoria_ref,
  'identificador_publico',s.identificador_publico,'bases_ref',s.bases_ref,
  'catalogo_ref',s.catalogo_ref,'catalogo_version',s.catalogo_version,
  'catalogo_sha256',s.catalogo_sha256,
  'politica_catalogo_ref',s.politica_catalogo_ref,
  'politica_catalogo_version',s.politica_catalogo_version,
  'politica_catalogo_sha256',s.politica_catalogo_sha256,
  'formulario_ref',s.formulario_ref,'formulario_version',s.formulario_version,
  'formulario_sha256',s.formulario_sha256,
  'requisitos_sha256',s.requisitos_sha256,
  'plazos_inscripcion',jsonb_build_array(jsonb_build_object(
   'plazo_ref',s.plazo_ref,'plazo_abre_en',s.plazo_abre_en,
   'plazo_cierra_en',s.plazo_cierra_en)));
END $f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE s vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
BEGIN
 IF current_setting('b97.fixture.politica_ausente',true)='true' THEN
  RAISE EXCEPTION 'B97 fixture: política ausente' USING ERRCODE='B9601';
 END IF;
 SELECT * INTO STRICT s FROM vec_bolsa_llamamientos.solicitud_inscripcion
 WHERE politica_catalogo_ref=p_catalogo_id AND politica_catalogo_version=p_version
  AND politica_catalogo_sha256=p_huella_sha256 LIMIT 1;
 RETURN jsonb_build_object('permitida',true,'politica_ref','asociacion.manual.acta',
  'version',p_version,'sha256',p_huella_sha256,
  'motivo_ref','revision.manual.b97','alcance_ref',s.convocatoria_ref);
END $f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text,
 p_referencias text[],p_idioma text) RETURNS jsonb
LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT jsonb_build_array(jsonb_build_object('categoria','Alfa B97'))
$f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
 p_solicitudes jsonb,p_idioma text) RETURNS jsonb
LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria','Alfa B97',
  'motivo_etiqueta_pendiente','Pendiente B97',
  'motivo_etiqueta_cumple','Cumple B97') ORDER BY x.orden),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) WITH ORDINALITY AS x(valor,orden)
$f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(
 p_persona_ref text) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT CASE WHEN current_setting('b97.fixture.sin_vinculo',true)='true'
  THEN jsonb_build_object('estado','sin_vinculo') ELSE
 jsonb_build_object('estado','acreditado','persona_ref',p_persona_ref,
  'persona_version',1,'candidato_ref','can_'||repeat('c',22),
  'vinculo_ref','vin_b97','version',1,'procedencia_ref','fuente.b97',
  'procedencia_version',1,'procedencia_sha256',repeat('d',64),
  'poblacion','externa','acreditado_en',clock_timestamp()) END
$f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(
 p_candidato_ref text) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT CASE WHEN p_candidato_ref='can_'||repeat('c',22) THEN
  jsonb_build_object('estado','acreditado',
   'persona',jsonb_build_object('ref','per_'||repeat('p',22),'version',1),
   'vinculo',jsonb_build_object('ref','vin_b97','version',1,
    'procedencia_ref','fuente.b97','procedencia_version',1,
    'procedencia_sha256',repeat('d',64),'poblacion','externa'))
  ELSE jsonb_build_object('estado','pendiente') END
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,
 consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  encode(sha256(p_decision),'hex'),
  'aud_v3_'||substr(encode(sha256(p_decision),'hex'),1,32),
  date_trunc('microseconds',clock_timestamp()),true
 FROM (SELECT convert_from(p_capacidad,'UTF8')::jsonb AS c) x
 CROSS JOIN (SELECT convert_from(p_decision,'UTF8')::jsonb AS d) y
$f$;

DO $datos$
DECLARE
 persona text:='per_'||repeat('p',22);
 convocatoria_id text:='proceso:bolsa:b97-fixture';
 convocatoria_ref text;
 solicitud_ref text;
 acta text:='acta:importacion-convoca:'||repeat('a',64);
 bolsa text:='bolsa:b97:fixture';
 instantanea text:='instantanea:b97:fixture';
 participacion text:='participacion:b97:fixture';
 bolsa_material bytea:=convert_to('{"bolsa":"b97"}','UTF8');
 instantanea_material bytea:=convert_to('{"instantanea":"b97"}','UTF8');
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 convocatoria_ref:='cv1_'||encode(sha256(convert_to(convocatoria_id,'UTF8')),'hex')||'_v1';
 solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
  persona||chr(31)||convocatoria_ref||chr(31)||'cat.alpha','UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.bolsa_constituida(
  bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
  vigente_desde,vigente_hasta,estado,registrada_en)
 VALUES(bolsa,1,encode(sha256(bolsa_material),'hex'),bolsa_material,'cat.alpha',
  ahora-interval '1 day',NULL,'vigente',ahora-interval '1 day');
 INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(
  instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
  bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
  referida_en,generada_en,registrada_en)
 VALUES(instantanea,1,encode(sha256(instantanea_material),'hex'),instantanea_material,
  bolsa,1,encode(sha256(bolsa_material),'hex'),1,
  ahora-interval '1 day',ahora-interval '1 day',ahora-interval '1 day');
 INSERT INTO vec_bolsa_llamamientos.constitucion(
  acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
  instantanea_ref,version_instantanea,huella_instantanea_sha256,
  categoria_ref,actor_ref,confirmada_en,registrada_en)
 VALUES(acta,bolsa,1,encode(sha256(bolsa_material),'hex'),
  instantanea,1,encode(sha256(instantanea_material),'hex'),
  'cat.alpha','actor:b97',ahora-interval '1 day',ahora-interval '1 day'+interval '1 second');
 INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(
  instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
 VALUES(instantanea,1,1,participacion,1);
 INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(
  participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
 VALUES(participacion,'can_'||repeat('c',22),acta,instantanea,1,ahora-interval '1 day');
 INSERT INTO vec_bolsa_llamamientos.recibo_carga_convoca(
  acta_ref,actor_ref,categoria_ref,bolsa_ref,huella_fichero_sha256,
  huella_entradas_sha256,huella_vinculos_sha256,recibo_constitucion,recibo_vinculos,
  decision_ref,auditoria_ref,consumida_en,registrada_en)
 VALUES(acta,'actor:b97','cat.alpha',bolsa,repeat('a',64),repeat('b',64),repeat('c',64),
  jsonb_build_object('acta_ref',acta,'bolsa_ref',bolsa,'instantanea_ref',instantanea,
   'version_bolsa',1,'version_instantanea',1),
  jsonb_build_object('nuevos',1,'existentes',0),
  'decision:b97:carga','aud_v3_'||repeat('e',32),ahora-interval '1 day',ahora-interval '1 day');
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion(
  solicitud_ref,persona_ref,contexto_actor_ref,contexto_actor_version,
  convocatoria_ref,convocatoria_id,secuencia,version_sha256,identificador_publico,
  categoria_ref,bases_ref,catalogo_ref,catalogo_version,catalogo_sha256,
  politica_catalogo_ref,politica_catalogo_version,politica_catalogo_sha256,
  formulario_ref,formulario_version,formulario_sha256,
  plazo_ref,plazo_abre_en,plazo_cierra_en,requisitos,requisitos_sha256,
  declaraciones,declaracion_ref,clave_sha256,material_sha256,canal,presentada_en)
 VALUES(solicitud_ref,persona,'vca_'||repeat('v',22),1,
  convocatoria_ref,convocatoria_id,1,repeat('f',64),'b97-fixture',
  'cat.alpha','bases.b97','categorias.b97',1,repeat('1',64),
  'politica.b97',1,repeat('2',64),'formulario.b97',1,repeat('3',64),
  'plazo.b97',ahora-interval '3 days',ahora-interval '1 day','[]'::jsonb,
  repeat('4',64),'[]'::jsonb,'declaracion_inscripcion_'||repeat('5',64),
  repeat('6',64),repeat('7',64),'externa_personal',ahora-interval '2 days');
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(solicitud_ref,1,'pendiente',persona,'prf_'||repeat('p',22),
  'cta_'||repeat('p',22),'[]','decision:b97:presentada',repeat('8',64),
  'aud_v3_'||repeat('9',32),ahora-interval '2 days'),
 (solicitud_ref,2,'admitida_a_convocatoria','per_'||repeat('r',22),
  'prf_'||repeat('r',22),'cta_'||repeat('r',22),'[]',
  'decision:b97:admitida',repeat('a',64),'aud_v3_'||repeat('b',32),ahora-interval '1 day');
END $datos$;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $recorrido$
DECLARE
 convocatoria_ref text:='cv1_'||encode(sha256(convert_to('proceso:bolsa:b97-fixture','UTF8')),'hex')||'_v1';
 solicitud_ref text;
 acta text:='acta:importacion-convoca:'||repeat('a',64);
 actor text:='per_'||repeat('r',22);
 perfil text:='prf_'||repeat('r',22);
 cuenta text:='cta_'||repeat('r',22);
 material text; recurso text; recurso_sha text;
 capacidad bytea; decision bytea; contexto bytea; captura jsonb;
 material_mal text; recurso_mal text; huella_mal text; capacidad_mal bytea;
 decision_mal bytea;
 actor_otro text; perfil_otro text; cuenta_otra text;
 primer jsonb; repetido jsonb; caso integer;
BEGIN
 solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
  'per_'||repeat('p',22)||chr(31)||convocatoria_ref||chr(31)||'cat.alpha','UTF8')),'hex');
 material:='{"esquema":"vec.bolsa.inscripcion.incorporar.v1",'
  ||'"solicitud_ref":'||to_json(solicitud_ref)::text||','
  ||'"evidencia_ref":'||to_json(acta)::text||','
  ||'"version_esperada":2,"clave_idempotencia":"ClaveB97Fixture0001"}';
 recurso:='{"ambitos":{"solicitud_ref":'||to_json(solicitud_ref)::text
  ||'},"atributos":{"material_sha256":"'||encode(sha256(convert_to(material,'UTF8')),'hex')||'"}}';
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 capacidad:=convert_to(jsonb_build_object(
  'operacion','bolsa.inscripcion.rrhh.incorporar',
  'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.incorporar.v1',
  'efecto_ref',solicitud_ref,'huella_efecto_sha256',recurso_sha)::text,'UTF8');
 contexto:=convert_to(jsonb_build_object('principal_ref',actor,
  'perfil_activo_ref',perfil,'cuenta_ref',cuenta)::text,'UTF8');
 captura:=jsonb_build_object('persona_ref',actor,'perfil_ref',perfil,
  'cuenta_ref',cuenta,'canal','interna_corporativa','idioma','es');
 material_mal:=replace(material,acta,'acta:importacion-convoca:'||repeat('f',64));
 recurso_mal:='{"ambitos":{"solicitud_ref":'||to_json(solicitud_ref)::text
  ||'},"atributos":{"material_sha256":"'||encode(sha256(convert_to(material_mal,'UTF8')),'hex')||'"}}';
 huella_mal:=encode(sha256(convert_to(recurso_mal,'UTF8')),'hex');
 capacidad_mal:=convert_to(jsonb_build_object(
  'operacion','bolsa.inscripcion.rrhh.incorporar',
  'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.incorporar.v1',
  'efecto_ref',solicitud_ref,'huella_efecto_sha256',huella_mal)::text,'UTF8');
 decision_mal:=convert_to(jsonb_build_object('decision_ref','decision:b97:acta-ajena',
  'principal_id',actor,'perfil_activo_ref',perfil,'concedida',true,
  'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
  'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
  'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',huella_mal,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.incorporar_inscripcion_v1(
   material_mal,captura,capacidad_mal,decision_mal,'{}'::bytea,contexto,1,1,
   '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
  RAISE EXCEPTION 'B97 fixture: acta ajena admitida';
 EXCEPTION WHEN SQLSTATE 'B9702' THEN NULL;
 END;
 PERFORM set_config('b97.fixture.politica_ausente','true',true);
 decision_mal:=convert_to(jsonb_build_object('decision_ref','decision:b97:sin-politica',
  'principal_id',actor,'perfil_activo_ref',perfil,'concedida',true,
  'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
  'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
  'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.incorporar_inscripcion_v1(
   material,captura,capacidad,decision_mal,'{}'::bytea,contexto,1,1,
   '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
  RAISE EXCEPTION 'B97 fixture: política ausente permitió asociación';
 EXCEPTION WHEN SQLSTATE 'B9703' THEN NULL;
 END;
 PERFORM set_config('b97.fixture.politica_ausente','false',true);
 PERFORM set_config('b97.fixture.sin_vinculo','true',true);
 decision_mal:=convert_to(jsonb_build_object('decision_ref','decision:b97:sin-vinculo',
  'principal_id',actor,'perfil_activo_ref',perfil,'concedida',true,
  'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
  'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
  'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.incorporar_inscripcion_v1(
   material,captura,capacidad,decision_mal,'{}'::bytea,contexto,1,1,
   '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
  RAISE EXCEPTION 'B97 fixture: persona sin vínculo incorporada';
 EXCEPTION WHEN SQLSTATE 'B9701' THEN NULL;
 END;
 PERFORM set_config('b97.fixture.sin_vinculo','false',true);
 FOR caso IN 1..2 LOOP
  decision:=convert_to(jsonb_build_object('decision_ref','decision:b97:'||caso,
   'principal_id',actor,'perfil_activo_ref',perfil,'concedida',true,
   'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
   'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
   'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
  IF caso=1 THEN
   primer:=vec_bolsa_llamamientos.incorporar_inscripcion_v1(
    material,captura,capacidad,decision,'{}'::bytea,contexto,1,1,
    '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
   IF primer->>'estado'<>'incorporada' OR primer->>'version'<>'3'
    OR primer->>'repetida'<>'false' OR primer->>'participacion_ref'<>'participacion:b97:fixture'
    OR primer->>'bolsa_ref'<>'bolsa:b97:fixture'
   THEN RAISE EXCEPTION 'B97 fixture: primer recibo incompleto %',primer; END IF;
  ELSE
   repetido:=vec_bolsa_llamamientos.incorporar_inscripcion_v1(
    material,captura,capacidad,decision,'{}'::bytea,contexto,1,1,
    '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
   IF repetido->>'repetida'<>'true' OR repetido->>'recibo_ref'<>primer->>'recibo_ref'
    OR repetido->>'decidida_en'<>primer->>'decidida_en'
   THEN RAISE EXCEPTION 'B97 fixture: replay divergente %',repetido; END IF;
  END IF;
 END LOOP;
 -- Una autorización nueva de actor, perfil o cuenta distintos no recupera
 -- el recibo histórico de la primera identidad RRHH.
 FOR caso IN 1..3 LOOP
  actor_otro:=CASE WHEN caso=1 THEN 'per_'||repeat('s',22) ELSE actor END;
  perfil_otro:=CASE WHEN caso=2 THEN 'prf_'||repeat('s',22) ELSE perfil END;
  cuenta_otra:=CASE WHEN caso=3 THEN 'cta_'||repeat('s',22) ELSE cuenta END;
  decision_mal:=convert_to(jsonb_build_object('decision_ref','decision:b97:identidad:'||caso,
   'principal_id',actor_otro,'perfil_activo_ref',perfil_otro,'concedida',true,
   'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
   'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
   'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
  BEGIN
   PERFORM vec_bolsa_llamamientos.incorporar_inscripcion_v1(
    material,jsonb_build_object('persona_ref',actor_otro,
     'perfil_ref',perfil_otro,'cuenta_ref',cuenta_otra,
     'canal','interna_corporativa','idioma','es'),
    capacidad,decision_mal,'{}'::bytea,
    convert_to(jsonb_build_object('principal_ref',actor_otro,
     'perfil_activo_ref',perfil_otro,'cuenta_ref',cuenta_otra)::text,'UTF8'),1,1,
    '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
   RAISE EXCEPTION 'B97 fixture: identidad RRHH % recuperó el recibo',caso;
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
  END;
 END LOOP;
 -- Otra clave con el mismo acta no recupera el recibo histórico.
 material_mal:=replace(material,'ClaveB97Fixture0001','ClaveB97Fixture9999');
 recurso_mal:='{"ambitos":{"solicitud_ref":'||to_json(solicitud_ref)::text
  ||'},"atributos":{"material_sha256":"'||encode(sha256(convert_to(material_mal,'UTF8')),'hex')||'"}}';
 huella_mal:=encode(sha256(convert_to(recurso_mal,'UTF8')),'hex');
 capacidad_mal:=convert_to(jsonb_build_object(
  'operacion','bolsa.inscripcion.rrhh.incorporar',
  'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.incorporar.v1',
  'efecto_ref',solicitud_ref,'huella_efecto_sha256',huella_mal)::text,'UTF8');
 decision_mal:=convert_to(jsonb_build_object('decision_ref','decision:b97:otra-clave',
  'principal_id',actor,'perfil_activo_ref',perfil,'concedida',true,
  'accion','bolsa.inscripcion.rrhh.incorporar','modulo_id','bolsa',
  'tipo_recurso','solicitud_inscripcion','finalidad','incorporar_inscripcion',
  'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',huella_mal,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.incorporar_inscripcion_v1(
   material_mal,captura,capacidad_mal,decision_mal,'{}'::bytea,contexto,1,1,
   '{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea);
  RAISE EXCEPTION 'B97 fixture: otra clave recuperó el recibo';
 EXCEPTION WHEN SQLSTATE 'B9603' THEN NULL;
 END;
END $recorrido$;
RESET SESSION AUTHORIZATION;
DO $conteos$
DECLARE v_convocatoria_ref text:='cv1_'||encode(sha256(convert_to(
 'proceso:bolsa:b97-fixture','UTF8')),'hex')||'_v1';
 solicitud text;
 detalle jsonb; lista jsonb; recibo text;
BEGIN
 solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
  'per_'||repeat('p',22)||chr(31)||v_convocatoria_ref||chr(31)||'cat.alpha','UTF8')),'hex');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
   WHERE x.solicitud_ref=solicitud AND x.version=3)<>1
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.inscripcion_acta_asociada x
   WHERE x.convocatoria_ref=v_convocatoria_ref AND x.categoria_ref='cat.alpha')<>1
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.inscripcion_incorporacion x
   WHERE x.solicitud_ref=solicitud)<>1
 THEN RAISE EXCEPTION 'B97 fixture: duplicación en replay'; END IF;
 SELECT r.recibo_ref INTO STRICT recibo
 FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
 WHERE r.solicitud_ref=solicitud AND r.version=3;
 detalle:=vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
  false,'per_'||repeat('r',22),solicitud,'es');
 IF detalle->>'estado'<>'incorporada' OR detalle->>'version'<>'3'
 OR detalle->>'recibo_ref'<>recibo
 OR detalle->>'participacion_ref'<>'participacion:b97:fixture'
 OR detalle->>'bolsa_ref'<>'bolsa:b97:fixture'
 THEN RAISE EXCEPTION 'B97 fixture: GET detalle divergente %',detalle; END IF;
 lista:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
  false,'per_'||repeat('r',22),
  jsonb_build_object('estado','incorporada','convocatoria_ref','','limite',10,'cursor',''),'es');
 IF jsonb_array_length(lista->'solicitudes')<>1
 OR lista#>>'{solicitudes,0,participacion_ref}'<>'participacion:b97:fixture'
 OR lista#>>'{solicitudes,0,bolsa_ref}'<>'bolsa:b97:fixture'
 THEN RAISE EXCEPTION 'B97 fixture: GET lista divergente %',lista; END IF;
END $conteos$;
ROLLBACK;
