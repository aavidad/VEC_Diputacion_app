\set ON_ERROR_STOP on
-- Ejecutar como postgres en PostgreSQL 18 desechable con AUTH-1 y AUTH-14.
-- Todas las filas sintéticas de esta prueba terminan en ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.version_rol
 (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
VALUES
 ('rol:rrhh_interno_certificado_seguimiento_ct_0123456789abcdef:v1',
  'rrhh_interno_certificado_seguimiento_ct_0123456789abcdef',1,repeat('a',64),
  '2026-09-24 00:00:00+00',jsonb_build_object(
    'rol_id','rrhh_interno_certificado_seguimiento_ct_0123456789abcdef',
    'version',1,'nombre','Consulta interna CT','estado','publicada',
    'publicada_por','seguridad:prueba','publicada_en','2026-09-24T00:00:00Z',
    'concesiones',jsonb_build_array(jsonb_build_object(
      'accion','contratacion_temporal.expediente.consultar',
      'modulo_id','contratacion_temporal',
      'tipo_recurso','expediente_contratacion_temporal',
      'finalidades',jsonb_build_array('tramitacion_expediente_contratacion_temporal'),
      'garantia_minima','sustancial')))),
 ('rol:rrhh_ordinario_prueba:v1','rrhh_ordinario_prueba',1,repeat('b',64),
  '2026-09-24 00:00:00+00',jsonb_build_object(
    'rol_id','rrhh_ordinario_prueba','version',1,'nombre','Consulta ordinaria',
    'estado','publicada','publicada_por','seguridad:prueba',
    'publicada_en','2026-09-24T00:00:00Z',
    'concesiones',jsonb_build_array(jsonb_build_object(
      'accion','contratacion_temporal.expediente.consultar',
      'modulo_id','contratacion_temporal',
      'tipo_recurso','expediente_contratacion_temporal',
      'finalidades',jsonb_build_array('tramitacion_expediente_contratacion_temporal'),
      'garantia_minima','alto'))));
INSERT INTO vec_autorizacion.control_vigencia_version_rol
 (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
SELECT version_rol_ref,1,'habilitada',repeat('c',64),'2026-09-24 00:00:00+00',
 jsonb_build_object('version_rol_ref',version_rol_ref,'revision',1,
   'estado','habilitada','actualizado_por','seguridad:prueba',
   'actualizado_en','2026-09-24T00:00:00Z')
FROM vec_autorizacion.version_rol;
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
 (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
SELECT version_rol_ref,1,'2026-09-24 00:00:00+00','seguridad:prueba',
       'acto:prueba:control:'||rol_id FROM vec_autorizacion.version_rol;
INSERT INTO vec_autorizacion.asignacion_perfil
 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
  version_rol_ref,huella_sha256,emitida_en,documento)
VALUES
 ('asignacion:perfil_interno_prueba:v1','perfil_interno_prueba',1,
  'prf_interno_prueba','per_prueba',
  'rol:rrhh_interno_certificado_seguimiento_ct_0123456789abcdef:v1',repeat('d',64),
  '2026-09-24 00:00:00+00',jsonb_build_object(
   'asignacion_id','perfil_interno_prueba','version',1,
   'perfil_activo_ref','prf_interno_prueba','principal_id','per_prueba',
   'version_rol_ref','rol:rrhh_interno_certificado_seguimiento_ct_0123456789abcdef:v1',
   'estado','activa','ambitos',jsonb_build_array(jsonb_build_object(
      'clave','organizacion_ref','valores',jsonb_build_array('org_prueba'))),
   'vigente_desde','2026-09-24T00:00:00Z','vigente_hasta','2026-10-31T23:00:00Z',
   'emitida_por','identidad:prueba','emitida_en','2026-09-24T00:00:00Z')),
 ('asignacion:perfil_ordinario_prueba:v1','perfil_ordinario_prueba',1,
  'prf_ordinario_prueba','per_prueba','rol:rrhh_ordinario_prueba:v1',repeat('e',64),
  '2026-09-24 00:00:00+00',jsonb_build_object(
   'asignacion_id','perfil_ordinario_prueba','version',1,
   'perfil_activo_ref','prf_ordinario_prueba','principal_id','per_prueba',
   'version_rol_ref','rol:rrhh_ordinario_prueba:v1','estado','activa',
   'ambitos',jsonb_build_array(jsonb_build_object(
      'clave','organizacion_ref','valores',jsonb_build_array('org_prueba'))),
   'vigente_desde','2026-09-24T00:00:00Z','vigente_hasta','2026-10-31T23:00:00Z',
   'emitida_por','identidad:prueba','emitida_en','2026-09-24T00:00:00Z'));
INSERT INTO vec_autorizacion.asignacion_perfil_actual
 (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
VALUES
 ('prf_interno_prueba','asignacion:perfil_interno_prueba:v1',
  '2026-09-24 00:00:00+00','identidad:prueba','acto:prueba:interno'),
 ('prf_ordinario_prueba','asignacion:perfil_ordinario_prueba:v1',
  '2026-09-24 00:00:00+00','identidad:prueba','acto:prueba:ordinario');
RESET ROLE;

SET SESSION AUTHORIZATION vec_otro_fuente_prueba;
DO $ajeno$ BEGIN
 IF NOT pg_catalog.has_function_privilege(session_user,
   'vec_autorizacion.obtener_instantanea(text,text)','EXECUTE')
    OR pg_catalog.has_table_privilege(session_user,
   'vec_autorizacion.asignacion_perfil','SELECT')
    OR EXISTS (SELECT 1 FROM vec_autorizacion.obtener_instantanea(
   'per_prueba','prf_interno_prueba'))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.obtener_instantanea(
   'per_prueba','prf_ordinario_prueba'))
 THEN RAISE EXCEPTION 'AUTH14: cruce de canal o regresión ordinaria'; END IF;
END $ajeno$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_interno_v3_fuente_autorizacion_desarrollo;
DO $interno$ BEGIN
 IF NOT pg_catalog.has_function_privilege(session_user,
   'vec_autorizacion.obtener_instantanea(text,text)','EXECUTE')
    OR pg_catalog.has_table_privilege(session_user,
   'vec_autorizacion.asignacion_perfil','SELECT')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.obtener_instantanea(
   'per_prueba','prf_interno_prueba'))
    OR EXISTS (SELECT 1 FROM vec_autorizacion.obtener_instantanea(
   'per_otra','prf_interno_prueba'))
 THEN RAISE EXCEPTION 'AUTH14: lector nominal o principal exacto'; END IF;
END $interno$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
