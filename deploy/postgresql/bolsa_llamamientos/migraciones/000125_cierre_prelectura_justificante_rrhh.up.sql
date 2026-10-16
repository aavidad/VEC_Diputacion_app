\set ON_ERROR_STOP on
-- B125: cierra metadatos previos del justificante al LOGIN técnico de Bolsa.
-- La preparación documental nominal consume V3 antes de revelar la relación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000125',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE; f record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'B125: entorno incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)',
   '1151cbf5722257075404cb49f515f330764f848bd3bb59765f2c40d36fd92130'),
  ('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)',
   'd804271421192e6437c79f8b24f0b2bbae574e6c4271582ed10bc0d41217a6af')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
   p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
    'TimeZone=UTC','lock_timeout=2s']::text[] OR
   p.proacl::text IS DISTINCT FROM
    '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}' OR
   encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'B125: preimagen divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)
 FROM vec_bolsa_llamamientos_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)
 FROM vec_bolsa_llamamientos_ejecutor;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE; f record;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)',
   '1151cbf5722257075404cb49f515f330764f848bd3bb59765f2c40d36fd92130'),
  ('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)',
   'd804271421192e6437c79f8b24f0b2bbae574e6c4271582ed10bc0d41217a6af')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
   p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
    'TimeZone=UTC','lock_timeout=2s']::text[] OR
   p.proacl::text IS DISTINCT FROM
    '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}' OR
   encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella OR
   has_function_privilege('vec_bolsa_llamamientos_ejecutor',p.oid,'EXECUTE') OR
   EXISTS(SELECT 1 FROM aclexplode(p.proacl) a WHERE a.grantee=0)
  THEN RAISE EXCEPTION 'B125: cierre divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
