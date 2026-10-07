\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL row_security=on;
DO $prueba$
DECLARE tipada text:='expediente:ct:'||repeat('a',64);
        historica text:='ref:'||repeat('b',64);
        mala text;
        f text;
BEGIN
 IF NOT vec_documentos.referencia_expediente_v1(tipada)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',tipada)
    OR NOT vec_documentos.referencia_expediente_v1(historica)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('dietas',historica)
    OR vec_documentos.referencia_opaca_v1(tipada)
    OR vec_documentos.referencia_expediente_modulo_v1('bolsa',tipada)
 THEN RAISE EXCEPTION 'Documentos-14: positivo o aislamiento de ID falló'; END IF;
 FOREACH mala IN ARRAY ARRAY[
   'expediente:bolsa:'||repeat('a',64),
   'expediente:CT:'||repeat('a',64),
   'Expediente:ct:'||repeat('a',64),
   'expediente:ct:'||repeat('A',64),
   'expediente:ct:'||repeat('a',63),
   'expediente:ct:'||repeat('a',65),
   'expediente:ct:'||repeat('a',64)||':extra',
   'expediente:ct:../'||repeat('a',64)
 ] LOOP
  IF vec_documentos.referencia_expediente_v1(mala) THEN
   RAISE EXCEPTION 'Documentos-14: aceptó %',mala; END IF;
 END LOOP;
 IF (SELECT count(*) FROM vec_documentos.modulo_referencia_expediente_v1)<>1
    OR NOT EXISTS (SELECT 1 FROM vec_documentos.modulo_referencia_expediente_v1
                   WHERE codigo='ct' AND modulo_id='contratacion_temporal'
                     AND version=1 AND patron_id='^[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'Documentos-14: catálogo divergente'; END IF;
 PERFORM set_config('vec.documentos.expediente_ref',tipada,true);
 INSERT INTO vec_documentos.referencia_externa(
   id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
   huella_sha256,custodio_id,custodia_ref,politica_ref,version_politica,
   huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
   huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
 VALUES('ref:'||repeat('1',64),'VEC-2026-999999','ref:'||repeat('2',64),
   'ref:'||repeat('3',64),'contratacion_temporal',tipada,'ref:'||repeat('4',64),1,
   repeat('a',64),'sistema.prueba','justificante:1','ref:'||repeat('5',64),1,
   repeat('b',64),clock_timestamp()+interval '1 day','conservacion','aprobada',
   repeat('c',64),'decision:prueba','auditoria:prueba',clock_timestamp());
 IF (SELECT count(*) FROM vec_documentos.referencia_externa WHERE expediente_ref=tipada)<>1
 THEN RAISE EXCEPTION 'Documentos-14: referencia CT no quedó consultable'; END IF;
 BEGIN
  INSERT INTO vec_documentos.referencia_externa(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
    huella_sha256,custodio_id,custodia_ref,politica_ref,version_politica,
    huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES('ref:'||repeat('6',64),'VEC-2026-999998','ref:'||repeat('7',64),
    'ref:'||repeat('3',64),'bolsa',tipada,'ref:'||repeat('4',64),1,
    repeat('a',64),'sistema.prueba','justificante:2','ref:'||repeat('5',64),1,
    repeat('b',64),clock_timestamp()+interval '1 day','conservacion','aprobada',
    repeat('c',64),'decision:prueba','auditoria:prueba',clock_timestamp());
  RAISE EXCEPTION 'Documentos-14: aceptó CT con módulo distinto';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
 FOREACH f IN ARRAY ARRAY[
   'vec_documentos.obtener_original_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_documentos.preparar_notificacion_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_documentos.preparar_notificacion_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
 ] LOOP
  IF strpos(pg_get_functiondef(to_regprocedure(f)),'vec_documentos.referencia_expediente_')=0
  THEN RAISE EXCEPTION 'Documentos-14: función no adaptada %',f; END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
