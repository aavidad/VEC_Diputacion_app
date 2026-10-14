\set ON_ERROR_STOP on
-- CT212: variante nativa del plan B2; conserva material y recibos legados.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000212',0));
DO $pre$
DECLARE f_val oid:=to_regprocedure('vec_contratacion_temporal.validar_plan_personal_ct155(jsonb,jsonb)');
 f_op oid:=to_regprocedure('vec_contratacion_temporal.operacion_plan_personal_ct155(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 f_ant oid:=to_regprocedure('vec_contratacion_temporal.antecedentes_plan_personal_ct155(text,text)');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
 OR f_val IS NULL OR f_op IS NULL OR f_ant IS NULL
 OR to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)') IS NULL
 OR NOT has_function_privilege(current_user,'vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)','EXECUTE')
 OR to_regprocedure('vec_contratacion_temporal.validar_plan_personal_ct212(jsonb,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_contratacion_temporal.antecedentes_plan_personal_ct212(text,text)') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef(f_val),'UTF8')),'hex')
  IS DISTINCT FROM '23b1bad87867f8a7c19ee0844d72ab8584a2b047998a10f1c586230358ec50c7'
 OR encode(sha256(convert_to(pg_get_functiondef(f_op),'UTF8')),'hex')
  IS DISTINCT FROM '42a55dcc6efe7c939b1f7d798b9f1ae78f96f489697c0f76a0c28d49e06d86f9'
 OR encode(sha256(convert_to(pg_get_functiondef(f_ant),'UTF8')),'hex')
  IS DISTINCT FROM '2613f213e0da4c31c6d9899d1b5460c60e570f47ef662df0c28e71d5b50460fc'
 THEN RAISE EXCEPTION 'CT212: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.antecedentes_plan_personal_ct212(org text,exp text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE a jsonb; p record; b jsonb; nativa jsonb;
