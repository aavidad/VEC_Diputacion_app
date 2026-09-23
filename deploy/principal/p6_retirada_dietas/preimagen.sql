\set ON_ERROR_STOP on
SET search_path = pg_catalog;
SELECT jsonb_build_object(
 'asignacion_ref', a.asignacion_ref,
 'huella_sha256', a.huella_sha256,
 'documento', a.documento
)::text
FROM vec_autorizacion.asignacion_perfil_actual p
JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
WHERE r.rol_id='dietas_r1d_provisional'
  AND a.documento->>'estado'='activa';
