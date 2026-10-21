\set ON_ERROR_STOP on
-- CT262 / F1-27 B. Proyección interna del selector nativo conservado en CT253.
-- Usa el mismo consumo nominal de detalle; no crea acción ni permiso.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000262',0));
DO $pre$
DECLARE n oid; detalle oid;
BEGIN
 n:=to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 detalle:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR detalle IS NULL OR n IS NULL
  OR to_regprocedure('vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[])') IS NULL
  OR to_regclass('vec_contratacion_temporal.propuesta_formalizacion') IS NULL
  OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
  OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
  OR to_regclass('vec_contratacion_temporal.no_incorporacion_v1') IS NULL
  OR to_regclass('vec_contratacion_temporal.propuesta_sustitucion_v1') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.propuesta_documental_vigente_ct262(text,jsonb)') IS NOT NULL
  OR NOT EXISTS (SELECT 1 FROM pg_class t WHERE t.oid=to_regclass('vec_contratacion_temporal.propuesta_formalizacion')
      AND t.relowner='vec_contratacion_temporal_propietario'::regrole
      AND t.relrowsecurity AND t.relforcerowsecurity)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=detalle
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=n
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
      AND encode(sha256(convert_to(pg_get_functiondef(n),'UTF8')),'hex')=
          'f29879a9e90b3d0daef565b657919d5b5c26b2d5b366091d3cb04b761177737a')
 THEN RAISE EXCEPTION 'CT262: preimagen o dependencia ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- La propuesta v7 deja de servir a los documentos cuando hubo después una no
-- incorporación efectiva (CT124) o una propuesta que la sustituye (CT128).
-- no_incorporacion_v1 solo es visible para sesiones del ejecutor (política de
-- CT124); la sesión del consultor RRHH la ve vacía. Por eso la señal efectiva
-- es la actuación de no incorporación del agregado actual, que sí lee; la
-- consulta a la tabla se conserva para las sesiones que la ven.
CREATE FUNCTION vec_contratacion_temporal.propuesta_documental_vigente_ct262(
 p_propuesta_ref text, p_actual jsonb
) RETURNS boolean
LANGUAGE sql STABLE SECURITY INVOKER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
AS $vigente$
 SELECT CASE WHEN p_propuesta_ref IS NULL
   OR jsonb_typeof(p_actual->'actuaciones') IS DISTINCT FROM 'array' THEN false
  ELSE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(p_actual->'actuaciones') a
       WHERE a->>'accion_clave'='contratacion_temporal.incorporacion.no_incorporacion')
   AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n
       WHERE n.propuesta_ref=p_propuesta_ref)
   AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
       WHERE x.propuesta_anterior_ref=p_propuesta_ref)
  END
