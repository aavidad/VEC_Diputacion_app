\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL row_security=on;
DO $prueba$
DECLARE cero text:='expediente:ct:'||repeat('0',64);
        uno text:='expediente:ct:'||repeat('0',63)||'1';
        otro text:='expediente:ct:'||repeat('a',64);
        generica text:='ref:'||repeat('b',64);
        generica_cero text:='ref:'||repeat('0',64);
        firma text; huella text;
BEGIN
 IF vec_documentos.referencia_expediente_v1(cero)
    OR vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',cero)
    OR NOT vec_documentos.referencia_expediente_v1(uno)
    OR NOT vec_documentos.referencia_expediente_v1(otro)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',uno)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',otro)
    OR vec_documentos.referencia_expediente_modulo_v1('bolsa',uno)
    OR vec_documentos.referencia_expediente_v1(generica)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica)
    OR vec_documentos.referencia_expediente_modulo_v1('dietas',generica)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica)
    OR vec_documentos.referencia_expediente_v1(generica_cero)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica_cero)
    OR vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',generica_cero)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica_cero)
    OR NOT vec_documentos.referencia_expediente_formato_v1('contratacion_temporal',cero)
 THEN RAISE EXCEPTION 'Documentos-15: predicado tipado u opaco divergente'; END IF;

 FOR firma,huella IN SELECT * FROM (VALUES
   ('vec_documentos.referencia_expediente_v1(text)',
    '5611172df952932366ee49c107541010869e5e64e256058d4e0d9e0bc6c5fb90'),
   ('vec_documentos.referencia_expediente_modulo_v1(text,text)',
    'ce27fb14b14eab2530fd79b8adf8d15d3bc42d8fa3ee54b03ab646ec8682f218')
 ) AS v(firma,huella) LOOP
  IF (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
      FROM pg_proc p WHERE p.oid=to_regprocedure(firma)) IS DISTINCT FROM huella
  THEN RAISE EXCEPTION 'Documentos-15: cuerpo posterior divergente %',firma; END IF;
 END LOOP;
 IF (SELECT count(*) FROM vec_documentos.modulo_referencia_expediente_v1)<>1
    OR (SELECT count(*) FROM pg_trigger t WHERE t.tgname='validar_expediente_tipado'
       AND NOT t.tgisinternal AND t.tgrelid IN (
         'vec_documentos.documento'::regclass,
         'vec_documentos.referencia_externa'::regclass,
         'vec_documentos.reserva_original_firmable'::regclass))<>3
 THEN RAISE EXCEPTION 'Documentos-15: catálogo o trigger cambió'; END IF;

 INSERT INTO vec_documentos.referencia_externa(
   id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
   huella_sha256,custodio_id,custodia_ref,politica_ref,version_politica,
   huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
   huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
 VALUES('ref:'||repeat('8',64),'VEC-2026-999997','ref:'||repeat('9',64),
   'ref:'||repeat('3',64),'contratacion_temporal',uno,'ref:'||repeat('4',64),1,
   repeat('a',64),'sistema.prueba','justificante:3','ref:'||repeat('5',64),1,
   repeat('b',64),clock_timestamp()+interval '1 day','conservacion','aprobada',
   repeat('c',64),'decision:prueba','auditoria:prueba',clock_timestamp());
 IF (SELECT count(*) FROM vec_documentos.referencia_externa
     WHERE id='ref:'||repeat('8',64) AND expediente_ref=uno)<>1
 THEN RAISE EXCEPTION 'Documentos-15: positivo tipado no quedó consultable'; END IF;

 BEGIN
  INSERT INTO vec_documentos.referencia_externa(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
    huella_sha256,custodio_id,custodia_ref,politica_ref,version_politica,
    huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES('ref:'||repeat('6',64),'VEC-2026-999996','ref:'||repeat('7',64),
    'ref:'||repeat('3',64),'contratacion_temporal',cero,'ref:'||repeat('4',64),1,
    repeat('a',64),'sistema.prueba','justificante:4','ref:'||repeat('5',64),1,
    repeat('b',64),clock_timestamp()+interval '1 day','conservacion','aprobada',
    repeat('c',64),'decision:prueba','auditoria:prueba',clock_timestamp());
  RAISE EXCEPTION 'Documentos-15: INSERT aceptó cero tipado';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
 IF EXISTS (SELECT 1 FROM vec_documentos.referencia_externa WHERE id='ref:'||repeat('6',64))
 THEN RAISE EXCEPTION 'Documentos-15: INSERT cero tipado persistido'; END IF;
END $prueba$;
ROLLBACK;
