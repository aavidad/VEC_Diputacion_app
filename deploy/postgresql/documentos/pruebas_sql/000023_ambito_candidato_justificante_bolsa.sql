\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $t$
DECLARE f text;p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.prosecdef IS NOT TRUE OR p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_documentos_propietario=X/vec_documentos_propietario}'
 OR position('candidato_propio_contexto_justificante_bolsa_v1(' IN f)=0
 OR position('"ambitos":{"candidato_ref":"%s"}' IN f)=0
 OR position('"ambitos":{"persona_ref":"%s"}' IN f)=0
 THEN RAISE EXCEPTION 'D23: ámbito o titularidad divergente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.prosecdef IS NOT TRUE OR p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
 OR position('d->>''principal_id'' IS DISTINCT FROM p_candidato_ref' IN f)=0
 OR position('p.persona_ref IS DISTINCT FROM p_persona_ref' IN f)=0
 THEN RAISE EXCEPTION 'D23: preparación propia divergente'; END IF;
END $t$;
ROLLBACK;
