\set ON_ERROR_STOP on
-- Clon causal AD160+AD164. Comprueba ACL, CAS e independencia durables;
-- los registros sintéticos del propietario no acreditan sesiones ni PDP.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $acl$
DECLARE nombre text; funcion regprocedure;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['preparar','aprobar','aplicar','recuperar'] LOOP
  funcion:=to_regprocedure('vec_autorizacion.'||nombre||'_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)');
  IF funcion IS NULL OR EXISTS(SELECT 1 FROM pg_roles r WHERE NOT r.rolsuper
    AND r.oid<>'vec_autorizacion_propietario'::regrole
    AND has_function_privilege(r.oid,funcion,'EXECUTE')) THEN
   RAISE EXCEPTION 'AD164: fachada abierta %',nombre; END IF;
 END LOOP;
 IF has_function_privilege('vec_autorizacion_cargos_ct_ejecutor',
   'vec_autorizacion.comprobar_independencia_cargo_ct_v1(text,bytea,text,text,boolean)','EXECUTE')
 OR NOT has_function_privilege('vec_contexto_actor_v1_propietario',
   'vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)','EXECUTE')
 OR strpos(pg_get_functiondef('vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)'::regprocedure),
   'comprobar_independencia_cargo_ct_v1(k,p.plan_canonico,h,NULL,true)')=0 THEN
  RAISE EXCEPTION 'AD164: comprobación o adjunto sin frontera'; END IF;
END $acl$;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $independencia$
DECLARE plan_bytes bytea; plan_sha text; clave text:=repeat('a',32);
 mala_clave text:=repeat('b',32); mala_plan bytea; mala_sha text;
 pendiente_clave text:=repeat('c',32); pendiente_plan bytea; pendiente_sha text;
 proponente text:='per_ad164_persona_proponente'; aprobador text:='per_ad164_persona_aprobadora';
 destino text:='per_ad164_persona_destinataria'; bloqueado boolean;
BEGIN
 plan_bytes:=convert_to(jsonb_build_object('clave',clave,'asignacion_canonica',
   encode(convert_to(jsonb_build_object('principal_id',destino)::text,'UTF8'),'base64'))::text,'UTF8');
 plan_sha:=encode(sha256(plan_bytes),'hex');
 INSERT INTO vec_autorizacion.cargo_ct_plan VALUES(clave,plan_bytes,plan_sha,proponente,
   'decision:ad164:proponer',clock_timestamp(),clock_timestamp()+interval '1 hour');
 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,plan_bytes,plan_sha,proponente,false);
 EXCEPTION WHEN insufficient_privilege THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'AD164: proponente pudo autoaprobar'; END IF;
 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,plan_bytes,plan_sha,destino,false);
 EXCEPTION WHEN insufficient_privilege THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'AD164: destinatario pudo aprobar'; END IF;
 IF vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,plan_bytes,plan_sha,aprobador,false) IS NOT TRUE THEN
  RAISE EXCEPTION 'AD164: dos personas distintas rechazadas'; END IF;
 INSERT INTO vec_autorizacion.cargo_ct_aprobacion VALUES(clave,plan_sha,aprobador,
   'decision:ad164:aprobar',convert_to(jsonb_build_object('principal_id',aprobador)::text,'UTF8'),
   convert_to('{}','UTF8'),1,1,clock_timestamp());
 -- Aplicación y replay validan las dos personas originales. No exigen una
 -- tercera persona ni confunden al lector actual con el aprobador histórico.
 IF vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,plan_bytes,plan_sha,NULL,true) IS NOT TRUE THEN
  RAISE EXCEPTION 'AD164: origen independiente no recuperable'; END IF;
 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,plan_bytes,repeat('f',64),NULL,true);
 EXCEPTION WHEN check_violation THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'AD164: CAS de huella aceptó divergencia'; END IF;

 mala_plan:=convert_to(jsonb_build_object('clave',mala_clave,'asignacion_canonica',
   encode(convert_to(jsonb_build_object('principal_id',destino)::text,'UTF8'),'base64'))::text,'UTF8');
 mala_sha:=encode(sha256(mala_plan),'hex');
 INSERT INTO vec_autorizacion.cargo_ct_plan VALUES(mala_clave,mala_plan,mala_sha,proponente,
   'decision:ad164:proponer:monopersona',clock_timestamp(),clock_timestamp()+interval '1 hour');
 INSERT INTO vec_autorizacion.cargo_ct_aprobacion VALUES(mala_clave,mala_sha,proponente,
   'decision:ad164:aprobar:monopersona',convert_to(jsonb_build_object('principal_id',proponente)::text,'UTF8'),
   convert_to('{}','UTF8'),1,1,clock_timestamp());
 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(mala_clave,mala_plan,mala_sha,NULL,true);
 EXCEPTION WHEN insufficient_privilege THEN bloqueado:=true;
 END;
 IF NOT bloqueado OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.cargo_ct_aprobacion
   WHERE cargo_ct_aprobacion.clave=mala_clave AND aprobador_ref=proponente) THEN
  RAISE EXCEPTION 'AD164: origen monopers no denegado o historia reescrita'; END IF;

 pendiente_plan:=convert_to(jsonb_build_object('clave',pendiente_clave,'asignacion_canonica',
   encode(convert_to(jsonb_build_object('principal_id',destino)::text,'UTF8'),'base64'))::text,'UTF8');
 pendiente_sha:=encode(sha256(pendiente_plan),'hex');
 INSERT INTO vec_autorizacion.cargo_ct_plan VALUES(pendiente_clave,pendiente_plan,pendiente_sha,proponente,
   'decision:ad164:proponer:pendiente',clock_timestamp(),clock_timestamp()+interval '1 hour');
 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(pendiente_clave,pendiente_plan,pendiente_sha,NULL,true);
 EXCEPTION WHEN insufficient_privilege THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'AD164: origen sin aprobación aceptado'; END IF;
END $independencia$;
ROLLBACK;
