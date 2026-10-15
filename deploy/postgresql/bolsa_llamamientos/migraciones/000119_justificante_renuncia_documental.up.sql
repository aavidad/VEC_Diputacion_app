\set ON_ERROR_STOP on
-- B119: el justificante pertenece a un llamamiento y una participación exactos.
-- Documentos conserva el original y revalida esta titularidad en su propia TX.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000119',0));

DO $pre$
DECLARE interna oid := pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)');
 externa oid := pg_catalog.to_regprocedure('vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR interna IS NULL OR externa IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.respuesta_justificante_documental_b119') IS NOT NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119') IS NOT NULL
    OR pg_catalog.to_regrole('vec_documentos_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_externo_documentos_bolsa_datos') IS NULL
    OR pg_catalog.to_regrole('vec_interno_documentos_bolsa_rrhh') IS NULL
    OR pg_catalog.to_regprocedure('vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(text,text)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(text,bigint,text,text)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B119: preimagen o dependencia ausente' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(interna),'UTF8')),'hex')
      IS DISTINCT FROM 'ec4db561e02ddcaa1551c5f6c40473c7686451bd8a130ca9088cb3aba2fac052'
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(externa),'UTF8')),'hex')
      IS DISTINCT FROM 'd34b7b3f4a9f0ebc703ed931ad26b9d89c640a27e5ad49bc593113b46f9e5d2b'
    OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=interna) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=externa) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    OR (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=interna) IS DISTINCT FROM
       '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
    OR (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=externa) IS DISTINCT FROM
       '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}'
 THEN RAISE EXCEPTION 'B119: definición B30 instalada divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(
 p_participacion_ref text,p_llamamiento_ref text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT CASE WHEN pg_catalog.octet_length(p_participacion_ref) BETWEEN 1 AND 512
   AND pg_catalog.octet_length(p_llamamiento_ref) BETWEEN 1 AND 512
   AND pg_catalog.strpos(p_participacion_ref,pg_catalog.chr(31))=0
   AND pg_catalog.strpos(p_llamamiento_ref,pg_catalog.chr(31))=0
  THEN 'justificante-renuncia:'||pg_catalog.encode(pg_catalog.sha256(
   pg_catalog.convert_to(p_participacion_ref||pg_catalog.chr(31)||p_llamamiento_ref,'UTF8')),'hex')
 END
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(text,text) FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119(
 recurso_ref text NOT NULL CHECK(recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$'),
 contexto_ref text NOT NULL CHECK(contexto_ref ~ '^rca_[A-Za-z0-9_-]{22,128}$'),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 candidato_ref text NOT NULL CHECK(candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 bolsa_ref text NOT NULL,
 participacion_ref text NOT NULL,
 llamamiento_ref text NOT NULL,
 preparada_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(preparada_en)),
 PRIMARY KEY(recurso_ref,contexto_ref)
);
ALTER TABLE vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 FORCE ROW LEVEL SECURITY;
CREATE POLICY preparacion_justificante_b119_propietario
 ON vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 FROM PUBLIC;
CREATE TRIGGER preparacion_justificante_b119_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.respuesta_justificante_documental_b119(
 respuesta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref)
  DEFERRABLE INITIALLY DEFERRED,
 documento_ref text NOT NULL UNIQUE,
 documento_sha256 text NOT NULL CHECK(documento_sha256 ~ '^[0-9a-f]{64}$'),
 recurso_ref text NOT NULL CHECK(recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$'),
 bolsa_ref text NOT NULL,
 participacion_ref text NOT NULL,
 llamamiento_ref text NOT NULL,
 candidato_ref text NOT NULL,
 persona_ref text NOT NULL,
 contexto_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 validada_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(validada_en))
);
ALTER TABLE vec_bolsa_llamamientos.respuesta_justificante_documental_b119 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.respuesta_justificante_documental_b119 FORCE ROW LEVEL SECURITY;
CREATE POLICY respuesta_justificante_b119_propietario
 ON vec_bolsa_llamamientos.respuesta_justificante_documental_b119 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.respuesta_justificante_documental_b119 FROM PUBLIC;
