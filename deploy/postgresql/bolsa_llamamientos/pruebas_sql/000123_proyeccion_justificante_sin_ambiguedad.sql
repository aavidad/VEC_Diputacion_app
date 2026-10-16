\set ON_ERROR_STOP on
-- En la copia propia, tras B123, pasar cuatro referencias ficticias vigentes:
-- -v b123_persona_ref=... -v b123_candidato_ref=... -v b123_bolsa_ref=...
-- -v b123_contexto_ref=... . El llamamiento debe seguir pendiente/contactado.
-- Todo el auxiliar y la preparación de esta prueba se deshacen con ROLLBACK.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $estructura$
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
 THEN RAISE EXCEPTION 'B123 prueba: función instalada divergente'; END IF;
END $estructura$;
CREATE TEMP TABLE b123_fixture(
 persona_ref text,candidato_ref text,bolsa_ref text,contexto_ref text
) ON COMMIT DROP;
INSERT INTO b123_fixture VALUES(
 :'b123_persona_ref',:'b123_candidato_ref',:'b123_bolsa_ref',:'b123_contexto_ref');
GRANT SELECT ON b123_fixture TO vec_externo_documentos_bolsa_datos;
-- Auxiliar transaccional: reproduce solo la llamada que hace D25 como
-- vec_documentos_propietario; session_user sigue siendo el LOGIN externo.
SET LOCAL ROLE vec_documentos_propietario;
CREATE FUNCTION vec_documentos.prueba_b123_proyeccion_rollback(
 p_persona_ref text,p_candidato_ref text,p_bolsa_ref text,p_contexto_ref text,p_corte timestamptz)
RETURNS TABLE(recurso_ref text,version_dueno bigint,participacion_ref text,
 llamamiento_ref text,candidato_ref text,persona_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(
  p_persona_ref,p_candidato_ref,p_bolsa_ref,p_contexto_ref,p_corte)
$f$;
REVOKE ALL ON FUNCTION vec_documentos.prueba_b123_proyeccion_rollback(
 text,text,text,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.prueba_b123_proyeccion_rollback(
 text,text,text,text,timestamptz) TO vec_externo_documentos_bolsa_datos;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_documentos_bolsa_datos;
DO $repeticion$
DECLARE f record; primera record; segunda record; esperado text;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b123_fixture;
 IF f.persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$' OR
  f.candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' OR
  f.bolsa_ref IS NULL OR octet_length(f.bolsa_ref) NOT BETWEEN 1 AND 256 OR
  f.contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B123 prueba: fixture ficticia inválida'; END IF;
 SELECT * INTO STRICT primera FROM vec_documentos.prueba_b123_proyeccion_rollback(
  f.persona_ref,f.candidato_ref,f.bolsa_ref,f.contexto_ref,clock_timestamp());
 SELECT * INTO STRICT segunda FROM vec_documentos.prueba_b123_proyeccion_rollback(
  f.persona_ref,f.candidato_ref,f.bolsa_ref,f.contexto_ref,clock_timestamp());
 esperado:='justificante-renuncia:'||encode(sha256(convert_to(
  primera.participacion_ref||chr(31)||primera.llamamiento_ref,'UTF8')),'hex');
 IF primera.recurso_ref IS DISTINCT FROM esperado OR
  primera.version_dueno IS DISTINCT FROM 1 OR
  primera.candidato_ref IS DISTINCT FROM f.candidato_ref OR
  primera.persona_ref IS DISTINCT FROM f.persona_ref OR
  to_jsonb(primera) IS DISTINCT FROM to_jsonb(segunda)
 THEN RAISE EXCEPTION 'B123 prueba: proyección o repetición divergente'; END IF;
END $repeticion$;
ROLLBACK;
\echo B123_PRUEBA_PROYECCION_REPETIDA_OK
