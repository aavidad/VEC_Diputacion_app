\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:resolucion-formalizacion:dependencia:000025-000069',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000069',0));
LOCK TABLE vec_contratacion_temporal.resolucion_formalizacion IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE MODE;
DO $proteger$
DECLARE v_origen text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_formalizacion)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
           WHERE origen_version='resolucion_formalizacion_o6')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
           WHERE tipo_evento='contratacion_temporal.resolucion_formalizacion_registrada')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.actuacion_expediente_integral
           WHERE actuacion_json->>'accion_clave'='registrar_resolucion_formalizacion') THEN
        RAISE EXCEPTION 'reversión denegada: historia de resolución conservada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF v_origen IS DISTINCT FROM $origen$CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text, 'resolucion_formalizacion_o6'::text])))$origen$ THEN
        RAISE EXCEPTION 'origen integral incompatible para retirar propuesta' USING ERRCODE='55000';
    END IF;
END
$proteger$;
-- Mantiene el control de versión general. Solo permite releer el v7
-- original tras esta actuación v8 y sin cambiar organización, centro o unidad.
DO $detalle_historico$
DECLARE v_def text; v_acl aclitem[]; v_owner oid;
 v_marca text := $marca$           AND publicacion.corte_global <= p_corte_global$marca$;
 v_extension text := $extension$           AND (
               p_consulta.version_observada <> 7 OR publicacion.version = 7
               OR NOT EXISTS (
                   SELECT 1 FROM (
                       SELECT actual.* FROM vec_contratacion_temporal.publicacion_version_rrhh actual
                       WHERE actual.expediente_ref = publicacion.expediente_ref
                         AND actual.corte_global <= p_corte_global
                       ORDER BY actual.corte_global DESC LIMIT 1
                   ) vigente
                   JOIN vec_contratacion_temporal.resolucion_formalizacion resolucion
                     ON resolucion.expediente_ref = vigente.expediente_ref
                    AND resolucion.organizacion_ref = vigente.organizacion_ref
                   WHERE vigente.version = 8
                     AND vigente.organizacion_ref = publicacion.organizacion_ref
                     AND vigente.centro_ref IS NOT DISTINCT FROM publicacion.centro_ref
                     AND vigente.unidad_ref IS NOT DISTINCT FROM publicacion.unidad_ref
               )
           )
$extension$;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN
 RAISE EXCEPTION 'materializador incompatible con recuperación v7' USING ERRCODE='55000'; END IF;
 EXECUTE replace(v_def,chr(10)||v_extension,'');
 IF (SELECT proacl FROM pg_proc WHERE oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure) IS DISTINCT FROM v_acl
 OR (SELECT proowner FROM pg_proc WHERE oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure) IS DISTINCT FROM v_owner THEN
 RAISE EXCEPTION 'recuperación alteró autoridad del materializador' USING ERRCODE='55000'; END IF;
END $detalle_historico$;
DROP FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.registrar_resolucion_formalizacion_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.resolucion_formalizacion;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text])));
-- No borra consumos/auditorías V3: AD3-25 DOWN los protege por separado.
COMMIT;
