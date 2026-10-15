\set ON_ERROR_STOP on
-- AD282: plazo de claves y confianza del gobierno ADMIN nominal en desarrollo.
-- Parte de las dos funciones de efecto instaladas; preserva recibos y replay.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pre$
DECLARE f regprocedure; actual text; esperado text;
BEGIN
 IF current_setting('server_version_num')::integer < 180000
 OR to_regclass('vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'AD282: PARO clave=dependencias actual=ausentes esperado=PG18_AD188_AD198' USING ERRCODE='55000'; END IF;
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'9104549c5d240f06ad9b343559e15e9d46a7a7de736e682a57d47ecb57aa1d1b'),  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973')
 ) AS v(f,h) LOOP
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
  IF actual IS DISTINCT FROM esperado THEN
   RAISE EXCEPTION 'AD282: PARO clave=preimagen actual=% esperado=%',coalesce(actual,'ausente'),esperado USING ERRCODE='55000';
  END IF;
  IF (SELECT p.proowner::regrole::text='vec_autorizacion_atestada_v3_propietario'
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog']
      AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM true THEN
   RAISE EXCEPTION 'AD282: PARO clave=seguridad_funcion actual=distinta esperado=propietario_definer_acl' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;
LOCK TABLE vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1,
 vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 IN SHARE MODE;
-- Definiciones finales explícitas; cuerpos y huellas coinciden con la revisión.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(p_plan text, p_aprobado text, p_material text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
AS $function$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1;s vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1;
 p jsonb;m jsonb;g jsonb;pre jsonb;sha text;k jsonb;claves jsonb;root vec_autorizacion_atestada_v3.raiz_confianza_version;anterior record;v_confirmacion record;previo record;
 secret bytea;ord bigint;i integer:=0;claves_sha text;ahora timestamptz;n integer;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD198: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000';END IF;
 IF octet_length(p_plan)>16384 OR octet_length(p_material)>32768 THEN RAISE EXCEPTION 'AD198: PARO clave=entrada actual=grande esperado=acotada' USING ERRCODE='22023';END IF;
 SELECT * INTO STRICT s FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=c.conjunto_version;
 n:=cardinality(s.audiencias);
 sha:=encode(sha256(convert_to(p_plan,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_aprobado OR sha IS DISTINCT FROM c.plan_sha256
 OR encode(sha256(convert_to(p_material,'UTF8')),'hex') IS DISTINCT FROM c.material_sha256
 THEN RAISE EXCEPTION 'AD198: PARO clave=huella actual=distinta esperado=aprobada' USING ERRCODE='42501';END IF;
 p:=p_plan::jsonb;m:=p_material::jsonb;g:=p->'configuracion';claves:=m->'claves';
 IF jsonb_typeof(p)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>8
 OR NOT p ?& ARRAY['version','operacion_ref','preparado_en','caduca_en','preimagen_sha256','conjunto_version','configuracion','clave_ordenes']
 OR p->>'version'<>'2' OR p->'conjunto_version' IS DISTINCT FROM to_jsonb(c.conjunto_version)
 OR p->>'operacion_ref' !~ '^gca_[A-Za-z0-9_-]{22,124}$'
 OR p->>'preimagen_sha256' IS DISTINCT FROM c.preimagen_sha256
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp()
 OR jsonb_typeof(g)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(g))<>5 OR NOT g ?& ARRAY['revision','secuencia','huella_sha256','publicada_en','expira_en']
 OR jsonb_typeof(m)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>1
 OR jsonb_typeof(claves)<>'array' OR jsonb_array_length(claves)<>n
 OR jsonb_typeof(p->'clave_ordenes')<>'array' OR jsonb_array_length(p->'clave_ordenes')<>n
 THEN RAISE EXCEPTION 'AD198: PARO clave=plan actual=incompatible esperado=ABI198' USING ERRCODE='22023';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Mismo cerrojo que AD188 y los publicadores del gobierno; además CAS de tablas.
 LOCK TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual,vec_autorizacion_atestada_v3.puntero_clave_emision IN SHARE ROW EXCLUSIVE MODE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AD198: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 SELECT * INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='gobierno_usuarios_admin' AND plan_sha256=sha;
 IF FOUND THEN
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p,m);
  RETURN jsonb_build_object('auditoria_ref',previo.auditoria_ref,'secuencia',previo.secuencia,'huella_sha256',previo.huella_sha256,'correlacion_ref',previo.correlacion_ref,'registrada_en',previo.registrada_en,'replay',true,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',previo.gobierno_usuarios_detalle->>'claves_sha256','configuracion_ref',g->>'revision');
 END IF;
 pre:=vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(c.conjunto_version);
 IF encode(sha256(convert_to(pre::text,'UTF8')),'hex') IS DISTINCT FROM c.preimagen_sha256 THEN RAISE EXCEPTION 'AD198: PARO clave=preimagen actual=distinta esperado=aprobada' USING ERRCODE='40001';END IF;
 SELECT x.* INTO STRICT anterior FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision ORDER BY a.orden DESC LIMIT 1;
 SELECT x.* INTO STRICT root FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=anterior.revision;
 ahora:=clock_timestamp();
 IF root.valida_desde>ahora OR root.valida_hasta<=ahora OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r WHERE r.raiz_clave_id=root.clave_id AND r.raiz_version=root.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r WHERE r.configuracion_revision=anterior.revision)
 OR c.entorno IS DISTINCT FROM 'desarrollo'
 OR (g->>'publicada_en')::timestamptz>ahora
 OR date_trunc('day',(g->>'publicada_en')::timestamptz)<>date_trunc('day',ahora)
 OR (g->>'expira_en')::timestamptz<(g->>'publicada_en')::timestamptz+interval '1 hour'
 OR (g->>'expira_en')::timestamptz>(g->>'publicada_en')::timestamptz+interval '400 days'
 OR (g->>'expira_en')::timestamptz>root.valida_hasta
 OR g->>'revision' !~ '^confianza:atestacion:ct:desarrollo:[0-9]{4}-[0-9]{2}-[0-9]{2}(:r[0-9]+)?$'
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(g,root) IS DISTINCT FROM g->>'huella_sha256'
 OR g->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR (g->>'secuencia')::bigint<=anterior.secuencia
 OR (g->>'secuencia')::bigint<=(pre->>'orden_configuracion')::bigint
 THEN RAISE EXCEPTION 'AD198: PARO clave=renovacion actual=incompatible esperado=misma_raiz_diaria_aprobada' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(claves) LOOP
  i:=i+1;
  IF jsonb_typeof(p->'clave_ordenes'->(i-1))<>'number' THEN RAISE EXCEPTION 'AD198: PARO clave=orden actual=no_numerica esperado=entero' USING ERRCODE='22023';END IF;
  ord:=(p->'clave_ordenes'->>(i-1))::bigint;
  IF i>1 AND ord<=(p->'clave_ordenes'->>(i-2))::bigint THEN RAISE EXCEPTION 'AD198: PARO clave=orden actual=no_creciente esperado=creciente' USING ERRCODE='22023';END IF;
  IF jsonb_typeof(k)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(k))<>10
  OR NOT k ?& ARRAY['audiencia','clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac','huella_secreto_sha256','emisor_id','valida_desde','valida_hasta']
  OR k->>'audiencia' IS DISTINCT FROM s.audiencias[i]
  OR k->>'clave_id' !~ '^clave:capacidad:admin:[a-z0-9:._-]{1,160}$'
  OR left(k->>'clave_id',length('clave:capacidad:admin:'||s.segmentos[i]||':')) IS DISTINCT FROM 'clave:capacidad:admin:'||s.segmentos[i]||':'
  OR k->>'emisor_id' !~ '^emisor:admin:[a-z0-9:._-]{1,120}$'
  OR (k->>'version')::bigint NOT BETWEEN 1 AND 9007199254740991 OR (k->>'revision_gobierno')::bigint NOT BETWEEN 1 AND 9007199254740991
  OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora
  OR (k->>'valida_hasta')::timestamptz>root.valida_hasta
  OR (k->>'valida_hasta')::timestamptz>(g->>'expira_en')::timestamptz
  OR (k->>'valida_hasta')::timestamptz<(p->>'preparado_en')::timestamptz+interval '1 hour'
  OR (k->>'valida_hasta')::timestamptz>(p->>'preparado_en')::timestamptz+interval '400 days'
  OR ord<=(pre->>'orden_claves')::bigint OR ord>9007199254740991
  OR k->>'huella_gobierno_sha256' !~ '^[0-9a-f]{64}$' OR k->>'huella_secreto_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AD198: PARO clave=clave actual=incompatible esperado=audiencia_del_conjunto_aprobada' USING ERRCODE='22023';END IF;
  secret:=decode(k->>'secreto_hmac','base64');
  IF octet_length(secret)<>32 OR encode(sha256(secret),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' OR x.secreto_hmac=secret)
  THEN RAISE EXCEPTION 'AD198: PARO clave=material actual=incompatible esperado=nuevo_dedicado_32bytes' USING ERRCODE='42501';END IF;
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::bigint,(k->>'revision_gobierno')::bigint,k->>'huella_gobierno_sha256',secret,k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia',(k->>'valida_desde')::timestamptz,(k->>'valida_hasta')::timestamptz,'acto_tecnico:admin:capacidades:clave:'||sha||':'||i::text);
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) VALUES(ord,k->>'clave_id',(k->>'version')::bigint,ahora,'acto_tecnico:admin:capacidades:puntero:'||sha||':'||i::text);
 END LOOP;
 -- Sólo configuración y enlace/puntero nuevos; raíz original intacta.
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(g->>'revision',(g->>'secuencia')::bigint,g->>'huella_sha256',(g->>'publicada_en')::timestamptz,(g->>'expira_en')::timestamptz,'acto:ct:desarrollo:configuracion:r'||(g->>'secuencia'));
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES(g->>'revision',root.clave_id,root.version);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)
 VALUES((g->>'secuencia')::bigint,g->>'revision',(g->>'publicada_en')::timestamptz,'acto:ct:desarrollo:puntero-configuracion:r'||(g->>'secuencia'));
 claves_sha:=encode(sha256(convert_to((SELECT jsonb_agg(value-'secreto_hmac') FROM jsonb_array_elements(claves))::text,'UTF8')),'hex');
 SELECT * INTO v_confirmacion FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','gobierno_usuarios_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'configuracion_origen_ref',anterior.revision,'configuracion_destino_ref',g->>'revision','claves_sha256',claves_sha,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||substr(sha,1,32)));
 INSERT INTO vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1(auditoria_ref,tipo,funcion,operador_login,solicitud_sha256,conjunto_version,resultado,clave_ids)
 VALUES(v_confirmacion.auditoria_ref,'confirmacion','aprovisionar_gobierno_capacidades_admin_v1',session_user,sha,c.conjunto_version,'permitido',
  ARRAY(SELECT x->>'clave_id' FROM jsonb_array_elements(claves) x));
 PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p,m);
 RETURN to_jsonb(v_confirmacion)||jsonb_build_object('replay',false,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',claves_sha,'configuracion_ref',g->>'revision');
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(p_plan text, p_aprobado text, p_material text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
AS $function$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1;p jsonb;m jsonb;g jsonb;pre jsonb;sha text;k jsonb;claves jsonb;root vec_autorizacion_atestada_v3.raiz_confianza_version;anterior record;v_confirmacion record;previo record;secret bytea;ord bigint;i integer:=0;ref text;claves_sha text;ahora timestamptz;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD188: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000';END IF;
 IF octet_length(p_plan)>16384 OR octet_length(p_material)>16384 THEN RAISE EXCEPTION 'AD188: PARO clave=entrada actual=grande esperado=acotada' USING ERRCODE='22023';END IF;
 sha:=encode(sha256(convert_to(p_plan,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_aprobado OR sha IS DISTINCT FROM c.plan_sha256
 OR encode(sha256(convert_to(p_material,'UTF8')),'hex') IS DISTINCT FROM c.material_sha256
 THEN RAISE EXCEPTION 'AD188: PARO clave=huella actual=distinta esperado=aprobada' USING ERRCODE='42501';END IF;
 p:=p_plan::jsonb;m:=p_material::jsonb;g:=p->'configuracion';claves:=m->'claves';
 IF jsonb_typeof(p)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>7
 OR NOT p ?& ARRAY['version','operacion_ref','preparado_en','caduca_en','preimagen_sha256','configuracion','clave_ordenes']
 OR p->>'version'<>'1' OR p->>'operacion_ref' !~ '^gcu_[A-Za-z0-9_-]{22,124}$'
 OR p->>'preimagen_sha256' IS DISTINCT FROM c.preimagen_sha256
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp()
 OR jsonb_typeof(g)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(g))<>5 OR NOT g ?& ARRAY['revision','secuencia','huella_sha256','publicada_en','expira_en']
 OR jsonb_typeof(m)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>1
 OR jsonb_typeof(claves)<>'array' OR jsonb_array_length(claves)<>2 OR jsonb_array_length(p->'clave_ordenes')<>2
 THEN RAISE EXCEPTION 'AD188: PARO clave=plan actual=incompatible esperado=ABI188' USING ERRCODE='22023';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Publicadores del gobierno existente usan este cerrojo; además CAS de tablas.
 LOCK TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual,vec_autorizacion_atestada_v3.puntero_clave_emision IN SHARE ROW EXCLUSIVE MODE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AD188: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 SELECT * INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='gobierno_usuarios_admin' AND plan_sha256=sha;
 IF FOUND THEN
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p,m);
  RETURN jsonb_build_object('auditoria_ref',previo.auditoria_ref,'secuencia',previo.secuencia,'huella_sha256',previo.huella_sha256,'correlacion_ref',previo.correlacion_ref,'registrada_en',previo.registrada_en,'replay',true,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',previo.gobierno_usuarios_detalle->>'claves_sha256','configuracion_ref',g->>'revision');
 END IF;
 pre:=vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1();
 IF encode(sha256(convert_to(pre::text,'UTF8')),'hex') IS DISTINCT FROM c.preimagen_sha256 THEN RAISE EXCEPTION 'AD188: PARO clave=preimagen actual=distinta esperado=aprobada' USING ERRCODE='40001';END IF;
 SELECT x.* INTO STRICT anterior FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision ORDER BY a.orden DESC LIMIT 1;
 SELECT x.* INTO STRICT root FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=anterior.revision;
 ahora:=clock_timestamp();
 IF root.valida_desde>ahora OR root.valida_hasta<=ahora OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r WHERE r.raiz_clave_id=root.clave_id AND r.raiz_version=root.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r WHERE r.configuracion_revision=anterior.revision)
 OR c.entorno IS DISTINCT FROM 'desarrollo'
 OR (g->>'publicada_en')::timestamptz>ahora
 OR date_trunc('day',(g->>'publicada_en')::timestamptz)<>date_trunc('day',ahora)
 OR (g->>'expira_en')::timestamptz<(g->>'publicada_en')::timestamptz+interval '1 hour'
 OR (g->>'expira_en')::timestamptz>(g->>'publicada_en')::timestamptz+interval '400 days'
 OR (g->>'expira_en')::timestamptz>root.valida_hasta
 OR g->>'revision' !~ '^confianza:atestacion:ct:desarrollo:[0-9]{4}-[0-9]{2}-[0-9]{2}(:r[0-9]+)?$'
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(g,root) IS DISTINCT FROM g->>'huella_sha256'
 OR g->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR (g->>'secuencia')::bigint<=anterior.secuencia
 OR (g->>'secuencia')::bigint<=(pre->>'orden_configuracion')::bigint
 THEN RAISE EXCEPTION 'AD188: PARO clave=renovacion actual=incompatible esperado=misma_raiz_diaria_aprobada' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(claves) LOOP
  i:=i+1;ord:=(p->'clave_ordenes'->>(i-1))::bigint;
  IF jsonb_typeof(k)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(k))<>10
  OR NOT k ?& ARRAY['audiencia','clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac','huella_secreto_sha256','emisor_id','valida_desde','valida_hasta']
  OR k->>'audiencia' IS DISTINCT FROM (CASE i WHEN 1 THEN 'vec.admin.usuarios.listar.v1' ELSE 'vec.admin.usuarios.consultar.v1' END)
  OR k->>'clave_id' !~ '^clave:capacidad:admin:usuarios:[a-z0-9:._-]{1,120}$' OR k->>'emisor_id' !~ '^emisor:admin:[a-z0-9:._-]{1,120}$'
  OR (k->>'version')::bigint NOT BETWEEN 1 AND 9007199254740991 OR (k->>'revision_gobierno')::bigint NOT BETWEEN 1 AND 9007199254740991
  OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora
  OR (k->>'valida_hasta')::timestamptz>root.valida_hasta
  OR (k->>'valida_hasta')::timestamptz>(g->>'expira_en')::timestamptz
  OR (k->>'valida_hasta')::timestamptz<(p->>'preparado_en')::timestamptz+interval '1 hour'
  OR (k->>'valida_hasta')::timestamptz>(p->>'preparado_en')::timestamptz+interval '400 days'
  OR ord<=(pre->>'orden_claves')::bigint OR ord>9007199254740991
  OR k->>'huella_gobierno_sha256' !~ '^[0-9a-f]{64}$' OR k->>'huella_secreto_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AD188: PARO clave=clave actual=incompatible esperado=audiencia_dedicada_aprobada' USING ERRCODE='22023';END IF;
  secret:=decode(k->>'secreto_hmac','base64');
  IF octet_length(secret)<>32 OR encode(sha256(secret),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' OR x.secreto_hmac=secret)
  THEN RAISE EXCEPTION 'AD188: PARO clave=material actual=incompatible esperado=nuevo_dedicado_32bytes' USING ERRCODE='42501';END IF;
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::bigint,(k->>'revision_gobierno')::bigint,k->>'huella_gobierno_sha256',secret,k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia',(k->>'valida_desde')::timestamptz,(k->>'valida_hasta')::timestamptz,'acto_tecnico:admin:usuarios:clave:'||sha||':'||i::text);
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) VALUES(ord,k->>'clave_id',(k->>'version')::bigint,ahora,'acto_tecnico:admin:usuarios:puntero:'||sha||':'||i::text);
 END LOOP;
 -- Sólo configuración y enlace/puntero nuevos; raíz original intacta.
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(g->>'revision',(g->>'secuencia')::bigint,g->>'huella_sha256',(g->>'publicada_en')::timestamptz,(g->>'expira_en')::timestamptz,'acto:ct:desarrollo:configuracion:r'||(g->>'secuencia'));
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES(g->>'revision',root.clave_id,root.version);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)
 VALUES((g->>'secuencia')::bigint,g->>'revision',(g->>'publicada_en')::timestamptz,'acto:ct:desarrollo:puntero-configuracion:r'||(g->>'secuencia'));
 claves_sha:=encode(sha256(convert_to((SELECT jsonb_agg(value-'secreto_hmac') FROM jsonb_array_elements(claves))::text,'UTF8')),'hex');
 SELECT * INTO v_confirmacion FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','gobierno_usuarios_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'configuracion_origen_ref',anterior.revision,'configuracion_destino_ref',g->>'revision','claves_sha256',claves_sha,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||substr(sha,1,32)));
 PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p,m);
 RETURN to_jsonb(v_confirmacion)||jsonb_build_object('replay',false,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',claves_sha,'configuracion_ref',g->>'revision');
END $function$;

DO $post$
DECLARE f regprocedure; actual text; esperado text;
BEGIN
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'7a2c53d075e5c45a7455f2e8175498b13e8c82e120e08fcd9cdc79194d3c73a3'),  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'8330b0ad391ce73d62fc3f2781b2ada7094d1e61d4b340afca6ba68dd769639f')
 ) AS v(f,h) LOOP
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
  IF actual IS DISTINCT FROM esperado THEN
   RAISE EXCEPTION 'AD282: PARO clave=postimagen actual=% esperado=%',coalesce(actual,'ausente'),esperado USING ERRCODE='55000';
  END IF;
  IF (SELECT p.proowner::regrole::text='vec_autorizacion_atestada_v3_propietario'
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']
      AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM true
  THEN
   RAISE EXCEPTION 'AD282: PARO clave=seguridad_final actual=distinta esperado=propietario_definer_acl' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $post$;
COMMIT;
