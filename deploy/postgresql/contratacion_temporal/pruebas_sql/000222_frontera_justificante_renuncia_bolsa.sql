\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE f oid := 'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure;
 v_ruta constant text := '/api/vec/bolsa/justificantes-renuncia/rrhh';
 motivo text; rechazado boolean; anteriores bigint;
BEGIN
 IF (SELECT p.proowner FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT p.proconfig FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s']
    OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_registrador_frontera=X/vec_contratacion_temporal_propietario}'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex')
      FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
       AND c.conname='auditoria_frontera_ruta_exacta_motivo_check') IS DISTINCT FROM
      'cae40bfa0378f3c4baf62c55d367fc6a51c1d53c16b12ebe66e138600021da2e'
 THEN RAISE EXCEPTION 'CT222: dueño, configuración, ACL o motivo cambiados' USING ERRCODE='55000'; END IF;
 IF pg_get_functiondef(f) NOT LIKE '%'||v_ruta||'%'
    OR (SELECT pg_get_constraintdef(c.oid,true) FROM pg_constraint c
       WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
        AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check') NOT LIKE '%'||v_ruta||'%'
 THEN RAISE EXCEPTION 'CT222: ruta nueva no instalada en función y CHECK' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO anteriores FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta a
 WHERE a.ruta=v_ruta AND a.superficie='api.bolsa.rrhh.documentos.ruta_exacta';
 FOREACH motivo IN ARRAY ARRAY['autenticacion_requerida','acceso_denegado','servicio_no_disponible'] LOOP
  PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
   'corr_'||repeat(CASE motivo WHEN 'autenticacion_requerida' THEN 'a' WHEN 'acceso_denegado' THEN 'b' ELSE 'c' END,32),motivo,
   'api.bolsa.rrhh.documentos.ruta_exacta',v_ruta,NULL);
 END LOOP;
 IF (SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta a
   WHERE a.ruta=v_ruta AND a.superficie='api.bolsa.rrhh.documentos.ruta_exacta')<>anteriores+3
 THEN RAISE EXCEPTION 'CT222: falta recibo de un motivo exacto' USING ERRCODE='55000'; END IF;

 rechazado:=false;
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
   'corr_'||repeat('f',32),'acceso_denegado','api.bolsa.rrhh.documentos.ruta_exacta',
   '/api/vec/bolsa/justificantes-renuncia/rrhh-ajena',NULL);
 EXCEPTION WHEN SQLSTATE '22023' THEN rechazado:=true; END;
 IF NOT rechazado THEN RAISE EXCEPTION 'CT222: ruta parecida admitida' USING ERRCODE='55000'; END IF;

 rechazado:=false;
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
   'corr_'||repeat('e',32),'acceso_denegado','api.bolsa.rrhh.documentos.ruta_exacta',
   v_ruta,'per_1234567890123456789012');
 EXCEPTION WHEN SQLSTATE '22023' THEN rechazado:=true; END;
 IF NOT rechazado THEN RAISE EXCEPTION 'CT222: actor en ruta nueva admitido' USING ERRCODE='55000'; END IF;

 PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
  'corr_'||repeat('d',32),'acceso_denegado','api.bolsa.rrhh.documentos.ruta_exacta',
  '/api/vec/bolsa/bolsas/bolsa:administrativo/documentos',NULL);
END $prueba$;
ROLLBACK;