CREATE TRIGGER respuesta_justificante_b119_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.respuesta_justificante_documental_b119 FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- La fuente de persona/candidato es ContextoActor; el documento sólo aporta
-- el recurso ya resuelto. La respuesta puede aún no existir al subir el PDF.
CREATE FUNCTION vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(
 p_recurso_ref text,p_version_dueno bigint,p_persona_ref text,p_contexto_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE preparacion vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119%ROWTYPE;
 vigente text; llamamiento text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_externo_documentos_bolsa_datos'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B119: consumidor documental incompatible' USING ERRCODE='42501'; END IF;
 IF p_recurso_ref !~ '^justificante-renuncia:[0-9a-f]{64}$'
    OR p_version_dueno IS DISTINCT FROM 1
    OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
 THEN RETURN false; END IF;
 SELECT * INTO preparacion FROM vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 p
 WHERE p.recurso_ref=p_recurso_ref AND p.contexto_ref=p_contexto_ref
  AND p.persona_ref=p_persona_ref FOR SHARE;
 IF NOT FOUND OR preparacion.recurso_ref IS DISTINCT FROM
   vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(
    preparacion.participacion_ref,preparacion.llamamiento_ref) THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:portal:'||preparacion.participacion_ref,0));
 SELECT p.participacion_ref INTO vigente
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(preparacion.candidato_ref) p
 WHERE p.bolsa_ref=preparacion.bolsa_ref;
 IF vigente IS DISTINCT FROM preparacion.participacion_ref THEN RETURN false; END IF;
 SELECT a.llamamiento_ref INTO llamamiento
 FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(
  preparacion.participacion_ref,pg_catalog.clock_timestamp(),ARRAY['contactado']::text[]) a;
 RETURN llamamiento IS NOT DISTINCT FROM preparacion.llamamiento_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(text,bigint,text,text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_documentos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(text,bigint,text,text) TO vec_documentos_propietario;

-- Documentos autoriza y audita la consulta RRHH. Bolsa confirma únicamente
-- que el original citado terminó ligado a una renuncia justificada real.
CREATE FUNCTION vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(
 p_recurso_ref text,p_persona_titular_ref text,p_bolsa_ref text,p_participacion_ref text,
 p_actor_rrhh text,p_perfil_rrhh text) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_interno_documentos_bolsa_rrhh'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B119: consulta RRHH incompatible' USING ERRCODE='42501'; END IF;
 IF p_recurso_ref !~ '^justificante-renuncia:[0-9a-f]{64}$'
    OR p_persona_titular_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_bolsa_ref IS NULL OR pg_catalog.octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_participacion_ref IS NULL OR pg_catalog.octet_length(p_participacion_ref) NOT BETWEEN 1 AND 512
    OR p_actor_rrhh !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_perfil_rrhh !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 THEN RETURN false; END IF;
 RETURN EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  JOIN vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j ON j.respuesta_ref=r.respuesta_ref
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=r.participacion_ref
  JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.llamamiento_ref=r.llamamiento_ref
  WHERE vc.candidato_ref=r.candidato_ref
   AND r.participacion_ref=p_participacion_ref
   AND r.bolsa_ref=p_bolsa_ref AND l.bolsa_ref=p_bolsa_ref
   AND l.participaciones ? r.participacion_ref
   AND r.respuesta='renuncia_justificada' AND r.justificante_ref IS NOT NULL
   AND j.documento_ref=r.justificante_ref AND j.documento_sha256=r.justificante_sha256
   AND j.recurso_ref=p_recurso_ref AND j.bolsa_ref=p_bolsa_ref
   AND j.participacion_ref=p_participacion_ref AND j.llamamiento_ref=r.llamamiento_ref
   AND j.candidato_ref=r.candidato_ref AND j.persona_ref=p_persona_titular_ref
   AND vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(r.participacion_ref,r.llamamiento_ref)=p_recurso_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text,text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text,text) TO vec_documentos_propietario;

-- Sólo la fachada D22 puede llamar estas proyecciones. D22 consume y audita
-- la decisión V3 en la misma transacción antes de entrar en Bolsa.
CREATE FUNCTION vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(
 p_persona_ref text,p_candidato_ref text,p_bolsa_ref text,p_contexto_ref text,p_corte timestamptz)
RETURNS TABLE(recurso_ref text,version_dueno bigint,participacion_ref text,
 llamamiento_ref text,candidato_ref text,persona_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
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
 ON CONFLICT(recurso_ref,contexto_ref) DO NOTHING;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119 p
 WHERE p.recurso_ref=recurso AND p.contexto_ref=p_contexto_ref;
 IF NOT FOUND OR previa.persona_ref IS DISTINCT FROM p_persona_ref OR
  previa.candidato_ref IS DISTINCT FROM p_candidato_ref OR previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR
  previa.participacion_ref IS DISTINCT FROM participacion OR previa.llamamiento_ref IS DISTINCT FROM abierto.llamamiento_ref
 THEN RAISE EXCEPTION 'B119: preparación propia divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT recurso,1::bigint,participacion,
  abierto.llamamiento_ref,p_candidato_ref,p_persona_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(text,text,text,text,timestamptz) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(text,text,text,text,timestamptz) TO vec_documentos_propietario;

CREATE FUNCTION vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_actor_ref text,p_perfil_ref text,p_contexto_ref text)
RETURNS TABLE(documento_ref text,documento_sha256 text,recurso_ref text,persona_titular_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_interno_documentos_bolsa_rrhh'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B119: proyección RRHH incompatible' USING ERRCODE='42501'; END IF;
 IF p_bolsa_ref IS NULL OR pg_catalog.octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_participacion_ref IS NULL OR pg_catalog.octet_length(p_participacion_ref) NOT BETWEEN 1 AND 512
    OR p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_perfil_ref !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B119: selector RRHH inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT j.documento_ref,j.documento_sha256,j.recurso_ref,j.persona_ref
 FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j
 JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento r ON r.respuesta_ref=j.respuesta_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.llamamiento_ref=j.llamamiento_ref
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=j.participacion_ref
 WHERE j.bolsa_ref=p_bolsa_ref AND j.participacion_ref=p_participacion_ref
  AND r.bolsa_ref=p_bolsa_ref AND r.participacion_ref=p_participacion_ref
  AND r.llamamiento_ref=j.llamamiento_ref AND r.respuesta='renuncia_justificada'
  AND r.justificante_ref=j.documento_ref AND r.justificante_sha256=j.documento_sha256
  AND l.bolsa_ref=p_bolsa_ref AND l.participaciones ? p_participacion_ref
  AND vc.candidato_ref=r.candidato_ref AND vc.candidato_ref=j.candidato_ref
  AND j.recurso_ref=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(j.participacion_ref,j.llamamiento_ref)
 ORDER BY j.validada_en DESC,j.respuesta_ref DESC LIMIT 1;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text) TO vec_documentos_propietario;

-- B30 conserva la semántica y el recibo de aceptar/renunciar. Se añade una
-- prueba de custodia sólo para renuncia justificada, antes de insertar y antes
-- de devolver un replay. La interna no admite invocaciones que la eludan.
-- Definiciones finales literales de B30, cotejadas contra HZ37.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_respuesta text, p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamp with time zone, p_vence_antes_de timestamp with time zone, p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamp with time zone, p_decision_ref text)
 RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamp with time zone, modo text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE v_abierto record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_respuesta NOT IN ('acepta','renuncia','renuncia_justificada')
    OR p_modo NOT IN ('firme','propuesta_rrhh') OR p_respondida_en IS NULL OR p_vence_antes_de <= p_contacto_en THEN
  RAISE EXCEPTION 'respuesta del portal inválida' USING ERRCODE='22023';
 END IF;
 -- Sin contacto leído por el servidor no hay llamamiento que responder.
 IF p_contacto_en IS NULL OR p_vence_antes_de IS NULL THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto' USING ERRCODE='VBP04';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||p_participacion_ref, 0));
 SELECT * INTO v_abierto FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p_participacion_ref, p_respondida_en, p_resultados_efectivos);
 IF v_abierto.llamamiento_ref IS NULL OR v_abierto.contacto_en IS DISTINCT FROM p_contacto_en THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto con ese contacto' USING ERRCODE='VBP04';
 END IF;
 -- Fuera de plazo no se admite: queda como «sin respuesta» para RRHH.
 IF p_respondida_en >= p_vence_antes_de THEN
  RAISE EXCEPTION 'respuesta fuera de plazo' USING ERRCODE='VBP05';
 END IF;
 IF p_respuesta='renuncia_justificada' AND NOT EXISTS (
  SELECT 1 FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j
  WHERE j.respuesta_ref=p_respuesta_ref AND j.documento_ref=p_justificante_ref
   AND j.documento_sha256=p_justificante_sha256 AND j.bolsa_ref=p_bolsa_ref
   AND j.participacion_ref=p_participacion_ref AND j.llamamiento_ref=v_abierto.llamamiento_ref
   AND j.candidato_ref=p_candidato_ref AND j.decision_ref=p_decision_ref
   AND j.recurso_ref=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(
    p_participacion_ref,v_abierto.llamamiento_ref)) THEN
  RAISE EXCEPTION 'B119: original justificado no acreditado' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref, recibo_ref, bolsa_ref, participacion_ref, candidato_ref,
   llamamiento_ref, respuesta, causa, justificante_ref, justificante_sha256, modo, contacto_en, vence_antes_de, regla_ref,
   clave_idempotencia, decision_ref, respondida_en)
 VALUES (p_respuesta_ref, p_recibo_ref, p_bolsa_ref, p_participacion_ref, p_candidato_ref, v_abierto.llamamiento_ref, p_respuesta,
   p_causa, p_justificante_ref, p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_regla_ref, p_clave,
   p_decision_ref, p_respondida_en);
 RETURN QUERY SELECT false, p_respuesta_ref, p_recibo_ref, p_respondida_en, p_modo;
