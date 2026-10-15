\set ON_ERROR_STOP on
-- AUT99: el catálogo documental de la renuncia de Bolsa se limita a los dos
-- actos propios. No cambia concesiones ni sesiones; AD281 despacha a la pareja nueva.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000099',0));
DO $pre$
DECLARE f record; actual pg_catalog.pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regrole('vec_autorizacion_propietario') IS NULL
    OR to_regrole('vec_autorizacion_motivos_externos') IS NULL
    OR to_regrole('vec_autorizacion_registro_externo') IS NULL
    OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 THEN RAISE EXCEPTION 'AUT99: preimagen estructural incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz)','3dbdb5273e89873cee995c12ff84a31fb174caf5fc1e0ef834a662b1eab7c177','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_motivos_externos=X/vec_autorizacion_propietario}','s'),
  ('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)','903570e631b0aea0197ec722827bc234d0754c3ad5e383bbdd786835925eb2b8','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_registro_externo=X/vec_autorizacion_propietario}','v'),
  ('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','90abe20e791c6c0ba0f8883d6014cd4403c54400bfea944cc4ec6aaf66c03100','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}','v')
 ) AS esperado(firma,huella,acl,volatilidad)
 LOOP
  IF to_regprocedure(f.firma) IS NULL THEN
   RAISE EXCEPTION 'AUT99: función previa ausente %',f.firma USING ERRCODE='55000';
  END IF;
  SELECT * INTO STRICT actual FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF actual.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
     OR actual.prokind IS DISTINCT FROM 'f'
     OR actual.provolatile IS DISTINCT FROM f.volatilidad::"char"
     OR actual.prosecdef IS NOT TRUE
     OR actual.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
     OR actual.proacl::text IS DISTINCT FROM f.acl
     OR encode(sha256(convert_to(pg_get_functiondef(actual.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'AUT99: definición o ACL previa divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_motivo_candidato_externo_v1(p_catalogo_id text, p_catalogo_version integer, p_huella text, p_entrada text, p_instante timestamp with time zone)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_motivos_autorizacion_desarrollo','vec_autorizacion_motivos_externos') IS NOT TRUE
    OR p_catalogo_id NOT IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo','motivos_inscripcion_bolsa_desarrollo','motivos_alegaciones_bolsa','motivos_autorizacion_justificante_renuncia_bolsa')
 THEN RAISE EXCEPTION 'motivo externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_historico(p_catalogo_id,p_catalogo_version,p_huella,p_entrada,p_instante);
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_candidato_externo_v3(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; v jsonb; m jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m := pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END;
 v := d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR NOT (m->'referencia'->>'catalogo_id' IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
      OR (m->'referencia'->>'catalogo_id'='motivos_inscripcion_bolsa_desarrollo' AND d->>'accion'='bolsa.inscripcion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_alegaciones_bolsa' AND d->>'accion'='bolsa.inscripcion.alegacion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_autorizacion_justificante_renuncia_bolsa'
        AND d->>'accion' IN ('bolsa.justificante_renuncia.subir_propio','bolsa.justificante_renuncia.descargar_propio')
        AND d->>'modulo_id'='bolsa'
        AND d->>'tipo_recurso'='justificante_renuncia'
        AND d->>'finalidad'='custodiar_justificante_renuncia_bolsa'))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=d->>'version_rol_ref'
        AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(p_decision,p_motivo,p_persona_version,p_perfil_version);
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone, revalidada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; m jsonb; v jsonb; registro record; viva timestamptz;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END;
 v:=d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR NOT (m->'referencia'->>'catalogo_id' IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
      OR (m->'referencia'->>'catalogo_id'='motivos_inscripcion_bolsa_desarrollo' AND d->>'accion'='bolsa.inscripcion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_alegaciones_bolsa' AND d->>'accion'='bolsa.inscripcion.alegacion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_autorizacion_justificante_renuncia_bolsa'
        AND d->>'accion' IN ('bolsa.justificante_renuncia.subir_propio','bolsa.justificante_renuncia.descargar_propio')
        AND d->>'modulo_id'='bolsa'
        AND d->>'tipo_recurso'='justificante_renuncia'
        AND d->>'finalidad'='custodiar_justificante_renuncia_bolsa'))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
       WHERE r.version_rol_ref=d->>'version_rol_ref'
         AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND OR registro.concedida IS NOT TRUE THEN RETURN; END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN; END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,registro.registrada_en,viva;
END $function$;


-- Predicado privado: el principal sigue siendo la persona del vínculo V2.
-- Documentos usa la candidatura vinculada como ámbito de los dos actos propios.
CREATE FUNCTION vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(d jsonb,m jsonb)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_user='vec_autorizacion_propietario'
 AND vec_autorizacion.decision_contexto_actor_v3_valida(d) IS TRUE
 AND d->>'modulo_id'='bolsa' AND d->>'tipo_recurso'='justificante_renuncia'
 AND d->>'finalidad'='custodiar_justificante_renuncia_bolsa'
 AND d->>'accion' IN ('bolsa.justificante_renuncia.subir_propio',
                     'bolsa.justificante_renuncia.descargar_propio')
 AND d->'campos_permitidos'=CASE WHEN d->>'accion'='bolsa.justificante_renuncia.subir_propio'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END
 AND d->'obligaciones'='[]'::jsonb
 AND d->>'principal_id'=d#>>'{vinculo_autenticacion_actor,principal_id}'
 AND d->>'perfil_activo_ref'=d#>>'{vinculo_autenticacion_actor,perfil_activo_ref}'
 AND d#>>'{vinculo_autenticacion_actor,superficie}'='externa_personal'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}'='false'
 AND d#>>'{vinculo_autenticacion_actor,garantia_observada}'='alto'
 AND m#>>'{referencia,catalogo_id}'='motivos_autorizacion_justificante_renuncia_bolsa'
 AND EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
  WHERE r.version_rol_ref=d->>'version_rol_ref'
   AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo',
                    'candidato_bolsa_portal_historial_propio_desarrollo'))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,
 registrada_en timestamptz,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE d jsonb;m jsonb;r record;viva timestamptz;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1(
   'vec_externo_documentos_bolsa_datos','vec_documentos_bolsa_subida_datos') IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR p_motivo IS NULL
 OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'AUT99: registrador documental denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=convert_from(p_decision,'UTF8')::jsonb;
  m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN
  RAISE EXCEPTION 'AUT99: decisión ilegible' USING ERRCODE='42501'; END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT99: decisión documental divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO r FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND OR r.concedida IS NOT TRUE THEN RETURN; END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN; END IF;
 RETURN QUERY SELECT r.concedida,r.codigo,r.decision_huella_sha256,r.registrada_en,viva;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric) TO vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE d jsonb;m jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1(
   'vec_externo_documentos_bolsa_datos','vec_documentos_bolsa_subida_datos') IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR p_motivo IS NULL
 OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'AUT99: revalidación documental denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=convert_from(p_decision,'UTF8')::jsonb;
  m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN
  RAISE EXCEPTION 'AUT99: decisión ilegible' USING ERRCODE='42501'; END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT99: decisión documental divergente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric) TO vec_autorizacion_atestada_v3_propietario;

