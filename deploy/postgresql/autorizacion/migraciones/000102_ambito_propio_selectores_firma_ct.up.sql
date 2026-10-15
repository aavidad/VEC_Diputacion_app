\set ON_ERROR_STOP on
-- AUT102: prueba privada de la asignación de firma propia que restringe una
-- lectura CT ya autorizada. La decisión de detalle y su consumo pertenecen a CT.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000102',0));
DO $pre$
DECLARE helper oid:=to_regprocedure('vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)');
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR helper IS NULL
  OR to_regprocedure('vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=helper
   AND p.proowner=to_regrole('vec_autorizacion_propietario') AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog']::text[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='f0ab7c7f98e7747980c52c6a64e735b40f3adfd5171d7ef22ce673cf8513351f'
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario',
    'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario']::aclitem[])
  OR to_regprocedure('vec_autorizacion.canon_version_rol_admin_v1(jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.canon_control_rol_admin_v1(jsonb)') IS NULL
  OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
 THEN RAISE EXCEPTION 'AUT102: preimagen, ACL o dependencias divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(
 p_principal text,p_organizacion text,p_unidad text,p_perfil_aux text,
 p_version_rol_aux text,p_asignacion_ref text,p_asignacion_huella text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE amb jsonb;rol record;ahora timestamptz(6);activas integer;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR session_user=current_user
  OR pg_has_role(session_user,'vec_contratacion_temporal_consultor_rrhh','MEMBER') IS NOT TRUE
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off' OR pg_is_in_recovery()
  OR p_principal IS NULL OR p_principal !~ '^per_[A-Za-z0-9_-]{2,159}$'
  OR p_organizacion IS NULL OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR p_perfil_aux IS NULL OR p_perfil_aux !~ '^prf_[A-Za-z0-9_-]{2,159}$'
  OR p_version_rol_aux IS NULL OR p_version_rol_aux !~ '^rol:[a-z][a-z0-9_]{2,63}:v[1-9][0-9]{0,8}$'
  OR p_asignacion_ref IS NULL OR length(p_asignacion_ref) NOT BETWEEN 3 AND 512
  OR p_asignacion_huella IS NULL OR p_asignacion_huella !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'AUT102: filtro nominal no admitido' USING ERRCODE='42501'; END IF;
 amb:=vec_autorizacion.ambitos_asignacion_firma_ct_v1(
  p_asignacion_ref,p_asignacion_huella,p_principal,p_perfil_aux,p_version_rol_aux);
 IF amb IS NULL OR amb->>'organizacion_ref' IS DISTINCT FROM p_organizacion
  OR amb->>'unidad_ref' IS DISTINCT FROM p_unidad THEN
  RAISE EXCEPTION 'AUT102: asignación propia o unidad no vigente' USING ERRCODE='42501'; END IF;
 ahora:=clock_timestamp();
 -- El certificado del canal ya seleccionó un perfil auxiliar fijo en Go.
 -- Aquí no se permite elegir entre dos perfiles propios de firma vigentes.
 SELECT count(*) INTO activas FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
 WHERE a.principal_id=p_principal AND a.documento->>'estado'='activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.rol_id IN('ct_cargo_tecnico_firma_dev','ct_cargo_jefatura_firma_dev',
    'ct_cargo_diputacion_firma_dev','ct_cargo_secretaria_firma_dev')
  AND r.documento->>'estado'='publicada' AND cv.estado='habilitada';
 IF activas IS DISTINCT FROM 1 THEN
  RAISE EXCEPTION 'AUT102: asignación propia ambigua' USING ERRCODE='42501'; END IF;
 SELECT r.version_rol_ref,r.version,r.rol_id,r.huella_sha256,r.documento,
  c.revision,c.estado,c.huella_sha256 AS control_huella,c.documento AS control_documento
  INTO STRICT rol FROM vec_autorizacion.version_rol r
  JOIN vec_autorizacion.control_vigencia_version_rol_actual a ON a.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision
  WHERE r.version_rol_ref=p_version_rol_aux FOR SHARE OF r,a,c;
 IF rol.documento->>'estado' IS DISTINCT FROM 'publicada'
  OR rol.rol_id NOT IN('ct_cargo_tecnico_firma_dev','ct_cargo_jefatura_firma_dev',
    'ct_cargo_diputacion_firma_dev','ct_cargo_secretaria_firma_dev')
  OR rol.documento->>'rol_id' IS DISTINCT FROM rol.rol_id
  OR rol.documento->>'version' IS DISTINCT FROM rol.version::text
  OR rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
  OR (rol.documento->>'publicada_en')::timestamptz>ahora
  OR rol.estado IS DISTINCT FROM 'habilitada'
  OR rol.control_documento->>'estado' IS DISTINCT FROM 'habilitada'
  OR rol.control_huella IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(rol.control_documento),'UTF8')),'hex')
  OR (rol.control_documento->>'actualizado_en')::timestamptz>ahora THEN
  RAISE EXCEPTION 'AUT102: versión auxiliar retirada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('principal_ref',p_principal,'organizacion_ref',p_organizacion,
  'unidad_ref',p_unidad,'perfil_ref',p_perfil_aux,'version_rol_ref',p_version_rol_aux,
  'version_rol_numero',rol.version,'version_rol_huella_sha256',rol.huella_sha256,
  'asignacion_ref',p_asignacion_ref,'asignacion_huella_sha256',p_asignacion_huella,
  'control_revision',rol.revision,'control_huella_sha256',rol.control_huella);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN
 RAISE EXCEPTION 'AUT102: asignación o control no acreditados' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text) TO vec_contratacion_temporal_propietario;
DO $post$
DECLARE f oid:=to_regprocedure('vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text)');
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner=to_regrole('vec_autorizacion_propietario') AND p.prosecdef
  AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='8d39eeaae8ad82700397125e1044acb45639303bdfd1fe2c025a7e443b7a3b10'
  AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario',
   'vec_contratacion_temporal_propietario=X/vec_autorizacion_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'AUT102: postimagen o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
