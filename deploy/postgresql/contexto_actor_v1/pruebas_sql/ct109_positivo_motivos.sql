\set ON_ERROR_STOP on
-- Conexión efectiva -U ct109_motivos_proyector. No SET ROLE ni suplantación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $publicar$
DECLARE t timestamptz:=clock_timestamp()-interval '1 minute';
 entradas jsonb;
BEGIN
 IF session_user<>'ct109_motivos_proyector' OR current_user<>session_user
 THEN RAISE EXCEPTION 'CT109: falta conexión nominal del proyector'; END IF;
 entradas:=jsonb_build_array(
 jsonb_build_object('clave','motivo_11111111111111111111111111111111',
 'vigente_desde',to_char(t,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',NULL),
 jsonb_build_object('clave','motivo_22222222222222222222222222222222',
 'vigente_desde',to_char(t,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',NULL));
 IF vec_autorizacion.publicar_motivos_autorizacion_v2(
 'evento_10900000000000000000000000000001',1,repeat('a',64),
 'motivos_ct109_positivo',1,repeat('b',64),t,entradas) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT109: catálogo de motivos no publicado'; END IF;
 IF vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
 'evento_vinculacion_motivo_rrhh_10900000000000000000000000000002',repeat('c',64),1,
 'publicacion_motivo_rrhh_10900000000000000000000000000002',repeat('d',64),
 'motivos_ct109_positivo',1,repeat('b',64),
 'motivo_11111111111111111111111111111111',t) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT109: vínculo de motivo cuadro no publicado'; END IF;
 IF vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
 'evento_vinculacion_motivo_rrhh_10900000000000000000000000000003',repeat('e',64),1,
 'publicacion_motivo_rrhh_10900000000000000000000000000003',repeat('f',64),
 'motivos_ct109_positivo',1,repeat('b',64),
 'motivo_22222222222222222222222222222222',t) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT109: vínculo de motivo detalle no publicado'; END IF;
END $publicar$;
COMMIT;
