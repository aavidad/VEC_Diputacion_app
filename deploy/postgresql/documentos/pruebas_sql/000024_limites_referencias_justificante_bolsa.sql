\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $t$
DECLARE f record;p pg_proc%ROWTYPE;x text;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','5cae735018e27a2525a99ba5828686ca2b5fc7962b56158fd2d3b208f90522e0'),
  ('vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','81484f3b70ab7f4d80d71530f96e21aa206ea00c893ebce554bc9f6444d46083'),
  ('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','bebdbd096ab43212fe86f09b42eb3af976098e26c3f4ace626f611960f851bf6')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
  OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  OR position('{1,256}' IN pg_get_functiondef(p.oid))<>0
  THEN RAISE EXCEPTION 'D24: postimagen divergente %',f.firma; END IF;
 END LOOP;
 -- Evalúa la expresión ejecutable en PostgreSQL 18 con los límites reales.
 FOR x IN SELECT unnest(ARRAY['a',repeat('a',255),repeat('a',256)]) LOOP
  IF coalesce(x ~ '^[A-Za-z0-9:_-]+$' AND length(x) BETWEEN 1 AND 256,false) IS NOT TRUE
  OR (x !~ '^[A-Za-z0-9:_-]+$' OR length(x) NOT BETWEEN 1 AND 256) IS TRUE
  THEN RAISE EXCEPTION 'D24: referencia admitida rechazada, longitud %',length(x); END IF;
 END LOOP;
 FOR x IN SELECT unnest(ARRAY['',repeat('a',257),'a!','ñ']) LOOP
  IF coalesce(x ~ '^[A-Za-z0-9:_-]+$' AND length(x) BETWEEN 1 AND 256,false) IS TRUE
  OR (x !~ '^[A-Za-z0-9:_-]+$' OR length(x) NOT BETWEEN 1 AND 256) IS NOT TRUE
  THEN RAISE EXCEPTION 'D24: referencia inválida admitida, longitud %',length(x); END IF;
 END LOOP;
 IF coalesce(NULL::text ~ '^[A-Za-z0-9:_-]+$' AND length(NULL::text) BETWEEN 1 AND 256,false) IS TRUE
 OR (NULL::text !~ '^[A-Za-z0-9:_-]+$' OR length(NULL::text) NOT BETWEEN 1 AND 256) IS NOT NULL
 THEN RAISE EXCEPTION 'D24: semántica NULL alterada'; END IF;
END $t$;
ROLLBACK;
