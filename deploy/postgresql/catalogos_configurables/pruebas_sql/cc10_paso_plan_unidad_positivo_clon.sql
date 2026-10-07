\set ON_ERROR_STOP on
-- CC10 (consulta R5 V2, opción A). SÓLO CLON DESECHABLE con CC7, CC8 y CC10,
-- como superusuario; todo en ROLLBACK. Reutiliza la siembra de
-- publicacion_plan_firma_cc8_positivo_clon.sql (stubs de AUT y AD167 sólo
-- dentro de la transacción) para publicar un plan y comprueba que
-- paso_plan_firma_con_unidad_v1: dice sí al paso exacto del plan publicado;
-- dice no con otra unidad, otra organización, otro documento, otro orden u
-- otro circuito, o con una entrada aún no vigente o ya vencida; rechaza un
-- paso mal formado; y no lo puede llamar el LOGIN del ejecutor CT. Publicar la
-- v2 del mismo catálogo deja la v1 en «publicado»: vale la v2; retirada la v2,
-- no hay plan vigente (como CC7). Además llama a las v3 de CT186 como el LOGIN
-- del ejecutor con la asignación real de una persona con unidad: sin plan con
-- su paso se paran en CC10; con él, CC10 pasa y se paran en la decisión.
-- el recorrido con decisiones atestadas reales (falta la extensión de K).
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';
-- Stub AUT (categoria Aplicacion aun no admite vec.catalogos.*): solo dentro del ROLLBACK.
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE sql AS $$ SELECT $5 LIKE 'vec.catalogos.%' $$;
-- Stub firma CT (AD167) para ejercitar solo leer_plan: devuelve el consumo con vigencia.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(p_consumo jsonb)
RETURNS jsonb LANGUAGE sql AS $$ SELECT p_consumo||jsonb_build_object('decision_valida_hasta',to_char(clock_timestamp()+interval '5 min','YYYY-MM-DD"T"HH24:MI:SS"Z"')) $$;
-- Una asignación real con organización y unidad: su paso se añade al plan.
SELECT set_config('t.org',e1->'valores'->>0,true), set_config('t.uni',e2->'valores'->>0,true),
 set_config('t.asig',jsonb_build_object('asignacion_ref',a.asignacion_ref,'asignacion_huella_sha256',a.huella_sha256,
  'principal_id',a.principal_id,'perfil_activo_ref',a.perfil_activo_ref,'version_rol_ref',a.version_rol_ref)::text,true)
 FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 CROSS JOIN LATERAL (SELECT e FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='organizacion_ref') o(e1)
 CROSS JOIN LATERAL (SELECT e FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='unidad_ref') u(e2)
 WHERE a.documento->>'estado'='activa' AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz
 ORDER BY a.asignacion_ref LIMIT 1;
