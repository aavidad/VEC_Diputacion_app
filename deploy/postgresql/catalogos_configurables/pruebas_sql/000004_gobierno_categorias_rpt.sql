\set ON_ERROR_STOP on
-- Solo clon desechable. Autoridad de persistencia bajo AD3 propietario;
-- estas identidades son dobles de prueba y no acreditan consumo V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $prueba$
DECLARE
 motivo text:='motivos_rpt:1:prueba_gobierno';
 documento text:='{"id":"rpt-gobierno-demo","modulo_id":"personal","version":1,"estado":"publicado","fuente_ref":"fuente:rpt:sintetica","creado_por":"actor:editor","publicado_por":"actor:a","entradas":[{"clave":"cat-gobierno-demo","etiqueta":"Categoria sintetica"}]}';
 contenido jsonb; huella text; doc_h text; vacias_h text; r jsonb; anterior jsonb; des jsonb; des_h text; pre jsonb; ajeno jsonb; ajeno_h text;
BEGIN
 IF pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.propuesta_gobierno','SELECT') THEN
  RAISE EXCEPTION 'AD3 tiene lectura directa del gobierno';
 END IF;
 doc_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento,'UTF8')),'hex');
 vacias_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex');
 contenido:=pg_catalog.jsonb_build_object('accion','publicar','catalogo_id','rpt-gobierno-demo','modulo_id','personal','version',1,'documento_canonico',documento,'documento_huella_sha256',doc_h,'preimagenes_control','{}'::jsonb,'preimagenes_huella_sha256',vacias_h,'categoria_id',NULL,'revision_esperada',NULL,'motivo_ref',motivo,'fuente_ref','fuente:rpt:sintetica');
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(contenido::text,'UTF8')),'hex');
 r:=vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:demo',contenido,huella,'actor:editor','decision:proponer','recibo:proponer',motivo);
 IF r->>'revision'<>'1' THEN RAISE EXCEPTION 'revision inicial incorrecta'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:demo',contenido||'{"fuente_ref":"fuente:cambiada"}'::jsonb,huella,'actor:editor','decision:alterar','recibo:proponer',motivo);
  RAISE EXCEPTION 'huella cambiada aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 r:=vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',huella,1,'actor:editor','decision:apr-a','audit:apr-a',repeat('2',64),'recibo:apr-a',motivo);
 IF r->>'revision'<>'2' THEN RAISE EXCEPTION 'primera aprobacion incorrecta'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',huella,2,'actor:editor','decision:apr-duplicada','audit:dup',repeat('3',64),'recibo:duplicado',motivo);
  RAISE EXCEPTION 'misma persona aprobo dos veces';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',repeat('0',64),2,'actor:a','decision:apr-h','audit:h',repeat('3',64),'recibo:apr-h',motivo);
  RAISE EXCEPTION 'aprobacion sobre otra huella';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 r:=vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',huella,2,'actor:a','decision:apr-b','audit:apr-b',repeat('3',64),'recibo:apr-b',motivo);
 anterior:=vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:demo',huella);
 IF pg_catalog.jsonb_array_length(anterior->'aprobaciones')<>2 OR anterior#>>'{propuesta,revision}'<>'3' THEN RAISE EXCEPTION 'doble aprobacion incorrecta'; END IF;
 -- Dos personas bastan: el editor aprueba, la otra identidad aprueba y
 -- confirma. El editor sigue sin poder publicar su propia propuesta.
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',huella,3,'actor:editor','decision:confirm-editor','audit:editor','recibo:confirm-editor',motivo);
  RAISE EXCEPTION 'editor confirmo su propuesta';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 -- Fallo posterior al efecto: la inserción de auditoría del recibo no puede
 -- dejar publicación ni transición parcial. La auditoría central se prueba en AD134.
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',huella,3,'actor:a','decision:fallo-audit',NULL,'recibo:confirmar',motivo);
  RAISE EXCEPTION 'confirmacion sin auditoria';
 EXCEPTION WHEN not_null_violation THEN NULL; END;
 IF vec_catalogos_configurables.leer_publicacion_categoria('rpt-gobierno-demo',1,doc_h,'cat-gobierno-demo')->>'encontrado'<>'false'
 OR vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:demo',huella) IS DISTINCT FROM anterior THEN
  RAISE EXCEPTION 'fallo audit dejo efecto parcial';
 END IF;
 r:=vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',huella,3,'actor:a','decision:confirmar','audit:confirmar','recibo:confirmar',motivo);
 IF r->>'revision'<>'4' OR vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',huella,3,'actor:a','decision:confirmar-replay','audit:replay','recibo:confirmar',motivo) IS DISTINCT FROM r THEN
  RAISE EXCEPTION 'replay altero recibo';
 END IF;
 -- Una reserva anterior a deshabilitar permanece recuperable y terminal.
 PERFORM vec_catalogos_configurables.reservar('personal','uso:gobierno:anterior','cat-gobierno-demo','rpt-gobierno-demo',1,doc_h,'actor:a','decision:reserva','recibo:reserva',motivo);
 pre:=pg_catalog.jsonb_build_object('cat-gobierno-demo',pg_catalog.jsonb_build_object('version',1,'huella_sha256',doc_h,'revision',1,'estado','habilitada'));
 des:=pg_catalog.jsonb_build_object('accion','deshabilitar','catalogo_id','rpt-gobierno-demo','modulo_id','personal','version',1,'documento_canonico',NULL,'documento_huella_sha256',NULL,'preimagenes_control',pre,'preimagenes_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex'),'categoria_id','cat-gobierno-demo','revision_esperada',1,'motivo_ref',motivo,'fuente_ref','fuente:rpt:sintetica');
 ajeno:=des||'{"modulo_id":"bolsa"}'::jsonb;
 ajeno_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ajeno::text,'UTF8')),'hex');
 BEGIN
  PERFORM vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:ajena',ajeno,ajeno_h,'actor:editor','decision:ajena','recibo:ajena',motivo);
  RAISE EXCEPTION 'modulo ajeno a publicacion admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 des_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(des::text,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:des',des,des_h,'actor:editor','decision:des-prop','recibo:des-prop',motivo);
 PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:des',des_h,1,'actor:a','decision:des-a','audit:des-a',repeat('4',64),'recibo:des-a',motivo);
 PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:des',des_h,2,'actor:b','decision:des-b','audit:des-b',repeat('5',64),'recibo:des-b',motivo);
 PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:des',des_h,3,'actor:a','decision:des-confirmar','audit:des-confirmar','recibo:des-confirmar',motivo);
 IF vec_catalogos_configurables.consultar_uso('personal','uso:gobierno:anterior','recibo:reserva')#>>'{datos,estado}'<>'reservado' THEN RAISE EXCEPTION 'uso anterior perdido'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.reservar('personal','uso:gobierno:posterior','cat-gobierno-demo','rpt-gobierno-demo',1,doc_h,'actor:a','decision:res-posterior','recibo:res-posterior',motivo);
  RAISE EXCEPTION 'nueva reserva tras deshabilitar';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 PERFORM vec_catalogos_configurables.terminar_uso('personal','uso:gobierno:anterior','recibo:reserva','confirmado','actor:a','decision:terminal','recibo:terminal',motivo);
 IF vec_catalogos_configurables.consultar_uso('personal','uso:gobierno:anterior','recibo:reserva')#>>'{datos,estado}'<>'confirmado' THEN RAISE EXCEPTION 'terminal anterior cerrado'; END IF;
END $prueba$;
RESET ROLE;
DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_catalogos_configurables.publicacion WHERE catalogo_id='rpt-gobierno-demo')<>1
 OR (SELECT count(*) FROM vec_catalogos_configurables.confirmacion_gobierno WHERE propuesta_ref LIKE 'propuesta:gobierno:%')<>2
 OR (SELECT count(*) FROM vec_catalogos_configurables.outbox_gobierno WHERE propuesta_ref LIKE 'propuesta:gobierno:%')<>8 THEN RAISE EXCEPTION 'duplicados o efecto parcial'; END IF;
 BEGIN
  UPDATE vec_catalogos_configurables.propuesta_gobierno SET editor_ref='actor:cambio' WHERE propuesta_ref='propuesta:gobierno:demo';
  RAISE EXCEPTION 'propuesta mutable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $historia$;
ROLLBACK;
