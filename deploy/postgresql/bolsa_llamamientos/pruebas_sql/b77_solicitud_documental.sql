\set ON_ERROR_STOP on
-- Fixture estructural sintética: inserta bajo propietario y deshace todo.
-- No sustituye un POST con COSE y consumo V3 reales.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $prueba$
<<prueba_b77>>
DECLARE
 candidato text:='can_AAAAAAAAAAAAAAAAAAAAAA';
 bolsa text:='bolsa:sintetica:rrhh17';
 participacion text:='participacion:sintetica:rrhh17';
 referencia text:='solicitud-documental:de0102ba8087b9361736f83e4ac1d0ae1a9be17497ee9e6c2240ab209007dcf9';
 recibo text:='recibo:solicitud-documental:e0ed615fc55bb083cc6d116e1e0ca07f4763c7ffac46d3ed1f5400faceb83cf3';
 contenido text:='df64d4ae309b0fb20920edb012bc693250d7efee3b1e91288fea365a00e303b9';
 recurso bytea:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"c880f55833ff54587697e24638453ff9faeb441d24b185a7047a7490a09d50c6"}}','UTF8');
 decision_doble bytea:=pg_catalog.convert_to('{"contexto_recurso_huella_sha256":"61858286bdd9bbb779c7255d43ebddf4a5ce5d70d41e3f72f36aef8a10dac1de"}','UTF8');