SET LOCAL session_replication_role=replica;
DO $p$
DECLARE r record;
BEGIN
 FOR r IN SELECT c.conrelid::regclass AS t,c.conname FROM pg_constraint c
  WHERE c.conrelid IN ('vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass,
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.atestacion_decision_v3'::regclass) AND c.contype IN ('c','f') LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.t,r.conname);
 END LOOP;
END $p$;
CREATE TEMP TABLE res(paso text, valor jsonb);
GRANT ALL ON res TO PUBLIC;
-- Siembra consumo+auditoria+atestacion selladas con esta TX.
CREATE FUNCTION pg_temp.sembrar(p_ref text,p_decision jsonb,p_capacidad jsonb,p_efecto text,p_huella text)
RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE dc bytea:=convert_to(p_decision::text,'UTF8'); cc bytea; ahora timestamptz(6):=clock_timestamp();
 hd text; cap jsonb; hc text:=encode(sha256(convert_to('consumo:'||p_ref,'UTF8')),'hex');
BEGIN
 hd:=encode(sha256(dc),'hex');
 cap:=p_capacidad||jsonb_build_object('huella_decision_sha256',hd);
 cc:=convert_to(cap::text,'UTF8');
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,huella_decision_sha256,decision_canonica,
  motivo_canonico,contexto_actor_canonico,payload_vec_ad_3,sobre_cose_sign1,evidencia_verificacion,raiz_publica_spki,
  capacidad_canonica,huella_capacidad_sha256,efecto_ref,huella_efecto_sha256,registrada_en)
 VALUES(p_ref,hd,dc,'\x00','\x00','\x00','\x00','\x00','\x00',cc,encode(sha256(cc),'hex'),p_efecto,p_huella,ahora);
 INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3(decision_ref,huella_decision_sha256,nonce,efecto_ref,
  huella_efecto_sha256,consumo_huella_sha256,consumida_en,transaccion_origen)
 VALUES(p_ref,hd,'n-'||p_ref,p_efecto,p_huella,hc,ahora,pg_current_xact_id());
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,decision_ref,efecto_ref,
  huella_efecto_sha256,anterior_sha256,huella_sha256,registrada_en,tipo_registro,version_consumo,actor_ref,
  perfil_activo_ref,finalidad_ref,proceso,canal,transaccion_origen)
 VALUES('aud-'||p_ref,900000000+abs(hashtext(p_ref))%99999999,p_ref,p_efecto,p_huella,repeat('0',64),encode(sha256(convert_to('auditoria:'||p_ref,'UTF8')),'hex'),ahora,'consumo_confirmado_v4',4,
  p_decision->>'principal_id',p_decision->>'perfil_activo_ref',p_decision->>'finalidad','prueba','sql',pg_current_xact_id());
 RETURN jsonb_build_object('decision_ref',p_ref,'efecto_ref',p_efecto,'huella_efecto_sha256',p_huella,
  'consumo_huella_sha256',hc,'auditoria_ref','aud-'||p_ref,'consumida_en',ahora,'consumo_nuevo',true);
END $f$;
-- Construye material + consumo para una operacion.
CREATE FUNCTION pg_temp.preparar(p_ref text,p_op text,p_cat jsonb,p_rev_esp bigint,p_h_esp text,p_clave text,p_actor text)
RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE canon bytea:=convert_to(p_cat::text,'UTF8'); h text; cat text:=p_cat->>'id'; ver text:=p_cat->>'version';
 accion text; traza jsonb; evento jsonb; tb bytea; eb bytea; m jsonb; mb bytea; ctx text; d jsonb; rec jsonb;
