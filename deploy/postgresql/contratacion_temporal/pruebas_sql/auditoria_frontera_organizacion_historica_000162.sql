-- Parte del ensayo transaccional CT162: ejecutar después del UP nuevo.
-- El conductor captura CT108 antes, abre BEGIN y acaba con ROLLBACK.
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contrato$
DECLARE f oid:='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
     AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
     AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR NOT has_function_privilege('vec_contratacion_temporal_registrador_frontera',f,'EXECUTE')
    OR has_table_privilege('vec_contratacion_temporal_registrador_frontera','vec_contratacion_temporal.auditoria_frontera_ruta_exacta','SELECT')
    OR has_table_privilege('vec_contratacion_temporal_registrador_frontera','vec_contratacion_temporal.auditoria_frontera_ruta_exacta','INSERT') THEN
  RAISE EXCEPTION 'CT162: contrato o ACL no mínimos';
 END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
    WHERE correlacion_ref='corr_16216216216216216216216216216216') THEN
  RAISE EXCEPTION 'CT162: correlación sintética ya utilizada';
 END IF;
END $contrato$;

-- RESTART es transaccional: también los nextval de este ejercicio se revierten
-- junto al UP. No se usa setval ni se reinicia PostgreSQL.
ALTER SEQUENCE vec_contratacion_temporal.auditoria_frontera_ruta_exacta_evento_id_seq
 RESTART WITH 900000000000000000;
SET LOCAL ROLE vec_contratacion_temporal_registrador_frontera;
DO $registrador$
DECLARE x record; actor text; total integer:=0; negativos integer:=0;
BEGIN
 FOR x IN SELECT * FROM (VALUES
   ('api.contratacion_temporal.ruta_exacta','/api/vec/contratacion-temporal/expedientes','autenticacion_requerida',NULL::text),
   ('api.contratacion_temporal.ruta_exacta','/api/vec/contratacion-temporal/expedientes/analisis','acceso_denegado','actor:ct162:sintetico'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica','autenticacion_requerida',NULL::text),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica','acceso_denegado','actor:ct162:sintetico'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica','acceso_denegado','x'),
   ('api.contratacion_temporal.ruta_exacta','/api/vec/contratacion-temporal/expedientes','acceso_denegado',repeat('a',512))
 ) v(superficie,ruta,motivo,actor) LOOP
  IF vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
      'corr_16216216216216216216216216216216',x.motivo,x.superficie,x.ruta,x.actor) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT162: registro positivo no confirmado';
  END IF;
  total:=total+1;
 END LOOP;
 FOR x IN SELECT * FROM (VALUES
   ('organizacion_historica_personal','/api/vec/contratacion-temporal/expedientes','acceso_denegado'),
   ('api.contratacion_temporal.ruta_exacta','/api/vec/personal/organizacion-historica','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica/','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica?organismo_ref=x','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica/empleados','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/registro-empleado','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/%6frganizacion-historica','acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica','otro'),
   (NULL::text,'/api/vec/personal/organizacion-historica','acceso_denegado'),
   ('organizacion_historica_personal',NULL::text,'acceso_denegado'),
   ('organizacion_historica_personal','/api/vec/personal/organizacion-historica',NULL::text)
 ) v(superficie,ruta,motivo) LOOP
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
    'corr_16216216216216216216216216216216',x.motivo,x.superficie,x.ruta,NULL);
   RAISE EXCEPTION 'CT162: material ajeno autorizado';
  EXCEPTION WHEN invalid_parameter_value THEN negativos:=negativos+1;
  END;
 END LOOP;
 FOREACH actor IN ARRAY ARRAY['',repeat('a',513),'actor con espacio'] LOOP
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
    'corr_16216216216216216216216216216216','acceso_denegado','organizacion_historica_personal',
    '/api/vec/personal/organizacion-historica',actor);
   RAISE EXCEPTION 'CT162: actor inválido autorizado';
  EXCEPTION WHEN invalid_parameter_value THEN negativos:=negativos+1;
  END;
 END LOOP;
 IF total<>6 OR negativos<>14 THEN RAISE EXCEPTION 'CT162: casos incompletos'; END IF;
 BEGIN
  PERFORM 1 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta;
  RAISE EXCEPTION 'CT162: registrador leyó auditoría';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $registrador$;
RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $inmutable$
BEGIN
 BEGIN
  UPDATE vec_contratacion_temporal.auditoria_frontera_ruta_exacta SET motivo='acceso_denegado'
   WHERE correlacion_ref='corr_16216216216216216216216216216216';
  RAISE EXCEPTION 'CT162: auditoría modificada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  DELETE FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
   WHERE correlacion_ref='corr_16216216216216216216216216216216';
  RAISE EXCEPTION 'CT162: auditoría borrada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  TRUNCATE vec_contratacion_temporal.auditoria_frontera_ruta_exacta;
  RAISE EXCEPTION 'CT162: auditoría truncada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  INSERT INTO vec_contratacion_temporal.auditoria_frontera_ruta_exacta
   (correlacion_ref,motivo,superficie,ruta,registrada_en)
  VALUES('corr_16216216216216216216216216216216','acceso_denegado',
   'organizacion_historica_personal','/api/vec/contratacion-temporal/expedientes',clock_timestamp());
  RAISE EXCEPTION 'CT162: CHECK permitió pareja cruzada';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
END $inmutable$;
RESET ROLE;
DO $conservacion$
DECLARE filas text; originales text; n bigint;
BEGIN
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') INTO filas
 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t
 WHERE t.correlacion_ref<>'corr_16216216216216216216216216216216';
 SELECT huella_filas INTO STRICT originales FROM ct162_preimagen;
 SELECT count(*) INTO n FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 WHERE correlacion_ref='corr_16216216216216216216216216216216';
 IF filas IS DISTINCT FROM originales OR n<>6 THEN RAISE EXCEPTION 'CT162: historia CT alterada'; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
     WHERE correlacion_ref='corr_16216216216216216216216216216216'
       AND superficie='organizacion_historica_personal'
       AND ruta='/api/vec/personal/organizacion-historica')<>3 THEN
  RAISE EXCEPTION 'CT162: 401/403 OH no conservados';
 END IF;
END $conservacion$;
SELECT 'CT162-ACL-CT-OH401403-LIMITES-ACTOR-14NEGATIVAS-INMUTABILIDAD-OK' AS resultado;
