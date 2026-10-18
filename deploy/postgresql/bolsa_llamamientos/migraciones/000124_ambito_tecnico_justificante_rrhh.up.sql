\set ON_ERROR_STOP on
-- B124: la descarga documental RRHH usa el ámbito técnico vigente de Bolsa.
-- Metadatos previos sin contenido; lectura B103/AD237 con permiso y auditoría;
-- recotejo documental dentro de las transacciones de preparación y descarga.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000124',0));
DO $pre$
DECLARE f record;p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regprocedure('vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb)') IS NULL OR
  to_regprocedure('vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()') IS NULL OR
  to_regclass('vec_bolsa_llamamientos.inscripcion_incorporacion') IS NULL OR
  to_regclass('vec_bolsa_llamamientos.respuesta_justificante_documental_b119') IS NULL OR
  to_regrole('vec_interno_documentos_bolsa_rrhh') IS NULL OR
  to_regprocedure('vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(text,text,text)') IS NOT NULL OR
  to_regprocedure('vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)') IS NOT NULL OR
  to_regprocedure('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)') IS NOT NULL OR
  to_regprocedure('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B124: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb)',
   '8a87ced64773cfe57d10ef3204a518b2a7c025f92bbe5e1ef17185d71f0a419d'),
  ('vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text)',
   'cd0caee66432396689fb86dab1e93d28f6cfa71ec4a970d0c5f0afa2aeced5a7'),
  ('vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text,text)',
   'f3a82bf5a40e9e4b87056f51a8cc35ccf592f92b1cc79331f7a8aa08a3d7f1b0'),
  ('vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()',
   '5d84eeda8d2aa914e2c36626d96759d3052777d0f0314faa7ad43143e0449bda')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'B124: dependencia divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_documento_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC' SET lock_timeout='2s' AS $f$
