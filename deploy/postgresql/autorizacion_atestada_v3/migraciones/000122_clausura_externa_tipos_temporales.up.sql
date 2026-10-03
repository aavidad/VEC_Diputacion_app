\set ON_ERROR_STOP on
-- AD3-122: cierra la búsqueda de tipos de la clausura externa.
-- Conserva cuerpos, firmas, OID, propietarios, ACL, canon e historia.
-- Requiere AD3-118 y AD3-121 tras AUT21. No reaplicar sobre una instalación ya corregida.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('autorizacion_atestada_v3:migracion:externa:000122',0));
DO $correctiva$
DECLARE esperado record; antes record; despues record; acl_actual pg_catalog.jsonb;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL THEN
  RAISE EXCEPTION 'AD3-122: migrador no autorizado' USING ERRCODE='55000';
 END IF;
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','3e28e1daf77a874398e653fe46cdf30b1859c6209913914a16e68f484f43674c',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_atestada_v3_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','7a0b6f00e6a08e215e9b1e0e57cc0343ad91304a2f64499b38b18aead95e4374|ccbb09ae6ad4871353c206e2a8719deb9d307d4155912f111b95f4326b38d177',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_atestada_v3_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','a823ac4f0406f8f718f3c37164217d95ba85af1c11b0f82c7e05df8ee872dae6|f8316fb86a19af4cc12644b8adf5347435078a77dbf17de704473040024d901f',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::jsonb,'[{"grantee": "vec_autorizacion_atestada_v3_propietario", "grantor": "vec_autorizacion_atestada_v3_propietario", "grantable": false, "privilege": "EXECUTE"}]'::jsonb)
 ) AS inventario(firma,huella,definidor,configuracion,acl) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'AD3-122: preimagen incompatible para %',esperado.firma USING ERRCODE='55000'; END IF;
  SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'grantee',r.rolname,'grantor',g.rolname,'grantable',a.is_grantable,'privilege',a.privilege_type)
    ORDER BY r.rolname,g.rolname,a.privilege_type,a.is_grantable),'[]'::pg_catalog.jsonb)
  INTO acl_actual FROM pg_catalog.aclexplode(coalesce(antes.proacl,pg_catalog.acldefault('f',antes.proowner))) a
  LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee LEFT JOIN pg_catalog.pg_roles g ON g.oid=a.grantor;
  IF antes.proowner IS DISTINCT FROM pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario')
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR pg_catalog.to_jsonb(antes.proconfig) IS DISTINCT FROM esperado.configuracion
     OR NOT (pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex')=ANY(pg_catalog.string_to_array(esperado.huella,'|')))
     OR acl_actual IS DISTINCT FROM esperado.acl THEN
   RAISE EXCEPTION 'AD3-122: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::pg_catalog.regprocedure);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig'
     OR (SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog' THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
         FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice)) IS DISTINCT FROM despues.proconfig THEN
   RAISE EXCEPTION 'AD3-122: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $correctiva$;
COMMIT;
