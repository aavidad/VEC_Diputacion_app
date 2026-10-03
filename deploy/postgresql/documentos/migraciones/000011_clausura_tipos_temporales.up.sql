\set ON_ERROR_STOP on
-- D11: clausura nominal de tipos temporales; requiere Documentos10.
-- Solo modifica search_path. Conserva cuerpo, firma, OID, propietario, ACL e historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('documentos:migracion:000011',0));
DO $clausura$
DECLARE esperado record; antes record; despues record; configuracion pg_catalog.text[];
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed' THEN
  RAISE EXCEPTION 'D11: migrador no autorizado' USING ERRCODE='55000';
 END IF;
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_documentos.abrir_imagen_personal_v1(text,text)','vec_documentos_propietario','a7f66ab053f198e8551a01186eb5e03ab6d15bd7a81c3626877fc809b711b33b',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario,vec_usuarios_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.abrir_imagen_personal_v1_externa(text,text)','vec_documentos_propietario','327467527f736e6b93a48a7ecab3d7b9aac081d0ca16d05adcd1ffc3a5280657',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.abrir_imagen_personal_v1_interna(text,text)','vec_documentos_propietario','1af7d2c4daf25a105eaa90d46a46c958ec61a61980fbc702346f0a93343ce9b4',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)','vec_documentos_propietario','2984a4e6028ae1e844ee05899c276f95a663dacc7863b4f78a524d683bb43f5e',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario,vec_usuarios_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.custodiar_imagen_personal_v1_externa(text,bytea,text,text)','vec_documentos_propietario','75ebf57c5e5611d040a9aad247a22d79c9eb0ec199158f4e7ea881747a62d244',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.custodiar_imagen_personal_v1_interna(text,bytea,text,text)','vec_documentos_propietario','76a2712b438209e607f19b9ae35ac6f50e92927a5d6b6329a8277d69147b1101',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.imagen_personal_historia_inmutable_v1()','vec_documentos_propietario','c24a2b57da6309a809806cec456ce5fc1410fa4b307afdf9c9fb9b44145f098c',false,'["search_path=pg_catalog"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.imagen_personal_transicion_v1()','vec_documentos_propietario','abfe665c129cdef50b3ac81c700e49810680da25769c2e030643bf252ed381eb',false,'["search_path=pg_catalog"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.retirar_imagen_personal_v1(text,text,text)','vec_documentos_propietario','c39bfc38981eb70f3e70f1fb8c7fafd1571d68a37417e6945d2e41b9da3bcbf9',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario,vec_usuarios_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.retirar_imagen_personal_v1_externa(text,text,text)','vec_documentos_propietario','7638d017ca2214c98564cf9faebd442a5539fec073a8ca2b10cefb1b35c429ea',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.retirar_imagen_personal_v1_interna(text,text,text)','vec_documentos_propietario','ac56c61f2d224a30ae16f938eb81842a1bae557e820ae67d3e9aa29de6441a09',true,'["search_path=pg_catalog", "row_security=on", "lock_timeout=2s"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.sesion_usuarios_imagen_v1()','vec_documentos_propietario','928f5de67c968da402eea35409a19a2a53aec8455d79d9500c9bb1808533096c',false,'["search_path=pg_catalog"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}'),
  ('vec_documentos.superficie_sesion_imagen_v1()','vec_documentos_propietario','1241f5a97f89cc35c1837aecfa07157ef3ffdb465436783dda9b40b7eef4cc92',false,'["search_path=pg_catalog"]'::pg_catalog.jsonb,'{vec_documentos_propietario=X/vec_documentos_propietario}')
 ) AS inventario(firma,propietario,huella,definidor,configuracion,acl) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'D11: preimagen ausente para %',esperado.firma USING ERRCODE='55000'; END IF;
  IF antes.proowner IS DISTINCT FROM pg_catalog.to_regrole(esperado.propietario)
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR pg_catalog.to_jsonb(antes.proconfig) IS DISTINCT FROM esperado.configuracion
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperado.huella
     OR antes.proacl IS DISTINCT FROM esperado.acl::pg_catalog.aclitem[] THEN
   RAISE EXCEPTION 'D11: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog'
    THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
   INTO configuracion FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice);
  IF 'search_path=pg_catalog'=ANY(antes.proconfig) THEN
   EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::pg_catalog.regprocedure);
  ELSIF NOT ('search_path=pg_catalog, pg_temp'=ANY(antes.proconfig)) THEN
   RAISE EXCEPTION 'D11: ruta incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig'
     OR despues.proconfig IS DISTINCT FROM configuracion THEN
   RAISE EXCEPTION 'D11: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $clausura$;
COMMIT;
