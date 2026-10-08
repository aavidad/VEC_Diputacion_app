\set ON_ERROR_STOP on
-- Ejecutar sólo sobre una base PG18 aislada después de instalar AD223.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';

DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_bolsa_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 check_actual text; n bigint; a bigint; estado text;
BEGIN
 IF f IS NULL OR nucleo IS NULL THEN RAISE EXCEPTION 'AD223 prueba: fachada o núcleo ausente'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT check_actual FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(check_actual,'vec_bolsa_llamamientos.ajustes_reglas.v1')=0
 OR strpos(check_actual,'vec_contratacion_temporal.ajustes_reglas.v1')=0
 THEN RAISE EXCEPTION 'AD223 prueba: audiencias fuera del contrato instalado'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
     WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
     AND audiencia_consumo='vec_bolsa_llamamientos.ajustes_reglas.v1'
     AND operacion='bolsa.reglas.ajustar'
     AND proceso='vec-server' AND canal_permitido='interna_corporativa')<>1
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
     aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f)<>2
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
     aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=0)
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT proconfig FROM pg_proc WHERE oid=nucleo) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 THEN RAISE EXCEPTION 'AD223 prueba: origen, ACL o entorno divergente'; END IF;
 SELECT count(*) INTO n FROM vec_autorizacion_atestada_v3.consumo_decision_v3;
 SELECT count(*) INTO a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_bolsa_v3_atestada(
   '{"operacion":"consultar","catalogo_id":"vec.bolsa.reglas.ajustes","organizacion_ref":"organizacion:desarrollo:dipgra"}'::jsonb,
   ''::bytea,''::bytea,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD223 prueba: consulta aceptada por fachada de ajuste';
 EXCEPTION WHEN insufficient_privilege THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  IF estado<>'42501' THEN RAISE EXCEPTION 'AD223 prueba: rechazo erróneo %',estado; END IF;
 END;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)<>n
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>a
 THEN RAISE EXCEPTION 'AD223 prueba: denegación escribió historia'; END IF;
END $test$;
ROLLBACK;
