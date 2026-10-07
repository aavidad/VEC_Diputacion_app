\set ON_ERROR_STOP on
-- Solo Dirección, migrador/superusuario del clon PostgreSQL 18 con B72 instalada.
-- No aplicar UP/DOWN ni ejecutar sobre principal. Datos exclusivamente sintéticos.
-- dobles de consumidores V3: noop autorización/PDP, no acredita criptografía ni PDP real.
-- No acredita SMTP, custodia ni entrega legal. Todo se restaura con ROLLBACK.
SELECT encode(sha256(convert_to(string_agg(pg_get_functiondef(p.oid),E'\n' ORDER BY p.oid),'UTF8')),'hex') AS consumidores_pre
 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN
 ('registrar_y_consumir_contacto_participacion_v3_atestada','registrar_y_consumir_consulta_contacto_v3_atestada') \gset
SELECT encode(sha256(convert_to(coalesce(string_agg(to_jsonb(c)::text,E'\n' ORDER BY c.contacto_ref),''),'UTF8')),'hex') AS contactos_pre
 FROM vec_bolsa_llamamientos.contacto_participacion c \gset
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $pre$ BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT (SELECT rolsuper FROM pg_roles WHERE rolname=current_user)
 OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'B72: requiere clon PostgreSQL 18, superusuario y B72 instalada'; END IF;
 RAISE NOTICE 'dobles de consumidores V3: noop autorización/PDP, no acredita criptografía ni PDP real';
END $pre$;
DO $dobles$
DECLARE f record; n integer:=0;
BEGIN
 FOR f IN SELECT p.oid,n.nspname,p.proname,pg_get_function_arguments(p.oid) args,pg_get_function_result(p.oid) resultado
 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN
 ('registrar_y_consumir_contacto_participacion_v3_atestada','registrar_y_consumir_consulta_contacto_v3_atestada') LOOP
  IF f.resultado IS DISTINCT FROM 'TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)'
  THEN RAISE EXCEPTION 'B72: firma de consumidor inesperada: %',f.proname; END IF;
  EXECUTE format($crear$CREATE OR REPLACE FUNCTION %I.%I(%s) RETURNS %s
   LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $cuerpo$
   SELECT 'decision:b72:doble'::text,convert_from($2,'UTF8')::jsonb->>'recurso_ref',
    repeat('a',64),repeat('b',64),'auditoria:b72:doble'::text,transaction_timestamp(),true $cuerpo$ $crear$,
   f.nspname,f.proname,f.args,f.resultado);
  n:=n+1;
 END LOOP;
 IF n<>2 THEN RAISE EXCEPTION 'B72: requiere exactamente dos consumidores'; END IF;
END $dobles$;
-- Preparación mínima basada en B70. replica solo durante altas constitutivas;
-- funciones, contactos, lectura e inmutabilidad se prueban con triggers normales.
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
 (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,vigente_desde,vigente_hasta,estado,registrada_en)
