\set ON_ERROR_STOP on
-- Dirección carga mediante parámetros enlazados dos GUC locales privados:
-- vec.ensayo.plan_fuentes / vec.ensayo.material_fuentes; no se imprimen.
-- Fixture de plan vigente, refs nuevas y HMAC sintéticos calculados por proveedor.
-- Sólo clon desechable. Todos los efectos de estos vectores terminan en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $vectores$
DECLARE p jsonb:=current_setting('vec.ensayo.plan_fuentes')::jsonb;
 m text:=current_setting('vec.ensayo.material_fuentes');
 pre jsonb; sha text; r jsonb; repetido jsonb; malo jsonb; plan_sha text;
 antes bigint; pe jsonb; cuentas bigint; rechazo boolean;
BEGIN
 IF p IS NULL OR m IS NULL THEN RAISE EXCEPTION 'IS15 pruebas: falta fixture privada'; END IF;
 plan_sha:=encode(pg_catalog.sha256(convert_to(p::text,'UTF8')),'hex');
 SELECT count(*) INTO antes FROM vec_identidad_sesiones_v1.cuenta;
 pre:=vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p,m);
 sha:=encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex');
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,repeat('0',64),p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS15: CAS divergente aceptado'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(jsonb_set(p,'{entorno}','"produccion"'),m);
 EXCEPTION WHEN invalid_parameter_value THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS15: producción aceptada'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p,m||' ');
 EXCEPTION WHEN invalid_parameter_value THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS15: material divergente aceptado'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p||'{"perfil":"administrador"}'::jsonb,m);
 EXCEPTION WHEN invalid_parameter_value THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS15: permiso del solicitante aceptado'; END IF;
 r:=vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,sha,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 repetido:=vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,sha,p->>'operacion_ref',plan_sha,'aprobacion_ensayo');
 IF r IS DISTINCT FROM repetido THEN RAISE EXCEPTION 'IS15: replay cambia recibo'; END IF;
 IF (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta)<>antes+4
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1 WHERE operacion_ref=p->>'operacion_ref')<>4
 THEN RAISE EXCEPTION 'IS15: cardinalidad tras replay divergente'; END IF;
 FOR pe IN SELECT value FROM jsonb_array_elements(r#>'{datos,personas}') LOOP
  SELECT count(*) INTO cuentas FROM vec_identidad_sesiones_v1.cuenta c
  JOIN vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1 t USING(cuenta_ref)
  WHERE t.persona_ref=pe->>'persona_ref'
  AND ((NOT c.cuenta_privilegiada AND c.cuenta_ref=pe->>'cuenta_ordinaria_ref')
   OR (c.cuenta_privilegiada AND c.cuenta_ref=pe->>'cuenta_privilegiada_ref' AND c.cuenta_ordinaria_ref=pe->>'cuenta_ordinaria_ref'));
  IF cuentas<>2 THEN RAISE EXCEPTION 'IS15: cuenta real/titularidad divergente'; END IF;
 END LOOP;
 IF r->>'huella_sha256' IS DISTINCT FROM encode(pg_catalog.sha256(convert_to((r-'huella_sha256')::text,'UTF8')),'hex')
 OR r::text ~ '(hmac_hex|clave_hmac|sujeto_id_hmac)' THEN RAISE EXCEPTION 'IS15: recibo inválido o excesivo'; END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,m,sha,p->>'operacion_ref',repeat('0',64),'aprobacion_ensayo');
 EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
 IF NOT rechazo THEN RAISE EXCEPTION 'IS15: replay material cambiado aceptado'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc pr JOIN pg_namespace n ON n.oid=pr.pronamespace CROSS JOIN LATERAL aclexplode(coalesce(pr.proacl,acldefault('f',pr.proowner))) a WHERE n.nspname='vec_identidad_sesiones_v1' AND pr.proname LIKE '%fuentes%admin%' AND a.grantee=0)
 THEN RAISE EXCEPTION 'IS15: función pública'; END IF;
END $vectores$;
ROLLBACK;
