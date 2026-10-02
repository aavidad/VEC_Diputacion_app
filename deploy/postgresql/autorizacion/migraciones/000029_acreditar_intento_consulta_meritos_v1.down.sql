\set ON_ERROR_STOP on
-- Sólo retirada de borrador sin consumidor instalado. Nunca sobre historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000029',0));
DO $retirada$ DECLARE anterior jsonb; actual text; funciones text; BEGIN
 IF to_regprocedure('vec_meritos.registrar_intento_consulta_propia_v1(text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT29: consumidor instalado, retirada prohibida' USING ERRCODE='55000'; END IF;
 anterior:=obj_description('vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text)'::regprocedure,'pg_proc')::jsonb;
 SELECT nspacl::text INTO actual FROM pg_namespace WHERE nspname='vec_autorizacion';
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('oid',p.oid,'def',pg_get_functiondef(p.oid),'acl',p.proacl) ORDER BY p.oid),'[]'::jsonb)::text,'UTF8')),'hex')
 INTO funciones FROM pg_proc p WHERE p.proowner='vec_meritos_propietario'::regrole AND p.prokind IN ('f','p');
 -- Cualquier cambio posterior de ACL o funciones del consumidor impide
 -- retirar un acceso al esquema que pudiera estar reutilizado.
 IF anterior IS NULL OR anterior->>'migracion' IS DISTINCT FROM 'AUT29'
 OR jsonb_typeof(anterior->'meritos_usage_preexisting') IS DISTINCT FROM 'boolean'
 OR actual IS DISTINCT FROM anterior->>'schema_acl_after'
 OR funciones IS DISTINCT FROM anterior->>'meritos_funciones_sha256'
 THEN RAISE EXCEPTION 'AUT29: preimagen de retirada incompatible' USING ERRCODE='55000'; END IF;
 IF anterior->'meritos_usage_preexisting'='false'::jsonb THEN
  REVOKE USAGE ON SCHEMA vec_autorizacion FROM vec_meritos_propietario;
  IF (SELECT nspacl::text FROM pg_namespace WHERE nspname='vec_autorizacion') IS DISTINCT FROM anterior->>'schema_acl_before'
  THEN RAISE EXCEPTION 'AUT29: ACL de retirada divergente' USING ERRCODE='55000'; END IF;
 END IF;
 DROP FUNCTION vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text);
END $retirada$;
COMMIT;