$vigente$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.propuesta_documental_vigente_ct262(text,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona numeric,p_perfil numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric,
 anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text, registrada_en timestamptz, auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text,
 generada_en timestamptz, expediente_ref text, version_expediente numeric,
 total smallint, recibo_sello_sha256 text, selector jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $lector$
DECLARE lectura record;
 actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 original vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
 seleccion jsonb;
BEGIN
 IF p_consulta.version_observada IS DISTINCT FROM 0::numeric THEN
  RAISE EXCEPTION 'selector documental no disponible' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT lectura FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona,p_perfil,
  p_payload,p_sobre,p_evidencia,p_raiz);
 -- Ninguna tabla de negocio se lee antes del consumo y el recibo de acceso.
 IF lectura.expediente_ref IS DISTINCT FROM p_consulta.expediente_ref
  OR lectura.version_expediente<7 THEN
  RAISE EXCEPTION 'selector documental no disponible' USING ERRCODE='42501';
 END IF;
 BEGIN
  SELECT v.* INTO STRICT actual FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
  WHERE a.expediente_ref=lectura.expediente_ref;
  SELECT * INTO STRICT original FROM vec_contratacion_temporal.expediente_version_integral v
  WHERE v.expediente_ref=lectura.expediente_ref AND v.version=7;
  SELECT * INTO STRICT propuesta FROM vec_contratacion_temporal.propuesta_formalizacion p
  WHERE p.expediente_ref=lectura.expediente_ref AND p.organizacion_ref=p_alcance.organizacion_ref
   AND p.propuesta_ref=original.operacion_ref AND p.version_resultante=7;
 EXCEPTION WHEN no_data_found OR too_many_rows THEN
  RAISE EXCEPTION 'selector documental no disponible' USING ERRCODE='42501';
 END;
 IF actual.version IS DISTINCT FROM lectura.version_expediente
  OR actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
  OR original.origen_version IS DISTINCT FROM 'propuesta_formalizacion_o6'
  OR original.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
  OR original.agregado_json_huella_sha256 IS DISTINCT FROM
     encode(sha256(convert_to(original.agregado_json::text,'UTF8')),'hex')
  OR actual.agregado_json_huella_sha256 IS DISTINCT FROM
     encode(sha256(convert_to(actual.agregado_json::text,'UTF8')),'hex')
  OR original.agregado_json#>>'{actuaciones,6,accion_clave}' IS DISTINCT FROM 'registrar_propuesta_formalizacion'
  OR original.agregado_json#>>'{actuaciones,6,recibo_ref}' IS DISTINCT FROM propuesta.recibo_ref
  OR actual.agregado_json#>'{actuaciones,6}' IS DISTINCT FROM original.agregado_json#>'{actuaciones,6}'
  OR propuesta.version_previa IS DISTINCT FROM 6
  OR propuesta.material_json IS DISTINCT FROM propuesta.material::jsonb
  OR propuesta.material_sha256 IS DISTINCT FROM encode(sha256(convert_to(propuesta.material,'UTF8')),'hex')
  OR propuesta.solicitud_json IS DISTINCT FROM propuesta.material_json->'Solicitud'
  OR propuesta.aceptacion_bolsa_json IS DISTINCT FROM propuesta.material_json->'AceptacionBolsaNativa'
  OR NOT (propuesta.material_json ? 'AceptacionBolsaNativa')
  OR propuesta.material_json ? 'AceptacionBolsa'
  OR propuesta.material_json->>'Etapa' IS DISTINCT FROM 'confirmacion'
  OR propuesta.solicitud_json->>'OrganizacionRef' IS DISTINCT FROM p_alcance.organizacion_ref
  OR propuesta.solicitud_json->>'ExpedienteRef' IS DISTINCT FROM lectura.expediente_ref
  OR propuesta.solicitud_json->>'LlamamientoRef' IS DISTINCT FROM propuesta.llamamiento_ref
  OR propuesta.solicitud_json->>'ResolucionLlamamientoAceptadaRef' IS DISTINCT FROM propuesta.resolucion_ref
  OR propuesta.solicitud_json->>'ReciboResolucionAceptadaRef' IS DISTINCT FROM propuesta.recibo_aceptacion_ref
  OR propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM propuesta.propuesta_ref
  OR propuesta.recibo_json->>'ReciboLocalRef' IS DISTINCT FROM propuesta.recibo_ref
  OR propuesta.recibo_json->>'AuditoriaRef' IS DISTINCT FROM propuesta.auditoria_ref
  OR propuesta.recibo_json->'VersionResultante' IS DISTINCT FROM '7'::jsonb
  OR propuesta.recibo_json->>'Estado' IS DISTINCT FROM 'confirmado'
  OR vec_contratacion_temporal.propuesta_documental_vigente_ct262(propuesta.propuesta_ref,actual.agregado_json) IS NOT TRUE THEN
  RAISE EXCEPTION 'selector documental no confiable' USING ERRCODE='42501';
 END IF;
 seleccion:=propuesta.aceptacion_bolsa_json->'Selector';
 IF jsonb_typeof(seleccion) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'selector documental no confiable' USING ERRCODE='42501';
 END IF;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(seleccion,ARRAY[
   'organizacion_ref','expediente_ref','clave_ct','unidad_ref','categoria_ref',
   'necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref']) IS NOT TRUE
  OR EXISTS (SELECT 1 FROM jsonb_each(seleccion) x
     WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
      OR octet_length(x.value#>>'{}') NOT BETWEEN 3 AND 160
      OR x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
  OR seleccion->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
  OR seleccion->>'expediente_ref' IS DISTINCT FROM lectura.expediente_ref
  OR seleccion->>'unidad_ref' IS DISTINCT FROM original.agregado_json#>>'{asignacion,unidad_ref}'
  OR seleccion->>'categoria_ref' IS DISTINCT FROM original.agregado_json#>>'{analisis,categoria_ref}'
  OR seleccion->>'unidad_ref' IS DISTINCT FROM actual.agregado_json#>>'{asignacion,unidad_ref}'
  OR seleccion->>'categoria_ref' IS DISTINCT FROM actual.agregado_json#>>'{analisis,categoria_ref}'
  OR seleccion->>'llamamiento_ref' IS DISTINCT FROM propuesta.llamamiento_ref THEN
  RAISE EXCEPTION 'selector documental no confiable' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT lectura.contenido_canonico,lectura.esquema,lectura.acceso_ref,
  lectura.secuencia,lectura.anterior_sha256,lectura.huella_sha256,
  lectura.vinculo_identidad_huella_sha256,lectura.alcance_huella_sha256,
  lectura.registrada_en,lectura.auditoria_vec_ref,lectura.auditoria_vec_huella_sha256,
  lectura.consumo_vec_huella_sha256,lectura.contenido_huella_sha256,
  lectura.resultado_huella_sha256,lectura.cursor_huella_sha256,lectura.generada_en,
  lectura.expediente_ref,lectura.version_expediente,lectura.total,
  lectura.recibo_sello_sha256,seleccion;
END $lector$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh;
DO $post$
DECLARE f oid; g oid;
BEGIN
 g:='vec_contratacion_temporal.propuesta_documental_vigente_ct262(text,jsonb)'::regprocedure;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=g
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole
  AND NOT p.prosecdef AND p.provolatile='s' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT262: vigencia de la propuesta divergente' USING ERRCODE='55000'; END IF;
 f:='vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT262: contrato o permisos divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
