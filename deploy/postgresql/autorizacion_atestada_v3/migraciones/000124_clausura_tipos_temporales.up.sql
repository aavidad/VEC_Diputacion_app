\set ON_ERROR_STOP on
-- AD3-124: clausura nominal de tipos temporales; requiere AD3-122; conserva los cuatro contratos compartidos.
-- Solo modifica search_path. Conserva cuerpo, firma, OID, propietario, ACL e historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('autorizacion_atestada_v3:migracion:000124',0));
DO $clausura$
DECLARE esperado record; antes record; despues record; configuracion pg_catalog.text[];
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed' THEN
  RAISE EXCEPTION 'AD3-124: migrador no autorizado' USING ERRCODE='55000';
 END IF;
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','cba791c72804178722ffdf0c29f652b647d699460d3b888d5e5c20d2dae1ab8d',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_usuarios_correos_interno_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_usuarios_correos_externo_propietario=X/vec_autorizacion_atestada_v3_propietario}'),
  ('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','ec06da37e80683d279e8f2c4b65c9bdb2358120f5706fa92bb438b6c67daf38c',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_usuarios_propietario=X/vec_autorizacion_atestada_v3_propietario}'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','4012b2005d7801e2f784d86566237dc1e0d0cdfcc2060a94be4b8c99cdc16de9',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_usuarios_propietario=X/vec_autorizacion_atestada_v3_propietario}'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','0c60a0cb47454c94dea1923982981c5972d8136a1c46309257f81d0f2a38c242',true,'["search_path=pg_catalog", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_usuarios_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 ) AS inventario(firma,propietario,huella,definidor,configuracion,acl) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'AD3-124: preimagen ausente para %',esperado.firma USING ERRCODE='55000'; END IF;
  IF antes.proowner IS DISTINCT FROM pg_catalog.to_regrole(esperado.propietario)
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR pg_catalog.to_jsonb(antes.proconfig) IS DISTINCT FROM esperado.configuracion
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperado.huella
     OR antes.proacl IS DISTINCT FROM esperado.acl::pg_catalog.aclitem[] THEN
   RAISE EXCEPTION 'AD3-124: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog'
    THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
   INTO configuracion FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice);
  IF 'search_path=pg_catalog'=ANY(antes.proconfig) THEN
   EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::pg_catalog.regprocedure);
  ELSIF NOT ('search_path=pg_catalog, pg_temp'=ANY(antes.proconfig)) THEN
   RAISE EXCEPTION 'AD3-124: ruta incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig'
     OR despues.proconfig IS DISTINCT FROM configuracion THEN
   RAISE EXCEPTION 'AD3-124: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $clausura$;
COMMIT;
