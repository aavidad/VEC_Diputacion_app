\set ON_ERROR_STOP on
-- Sólo el clon sintético desechable de dirección, después de AD173.
-- DBA con lectura de catálogo y de la corriente común. No instala migraciones.
-- Para exigir un consumo auténtico previamente emitido:
-- psql -v ad173_exigir_consumo=1 -f ad173_consumo_fecha_nominal.sql
\if :{?ad173_exigir_consumo}
\else
\set ad173_exigir_consumo 0
\endif
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT set_config('vec.ad173_exigir_consumo', :'ad173_exigir_consumo', true);

DO $contratos$
DECLARE f oid; r record;
BEGIN
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND p.proargnames=ARRAY['p_perfil_mutacion','p_capacidad_canonica','p_decision_canonica','p_motivo_canonico','p_contexto_actor_canonico','p_persona_version','p_perfil_version','p_payload_vec_ad_3','p_sobre_cose_sign1','p_evidencia_verificacion','p_raiz_publica_spki','decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']
   AND (encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex'),encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')) IN (
    ('777f6a6e94c57cfdd8516d082441c1662e303c4b11aa1f187a542a9a11eeb2d6','528f35de95885283cc9987fc6f2ca8f09db59e61672b10f3c69134b1d91c23f6'),
    ('6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9','bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c')))
 THEN RAISE EXCEPTION 'AD173: PARO clave=nucleo_post actual=divergente esperado=POST173_11_argumentos_7_resultados'; END IF;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD173: PARO clave=ACL_nucleo actual=divergente esperado=solo_propietario'; END IF;
 FOR r IN SELECT c.relname FROM pg_class c WHERE c.oid IN (
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass) LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=format('vec_autorizacion_atestada_v3.%I',r.relname)::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
  OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=format('vec_autorizacion_atestada_v3.%I',r.relname)::regclass
    AND t.tgname IN ('inmutable','no_truncar') AND NOT t.tgisinternal AND t.tgenabled='O')<>2
  THEN RAISE EXCEPTION 'AD173: PARO clave=proteccion_% actual=divergente esperado=RLS_forzada_dos_disparadores',r.relname; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.confrelid='vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass
   AND c.contype='f' AND c.convalidated
   AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(id,n)
    JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.n)
    =ARRAY['decision_ref','efecto_ref','huella_efecto_sha256']
   AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY k(id,n)
    JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.n)
    =ARRAY['decision_ref','efecto_ref','huella_efecto_sha256'])
 THEN RAISE EXCEPTION 'AD173: PARO clave=FK_consumo actual=divergente esperado=terna_exacta'; END IF;
END $contratos$;

