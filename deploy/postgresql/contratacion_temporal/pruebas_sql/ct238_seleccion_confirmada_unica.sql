\set ON_ERROR_STOP on
-- Copia ficticia de la principal, tras CT238. Todo se revierte.
-- Prueba la restricción física; estas filas no acreditan actos ni consumo V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
SET LOCAL enable_seqscan=off;
DO $test$ DECLARE
 original vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 copia jsonb;solicitud jsonb;huella text;clave uuid;restriccion text;plan json;
 org text;exp text;n integer;espera_rechazo boolean;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_index i
   WHERE i.indexrelid='vec_contratacion_temporal.seleccion_confirmada_unica_expediente_ct238'::regclass
     AND i.indisunique AND i.indisvalid AND i.indisready)
 THEN RAISE EXCEPTION 'CT238: índice no disponible'; END IF;
 SELECT * INTO STRICT original
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE tipo_seleccion='llamamiento' AND situacion='confirmada'
 ORDER BY clave_idempotencia LIMIT 1;
 org:=original.solicitud_json->>'organizacion_ref';
 exp:=original.solicitud_json->>'expediente_ref';

 -- Una segunda intención singular no puede confirmar el mismo expediente.
 clave:=gen_random_uuid();
 solicitud:=jsonb_set(original.solicitud_json,'{clave_idempotencia}',to_jsonb(clave::text));
 huella:=vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(solicitud);
 solicitud:=jsonb_set(solicitud,'{huella_semantica}',to_jsonb(huella));
 copia:=to_jsonb(original)||jsonb_build_object('clave_idempotencia',clave,
   'solicitud_json',solicitud,'huella_semantica',huella,
   'reserva_ref',vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2());
 BEGIN
  INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
   SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6,copia)).*;
  RAISE EXCEPTION 'CT238: segunda selección singular admitida';
 EXCEPTION WHEN unique_violation THEN
  GET STACKED DIAGNOSTICS restriccion=CONSTRAINT_NAME;
  IF restriccion<>'seleccion_confirmada_unica_expediente_ct238'
  THEN RAISE EXCEPTION 'CT238: rechazo por otra restricción: %',restriccion; END IF;
 END;

 -- La oferta comparte el límite; otra organización u otro expediente no chocan.
 FOR n IN 1..3 LOOP
  clave:=gen_random_uuid();espera_rechazo:=n=1;
  solicitud:=jsonb_build_object(
   'organizacion_ref',CASE WHEN n=3 THEN 'organizacion:ct238:otra' ELSE org END,
   'expediente_ref',CASE WHEN n=2 THEN 'expediente:ct238:otro' ELSE exp END,
   'version_expediente',1,'clave_idempotencia',clave::text,
   'actor_ref','per_'||repeat('a',22),'perfil_ref','perfil:ct238',
   'categoria_ref','categoria:ct238','centro_ref','centro:ct238','huella_intencion',repeat('a',64));
  huella:=vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(solicitud);
  solicitud:=jsonb_set(solicitud,'{huella_intencion}',to_jsonb(huella));
  BEGIN
   INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
    clave_idempotencia,huella_semantica,solicitud_json,reserva_ref,
    situacion,ventana_orden_abierta,recibo_json,tipo_seleccion,comando_oferta_json)
   VALUES(clave,huella,solicitud,'reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(clave::text,'UTF8')),'hex'),
    'confirmada',true,'{}','oferta','{}');
   IF espera_rechazo THEN RAISE EXCEPTION 'CT238: oferta y llamamiento confirmados juntos'; END IF;
  EXCEPTION WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS restriccion=CONSTRAINT_NAME;
   IF NOT espera_rechazo OR restriccion<>'seleccion_confirmada_unica_expediente_ct238'
   THEN RAISE EXCEPTION 'CT238: rechazo inesperado: %',restriccion; END IF;
  END;
 END LOOP;

 -- Recuperar la misma selección no crea otra fila ni cambia su recibo.
 UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET situacion='confirmada' WHERE clave_idempotencia=original.clave_idempotencia;
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   WHERE e.clave_idempotencia=original.clave_idempotencia
     AND e.recibo_json=original.recibo_json AND e.artefacto_canonico=original.artefacto_canonico)
 THEN RAISE EXCEPTION 'CT238: recibo original alterado'; END IF;

 -- Coteja la elegibilidad del índice para el mismo filtro de CT231.
 EXECUTE 'EXPLAIN (FORMAT JSON) SELECT e.clave_idempotencia
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 WHERE e.situacion=''confirmada'' AND e.solicitud_json->>''organizacion_ref''=$1
 AND e.solicitud_json->>''expediente_ref''=$2
 AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
 WHERE x.seleccion_clave=e.clave_idempotencia)' INTO plan USING org,exp;
 IF position('seleccion_confirmada_unica_expediente_ct238' IN plan::text)=0
 THEN RAISE EXCEPTION 'CT238: filtro CT231 sin índice'; END IF;
END $test$;
ROLLBACK;
