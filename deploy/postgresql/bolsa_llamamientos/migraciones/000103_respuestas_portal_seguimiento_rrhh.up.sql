\set ON_ERROR_STOP on
-- B103: lectura mínima de respuestas firmes del mismo llamamiento para RRHH.
-- El asiento AD237 y la proyección se confirman en la misma transacción.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000103', 0));

DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(bytea,bytea,jsonb)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'B103: preimagen de lectura nominal incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(
 p_bolsa_ref text, p_llamamiento_ref text, p_contexto_canonico bytea,
 p_vinculo_canonico bytea, p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE
 v_participaciones jsonb;
 v_respuestas jsonb;
 v_auditoria_ref text;
 v_secuencia numeric;
 v_huella text;
 v_correlacion text;
 v_registrada_en timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user = current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512
    OR p_llamamiento_ref !~ '^llamamiento:[0-9a-f]{64}$'
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.respuestas_llamamiento.consultar'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM p_llamamiento_ref
    OR p_orden->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
    OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'seguimiento_contacto_rrhh'
    OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
    OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_respuestas_llamamiento_correcta'
 THEN RAISE EXCEPTION 'B103: lectura de respuesta no autorizada' USING ERRCODE='42501';
 END IF;

 SELECT e.participaciones INTO v_participaciones
   FROM vec_bolsa_llamamientos.llamamiento_emitido e
  WHERE e.bolsa_ref=p_bolsa_ref AND e.llamamiento_ref=p_llamamiento_ref;
 IF v_participaciones IS NULL OR jsonb_typeof(v_participaciones)<>'array'
    OR jsonb_array_length(v_participaciones) NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'B103: llamamiento no disponible' USING ERRCODE='42501';
 END IF;

 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'participacion_ref',r.participacion_ref,'respuesta',r.respuesta)
   ORDER BY p.ordinalidad),'[]'::jsonb) INTO v_respuestas
   FROM jsonb_array_elements_text(v_participaciones) WITH ORDINALITY p(participacion_ref,ordinalidad)
   JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento r
     ON r.bolsa_ref=p_bolsa_ref AND r.llamamiento_ref=p_llamamiento_ref
    AND r.participacion_ref=p.participacion_ref AND r.modo='firme';

 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en
   INTO v_auditoria_ref,v_secuencia,v_huella,v_correlacion,v_registrada_en
   FROM vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(
     p_contexto_canonico,p_vinculo_canonico,p_orden) a;
 IF v_auditoria_ref IS NULL OR v_secuencia IS NULL OR v_huella !~ '^[a-f0-9]{64}$'
    OR v_correlacion IS DISTINCT FROM p_orden->>'correlacion_ref'
    OR v_registrada_en IS NULL THEN
  RAISE EXCEPTION 'B103: acuse de lectura no confiable' USING ERRCODE='42501';
 END IF;

 RETURN jsonb_build_object(
   'esquema','vec.bolsa.rrhh.respuestas-portal.v1',
   'llamamiento_ref',p_llamamiento_ref,
   'respuestas',v_respuestas,
   'auditoria_ref',v_auditoria_ref,
   'correlacion_ref',v_correlacion,
   'consultada_en',to_char(v_registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;

DO $post$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1(text,text,bytea,bytea,jsonb)');
BEGIN
 IF f IS NULL OR NOT EXISTS (
   SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole
    AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
    AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
    OR EXISTS (
      SELECT 1 FROM pg_proc p, LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE')
 THEN RAISE EXCEPTION 'B103: postimagen de permisos incompatible' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
