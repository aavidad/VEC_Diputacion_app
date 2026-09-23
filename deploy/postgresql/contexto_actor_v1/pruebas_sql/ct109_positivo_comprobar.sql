\set ON_ERROR_STOP on
-- Se ejecuta tras las dos consultas nominales reales del helper Go.
-- No imprime material de autorización, contexto, COSE ni recibos.
DO $resultado$
DECLARE n integer;
BEGIN
 IF current_setting('session_replication_role') <> 'origin'
 OR EXISTS (SELECT 1 FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid
 JOIN pg_namespace s ON s.oid=c.relnamespace
 WHERE s.nspname IN ('vec_contratacion_temporal','vec_autorizacion',
 'vec_autorizacion_atestada_v3','vec_contexto_actor_v1','vec_identidad_sesiones_v1')
 AND t.tgenabled NOT IN ('O','A'))
 THEN RAISE EXCEPTION 'CT109: prueba con triggers desactivados'; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh)<>2
 OR (SELECT count(*) FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2)<>2
 OR (SELECT count(*) FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2)<>2
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)<>2
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)<>2
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>2
 THEN RAISE EXCEPTION 'CT109: cardinalidad durable distinta de dos consultas'; END IF;
 SELECT count(*) INTO n
 FROM vec_contratacion_temporal.registro_acceso_rrhh r
 JOIN vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 p USING(acceso_ref)
 JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
 ON c.decision_ref=r.decision_ref AND c.consumo_huella_sha256=r.consumo_vec_huella_sha256
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 ON a.auditoria_ref=r.auditoria_vec_ref AND a.huella_sha256=r.auditoria_vec_huella_sha256
 WHERE r.organizacion_ref='org_diputaciondemo0001'
 AND r.ambito_ref='org_diputaciondemo0001' AND r.total=1
 AND r.resultado_generico='entregado'
 AND r.sesion_id='ses_ct109_positivo_00000000000001'
 AND r.actor_ref='per_corporativa_rrhh_000000000001'
 AND r.perfil_id='prf_corporativo_rrhh_000000000001'
 AND p.resultado_huella_sha256=r.resultado_huella_sha256
 AND p.decision_ref=c.decision_ref AND p.auditoria_vec_ref=a.auditoria_ref
 AND p.consumo_vec_huella_sha256=c.consumo_huella_sha256
 AND p.recibo_sello_sha256=encode(sha256(p.recibo_canonico),'hex')
 AND ((r.tipo_consulta='cuadro' AND cardinality(p.resumenes)=1
 AND (p.resumenes[1]).expediente_ref='expediente:ct109:positivo'
 AND (p.resumenes[1]).version=1 AND NOT p.hay_mas)
 OR (r.tipo_consulta='detalle' AND r.expediente_ref='expediente:ct109:positivo'
 AND r.version_expediente=1 AND (p.detalle).resumen.expediente_ref=r.expediente_ref
 AND (p.detalle).resumen.version=1));
 IF n<>2 OR (SELECT count(DISTINCT tipo_consulta)
 FROM vec_contratacion_temporal.registro_acceso_rrhh)<>2
 THEN RAISE EXCEPTION 'CT109: recibo, contenido, consumo o identidad divergentes'; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta)<>1
 OR (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta_version)<>1
 OR (SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral)<>1
 OR (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh)<>1
 OR (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta)<>1
 THEN RAISE EXCEPTION 'CT109: consulta modificó historia del expediente'; END IF;
END $resultado$;