SELECT 'bolsa:b72:'||n,1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:b72:sintetica',
 '2026-09-01T00:00:00Z'::timestamptz,NULL,'vigente','2026-09-01T00:00:00Z'::timestamptz FROM generate_series(1,2) n;
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
 (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
SELECT 'inst:b72:'||n,1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:b72:'||n,1,
 encode(sha256('{}'::bytea),'hex'),CASE WHEN n=1 THEN 2 ELSE 1 END,
 '2026-09-01T00:00:00Z'::timestamptz,'2026-09-01T00:00:00Z'::timestamptz,'2026-09-01T00:00:00Z'::timestamptz FROM generate_series(1,2) n;
INSERT INTO vec_bolsa_llamamientos.constitucion
 (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:b72:'||n,'bolsa:b72:'||n,1,encode(sha256('{}'::bytea),'hex'),'inst:b72:'||n,1,
 encode(sha256('{}'::bytea),'hex'),'categoria:b72:sintetica','per_actoractoractoractoractor',
 '2026-09-01T00:00:00Z'::timestamptz,'2026-09-01T00:00:00Z'::timestamptz FROM generate_series(1,2) n;
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
 (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES ('inst:b72:1',1,1,'part:b72:1',1),('inst:b72:1',1,2,'part:b72:2',2),('inst:b72:2',1,1,'part:b72:9',1);
INSERT INTO vec_bolsa_llamamientos.oferta_publicada
 (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
SELECT 'oferta:'||repeat(letra,64),'recibo:oferta:'||repeat(letra,64),'bolsa:b72:'||bolsa,
 'per_actoractoractoractoractor','b72-oferta-'||letra,'{"categoria":"Categoría sintética","centro":"Centro de prueba","descripcion":"Ofrecimiento sintético"}'::jsonb,
 '{}'::jsonb,'2026-09-29T00:00:00Z'::timestamptz,'2026-10-03T00:00:00Z'::timestamptz,repeat('d',64),'decision:b72:oferta:'||letra
FROM (VALUES ('a',1),('b',1),('c',2)) v(letra,bolsa);
SET LOCAL session_replication_role=origin;
CREATE FUNCTION pg_temp.b72_decision(recurso text,consulta boolean) RETURNS bytea LANGUAGE sql IMMUTABLE AS $f$
 SELECT convert_to(jsonb_build_object('principal_id','per_actoractoractoractoractor',
 'accion',CASE WHEN consulta THEN 'bolsa.contacto_participacion.consultar' ELSE 'bolsa.contacto_participacion.registrar' END,
 'modulo_id','bolsa','tipo_recurso','participacion_bolsa',
 'finalidad',CASE WHEN consulta THEN 'consulta_contactos_participacion' ELSE 'gestion_contactos_participacion' END,
 'recurso_ref',recurso,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8') $f$;
CREATE FUNCTION pg_temp.b72_registrar(id text,bolsa text,part text,resultado text,oferta text,
 evidencia text DEFAULT NULL,huella text DEFAULT NULL,anotacion text DEFAULT 'Ensayo sintético',
 instante timestamptz DEFAULT '2026-10-01T10:00:00Z')
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text) LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v3(
 'contacto:b72:'||id,bolsa,part,NULL,'correo',instante,'per_actoractoractoractoractor',resultado,anotacion,
 'clave:b72:'||id,'recibo:b72:'||id,''::bytea,pg_temp.b72_decision(part,false),''::bytea,''::bytea,1::numeric,1::numeric,
 ''::bytea,''::bytea,''::bytea,''::bytea,oferta,evidencia,huella) $f$;
CREATE FUNCTION pg_temp.b72_listar(bolsa text,part text DEFAULT NULL,cursor text DEFAULT NULL,
 limite integer DEFAULT 100,oferta text DEFAULT NULL)
RETURNS TABLE(contacto_ref text,bolsa_ref text,participacion_ref text,llamamiento_ref text,canal text,
 instante timestamptz,actor text,resultado text,anotacion text,oferta_ref text,evidencia_ref text,evidencia_huella text)
LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.listar_contactos_participacion_v2(bolsa,part,cursor,limite,''::bytea,
 pg_temp.b72_decision(coalesce(part,bolsa),true),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea,oferta) $f$;
CREATE FUNCTION pg_temp.b72_error(sentencia text,esperado text,mensaje text DEFAULT NULL)
RETURNS void LANGUAGE plpgsql AS $f$
DECLARE estado text; detalle text;
BEGIN
 BEGIN EXECUTE sentencia;
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,detalle=MESSAGE_TEXT; END;
 IF estado IS DISTINCT FROM esperado OR (mensaje IS NOT NULL AND strpos(coalesce(detalle,''),mensaje)=0)
 THEN RAISE EXCEPTION 'B72: esperaba % (%), obtuvo % (%)',esperado,mensaje,estado,detalle; END IF;
END $f$;
DO $prueba$
DECLARE r record; ids text[]; n bigint; q text; a text:='oferta:'||repeat('a',64);
 b text:='oferta:'||repeat('b',64); c text:='oferta:'||repeat('c',64);
BEGIN
 SELECT * INTO STRICT r FROM pg_temp.b72_registrar('enviado','bolsa:b72:1','part:b72:1','enviado',a);
 IF r.reutilizado IS DISTINCT FROM false OR r.recibo_ref IS DISTINCT FROM 'recibo:b72:enviado'
 OR r.contacto_ref IS DISTINCT FROM 'contacto:b72:enviado' THEN RAISE EXCEPTION 'B72: recibo de alta incorrecto'; END IF;
 SELECT * INTO STRICT r FROM pg_temp.b72_registrar('enviado','bolsa:b72:1','part:b72:1','enviado',a);
 IF r.reutilizado IS DISTINCT FROM true OR r.recibo_ref IS DISTINCT FROM 'recibo:b72:enviado'
 OR r.contacto_ref IS DISTINCT FROM 'contacto:b72:enviado' THEN RAISE EXCEPTION 'B72: replay perdió recibo'; END IF;
 PERFORM pg_temp.b72_registrar('entrega','bolsa:b72:1','part:b72:1','entrega_declarada',a,'evidencia:b72:sintetica',repeat('e',64),'Ensayo sintético','2026-10-01T11:00:00Z');
 SELECT * INTO STRICT r FROM pg_temp.b72_registrar('entrega','bolsa:b72:1','part:b72:1','entrega_declarada',a,'evidencia:b72:sintetica',repeat('e',64),'Ensayo sintético','2026-10-01T11:00:00Z');
 IF r.reutilizado IS DISTINCT FROM true OR r.recibo_ref IS DISTINCT FROM 'recibo:b72:entrega' THEN RAISE EXCEPTION 'B72: replay evidencia perdió recibo'; END IF;
 PERFORM pg_temp.b72_registrar('rebote','bolsa:b72:1','part:b72:2','no_entregado',a,NULL,NULL,'Ensayo sintético','2026-10-01T11:00:00Z');
 PERFORM pg_temp.b72_registrar('otra','bolsa:b72:1','part:b72:1','enviado',b);
 PERFORM pg_temp.b72_registrar('ajena','bolsa:b72:2','part:b72:9','enviado',c);
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion WHERE contacto_ref IN('contacto:b72:entrega','contacto:b72:enviado'))<>2
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion WHERE contacto_ref='contacto:b72:entrega'
 AND evidencia_ref='evidencia:b72:sintetica' AND evidencia_huella=repeat('e',64) AND recibo_ref='recibo:b72:entrega' AND oferta_ref=a)
 THEN RAISE EXCEPTION 'B72: duplicó historia o perdió evidencia'; END IF;
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L,%L,%L)',
 'entrega','bolsa:b72:1','part:b72:1','entrega_declarada',a,'evidencia:b72:distinta',repeat('e',64)),'VBC01');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L,%L,%L)',
 'entrega','bolsa:b72:1','part:b72:1','entrega_declarada',a,'evidencia:b72:sintetica',repeat('f',64)),'VBC01');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L)',
 'enviado','bolsa:b72:1','part:b72:1','enviado',b),'VBC01');
 PERFORM pg_temp.b72_error('SELECT * FROM pg_temp.b72_registrar(''enviado'',''bolsa:b72:1'',''part:b72:1'',''enviado'',NULL)','VBC01');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L,NULL,NULL,%L)',
 'enviado','bolsa:b72:1','part:b72:1','enviado',a,'Anotación distinta'),'VBC01');
 -- v1 no admite enviado: mismo comando sin oferta debe fallar 22023,
 -- sin reutilizar recibo. Su guarda B72 es una defensa adicional.
 q:=$q$SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v1(
 'contacto:b72:enviado','bolsa:b72:1','part:b72:1',NULL,'correo','2026-10-01T10:00:00Z',
 'per_actoractoractoractoractor','enviado','Ensayo sintético','clave:b72:enviado','recibo:b72:enviado',
 ''::bytea,pg_temp.b72_decision('part:b72:1',false),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea)$q$;
 PERFORM pg_temp.b72_error(q,'22023');
 -- v2 admite no_entregado; se exige la guarda exacta y no una divergencia genérica.
 q:=$q$SELECT * FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v2(
 'contacto:b72:rebote','bolsa:b72:1','part:b72:2',NULL,'correo','2026-10-01T11:00:00Z',
 'per_actoractoractoractoractor','no_entregado','Ensayo sintético','clave:b72:rebote','recibo:b72:rebote',
 ''::bytea,pg_temp.b72_decision('part:b72:2',false),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea,NULL,NULL,NULL,NULL)$q$;
 PERFORM pg_temp.b72_error(q,'VBC01','contacto de oferta requiere versión tres');
 SELECT array_agg(x.contacto_ref) INTO ids FROM pg_temp.b72_listar('bolsa:b72:1',NULL,NULL,1,a) x;
 IF ids IS DISTINCT FROM ARRAY['contacto:b72:rebote'] THEN RAISE EXCEPTION 'B72: orden/límite incorrecto'; END IF;
 SELECT array_agg(x.contacto_ref) INTO ids FROM pg_temp.b72_listar('bolsa:b72:1',NULL,'contacto:b72:rebote',1,a) x;
 IF ids IS DISTINCT FROM ARRAY['contacto:b72:entrega'] THEN RAISE EXCEPTION 'B72: cursor no desempata'; END IF;
 SELECT array_agg(x.contacto_ref) INTO ids FROM pg_temp.b72_listar('bolsa:b72:1',NULL,'contacto:b72:entrega',100,a) x;
 IF ids IS DISTINCT FROM ARRAY['contacto:b72:enviado'] THEN RAISE EXCEPTION 'B72: segunda página incorrecta'; END IF;
 SELECT count(*) INTO n FROM pg_temp.b72_listar('bolsa:b72:1','part:b72:1',NULL,100,a);
 IF n<>2 THEN RAISE EXCEPTION 'B72: filtro participación incorrecto'; END IF;
 SELECT count(*) INTO n FROM pg_temp.b72_listar('bolsa:b72:1',NULL,NULL,100,a);
 IF n<>3 THEN RAISE EXCEPTION 'B72: filtro oferta incorrecto'; END IF;
 SELECT count(*) INTO n FROM pg_temp.b72_listar('bolsa:b72:1');
 IF n<>4 THEN RAISE EXCEPTION 'B72: filtro bolsa incorrecto'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_temp.b72_listar('bolsa:b72:1',NULL,NULL,100,a) WHERE contacto_ref='contacto:b72:entrega'
 AND evidencia_ref='evidencia:b72:sintetica' AND evidencia_huella=repeat('e',64)) THEN RAISE EXCEPTION 'B72: GET perdió evidencia'; END IF;
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_listar(%L,NULL,NULL,100,%L)','bolsa:b72:1',c),'23503');
 PERFORM pg_temp.b72_error('SELECT * FROM pg_temp.b72_listar(''bolsa:b72:1'',''part:b72:9'')','23503');
 FOREACH q IN ARRAY ARRAY['contacto:b72:ajena','contacto:b72:otra','contacto:b72:inexistente'] LOOP
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_listar(%L,NULL,%L,100,%L)','bolsa:b72:1',q,a),'22023'); END LOOP;
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_listar(%L,%L,%L,100,%L)',
 'bolsa:b72:1','part:b72:1','contacto:b72:rebote',a),'22023');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L)',
 'cruce-oferta','bolsa:b72:1','part:b72:1','enviado',c),'23503');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L)',
 'cruce-part','bolsa:b72:1','part:b72:9','enviado',a),'23503');
 PERFORM pg_temp.b72_error(format('SELECT * FROM pg_temp.b72_registrar(%L,%L,%L,%L,%L)',
 'sin-evidencia','bolsa:b72:1','part:b72:1','entrega_declarada',a),'22023');
 PERFORM pg_temp.b72_error('SELECT * FROM pg_temp.b72_listar(''bolsa:b72:1'',NULL,NULL,0)','22023');
 PERFORM pg_temp.b72_error('SELECT * FROM pg_temp.b72_listar(''bolsa:b72:1'',NULL,NULL,101)','22023');
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_participacion WHERE bolsa_ref IN('bolsa:b72:1','bolsa:b72:2'))<>5
 THEN RAISE EXCEPTION 'B72: errores/replay modificaron historia'; END IF;
 PERFORM pg_temp.b72_error('UPDATE vec_bolsa_llamamientos.contacto_participacion SET anotacion=''Mutación sintética'' WHERE contacto_ref=''contacto:b72:enviado''','55000');
 PERFORM pg_temp.b72_error('DELETE FROM vec_bolsa_llamamientos.contacto_participacion WHERE contacto_ref=''contacto:b72:enviado''','55000');
 RAISE NOTICE 'B72 FUNCIONAL-OK: recibo/evidencia/replay único, legacy v1 22023 y v2 VBC01, filtros/cursor/cruces';