END $function$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_v1(p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_respuesta text, p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamp with time zone, p_vence_antes_de timestamp with time zone, p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamp with time zone, modo text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE v_participacion text; v_previa vec_bolsa_llamamientos.respuesta_portal_llamamiento%ROWTYPE; v_consumo record;
 v_dato text; v_decision_ref text;
 v_doc_replay boolean; v_doc_contexto jsonb; v_doc_capacidad jsonb; v_doc_decision jsonb;
 v_doc_persona text; v_doc_registro text; v_doc_llamamiento text; v_doc_recurso text;
 v_doc_abierto record; v_doc_evidencia record;

BEGIN
 v_participacion := vec_bolsa_llamamientos.exigir_portal_candidato_v1(p_candidato_ref, p_bolsa_ref,
   'bolsa.participaciones_propias.responder_llamamiento', p_capacidad, p_decision, p_contexto);
 -- Como B2, la decisión viva se consume antes de resolver el replay: un
 -- reintento no devuelve el recibo sin una autorización nueva y verificada.
 -- Si preparar_respuesta_portal_v1 ya la consumió en esta transacción con
 -- este mismo material, se usa esa (una sola vez); si no, se consume aquí.
 BEGIN
  v_dato := vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['responder'], p_candidato_ref);
 EXCEPTION WHEN insufficient_privilege THEN v_dato := NULL;
 END;
 IF v_dato IS NOT NULL AND left(v_dato, 64) = encode(sha256(p_decision), 'hex') AND octet_length(v_dato) > 64 THEN
  v_decision_ref := substr(v_dato, 65);
  PERFORM set_config('vec_bolsa_llamamientos.marca_consumo', '', true);
 ELSE
  SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
   p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
  IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
   RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
  END IF;
  v_decision_ref := v_consumo.decision_ref;
 END IF;
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  WHERE r.participacion_ref = v_participacion AND r.clave_idempotencia = p_clave;
 v_doc_replay:=FOUND;
 IF p_respuesta='renuncia_justificada' THEN
  IF p_justificante_ref !~ '^ref:[0-9a-f]{64}$' OR
     p_justificante_sha256 !~ '^[0-9a-f]{64}$' THEN
   RAISE EXCEPTION 'B119: referencia documental inválida' USING ERRCODE='42501'; END IF;
  BEGIN
   v_doc_contexto:=pg_catalog.convert_from(p_contexto,'UTF8')::jsonb;
   v_doc_capacidad:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
   v_doc_decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
   RAISE EXCEPTION 'B119: identidad documental ilegible' USING ERRCODE='42501'; END;
  v_doc_persona:=v_doc_contexto->>'persona_ref';
  v_doc_registro:=v_doc_capacidad->>'contexto_ref';
  IF v_doc_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR
     v_doc_registro !~ '^rca_[A-Za-z0-9_-]{22,128}$' OR
     v_doc_decision->>'principal_id' IS DISTINCT FROM v_doc_persona OR
     v_doc_decision#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM v_doc_registro THEN
   RAISE EXCEPTION 'B119: identidad documental divergente' USING ERRCODE='42501'; END IF;
  IF v_doc_replay THEN
   v_doc_llamamiento:=v_previa.llamamiento_ref;
   SELECT * INTO v_doc_evidencia FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j
    WHERE j.respuesta_ref=v_previa.respuesta_ref;
   IF NOT FOUND OR v_doc_evidencia.documento_ref IS DISTINCT FROM p_justificante_ref OR
      v_doc_evidencia.documento_sha256 IS DISTINCT FROM p_justificante_sha256 OR
      v_doc_evidencia.persona_ref IS DISTINCT FROM v_doc_persona OR
      v_doc_evidencia.candidato_ref IS DISTINCT FROM p_candidato_ref OR
      v_doc_evidencia.participacion_ref IS DISTINCT FROM v_participacion OR
      v_doc_evidencia.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR
      v_doc_evidencia.llamamiento_ref IS DISTINCT FROM v_doc_llamamiento THEN
    RAISE EXCEPTION 'B119: replay sin original acreditado' USING ERRCODE='42501'; END IF;
  ELSE
   PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_bolsa_llamamientos:portal:'||v_participacion,0));
   SELECT * INTO v_doc_abierto FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(
    v_participacion,p_respondida_en,p_resultados_efectivos);
   v_doc_llamamiento:=v_doc_abierto.llamamiento_ref;
   IF v_doc_llamamiento IS NULL OR v_doc_abierto.contacto_en IS DISTINCT FROM p_contacto_en THEN
    RAISE EXCEPTION 'B119: llamamiento documental cerrado' USING ERRCODE='VBP04'; END IF;
  END IF;
  v_doc_recurso:=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(v_participacion,v_doc_llamamiento);
  IF v_doc_recurso IS NULL OR
   vec_documentos.validar_justificante_renuncia_bolsa_v1(
    p_justificante_ref,p_justificante_sha256,v_doc_recurso,v_doc_persona,v_doc_registro) IS NOT TRUE THEN
   RAISE EXCEPTION 'B119: justificante no custodiado' USING ERRCODE='42501'; END IF;
  IF v_doc_replay THEN
   IF v_doc_evidencia.recurso_ref IS DISTINCT FROM v_doc_recurso THEN
    RAISE EXCEPTION 'B119: replay de otro llamamiento' USING ERRCODE='42501'; END IF;
  ELSE
   INSERT INTO vec_bolsa_llamamientos.respuesta_justificante_documental_b119(
    respuesta_ref,documento_ref,documento_sha256,recurso_ref,bolsa_ref,participacion_ref,
    llamamiento_ref,candidato_ref,persona_ref,contexto_ref,decision_ref,validada_en)
   VALUES(p_respuesta_ref,p_justificante_ref,p_justificante_sha256,v_doc_recurso,p_bolsa_ref,
    v_participacion,v_doc_llamamiento,p_candidato_ref,v_doc_persona,v_doc_registro,
    v_decision_ref,pg_catalog.clock_timestamp());
  END IF;
 END IF;
 IF v_doc_replay THEN
  IF v_previa.respuesta <> p_respuesta OR v_previa.causa IS DISTINCT FROM p_causa OR v_previa.justificante_ref IS DISTINCT FROM p_justificante_ref
     OR v_previa.justificante_sha256 IS DISTINCT FROM p_justificante_sha256 OR v_previa.candidato_ref <> p_candidato_ref THEN
   RAISE EXCEPTION 'clave idempotente reutilizada con otra respuesta' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true, v_previa.respuesta_ref, v_previa.recibo_ref, v_previa.respondida_en, v_previa.modo;
  RETURN;
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(
  p_respuesta_ref, p_recibo_ref, p_candidato_ref, p_bolsa_ref, v_participacion, p_respuesta, p_causa, p_justificante_ref,
  p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_resultados_efectivos, p_regla_ref, p_clave,
  p_respondida_en, v_decision_ref);
