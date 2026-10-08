\set ON_ERROR_STOP on
-- Requiere fixture privado que fije vec.ensayo.plan_identidad y
-- vec.ensayo.material_identidad; ejecutar sólo en clon desechable.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $ensayo$
DECLARE p jsonb:=current_setting('vec.ensayo.plan_identidad')::jsonb;
 m text:=current_setting('vec.ensayo.material_identidad');q jsonb;
 pre_ca jsonb;pre_is jsonb;sha_ca text;sha_is text;plan_sha text;is_r jsonb;ca_r jsonb;replay jsonb;rechazo boolean:=false;
 antes_personas bigint;antes_proyecciones bigint;antes_perfiles bigint;antes_cuentas bigint;antes_is bigint;
 legado record;legado_operacion text;invariante_falla boolean;
BEGIN
 plan_sha:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 SELECT count(*) INTO antes_personas FROM vec_contexto_actor_v1.persona_versiones;
 SELECT count(*) INTO antes_proyecciones FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones;
 SELECT count(*) INTO antes_perfiles FROM vec_contexto_actor_v1.perfil_versiones;
 SELECT count(*) INTO antes_cuentas FROM vec_identidad_sesiones_v1.cuenta;
 SELECT count(*) INTO antes_is FROM vec_identidad_sesiones_v1.identidad_interna_sintetica_v1;
 SELECT t.cuenta_ref,t.persona_ref,t.operacion_ref INTO legado FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t
 WHERE t.tipo_operacion='fuentes_iniciales_admin_v1' AND clock_timestamp()>=t.vigente_desde AND clock_timestamp()<t.vigente_hasta
 ORDER BY t.cuenta_ref LIMIT 1;
 IF NOT FOUND OR NOT vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(legado.cuenta_ref,legado.persona_ref,clock_timestamp()) THEN
  RAISE EXCEPTION 'CA37: fixture no acredita titularidad ADMIN canónica'; END IF;
 legado_operacion:=legado.operacion_ref;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND c.contype='p')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND c.contype='f' AND c.confrelid='vec_contexto_actor_v1.fuentes_iniciales_admin_v1'::regclass)
 OR NOT EXISTS(SELECT 1 FROM pg_trigger g WHERE g.tgrelid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND NOT g.tgisinternal AND g.tgenabled='O')
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND c.relrowsecurity AND c.relforcerowsecurity)
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND a.grantee=0 AND a.privilege_type IN('SELECT','INSERT','UPDATE','DELETE','TRUNCATE')) THEN
  RAISE EXCEPTION 'CA37: garantías históricas de titularidad no preservadas'; END IF;
 FOR q IN SELECT jsonb_set(p,'{organizacion,procedencia_huella_sha256}',to_jsonb((CASE WHEN left(p#>>'{organizacion,procedencia_huella_sha256}',1)='a' THEN 'b' ELSE 'a' END)||substr(p#>>'{organizacion,procedencia_huella_sha256}',2)))
 UNION ALL SELECT jsonb_set(p,'{organizacion,vigente_hasta}',to_jsonb(to_char((p#>>'{organizacion,vigente_hasta}')::timestamptz+interval '1 second','YYYY-MM-DD"T"HH24:MI:SS"Z"')))
 UNION ALL SELECT jsonb_set(p,'{organizacion,version_esperada}',to_jsonb(CASE WHEN (p#>>'{organizacion,version_esperada}')::numeric=1 THEN 2 ELSE 1 END)) LOOP
  invariante_falla:=false;
  BEGIN PERFORM vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(q);
  EXCEPTION WHEN serialization_failure THEN invariante_falla:=true; END;
  IF NOT invariante_falla THEN RAISE EXCEPTION 'CA37: divergencia de organización aceptada'; END IF;
 END LOOP;
 pre_ca:=vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(p);
 pre_is:=vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(p,m);
 sha_ca:=encode(sha256(convert_to(pre_ca::text,'UTF8')),'hex');
 sha_is:=encode(sha256(convert_to(pre_is::text,'UTF8')),'hex');
 BEGIN
  is_r:=vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,sha_is,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
  PERFORM vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(p,is_r,repeat('0',64),p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo OR (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones)<>antes_personas
 OR (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones)<>antes_proyecciones
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta)<>antes_cuentas
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.identidad_interna_sintetica_v1)<>antes_is THEN
  RAISE EXCEPTION 'CA37: CAS no preserva atomicidad'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t WHERE t.cuenta_ref=legado.cuenta_ref AND t.persona_ref=legado.persona_ref AND t.operacion_ref=legado_operacion AND t.tipo_operacion='fuentes_iniciales_admin_v1') THEN
  RAISE EXCEPTION 'CA37: rollback alteró titularidad ADMIN histórica'; END IF;
 is_r:=vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,sha_is,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 ca_r:=vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(p,is_r,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 replay:=vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(p,is_r,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 IF ca_r IS DISTINCT FROM replay OR ca_r->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to((ca_r-'huella_sha256')::text,'UTF8')),'hex')
 OR (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones)<>antes_personas+1
 OR (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones)<>antes_proyecciones+1
 OR (SELECT count(*) FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 WHERE operacion_identidad_ref=p->>'operacion_ref' AND tipo_operacion='identidad_interna_sintetica_v1')<>1
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t JOIN vec_contexto_actor_v1.persona_versiones pv ON pv.persona_ref=t.persona_ref AND pv.version=t.version JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv ON cv.cuenta_ref=t.cuenta_ref AND cv.version=t.version WHERE t.operacion_identidad_ref=p->>'operacion_ref' AND (pv.procedencia_autoridad<>'no_autoritativa' OR cv.procedencia_autoridad<>'no_autoritativa'))
 OR (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones)<>antes_perfiles
 THEN RAISE EXCEPTION 'CA37: replay o efecto ordinario divergente'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref IN(p#>>'{procedencia,referencia}',p#>>'{persona,fuente_titularidad,referencia}') AND procedencia_autoridad<>'no_autoritativa') THEN
  RAISE EXCEPTION 'CA37: declaración preparatoria elevada a fuente maestra'; END IF;
 IF NOT vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(ca_r#>>'{datos,cuenta_ordinaria_ref}',p#>>'{persona,persona_ref}',(ca_r->>'registrada_en')::timestamptz) THEN
  RAISE EXCEPTION 'CA37: fachada canónica no resuelve titularidad'; END IF;
 IF position('proyeccion_cuenta_actual' IN (SELECT prosrc FROM pg_proc WHERE oid=to_regprocedure('vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(text,text,timestamp with time zone)')))>0
 OR position('persona_actual' IN (SELECT prosrc FROM pg_proc WHERE oid=to_regprocedure('vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(text,text,timestamp with time zone)')))>0 THEN
  RAISE EXCEPTION 'CA37: cotejo histórico depende de puntero actual'; END IF;
 IF ca_r#>>'{datos,cuenta_ordinaria_ref}' IS DISTINCT FROM is_r#>>'{datos,cuenta_ordinaria_ref}' THEN
  RAISE EXCEPTION 'CA37: cuenta IS/CA divergente'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(p,is_r||'{"huella_sha256":"falsa"}'::jsonb,sha_ca,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'CA37: recibo IS inventado aceptado'; END IF;
 SET CONSTRAINTS ALL IMMEDIATE;
END $ensayo$;
ROLLBACK;
