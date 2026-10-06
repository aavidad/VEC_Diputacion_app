\set ON_ERROR_STOP on
-- CT183: accesos RRHH sin bloqueo común, cola, sellado, verificación, control
-- congelado y plazo. Ejecutar como superusuario migrador después de CT183 en
-- una base con algún acceso registrado. Termina en ROLLBACK; solo consume
-- números de la secuencia.
BEGIN;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='120s';
DO $estructura$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid IN (
   'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)'::regprocedure,
   'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'::regprocedure)
   AND (p.prosrc ~ 'FOR\s+UPDATE' OR p.prosrc ~ 'UPDATE\s+vec_contratacion_temporal\.control_cadena_accesos_rrhh'
    OR p.prosrc !~ 'reservar_acceso_rrhh_v1\(\) asiento_ct183'))
 OR (SELECT proacl FROM pg_proc WHERE oid='vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(integer)'::regprocedure)
  IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
   'vec_auditoria_encadenador=X/vec_contratacion_temporal_propietario']::aclitem[]
 OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.registro_acceso_rrhh)
 THEN RAISE EXCEPTION 'CT183 prueba: estructura o permisos inesperados'; END IF;
END $estructura$;
UPDATE vec_contratacion_temporal.sellado_acceso_rrhh_v1 SET latido=clock_timestamp();

-- Acceso sintético copiado del último, con claves únicas nuevas, el número
-- de la reserva y la huella que exige la CHECK de la tabla.
CREATE FUNCTION pg_temp.ct183_fila(g text,p_anterior text) RETURNS vec_contratacion_temporal.registro_acceso_rrhh
LANGUAGE plpgsql AS $f$
DECLARE base jsonb;r record;consumo text;prueba bytea;anterior text;
BEGIN
 SELECT to_jsonb(a) INTO STRICT base FROM vec_contratacion_temporal.registro_acceso_rrhh a ORDER BY a.secuencia DESC LIMIT 1;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.reservar_acceso_rrhh_v1();
 anterior:=coalesce(p_anterior,r.anterior_sha256);
 consumo:=encode(sha256(convert_to('ct183-consumo-'||g||clock_timestamp(),'UTF8')),'hex');
 prueba:=decode(substr(base->>'prueba_canonica',3),'hex')||convert_to('ct183-'||g||clock_timestamp(),'UTF8');
 RETURN jsonb_populate_record(NULL::vec_contratacion_temporal.registro_acceso_rrhh,base||jsonb_build_object(
  'secuencia',r.secuencia,'anterior_sha256',anterior,
  'acceso_ref','acceso:rrhh:'||substr(encode(sha256(convert_to('acceso:rrhh:'||consumo,'UTF8')),'hex'),1,32),
  'consumo_vec_huella_sha256',consumo,
  'decision_ref','decision:ct183:'||g||':'||md5(consumo),'correlacion_ref','correlacion:ct183:'||md5(consumo),
  'auditoria_vec_ref','auditoria:ct183:'||md5(consumo),
  'auditoria_vec_huella_sha256',encode(sha256(convert_to('a'||consumo,'UTF8')),'hex'),
  'capacidad_huella_sha256',encode(sha256(convert_to('c'||consumo,'UTF8')),'hex'),
  'decision_huella_sha256',encode(sha256(convert_to('d'||consumo,'UTF8')),'hex'),
  'prueba_canonica','\x'||encode(prueba,'hex'),
  'huella_sha256',encode(sha256(decode(anterior,'hex')||prueba),'hex'),
  'registrada_en',to_jsonb(date_trunc('microseconds',clock_timestamp()))));
END $f$;
CREATE FUNCTION pg_temp.ct183_insertar(f vec_contratacion_temporal.registro_acceso_rrhh) RETURNS void
LANGUAGE plpgsql AS $f$
DECLARE columnas text;
BEGIN
 SELECT string_agg(quote_ident(attname),',' ORDER BY attnum) INTO STRICT columnas FROM pg_attribute
  WHERE attrelid='vec_contratacion_temporal.registro_acceso_rrhh'::regclass AND attnum>0 AND NOT attisdropped AND attgenerated='';
 EXECUTE format('INSERT INTO vec_contratacion_temporal.registro_acceso_rrhh(%s) SELECT %s FROM (SELECT ($1).*) x',columnas,columnas) USING f;
END $f$;
CREATE TEMP TABLE ct183_nuevos(secuencia numeric) ON COMMIT DROP;
DO $accesos$
DECLARE f vec_contratacion_temporal.registro_acceso_rrhh;
BEGIN
 FOR g IN 1..2 LOOP
  f:=pg_temp.ct183_fila(g::text,NULL);
  PERFORM pg_temp.ct183_insertar(f);
  INSERT INTO ct183_nuevos VALUES (f.secuencia);
 END LOOP;
 IF (SELECT count(*) FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 p JOIN ct183_nuevos USING (secuencia))<>2 THEN
  RAISE EXCEPTION 'CT183 prueba: los accesos no quedan en la cola'; END IF;
 BEGIN
  UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh SET actualizada_en=clock_timestamp();
  RAISE EXCEPTION 'CT183 prueba: control no congelado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
 -- Un registrador antiguo enlazaría con la cabeza: se rechaza.
 BEGIN
  f:=pg_temp.ct183_fila('antiguo',(SELECT cabeza_sha256 FROM vec_contratacion_temporal.control_cadena_accesos_rrhh));
  PERFORM pg_temp.ct183_insertar(f);
  RAISE EXCEPTION 'CT183 prueba: acceso sin marcador admitido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN
  IF SQLERRM<>'acceso RRHH fuera de la cadena con sellado diferido' THEN RAISE; END IF;
 END;
END $accesos$;

DO $sellado$
DECLARE v jsonb;n integer;
BEGIN
 n:=vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(50000);
 v:=vec_contratacion_temporal.verificar_cadena_accesos_rrhh_v1();
 IF n<2 OR v->>'estado'<>'verificada' OR (v->>'pendientes')::int<>0
 OR (SELECT count(*) FROM vec_contratacion_temporal.eslabon_acceso_rrhh_v1 e JOIN ct183_nuevos USING (secuencia))<>2 THEN
  RAISE EXCEPTION 'CT183 prueba: sellado o verificación %',v; END IF;
 BEGIN
  UPDATE vec_contratacion_temporal.eslabon_acceso_rrhh_v1 SET sellado_en=clock_timestamp()
   WHERE secuencia=(SELECT max(secuencia) FROM ct183_nuevos);
  RAISE EXCEPTION 'CT183 prueba: eslabón modificable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
END $sellado$;

-- Plazo: con el latido caducado no se registra ningún acceso.
UPDATE vec_contratacion_temporal.sellado_acceso_rrhh_v1 SET latido=now()-interval '1 hour';
DO $plazo$
DECLARE f vec_contratacion_temporal.registro_acceso_rrhh;
BEGIN
 f:=pg_temp.ct183_fila('plazo',NULL);
 BEGIN
  PERFORM pg_temp.ct183_insertar(f);
  RAISE EXCEPTION 'CT183 prueba: acceso admitido con el sellado detenido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN
  IF SQLERRM<>'sellado de accesos RRHH detenido: acceso rechazado' THEN RAISE; END IF;
 END;
END $plazo$;
SELECT 'CT183 prueba: correcta' AS resultado;
ROLLBACK;