END $function$;

DO $b30_post$
DECLARE interna oid := 'vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)'::regprocedure;
 externa oid := 'vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(interna),'UTF8')),'hex') IS DISTINCT FROM
  '619ff6a40772c36651bde9cb28a705f404da39a6cacf9e635e7e46d2adde20ef'
 OR encode(sha256(convert_to(pg_get_functiondef(externa),'UTF8')),'hex') IS DISTINCT FROM
  '63563f41587997c45028a9d6f4b94ebdda6c4aca6aaeb5068e428da8b489aa6a'
 OR (SELECT proacl::text FROM pg_proc WHERE oid=interna) IS DISTINCT FROM
  '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR (SELECT proacl::text FROM pg_proc WHERE oid=externa) IS DISTINCT FROM
  '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}'
 OR EXISTS (SELECT 1 FROM pg_proc WHERE oid IN (interna,externa)
   AND (proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR NOT prosecdef))
 THEN RAISE EXCEPTION 'B119: postimagen B30 o ACL divergente' USING ERRCODE='55000'; END IF;
END $b30_post$;


-- El mismo registrador exterior de Bolsa conserva su ACL y añade sólo la
-- ruta exacta del justificante propio para registrar denegaciones de frontera.
DO $frontera_pre$
DECLARE f oid := 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '56b1fa45ca6af8cfe92a80aa14b8b71ad33dc4e9cac82aea7f23368b020eb160'
 OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  'vec_bolsa_llamamientos_propietario'::regrole
 OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
 THEN RAISE EXCEPTION 'B119: registrador de frontera HZ37 divergente' USING ERRCODE='55000'; END IF;
