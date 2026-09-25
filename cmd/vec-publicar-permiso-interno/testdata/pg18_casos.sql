\set ON_ERROR_STOP on
-- Ejecutar tras el positivo y replay de la CLI en vec_permiso_v3.
DO $aserciones$
DECLARE v jsonb; a jsonb;
BEGIN
  IF (SELECT count(*) FROM vec_autorizacion.asignacion_perfil
       WHERE perfil_activo_ref='prf_0123456789abcdefghijkl') <> 1
     OR (SELECT count(*) FROM vec_autorizacion.version_rol
       WHERE rol_id LIKE 'rrhh_interno_certificado_seguimiento_ct_%') <> 1
     OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil
       WHERE perfil_activo_ref='prf_alto123456789abcdefghijk') <> 1
  THEN RAISE EXCEPTION 'publicación/replay alteró cardinalidad'; END IF;
  SELECT r.documento,a0.documento INTO v,a
    FROM vec_autorizacion.asignacion_perfil_actual aa
    JOIN vec_autorizacion.asignacion_perfil a0 ON a0.asignacion_ref=aa.asignacion_ref
    JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a0.version_rol_ref
   WHERE aa.perfil_activo_ref='prf_0123456789abcdefghijkl';
  IF jsonb_array_length(v->'concesiones') <> 1
     OR v#>>'{concesiones,0,accion}' <> 'contratacion_temporal.expediente.consultar'
     OR v#>>'{concesiones,0,garantia_minima}' <> 'sustancial'
     OR jsonb_array_length(v#>'{concesiones,0,campos_permitidos}') <> 14
     OR jsonb_array_length(a->'ambitos') <> 3
     OR a->>'principal_id' <> 'per_0123456789abcdefghijkl'
     OR (a->>'vigente_hasta')::timestamptz > '2026-11-01 00:00:00+00'::timestamptz
  THEN RAISE EXCEPTION 'contrato del permiso interno incumplido'; END IF;
END $aserciones$;
