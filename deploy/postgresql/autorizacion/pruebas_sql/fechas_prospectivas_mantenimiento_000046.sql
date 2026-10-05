\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
DO $vector$ DECLARE original jsonb:=$original${"asignacion_id":"ensayo:fechas:aplicacion","version":1,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v4","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-04T13:00:00.000000Z","vigente_hasta":"2026-10-04T17:00:00.000000Z","emitida_por":"bootstrap:ensayo","emitida_en":"2026-10-04T12:59:00.000000Z"}$original$;
 p jsonb:=$plan${"rol_destino_doc":{"publicada_en":"2026-10-04T13:30:00.000000Z"}}$plan$;
 historico jsonb:=$historico${"asignacion_id":"ensayo:fechas:aplicacion","version":2,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v5","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-04T13:00:00.000000Z","vigente_hasta":"2026-10-04T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-04T13:30:00.000000Z"}$historico$;
 nuevo jsonb:=$nuevo${"asignacion_id":"ensayo:fechas:aplicacion","version":2,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v5","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-04T13:30:00.000000Z","vigente_hasta":"2026-10-04T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-04T13:30:00.000000Z"}$nuevo$;
 d jsonb;BEGIN
 d:=vec_autorizacion.documento_asignacion_destino_mantenimiento_v1(original,p,'ensayo');
 IF d IS DISTINCT FROM historico OR encode(sha256(convert_to(d::text,'UTF8')),'hex') IS DISTINCT FROM encode(sha256(convert_to(historico::text,'UTF8')),'hex') THEN RAISE EXCEPTION 'AUT46 vector: historial_bytes_distintos';END IF;
 d:=vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(d);
 IF d IS DISTINCT FROM nuevo OR d->'vigente_hasta' IS DISTINCT FROM original->'vigente_hasta' OR (d-'vigente_desde') IS DISTINCT FROM(historico-'vigente_desde') THEN RAISE EXCEPTION 'AUT46 vector: destino_fechas_o_identidad_divergentes';END IF;
 IF vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(d) IS DISTINCT FROM d THEN RAISE EXCEPTION 'AUT46 vector: replay_nuevo_distinto';END IF;
 BEGIN
  PERFORM vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(jsonb_set(historico,'{emitida_en}',historico->'vigente_hasta'));
  RAISE EXCEPTION 'AUT46 vector: inicio_fin_iguales_aceptados';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
END $vector$;
ROLLBACK;