END $frontera_pre$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(p_correlacion text, p_motivo text, p_superficie text, p_ruta text, p_actor text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '2s'
AS $function$
DECLARE login pg_catalog.pg_roles%ROWTYPE; grupo pg_catalog.pg_roles%ROWTYPE;
        fachada oid := 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
BEGIN
 SELECT * INTO login FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO grupo FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_portal_externo';
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR login.oid IS NULL OR NOT login.rolcanlogin OR NOT login.rolinherit
    OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole OR login.rolreplication OR login.rolbypassrls
    OR grupo.oid IS NULL OR grupo.rolcanlogin OR NOT grupo.rolinherit
    OR grupo.rolsuper OR grupo.rolcreatedb OR grupo.rolcreaterole OR grupo.rolreplication OR grupo.rolbypassrls
    OR (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid
      AND m.roleid=grupo.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=login.oid OR m.member=grupo.oid)
    OR NOT pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','USAGE')
    OR pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','CREATE')
    OR NOT pg_catalog.has_function_privilege(login.oid,fachada,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=fachada
       AND a.privilege_type='EXECUTE' AND
         (a.grantee=0 OR (a.grantee=grupo.oid AND a.is_grantable)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND n.nspname<>'vec_bolsa_llamamientos'
      AND pg_catalog.has_schema_privilege(login.oid,n.oid,'USAGE,CREATE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND (n.nspowner IN (login.oid,grupo.oid) OR EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=login.oid
          OR (a.grantee=grupo.oid AND (n.nspname<>'vec_bolsa_llamamientos' OR a.privilege_type<>'USAGE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND (c.relowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'
      AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee IN (login.oid,grupo.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND (p.proowner IN (login.oid,grupo.oid)
       OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=login.oid
         OR (a.grantee=grupo.oid AND (p.oid<>fachada OR a.privilege_type<>'EXECUTE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname LIKE 'vec_%' AND (t.typowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND p.oid<>fachada AND pg_catalog.has_function_privilege(login.oid,p.oid,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind IN ('r','p','v','m','f') THEN
        pg_catalog.has_table_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
        OR pg_catalog.has_any_column_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES') ELSE false END)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind='S' THEN
       pg_catalog.has_sequence_privilege(login.oid,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
    OR pg_catalog.pg_is_in_recovery() OR pg_catalog.current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Bolsa: registrador exterior invalido' USING ERRCODE='42501'; END IF;
 IF p_correlacion IS NULL OR (p_correlacion<>'corr_no_disponible' AND p_correlacion !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado')
    OR p_superficie IS DISTINCT FROM 'api.bolsa.candidato.ruta_exacta'
    OR p_ruta IS NULL OR p_ruta NOT IN (
      '/api/vec/bolsa/mi-bolsa','/api/vec/bolsa/mi-bolsa/historial',
      '/api/vec/bolsa/mi-bolsa/solicitudes','/api/vec/bolsa/mi-bolsa/respuestas',
      '/api/vec/bolsa/mi-bolsa/disposiciones','/api/vec/bolsa/mi-bolsa/contacto','/api/vec/bolsa/mi-bolsa/solicitudes-documentales','/api/vec/bolsa/mi-bolsa/preferencias','/api/vec/bolsa/mi-bolsa/justificantes')
    OR (p_motivo='autenticacion_requerida' AND p_actor IS NOT NULL)
    OR (p_motivo='acceso_denegado' AND (p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'))
 THEN RAISE EXCEPTION 'Bolsa: denegacion exterior invalida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.denegacion_frontera_portal_externo
  (correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
 VALUES (p_correlacion,p_motivo,p_superficie,p_ruta,p_actor,
  pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()));
 RETURN true;
END $function$;
DO $frontera_post$
DECLARE f oid := 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '3379b34b44f2412bff1f001c6fd863937daae15e3f65752e67880af09504fd3c'
 OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
 THEN RAISE EXCEPTION 'B119: registrador de frontera postimagen divergente' USING ERRCODE='55000'; END IF;
END $frontera_post$;

COMMIT;
