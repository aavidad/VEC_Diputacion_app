\set ON_ERROR_STOP on
SET search_path=pg_catalog;
SELECT jsonb_build_object(
 'activa',jsonb_build_object('asignacion_ref',a.asignacion_ref,'huella_sha256',a.huella_sha256,'documento',a.documento),
 'actualizada_por',p.actualizada_por,'acto_ref',p.acto_ref)::text
FROM vec_autorizacion.asignacion_perfil_actual p
JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
WHERE a.version_rol_ref='rol:dietas_r1d_provisional:v1' AND a.version=3;
