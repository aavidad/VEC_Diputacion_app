\set ON_ERROR_STOP on
-- Sólo en el clon sintético dirigido, después del UP nuevo, dentro de la
-- misma transacción de ensayo. El conductor termina con ROLLBACK.
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
CREATE TEMP TABLE ct168_prueba_preimagen AS
 SELECT count(*) AS filas,
 encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(x) ORDER BY evento_id)::text,'[]'),'UTF8')),'hex') AS huella
 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta x;
DO $pre$
DECLARE f oid:='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_registrador_frontera',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_reglas_baremo_ejecutor_gobierno',f,'EXECUTE')
 OR has_table_privilege('vec_contratacion_temporal_registrador_frontera','vec_contratacion_temporal.auditoria_frontera_ruta_exacta','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
   WHERE correlacion_ref='corr_16816816816816816816816816816816') THEN
  RAISE EXCEPTION 'CT168: ACL o correlación de ensayo incompatible';
 END IF;
END $pre$;
-- RESTART pertenece a esta transacción y ROLLBACK restaura la secuencia.
ALTER SEQUENCE vec_contratacion_temporal.auditoria_frontera_ruta_exacta_evento_id_seq RESTART WITH 900000000000000000;
SET LOCAL ROLE vec_contratacion_temporal_registrador_frontera;
DO $ejercicio$
DECLARE x record;positivos integer:=0;negativos integer:=0;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('api.contratacion_temporal.ruta_exacta','/api/vec/contratacion-temporal/expedientes','acceso_denegado','actor:ct168:sintetico'),
  ('organizacion_historica_personal','/api/vec/personal/organizacion-historica','autenticacion_requerida',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar','acceso_denegado',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/consultar','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/borradores/alta','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/versiones/consultar','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/recibos/recuperar','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/borradores/alta','autenticacion_requerida',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/versiones/consultar','autenticacion_requerida',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/recibos/recuperar','autenticacion_requerida',NULL::text)
 ) v(superficie,ruta,motivo,actor) LOOP
  IF vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
   'corr_16816816816816816816816816816816',x.motivo,x.superficie,x.ruta,x.actor) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT168: registro no confirmado';
  END IF;
  positivos:=positivos+1;
 END LOOP;
 FOR x IN SELECT * FROM (VALUES
  ('api.contratacion_temporal.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar','acceso_denegado',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/bolsa/reglas-baremo/borradores/alta','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/seleccion/preparacion-bases/consultar','acceso_denegado',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar/','acceso_denegado',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar?actor=x','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/versiones','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/recibos/recuperar/detalle','acceso_denegado',NULL::text),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/recibos/%72ecuperar','acceso_denegado',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar','autenticacion_requerida',NULL::text),
  ('api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/consultar','acceso_denegado','actor:libre'),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/borradores/alta','acceso_denegado','actor:libre'),
  ('api.bolsa.reglas_baremo.ruta_exacta','/api/vec/bolsa/reglas-baremo/versiones/consultar','csrf_detalle',NULL::text)
 ) v(superficie,ruta,motivo,actor) LOOP
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
    'corr_16816816816816816816816816816816',x.motivo,x.superficie,x.ruta,x.actor);
   RAISE EXCEPTION 'CT168: pareja, causa o actor ajenos aceptados';
  EXCEPTION WHEN invalid_parameter_value THEN negativos:=negativos+1; END;
 END LOOP;
 IF positivos<>10 OR negativos<>12 THEN RAISE EXCEPTION 'CT168: ejercicio incompleto'; END IF;
 BEGIN
  PERFORM 1 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta;
  RAISE EXCEPTION 'CT168: registrador leyó historia';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ejercicio$;
RESET ROLE;
DO $conservacion$
DECLARE originales text;actuales text;n bigint;
BEGIN
 SELECT huella INTO STRICT originales FROM ct168_prueba_preimagen;
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(x) ORDER BY evento_id)::text,'[]'),'UTF8')),'hex')
 INTO actuales FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta x
 WHERE correlacion_ref<>'corr_16816816816816816816816816816816';
 SELECT count(*) INTO n FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 WHERE correlacion_ref='corr_16816816816816816816816816816816';
 IF originales IS DISTINCT FROM actuales OR n<>10 THEN RAISE EXCEPTION 'CT168: historia alterada'; END IF;
 BEGIN
  UPDATE vec_contratacion_temporal.auditoria_frontera_ruta_exacta SET motivo='acceso_denegado'
  WHERE correlacion_ref='corr_16816816816816816816816816816816';
  RAISE EXCEPTION 'CT168: auditoría mutable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  INSERT INTO vec_contratacion_temporal.auditoria_frontera_ruta_exacta
   (correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
  VALUES('corr_16816816816816816816816816816816','acceso_denegado',
   'api.seleccion.preparacion_bases.ruta_exacta','/api/vec/seleccion/preparacion-bases/guardar','actor:libre',clock_timestamp());
  RAISE EXCEPTION 'CT168: CHECK aceptó actor en Selección';
 EXCEPTION WHEN check_violation THEN NULL; END;
END $conservacion$;
SELECT 'CT168-10POSITIVOS-12NEGATIVOS-ACL-HISTORIA-OK' AS resultado;
