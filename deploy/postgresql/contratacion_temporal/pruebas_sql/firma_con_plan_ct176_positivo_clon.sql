\set ON_ERROR_STOP on
-- SÓLO CLON DESECHABLE, como superusuario; nunca en la principal ni en cidonia. Todo en ROLLBACK.
-- Dobles: núcleo AD consumir_decision_mutacion_v3_interna (siembra consumo,
-- auditoría v4 y atestación selladas con esta TX) y CT172
-- registrar_firma_verificada_v2 (consume la interior con ese doble y escribe
-- firma/revisión/custodia mínimas). Reales: CT176, AD177 (exterior y
-- recuperar), AD167, CC7 leer_plan y la guarda diferida.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';
DO $p$
DECLARE r record;
BEGIN
 FOR r IN SELECT c.conrelid::regclass AS t,c.conname FROM pg_constraint c
  WHERE c.conrelid IN ('vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass,
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.atestacion_decision_v3'::regclass,
   'vec_contratacion_temporal.firma_documento_v1'::regclass,
   'vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass,
   'vec_contratacion_temporal.firma_documento_custodia_v1'::regclass) AND c.contype IN ('c','f') LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.t,r.conname);
 END LOOP;
END $p$;
-- Se desactivan las guardas CT172 ajenas a CT176; la de CT176 queda activa.
ALTER TABLE vec_contratacion_temporal.firma_documento_v1 DISABLE TRIGGER cabeza_global_ai;
ALTER TABLE vec_contratacion_temporal.firma_documento_v1 DISABLE TRIGGER impedir_degradacion_pdf_bi;
ALTER TABLE vec_contratacion_temporal.firma_documento_v1 DISABLE TRIGGER hija_pdf_v2_ai;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
 p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz, consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c jsonb:=convert_from(p_capacidad_canonica,'UTF8')::jsonb; d jsonb:=convert_from(p_decision_canonica,'UTF8')::jsonb;
 ahora timestamptz(6):=clock_timestamp(); hd text:=encode(sha256(p_decision_canonica),'hex');
 ref text:=d->>'decision_ref'; hc text:=encode(sha256(convert_to('consumo:'||(d->>'decision_ref'),'UTF8')),'hex');