BEGIN
 IF vec_bolsa_llamamientos.b77_huella_partes_v1('solicitud-documental',candidato,bolsa,'clave:sintetica:rrhh17')
      IS DISTINCT FROM pg_catalog.substr(referencia,pg_catalog.length('solicitud-documental:')+1)
    OR vec_bolsa_llamamientos.b77_huella_partes_v1('recibo',pg_catalog.substr(referencia,pg_catalog.length('solicitud-documental:')+1))
      IS DISTINCT FROM pg_catalog.substr(recibo,pg_catalog.length('recibo:solicitud-documental:')+1)
    OR vec_bolsa_llamamientos.b77_huella_partes_v1('contenido-solicitud-documental',candidato,bolsa,
         'documento:sintetico:rrhh17',pg_catalog.repeat('a',64),'2026-10-02') IS DISTINCT FROM contenido THEN
  RAISE EXCEPTION 'B77 huella Go/SQL distinta'; END IF;
 IF vec_bolsa_llamamientos.b77_huella_partes_v1('contenido-solicitud-documental',candidato,bolsa,
   'documento:sintetico:rrhh17',pg_catalog.repeat('a',64),'') IS DISTINCT FROM
   '9bff03e2c2b23f401a5208d2073851a14dcf26094f57539c07cc6e10b9e7fff2' THEN
  RAISE EXCEPTION 'B77 huella sin fecha Go/SQL distinta'; END IF;
 IF vec_bolsa_llamamientos.b77_documento_ref_opaco_v1('dni:prueba') IS NOT FALSE
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1('NIE:prueba') IS NOT FALSE
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1('documento:12345678Z') IS NOT FALSE
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1('documento:X1234567L') IS NOT FALSE
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1('documento:sintetico:rrhh17') IS NOT TRUE
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(
      'documento:'||pg_catalog.repeat('a',64)) IS NOT TRUE THEN
  RAISE EXCEPTION 'B77 referencia documental no opaca'; END IF;
 IF vec_bolsa_llamamientos.b77_huella_partes_v1(
   'regularizacion-documental-v1','solicitud-documental:'||pg_catalog.repeat('a',64),'1',pg_catalog.repeat('b',64),
   bolsa,participacion,'documento:sintetico:rrhh17',pg_catalog.repeat('c',64),'2026-10-02',
   '1790899200000000',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('documento validado','UTF8')),'hex'),
   'actor:sintetico','validador:sintetico','clave:regularizar:rrhh17','recibo:regularizar:rrhh17')
     IS DISTINCT FROM 'c880f55833ff54587697e24638453ff9faeb441d24b185a7047a7490a09d50c6'
    OR pg_catalog.encode(pg_catalog.sha256(recurso),'hex') IS DISTINCT FROM
       '61858286bdd9bbb779c7255d43ebddf4a5ce5d70d41e3f72f36aef8a10dac1de'
    OR vec_bolsa_llamamientos.b77_huella_partes_v1('campo',E'valor\x1falterado') IS NOT NULL THEN
  RAISE EXCEPTION 'B77 canon del recurso firmado distinto'; END IF;
 -- Doble nominal de decisión sin COSE: un contexto alterado debe fallar ANTES
 -- de B76. Con contexto correcto se alcanza B76 y falta la participación
 -- sintética (23503); no se afirma una autorización positiva.
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   'solicitud-documental:'||pg_catalog.repeat('a',64),1,pg_catalog.repeat('b',64),bolsa,participacion,
   '2026-10-03T00:00:00Z','2026-10-02T00:00:00Z','documento validado','actor:sintetico',
   'clave:regularizar:rrhh17','recibo:regularizar:rrhh17','2026-10-03T00:00:00Z',
   'validador:sintetico','2026-10-03T00:00:00Z','documento:sintetico:rrhh17',pg_catalog.repeat('c',64),
   '2026-10-02'::date,NULL,decision_doble,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
   pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}}','UTF8'));
  RAISE EXCEPTION 'B77 contexto alterado aceptado';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
   'solicitud-documental:'||pg_catalog.repeat('a',64),1,pg_catalog.repeat('b',64),bolsa,participacion,
   '2026-10-03T00:00:00Z','2026-10-02T00:00:00Z','documento validado','actor:sintetico',
   'clave:regularizar:rrhh17','recibo:regularizar:rrhh17','2026-10-03T00:00:00Z',
   'validador:sintetico','2026-10-03T00:00:00Z','documento:sintetico:rrhh17',pg_catalog.repeat('c',64),
   '2026-10-02'::date,NULL,decision_doble,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,recurso);
  RAISE EXCEPTION 'B77 doble nominal alcanzó efecto';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 INSERT INTO vec_bolsa_llamamientos.solicitud_documental_rrhh(
  solicitud_ref,recibo_ref,contenido_sha256,bolsa_ref,participacion_ref,candidato_ref,tipo,
  documento_ref,documento_sha256,fecha_fin_causa,version,estado_origen,clave_idempotencia,
  decision_ref,auditoria_ref,registrada_en)
 VALUES(referencia,recibo,contenido,bolsa,participacion,candidato,'documental_rrhh',
  'documento:sintetico:rrhh17',pg_catalog.repeat('a',64),'2026-10-02',1,'pendiente_rrhh',
  'clave:sintetica:rrhh17','decision:sintetica:rrhh17','auditoria:sintetica:rrhh17','2026-10-02T12:00:00Z');
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1('2026-10-02T13:00:00Z') a
     WHERE a.referencia=prueba_b77.referencia AND a.detalle->>'solicitud'='documental_rrhh'
       AND (a.detalle ? 'documento_ref') IS FALSE AND (a.detalle ? 'contenido_sha256') IS FALSE)=0 THEN
  RAISE EXCEPTION 'B77 aviso documental pendiente ausente'; END IF;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(
   bolsa,participacion,'regularizar','2026-10-03T00:00:00Z',NULL,'motivo sintetico','actor:sintetico',
   'clave:resolucion:sintetica','recibo:resolucion:sintetica','2026-10-03T00:00:00Z',
   'solicitud_candidato',referencia,pg_catalog.repeat('a',64),'validador:sintetico',
   '2026-10-03T00:00:00Z','2026-10-02T00:00:00Z','2026-10-02T00:00:00Z',
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B77 v2 permitió el bypass documental';
 EXCEPTION WHEN sqlstate '42501' THEN NULL;
 END;
 INSERT INTO vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh(
  solicitud_ref,version_esperada,contenido_sha256,resultado,motivo,actor_ref,clave_idempotencia,
  recibo_ref,decision_ref,auditoria_ref,resuelta_en)
 VALUES(referencia,1,contenido,'rechazada','documento insuficiente','actor:sintetico',
  'clave:rechazo:sintetica','recibo:rechazo:sintetico','decision:rechazo:sintetico',
  'auditoria:rechazo:sintetico','2026-10-02T13:00:00Z');
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1('2026-10-02T14:00:00Z') a
  WHERE a.referencia=prueba_b77.referencia) THEN RAISE EXCEPTION 'B77 aviso siguió pendiente tras rechazo'; END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(
   coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid='vec_bolsa_llamamientos.b77_huella_partes_v1(text[])'::pg_catalog.regprocedure
     AND a.grantee<>p.proowner) THEN RAISE EXCEPTION 'B77 helper público'; END IF;
END $prueba$;
ROLLBACK;
