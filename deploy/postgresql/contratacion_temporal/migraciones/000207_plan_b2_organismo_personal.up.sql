-- CT207: la organización de CT y el organismo propietario de Personal
-- son identificadores distintos. La prueba de origen recibe el organismo
-- guardado en el material del plan CT, cuya fila y huella ya se cotejaron.
-- Definición literal de pg_get_functiondef del clon causal posterior a CT206.
-- No modificar SQL histórico ni reaplicar esta UP sobre historia.
BEGIN;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '15s';
DO $ct207_pre$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_contratacion_temporal.operacion_plan_personal_ct155(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'); p record;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT207: PARO clave=funcion actual=ausente esperado=CT155_postCT206' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex') huella,
        x.proowner::regrole::text propietario,x.prosecdef definidor,x.provolatile volatilidad,x.proconfig configuracion,x.proacl::text acl
 INTO p FROM pg_catalog.pg_proc x WHERE x.oid=f;
 IF p.huella IS DISTINCT FROM '60f934d163e8809a52c7dadfe6af6c0076172182aa76a57394b707431721082a' THEN
   RAISE EXCEPTION 'CT207: PARO clave=preimagen actual=% esperado=60f934d163e8809a52c7dadfe6af6c0076172182aa76a57394b707431721082a',p.huella USING ERRCODE='55000';
 END IF;
 IF p.propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario' OR p.definidor IS DISTINCT FROM false
     OR p.volatilidad IS DISTINCT FROM 'v' OR p.configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[]
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}' THEN
   RAISE EXCEPTION 'CT207: PARO clave=metadatos actual=%,%,%,%,% esperado=propietario,invoker,volatile,config_CT155,acl_unica',
     p.propietario,p.definidor,p.volatilidad,p.configuracion,p.acl USING ERRCODE='55000';
 END IF;