BEGIN
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,huella_decision_sha256,decision_canonica,
  motivo_canonico,contexto_actor_canonico,payload_vec_ad_3,sobre_cose_sign1,evidencia_verificacion,raiz_publica_spki,
  capacidad_canonica,huella_capacidad_sha256,efecto_ref,huella_efecto_sha256,registrada_en)
 VALUES(ref,hd,p_decision_canonica,'\x00','\x00','\x00','\x00','\x00','\x00',p_capacidad_canonica,encode(sha256(p_capacidad_canonica),'hex'),
  c->>'efecto_ref',c->>'huella_efecto_sha256',ahora);
 INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3(decision_ref,huella_decision_sha256,nonce,efecto_ref,
  huella_efecto_sha256,consumo_huella_sha256,consumida_en,transaccion_origen)
 VALUES(ref,hd,'n-'||ref,c->>'efecto_ref',c->>'huella_efecto_sha256',hc,ahora,pg_current_xact_id());
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,decision_ref,efecto_ref,
  huella_efecto_sha256,anterior_sha256,huella_sha256,registrada_en,tipo_registro,version_consumo,actor_ref,
  perfil_activo_ref,finalidad_ref,proceso,canal,transaccion_origen)
 VALUES('aud-'||ref,900000000+abs(hashtext(ref))%99999999,ref,c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('0',64),
  encode(sha256(convert_to('auditoria:'||ref,'UTF8')),'hex'),ahora,'consumo_confirmado_v4',4,
  d->>'principal_id',d->>'perfil_activo_ref',d->>'finalidad','prueba','sql',pg_current_xact_id());
 RETURN QUERY SELECT ref,c->>'efecto_ref',c->>'huella_efecto_sha256',hc,'aud-'||ref,ahora,true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO postgres;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO PUBLIC;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(
 p_solicitud text,p_comprobada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_descriptor_nominal bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s jsonb:=p_solicitud::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb; k record; previa record;
 firma text; recibo text; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
 h text:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex'); dh text:=encode(sha256(p_descriptor_nominal),'hex');
BEGIN
 SELECT * INTO STRICT k FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('firma_vec_documento_ct_v2',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_contratacion_temporal.firma_documento_v1 f WHERE f.clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
  RETURN jsonb_build_object('FirmaRef',previa.firma_ref,'ReciboRef',previa.recibo_ref,'ActorRef',previa.actor_ref,
   'PerfilRef',previa.perfil_ref,'RegistradaEn',previa.registrada_en,'SolicitudHuella',previa.solicitud_huella_sha256,
   'YaRegistrada',true,'DocumentoCustodiaRef',s->>'DocumentoCustodiaRef');
 END IF;
 firma:='firma-ct:'||gen_random_uuid()::text; recibo:='recibo-firma-ct:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.firma_documento_v1(firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,
  secuencia,clave_idempotencia,solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,
  politica_verificacion,actor_ref,perfil_ref,decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en,
  via_registro,unidad_firmante_ref)
 VALUES(firma,s->>'OrganizacionRef',s->>'ExpedienteRef',3,s->>'Documento',(SELECT coalesce(max(secuencia),0)+1 FROM vec_contratacion_temporal.firma_documento_v1),
  s->>'ClaveIdempotencia',h,s->>'CatalogoRef',s->>'CatalogoHuella',s->>'PasoRef',(s->>'PasoOrden')::integer,'firmado',
  s->>'PoliticaVerificacion',d->>'principal_id',d->>'perfil_activo_ref',k.decision_ref,k.consumo_huella_sha256,k.auditoria_ref,
  recibo,ahora,s->>'Via',s->>'UnidadFirmanteRef');
 INSERT INTO vec_contratacion_temporal.firma_documento_revision_pdf_v2 VALUES(firma,1,NULL,NULL,'doc-entrada',1,repeat('9',64),100,1,
  ARRAY[0,1,2,3]::bigint[],repeat('9',64),repeat('9',64),200,'{}','{}',repeat('9',64),clock_timestamp(),'rol','cuenta','vinculo',1,repeat('9',64),
  'evidencia:x',repeat('9',64),'esquema',k.decision_ref,k.consumo_huella_sha256,k.auditoria_ref,p_descriptor_nominal,dh);
 INSERT INTO vec_contratacion_temporal.firma_documento_custodia_v1 VALUES(firma,recibo,ahora,s->>'DocumentoCustodiaRef',1,repeat('9',64));
 RETURN jsonb_build_object('FirmaRef',firma,'ReciboRef',recibo,'ActorRef',d->>'principal_id','PerfilRef',d->>'perfil_activo_ref',
  'RegistradaEn',ahora,'SolicitudHuella',h,'YaRegistrada',false,'DocumentoCustodiaRef',s->>'DocumentoCustodiaRef');
END $f$;
ALTER FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) OWNER TO postgres;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) TO PUBLIC;

CREATE TEMP TABLE caso(nombre text PRIMARY KEY, valor text);
GRANT SELECT ON caso TO PUBLIC;

DO $sembrar$
DECLARE cat jsonb; canon bytea; h text; org text:='organizacion:central'; clave text:='prueba-ct176-pos-0001';
 solicitud text; descriptor text; recurso text; sh text; dh text; env text; ctx_i text; ctx_e text;
 di jsonb; ci jsonb; de jsonb; ce jsonb; base jsonb; dib bytea; deb bytea; n int;