BEGIN
 a:=vec_contratacion_temporal.antecedentes_plan_personal_ct155(org,exp);
 IF a IS NULL THEN RETURN NULL; END IF;
 -- La aceptación CT es resolucion_ref; la propuesta se identifica por su
 -- recibo. El selector nativo se toma del material original de esa propuesta.
 SELECT * INTO p FROM vec_contratacion_temporal.propuesta_formalizacion
  WHERE organizacion_ref=org AND expediente_ref=exp
    AND recibo_ref=a->>'propuesta_recibo_ref'
    AND resolucion_ref=a->>'aceptacion_ref';
 IF NOT FOUND THEN RETURN NULL; END IF;
 IF NOT (p.material_json ? 'AceptacionBolsaNativa') THEN RETURN a; END IF;
 b:=p.material_json->'AceptacionBolsaNativa';
 IF (p.material_json ? 'AceptacionBolsa') OR p.aceptacion_bolsa_json IS DISTINCT FROM b
    OR p.resolucion_ref IS DISTINCT FROM a->>'aceptacion_ref'
    OR p.recibo_ref IS DISTINCT FROM a->>'propuesta_recibo_ref'
    OR p.llamamiento_ref IS DISTINCT FROM b#>>'{Selector,llamamiento_ref}'
    OR b#>>'{Selector,organizacion_ref}' IS DISTINCT FROM org
    OR b#>>'{Selector,expediente_ref}' IS DISTINCT FROM exp
    OR b#>>'{Selector,categoria_ref}' IS DISTINCT FROM a->>'categoria_ref'
 THEN RAISE EXCEPTION 'CT212: propuesta nativa divergente' USING ERRCODE='55000'; END IF;
 nativa:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(b->'Selector');
 IF nativa->>'estado' NOT IN ('pendiente','acreditado')
    OR nativa#>>'{anclaje,respuesta_ref}' IS DISTINCT FROM b->>'RespuestaRef'
    OR (nativa#>>'{anclaje,respondida_en}')::timestamptz
       IS DISTINCT FROM (b->>'RespondidaEn')::timestamptz
 THEN RAISE EXCEPTION 'CT212: anclaje nativo no acreditado' USING ERRCODE='55000'; END IF;
 RETURN (a-'bolsa')||jsonb_build_object('bolsa_nativa',b->'Selector');
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.antecedentes_plan_personal_ct212(text,text) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.validar_plan_personal_ct212(p jsonb, s jsonb)
 RETURNS void
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog, pg_temp'
AS $function$
DECLARE k text; nodo jsonb; desde date; hasta date;
BEGIN
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','bolsa_nativa','persona_ref','persona_version','persona_fuente','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','fuente_organizacion','fuente_plantilla','fuente_rpt','version_plaza_ref','version_puesto_ref','revision_plaza','revision_puesto','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','catalogo_rpt_version','catalogo_rpt_sha256','categoria_ref','vinculo_revision','vinculo_recibo_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256','ejercicio_sintetico']) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT155: plan incompleto' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','analisis_recibo_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','persona_ref','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','version_plaza_ref','version_puesto_ref','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','categoria_ref','vinculo_recibo_ref','documento_ref'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'CT155: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['version_expediente','analisis_version','persona_version','revision_plaza','revision_puesto','catalogo_rpt_version','vinculo_revision'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'number' OR p->>k!~'^[1-9][0-9]{0,15}$' OR (p->>k)::numeric>9007199254740991 THEN RAISE EXCEPTION 'CT155: versión inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['analisis_sha256','catalogo_rpt_sha256','documento_sha256'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[0-9a-f]{64}$' OR p->>k=repeat('0',64) THEN RAISE EXCEPTION 'CT155: huella inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['persona_fuente'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version','sha256']) IS NOT TRUE
 OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,15}$'
 OR (nodo->>'version')::numeric>9007199254740991 OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64)
 THEN RAISE EXCEPTION 'CT155: fuente inválida' USING ERRCODE='22023'; END IF;
 END LOOP;

 nodo:=p->'fuente_organizacion';
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: fuente organización inválida' USING ERRCODE='22023';END IF;
 FOREACH k IN ARRAY ARRAY['fuente_plantilla','fuente_rpt'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','revision','fuente_ref','fuente_sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' OR nodo->>'revision'!~'^[1-9][0-9]{0,15}$' OR (nodo->>'revision')::numeric>9007199254740991 OR nodo->>'fuente_ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'fuente_sha256'!~'^[0-9a-f]{64}$' OR nodo->>'fuente_sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: instrumento inválido' USING ERRCODE='22023';END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['regimen','modalidad'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version']) IS NOT TRUE OR nodo->>'ref'!~'^[a-z][a-z0-9_:-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,9}$' OR (nodo->>'version')::numeric>2147483647
 THEN RAISE EXCEPTION 'CT155: catálogo inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'bolsa_nativa',
    ARRAY['organizacion_ref','expediente_ref','clave_ct','unidad_ref','categoria_ref',
      'necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref']) IS NOT TRUE
 OR (p ? 'bolsa')
 OR EXISTS(SELECT 1 FROM jsonb_each(p->'bolsa_nativa') e
    WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string'
      OR octet_length(e.value#>>'{}') NOT BETWEEN 3 AND 160
      OR e.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
 OR p#>>'{bolsa_nativa,organizacion_ref}' IS DISTINCT FROM p->>'organizacion_ref'
 OR p#>>'{bolsa_nativa,expediente_ref}' IS DISTINCT FROM p->>'expediente_ref'
 OR p#>>'{bolsa_nativa,unidad_ref}' IS DISTINCT FROM p->>'unidad_ct_ref'
 OR p#>>'{bolsa_nativa,categoria_ref}' IS DISTINCT FROM p->>'categoria_ref'
 OR p#>>'{bolsa_nativa,recibo_respuesta_ref}' IS DISTINCT FROM p->>'persona_recibo_bolsa_ref'
 OR jsonb_typeof(p->'ejercicio_sintetico') IS DISTINCT FROM 'boolean'
 OR p->>'motivo_clave'!~'^[a-z][a-z0-9_.-]{1,79}$'
 OR jsonb_typeof(p->'clase_ocupacion') IS DISTINCT FROM 'string'
 OR p->>'clase_ocupacion' !~ '^[a-z][a-z0-9_]{0,63}$'
 OR p->>'desde'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR (p->>'hasta'<>'' AND p->>'hasta'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
 THEN RAISE EXCEPTION 'CT155: intención inválida' USING ERRCODE='22023'; END IF;
 BEGIN desde:=(p->>'desde')::date;hasta:=nullif(p->>'hasta','')::date;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT155: fecha inválida' USING ERRCODE='22023'; END;
 IF NOT isfinite(desde) OR to_char(desde,'YYYY-MM-DD')<>p->>'desde' OR (hasta IS NOT NULL AND (NOT isfinite(hasta) OR hasta<=desde OR to_char(hasta,'YYYY-MM-DD')<>p->>'hasta'))
 OR (p->>'analisis_version')::numeric>(p->>'version_expediente')::numeric
 OR (SELECT jsonb_object_agg(x.clave,p->x.clave) FROM unnest(ARRAY['organizacion_ref','expediente_ref','version_expediente','puesto_ref','plaza_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256']) AS x(clave))
 IS DISTINCT FROM s-ARRAY['clave_idempotencia','version_plantilla_ref','version_rpt_ref']
 OR p->>'version_plaza_ref' IS DISTINCT FROM s->>'version_plantilla_ref' OR p->>'version_puesto_ref' IS DISTINCT FROM s->>'version_rpt_ref'
 OR p->>'version_plaza_ref' IS DISTINCT FROM ('plantilla:' || (p#>>'{fuente_plantilla,ref}'))
 OR p->>'version_puesto_ref' IS DISTINCT FROM ('rpt:' || (p#>>'{fuente_rpt,ref}'))
 THEN RAISE EXCEPTION 'CT155: selección divergente' USING ERRCODE='22023'; END IF;
END $function$;


CREATE OR REPLACE FUNCTION vec_contratacion_temporal.operacion_plan_personal_ct155(op text, p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE m jsonb;s jsonb;p jsonb;h jsonb;a jsonb;nativa jsonb; ct record; d jsonb; previa record; plan record; version_actual numeric;
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
 IF op='antecedentes' THEN RETURN vec_contratacion_temporal.antecedentes_plan_personal_ct212(org,exp); END IF;
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
 IF p ? 'bolsa_nativa' THEN
  PERFORM vec_contratacion_temporal.validar_plan_personal_ct212(p,s);
 ELSE
  PERFORM vec_contratacion_temporal.validar_plan_personal_ct155(p,s);
 END IF;
 SELECT * INTO previa FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND clave_idempotencia=(s->>'clave_idempotencia')::uuid;
 IF FOUND THEN IF previa.material_sha256 IS DISTINCT FROM sha OR previa.expediente_ref IS DISTINCT FROM exp THEN RAISE EXCEPTION 'CT155: idempotencia reutilizada' USING ERRCODE='55000'; END IF;RETURN previa.contrato_json; END IF;
 IF version_actual IS DISTINCT FROM (p->>'version_expediente')::numeric OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 THEN RAISE EXCEPTION 'CT155: versión u origen en conflicto' USING ERRCODE='55000'; END IF;
 a:=vec_contratacion_temporal.antecedentes_plan_personal_ct212(org,exp);
 IF a IS NULL OR (SELECT jsonb_object_agg(k,p->k) FROM unnest(ARRAY['version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','categoria_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref']) k)
 IS DISTINCT FROM a-ARRAY['organizacion_ref','expediente_ref','bolsa','bolsa_nativa','vinculo','documento_ref','documento_sha256','documento_version']
 OR ((p ? 'bolsa_nativa') AND
     (NOT (a ? 'bolsa_nativa') OR p->'bolsa_nativa' IS DISTINCT FROM a->'bolsa_nativa'))
 OR (NOT (p ? 'bolsa_nativa') AND (a ? 'bolsa_nativa'))
 OR (p->'bolsa')-'apertura_registro_sha256' IS DISTINCT FROM (a->'bolsa')-'apertura_registro_sha256'
 OR (a#>>'{bolsa,apertura_registro_sha256}'<>'' AND p#>>'{bolsa,apertura_registro_sha256}' IS DISTINCT FROM a#>>'{bolsa,apertura_registro_sha256}')
 OR p->>'documento_ref' IS DISTINCT FROM a->>'documento_ref' OR p->>'documento_sha256' IS DISTINCT FROM a->>'documento_sha256'
 OR p->>'vinculo_revision' IS DISTINCT FROM a#>>'{vinculo,revision}' OR p->>'vinculo_recibo_ref' IS DISTINCT FROM a#>>'{vinculo,recibo_ref}'
 OR p->>'catalogo_rpt_id' IS DISTINCT FROM a#>>'{vinculo,catalogo_id}' OR p->>'catalogo_rpt_modulo' IS DISTINCT FROM a#>>'{vinculo,modulo_id}' OR p->>'catalogo_rpt_version' IS DISTINCT FROM a#>>'{vinculo,catalogo_version}' OR p->>'catalogo_rpt_sha256' IS DISTINCT FROM a#>>'{vinculo,catalogo_huella_sha256}'
 THEN RAISE EXCEPTION 'CT155: antecedentes divergentes' USING ERRCODE='55000'; END IF;
 IF p ? 'bolsa_nativa' THEN
  nativa:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(p->'bolsa_nativa');
  IF nativa->>'estado' IS DISTINCT FROM 'acreditado'
    OR nativa#>>'{anclaje,recibo_respuesta_ref}' IS DISTINCT FROM p->>'persona_recibo_bolsa_ref'
    OR nativa#>>'{persona,ref}' IS DISTINCT FROM p->>'persona_ref'
    OR nativa#>'{persona,version}' IS DISTINCT FROM p->'persona_version'
    OR nativa#>>'{vinculo,procedencia_ref}' IS DISTINCT FROM p#>>'{persona_fuente,ref}'
    OR nativa#>'{vinculo,procedencia_version}' IS DISTINCT FROM p#>'{persona_fuente,version}'
    OR nativa#>>'{vinculo,procedencia_sha256}' IS DISTINCT FROM p#>>'{persona_fuente,sha256}' THEN
   RAISE EXCEPTION 'CT212: persona nativa no acreditada' USING ERRCODE='55000';
  END IF;
 END IF;
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
 IF plan.material_json#>'{material,bolsa_nativa}' IS NOT NULL THEN
  p:=plan.material_json->'material';
  nativa:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(p->'bolsa_nativa');
  IF nativa->>'estado' IS DISTINCT FROM 'acreditado'
    OR nativa#>>'{anclaje,recibo_respuesta_ref}' IS DISTINCT FROM p->>'persona_recibo_bolsa_ref'
    OR nativa#>>'{persona,ref}' IS DISTINCT FROM p->>'persona_ref'
    OR nativa#>'{persona,version}' IS DISTINCT FROM p->'persona_version'
    OR nativa#>>'{vinculo,procedencia_ref}' IS DISTINCT FROM p#>>'{persona_fuente,ref}'
    OR nativa#>'{vinculo,procedencia_version}' IS DISTINCT FROM p#>'{persona_fuente,version}'
    OR nativa#>>'{vinculo,procedencia_sha256}' IS DISTINCT FROM p#>>'{persona_fuente,sha256}' THEN
   RAISE EXCEPTION 'CT212: plan nativo divergente' USING ERRCODE='55000';
  END IF;
 END IF;
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


REVOKE ALL ON FUNCTION vec_contratacion_temporal.validar_plan_personal_ct212(jsonb,jsonb) FROM PUBLIC;
-- La función de operación conserva su OID, firma, propietario y ACL originales.
COMMIT;
