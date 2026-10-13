\set ON_ERROR_STOP on
-- Reversión sólo en una base desechable sin consumo AUT89.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000089',0));
DO $pre$
DECLARE h text;
BEGIN
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p
 WHERE p.oid=to_regprocedure('vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text)');
 IF h IS DISTINCT FROM 'cd28e4aaae2dcf55c098941165028cb8cd41be98b439d836437a4ced390dc091'
 THEN RAISE EXCEPTION 'AUT89 DOWN: cotejo divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(p_orden jsonb, p_principal text, p_perfil text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
 accion text;finalidad text;tipo text;campos jsonb;patron text;
BEGIN
 accion:=p_orden->>'accion';
 CASE accion
 WHEN 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  finalidad:='seguimiento_contacto_rrhh';tipo:='llamamiento_bolsa';
  campos:='["llamamiento_ref","respuestas[].participacion_ref","respuestas[].respuesta"]'::jsonb;patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.datos_contacto_participacion.origen.consultar' THEN
  finalidad:='consulta_origen_datos_contacto_participacion';tipo:='participacion_bolsa';
  campos:='["enmascarados","origen","participacion_ref","registrada_en","sin_contacto","version"]'::jsonb;patron:='^[A-Za-z0-9][A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]?$';
 ELSE RETURN false; END CASE;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM accion
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM finalidad
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' !~ patron
 OR jsonb_typeof(p_orden->'unidad_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_orden->'ambito_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_orden->'bolsa_ref') IS DISTINCT FROM 'string'
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'unidad_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'ambito_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'bolsa_ref',512) IS NOT TRUE

 OR p_orden->'campos_permitidos' IS DISTINCT FROM
    campos
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND c.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND c.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND c.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND c.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND CASE WHEN jsonb_typeof(a.documento->'ambitos')='array' THEN
   jsonb_array_length(a.documento->'ambitos')=2
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','unidad_ref',
     'valores',jsonb_build_array(p_orden->>'unidad_ref')),
    jsonb_build_object('clave','ambito_ref',
     'valores',jsonb_build_array(p_orden->>'ambito_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'=accion)=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'=accion
    AND concesion->>'modulo_id'='bolsa'
    AND concesion->>'tipo_recurso'=tipo
    AND concesion->'finalidades'=jsonb_build_array(finalidad)
    AND concesion->'campos_permitidos'=p_orden->'campos_permitidos'
    AND concesion->>'garantia_minima'='alto'
    AND concesion->'obligaciones' IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $function$;


REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text) TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
