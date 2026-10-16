\set ON_ERROR_STOP on
-- B123: identifica la clave primaria en el conflicto de la preparación B119.
-- La función devuelve recurso_ref y el nombre sin calificar era ambiguo.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000123',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  NOT EXISTS(SELECT 1 FROM pg_constraint WHERE
   conrelid='vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119'::regclass AND
   conname='preparacion_justificante_renuncia_b119_pkey' AND contype='p' AND
   pg_get_constraintdef(oid,false)='PRIMARY KEY (recurso_ref, contexto_ref)')
 THEN RAISE EXCEPTION 'B123: entorno o clave incompatibles' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(text,text,text,text,timestamptz)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_documentos_propietario=X/vec_bolsa_llamamientos_propietario}' OR
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
   'fa9d41edd40eb960e7c99ba1bc7d511a44568fd70c1ab33c5b576db7167c890c'
 THEN RAISE EXCEPTION 'B123: preimagen divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(p_persona_ref text, p_candidato_ref text, p_bolsa_ref text, p_contexto_ref text, p_corte timestamp with time zone)
 RETURNS TABLE(recurso_ref text, version_dueno bigint, participacion_ref text, llamamiento_ref text, candidato_ref text, persona_ref text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE participacion text; abierto record; recurso text; previa record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_externo_documentos_bolsa_datos'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B119: proyección propia incompatible' USING ERRCODE='42501'; END IF;
 IF p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_bolsa_ref IS NULL OR pg_catalog.octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
    OR p_corte IS NULL OR NOT pg_catalog.isfinite(p_corte)
    OR p_corte>pg_catalog.clock_timestamp()+interval '1 second'
 THEN RAISE EXCEPTION 'B119: selector propio inválido' USING ERRCODE='42501'; END IF;
 -- D22 ya consumió la decisión que liga persona, contexto y candidato único.
 -- Aquí sólo se consultan la participación y el llamamiento propios de Bolsa.
 SELECT p.participacion_ref INTO participacion
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 WHERE p.bolsa_ref=p_bolsa_ref;
 IF participacion IS NULL THEN
  RAISE EXCEPTION 'B119: participación ajena' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:portal:'||participacion,0));
 SELECT * INTO abierto FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(
  participacion,p_corte,ARRAY['contactado']::text[]);
 IF abierto.llamamiento_ref IS NULL THEN
  RAISE EXCEPTION 'B119: llamamiento no pendiente' USING ERRCODE='42501'; END IF;
 recurso:=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(participacion,abierto.llamamiento_ref);
 INSERT INTO vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119(
  recurso_ref,contexto_ref,persona_ref,candidato_ref,bolsa_ref,participacion_ref,llamamiento_ref,preparada_en)
 VALUES(recurso,p_contexto_ref,p_persona_ref,p_candidato_ref,p_bolsa_ref,participacion,
  abierto.llamamiento_ref,pg_catalog.clock_timestamp())
 ON CONFLICT ON CONSTRAINT preparacion_justificante_renuncia_b119_pkey DO NOTHING;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 p
 WHERE p.recurso_ref=recurso AND p.contexto_ref=p_contexto_ref;
 IF NOT FOUND OR previa.persona_ref IS DISTINCT FROM p_persona_ref OR
  previa.candidato_ref IS DISTINCT FROM p_candidato_ref OR previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR
  previa.participacion_ref IS DISTINCT FROM participacion OR previa.llamamiento_ref IS DISTINCT FROM abierto.llamamiento_ref
 THEN RAISE EXCEPTION 'B119: preparación propia divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT recurso,1::bigint,participacion,
  abierto.llamamiento_ref,p_candidato_ref,p_persona_ref;
END $function$
;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(text,text,text,text,timestamptz)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_documentos_propietario=X/vec_bolsa_llamamientos_propietario}' OR
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
   '25935fee192f86457347841a0824238ef267f1e1343cdd8ab6213314f1df1c28'
 THEN RAISE EXCEPTION 'B123: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