RESET ROLE;
DO $post$
DECLARE f record; actual pg_catalog.pg_proc%ROWTYPE;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz)','46fcbb5aa9bb611f84ec4a45ba27fca6e21cc2632ba9904b7a8899695ce5bd32','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_motivos_externos=X/vec_autorizacion_propietario}','s'),
  ('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)','aaf798aa5d9e532fc5d4ea644f0e6d888ae97fd5699d38b377e0ce5d1f0b41ca','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_registro_externo=X/vec_autorizacion_propietario}','v'),
  ('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','e273354738a2382c534c0dba6e5554c2bb6f421a215152bdf767099341f134e4','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}','v')
 ) AS esperado(firma,huella,acl,volatilidad)
 LOOP
  SELECT * INTO STRICT actual FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF actual.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
     OR actual.prokind IS DISTINCT FROM 'f'
     OR actual.provolatile IS DISTINCT FROM f.volatilidad::"char"
     OR actual.prosecdef IS NOT TRUE
     OR actual.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
     OR actual.proacl::text IS DISTINCT FROM f.acl
     OR encode(sha256(convert_to(pg_get_functiondef(actual.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'AUT99: definición o ACL final divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
DO $nominal$
DECLARE f record;p pg_proc%ROWTYPE;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(jsonb,jsonb)',
   'a5793590949922abc706e9e424b4dc72cd7f669b5f54020cef0145344965f6bd',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario}','s'),
  ('vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric)',
   '5dc7ffeee41a6adabe9626c930b6f82e64cc6b3e7a0d0d3e792d8ec421aaa4fd',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}','v'),
  ('vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric)',
   '8e540ea7af803a6609b7072a19c1aaf8d39fa0b8ce0e819c97c754e94f46926a',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}','v')
 ) x(firma,huella,acl,volatilidad) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
  OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM f.volatilidad::"char"
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR p.proacl::text IS DISTINCT FROM f.acl
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'AUT99: pareja nominal divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $nominal$;
COMMIT;
