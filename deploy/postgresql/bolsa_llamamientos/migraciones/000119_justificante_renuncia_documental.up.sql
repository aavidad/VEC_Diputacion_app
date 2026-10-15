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
DO $evolucion_b30$
DECLARE f_interna pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)'::pg_catalog.regprocedure;
 f_externa pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 original text; nuevo text; marca text; insercion text;
BEGIN
 original:=pg_catalog.pg_get_functiondef(f_interna);
 marca:=' INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(';
 insercion:=$i$ IF p_respuesta='renuncia_justificada' AND NOT EXISTS (
  SELECT 1 FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j
  WHERE j.respuesta_ref=p_respuesta_ref AND j.documento_ref=p_justificante_ref
   AND j.documento_sha256=p_justificante_sha256 AND j.bolsa_ref=p_bolsa_ref
   AND j.participacion_ref=p_participacion_ref AND j.llamamiento_ref=v_abierto.llamamiento_ref
   AND j.candidato_ref=p_candidato_ref AND j.decision_ref=p_decision_ref
   AND j.recurso_ref=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(
    p_participacion_ref,v_abierto.llamamiento_ref)) THEN
  RAISE EXCEPTION 'B119: original justificado no acreditado' USING ERRCODE='42501'; END IF;
$i$;
 IF pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca) THEN
  RAISE EXCEPTION 'B119: marca interna B30 divergente' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,marca,insercion||marca);
 EXECUTE nuevo;

 original:=pg_catalog.pg_get_functiondef(f_externa);
 marca:=' v_dato text; v_decision_ref text;';
 insercion:=$d$ v_dato text; v_decision_ref text;
 v_doc_replay boolean; v_doc_contexto jsonb; v_doc_capacidad jsonb; v_doc_decision jsonb;
 v_doc_persona text; v_doc_registro text; v_doc_llamamiento text; v_doc_recurso text;
 v_doc_abierto record; v_doc_evidencia record;
$d$;
 IF pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca) THEN
  RAISE EXCEPTION 'B119: declaración externa B30 divergente' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,marca,insercion);
 marca:=$m$ IF FOUND THEN
  IF v_previa.respuesta$m$;
 insercion:=$i$ v_doc_replay:=FOUND;
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
  IF v_previa.respuesta$i$;
 IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,marca,''))<>pg_catalog.length(marca) THEN
  RAISE EXCEPTION 'B119: replay externo B30 divergente' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(nuevo,marca,insercion);
 EXECUTE nuevo;
END $evolucion_b30$;

-- El mismo registrador exterior de Bolsa conserva su ACL y añade sólo la
-- ruta exacta del justificante propio para registrar denegaciones de frontera.
DO $frontera_b119$
DECLARE f pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
 original text; nuevo text;
 marca text := $m$'/api/vec/bolsa/mi-bolsa/preferencias')$m$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
     '56b1fa45ca6af8cfe92a80aa14b8b71ad33dc4e9cac82aea7f23368b020eb160'
    OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    OR (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
 THEN RAISE EXCEPTION 'B119: registrador de frontera HZ37 divergente' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,marca,
  $m$'/api/vec/bolsa/mi-bolsa/preferencias','/api/vec/bolsa/mi-bolsa/justificantes')$m$);
 EXECUTE nuevo;
 IF (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
    OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(f),'/api/vec/bolsa/mi-bolsa/justificantes')=0
 THEN RAISE EXCEPTION 'B119: registrador de frontera no conservado' USING ERRCODE='55000'; END IF;
END $frontera_b119$;

COMMIT;
