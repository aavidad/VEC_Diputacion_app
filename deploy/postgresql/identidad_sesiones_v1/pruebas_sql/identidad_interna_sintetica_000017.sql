\set ON_ERROR_STOP on
-- Sólo clon desechable, después de IS17. Dirección carga una fixture sintética
-- vigente mediante parámetros enlazados en vec.ensayo.plan_identidad y
-- vec.ensayo.material_identidad; no imprimir el material privado.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $vectores$
DECLARE p jsonb:=current_setting('vec.ensayo.plan_identidad')::jsonb;
 m text:=current_setting('vec.ensayo.material_identidad');
 pre jsonb; sha text; r jsonb; repetido jsonb; plan_sha text;
 antes bigint; privilegiadas bigint; politicas bigint; rechazo boolean; malo jsonb;
BEGIN
 IF p IS NULL OR m IS NULL THEN RAISE EXCEPTION 'IS17 pruebas: falta fixture privada'; END IF;
 plan_sha:=encode(pg_catalog.sha256(convert_to(p::text,'UTF8')),'hex');
 SELECT count(*) INTO antes FROM vec_identidad_sesiones_v1.cuenta;
 SELECT count(*) INTO privilegiadas FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_privilegiada;
 SELECT count(*) INTO politicas FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1;
 pre:=vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(p,m);
 sha:=encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex');
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,repeat('0',64),p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS17: CAS divergente aceptado'; END IF;
 FOR malo IN SELECT jsonb_set(p,'{entorno}','"produccion"')
 UNION ALL SELECT p||'{"perfil":"administrador"}'::jsonb
 UNION ALL SELECT p||'{"politica_admin":{}}'::jsonb
 UNION ALL SELECT jsonb_set(p,'{organizacion,version_esperada}','0')
 UNION ALL SELECT (p #- '{organizacion,procedencia_huella_sha256}')
 UNION ALL SELECT jsonb_set(p,'{organizacion,procedencia_huella_sha256}',to_jsonb(repeat('0',64)))
 UNION ALL SELECT jsonb_set(p,'{persona,version_esperada}','1')
 UNION ALL SELECT jsonb_set(p,'{persona,fuente_titularidad}','null')
 UNION ALL SELECT jsonb_set(p,'{persona}',p->'persona'||'{"operacion_cuenta_privilegiada_ref":"opr_no_admitida"}'::jsonb)
 UNION ALL SELECT jsonb_set(p,'{caduca_en}','"2000-01-01T00:00:00Z"') LOOP
  rechazo:=false;
  BEGIN
   PERFORM vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(malo,m);
  EXCEPTION WHEN invalid_parameter_value THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'IS17: plan inválido aceptado'; END IF;
 END LOOP;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(p,m||' ');
 EXCEPTION WHEN invalid_parameter_value THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS17: material divergente aceptado'; END IF;
 r:=vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,sha,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 repetido:=vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,sha,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 IF r IS DISTINCT FROM repetido OR r IS DISTINCT FROM vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(p->>'operacion_ref',plan_sha,'aprobacion_ensayo')
 THEN RAISE EXCEPTION 'IS17: replay cambia recibo'; END IF;
 IF (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta)<>antes+1
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_privilegiada)<>privilegiadas
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1)<>politicas
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.titularidad_identidad_interna_sintetica_v1 WHERE operacion_ref=p->>'operacion_ref')<>1
 THEN RAISE EXCEPTION 'IS17: cardinalidad tras replay divergente'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta c
 JOIN vec_identidad_sesiones_v1.titularidad_identidad_interna_sintetica_v1 t USING(cuenta_ref)
 WHERE c.cuenta_ref=r#>>'{datos,cuenta_ordinaria_ref}' AND NOT c.cuenta_privilegiada AND c.cuenta_ordinaria_ref IS NULL
 AND t.persona_ref=p#>>'{persona,persona_ref}' AND t.alcance_fuente='sintetico_declarado')
 THEN RAISE EXCEPTION 'IS17: cuenta real/titularidad divergente'; END IF;
 IF r->>'huella_sha256' IS DISTINCT FROM encode(pg_catalog.sha256(convert_to((r-'huella_sha256')::text,'UTF8')),'hex')
 OR r::text ~ '(hmac_hex|clave_hmac|sujeto_id_hmac|cuenta_privilegiada_ref)'
 THEN RAISE EXCEPTION 'IS17: recibo inválido o excesivo'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p,m,sha,p->>'operacion_ref',repeat('0',64),'aprobacion_ensayo');
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS17: replay con otra huella aceptado'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(jsonb_set(p,'{operacion_ref}',to_jsonb('piis_'||repeat('z',30))),m);
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS17: identidad duplicada con otra operación'; END IF;
 rechazo:=false;
 BEGIN
  UPDATE vec_identidad_sesiones_v1.identidad_interna_sintetica_v1 SET plan_sha256=repeat('0',64) WHERE operacion_ref=p->>'operacion_ref';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS17: historia mutable'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc pr JOIN pg_namespace n ON n.oid=pr.pronamespace CROSS JOIN LATERAL aclexplode(coalesce(pr.proacl,acldefault('f',pr.proowner))) a WHERE n.nspname='vec_identidad_sesiones_v1' AND pr.proname LIKE '%identidad_interna_sintetica%' AND a.grantee=0)
 THEN RAISE EXCEPTION 'IS17: función pública'; END IF;
END $vectores$;
SET CONSTRAINTS ALL IMMEDIATE;
ROLLBACK;
