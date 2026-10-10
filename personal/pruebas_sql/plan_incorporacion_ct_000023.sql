\set ON_ERROR_STOP on
-- Ensayo focal en el clon propio de Dirección, tras AD3-129/Personal23 y
-- Personal44 (que baja statement_timeout de las dos fachadas a 15 s).
-- No instala migraciones ni sustituye el consumidor V3. Los efectos positivos
-- requieren el recorrido Go con atestación real; estos casos son SQL/ACL.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $acl$
DECLARE f regprocedure; p record; t regclass;
BEGIN
 FOREACH f IN ARRAY ARRAY['vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
 SELECT proowner,prosecdef,provolatile,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_personal_propietario'::regrole OR NOT p.prosecdef OR p.provolatile<>'v'
	    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
    OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc q CROSS JOIN LATERAL aclexplode(coalesce(q.proacl,acldefault('f',q.proowner))) a WHERE q.oid=f AND a.grantee=0) THEN
  RAISE EXCEPTION 'Personal23: fachada incompatible'; END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['vec_personal.plan_incorporacion_ct'::regclass,
   'vec_personal.ejecucion_plan_incorporacion_ct'::regclass,'vec_personal.acceso_plan_incorporacion_ct'::regclass,
   'vec_personal.clases_ocupacion_plan_ct_catalogo'::regclass] LOOP
  IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid=t)
     OR has_table_privilege('vec_personal_ejecutor',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
     OR has_table_privilege('vec_contratacion_temporal_ejecutor',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
     OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
     OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>(CASE WHEN t='vec_personal.clases_ocupacion_plan_ct_catalogo'::regclass THEN 3 ELSE 2 END) THEN
   RAISE EXCEPTION 'Personal23: tabla o historia expuesta %',t; END IF;
 END LOOP;
 IF has_function_privilege('vec_personal_ejecutor','vec_personal.estado_plan_ct_interno(vec_personal.plan_incorporacion_ct)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb)','EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'Personal23: helper o puerto expuesto'; END IF;
END $acl$;
DO $denegacion$
DECLARE antes bigint; antes_b2 bigint;
BEGIN
 SELECT count(*) INTO antes FROM vec_personal.acceso_plan_incorporacion_ct;
 SELECT count(*) INTO antes_b2 FROM vec_personal.registro_empleado_b2_recibo;
 BEGIN
  PERFORM vec_personal.plan_incorporacion_ct_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'Personal23: llamada sin V3 admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.registrar_acto_plan_incorporacion_ct_v1('alta',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'Personal23: acto sin V3 admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.registrar_acto_plan_incorporacion_ct_v1('hecho',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'Personal23: hecho sin V3 admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.probar_origen_incorporacion_plan_v1(NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'Personal23: prueba de origen sin consumidor admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 IF (SELECT count(*) FROM vec_personal.acceso_plan_incorporacion_ct)<>antes
    OR (SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo)<>antes_b2 THEN
  RAISE EXCEPTION 'Personal23: denegación produjo historia'; END IF;
END $denegacion$;
SET LOCAL ROLE vec_personal_propietario;
DO $estructura$
DECLARE da text:='{"idempotencia_ref":"ed329ec1-e31a-4f4f-bb76-8d76ebd15fed","organismo_ref":"org:sintetico","clase_ocupacion":"temporal"}';
 negocio text; ph text; p vec_personal.plan_incorporacion_ct%ROWTYPE; estado jsonb;
 c vec_personal.clases_ocupacion_plan_ct_catalogo%ROWTYPE; canon_nuevo text;
 canon_reserva text:='{"opciones":[{"valor":"reserva","texto_clave":"rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.reserva","etiquetas":{"es":"Reserva","en":"Reserved"}}]}';
BEGIN
 SELECT * INTO STRICT c FROM vec_personal.clases_ocupacion_plan_ct_catalogo
 WHERE ref='personal:incorporacion_ct:clases_ocupacion' AND version=1;
 negocio:=encode(sha256(convert_to(da,'UTF8')),'hex');
 ph:=encode(sha256(convert_to(negocio||'|perplan_00000000000000000000000000000001|perplanrec_00000000000000000000000000000001|alta_empleado||11111111-1111-4111-8111-111111111111|22222222-2222-4222-8222-222222222222|uso:sintetico|reserva:sintetica|confirmacion:sintetica|'||c.ref||'|'||c.version::text||'|'||c.huella_sha256,'UTF8')),'hex');
 INSERT INTO vec_personal.plan_incorporacion_ct VALUES('ed329ec1-e31a-4f4f-bb76-8d76ebd15fed',
 'perplan_00000000000000000000000000000001','perplanrec_00000000000000000000000000000001',
 'org:sintetico',da,da::jsonb,negocio,'alta_empleado','',
 '11111111-1111-4111-8111-111111111111','22222222-2222-4222-8222-222222222222',
 'uso:sintetico','reserva:sintetica','confirmacion:sintetica',c.ref,c.version,c.huella_sha256,ph,'decision:sintetica','auditoria:sintetica',repeat('0',64),clock_timestamp()) RETURNING * INTO p;
 estado:=vec_personal.estado_plan_ct_interno(p);
 IF estado->>'estado' IS DISTINCT FROM 'preparado' OR estado->'recibo_alta_relacion' IS DISTINCT FROM 'null'::jsonb
    OR estado->'recibo_ocupacion' IS DISTINCT FROM 'null'::jsonb THEN
  RAISE EXCEPTION 'Personal23: plan sin efectos declaró progreso'; END IF;
 -- La versión nueva retira titular; el plan anterior conserva su versión v1.
 SELECT jsonb_build_object('opciones',jsonb_agg(o ORDER BY ord))::text INTO canon_nuevo
 FROM jsonb_array_elements(c.datos->'opciones') WITH ORDINALITY x(o,ord) WHERE o->>'valor'<>'titular';
 INSERT INTO vec_personal.clases_ocupacion_plan_ct_catalogo VALUES(c.ref,2,canon_nuevo,canon_nuevo::jsonb,
   encode(sha256(convert_to(canon_nuevo,'UTF8')),'hex'),clock_timestamp());
 BEGIN
  INSERT INTO vec_personal.clases_ocupacion_plan_ct_catalogo VALUES(c.ref,3,canon_reserva,canon_reserva::jsonb,
    encode(sha256(convert_to(canon_reserva,'UTF8')),'hex'),clock_timestamp());
  RAISE EXCEPTION 'Personal23: reserva publicada como ocupación efectiva';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 estado:=vec_personal.estado_plan_ct_interno(p);
 IF estado->'plan'->>'clases_ocupacion_catalogo_version' IS DISTINCT FROM '1'
    OR estado->'plan'->>'clases_ocupacion_catalogo_huella_sha256' IS DISTINCT FROM c.huella_sha256 THEN
  RAISE EXCEPTION 'Personal23: publicación nueva reescribió catálogo original'; END IF;
 p.datos:=jsonb_set(p.datos,'{clase_ocupacion}','"titular"'::jsonb);
 PERFORM vec_personal.validar_clase_plan_ct_interna(p);
 p.clases_ocupacion_catalogo_version:=2;
 p.clases_ocupacion_catalogo_huella_sha256:=encode(sha256(convert_to(canon_nuevo,'UTF8')),'hex');
 BEGIN
  PERFORM vec_personal.validar_clase_plan_ct_interna(p);
  RAISE EXCEPTION 'Personal23: clase retirada admitida en versión nueva';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 p.datos:=jsonb_set(p.datos,'{clase_ocupacion}','"temporal"'::jsonb);
 PERFORM vec_personal.validar_clase_plan_ct_interna(p);
 p.datos:=jsonb_set(p.datos,'{clase_ocupacion}','"sin_publicar"'::jsonb);
 BEGIN
  PERFORM vec_personal.validar_clase_plan_ct_interna(p);
  RAISE EXCEPTION 'Personal23: clase ausente admitida';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 BEGIN
  UPDATE vec_personal.clases_ocupacion_plan_ct_catalogo SET datos_canon='{}' WHERE ref=c.ref AND version=1;
  RAISE EXCEPTION 'Personal23: catálogo histórico modificado';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  UPDATE vec_personal.plan_incorporacion_ct SET clave_alta_relacion=gen_random_uuid() WHERE plan_ref=p.plan_ref;
  RAISE EXCEPTION 'Personal23: cambió clave reservada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  DELETE FROM vec_personal.plan_incorporacion_ct WHERE plan_ref=p.plan_ref;
  RAISE EXCEPTION 'Personal23: historia borrada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  TRUNCATE vec_personal.acceso_plan_incorporacion_ct;
  RAISE EXCEPTION 'Personal23: historia truncada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.validar_datos_plan_ct_interno(da::jsonb,da);
  RAISE EXCEPTION 'Personal23: datos parciales admitidos';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $estructura$;
ROLLBACK;
