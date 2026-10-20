\set ON_ERROR_STOP on
-- AD304: ejecutar en PostgreSQL 18 desechable tras instalar el UP; ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $prueba$
DECLARE seis record; siete record; v integer; rechazado boolean; tabla record; def text;
BEGIN
 SELECT * INTO seis FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=6;
 SELECT * INTO siete FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=7;
 IF NOT FOUND OR cardinality(seis.audiencias)<>10 OR cardinality(siete.audiencias)<>12
 OR cardinality(siete.segmentos)<>12
 OR siete.audiencias IS DISTINCT FROM seis.audiencias||ARRAY[
  'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1',
  'vec_autorizacion.gobierno_rol_nuevo.cierre.v1']::text[]
 OR siete.segmentos IS DISTINCT FROM seis.segmentos||ARRAY[
  'perfiles:gobierno-rol-nuevo:propuesta',
  'perfiles:gobierno-rol-nuevo:cierre']::text[]
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>7
 THEN RAISE EXCEPTION 'AD304 prueba: fila 7 o prefijo 6 divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO tabla FROM pg_class WHERE oid='vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass;
 SELECT pg_get_constraintdef(c.oid,true) INTO def FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF tabla.relowner IS DISTINCT FROM to_regrole('vec_autorizacion_atestada_v3_propietario')
 OR tabla.relrowsecurity OR tabla.relforcerowsecurity
 OR EXISTS(SELECT 1 FROM aclexplode(coalesce(tabla.relacl,acldefault('r',tabla.relowner))) a
           WHERE a.grantee<>tabla.relowner)
 OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=tabla.oid AND NOT t.tgisinternal)<>2
 OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=tabla.oid AND t.tgenabled='O'
     AND t.tgname IN('inmutable','no_truncar'))<>2
 -- Dos preimágenes exactas: main y la candidata AD301 revisada.
 OR def IS NULL OR encode(sha256(convert_to(def,'UTF8')),'hex') NOT IN (
  '411555a7d3a6e423d1c3538236a7c3da20b0b8b7280a1b07151eaf6cffb2efa4',
  'b8a3f4abaebb452c245e2e33e191f776cea16a3e393ca11ae6ddf8c1901f10c8')
 THEN RAISE EXCEPTION 'AD304 prueba: clases o permisos ampliados' USING ERRCODE='42501'; END IF;
 FOR v IN 6..7 LOOP
  rechazado:=false;
  BEGIN
   UPDATE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1
   SET audiencias=audiencias WHERE version=v;
  EXCEPTION WHEN SQLSTATE '55000' THEN rechazado:=true;
  END;
  IF NOT rechazado THEN RAISE EXCEPTION 'AD304 prueba: UPDATE admitido en fila %',v; END IF;
  rechazado:=false;
  BEGIN
   DELETE FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=v;
  EXCEPTION WHEN SQLSTATE '55000' THEN rechazado:=true;
  END;
  IF NOT rechazado THEN RAISE EXCEPTION 'AD304 prueba: DELETE admitido en fila %',v; END IF;
 END LOOP;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>7
 THEN RAISE EXCEPTION 'AD304 prueba: historia del catálogo cambió' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
