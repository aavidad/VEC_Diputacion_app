\set ON_ERROR_STOP on
-- B109: la concesión RRHH con unidad y ámbito permite consultar toda Bolsa
-- constituida. La frontera nominal se cierra en AD251, en la misma transacción.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000109',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer)') IS NOT NULL
 THEN RAISE EXCEPTION 'B109: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE INDEX situacion_participacion_auditoria_rrhh_fecha_v1
 ON vec_bolsa_llamamientos.situacion_participacion(participacion_ref,registrada_en DESC,recibo_ref);
CREATE FUNCTION vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(
 p_unidad_ref text,p_ambito_ref text,p_busqueda text,p_despues_ref text,p_limite integer)
RETURNS TABLE(referencia text,etiqueta text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_unidad_ref IS NULL OR p_unidad_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_ambito_ref IS NULL OR p_ambito_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_busqueda IS NULL OR octet_length(p_busqueda)>80 OR p_busqueda ~ '[[:cntrl:]]'
 OR p_despues_ref IS NULL OR octet_length(p_despues_ref)>512 OR p_despues_ref ~ '[[:cntrl:]]'
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 51
 THEN RAISE EXCEPTION 'B109: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT b.bolsa_ref,
  left(COALESCE(NULLIF(convert_from(b.bolsa_canonica,'UTF8')::jsonb->>'categoria_nombre',''),
   b.categoria_ref,b.bolsa_ref),160)
 FROM (SELECT DISTINCT ON (x.bolsa_ref COLLATE "C") x.bolsa_ref,x.bolsa_canonica,x.categoria_ref
   FROM vec_bolsa_llamamientos.bolsa_constituida x
   ORDER BY x.bolsa_ref COLLATE "C",x.version DESC) b
 WHERE b.bolsa_ref COLLATE "C">p_despues_ref COLLATE "C"
 AND (p_busqueda='' OR strpos(lower(b.bolsa_ref),lower(p_busqueda))>0
  OR strpos(lower(COALESCE(b.categoria_ref,'')),lower(p_busqueda))>0)
 ORDER BY b.bolsa_ref COLLATE "C" LIMIT p_limite;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.recursos_auditoria_rrhh_v1(
 p_unidad_ref text,p_ambito_ref text,p_bolsa_ref text)
RETURNS TABLE(recurso_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_unidad_ref IS NULL OR p_unidad_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_ambito_ref IS NULL OR p_ambito_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref)>512
 THEN RAISE EXCEPTION 'B109: selector inválido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT b.bolsa_ref FROM vec_bolsa_llamamientos.bolsa_constituida b
 WHERE b.bolsa_ref=p_bolsa_ref LIMIT 1;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recursos_auditoria_rrhh_v1(text,text,text) FROM PUBLIC;
CREATE FUNCTION vec_bolsa_llamamientos.registros_auditoria_rrhh_v1(
 p_unidad_ref text,p_ambito_ref text,p_bolsa_ref text,p_actor_filtro text,
 p_desde timestamptz,p_hasta timestamptz,p_antes_en timestamptz,p_antes_id text,p_limite integer)
RETURNS TABLE(id text,fuente text,modulo_id text,accion text,actor_ref text,
 ocurrido_en timestamptz,resultado text,expediente_ref text,recibo_ref text,
 antes_sha256 text,despues_sha256 text,motivo text,antes jsonb,despues jsonb,
 datos_disponibles boolean,perfil_ref text,finalidad_ref text,canal text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_unidad_ref IS NULL OR p_unidad_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_ambito_ref IS NULL OR p_ambito_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_bolsa_ref IS NULL OR p_actor_filtro IS NULL
 OR p_desde IS NULL OR p_hasta IS NULL OR p_desde>=p_hasta OR p_hasta-p_desde>interval '31 days'
 OR p_antes_id IS NULL OR p_limite NOT BETWEEN 1 AND 101
 THEN RAISE EXCEPTION 'B109: proyección inválida' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b
  WHERE b.bolsa_ref=p_bolsa_ref)
 THEN RAISE EXCEPTION 'B109: bolsa ausente' USING ERRCODE='42501'; END IF;
 RETURN QUERY WITH participaciones AS MATERIALIZED (
  SELECT DISTINCT e.participacion_ref
  FROM vec_bolsa_llamamientos.constitucion c
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea
  WHERE c.bolsa_ref=p_bolsa_ref
 ), hechos AS (
  SELECT 'bolsa:c:'||c.acta_ref AS rid,c.registrada_en AS fecha,
   'bolsa.constitucion'::text AS acc,COALESCE(c.actor_ref,'') AS actor,
   'confirmado'::text AS resultado,c.acta_ref AS recibo,'constitucion_bolsa'::text AS motivo
  FROM vec_bolsa_llamamientos.constitucion c WHERE c.bolsa_ref=p_bolsa_ref
  UNION ALL
  SELECT 'bolsa:s:'||s.recibo_ref,s.registrada_en,
   COALESCE(o.operacion,'bolsa.situacion_participacion'),COALESCE(s.actor,''),
   'confirmado'::text,s.recibo_ref,'situacion_participacion'::text
  FROM participaciones p JOIN vec_bolsa_llamamientos.situacion_participacion s
   ON s.participacion_ref=p.participacion_ref
  LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
   ON o.participacion_ref=s.participacion_ref AND o.desde=s.desde
  WHERE NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion t
   WHERE t.participacion_ref=s.participacion_ref AND t.recibo_ref=s.recibo_ref)
  UNION ALL
  SELECT 'bolsa:t:'||t.recibo_ref||':'||t.campo,t.registrada_en,
   COALESCE(o.operacion,'bolsa.valor_participacion'),COALESCE(t.actor,''),
   'confirmado'::text,t.recibo_ref,'valor_participacion'::text
  FROM participaciones p JOIN vec_bolsa_llamamientos.traza_valor_participacion t
   ON t.participacion_ref=p.participacion_ref
  LEFT JOIN vec_bolsa_llamamientos.situacion_participacion s
   ON s.participacion_ref=t.participacion_ref AND s.recibo_ref=t.recibo_ref
  LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
   ON o.participacion_ref=s.participacion_ref AND o.desde=s.desde
 ) SELECT h.rid,'bolsa'::text,'bolsa'::text,h.acc,h.actor,h.fecha,h.resultado,
  p_bolsa_ref,COALESCE(h.recibo,''),''::text,''::text,h.motivo,
  '{}'::jsonb,'{}'::jsonb,false,''::text,''::text,'interna_corporativa'::text
 FROM hechos h
 WHERE h.fecha>=p_desde AND h.fecha<p_hasta
  AND (p_actor_filtro='' OR h.actor=p_actor_filtro)
  AND (p_antes_en IS NULL OR (h.fecha,h.rid COLLATE "C")<(p_antes_en,p_antes_id COLLATE "C"))
 ORDER BY h.fecha DESC,h.rid COLLATE "C" DESC LIMIT p_limite;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registros_auditoria_rrhh_v1(
 text,text,text,text,timestamptz,timestamptz,timestamptz,text,integer) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer),
 vec_bolsa_llamamientos.recursos_auditoria_rrhh_v1(text,text,text),
 vec_bolsa_llamamientos.registros_auditoria_rrhh_v1(text,text,text,text,timestamptz,timestamptz,timestamptz,text,integer)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
