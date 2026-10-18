\set ON_ERROR_STOP on
-- CT208: sólo el propietario de CT resuelve expedientes de una organización.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000208',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_contratacion_temporal.expediente_alta') IS NULL
 OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(text,text,text,integer)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT208: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE INDEX expediente_alta_auditoria_rrhh_org_ref_v1
 ON vec_contratacion_temporal.expediente_alta(organizacion_ref,expediente_ref);
CREATE FUNCTION vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(
 p_organizacion_ref text,p_busqueda text,p_despues_ref text,p_limite integer)
RETURNS TABLE(referencia text,etiqueta text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_busqueda IS NULL OR octet_length(p_busqueda)>80 OR p_busqueda ~ '[[:cntrl:]]'
 OR p_despues_ref IS NULL OR octet_length(p_despues_ref)>160 OR p_despues_ref ~ '[[:cntrl:]]'
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 51
 THEN RAISE EXCEPTION 'CT208: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.expediente_ref,a.numero_visible FROM vec_contratacion_temporal.expediente_alta a
 JOIN vec_contratacion_temporal.expediente_integral_actual v USING(expediente_ref)
 WHERE a.organizacion_ref=p_organizacion_ref AND a.expediente_ref COLLATE "C">p_despues_ref COLLATE "C"
 AND (p_busqueda='' OR strpos(lower(a.expediente_ref),lower(p_busqueda))>0
   OR strpos(lower(a.numero_visible),lower(p_busqueda))>0)
 ORDER BY a.expediente_ref COLLATE "C" LIMIT p_limite;
END $f$;
CREATE FUNCTION vec_contratacion_temporal.recursos_auditoria_rrhh_v1(
 p_organizacion_ref text,p_expediente_ref text)
RETURNS TABLE(recurso_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_expediente_ref IS NULL OR octet_length(p_expediente_ref)>160
 THEN RAISE EXCEPTION 'CT208: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.expediente_ref FROM vec_contratacion_temporal.expediente_alta a
 JOIN vec_contratacion_temporal.expediente_integral_actual v USING(expediente_ref)
 WHERE a.organizacion_ref=p_organizacion_ref AND a.expediente_ref=p_expediente_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(text,text,text,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.recursos_auditoria_rrhh_v1(text,text) FROM PUBLIC;
CREATE FUNCTION vec_contratacion_temporal.registros_auditoria_rrhh_v1(
 p_organizacion_ref text,p_expediente_ref text,p_actor_filtro text,
 p_desde timestamptz,p_hasta timestamptz,p_antes_en timestamptz,p_antes_id text,p_limite integer)
RETURNS TABLE(id text,fuente text,modulo_id text,accion text,actor_ref text,
 ocurrido_en timestamptz,resultado text,expediente_ref text,recibo_ref text,
 antes_sha256 text,despues_sha256 text,motivo text,antes jsonb,despues jsonb,
 datos_disponibles boolean,perfil_ref text,finalidad_ref text,canal text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_organizacion_ref IS NULL OR p_expediente_ref IS NULL OR p_actor_filtro IS NULL
 OR p_desde IS NULL OR p_hasta IS NULL OR p_desde>=p_hasta OR p_hasta-p_desde>interval '31 days'
 OR p_antes_id IS NULL OR p_limite NOT BETWEEN 1 AND 101
 THEN RAISE EXCEPTION 'CT208: proyección inválida' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta a
  WHERE a.organizacion_ref=p_organizacion_ref AND a.expediente_ref=p_expediente_ref)
 THEN RAISE EXCEPTION 'CT208: expediente ajeno' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'ct:v:'||lpad(v.version::text,16,'0'),'ct'::text,
  'contratacion_temporal'::text,COALESCE(v.origen_version,'version_expediente'),
  COALESCE(CASE WHEN v.version=1 THEN al.actor_ref ELSE a.actuacion_json->>'actor_ref' END,''),
  v.registrada_en,COALESCE(v.estado,'registrado'),v.expediente_ref,
  COALESCE(CASE WHEN v.version=1 THEN al.recibo_ref ELSE a.recibo_ref END,''),
  COALESCE(prev.agregado_json_huella_sha256,''),v.agregado_json_huella_sha256,
  COALESCE(a.actuacion_json->>'motivo_clave',a.actuacion_json->>'causa_clave',''),
  CASE WHEN prev.version IS NULL THEN '{}'::jsonb ELSE
   jsonb_build_object('fase',COALESCE(prev.fase_clave,''),'estado',COALESCE(prev.estado,'')) END,
  jsonb_build_object('fase',COALESCE(v.fase_clave,''),'estado',COALESCE(v.estado,'')),
  true,''::text,''::text,'interna_corporativa'::text
 FROM vec_contratacion_temporal.expediente_alta alta
 JOIN vec_contratacion_temporal.expediente_version_integral v
  ON v.expediente_ref=alta.expediente_ref
 LEFT JOIN vec_contratacion_temporal.expediente_version_integral prev
  ON prev.expediente_ref=v.expediente_ref AND prev.version=v.version-1
 LEFT JOIN vec_contratacion_temporal.actuacion_expediente_integral a
  ON a.expediente_ref=v.expediente_ref AND a.version_expediente=v.version
 LEFT JOIN vec_contratacion_temporal.actuacion_alta al
  ON al.expediente_ref=v.expediente_ref AND v.version=1
 WHERE alta.organizacion_ref=p_organizacion_ref AND alta.expediente_ref=p_expediente_ref
  AND v.registrada_en>=p_desde AND v.registrada_en<p_hasta
  AND (p_actor_filtro='' OR (CASE WHEN v.version=1 THEN al.actor_ref ELSE a.actuacion_json->>'actor_ref' END)=p_actor_filtro)
  AND (p_antes_en IS NULL OR (v.registrada_en,('ct:v:'||lpad(v.version::text,16,'0')) COLLATE "C")
   < (p_antes_en,p_antes_id COLLATE "C"))
 ORDER BY v.registrada_en DESC,('ct:v:'||lpad(v.version::text,16,'0')) COLLATE "C" DESC
 LIMIT p_limite;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registros_auditoria_rrhh_v1(
 text,text,text,timestamptz,timestamptz,timestamptz,text,integer) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(text,text,text,integer),
 vec_contratacion_temporal.recursos_auditoria_rrhh_v1(text,text),
 vec_contratacion_temporal.registros_auditoria_rrhh_v1(text,text,text,timestamptz,timestamptz,timestamptz,text,integer)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
