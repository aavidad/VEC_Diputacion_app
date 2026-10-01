\set ON_ERROR_STOP on
-- AUT21: cierra la búsqueda de tipos de la clausura externa.
-- Conserva cuerpos, firmas, OID, propietarios, ACL, canon e historia.
-- Requiere AUT17/18 tras AUT20 y CTX17. No reaplicar sobre una instalación ya corregida.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('autorizacion:migracion:externa:000021',0));
DO $correctiva$
DECLARE esperado record; antes record; despues record; acl_actual pg_catalog.jsonb;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL THEN
  RAISE EXCEPTION 'AUT21: migrador no autorizado' USING ERRCODE='55000';
 END IF;
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion.obtener_instantanea_usuarios_externo_v1(text,text)','caad10fc9547930026b48412e2c84608f2c98a99b7c48776598b7870e2cddddc',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_fuente_usuarios_externa", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()','0180a5752c84875defdc49235e5d4a9cca4d94fc702a9c301214b9cbdcd386a0',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)','c830fc51eeb8e1a841ba33a176c6ded203bc9e8c2627c7d01c6a24fba48e17e1',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_publicador_usuarios_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','bc5308a55c5c77fc1ce66bcadddf7e0db5729df715b2781b52a8d506688201c0',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_publicador_usuarios_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric)','2c10d1aa680dead1bf66fe7ebfbc544bac3d8eace5e9231ad6f019c18cf5ecd9',true,'[["search_path=pg_catalog"], ["search_path=pg_catalog, pg_temp"]]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)','cf26c29fa9f8c6731a7fbe71f8971e5fcdfda6b9177e51ce79a690a410934f0c',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_registro_usuarios_externo", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)','f4ba37c90941e0b79c12a5b2cc3ba655f376e53519fbb1191aae5e39c06c988e',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.resolver_motivo_usuarios_externo_v1(text,integer,text,text,timestamp with time zone)','d4419cd655e9865fd11e5e58577b8e659f43481f78089e482dea4c93497a0eb3',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_motivos_usuarios_externos", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric)','ad51d827e094091f415d3e64ecd3b03fb4e3d5858fa3b552acb2991730014723',true,'[["search_path=pg_catalog"], ["search_path=pg_catalog, pg_temp"]]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric)','dc5c11cfbc507eafb8cf724ac32ec92f0d9fde5958f60f647ccf7e553febb234',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}, {"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','d65ac5ed77ce2c79af83126dbe92ff3fce82f32eadb64ac89e7a41f0b67a6d29',true,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb)','6a32d06608d62cb3caaaba1c15e3600b2ab998602154939821c29279a89d2fc8',false,'["search_path=pg_catalog"]'::jsonb,'[{"grantee": "vec_autorizacion_propietario", "grantor": "vec_autorizacion_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb)
 ) AS inventario(firma,huella,definidor,configuracion,acl) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT21: preimagen incompatible para %',esperado.firma USING ERRCODE='55000'; END IF;
  SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'grantee',r.rolname,'grantor',g.rolname,'grantable',a.is_grantable,'privilege',a.privilege_type)
    ORDER BY r.rolname,g.rolname,a.privilege_type,a.is_grantable),'[]'::pg_catalog.jsonb)
  INTO acl_actual FROM pg_catalog.aclexplode(coalesce(antes.proacl,pg_catalog.acldefault('f',antes.proowner))) a
  LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee LEFT JOIN pg_catalog.pg_roles g ON g.oid=a.grantor;
  IF antes.proowner IS DISTINCT FROM pg_catalog.to_regrole('vec_autorizacion_propietario')
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR NOT (pg_catalog.to_jsonb(antes.proconfig)=esperado.configuracion
       OR (pg_catalog.jsonb_typeof(esperado.configuracion->0)='array' AND EXISTS(
          SELECT 1 FROM pg_catalog.jsonb_array_elements(esperado.configuracion) c WHERE c=pg_catalog.to_jsonb(antes.proconfig))))
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperado.huella
     OR acl_actual IS DISTINCT FROM esperado.acl THEN
   RAISE EXCEPTION 'AUT21: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::pg_catalog.regprocedure);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig'
     OR (SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog' THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
         FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice)) IS DISTINCT FROM despues.proconfig THEN
   RAISE EXCEPTION 'AUT21: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $correctiva$;
COMMIT;
