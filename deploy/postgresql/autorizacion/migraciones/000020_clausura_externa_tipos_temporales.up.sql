\set ON_ERROR_STOP on
-- AUT20: cierra la búsqueda de tipos de la clausura externa.
-- Conserva cuerpos, firmas, OID, propietarios, ACL, canon e historia.
-- Requiere AUT15/16/19 y CTX15/17. No reaplicar sobre una instalación ya corregida.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('autorizacion:migracion:externa:000020',0));
DO $correctiva$
DECLARE esperado record; antes record; despues record; acl_actual pg_catalog.jsonb;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL THEN
  RAISE EXCEPTION 'AUT20: migrador no autorizado' USING ERRCODE='55000';
 END IF;
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)','5d59dfed401906eb4b4f378c9bfb38146136411b1657f5aecf7d26ef85a4d5be',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)','b15b19661f3d1b2fcb9ab33125195e7e51c956515eafde5be708c2343860cdc3',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.lista_textos_v3_canonica(jsonb)','d14a6e54c7f9d4405febb1a29b90f3ffb5edbdc31020e58a5998e8519dc27abc',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.login_candidato_externo_v1(text,text)','250f5a9f72a02d8f83a205f8844ac0faa596e706ca04d1c5ae39a99df73e5a4d',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.manifiesto_politicas_v3_canonico(jsonb)','da368126a2223dfb3cba8a00350447247af954b49f14a56aeec5894d18655ab1',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb)','a9b98ead2f6acb766ea0cdf18c1f29e6689e88999e8587a35dc912a774bbc91b',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text)','92e71941bc26301abe3a68dbfec059d191cb1eb60807044589744383096451be',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_fuente_externa", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicador_candidato_externo_interno_valido_v1()','1990a07d5658bc95f1722d905aeef0852522d7e4b92631bca59a284889ea75ce',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)','4c1801f62da2beb36fea79199fbf3006572ca1f1444c3030729e14458e873811',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_publicador_candidato_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','b58b39f2c35341a545640bfde366f07a89ee65831306950c7f3ea342ee846ff0',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_publicador_candidato_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)','ade14715c95cc1ef2c07b72b45074805cfe93857e441ba64e78a9148b39e5d09',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_registro_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','4dd4719d58c64526ec9f6001a2227d479a5182001cf9c125548898cb52fc2fa6',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','c26a73e66139837235aa58cf8853a0fa6ba2437d30cf085954863f6dc8b3c8da',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamp with time zone)','56de5fba7c797d1173a30806038978a469552f5c856a2dc86c556494e0feb4e1',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_motivos_externos", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric)','0d0b6beeb005323bfdb55cd3b9178ced5223f057ad7ce0792ce5cbf5f7986342',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','eeed4d2d437d463a5fbed163e23168126184dd2003d095fc4c90ceb0ee54052c',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','ce235e0456ea5dd4ede2ccf5d832f649e3a53c3c5f4f3fe57c4da06fba076fa2',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','d2c9a5b7cb02e784fe6633b37444b0f86f3b1124b2a8ba298939d183e09a9058',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','bb3d5019ebb1baf9b370db2fd75681eb6259b46ea33b3b71c2c03c2b6247c426',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)','4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.texto_ascii_visible_v3_valido(text,integer)','f7dca001802f15db281f9bd0e9067a33813f1a4d3579a3cd98799e586f43a7b7',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.texto_json_go_v3(text)','fc731f9aae3334ac6eb2fd0480cad8044326c526bf57f894d4fb83acc093c67c',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.validar_avance_asignacion_actual_externa()','f6b8e24db722c1d65b1cae02d9e48cf0af0ac08e49842fada0b5b3abc1793c16',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.vinculo_contexto_actor_v2_canonico(jsonb)','2a047d37516eed0e6311fb9efeba10cda6a9feca3becf3e792e1470d0e2db8d2',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.vinculo_contexto_actor_v2_valido(jsonb)','c3ceb7555f92db11efd0bfc9e01fdb0107a952da1a404c241dd7715625743845',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb)
 ) AS inventario(firma,huella,definidor,configuracion,acl) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT20: preimagen incompatible para %',esperado.firma USING ERRCODE='55000'; END IF;
  SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'grantee',r.rolname,'grantor',g.rolname,'grantable',a.is_grantable,'privilege',a.privilege_type)
    ORDER BY r.rolname,g.rolname,a.privilege_type,a.is_grantable),'[]'::pg_catalog.jsonb)
  INTO acl_actual FROM pg_catalog.aclexplode(coalesce(antes.proacl,pg_catalog.acldefault('f',antes.proowner))) a
  LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee LEFT JOIN pg_catalog.pg_roles g ON g.oid=a.grantor;
  IF antes.proowner IS DISTINCT FROM pg_catalog.to_regrole('vec_autorizacion_propietario')
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR pg_catalog.to_jsonb(antes.proconfig) IS DISTINCT FROM esperado.configuracion
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperado.huella
     OR acl_actual IS DISTINCT FROM esperado.acl THEN
   RAISE EXCEPTION 'AUT20: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::pg_catalog.regprocedure);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig'
     OR (SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog' THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
         FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice)) IS DISTINCT FROM despues.proconfig THEN
   RAISE EXCEPTION 'AUT20: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $correctiva$;
COMMIT;