END $prueba$;
DO $acl$
DECLARE f regprocedure; t regclass:='vec_bolsa_llamamientos.contacto_participacion'::regclass;
BEGIN
 FOREACH f IN ARRAY ARRAY[
 'vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text)'::regprocedure,
 'vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'::regprocedure] LOOP
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee<>p.proowner AND (a.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_bolsa_llamamientos_propietario'::regrole
 AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s'])
 THEN RAISE EXCEPTION 'B72: ACL/definición de función incorrecta: %',f; END IF; END LOOP;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=t AND a.grantee<>c.relowner)
 OR EXISTS(SELECT 1 FROM pg_attribute c CROSS JOIN LATERAL aclexplode(c.attacl) a WHERE c.attrelid=t AND NOT c.attisdropped AND a.grantee<>'vec_bolsa_llamamientos_propietario'::regrole)
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=t AND relrowsecurity AND relforcerowsecurity)
 OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_policy WHERE polrelid=t AND polroles=ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid])
 THEN RAISE EXCEPTION 'B72: ACL/RLS de tabla incorrecta'; END IF;
 RAISE NOTICE 'B72 ACL-OK: funciones solo ejecutor/propietario, tabla cerrada y RLS forzada';
END $acl$;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $denegacion$ BEGIN
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.contacto_participacion LIMIT 1;
 RAISE EXCEPTION 'B72: ejecutor pudo leer tabla directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion$;
RESET ROLE;
ROLLBACK;
SELECT encode(sha256(convert_to(string_agg(pg_get_functiondef(p.oid),E'\n' ORDER BY p.oid),'UTF8')),'hex')=:'consumidores_pre' AS consumidores_restaurados
 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN
 ('registrar_y_consumir_contacto_participacion_v3_atestada','registrar_y_consumir_consulta_contacto_v3_atestada') \gset
SELECT encode(sha256(convert_to(coalesce(string_agg(to_jsonb(c)::text,E'\n' ORDER BY c.contacto_ref),''),'UTF8')),'hex')=:'contactos_pre' AS contactos_restaurados
 FROM vec_bolsa_llamamientos.contacto_participacion c \gset
\if :consumidores_restaurados
\else
 \echo 'B72: FALLO, consumidores no restaurados'
 \quit 1
\endif
\if :contactos_restaurados
\else
 \echo 'B72: FALLO, contactos no restaurados'
 \quit 1
\endif
\echo 'B72 ROLLBACK-OK: consumidores V3 e historia idénticos; dobles de consumidores V3: noop autorización/PDP, no acredita criptografía ni PDP real'
