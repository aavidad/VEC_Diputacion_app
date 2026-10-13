\set ON_ERROR_STOP on
-- AUT82: revalida el permiso nominal gobernado de lectura de auditoría RRHH.
-- No publica roles, concesiones ni asignaciones: esos actos pertenecen a ADMIN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000082',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(jsonb,text,text)') IS NOT NULL
 OR to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
 OR to_regclass('vec_autorizacion.version_rol') IS NULL
 THEN RAISE EXCEPTION 'AUT82: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(
 p_orden jsonb,p_principal text,p_perfil text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
 campos constant jsonb:='["accion","actor_ref","antes","antes_sha256","canal","datos_disponibles","despues","despues_sha256","expediente_ref","finalidad_ref","fuente","id","modulo_id","motivo","ocurrido_en","perfil_ref","recibo_ref","resultado"]'::jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'auditoria'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'consulta_auditoria_rrhh'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'fuente' NOT IN ('ct','bolsa','persona')
 OR p_orden->>'operacion' NOT IN ('opciones','buscar','consultar')
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'organizacion_ref',160) IS NOT TRUE
 OR p_orden->'campos_permitidos' IS DISTINCT FROM campos
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
  AND r.documento->>'rol_id' IS NOT DISTINCT FROM 'consulta_auditoria_rrhh'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND jsonb_typeof(r.documento->'concesiones')='array'
  AND jsonb_array_length(r.documento->'concesiones')=1
  AND c.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND c.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND c.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND c.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND CASE WHEN jsonb_typeof(a.documento->'ambitos')='array' THEN
   jsonb_array_length(a.documento->'ambitos')=3
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') x
    WHERE CASE WHEN jsonb_typeof(x->'valores')='array'
     THEN jsonb_array_length(x->'valores')<>1 ELSE true END
     OR x->>'clave' NOT IN('organizacion_ref','unidad_ref','ambito_ref'))
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(p_orden->>'organizacion_ref')),
    jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(p_orden->>'unidad_ref')),
    jsonb_build_object('clave','ambito_ref','valores',jsonb_build_array(p_orden->>'ambito_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='vec.auditoria.consultar')=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='vec.auditoria.consultar'
    AND concesion->>'modulo_id'='auditoria'
    AND concesion->>'tipo_recurso'='historial_auditoria'
    AND concesion->'finalidades'='["consulta_auditoria_rrhh"]'::jsonb
    AND concesion->'campos_permitidos'=campos
    AND concesion->>'garantia_minima'='alto'
    AND concesion->'obligaciones' IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(jsonb,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(jsonb,text,text)
 TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_actual(text,integer,text,text)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