DECLARE v record; conjunto jsonb; unidad text; ambito text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR
  p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256 OR
  p_participacion_ref IS NULL OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 256 OR
  (p_documento_ref IS NOT NULL AND p_documento_ref !~ '^ref:[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'B124: relación no disponible' USING ERRCODE='42501'; END IF;
 SELECT j.llamamiento_ref,j.persona_ref,j.candidato_ref,j.documento_ref,
  i.solicitud_ref,i.persona_ref AS persona_incorporada,i.candidato_ref AS candidato_incorporado,
  i.bolsa_ref AS bolsa_incorporada,s.persona_ref AS persona_solicitud,
  s.unidad_ref,s.ambito_ref
 INTO v
 FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 j
 JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento r ON r.respuesta_ref=j.respuesta_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.llamamiento_ref=j.llamamiento_ref
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=j.participacion_ref
 LEFT JOIN vec_bolsa_llamamientos.inscripcion_incorporacion i ON i.participacion_ref=j.participacion_ref
 LEFT JOIN vec_bolsa_llamamientos.solicitud_inscripcion s ON s.solicitud_ref=i.solicitud_ref
 WHERE j.bolsa_ref=p_bolsa_ref AND j.participacion_ref=p_participacion_ref AND
  (p_documento_ref IS NULL OR j.documento_ref=p_documento_ref) AND
  r.bolsa_ref=j.bolsa_ref AND r.participacion_ref=j.participacion_ref AND
  r.llamamiento_ref=j.llamamiento_ref AND r.candidato_ref=j.candidato_ref AND
  r.respuesta='renuncia_justificada' AND r.modo='firme' AND
  r.justificante_ref=j.documento_ref AND r.justificante_sha256=j.documento_sha256 AND
  l.bolsa_ref=j.bolsa_ref AND l.participaciones ? j.participacion_ref AND
  vc.candidato_ref=j.candidato_ref AND
  j.recurso_ref=vec_bolsa_llamamientos.recurso_justificante_renuncia_v1(j.participacion_ref,j.llamamiento_ref)
 ORDER BY j.validada_en DESC,j.respuesta_ref DESC LIMIT 1
 FOR SHARE OF j,r,l,vc;
 IF NOT FOUND THEN RAISE EXCEPTION 'B124: relación no disponible' USING ERRCODE='42501'; END IF;
 IF v.solicitud_ref IS NOT NULL THEN
  IF v.persona_incorporada IS DISTINCT FROM v.persona_ref OR
   v.candidato_incorporado IS DISTINCT FROM v.candidato_ref OR
   v.bolsa_incorporada IS DISTINCT FROM p_bolsa_ref OR
   v.persona_solicitud IS DISTINCT FROM v.persona_ref
  THEN RAISE EXCEPTION 'B124: incorporación divergente' USING ERRCODE='42501'; END IF;
  unidad:=v.unidad_ref; ambito:=v.ambito_ref;
 ELSE
  -- Bolsa histórica: el mismo conjunto gobernado que usa B114.
  conjunto:=vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
  unidad:=conjunto->>'unidad_ref'; ambito:=conjunto->>'ambito_ref';
 END IF;
 IF coalesce(unidad ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$',false) IS NOT TRUE OR
  coalesce(ambito ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'B124: ámbito no disponible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('bolsa_ref',p_bolsa_ref,'participacion_ref',p_participacion_ref,
  'llamamiento_ref',v.llamamiento_ref,'unidad_ref',unidad,'ambito_ref',ambito);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
  current_setting('role')<>'none' OR
  NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL AND
    NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND
    m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole AND m.inherit_option AND
    NOT m.set_option AND NOT m.admin_option AND
    (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1 AND
    NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=m.roleid)) OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B124: metadatos RRHH denegados' USING ERRCODE='42501'; END IF;
 RETURN vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,NULL);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)
 TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(
 p_bolsa_ref text,p_participacion_ref text,p_contexto_canonico bytea,
 p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC' SET lock_timeout='2s' AS $f$
DECLARE previo jsonb; posterior jsonb; lectura jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
  current_setting('role')<>'none' OR
  NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL AND
    NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND
    m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole AND m.inherit_option AND
    NOT m.set_option AND NOT m.admin_option AND
    (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1 AND
    NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=m.roleid)) OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B124: lectura RRHH denegada' USING ERRCODE='42501'; END IF;
 previo:=vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,NULL);
 IF jsonb_typeof(p_orden) IS DISTINCT FROM 'object' OR
  p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.respuestas_llamamiento.consultar' OR
  p_orden->>'recurso_ref' IS DISTINCT FROM previo->>'llamamiento_ref' OR
  p_orden->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref OR
  p_orden->>'unidad_ref' IS DISTINCT FROM previo->>'unidad_ref' OR
  p_orden->>'ambito_ref' IS DISTINCT FROM previo->>'ambito_ref'
 THEN RAISE EXCEPTION 'B124: captura RRHH divergente' USING ERRCODE='42501'; END IF;
 lectura:=vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(
  p_bolsa_ref,previo->>'llamamiento_ref',p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF lectura->>'esquema' IS DISTINCT FROM 'vec.bolsa.rrhh.respuestas-portal.v1' OR
  lectura->>'llamamiento_ref' IS DISTINCT FROM previo->>'llamamiento_ref' OR
  lectura->>'correlacion_ref' IS DISTINCT FROM p_orden->>'correlacion_ref' OR
  nullif(lectura->>'auditoria_ref','') IS NULL OR
  NOT EXISTS(SELECT 1 FROM jsonb_array_elements(lectura->'respuestas') x
   WHERE x->>'participacion_ref'=p_participacion_ref AND
    x->>'respuesta'='renuncia_justificada')
 THEN RAISE EXCEPTION 'B124: lectura RRHH sin acuse o respuesta' USING ERRCODE='42501'; END IF;
 posterior:=vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,NULL);
 IF posterior IS DISTINCT FROM previo
 THEN RAISE EXCEPTION 'B124: ámbito RRHH cambiado' USING ERRCODE='42501'; END IF;
 RETURN posterior||jsonb_build_object('auditoria_ref',lectura->>'auditoria_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(
 p_bolsa_ref text,p_participacion_ref text,p_documento_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR
  session_user<>'vec_interno_documentos_bolsa_rrhh' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B124: cotejo documental denegado' USING ERRCODE='42501'; END IF;
 RETURN vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,p_documento_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)
 TO vec_documentos_propietario;
RESET ROLE;
DO $post$
DECLARE f record;p pg_proc%ROWTYPE;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.resolver_base_justificante_rrhh_v1(text,text,text)',
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}',
   'c7b49bb613df49abe0c1401d5c9b067ef29b5b2702944510846f2133b8e6c2fc'),
  ('vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(text,text)',
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}',
   '84bed8d59c116a9e6d310345789e42d53e9d164babc2ed1d50b51aea643ccbf5'),
  ('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(text,text,bytea,bytea,jsonb)',
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}',
   '20370b0be2519606f9ede575cbf783df4c0f7bb982651421648a0063205cdfa2'),
  ('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)',
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_documentos_propietario=X/vec_bolsa_llamamientos_propietario}',
   'c31a6887e43a231a8c5c9a081ed7784ea3532c9d0035aa8d75ee7ca71118687a')
 ) v(firma,acl,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
   p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
    'TimeZone=UTC','lock_timeout=2s']::text[] OR p.proacl::text IS DISTINCT FROM f.acl OR
   encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'B124: postimagen divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
