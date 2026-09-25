\set ON_ERROR_STOP on
SET search_path=pg_catalog;
SELECT jsonb_build_object(
 'original',jsonb_build_object('asignacion_ref',o.asignacion_ref,'huella_sha256',o.huella_sha256,'documento',o.documento),
 'revocada',jsonb_build_object('asignacion_ref',r.asignacion_ref,'huella_sha256',r.huella_sha256,'documento',r.documento),
 'actualizada_por',p.actualizada_por,'acto_ref',p.acto_ref)::text
FROM vec_autorizacion.asignacion_perfil_actual p
JOIN vec_autorizacion.asignacion_perfil r ON r.asignacion_ref=p.asignacion_ref
JOIN vec_autorizacion.asignacion_perfil o ON o.asignacion_id=r.asignacion_id AND o.version=1
WHERE r.version_rol_ref='rol:dietas_r1d_provisional:v1' AND r.version=2;
