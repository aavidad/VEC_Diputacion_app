\set ON_ERROR_STOP on
-- P47: Personal resuelve la pertenencia de la persona a la organización.
-- Incluye relaciones de ejercicio como hechos trazables, sin afirmar eficacia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000047',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
 OR to_regclass('vec_personal.relacion_alta_ejercicio') IS NULL
 OR to_regprocedure('vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(text,text,text,integer)') IS NOT NULL
 THEN RAISE EXCEPTION 'P47: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
CREATE INDEX relacion_servicio_auditoria_rrhh_org_persona_v1
 ON vec_personal.relacion_servicio_historia(organismo_ref,persona_ref,conocido_desde DESC,relacion_ref,revision);
CREATE INDEX relacion_alta_auditoria_rrhh_org_persona_v1
 ON vec_personal.relacion_alta_ejercicio(organizacion_ref,persona_sintetica_ref,registrada_en DESC,relacion_ref);
CREATE FUNCTION vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(
 p_organizacion_ref text,p_busqueda text,p_despues_ref text,p_limite integer)
RETURNS TABLE(referencia text,etiqueta text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_busqueda IS NULL OR octet_length(p_busqueda)>80 OR p_busqueda ~ '[[:cntrl:]]'
 OR p_despues_ref IS NULL OR octet_length(p_despues_ref)>512 OR p_despues_ref ~ '[[:cntrl:]]'
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 51
 THEN RAISE EXCEPTION 'P47: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY WITH sujetos AS (
  SELECT r.persona_ref AS ref FROM vec_personal.relacion_servicio_historia r
   WHERE r.organismo_ref=p_organizacion_ref
  UNION SELECT e.persona_sintetica_ref FROM vec_personal.relacion_alta_ejercicio e
   WHERE e.organizacion_ref=p_organizacion_ref
 ) SELECT s.ref,s.ref FROM sujetos s
 WHERE s.ref COLLATE "C">p_despues_ref COLLATE "C"
 AND (p_busqueda='' OR strpos(lower(s.ref),lower(p_busqueda))>0)
 ORDER BY s.ref COLLATE "C" LIMIT p_limite;
END $f$;
CREATE FUNCTION vec_personal.recursos_persona_auditoria_rrhh_v1(
 p_organizacion_ref text,p_persona_ref text)
RETURNS TABLE(recurso_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_persona_ref IS NULL OR octet_length(p_persona_ref)>512
 THEN RAISE EXCEPTION 'P47: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT p_persona_ref WHERE
  EXISTS(SELECT 1 FROM vec_personal.relacion_servicio_historia h
   WHERE h.organismo_ref=p_organizacion_ref AND h.persona_ref=p_persona_ref)
  OR EXISTS(SELECT 1 FROM vec_personal.relacion_alta_ejercicio e
   WHERE e.organizacion_ref=p_organizacion_ref AND e.persona_sintetica_ref=p_persona_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(text,text,text,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.recursos_persona_auditoria_rrhh_v1(text,text) FROM PUBLIC;
CREATE FUNCTION vec_personal.registros_persona_auditoria_rrhh_v1(
 p_organizacion_ref text,p_persona_ref text,p_actor_filtro text,
 p_desde timestamptz,p_hasta timestamptz,p_antes_en timestamptz,p_antes_id text,p_limite integer)
RETURNS TABLE(id text,fuente text,modulo_id text,accion text,actor_ref text,
 ocurrido_en timestamptz,resultado text,expediente_ref text,recibo_ref text,
 antes_sha256 text,despues_sha256 text,motivo text,antes jsonb,despues jsonb,
 datos_disponibles boolean,perfil_ref text,finalidad_ref text,canal text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_persona_ref IS NULL OR p_actor_filtro IS NULL
 OR p_desde IS NULL OR p_hasta IS NULL OR p_desde>=p_hasta OR p_hasta-p_desde>interval '31 days'
 OR p_antes_id IS NULL OR p_limite NOT BETWEEN 1 AND 101
 THEN RAISE EXCEPTION 'P47: proyección inválida' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_personal.relacion_servicio_historia h
  WHERE h.organismo_ref=p_organizacion_ref AND h.persona_ref=p_persona_ref)
 AND NOT EXISTS(SELECT 1 FROM vec_personal.relacion_alta_ejercicio e
  WHERE e.organizacion_ref=p_organizacion_ref AND e.persona_sintetica_ref=p_persona_ref)
 THEN RAISE EXCEPTION 'P47: persona ajena' USING ERRCODE='42501'; END IF;
 RETURN QUERY WITH hechos AS (
  SELECT 'persona:r:'||h.relacion_ref||':v'||h.revision::text AS rid,
   h.conocido_desde AS fecha,'personal.relacion.revision'::text AS acc,
   ''::text AS actor,COALESCE(h.estado,'registrado') AS res,
   COALESCE(h.acto_ref,'') AS recibo,COALESCE(h.fuente_huella_sha256,'') AS sha,
   'relacion_servicio'::text AS motivo,''::text AS perfil
  FROM vec_personal.relacion_servicio_historia h
  WHERE h.organismo_ref=p_organizacion_ref AND h.persona_ref=p_persona_ref
  UNION ALL
  SELECT 'persona:e:'||e.relacion_ref,e.registrada_en,
   'personal.relacion.alta_ejercicio'::text,COALESCE(a.actor_ref,''),
   CASE WHEN e.eficacia_administrativa THEN 'registrado' ELSE 'ejercicio' END,
   COALESCE(a.recibo_ref,''),COALESCE(a.material_sha256,''),
   'alta_ejercicio'::text,COALESCE(a.perfil_ref,'')
  FROM vec_personal.relacion_alta_ejercicio e
  LEFT JOIN vec_personal.registro_alta_ejercicio a
   ON a.relacion_ref=e.relacion_ref AND a.organizacion_ref=e.organizacion_ref
  WHERE e.organizacion_ref=p_organizacion_ref AND e.persona_sintetica_ref=p_persona_ref
 ) SELECT h.rid,'persona'::text,'personal'::text,h.acc,h.actor,h.fecha,h.res,
  p_persona_ref,h.recibo,''::text,h.sha,h.motivo,'{}'::jsonb,'{}'::jsonb,false,
  h.perfil,''::text,'interna_corporativa'::text
 FROM hechos h
 WHERE h.fecha>=p_desde AND h.fecha<p_hasta
  AND (p_actor_filtro='' OR h.actor=p_actor_filtro)
  AND (p_antes_en IS NULL OR (h.fecha,h.rid COLLATE "C")<(p_antes_en,p_antes_id COLLATE "C"))
 ORDER BY h.fecha DESC,h.rid COLLATE "C" DESC LIMIT p_limite;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.registros_persona_auditoria_rrhh_v1(
 text,text,text,timestamptz,timestamptz,timestamptz,text,integer) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_personal TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(text,text,text,integer),
 vec_personal.recursos_persona_auditoria_rrhh_v1(text,text),
 vec_personal.registros_persona_auditoria_rrhh_v1(text,text,text,timestamptz,timestamptz,timestamptz,text,integer)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