BEGIN
 h:=encode(sha256(canon),'hex');
 accion:=CASE p_op WHEN 'crear' THEN 'vec.catalogos.borrador.creado' WHEN 'actualizar' THEN 'vec.catalogos.borrador.actualizado'
   WHEN 'publicar' THEN 'vec.catalogos.publicado' ELSE 'vec.catalogos.retirado' END;
 traza:=jsonb_build_object('actor_id',p_actor,'action',accion,'module_id','contratacion_temporal','subject_ref',cat||':'||ver,
   'after_hash',h,'result','correcto')||CASE WHEN p_h_esp IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('before_hash',p_h_esp) END;
 evento:=jsonb_build_object('actor_id',p_actor,'type',accion,'module_id','contratacion_temporal','subject_ref',cat||':'||ver,
   'payload',jsonb_build_object('huella_sha256',h));
 tb:=convert_to(traza::text,'UTF8'); eb:=convert_to(evento::text,'UTF8');
 m:=jsonb_build_object('esquema','vec.catalogos.plan-firma.gobierno.v1','operacion',p_op,'catalogo_id',cat,'version',(ver)::bigint,
  'revision_esperada',p_rev_esp,'huella_esperada',to_jsonb(p_h_esp),'clave_operacion',p_clave,
  'catalogo_canonico_base64',translate(encode(canon,'base64'),E'\n',''),'catalogo_sha256',h,
  'traza_canonica_base64',translate(encode(tb,'base64'),E'\n',''),'traza_sha256',encode(sha256(tb),'hex'),
  'evento_canonico_base64',translate(encode(eb,'base64'),E'\n',''),'evento_sha256',encode(sha256(eb),'hex'));
 mb:=convert_to(m::text,'UTF8');
 ctx:=encode(sha256(convert_to('{"ambitos":{},"atributos":{"estado":"'||(p_cat->>'estado')||'","material_sha256":"'||encode(sha256(mb),'hex')||'","revision":"'||(p_cat->>'revision')||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('decision_ref',p_ref,'concedida',true,'codigo','concedida',
  'accion','vec.catalogos.'||p_op,'modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable',
  'finalidad','gestionar_contratacion_temporal','recurso_ref',cat||':'||ver,'contexto_recurso_huella_sha256',ctx,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','administracion_privilegiada','cuenta_privilegiada',true),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'principal_id',p_actor,'perfil_activo_ref','perfil:gob',
  'version_rol_ref','rol:gob:1','asignacion_ref','asig:'||p_ref,
  'valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 rec:=pg_temp.sembrar(p_ref,d,jsonb_build_object('operacion','vec.catalogos.'||p_op,
  'audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','efecto_ref',cat||':'||ver,'huella_efecto_sha256',ctx),
  cat||':'||ver,ctx);
 RETURN jsonb_build_object('material',encode(mb,'base64'),'consumo',jsonb_build_object('consumo',rec,'actor_ref',p_actor,
  'perfil_ref','perfil:gob','accion','vec.catalogos.'||p_op,'finalidad','gestionar_contratacion_temporal','proceso','prueba','canal','sql'),'h',h);
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.sembrar(text,jsonb,jsonb,text,text), pg_temp.preparar(text,text,jsonb,bigint,text,text,text) TO PUBLIC;
DO $s$
DECLARE base jsonb; b2 jsonb; pub jsonb; ret jsonb; x jsonb;
BEGIN
 base:='{"id":"ct.plan.firma.sintetico","version":1,"revision":1,"modulo_id":"contratacion_temporal","nombre":"Plan","fuente_ref":"fuente:rrhh:sintetica","motivo_creacion":"Ejercicio","estado":"borrador","creado_por":"actor:creador:001","creado_en":"2026-10-03T10:00:00Z","ultima_modificacion_en":"0001-01-01T00:00:00Z","publicado_en":"0001-01-01T00:00:00Z","retirado_en":"0001-01-01T00:00:00Z","entradas":[{"clave":"paso_1","etiqueta":"Paso 1","orden":1,"vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"accion_competencial":"contratacion_temporal.documento.firma_vec.registrar","cargo_ref":"cargo:direccion","circuito_ref":"catalogo:circuito:sintetico","circuito_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","circuito_version":"1","documento":"informe_definitivo","esquema":"ct.plan-competencia-firma.v2","esquema_contexto":"vec.contexto.firma.ct.v1","finalidad":"gestionar_contratacion_temporal","mapeo_fuente_ref":"fuente:plan:ct","mapeo_version":"1","organizacion_ref":"organizacion:central","paso_orden":"1","paso_ref":"paso:direccion","perfil_esperado_ref":"perfil:firma:direccion","rol_id":"ct_direccion_rrhh","tipo_recurso":"documento_contratacion_temporal","unidad_ref":"unidad:rrhh"}}]}';
 -- Entradas del mismo paso en otras unidades: una aún no vigente, otra ya
 -- vencida y la de la asignación real con unidad.
 base:=jsonb_set(base,'{entradas}',(base->'entradas')||jsonb_build_array(
  jsonb_set(jsonb_set(base->'entradas'->0,'{clave}','"paso_futuro"'),'{atributos,unidad_ref}','"unidad:futura"')
   ||'{"orden":2,"etiqueta":"Paso futuro","vigente_desde":"2099-01-01T00:00:00Z"}'::jsonb,
  jsonb_set(jsonb_set(base->'entradas'->0,'{clave}','"paso_vencido"'),'{atributos,unidad_ref}','"unidad:vencida"')
   ||'{"orden":3,"etiqueta":"Paso vencido","vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":"2026-02-01T00:00:00Z"}'::jsonb,
  jsonb_set(jsonb_set(jsonb_set(base->'entradas'->0,'{clave}','"paso_asignacion"'),'{atributos,unidad_ref}',to_jsonb(current_setting('t.uni'))),
   '{atributos,organizacion_ref}',to_jsonb(current_setting('t.org')))||'{"orden":4,"etiqueta":"Paso asignación"}'::jsonb));
 b2:=base||'{"revision":2,"ultima_modificacion_por":"actor:editor:001","ultima_modificacion_en":"2026-10-03T10:30:00Z","motivo_modificacion":"Ajuste","nombre":"Plan v2"}'::jsonb;
 pub:=b2||'{"estado":"publicado","publicado_por":"actor:publicador:001","publicado_en":"2026-10-03T11:00:00Z","aprobacion_ref":"aprobacion:001","motivo_publicacion":"Revisado"}'::jsonb;
 ret:=pub||'{"estado":"retirado","retirado_por":"actor:retirador:001","retirado_en":"2026-10-03T12:00:00Z","retirada_aprobacion_ref":"aprobacion:ret:001","motivo_retirada":"Fin"}'::jsonb;
 INSERT INTO res VALUES('crear',pg_temp.preparar('dec:crear','crear',base,0,NULL,'clave-crear-000000001','actor:creador:001'));
 INSERT INTO res VALUES('crear_replay',pg_temp.preparar('dec:crear2','crear',base,0,NULL,'clave-crear-000000001','actor:creador:001'));
 INSERT INTO res VALUES('crear_otra',pg_temp.preparar('dec:crear3','crear',base,0,NULL,'clave-crear-000000002','actor:creador:001'));
 INSERT INTO res VALUES('actualizar',pg_temp.preparar('dec:act','actualizar',b2,1,encode(sha256(convert_to(base::text,'UTF8')),'hex'),'clave-actua-000000001','actor:editor:001'));
 INSERT INTO res VALUES('publicar_mismo',pg_temp.preparar('dec:pubm','publicar',pub||'{"publicado_por":"actor:editor:001"}'::jsonb,2,encode(sha256(convert_to(b2::text,'UTF8')),'hex'),'clave-publi-000000009','actor:editor:001'));
 INSERT INTO res VALUES('publicar',pg_temp.preparar('dec:pub','publicar',pub,2,encode(sha256(convert_to(b2::text,'UTF8')),'hex'),'clave-publi-000000001','actor:publicador:001'));
 INSERT INTO res VALUES('act_creado_en',pg_temp.preparar('dec:actce','actualizar',b2||'{"creado_en":"2026-10-03T09:00:00Z"}'::jsonb,1,encode(sha256(convert_to(base::text,'UTF8')),'hex'),'clave-actua-000000002','actor:editor:001'));
 INSERT INTO res VALUES('otro_crear',pg_temp.preparar('dec:ocr','crear',jsonb_set(base,'{id}','"ct.plan.firma.otro"')||'{"creado_por":"actor:creador:002"}'::jsonb,0,NULL,'clave-otro-0000000001','actor:creador:002'));
 INSERT INTO res VALUES('otro_publicar',pg_temp.preparar('dec:opu','publicar',jsonb_set(base,'{id}','"ct.plan.firma.otro"')||'{"creado_por":"actor:creador:002","estado":"publicado","publicado_por":"actor:publicador:002","publicado_en":"2026-10-03T11:00:00Z","aprobacion_ref":"aprobacion:002","motivo_publicacion":"Revisado"}'::jsonb,1,encode(sha256(convert_to((jsonb_set(base,'{id}','"ct.plan.firma.otro"')||'{"creado_por":"actor:creador:002"}'::jsonb)::text,'UTF8')),'hex'),'clave-otro-0000000002','actor:publicador:002'));
 INSERT INTO res VALUES('retirar',pg_temp.preparar('dec:ret','retirar',ret,2,encode(sha256(convert_to(pub::text,'UTF8')),'hex'),'clave-retir-000000001','actor:retirador:001'));
 -- v2 del mismo catálogo, publicada sin retirar la v1, y su retirada.
 x:=jsonb_set(base,'{version}','2')||'{"version_anterior_ref":"ct.plan.firma.sintetico:1","creado_por":"actor:creador:003","creado_en":"2026-10-03T13:00:00Z","nombre":"Plan nueva version"}'::jsonb;
 INSERT INTO res VALUES('crear_v2',pg_temp.preparar('dec:crv2','crear',x,0,NULL,'clave-crev2-000000001','actor:creador:003'));
 INSERT INTO res VALUES('publicar_v2',pg_temp.preparar('dec:pbv2','publicar',x||'{"estado":"publicado","publicado_por":"actor:publicador:004","publicado_en":"2026-10-03T14:00:00Z","aprobacion_ref":"aprobacion:004","motivo_publicacion":"Nueva"}'::jsonb,1,encode(sha256(convert_to(x::text,'UTF8')),'hex'),'clave-pbv2-0000000001','actor:publicador:004'));
 INSERT INTO res VALUES('retirar_v2',pg_temp.preparar('dec:rtv2','retirar',x||'{"estado":"retirado","publicado_por":"actor:publicador:004","publicado_en":"2026-10-03T14:00:00Z","aprobacion_ref":"aprobacion:004","motivo_publicacion":"Nueva","retirado_por":"actor:retirador:005","retirado_en":"2026-10-03T15:00:00Z","retirada_aprobacion_ref":"aprobacion:ret:005","motivo_retirada":"Fin"}'::jsonb,1,encode(sha256(convert_to((x||'{"estado":"publicado","publicado_por":"actor:publicador:004","publicado_en":"2026-10-03T14:00:00Z","aprobacion_ref":"aprobacion:004","motivo_publicacion":"Nueva"}'::jsonb)::text,'UTF8')),'hex'),'clave-rtv2-0000000001','actor:retirador:005'));
END $s$;
SET LOCAL session_replication_role=origin;
CREATE FUNCTION pg_temp.ejecutar(p text) RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE v jsonb; r record;
BEGIN
 SELECT valor INTO v FROM res WHERE paso=p;
 SELECT * INTO r FROM vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(decode(v->>'material','base64'),v->'consumo');
 RETURN to_jsonb(r);
EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('error',SQLSTATE,'msg',SQLERRM);
END $f$;
-- Llama a CC10 como el propietario CT (quien la usa en CT186).
CREATE FUNCTION pg_temp.paso(circ text DEFAULT repeat('a',64),doc text DEFAULT 'informe_definitivo',orden integer DEFAULT 1,
 org text DEFAULT 'organizacion:central',uni text DEFAULT 'unidad:rrhh') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE r boolean;
BEGIN
 SET LOCAL ROLE vec_contratacion_temporal_propietario;
 BEGIN r:=vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(circ,doc,orden,org,uni);
 EXCEPTION WHEN OTHERS THEN RESET ROLE; RETURN 'error:'||SQLSTATE;
 END;
 RESET ROLE;
 RETURN r::text;
END $f$;
CREATE FUNCTION pg_temp.ok(caso text,cond boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF cond IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;
CREATE ROLE prueba_cc10_ct LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_cc10_ct WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- Llama a una v3 de CT186 con la asignación real como decisión: devuelve el
-- mensaje con que se para.
CREATE FUNCTION pg_temp.v3(funcion text) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE sol text:=jsonb_build_object('OrganizacionRef',current_setting('t.org'),'ExpedienteRef','exp:cc10','VersionExpediente',1,
  'Documento','informe_definitivo','FirmantePrincipalCandidatoRef','per_cc10','ClaveIdempotencia','clave-cc10-00000001',
  'PasoOrden',1,'CatalogoHuella',repeat('a',64),'Via','certificado_vec','UnidadRef',current_setting('t.uni'))::text;
BEGIN
 EXECUTE format('SELECT vec_contratacion_temporal.%s(%L,%L,%L,NULL,NULL,1,1,NULL,NULL,NULL,NULL)',funcion,sol,'\x7b7d'::bytea,
  convert_to(current_setting('t.asig'),'UTF8'));
 RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLERRM;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.v3(text) TO prueba_cc10_ct;
SELECT pg_temp.ok('hay_asignacion_con_unidad',current_setting('t.uni',true) IS NOT NULL);
SELECT pg_temp.ok('sin_plan_publicado_no',pg_temp.paso()='false');
SET SESSION AUTHORIZATION prueba_cc10_ct;
SELECT pg_temp.ok('v3_sin_plan_paran_en_cc10',pg_temp.v3('consultar_firmas_r5_atestadas_v3')='lectura V2 unidad sin paso en el plan'
 AND pg_temp.v3('recuperar_firmas_r5_atestadas_v3')='lectura V2 unidad sin paso en el plan');
RESET SESSION AUTHORIZATION;
SELECT 'crear' paso, pg_temp.ejecutar('crear')->>'estado';
SELECT 'actualizar' paso, pg_temp.ejecutar('actualizar')->>'estado';
SELECT 'publicar' paso, pg_temp.ejecutar('publicar')->>'estado';
SELECT pg_temp.ok('paso_exacto_si',pg_temp.paso()='true');
SELECT pg_temp.ok('otra_unidad_no',pg_temp.paso(uni=>'unidad:otra')='false');
SELECT pg_temp.ok('otra_organizacion_no',pg_temp.paso(org=>'organizacion:otra')='false');
SELECT pg_temp.ok('otro_documento_no',pg_temp.paso(doc=>'resolucion')='false');
SELECT pg_temp.ok('otro_orden_no',pg_temp.paso(orden=>2)='false');
SELECT pg_temp.ok('otro_circuito_no',pg_temp.paso(circ=>repeat('b',64))='false');
SELECT pg_temp.ok('entrada_no_vigente_o_vencida_no',pg_temp.paso(uni=>'unidad:futura')='false' AND pg_temp.paso(uni=>'unidad:vencida')='false');
SET SESSION AUTHORIZATION prueba_cc10_ct;
SELECT pg_temp.ok('v3_con_plan_pasan_cc10',pg_temp.v3('consultar_firmas_r5_atestadas_v3')='lectura V2 divergente'
 AND pg_temp.v3('recuperar_firmas_r5_atestadas_v3')='lectura V2 divergente');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.ok('paso_mal_formado',pg_temp.paso(uni=>'Unidad con espacios')='error:22023' AND pg_temp.paso(circ=>'x')='error:22023');
SELECT pg_temp.ok('ejecutor_sin_execute',NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
 'vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)','EXECUTE'));
SELECT 'crear_v2' paso, pg_temp.ejecutar('crear_v2')->>'estado';
SELECT 'publicar_v2' paso, pg_temp.ejecutar('publicar_v2')->>'estado';
SELECT pg_temp.ok('v2_publicada_con_v1_publicada_si',pg_temp.paso()='true' AND (SELECT count(*) FROM vec_catalogos_configurables.plan_firma_control
 WHERE catalogo_id='ct.plan.firma.sintetico' AND estado='publicado')=2);
SELECT 'retirar_v2' paso, pg_temp.ejecutar('retirar_v2')->>'estado';
SELECT pg_temp.ok('v2_retirada_no',pg_temp.paso()='false' AND (SELECT estado FROM vec_catalogos_configurables.plan_firma_control
 WHERE catalogo_id='ct.plan.firma.sintetico' AND version=1)='publicado');
ROLLBACK;
