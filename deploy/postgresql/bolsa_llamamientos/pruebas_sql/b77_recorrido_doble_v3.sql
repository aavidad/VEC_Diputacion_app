\set ON_ERROR_STOP on
-- Recorrido funcional en clon: B77 y B76 son las funciones reales.
-- SOLO las dos fachadas del consumidor V3 son DOBLES transaccionales.
-- No acredita COSE/PDP reales; todo se revierte al final.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(
 p_operacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'operacion' IS DISTINCT FROM p_operacion OR d->>'accion' IS DISTINCT FROM p_operacion
    OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref' THEN
  RAISE EXCEPTION 'doble V3 documental denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble:b77:'||pg_catalog.encode(pg_catalog.sha256(p_decision||
   pg_catalog.convert_to(pg_catalog.clock_timestamp()::text,'UTF8')),'hex'),
   c->>'efecto_ref',c->>'huella_efecto_sha256',pg_catalog.repeat('0',64),
   'auditoria:doble:b77:'||pg_catalog.encode(pg_catalog.sha256(p_capacidad||
     pg_catalog.convert_to(pg_catalog.clock_timestamp()::text,'UTF8')),'hex'),
   pg_catalog.clock_timestamp(),true;
END $f$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF p_capacidad IS DISTINCT FROM '\x00'::bytea THEN
  RAISE EXCEPTION 'doble V3 B76 denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble:b76',pg_catalog.convert_from(p_payload,'UTF8'),
   pg_catalog.convert_from(p_decision,'UTF8')::jsonb->>'contexto_recurso_huella_sha256',
   pg_catalog.repeat('0',64),'auditoria:doble:b76',pg_catalog.clock_timestamp(),true;
END $f$;

DO $prueba$
DECLARE
 b text; p text; cand text; t timestamptz; acto timestamptz; version_politica bigint;
 politica text[]; fecha date; fecha_solicitud date; doc text:='documento:sintetico:rrhh17'; doc_sha text:=pg_catalog.repeat('a',64);
 clave text; sol_hash text; sol text; recibo_sol text; contenido text; cap bytea; dec bytea; contexto bytea;
 r record; primera timestamptz; original_estado bigint; original_op bigint; original_res bigint;
 motivo text:='Documento sintético validado'; actor text:='persona:rrhh-b77'; validador text:='persona:validadora-b77';
 clave_b8 text:='b77:regularizar:sintetico'; recibo_b8 text; recurso bytea; huella_comando text;
 decision_b76 bytea; recibo_resolucion text; desde_original timestamptz;
 recurso_ajeno bytea; decision_ajena bytea; huella_ajena text;
 renuncia_desde timestamptz; excluido_desde timestamptz; sanciones_antes bigint;
BEGIN
 SELECT c.bolsa_ref,e.participacion_ref INTO STRICT b,p
 FROM vec_bolsa_llamamientos.constitucion_entrada e
 JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
 WHERE (SELECT pg_catalog.max(s.desde) FROM vec_bolsa_llamamientos.situacion_participacion s
        WHERE s.participacion_ref=e.participacion_ref) < pg_catalog.clock_timestamp()-interval '5 minutes'
 LIMIT 1;
 SELECT v.candidato_ref INTO cand FROM vec_bolsa_llamamientos.vinculo_candidato v WHERE v.participacion_ref=p LIMIT 1;
 IF cand IS NULL THEN
  cand:='can_'||pg_catalog.repeat('R',22);
  INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(
    participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
  SELECT e.participacion_ref,cand,c.acta_ref,e.instantanea_ref,e.version_instantanea,
    pg_catalog.clock_timestamp()-interval '1 day'
  FROM vec_bolsa_llamamientos.constitucion_entrada e
  JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
  WHERE e.participacion_ref=p LIMIT 1;
 END IF;
 fecha:=(pg_catalog.clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date-1;
 t:=pg_catalog.date_trunc('second',pg_catalog.clock_timestamp())-interval '2 minutes';
 SELECT x.version,x.transiciones INTO STRICT version_politica,politica
 FROM vec_bolsa_llamamientos.politica_transiciones_situacion x ORDER BY x.version DESC LIMIT 1;
 PERFORM * FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(
  'catalogo:b77:ensayo',pg_catalog.repeat('c',64),
  politica||ARRAY['renuncia>en_revision','no_disponible>en_revision',
   'en_revision>disponible','en_revision>excluido','excluido>disponible']);
 SELECT pg_catalog.max(x.version) INTO version_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion x;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
   participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES(p,'renuncia',t-interval '1 second','Fixture sintética RRHH17',actor,t-interval '1 second','b77:renuncia','recibo:b77:renuncia',version_politica),
       (p,'en_revision',t,'Fixture sintética RRHH17',actor,t,'b77:revision','recibo:b77:revision',version_politica);
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
   participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,
   actor,validador,validada_en,registrada_en,clave_idempotencia,situacion_esperada_desde)
 VALUES(p,t,'revisar','resolucion','resolucion:sintetica:b77',pg_catalog.repeat('d',64),
   actor,validador,t,t,'b77:revision',t-interval '1 second');
 SELECT pg_catalog.count(*) INTO original_estado FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p;
 SELECT pg_catalog.count(*) INTO original_op FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p;
 SELECT pg_catalog.count(*) INTO original_res FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh;

 -- Solicitud real B77 y replay con material V3 nuevo. El doble V3 solo
 -- sustituye la comprobación criptográfica; el cotejo candidato/estado es real.
 contexto:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('vinculos',
  pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('tipo','candidato','estado','activo','referencia',cand)))::text,'UTF8');
 FOR i IN 1..2 LOOP
  fecha_solicitud:=CASE WHEN i=1 THEN fecha ELSE NULL END;
  clave:='clave:b77:documental:'||i::text;
  sol_hash:=vec_bolsa_llamamientos.b77_huella_partes_v1('solicitud-documental',cand,b,clave);
  sol:='solicitud-documental:'||sol_hash;
  recibo_sol:='recibo:solicitud-documental:'||vec_bolsa_llamamientos.b77_huella_partes_v1('recibo',sol_hash);
  contenido:=vec_bolsa_llamamientos.b77_huella_partes_v1('contenido-solicitud-documental',
    cand,b,doc,doc_sha,coalesce(pg_catalog.to_char(fecha_solicitud,'YYYY-MM-DD'),''));
  cap:=pg_catalog.convert_to(pg_catalog.jsonb_build_object(
    'operacion','bolsa.participaciones_propias.presentar_solicitud_documental',
    'audiencia_consumo','vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1',
    'efecto_ref','mi-bolsa:'||cand,'huella_efecto_sha256',pg_catalog.repeat('e',64))::text,'UTF8');
  dec:=pg_catalog.convert_to(pg_catalog.jsonb_build_object(
    'principal_id',cand,'accion','bolsa.participaciones_propias.presentar_solicitud_documental',
    'recurso_ref','mi-bolsa:'||cand,'tipo_recurso','participaciones_candidato',
    'modulo_id','bolsa','finalidad','gestion_participaciones_propias',
    'contexto_recurso_huella_sha256',pg_catalog.repeat('e',64),
    'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8');
  SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
    sol,recibo_sol,contenido,cand,b,doc,doc_sha,fecha_solicitud,clave,pg_catalog.clock_timestamp(),
    cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
  IF r.reutilizada OR r.solicitud_ref IS DISTINCT FROM sol OR r.recibo_ref IS DISTINCT FROM recibo_sol
     OR r.contenido_sha256 IS DISTINCT FROM contenido OR r.version<>1 OR r.estado<>'pendiente_rrhh' THEN
   RAISE EXCEPTION 'B77 solicitud positiva inválida'; END IF;
  IF (SELECT s.fecha_fin_causa FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
      WHERE s.solicitud_ref=sol) IS DISTINCT FROM fecha_solicitud THEN
   RAISE EXCEPTION 'B77 inventó fecha de fin de causa'; END IF;
  primera:=r.registrada_en;
  SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
    sol,recibo_sol,contenido,cand,b,doc,doc_sha,fecha_solicitud,clave,pg_catalog.clock_timestamp()+interval '1 hour',
    cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
  IF NOT r.reutilizada OR r.registrada_en IS DISTINCT FROM primera OR r.recibo_ref IS DISTINCT FROM recibo_sol THEN
   RAISE EXCEPTION 'B77 replay solicitud alterado'; END IF;
  IF i=1 THEN
   BEGIN
    PERFORM * FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
     sol,recibo_sol,contenido,cand,b,'dni:prueba',doc_sha,fecha_solicitud,clave,
     pg_catalog.clock_timestamp(),cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
    RAISE EXCEPTION 'B77 aceptó etiqueta DNI';
   EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
   BEGIN
    PERFORM * FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
     sol,recibo_sol,contenido,cand,b,'NIE:prueba',doc_sha,fecha_solicitud,clave,
     pg_catalog.clock_timestamp(),cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
    RAISE EXCEPTION 'B77 aceptó etiqueta NIE';
   EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
   IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
       WHERE s.participacion_ref=p)<>1 THEN RAISE EXCEPTION 'B77 DNI/NIE alteró pendiente'; END IF;
  END IF;
  BEGIN
   PERFORM * FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
    sol,recibo_sol,vec_bolsa_llamamientos.b77_huella_partes_v1('contenido-solicitud-documental',
     cand,b,doc||':otro',doc_sha,coalesce(pg_catalog.to_char(fecha_solicitud,'YYYY-MM-DD'),'')),
    cand,b,doc||':otro',doc_sha,fecha_solicitud,clave,pg_catalog.clock_timestamp(),
    cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'B77 misma clave otro documento aceptado';
  EXCEPTION WHEN sqlstate 'VBP01' THEN NULL; END;
  IF i=1 THEN
   -- Rechazo expreso con su autorización nominal, sin nueva situación B2.
   cap:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('operacion','bolsa.solicitudes_documentales.resolver',
    'audiencia_consumo','vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1',
    'efecto_ref',sol,'huella_efecto_sha256',pg_catalog.repeat('f',64))::text,'UTF8');
   dec:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('principal_id',actor,
    'accion','bolsa.solicitudes_documentales.resolver','recurso_ref',sol,'tipo_recurso','solicitud_documental_bolsa',
    'modulo_id','bolsa','finalidad','gestion_situacion_participacion',
    'contexto_recurso_huella_sha256',pg_catalog.repeat('f',64),
    'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8');
   SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.rechazar_solicitud_documental_rrhh_v1(
    sol,1,contenido,'Documento sintético insuficiente',actor,'clave:b77:rechazo','recibo:b77:rechazo',
    pg_catalog.clock_timestamp(),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   IF r.reutilizada OR r.estado<>'rechazada' OR r.recibo_ref<>'recibo:b77:rechazo' THEN
    RAISE EXCEPTION 'B77 rechazo sin recibo'; END IF;
   SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.rechazar_solicitud_documental_rrhh_v1(
    sol,1,contenido,'Documento sintético insuficiente',actor,'clave:b77:rechazo','recibo:b77:rechazo',
    pg_catalog.clock_timestamp()+interval '1 hour',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   IF NOT r.reutilizada OR r.recibo_ref<>'recibo:b77:rechazo' THEN RAISE EXCEPTION 'B77 replay rechazo'; END IF;
   IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_estado THEN RAISE EXCEPTION 'B77 rechazo cambió B2'; END IF;
  END IF;
 END LOOP;

 -- Favorable real B76+B77 en una TX. El recurso tiene la preimagen de Go;
 -- decisión y COSE son dobles declarados y se sustituyen por V3 real en E2E.
 acto:=pg_catalog.clock_timestamp();
 clave_b8:='b77:regularizar:sintetico';
 recibo_b8:='recibo:situacion:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(p||pg_catalog.chr(31)||clave_b8,'UTF8')),'hex');
 huella_comando:=vec_bolsa_llamamientos.b77_huella_partes_v1(
  'regularizacion-documental-v1',sol,'1',contenido,b,p,doc,doc_sha,
  pg_catalog.to_char(fecha,'YYYY-MM-DD'),
  ((EXTRACT(EPOCH FROM t)*1000000)::bigint)::text,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(motivo,'UTF8')),'hex'),
  actor,validador,clave_b8,recibo_b8);
 recurso:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"'||huella_comando||'"}}','UTF8');
 decision_b76:=pg_catalog.convert_to(pg_catalog.jsonb_build_object(
  'principal_id',actor,'accion','bolsa.situacion_participacion.cambiar',
  'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
  'recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',pg_catalog.encode(pg_catalog.sha256(recurso),'hex'))::text,'UTF8');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
  sol,1,contenido,b,p,acto,t,motivo,actor,clave_b8,recibo_b8,acto,validador,acto,doc,doc_sha,fecha,
  '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
 IF r.reutilizada OR r.recibo_ref IS DISTINCT FROM recibo_b8 OR r.situacion<>'disponible'
    OR r.recibo_resolucion_ref IS NULL OR r.resuelta_en IS NULL THEN
  RAISE EXCEPTION 'B77 favorable no conservó ambos recibos'; END IF;
 recibo_resolucion:=r.recibo_resolucion_ref; desde_original:=r.desde;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
  sol,1,contenido,b,p,acto+interval '1 hour',t,motivo,actor,clave_b8,recibo_b8,
  acto+interval '1 hour',validador,acto+interval '1 hour',doc,doc_sha,fecha,
  '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
 IF NOT r.reutilizada OR r.recibo_ref IS DISTINCT FROM recibo_b8
    OR r.recibo_resolucion_ref IS DISTINCT FROM recibo_resolucion OR r.desde IS DISTINCT FROM desde_original THEN
  RAISE EXCEPTION 'B77 replay favorable alteró recibos o instante'; END IF;
 -- Mutaciones del material firmado no recuperan el recibo anterior.
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   sol,1,contenido,b,p,acto+interval '1 hour',t,motivo,actor,clave_b8,recibo_b8,
   acto+interval '1 hour',validador,acto+interval '1 hour',doc,pg_catalog.repeat('b',64),fecha,
   '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
  RAISE EXCEPTION 'B77 documento alterado aceptado';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   sol,1,contenido,b||':ajena',p,acto+interval '1 hour',t,motivo,actor,clave_b8,recibo_b8,
   acto+interval '1 hour',validador,acto+interval '1 hour',doc,doc_sha,fecha,
   '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
  RAISE EXCEPTION 'B77 bolsa ajena aceptada';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   sol,1,contenido,b,p,acto+interval '1 hour',t,motivo,'persona:ajena',clave_b8,recibo_b8,
   acto+interval '1 hour',validador,acto+interval '1 hour',doc,doc_sha,fecha,
   '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
  RAISE EXCEPTION 'B77 actor ajeno aceptado';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 -- CAS obsoleto firmado de forma coherente alcanza el replay B76, pero B77
 -- lo rechaza frente al acto original sin duplicar historia.
 huella_ajena:=vec_bolsa_llamamientos.b77_huella_partes_v1(
  'regularizacion-documental-v1',sol,'1',contenido,b,p,doc,doc_sha,
  pg_catalog.to_char(fecha,'YYYY-MM-DD'),
  ((EXTRACT(EPOCH FROM (t-interval '1 second'))*1000000)::bigint)::text,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(motivo,'UTF8')),'hex'),
  actor,validador,clave_b8,recibo_b8);
 recurso_ajeno:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"'||huella_ajena||'"}}','UTF8');
 decision_ajena:=pg_catalog.convert_to(pg_catalog.jsonb_build_object(
  'principal_id',actor,'accion','bolsa.situacion_participacion.cambiar',
  'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
  'recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',pg_catalog.encode(pg_catalog.sha256(recurso_ajeno),'hex'))::text,'UTF8');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   sol,1,contenido,b,p,acto+interval '1 hour',t-interval '1 second',motivo,actor,clave_b8,recibo_b8,
   acto+interval '1 hour',validador,acto+interval '1 hour',doc,doc_sha,fecha,
   '\x00',decision_ajena,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso_ajeno);
  RAISE EXCEPTION 'B77 CAS obsoleto aceptado';
 EXCEPTION WHEN sqlstate 'VBS01' THEN NULL; END;
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_estado+1
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_op+1
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh)
      IS DISTINCT FROM original_res+2 THEN
  RAISE EXCEPTION 'B77 duplicó historia o cerró de forma parcial'; END IF;
 -- Recepción genérica desde exclusión con sanción viva: un escrito no es una
 -- decisión y B73/B76 seguirán impidiendo una reincorporación favorable.
 renuncia_desde:=acto+interval '1 microsecond';
 excluido_desde:=acto+interval '2 microseconds';
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
  participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,
  politica_transiciones_version)
 VALUES(p,'renuncia',renuncia_desde,'Renuncia sintética RRHH17',actor,renuncia_desde,
  'b77:renuncia:sancionada','recibo:b77:renuncia:sancionada',version_politica),
       (p,'excluido',excluido_desde,'Exclusión sintética RRHH17',actor,excluido_desde,
  'b77:exclusion:sintetica','recibo:b77:exclusion',version_politica);
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
  participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,
  actor,validador,validada_en,registrada_en,clave_idempotencia,situacion_esperada_desde)
 VALUES(p,excluido_desde,'excluir','resolucion','resolucion:sintetica:exclusion',pg_catalog.repeat('d',64),
  actor,validador,excluido_desde,excluido_desde,'b77:exclusion:sintetica',renuncia_desde);
 SELECT pg_catalog.count(*) INTO sanciones_antes FROM vec_bolsa_llamamientos.sancion_participacion WHERE participacion_ref=p;
 INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
  sancion_ref,participacion_ref,bolsa_ref,consecuencia,consecuencia_etiqueta,efecto,causa,
  fecha_notificacion,resolucion_ref,resolucion_sha256,resuelta_por,regla_ref,regla_huella_sha256,
  suspension_hasta,recurso_vence,recurso_regla_ref,recurso_regla_huella_sha256,
  situacion_desde,recibo_ref,actor,registrada_en,clave_idempotencia)
 VALUES('sancion:'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('b77-exclusion-viva','UTF8')),'hex'),
  p,b,'baja','Baja sintética','excluir','Causa sintética',CURRENT_DATE,
  'resolucion:b77:exclusion',pg_catalog.repeat('e',64),validador,'regla:b77',pg_catalog.repeat('f',64),
  NULL,CURRENT_DATE+10,'regla:recurso:b77',pg_catalog.repeat('a',64),excluido_desde,
  'recibo:b77:sancion',actor,excluido_desde,'b77:sancion:sintetica');
 clave:='clave:b77:documental:excluido';
 sol_hash:=vec_bolsa_llamamientos.b77_huella_partes_v1('solicitud-documental',cand,b,clave);
 sol:='solicitud-documental:'||sol_hash;
 recibo_sol:='recibo:solicitud-documental:'||vec_bolsa_llamamientos.b77_huella_partes_v1('recibo',sol_hash);
 contenido:=vec_bolsa_llamamientos.b77_huella_partes_v1('contenido-solicitud-documental',
  cand,b,doc,doc_sha,'');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
  sol,recibo_sol,contenido,cand,b,doc,doc_sha,NULL,clave,pg_catalog.clock_timestamp(),
  cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.estado<>'pendiente_rrhh' THEN RAISE EXCEPTION 'B77 excluido no pudo presentar'; END IF;
 primera:=r.registrada_en;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.solicitar_documental_portal_v1(
  sol,recibo_sol,contenido,cand,b,doc,doc_sha,NULL,clave,pg_catalog.clock_timestamp()+interval '1 hour',
  cap,dec,'\x00',contexto,1,1,'\x00','\x00','\x00','\x00');
 IF NOT r.reutilizada OR r.registrada_en IS DISTINCT FROM primera THEN
  RAISE EXCEPTION 'B77 replay excluido alteró recibo'; END IF;
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_estado+3
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.sancion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM sanciones_antes+1 THEN
  RAISE EXCEPTION 'B77 presentar con sanción cambió situación o sanción'; END IF;
 IF (SELECT a.situacion FROM vec_bolsa_llamamientos.situacion_participacion a
     WHERE a.participacion_ref=p AND a.desde<excluido_desde ORDER BY a.desde DESC LIMIT 1) IS DISTINCT FROM 'renuncia'
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
      WHERE o.participacion_ref=p AND o.desde=excluido_desde AND o.operacion='excluir') THEN
  RAISE EXCEPTION 'B77 fixture sanción carece antecedente renuncia/B8'; END IF;
 -- CAS y material V3 son coherentes; la única causa de denegación restante
 -- es la sanción B73 viva. La excepción revierte el intento entero.
 clave_b8:='b77:regularizar:sancionada';
 recibo_b8:='recibo:situacion:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(p||pg_catalog.chr(31)||clave_b8,'UTF8')),'hex');
 huella_comando:=vec_bolsa_llamamientos.b77_huella_partes_v1(
  'regularizacion-documental-v1',sol,'1',contenido,b,p,doc,doc_sha,
  pg_catalog.to_char(fecha,'YYYY-MM-DD'),
  ((EXTRACT(EPOCH FROM excluido_desde)*1000000)::bigint)::text,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(motivo,'UTF8')),'hex'),
  actor,validador,clave_b8,recibo_b8);
 recurso:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"'||huella_comando||'"}}','UTF8');
 decision_b76:=pg_catalog.convert_to(pg_catalog.jsonb_build_object(
  'principal_id',actor,'accion','bolsa.situacion_participacion.cambiar',
  'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
  'recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',pg_catalog.encode(pg_catalog.sha256(recurso),'hex'))::text,'UTF8');
 BEGIN
  acto:=pg_catalog.clock_timestamp();
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   sol,1,contenido,b,p,acto,excluido_desde,motivo,actor,clave_b8,recibo_b8,
   acto,validador,acto,doc,doc_sha,fecha,
   '\x00',decision_b76,'\x00','\x00',1,1,pg_catalog.convert_to(p,'UTF8'),'\x00','\x00','\x00',recurso);
  RAISE EXCEPTION 'B77 sanción viva permitió regularizar';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_estado+3
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM original_op+2
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh)
      IS DISTINCT FROM original_res+2
    OR (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.sancion_participacion WHERE participacion_ref=p)
      IS DISTINCT FROM sanciones_antes+1 THEN
  RAISE EXCEPTION 'B77 sanción viva dejó efecto parcial'; END IF;
END $prueba$;
ROLLBACK;
