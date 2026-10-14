\set ON_ERROR_STOP on
-- AUT96: cotejo nominal de la lectura propia de preferencias entre bolsas en el portal interno.
-- No crea un rol ni concede una acción; exige una versión gobernada y vigente.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000096',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamptz,timestamptz,timestamptz)') IS NULL
 OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 THEN RAISE EXCEPTION 'AUT96: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
-- La concesión interna se publica después mediante un acto ADMIN propio.
-- AUT16 conserva sus siete acciones anteriores y no publica GET/PUT de
-- preferencias. Esta migración solo instala el cotejo vivo de lectura.


CREATE FUNCTION vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(
 p_orden jsonb,p_principal text,p_perfil text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.consultar_propias'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||(p_orden->>'candidato_ref')
 OR p_orden->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
  '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb
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
  AND r.rol_id IS NOT DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
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
   jsonb_array_length(a.documento->'ambitos')=1
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','candidato_ref',
     'valores',jsonb_build_array(p_orden->>'candidato_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias')=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias'
    AND concesion->>'modulo_id'='bolsa'
    AND concesion->>'tipo_recurso'='participaciones_candidato'
    AND concesion->'finalidades'='["gestion_preferencias_bolsas_propias"]'::jsonb
    AND concesion->'campos_permitidos'=p_orden->'campos_permitidos'
    AND concesion->>'garantia_minima'='alto'
    AND coalesce(concesion->'obligaciones','[]'::jsonb)
      IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text),
 vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamptz,timestamptz,timestamptz)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
COMMIT;
