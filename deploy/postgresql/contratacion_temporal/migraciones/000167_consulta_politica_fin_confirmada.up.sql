\set ON_ERROR_STOP on
-- CT167: snapshot de política de fin de una operación ya confirmada.
-- Sólo ayuda a recomponer la petición original; no acredita su replay ni
-- concede autorización. Esos controles siguen en la operación existente.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:000167:politica_fin',0));

DO $precondicion$
BEGIN
 IF pg_catalog.to_regclass('vec_contratacion_temporal.alias_consulta_operacion_analisis') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.confirmacion_operacion_analisis') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.alias_ambito_alta') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.confirmacion_agregado_alta') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(jsonb,text,text,text,text,text,numeric)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(jsonb,text,text,text,text)') IS NOT NULL THEN
    RAISE EXCEPTION 'CT167 preimagen incompatible' USING ERRCODE='55000';
 END IF;
END
$precondicion$;

-- El HMAC de ámbito se calcula fuera de SQL con la clave, organización,
-- expediente, actor y perfil. Esta lectura no recibe datos funcionales ni
-- política del solicitante. El recibo sigue validándose en el camino normal.
CREATE FUNCTION vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(
 p_ambitos jsonb, p_organizacion_ref text, p_expediente_ref text,
 p_actor_ref text, p_perfil_ref text, p_operacion text,
 p_version_anterior numeric
)
RETURNS TABLE (politica_fin jsonb)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE v_raices text[]; v_raiz text; v_fila record; v_politica jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
    RAISE EXCEPTION 'identidad de consulta de análisis no autorizada' USING ERRCODE='42501';
 END IF;
 IF p_ambitos IS NULL OR pg_catalog.jsonb_typeof(p_ambitos)<>'array'
    OR pg_catalog.jsonb_array_length(p_ambitos) NOT BETWEEN 1 AND 4
    OR p_organizacion_ref IS NULL OR p_expediente_ref IS NULL
    OR p_actor_ref IS NULL OR p_perfil_ref IS NULL OR p_operacion IS NULL
    OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_operacion NOT IN ('registrar','rectificar')
    OR p_version_anterior IS NULL OR p_version_anterior<>pg_catalog.trunc(p_version_anterior)
    OR p_version_anterior NOT BETWEEN 1 AND 9007199254740990::numeric
    OR EXISTS (
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v)
        WHERE pg_catalog.jsonb_typeof(e.v)<>'string'
           OR e.v #>> '{}' !~
              '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) THEN
    RAISE EXCEPTION 'consulta de política de análisis inválida' USING ERRCODE='22023';
 END IF;
 SELECT pg_catalog.array_agg(DISTINCT a.ambito_raiz_hmac)
   INTO v_raices
   FROM vec_contratacion_temporal.alias_consulta_operacion_analisis a
  WHERE a.alias_ambito_consulta_hmac IN (
      SELECT e.v #>> '{}' FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v));
 IF pg_catalog.cardinality(v_raices)>1 THEN
    RAISE EXCEPTION 'ámbitos de análisis divergentes' USING ERRCODE='23505';
 END IF;
 v_raiz := v_raices[1];
 IF v_raiz IS NULL THEN RETURN; END IF;
 SELECT r.organizacion_ref, r.expediente_ref, r.actor_ref, r.perfil_ref,
        r.operacion, r.version_expediente, r.recibo_ref,
        c.recibo_json, v.origen_version, v.operacion_ref,
        v.agregado_json
   INTO v_fila
   FROM vec_contratacion_temporal.reserva_operacion_analisis r
   JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c
     ON c.ambito_raiz_hmac=r.ambito_raiz_hmac
   JOIN vec_contratacion_temporal.expediente_version_integral v
     ON v.expediente_ref=r.expediente_ref
    AND v.version=r.version_expediente+1
  WHERE r.ambito_raiz_hmac=v_raiz;
 IF NOT FOUND THEN
    RAISE EXCEPTION 'historia confirmada de análisis incompleta' USING ERRCODE='55000';
 END IF;
 IF v_fila.organizacion_ref IS DISTINCT FROM p_organizacion_ref
    OR v_fila.expediente_ref IS DISTINCT FROM p_expediente_ref
    OR v_fila.actor_ref IS DISTINCT FROM p_actor_ref
    OR v_fila.perfil_ref IS DISTINCT FROM p_perfil_ref
    OR v_fila.operacion IS DISTINCT FROM p_operacion
    OR v_fila.version_expediente IS DISTINCT FROM p_version_anterior THEN
    RAISE EXCEPTION 'clave de análisis usada con otros datos' USING ERRCODE='23505';
 END IF;
 IF v_fila.origen_version<>'analisis_o3'
    OR v_fila.operacion_ref IS DISTINCT FROM
      'operacion:analisis:'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_raiz||':'||v_fila.recibo_ref,'UTF8')),'hex'),1,32)
    OR v_fila.recibo_json->>'recibo_ref' IS DISTINCT FROM v_fila.recibo_ref
    OR v_fila.recibo_json->>'organizacion_ref' IS DISTINCT FROM v_fila.organizacion_ref
    OR v_fila.recibo_json->>'expediente_ref' IS DISTINCT FROM v_fila.expediente_ref
    OR v_fila.recibo_json->>'version_resultante' IS DISTINCT FROM
       (v_fila.version_expediente+1)::text THEN
    RAISE EXCEPTION 'historia confirmada de análisis incoherente' USING ERRCODE='55000';
 END IF;
 v_politica := v_fila.agregado_json#>'{analisis,periodo,politica_fin}';
 IF v_politica IS NOT NULL AND
    vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(
      v_fila.agregado_json#>'{analisis,periodo}') IS NOT TRUE THEN
    RAISE EXCEPTION 'snapshot de análisis inválido' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT v_politica;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(
 p_ambitos jsonb, p_organizacion_ref text, p_expediente_ref text,
 p_actor_ref text, p_perfil_ref text
)
RETURNS TABLE (politica_fin jsonb)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE v_raices text[]; v_raiz text; v_fila record; v_politica jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
    RAISE EXCEPTION 'identidad de consulta de alta no autorizada' USING ERRCODE='42501';
 END IF;
 IF p_ambitos IS NULL OR pg_catalog.jsonb_typeof(p_ambitos)<>'array'
    OR pg_catalog.jsonb_array_length(p_ambitos) NOT BETWEEN 1 AND 4
    OR p_organizacion_ref IS NULL OR p_expediente_ref IS NULL
    OR p_actor_ref IS NULL OR p_perfil_ref IS NULL
    OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (p_expediente_ref<>'' AND
        p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
    OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR EXISTS (
       SELECT 1 FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v)
        WHERE pg_catalog.jsonb_typeof(e.v)<>'string'
           OR e.v #>> '{}' !~
              '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    ) THEN
    RAISE EXCEPTION 'consulta de política de alta inválida' USING ERRCODE='22023';
 END IF;
 SELECT pg_catalog.array_agg(DISTINCT a.ambito_raiz_hmac)
   INTO v_raices
   FROM vec_contratacion_temporal.alias_ambito_alta a
  WHERE a.alias_hmac IN (
      SELECT e.v #>> '{}' FROM pg_catalog.jsonb_array_elements(p_ambitos) e(v));
 IF pg_catalog.cardinality(v_raices)>1 THEN
    RAISE EXCEPTION 'ámbitos de alta divergentes' USING ERRCODE='23505';
 END IF;
 v_raiz := v_raices[1];
 IF v_raiz IS NULL THEN RETURN; END IF;
 SELECT i.organizacion_ref, i.expediente_ref, i.actor_ref, i.perfil_ref,
        i.recibo_ref, c.ambito_hmac, c.expediente_ref AS confirmado_expediente_ref,
        c.recibo_ref AS confirmado_recibo_ref,
        a.organizacion_ref AS alta_organizacion_ref,
        a.actor_ref AS alta_actor_ref, a.perfil_ref AS alta_perfil_ref,
        v.origen_version, v.operacion_ref, v.agregado_json
   INTO v_fila
   FROM vec_contratacion_temporal.identidad_reserva_alta i
   LEFT JOIN vec_contratacion_temporal.confirmacion_agregado_alta c
     ON c.ambito_hmac=i.ambito_hmac
   LEFT JOIN vec_contratacion_temporal.expediente_alta a
     ON a.expediente_ref=c.expediente_ref
   LEFT JOIN vec_contratacion_temporal.expediente_version_integral v
     ON v.expediente_ref=c.expediente_ref AND v.version=1
  WHERE i.ambito_hmac=v_raiz;
 IF NOT FOUND THEN
    RAISE EXCEPTION 'identidad de alta ausente' USING ERRCODE='55000';
 END IF;
 IF v_fila.organizacion_ref IS DISTINCT FROM p_organizacion_ref
    OR (p_expediente_ref<>'' AND
        v_fila.expediente_ref IS DISTINCT FROM p_expediente_ref)
    OR v_fila.actor_ref IS DISTINCT FROM p_actor_ref
    OR v_fila.perfil_ref IS DISTINCT FROM p_perfil_ref THEN
    RAISE EXCEPTION 'clave de alta usada con otros datos' USING ERRCODE='23505';
 END IF;
 IF v_fila.ambito_hmac IS NULL THEN RETURN; END IF;
 IF v_fila.confirmado_expediente_ref IS DISTINCT FROM v_fila.expediente_ref
    OR v_fila.confirmado_recibo_ref IS DISTINCT FROM v_fila.recibo_ref
    OR v_fila.alta_organizacion_ref IS DISTINCT FROM v_fila.organizacion_ref
    OR v_fila.alta_actor_ref IS DISTINCT FROM v_fila.actor_ref
    OR v_fila.alta_perfil_ref IS DISTINCT FROM v_fila.perfil_ref
    OR v_fila.origen_version IS DISTINCT FROM 'alta_o2'
    OR v_fila.operacion_ref IS DISTINCT FROM 'alta:'||v_fila.recibo_ref
    OR v_fila.agregado_json IS NULL THEN
    RAISE EXCEPTION 'historia confirmada de alta incompleta' USING ERRCODE='55000';
 END IF;
 v_politica := v_fila.agregado_json#>'{solicitud,periodo,politica_fin}';
 IF v_politica IS NOT NULL AND
    vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(
      v_fila.agregado_json#>'{solicitud,periodo}') IS NOT TRUE THEN
    RAISE EXCEPTION 'snapshot de alta inválido' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT v_politica;
END
$funcion$;

REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(jsonb,text,text,text,text,text,numeric),
 vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(jsonb,text,text,text,text)
FROM PUBLIC, vec_contratacion_temporal_migrador,
 vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION
 vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(jsonb,text,text,text,text,text,numeric),
 vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(jsonb,text,text,text,text)
TO vec_contratacion_temporal_ejecutor;
COMMIT;
