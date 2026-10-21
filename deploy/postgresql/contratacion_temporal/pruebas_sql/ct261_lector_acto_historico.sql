\set ON_ERROR_STOP on
-- Comprobación sin efectos del contrato instalado y de las historias presentes.
-- La llamada nominal con consumo V3 requiere el escenario de aceptación de la copia.
BEGIN READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f oid; cuerpo text; divergentes bigint;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_preparacion_resolucion_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_contratacion_temporal_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT261: contrato instalado divergente' USING ERRCODE='55000'; END IF;
 cuerpo:=pg_get_functiondef(f);
 IF position('v_lectura.version_expediente < 7' IN cuerpo)=0
  OR position('v_actual.agregado_json#>''{actuaciones,7}''' IN cuerpo)=0
  OR position('v_resolucion.recibo_json' IN cuerpo)=0
  OR position('consultar_detalle_rrhh_atestado_v1' IN cuerpo)=0
 THEN RAISE EXCEPTION 'CT261: lectura sin guardas históricas o nominales' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO divergentes
 FROM vec_contratacion_temporal.resolucion_formalizacion r
 JOIN vec_contratacion_temporal.expediente_version_integral v8
   ON v8.expediente_ref=r.expediente_ref AND v8.version=8
 JOIN vec_contratacion_temporal.expediente_integral_actual a
   ON a.expediente_ref=r.expediente_ref AND a.version>8
 JOIN vec_contratacion_temporal.expediente_version_integral actual
   ON actual.expediente_ref=a.expediente_ref AND actual.version=a.version
 WHERE actual.agregado_json#>'{actuaciones,7}' IS DISTINCT FROM
       v8.agregado_json#>'{actuaciones,7}';
 IF divergentes<>0 THEN
  RAISE EXCEPTION 'CT261: una historia posterior alteró el acto v8' USING ERRCODE='55000';
 END IF;
END $prueba$;
ROLLBACK;