BEGIN
 cat:='{"id":"ct.plan.firma.sintetico","version":1,"revision":1,"modulo_id":"contratacion_temporal","nombre":"Plan","fuente_ref":"fuente:rrhh:sintetica","motivo_creacion":"Ejercicio","estado":"publicado","creado_por":"actor:creador:001","creado_en":"2026-10-03T10:00:00Z","ultima_modificacion_en":"0001-01-01T00:00:00Z","publicado_por":"actor:publicador:001","publicado_en":"2026-10-03T11:00:00Z","aprobacion_ref":"aprobacion:001","motivo_publicacion":"Revisado","retirado_en":"0001-01-01T00:00:00Z","entradas":[{"clave":"paso_1","etiqueta":"Paso 1","orden":1,"vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"accion_competencial":"contratacion_temporal.documento.firma_vec.registrar","cargo_ref":"cargo:direccion","circuito_ref":"catalogo:circuito:sintetico","circuito_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","circuito_version":"1","documento":"informe_definitivo","esquema":"ct.plan-competencia-firma.v2","esquema_contexto":"vec.contexto.firma.ct.v1","finalidad":"gestionar_contratacion_temporal","mapeo_fuente_ref":"fuente:plan:ct","mapeo_version":"1","organizacion_ref":"organizacion:central","paso_orden":"1","paso_ref":"paso:direccion","perfil_esperado_ref":"perfil:firma:direccion","rol_id":"ct_direccion_rrhh","tipo_recurso":"documento_contratacion_temporal","unidad_ref":"unidad:rrhh"}}]}';
 canon:=convert_to(cat::text,'UTF8'); h:=encode(sha256(canon),'hex');
 PERFORM vec_catalogos_configurables.validar_plan_nominal_firma_v1(canon,h);
 INSERT INTO vec_catalogos_configurables.plan_firma_control(catalogo_id,version,revision,estado,canonico_actual,huella_actual,
  publicacion_sha256,publicacion_revision,creado_por,ultimo_editor,publicado_por)
 VALUES('ct.plan.firma.sintetico',1,1,'publicado',canon,h,h,1,'actor:creador:001','actor:creador:001','actor:publicador:001');
 INSERT INTO vec_catalogos_configurables.plan_firma_historia(catalogo_id,version,revision,operacion,estado,canonico_exacto,huella_sha256,actor_ref,decision_ref,recibo_ref)
 VALUES('ct.plan.firma.sintetico',1,1,'publicar','publicado',canon,h,'actor:publicador:001','dec:pub','recibo:'||gen_random_uuid());
 INSERT INTO vec_catalogos_configurables.plan_firma_publicacion(catalogo_id,version,revision,canonico_exacto,publicacion_sha256,publicado_por,publicada_en,aprobacion_ref)
 VALUES('ct.plan.firma.sintetico',1,1,canon,h,'actor:publicador:001','2026-10-03T11:00:00Z','aprobacion:001');
 FOR n IN 1..2 LOOP
  solicitud:='{"OrganizacionRef":"'||org||'","ExpedienteRef":"exp-sintetico-176","ClaveIdempotencia":"'||clave||'","Via":"certificado_vec","PoliticaVerificacion":"politica:vec:firma:verificacion-autonoma:v2","PerfilActivoOperadorRef":"perfil:operador","CatalogoRef":"catalogo:circuito:sintetico","CatalogoVersion":1,"CatalogoHuella":"'||repeat('a',64)||'","Documento":"informe_definitivo","PasoRef":"paso:direccion","PasoOrden":1,"UnidadFirmanteRef":"unidad:rrhh","DocumentoCustodiaRef":"doc-custodia-1"}';
  descriptor:='{"esquema":"ct.descriptor.sintetico","seleccion":{"perfil_esperado_ref":"perfil:firma:direccion","rol_id":"ct_direccion_rrhh","cargo_ref":"cargo:direccion"},"recurso":{"tipo_recurso":"documento_contratacion_temporal"},"accion":"contratacion_temporal.documento.firma_vec.registrar","finalidad":"gestionar_contratacion_temporal","motivo":"'||repeat('m',600)||'"}';
  recurso:='operacion-firma-vec-ct:'||clave;
  sh:=encode(sha256(convert_to(solicitud,'UTF8')),'hex'); dh:=encode(sha256(convert_to(descriptor,'UTF8')),'hex');
  ctx_i:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"},"atributos":{"descriptor_firma_sha256":"'||dh||'","material_sha256":"'||sh||'"}}','UTF8')),'hex');
  base:=jsonb_build_object('concedida',true,'codigo','concedida','accion','contratacion_temporal.documento.firma_vec.registrar',
   'modulo_id','contratacion_temporal','tipo_recurso','firma_vec_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
   'recurso_ref',recurso,'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),
   'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'principal_id','per_firmante','perfil_activo_ref','perfil:operador',
   'version_rol_ref','rol:firma:1','valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
  di:=base||jsonb_build_object('decision_ref','decision:ct176-int-'||n,'contexto_recurso_huella_sha256',ctx_i);
  dib:=convert_to(di::text,'UTF8');
  ci:=jsonb_build_object('operacion','contratacion_temporal.documento.firma_vec.registrar','audiencia_consumo','vec_contratacion_temporal.firma_vec.v2',
   'efecto_ref',recurso,'huella_efecto_sha256',ctx_i,'huella_decision_sha256',encode(sha256(dib),'hex'));
  env:='{"esquema":"ct.plan-autorizado-firma.v2","descriptor":'||descriptor||',"plan":{"catalogo_id":"ct.plan.firma.sintetico","catalogo_version":1,"catalogo_huella_sha256":"'||h||'","entrada_clave":"paso_1"},"decision_interior_sha256":"'||encode(sha256(dib),'hex')||'"}';
  ctx_e:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"},"atributos":{"material_sha256":"'||sh||'","plan_firma_sha256":"'||encode(sha256(convert_to(env,'UTF8')),'hex')||'"}}','UTF8')),'hex');
  de:=base||jsonb_build_object('decision_ref','decision:ct176-ext-'||n,'contexto_recurso_huella_sha256',ctx_e);
  deb:=convert_to(de::text,'UTF8');
  ce:=jsonb_build_object('operacion','contratacion_temporal.documento.firma_vec.registrar','audiencia_consumo','vec_contratacion_temporal.firma_vec.v2',
   'efecto_ref',recurso,'huella_efecto_sha256',ctx_e,'huella_decision_sha256',encode(sha256(deb),'hex'));
  INSERT INTO caso VALUES('solicitud'||n,solicitud),('env'||n,env),('ci'||n,ci::text),('di'||n,di::text),('ce'||n,ce::text),('de'||n,de::text);
 END LOOP;
END $sembrar$;

SET SESSION AUTHORIZATION vec_ct_o207_runtime;
DO $llamar$
DECLARE s text; e bytea; ci bytea; di bytea; ce bytea; de bytea; r jsonb;
BEGIN
 FOR n IN 1..2 LOOP
  SELECT valor INTO s FROM caso WHERE nombre='solicitud'||n;
  SELECT convert_to(valor,'UTF8') INTO e FROM caso WHERE nombre='env'||n;
  SELECT convert_to(valor,'UTF8') INTO ci FROM caso WHERE nombre='ci'||n;
  SELECT convert_to(valor,'UTF8') INTO di FROM caso WHERE nombre='di'||n;
  SELECT convert_to(valor,'UTF8') INTO ce FROM caso WHERE nombre='ce'||n;
  SELECT convert_to(valor,'UTF8') INTO de FROM caso WHERE nombre='de'||n;
  r:=vec_contratacion_temporal.registrar_firma_con_plan_v2(s,clock_timestamp(),e,
   ci,di,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00',
   ce,de,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE NOTICE 'llamada %: %',n,r;
 END LOOP;
END $llamar$;
RESET SESSION AUTHORIZATION;
SET CONSTRAINTS ALL IMMEDIATE;
SELECT 'hijas' paso, count(*), min(catalogo_id), min(entrada_clave), min(decision_interior_ref), min(decision_exterior_ref) FROM vec_contratacion_temporal.firma_documento_plan_v2;
SELECT 'firmas' paso, count(*) FROM vec_contratacion_temporal.firma_documento_v1;
SET CONSTRAINTS ALL DEFERRED;
-- Negativo de la guarda: CT172 directo, sin hija CT176 -> 23514 al comprobar.
DO $g$
DECLARE s text:=replace(replace((SELECT valor FROM caso WHERE nombre='solicitud1'),'prueba-ct176-pos-0001','prueba-ct176-sin-plan1'),'doc-custodia-1','doc-custodia-2');
 r jsonb; di jsonb:=(SELECT valor FROM caso WHERE nombre='di1')::jsonb||'{"decision_ref":"decision:ct176-directa"}';
BEGIN
 r:=vec_contratacion_temporal.registrar_firma_verificada_v2(s,clock_timestamp(),
  convert_to(((SELECT valor FROM caso WHERE nombre='ci1')::jsonb||jsonb_build_object('huella_decision_sha256',encode(sha256(convert_to(di::text,'UTF8')),'hex')))::text,'UTF8'),
  convert_to(di::text,'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','\x00');
 SET CONSTRAINTS ALL IMMEDIATE;
 RAISE NOTICE 'guarda: firma V2 sin plan ACEPTADA (mal)';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'guarda: % %',SQLSTATE,SQLERRM;
END $g$;
ROLLBACK;
