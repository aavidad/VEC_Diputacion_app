\set ON_ERROR_STOP on
-- B108: la ficha RRHH distingue contacto ausente de participación inexistente.
-- Cotejo vivo, sobre opcional y asiento positivo comparten la transacción.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000108',0));

DO $pre$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.consultar_origen_datos_contacto_participacion_rrhh_v1(text,text,bytea,bytea,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'PARO clave=B108.dependencia actual=ausente_o_instalada esperado=AD249_y_B108_nueva' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_origen_datos_contacto_participacion_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_contexto_canonico bytea,
 p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(sin_contacto boolean,version bigint,clave_ref text,nonce bytea,cifrado bytea,
 registrada_en timestamptz,recibo_ref text,origen text,vigente_hasta timestamptz,
 ultimo_dia date,regla_ref text,regla_huella_sha256 text,confirmada_en timestamptz,
 auditoria_ref text,consultada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE dato record;acuse record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.datos_contacto_participacion.origen.consultar'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
 OR p_orden->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 THEN RAISE EXCEPTION 'B108: lectura de contacto no autorizada' USING ERRCODE='42501'; END IF;

 -- El cotejo común precede a cualquier lectura del sobre cifrado.
 SELECT * INTO STRICT acuse FROM vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF acuse.auditoria_ref !~ '^aud_v3_lc_[0-9a-f]{32}$'
 OR acuse.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
 OR acuse.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B108: asiento de lectura no confirmado' USING ERRCODE='42501'; END IF;

 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
  JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
  WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref)
 THEN RAISE EXCEPTION 'B108: recurso inexistente' USING ERRCODE='P0002'; END IF;

 SELECT d.version,d.clave_ref,d.nonce,d.cifrado,d.registrada_en,d.recibo_ref INTO dato
 FROM vec_bolsa_llamamientos.datos_contacto_participacion d
 WHERE d.participacion_ref=p_participacion_ref ORDER BY d.version DESC LIMIT 1;
 RETURN QUERY SELECT dato.version IS NULL,coalesce(dato.version,0::bigint),
  dato.clave_ref,dato.nonce,dato.cifrado,dato.registrada_en,dato.recibo_ref,
  o.origen,o.vigente_hasta,o.ultimo_dia,o.regla_ref,o.regla_huella_sha256,cf.confirmada_en,
  acuse.auditoria_ref,acuse.registrada_en
 FROM (SELECT 1) unica
 LEFT JOIN vec_bolsa_llamamientos.origen_datos_contacto_participacion o
  ON o.participacion_ref=p_participacion_ref AND o.version=dato.version
 LEFT JOIN vec_bolsa_llamamientos.confirmacion_contacto_participacion cf
  ON cf.participacion_ref=p_participacion_ref AND cf.version=dato.version;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_origen_datos_contacto_participacion_rrhh_v1(text,text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_origen_datos_contacto_participacion_rrhh_v1(text,text,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