END $ct207_pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.operacion_plan_personal_ct155(op text, p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE m jsonb;s jsonb;p jsonb;h jsonb;a jsonb; ct record; d jsonb; previa record; plan record; version_actual numeric;
 org text;exp text;unidad text;actual_unidad text;accion text;sha text;ahora timestamptz;recibo text;eventoref text;planref text;intencion text;intrec text;solpersonal text;idem uuid;contrato jsonb;salida jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.sesion_plan_personal_ct155();
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536 THEN RAISE EXCEPTION 'CT155: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT155: material inválido' USING ERRCODE='22023'; END;
 IF p_material IS DISTINCT FROM vec_contratacion_temporal.canon_plan_personal_ct155(m) THEN RAISE EXCEPTION 'CT155: material no canónico' USING ERRCODE='22023'; END IF;
 s:=CASE WHEN op='registrar' THEN m->'solicitud' ELSE m END;org:=s->>'organizacion_ref';exp:=s->>'expediente_ref';
 unidad:=CASE op WHEN 'registrar' THEN m#>>'{material,unidad_ct_ref}' WHEN 'confirmar' THEN m->>'unidad_ct_ref' ELSE m->>'unidad_ref' END;
 accion:=CASE op WHEN 'registrar' THEN 'contratacion_temporal.incorporacion_personal.plan.registrar' WHEN 'confirmar' THEN 'contratacion_temporal.incorporacion_personal.origen.confirmar' WHEN 'leer' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' WHEN 'origen' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' WHEN 'antecedentes' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' END;
 IF accion IS NULL OR d->>'accion' IS DISTINCT FROM accion THEN RAISE EXCEPTION 'CT155: operación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT ct FROM vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT vv.agregado_json#>>'{asignacion,unidad_ref}' INTO actual_unidad FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_version_integral vv ON vv.expediente_ref=aa.expediente_ref AND vv.version=aa.version JOIN vec_contratacion_temporal.expediente_alta ea ON ea.expediente_ref=aa.expediente_ref WHERE aa.expediente_ref=exp AND ea.organizacion_ref=org;
 IF actual_unidad IS NULL OR unidad IS DISTINCT FROM actual_unidad THEN RAISE EXCEPTION 'CT155: unidad del expediente divergente' USING ERRCODE='42501';END IF;
 -- El permiso nuevo y su auditoría se consumen también para recuperar.
 PERFORM set_config('vec.ct115.organizacion_ref',org,true);PERFORM set_config('vec.ct115.expediente_ref',exp,true);
 IF op IN ('leer','origen','antecedentes') THEN
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','unidad_ref']) IS NOT TRUE THEN RAISE EXCEPTION 'CT155: selector inválido' USING ERRCODE='22023'; END IF;
 IF op='antecedentes' THEN RETURN vec_contratacion_temporal.antecedentes_plan_personal_ct155(org,exp); END IF;
 IF op='leer' THEN SELECT contrato_json INTO salida FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 ELSE SELECT recibo_json INTO salida FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;END IF;
 RETURN salida;END IF;
 -- Un bloqueo por expediente serializa plan y origen; las restricciones UUID
 -- impiden reutilizar la misma intención para otro expediente de organización.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec.ct155:'||org||':'||exp,0));
 SELECT aa.version INTO version_actual FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_alta ea ON ea.expediente_ref=aa.expediente_ref WHERE ea.organizacion_ref=org AND aa.expediente_ref=exp FOR UPDATE OF aa;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT155: expediente ausente' USING ERRCODE='55000'; END IF;
 sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');ahora:=date_trunc('microseconds',clock_timestamp());
 IF op='registrar' THEN
 p:=m->'material';
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['solicitud','material']) IS NOT TRUE OR s->>'clave_idempotencia'!~'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' OR s->>'clave_idempotencia'='00000000-0000-0000-0000-000000000000'
 THEN RAISE EXCEPTION 'CT155: intención inválida' USING ERRCODE='22023'; END IF;
 PERFORM vec_contratacion_temporal.validar_plan_personal_ct155(p,s);
 SELECT * INTO previa FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND clave_idempotencia=(s->>'clave_idempotencia')::uuid;
 IF FOUND THEN IF previa.material_sha256 IS DISTINCT FROM sha OR previa.expediente_ref IS DISTINCT FROM exp THEN RAISE EXCEPTION 'CT155: idempotencia reutilizada' USING ERRCODE='55000'; END IF;RETURN previa.contrato_json; END IF;
 IF version_actual IS DISTINCT FROM (p->>'version_expediente')::numeric OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 THEN RAISE EXCEPTION 'CT155: versión u origen en conflicto' USING ERRCODE='55000'; END IF;
 a:=vec_contratacion_temporal.antecedentes_plan_personal_ct155(org,exp);
 IF a IS NULL OR (SELECT jsonb_object_agg(k,p->k) FROM unnest(ARRAY['version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','categoria_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref']) k)
 IS DISTINCT FROM a-ARRAY['organizacion_ref','expediente_ref','bolsa','vinculo','documento_ref','documento_sha256','documento_version']
 OR (p->'bolsa')-'apertura_registro_sha256' IS DISTINCT FROM (a->'bolsa')-'apertura_registro_sha256'
 OR (a#>>'{bolsa,apertura_registro_sha256}'<>'' AND p#>>'{bolsa,apertura_registro_sha256}' IS DISTINCT FROM a#>>'{bolsa,apertura_registro_sha256}')
 OR p->>'documento_ref' IS DISTINCT FROM a->>'documento_ref' OR p->>'documento_sha256' IS DISTINCT FROM a->>'documento_sha256'
 OR p->>'vinculo_revision' IS DISTINCT FROM a#>>'{vinculo,revision}' OR p->>'vinculo_recibo_ref' IS DISTINCT FROM a#>>'{vinculo,recibo_ref}'
 OR p->>'catalogo_rpt_id' IS DISTINCT FROM a#>>'{vinculo,catalogo_id}' OR p->>'catalogo_rpt_modulo' IS DISTINCT FROM a#>>'{vinculo,modulo_id}' OR p->>'catalogo_rpt_version' IS DISTINCT FROM a#>>'{vinculo,catalogo_version}' OR p->>'catalogo_rpt_sha256' IS DISTINCT FROM a#>>'{vinculo,catalogo_huella_sha256}'
 THEN RAISE EXCEPTION 'CT155: antecedentes divergentes' USING ERRCODE='55000'; END IF;
 planref:='plan_ct_b2_'||replace(gen_random_uuid()::text,'-','');recibo:='recibo_ct_b2_'||replace(gen_random_uuid()::text,'-','');intencion:='intencion_ct_b2_'||replace(gen_random_uuid()::text,'-','');intrec:='recibo_intencion_ct_b2_'||replace(gen_random_uuid()::text,'-','');solpersonal:='solicitud_ct_b2_'||replace(gen_random_uuid()::text,'-','');idem:=gen_random_uuid();eventoref:='evento:plan-personal-b2:'||gen_random_uuid()::text;
 contrato:=jsonb_build_object('protocolo','personal_b2_v1','plan_ref',planref,'plan_version',1,'plan_recibo_ref',recibo,'plan_sha256',sha,'intencion_ref',intencion,'intencion_recibo_ref',intrec,'intencion_version',1,'solicitud_personal_ref',solpersonal,'idempotencia_personal_uuid',idem::text,'material',p,'solicitud',s,'registrado_en',to_char(ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(org,exp,version_actual,planref,eventoref,'ct.plan-incorporacion-personal.v1',recibo,ahora);
 INSERT INTO vec_contratacion_temporal.plan_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,version_expediente,clave_idempotencia,plan_ref,recibo_ref,intencion_ref,intencion_recibo_ref,solicitud_personal_ref,idempotencia_personal_uuid,material,material_json,material_sha256,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,contrato_json)
 VALUES(org,exp,version_actual,(s->>'clave_idempotencia')::uuid,planref,recibo,intencion,intrec,solpersonal,idem,p_material,m,sha,d->>'principal_id',d->>'perfil_activo_ref',ct.decision_ref,ct.consumo_huella_sha256,ct.auditoria_ref,eventoref,ahora,contrato);
 RETURN contrato;
 END IF;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','plan_ref','plan_version','plan_sha256','hechos']) IS NOT TRUE THEN RAISE EXCEPTION 'CT155: confirmación inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO plan FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 IF NOT FOUND OR plan.plan_ref IS DISTINCT FROM m->>'plan_ref' OR m->>'plan_version'<>'1' OR plan.material_sha256 IS DISTINCT FROM m->>'plan_sha256' THEN RAISE EXCEPTION 'CT155: plan divergente' USING ERRCODE='55000'; END IF;
 h:=vec_personal.probar_origen_incorporacion_plan_v1(plan.material_json#>>'{material,organismo_ref}',exp,plan.plan_ref,1,plan.material_sha256,m->'hechos');
 IF h IS NULL OR h IS DISTINCT FROM m->'hechos' THEN RAISE EXCEPTION 'CT155: Personal no acredita el origen' USING ERRCODE='55000'; END IF;
 SELECT * INTO previa FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 IF FOUND THEN IF previa.material_sha256 IS DISTINCT FROM sha THEN RAISE EXCEPTION 'CT155: origen reutilizado' USING ERRCODE='55000'; END IF;RETURN previa.recibo_json;END IF;
 IF version_actual IS DISTINCT FROM plan.version_expediente OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 THEN RAISE EXCEPTION 'CT155: versión u origen en conflicto' USING ERRCODE='55000'; END IF;
 recibo:='recibo:incorporacion-personal-b2:'||gen_random_uuid()::text;eventoref:='evento:incorporacion-personal-b2:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('protocolo','personal_b2_v1','confirmacion',m,'recibo_ref',recibo,'auditoria_ref',ct.auditoria_ref,'outbox_ref',eventoref,'registrado_en',to_char(ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'firma_oficial',false,'eficacia_administrativa',false);
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(org,exp,version_actual,plan.plan_ref,eventoref,'ct.incorporacion-personal.v1',recibo,ahora);
 INSERT INTO vec_contratacion_temporal.origen_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,plan_ref,recibo_ref,material,material_json,material_sha256,relacion_ref,ocupacion_ref,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,recibo_json)
 VALUES(org,exp,plan.plan_ref,recibo,p_material,m,sha,h->>'relacion_ref',h->>'ocupacion_ref',d->>'principal_id',d->>'perfil_activo_ref',ct.decision_ref,ct.consumo_huella_sha256,ct.auditoria_ref,eventoref,ahora,salida);
 RETURN salida;
END $function$
;
DO $ct207_post$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_contratacion_temporal.operacion_plan_personal_ct155(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'); p record;
BEGIN
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex') huella,
        x.proowner::regrole::text propietario,x.prosecdef definidor,x.provolatile volatilidad,x.proconfig configuracion,x.proacl::text acl
 INTO p FROM pg_catalog.pg_proc x WHERE x.oid=f;
 IF p.huella IS DISTINCT FROM '42a55dcc6efe7c939b1f7d798b9f1ae78f96f489697c0f76a0c28d49e06d86f9' THEN
   RAISE EXCEPTION 'CT207: PARO clave=postimagen actual=% esperado=42a55dcc6efe7c939b1f7d798b9f1ae78f96f489697c0f76a0c28d49e06d86f9',p.huella USING ERRCODE='55000';
 END IF;
 IF p.propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario' OR p.definidor IS DISTINCT FROM false
     OR p.volatilidad IS DISTINCT FROM 'v' OR p.configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[]
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}' THEN
   RAISE EXCEPTION 'CT207: PARO clave=metadatos_post actual=%,%,%,%,% esperado=metadatos_CT155',
     p.propietario,p.definidor,p.volatilidad,p.configuracion,p.acl USING ERRCODE='55000';
 END IF;
END $ct207_post$;
COMMIT;
