\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
DO $vector$ DECLARE original jsonb:=$original${"asignacion_id":"ensayo:fechas:aplicacion","version":2,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v5","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-04T13:30:00.000000Z","vigente_hasta":"2026-10-04T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-04T13:30:00.000000Z"}$original$;
 p jsonb:=$plan${"rol_destino_doc":{"publicada_en":"2026-10-04T14:00:00.000000Z"}}$plan$;
 esperado jsonb:=$esperado${"asignacion_id":"ensayo:fechas:aplicacion","version":3,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v6","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-04T14:00:00.000000Z","vigente_hasta":"2026-10-04T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-04T14:00:00.000000Z"}$esperado$;
 d jsonb;BEGIN
 d:=vec_autorizacion.documento_asignacion_destino_mantenimiento_lote_v1(original,p,'ensayo');
 IF d IS DISTINCT FROM esperado OR d->'vigente_hasta' IS DISTINCT FROM original->'vigente_hasta' OR d->'ambitos' IS DISTINCT FROM original->'ambitos' OR d->'principal_id' IS DISTINCT FROM original->'principal_id' OR d->'perfil_activo_ref' IS DISTINCT FROM original->'perfil_activo_ref' THEN RAISE EXCEPTION 'AUT45 vector: destino_fechas_o_identidad_divergentes';END IF;
 IF vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(d) IS DISTINCT FROM d THEN RAISE EXCEPTION 'AUT45 vector: replay_nuevo_distinto';END IF;
END $vector$;
ROLLBACK;