DO $familias$
DECLARE condicion text; base jsonb; v jsonb; permitido boolean; campo text; fecha text; r record;
BEGIN
 SELECT substr(pg_get_constraintdef(oid,false),8,length(pg_get_constraintdef(oid,false))-8)
 INTO STRICT condicion FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND conname='auditoria_tipo_disjunto_v4' AND contype='c' AND convalidated;
 base:=jsonb_build_object('auditoria_ref','aud_v3_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  'secuencia',7,'decision_ref','decision:ad173:pruéba','efecto_ref','efecto:ad173:prueba',
  'huella_efecto_sha256',repeat('a',64),'anterior_sha256',repeat('0',64),
  'huella_sha256',repeat('b',64),'registrada_en','2026-10-03T12:34:56.123456Z',
  'tipo_registro','consumo_confirmado');
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING base;
 IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD173: PARO clave=v1 actual=rechazado esperado=true'; END IF;
 v:=base||jsonb_build_object('tipo_registro','consumo_confirmado_v2','version_consumo',2,'proceso','rrhh_prueba','canal','interna_corporativa');
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v;
 IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD173: PARO clave=v2 actual=rechazado esperado=true'; END IF;
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object('actor_ref','persona:ad173:prueba');
 IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v2_nominal actual=aceptable esperado=false'; END IF;
 v:=v||jsonb_build_object('tipo_registro','consumo_confirmado_v3','version_consumo',3,
  'actor_ref','per_aaaaaaaaaaaaaaaaaaaaaa','perfil_activo_ref','prf_bbbbbbbbbbbbbbbbbbbbbb','finalidad_ref','gestion_personal');
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v;
 IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD173: PARO clave=v3 actual=rechazado esperado=true'; END IF;
 FOREACH campo IN ARRAY ARRAY['version_consumo','decision_ref','efecto_ref','huella_efecto_sha256','proceso','canal','actor_ref','perfil_activo_ref','finalidad_ref'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,NULL);
  -- CHECK permite NULL. Se exige FALSE para estas coordenadas, no NULL.
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v3_sin_% actual=aceptable esperado=false',campo; END IF;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['intento_ref','intento_material_sha256','registro_contexto_ref','contexto_sha256','procedencia_sha256','autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id','recurso_ref','resultado','motivo_ref','correlacion_ref','vinculo_sha256','evento_ref','evento_material_sha256','fuente_ref','fuente_sha256','operador_login','plan_sha256','aprobacion_ref'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,'mezcla');
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v3_con_% actual=aceptable esperado=false',campo; END IF;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['actor_ref','perfil_activo_ref','finalidad_ref'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,'');
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v3_vacio_% actual=aceptable esperado=false',campo; END IF;
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,repeat('a',513));
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v3_largo_% actual=aceptable esperado=false',campo; END IF;
 END LOOP;
 FOREACH fecha IN ARRAY ARRAY['infinity','-infinity','0001-01-01T00:00:00 BC','10000-01-01T00:00:00Z'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object('registrada_en',fecha);
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD173: PARO clave=v3_fecha_fuera actual=aceptable esperado=false'; END IF;
 END LOOP;
 FOREACH fecha IN ARRAY ARRAY['0001-01-01T00:00:00.000000Z','9999-12-31T23:59:59.999999Z'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object('registrada_en',fecha);
  IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD173: PARO clave=v3_fecha_extremo actual=rechazado esperado=true'; END IF;
 END LOOP;
 -- Todas las familias conservadas se contrastan; no se cambia ninguna fila.
 FOR r IN SELECT to_jsonb(a) AS fila FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.tipo_registro<>'consumo_confirmado_v3' LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING r.fila;
  IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD173: PARO clave=familia_historica actual=rechazada esperado=true'; END IF;
 END LOOP;
END $familias$;

DO $vector$
DECLARE huella text; alterada text; campos text[]; i integer;
BEGIN
 campos:=ARRAY['consumo_confirmado_v3','3','7',repeat('0',64),'decision:ad173:pruéba','efecto:ad173:prueba',repeat('a',64),repeat('b',64),'rrhh_prueba','interna_corporativa','2026-10-03T12:34:56.123456Z','2026-10-03T12:34:56.123456Z','per_aaaaaaaaaaaaaaaaaaaaaa','prf_bbbbbbbbbbbbbbbbbbbbbb','gestion_personal'];
 SELECT encode(sha256(string_agg(vec_autorizacion_atestada_v3.encuadrar_mac(valor),''::bytea ORDER BY orden)),'hex') INTO huella
 FROM unnest(campos) WITH ORDINALITY v(valor,orden);
 IF huella<>'0a8319971a070a9684e3deee8659d2a149b274e0c80f5973155c2a3d4f74050d'
 THEN RAISE EXCEPTION 'AD173: PARO clave=vector_UTF8 actual=% esperado=0a8319971a070a9684e3deee8659d2a149b274e0c80f5973155c2a3d4f74050d',huella; END IF;
 -- Alteración individual de las cinco coordenadas añadidas.
 FOR i IN 11..15 LOOP
  SELECT encode(sha256(string_agg(vec_autorizacion_atestada_v3.encuadrar_mac(CASE WHEN orden=i THEN valor||'x' ELSE valor END),''::bytea ORDER BY orden)),'hex') INTO alterada
  FROM unnest(campos) WITH ORDINALITY v(valor,orden);
  IF alterada=huella THEN RAISE EXCEPTION 'AD173: PARO clave=vector_coordenada_% actual=igual esperado=huella_distinta',i; END IF;
 END LOOP;
 IF to_char('2026-10-03T14:34:56.123456+02:00'::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM campos[11]
 THEN RAISE EXCEPTION 'AD173: PARO clave=UTC6 actual=divergente esperado=instante_equivalente'; END IF;
END $vector$;

DO $consumos_reales$
DECLARE r record; huella text; filas bigint:=0;
BEGIN
 FOR r IN SELECT a.*,c.consumo_huella_sha256,c.consumida_en,
    convert_from(t.decision_canonica,'UTF8')::jsonb AS decision,
    convert_from(t.contexto_actor_canonico,'UTF8')::jsonb AS contexto
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
   ON (a.decision_ref,a.efecto_ref,a.huella_efecto_sha256)=(c.decision_ref,c.efecto_ref,c.huella_efecto_sha256)
  JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
   ON (c.decision_ref,c.efecto_ref,c.huella_efecto_sha256)=(t.decision_ref,t.efecto_ref,t.huella_efecto_sha256)
  WHERE a.tipo_registro='consumo_confirmado_v3' LOOP
  filas:=filas+1;
  IF r.registrada_en IS DISTINCT FROM r.consumida_en
  THEN RAISE EXCEPTION 'AD173: PARO clave=instante_nominal actual=distinto esperado=registrada_igual_consumida'; END IF;
  IF r.actor_ref IS DISTINCT FROM r.decision->>'principal_id'
   OR r.actor_ref IS DISTINCT FROM r.contexto->>'principal_ref'
   OR r.perfil_activo_ref IS DISTINCT FROM r.decision->>'perfil_activo_ref'
   OR r.perfil_activo_ref IS DISTINCT FROM r.contexto->>'perfil_activo_ref'
   OR r.finalidad_ref IS DISTINCT FROM r.decision->>'finalidad'
   OR r.canal IS DISTINCT FROM r.decision#>>'{vinculo_autenticacion_actor,superficie}'
  THEN RAISE EXCEPTION 'AD173: PARO clave=coordenadas_nominales actual=distintas esperado=decision_contexto_canonicos'; END IF;
  IF r.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(r.consumo_huella_sha256,1,32)
  THEN RAISE EXCEPTION 'AD173: PARO clave=referencia_recibo actual=distinta esperado=derivacion_original'; END IF;
  SELECT encode(sha256(string_agg(vec_autorizacion_atestada_v3.encuadrar_mac(valor),''::bytea ORDER BY orden)),'hex') INTO huella
  FROM unnest(ARRAY[r.tipo_registro,r.version_consumo::text,r.secuencia::text,r.anterior_sha256,r.decision_ref,r.efecto_ref,r.huella_efecto_sha256,r.consumo_huella_sha256,r.proceso,r.canal,to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(r.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),r.actor_ref,r.perfil_activo_ref,r.finalidad_ref]) WITH ORDINALITY v(valor,orden);
  IF huella IS DISTINCT FROM r.huella_sha256
  THEN RAISE EXCEPTION 'AD173: PARO clave=huella_nominal actual=distinta esperado=15_campos'; END IF;
 END LOOP;
 IF filas<>(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='consumo_confirmado_v3')
 THEN RAISE EXCEPTION 'AD173: PARO clave=join_consumo actual=incompleto esperado=FK_exacta'; END IF;
 IF current_setting('vec.ad173_exigir_consumo')='1' AND filas=0
 THEN RAISE EXCEPTION 'AD173: PARO clave=fixture_nominal actual=0 esperado=consumo_firmado_real'; END IF;
 RAISE NOTICE 'AD173: filas nominales reales cotejadas=%; cero no acredita consumo positivo',filas;
END $consumos_reales$;
ROLLBACK;
SELECT 'AD173-CONTRATOS-FAMILIAS-VECTOR-OK; consumo positivo sólo con fixture real';
