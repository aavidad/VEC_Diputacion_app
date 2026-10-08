\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:orq1:ad226', 0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'AD226: PARO clave=migrador actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';
 END IF;
 IF to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD226: PARO clave=dependencias actual=ausentes esperado=AD225_CT198_y_configuracion_origen' USING ERRCODE='55000';
 END IF;
END
$pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $origen$
DECLARE
 anterior vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1%ROWTYPE;
 nuevo vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1%ROWTYPE;
 total_anterior integer;
 total_nuevo integer;
 login_o oid;
BEGIN
 LOCK TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 IN SHARE ROW EXCLUSIVE MODE;
 SELECT count(*) INTO total_anterior FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec_contratacion_temporal.confirmar_alta_atestada.v1'
   AND operacion='contratacion_temporal.llamamiento.reanudar_orden'
   AND canal_permitido='interna_corporativa';
 IF total_anterior IS DISTINCT FROM 1 THEN
  RAISE EXCEPTION 'AD226: PARO clave=origen_previo actual=% esperado=1',coalesce(total_anterior,0) USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT anterior FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec_contratacion_temporal.confirmar_alta_atestada.v1'
   AND operacion='contratacion_temporal.llamamiento.reanudar_orden'
   AND canal_permitido='interna_corporativa';
 SELECT oid INTO login_o FROM pg_roles WHERE rolname=anterior.login_nombre
   AND rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls;
 IF login_o IS NULL OR NOT pg_has_role(login_o,'vec_contratacion_temporal_ejecutor','MEMBER') THEN
  RAISE EXCEPTION 'AD226: PARO clave=login_origen actual=no_acreditado esperado=LOGIN_CT_ejecutor' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO total_nuevo FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE operacion='contratacion_temporal.llamamiento.reanudar_solicitud';
 IF total_nuevo=0 THEN
  INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido,configurada_en)
  SELECT login_nombre,audiencia_consumo,'contratacion_temporal.llamamiento.reanudar_solicitud',
         proceso,canal_permitido,clock_timestamp()
  FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
  WHERE login_nombre=anterior.login_nombre
    AND audiencia_consumo=anterior.audiencia_consumo
    AND operacion=anterior.operacion;
 ELSIF total_nuevo=1 THEN
  SELECT * INTO STRICT nuevo FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
  WHERE operacion='contratacion_temporal.llamamiento.reanudar_solicitud';
  IF nuevo.login_nombre IS DISTINCT FROM anterior.login_nombre
     OR nuevo.audiencia_consumo IS DISTINCT FROM anterior.audiencia_consumo
     OR nuevo.proceso IS DISTINCT FROM anterior.proceso
     OR nuevo.canal_permitido IS DISTINCT FROM anterior.canal_permitido THEN
   RAISE EXCEPTION 'AD226: PARO clave=origen_nuevo actual=divergente esperado=misma_ligadura_origen_previo' USING ERRCODE='55000';
  END IF;
 ELSE
  RAISE EXCEPTION 'AD226: PARO clave=origen_nuevo actual=% esperado=0_o_1',total_nuevo USING ERRCODE='55000';
 END IF;
END
$origen$;
RESET ROLE;
COMMIT;
