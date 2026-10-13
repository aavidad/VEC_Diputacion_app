\set ON_ERROR_STOP on
-- AUT85: resolver la preimagen explícita de una propuesta externa ya registrada.
-- La primera ejecución sigue gobernada por AUT84: cabeza, control y asignaciones CAS.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE h text;p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_admin_version_rol_bolsa_externa_ejecutor') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_historica_v1(text,text,numeric,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT85: precondición nominal divergente' USING ERRCODE='55000';END IF;
 SELECT * INTO p FROM pg_proc
 WHERE oid=to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()');
 h:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF h IS DISTINCT FROM '8ba8bdb4c54e07d79921f0ce0c76df3583874391964993ffab6b6caf245e448e'
 OR p.proowner<>to_regrole('vec_autorizacion_propietario')
 OR p.prosecdef IS NOT TRUE
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
 THEN RAISE EXCEPTION 'AUT85: base actual divergente actual=% esperado=8ba8bdb4c54e07d79921f0ce0c76df3583874391964993ffab6b6caf245e448e',h USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_historica_v1(
 p_base_ref text,p_base_sha256 text,p_control_revision numeric,p_control_sha256 text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE r record;c record;n integer;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('role')<>'none'
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS NOT TRUE
 OR p_base_ref IS NULL OR p_base_sha256 IS NULL OR p_control_sha256 IS NULL
 OR p_base_ref NOT IN ('rol:candidato_bolsa_portal_historial_propio_desarrollo:v1',
  'rol:candidato_bolsa_portal_historial_propio_desarrollo:v2')
 OR p_base_sha256 !~ '^[0-9a-f]{64}$'
 OR p_control_sha256 !~ '^[0-9a-f]{64}$'
 OR p_control_revision IS NULL OR p_control_revision<1 OR p_control_revision<>trunc(p_control_revision)
 THEN RAISE EXCEPTION 'AUT85: preimagen solicitada inválida' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE version_rol_ref=p_base_ref FOR SHARE;
 SELECT * INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol
 WHERE version_rol_ref=r.version_rol_ref AND revision=p_control_revision FOR SHARE;
 n:=jsonb_array_length(r.documento->'concesiones');
 IF r.rol_id<>'candidato_bolsa_portal_historial_propio_desarrollo'
 OR r.version NOT IN (1,2)
 OR r.version_rol_ref IS DISTINCT FROM p_base_ref
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR c.estado IS DISTINCT FROM 'habilitada'
 OR c.documento->>'version_rol_ref' IS DISTINCT FROM p_base_ref
 OR c.revision IS DISTINCT FROM p_control_revision
 OR r.huella_sha256 IS DISTINCT FROM p_base_sha256
 OR c.huella_sha256 IS DISTINCT FROM p_control_sha256
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 OR (r.version=1 AND n<>7)
 OR (r.version=2 AND n NOT IN (9,12))
 OR (r.version=2 AND n=9 AND NOT (r.documento->'concesiones' @>
  '[{"accion":"bolsa.preferencias_bolsas.consultar_propias"},
    {"accion":"bolsa.preferencias_bolsas.ordenar_propias"}]'::jsonb))
 OR (r.version=2 AND n=12 AND NOT (r.documento->'concesiones' @>
  '[{"accion":"bolsa.inscripcion.convocatorias.listar"},
    {"accion":"bolsa.inscripcion.convocatoria.consultar"},
    {"accion":"bolsa.inscripcion.propias.listar"},
    {"accion":"bolsa.inscripcion.propia.consultar"},
    {"accion":"bolsa.inscripcion.presentar"}]'::jsonb))
 THEN RAISE EXCEPTION 'AUT85: preimagen histórica divergente' USING ERRCODE='40001';END IF;
 RETURN jsonb_build_object('rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento),
  'control_vigencia',c.documento,'tipo_perfil','administrable');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_historica_v1(text,text,numeric,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_historica_v1(text,text,numeric,text)
 TO vec_admin_version_rol_bolsa_externa_ejecutor;
COMMIT;
