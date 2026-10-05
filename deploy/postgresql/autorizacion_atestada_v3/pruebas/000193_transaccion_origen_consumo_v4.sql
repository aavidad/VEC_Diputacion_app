\set ON_ERROR_STOP on
-- Prueba estructural y vectores; no crea consumos manuales ni acredita camino causal.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
DO $estructura$
DECLARE tabla text;f oid;s text;definicion text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['consumo_decision_v3','auditoria_consumo_v3'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=to_regclass('vec_autorizacion_atestada_v3.'||tabla)
   AND a.attname='transaccion_origen' AND a.atttypid='xid8'::regtype AND NOT a.attnotnull
   AND NOT a.atthasdef AND NOT a.attisdropped)
  THEN RAISE EXCEPTION 'AD193: columna divergente %',tabla; END IF;
 END LOOP;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 SELECT p.prosrc,pg_get_functiondef(p.oid) INTO STRICT s,definicion FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM 'bb21afce73af87d532574c99da4f3ea8cd0534edc4910f14018d9ee55eb2a79b'
 OR encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM 'f581dbf9aa01d454caa6906ece774f97cf16cca9e9b8910d0eb348e23ef8c34b'
 OR strpos(s,'persona_denominacion_publicar')=0 OR strpos(s,'usuarios_admin_listar')=0
 THEN RAISE EXCEPTION 'AD193: núcleo posterior o ramas AD184/185 divergentes'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT definicion FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF strpos(definicion,'contexto_admin_pre_v2')=0
 THEN RAISE EXCEPTION 'AD193: rama AD192 ausente del CHECK'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
  WHERE (tipo_registro='consumo_confirmado_v4' AND (version_consumo IS DISTINCT FROM 4 OR transaccion_origen IS NULL))
   OR (tipo_registro<>'consumo_confirmado_v4' AND transaccion_origen IS NOT NULL))
 THEN RAISE EXCEPTION 'AD193: familia/sello divergentes'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)');
 SELECT prosrc INTO STRICT s FROM pg_proc WHERE oid=f;
 IF strpos(s,'r.consumo_xmin')>0 OR strpos(s,'r.auditoria_xmin')>0
 OR strpos(s,'r.consumo_transaccion_origen IS DISTINCT FROM pg_current_xact_id()')=0
 OR strpos(s,'r.auditoria_transaccion_origen IS DISTINCT FROM pg_current_xact_id()')=0
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole)
   OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'AD193: comprobador/ACL divergente'; END IF;
 -- Fachadas AD184/AD185: consumo fresco v4 sellado; cotejo admite v3 histórica o v4.
 FOREACH s IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  SELECT prosrc INTO STRICT definicion FROM pg_proc WHERE oid=s::regprocedure;
  IF strpos(definicion,$m$a.tipo_registro='consumo_confirmado_v4' AND a.version_consumo=4 AND a.transaccion_origen=pg_catalog.pg_current_xact_id() AND c.transaccion_origen=a.transaccion_origen$m$)=0
  OR strpos(definicion,'consumo_confirmado_v3')>0
  THEN RAISE EXCEPTION 'AD193: fachada de consumo fresco sin familia v4 sellada %',s; END IF;
 END LOOP;
 SELECT prosrc INTO STRICT definicion FROM pg_proc
 WHERE oid='vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'::regprocedure;
 IF strpos(definicion,$m$a.tipo_registro='consumo_confirmado_v3' AND a.version_consumo=3 AND a.transaccion_origen IS NULL AND s.transaccion_origen IS NULL$m$)=0
 OR strpos(definicion,$m$a.tipo_registro='consumo_confirmado_v4' AND a.version_consumo=4 AND a.transaccion_origen IS NOT NULL AND s.transaccion_origen=a.transaccion_origen$m$)=0
 THEN RAISE EXCEPTION 'AD193: cotejo de denominación sin familias v3/v4'; END IF;
 -- En todos los esquemas: un consumidor futuro de otro módulo tampoco puede exigir sólo v3.
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.prosrc LIKE '%consumo_confirmado_v3%' AND p.oid NOT IN(
   'vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'::regprocedure))
 THEN RAISE EXCEPTION 'AD193: queda una función que exige sólo la familia v3'; END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1('{}'::jsonb);
  RAISE EXCEPTION 'AD193: aceptó recibo vacío';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $estructura$;
DO $vectores$
DECLARE campos text[]:=ARRAY['consumo_confirmado_v4','4','1','0000000000000000000000000000000000000000000000000000000000000000','decision:prueba','efecto:prueba','1111111111111111111111111111111111111111111111111111111111111111','2222222222222222222222222222222222222222222222222222222222222222','proceso.prueba','interna_corporativa','2026-10-04T00:00:00.000000Z','2026-10-04T00:00:00.000000Z','persona:prueba','perfil:prueba','finalidad:prueba','9007199254740993'];campo text;preimagen bytea:=''::bytea;h text;
BEGIN
 FOREACH campo IN ARRAY campos LOOP
  preimagen:=preimagen||vec_autorizacion_atestada_v3.encuadrar_mac(campo);
 END LOOP;
 h:=encode(sha256(preimagen),'hex');
 IF h IS DISTINCT FROM '1da186ae6fb3e4f7a514976f944526653355d1f0b7e9105aa25cb14cb4ec8e4d'
 THEN RAISE EXCEPTION 'AD193: vector v4 divergente %',h; END IF;
 -- Valores fuera de JSON safe integer y máximo completo, siempre string decimal.
 FOREACH campo IN ARRAY ARRAY['9007199254740993','18446744073709551615'] LOOP
  IF (campo::xid8)::text IS DISTINCT FROM campo
   OR jsonb_typeof(to_jsonb((campo::xid8)::text)) IS DISTINCT FROM 'string'
   OR to_jsonb((campo::xid8)::text)#>>'{}' IS DISTINCT FROM campo
  THEN RAISE EXCEPTION 'AD193: xid8 truncado o JSON numérico'; END IF;
 END LOOP;
 IF encode(sha256(preimagen||vec_autorizacion_atestada_v3.encuadrar_mac('1')),'hex')=h
 THEN RAISE EXCEPTION 'AD193: sello no ligado al eslabón'; END IF;
END $vectores$;
ROLLBACK;
