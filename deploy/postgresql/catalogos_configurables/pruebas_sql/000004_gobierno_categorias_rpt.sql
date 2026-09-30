\set ON_ERROR_STOP on
-- Clon desechable. Dobles de persona en el core: no acreditan consumo V3.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE preimagen_legacy AS SELECT * FROM vec_catalogos_configurables.publicacion WHERE circuito='doble_aprobacion';
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE
 motivo text:='motivos_rpt:1:prueba_gobierno';
 doc text:='{"id":"rpt-gobierno-demo","modulo_id":"personal","version":1,"estado":"publicado","fuente_ref":"fuente:rpt:sintetica","creado_por":"actor:a","publicado_por":"actor:b","aprobacion_ref":"recibo:aprobar","entradas":[{"clave":"cat-gobierno-demo","etiqueta":"Categoria sintetica","atributos":{"estado":"habilitada"}},{"clave":"cat-gobierno-otra","etiqueta":"Otra categoria sintetica","atributos":{"estado":"habilitada"}}]}';
 d jsonb; h text; doc_h text; r jsonb; anterior jsonb; pre jsonb; des jsonb; des_h text; doc2 text; doc2_h text;
 incompleta jsonb; inc_h text;
BEGIN
 IF pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.propuesta_gobierno','SELECT') THEN RAISE EXCEPTION 'AD3 tiene lectura directa'; END IF;
 doc_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc,'UTF8')),'hex');
 d:=pg_catalog.jsonb_build_object('accion','publicar','catalogo_id','rpt-gobierno-demo','modulo_id','personal','version',1,'documento_canonico',doc,'documento_huella_sha256',doc_h,'preimagenes_control','{}'::jsonb,'preimagenes_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex'),'categoria_id',NULL,'revision_esperada',NULL,'motivo_ref',motivo,'fuente_ref','fuente:rpt:sintetica');
 h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d::text,'UTF8')),'hex');
 r:=vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:demo',d,h,'actor:a','decision:proponer','recibo:proponer',motivo);
 IF r->>'revision'<>'1' OR r->>'estado'<>'propuesta' THEN RAISE EXCEPTION 'propuesta incorrecta'; END IF;
 IF vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:demo',d,h,'actor:a','decision:proponer-replay','recibo:proponer',motivo) IS DISTINCT FROM r THEN RAISE EXCEPTION 'replay propuesta'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',h,1,'actor:a','decision:autoprobar','audit:auto',repeat('1',64),'recibo:aprobar',motivo);
  RAISE EXCEPTION 'A aprobo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',repeat('0',64),1,'actor:b','decision:huella','audit:huella',repeat('1',64),'recibo:aprobar',motivo);
  RAISE EXCEPTION 'otra huella aprobada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,1,'actor:b','decision:prematura','audit:prematura','recibo:prematura',motivo);
  RAISE EXCEPTION 'confirmacion prematura';
 EXCEPTION WHEN SQLSTATE '40001' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:b','decision:cas','audit:cas',repeat('2',64),'recibo:aprobar',motivo);
  RAISE EXCEPTION 'CAS aprobacion obsoleto admitido';
 EXCEPTION WHEN SQLSTATE '40001' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',h,1,'actor:c','decision:otro','audit:otro',repeat('2',64),'recibo:aprobar',motivo);
  RAISE EXCEPTION 'aprobo persona diferente del publicador declarado';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 r:=vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',h,1,'actor:b','decision:aprobar','audit:aprobar',repeat('2',64),'recibo:aprobar',motivo);
 IF r->>'revision'<>'2' OR r->>'estado'<>'aprobada' THEN RAISE EXCEPTION 'aprobacion incorrecta'; END IF;
 IF vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:demo',h,1,'actor:b','decision:aprobar-replay','audit:replay',repeat('3',64),'recibo:aprobar',motivo) IS DISTINCT FROM r THEN RAISE EXCEPTION 'replay aprobacion'; END IF;
 anterior:=vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:demo',h);
 IF pg_catalog.jsonb_array_length(anterior->'aprobaciones')<>1 THEN RAISE EXCEPTION 'aprobaciones duplicadas'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:a','decision:a','audit:a','recibo:a',motivo);
  RAISE EXCEPTION 'A confirmo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:c','decision:c','audit:c','recibo:c',motivo);
  RAISE EXCEPTION 'C confirmo aprobacion de B';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:b','decision:sin-audit',NULL,'recibo:confirmar',motivo);
  RAISE EXCEPTION 'confirmacion sin auditoria';
 EXCEPTION WHEN not_null_violation THEN NULL; END;
 IF vec_catalogos_configurables.leer_publicacion_categoria('rpt-gobierno-demo',1,doc_h,'cat-gobierno-demo')->>'encontrado'<>'false'
 OR vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:demo',h) IS DISTINCT FROM anterior THEN RAISE EXCEPTION 'fallo parcial'; END IF;
 r:=vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:b','decision:confirmar','audit:confirmar','recibo:confirmar',motivo);
 IF r->>'revision'<>'3' OR r->>'estado'<>'confirmada'
 OR vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:demo',h,2,'actor:b','decision:confirmar-replay','audit:replay','recibo:confirmar',motivo) IS DISTINCT FROM r THEN RAISE EXCEPTION 'confirmacion o replay incorrectos'; END IF;
 PERFORM vec_catalogos_configurables.reservar('personal','uso:gobierno:anterior','cat-gobierno-demo','rpt-gobierno-demo',1,doc_h,'actor:b','decision:reserva','recibo:reserva',motivo);
 pre:=pg_catalog.jsonb_build_object('cat-gobierno-demo',pg_catalog.jsonb_build_object('version',1,'huella_sha256',doc_h,'revision',1,'estado','habilitada'),'cat-gobierno-otra',pg_catalog.jsonb_build_object('version',1,'huella_sha256',doc_h,'revision',1,'estado','habilitada'));
 doc2:=(doc::jsonb||pg_catalog.jsonb_build_object('version',2,'aprobacion_ref','recibo:des-aprobar','entradas',pg_catalog.jsonb_build_array((doc::jsonb->'entradas'->0)||'{"atributos":{"estado":"deshabilitada"}}'::jsonb,doc::jsonb->'entradas'->1)))::text;
 doc2_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc2,'UTF8')),'hex');
 des:=d||pg_catalog.jsonb_build_object('accion','deshabilitar','version',2,'documento_canonico',doc2,'documento_huella_sha256',doc2_h,'categoria_id','cat-gobierno-demo','revision_esperada',1,'preimagenes_control',pre,'preimagenes_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex'));
 -- Omitir una categoria anterior no puede publicarse ni deshabilitar parcialmente.
 doc2:=pg_catalog.jsonb_set(doc2::jsonb,'{entradas}',pg_catalog.jsonb_build_array(doc2::jsonb->'entradas'->0))::text;
 incompleta:=des||pg_catalog.jsonb_build_object('documento_canonico',doc2,'documento_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc2,'UTF8')),'hex'));
 inc_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(incompleta::text,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:incompleta',incompleta,inc_h,'actor:a','decision:inc-prop','recibo:inc-prop',motivo);
 PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:incompleta',inc_h,1,'actor:b','decision:inc-apr','audit:inc',repeat('4',64),'recibo:des-aprobar',motivo);
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:incompleta',inc_h,2,'actor:b','decision:inc-con','audit:inc-con','recibo:inc-con',motivo);
  RAISE EXCEPTION 'version incompleta publicada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 -- La siguiente propuesta usa otro recibo: todos los recibos son unicos.
 doc2:=pg_catalog.jsonb_set((des->>'documento_canonico')::jsonb,'{aprobacion_ref}','"recibo:des-aprobar-real"'::jsonb)::text;
 des:=des||pg_catalog.jsonb_build_object('documento_canonico',doc2,'documento_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc2,'UTF8')),'hex'));
 des_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(des::text,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.registrar_propuesta_gobierno('propuesta:gobierno:des',des,des_h,'actor:a','decision:des-prop','recibo:des-prop',motivo);
 PERFORM vec_catalogos_configurables.aprobar_propuesta_gobierno('propuesta:gobierno:des',des_h,1,'actor:b','decision:des-apr','audit:des',repeat('5',64),'recibo:des-aprobar-real',motivo);
 r:=vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:des',des_h,2,'actor:b','decision:des-con','audit:des-con','recibo:des-con',motivo);
 IF r->>'version'<>'2' OR r->>'revision'<>'3' THEN RAISE EXCEPTION 'deshabilitar no publico version completa'; END IF;
 -- Dos propuestas sobre una misma versión no pueden confirmar dos publicaciones.
 anterior:=vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:incompleta',inc_h);
 BEGIN
  PERFORM vec_catalogos_configurables.confirmar_propuesta_gobierno('propuesta:gobierno:incompleta',inc_h,2,'actor:b','decision:version-perdedora','audit:perdedora','recibo:version-perdedora',motivo);
  RAISE EXCEPTION 'segunda publicacion de version vencida';
 EXCEPTION WHEN SQLSTATE '40001' THEN NULL; END;
 IF vec_catalogos_configurables.consultar_aprobaciones_gobierno('propuesta:gobierno:incompleta',inc_h) IS DISTINCT FROM anterior THEN RAISE EXCEPTION 'CAS perdedor cambio historia'; END IF;

 IF vec_catalogos_configurables.consultar_uso('personal','uso:gobierno:anterior','recibo:reserva')#>>'{datos,estado}'<>'reservado' THEN RAISE EXCEPTION 'reserva previa perdida'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.reservar('personal','uso:gobierno:posterior','cat-gobierno-demo','rpt-gobierno-demo',2,des->>'documento_huella_sha256','actor:b','decision:nueva','recibo:nueva',motivo);
  RAISE EXCEPTION 'nueva reserva sobre deshabilitada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 PERFORM vec_catalogos_configurables.terminar_uso('personal','uso:gobierno:anterior','recibo:reserva','confirmado','actor:b','decision:terminal','recibo:terminal',motivo);
 IF vec_catalogos_configurables.consultar_uso('personal','uso:gobierno:anterior','recibo:reserva')#>>'{datos,estado}'<>'confirmado' THEN RAISE EXCEPTION 'terminal anterior perdido'; END IF;
END $prueba$;
RESET ROLE;
DO $historia$
DECLARE f oid:='vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)'::regprocedure;
BEGIN
 IF (SELECT count(*) FROM vec_catalogos_configurables.publicacion WHERE catalogo_id='rpt-gobierno-demo')<>2
 OR (SELECT count(*) FROM vec_catalogos_configurables.confirmacion_gobierno WHERE propuesta_ref LIKE 'propuesta:gobierno:%')<>2
 OR (SELECT count(*) FROM vec_catalogos_configurables.outbox_gobierno WHERE propuesta_ref LIKE 'propuesta:gobierno:%')<>8
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.publicacion WHERE catalogo_id='rpt-gobierno-demo' AND (circuito<>'propuesta_aprobacion_rrhh' OR propuesta_ref IS NULL OR aprobacion_b_ref IS NOT NULL))
 OR (SELECT estado FROM vec_catalogos_configurables.categoria_control WHERE categoria_id='cat-gobierno-demo')<>'deshabilitada'
 OR (SELECT version FROM vec_catalogos_configurables.categoria_control WHERE categoria_id='cat-gobierno-otra')<>2
 OR (SELECT count(*) FROM vec_catalogos_configurables.historia WHERE categoria_id='cat-gobierno-demo' AND accion='deshabilitar')<>1 THEN RAISE EXCEPTION 'historia/version/publicacion incoherente'; END IF;
 IF EXISTS(SELECT * FROM pg_temp.preimagen_legacy EXCEPT SELECT * FROM vec_catalogos_configurables.publicacion WHERE circuito='doble_aprobacion') THEN RAISE EXCEPTION 'publicaciones legacy alteradas'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=5s','statement_timeout=30s']) THEN RAISE EXCEPTION 'metadatos Cat1 alterados'; END IF;
 BEGIN
  UPDATE vec_catalogos_configurables.propuesta_gobierno SET editor_ref='actor:cambio' WHERE propuesta_ref='propuesta:gobierno:demo';
  RAISE EXCEPTION 'propuesta mutable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $historia$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $legacy$
DECLARE doc text:='{"id":"rpt-legacy-demo","modulo_id":"personal","version":1,"estado":"publicado","entradas":[{"clave":"cat-legacy-demo","etiqueta":"Categoria legacy"}]}';h text;ph text;
BEGIN
 h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc,'UTF8')),'hex');ph:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex');
 BEGIN
  PERFORM vec_catalogos_configurables.publicar('rpt-legacy-demo',1,h,doc,'{}'::jsonb,ph,'recibo:inventado',NULL,'actor:b','decision:falsa','recibo:falso','motivos_rpt:1:prueba_gobierno');
  RAISE EXCEPTION 'NULL sin propuesta fue publicado';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 PERFORM vec_catalogos_configurables.publicar('rpt-legacy-demo',1,h,doc,'{}'::jsonb,ph,'recibo:legacy-a','recibo:legacy-b','actor:b','decision:legacy','recibo:legacy','motivos_rpt:1:prueba_gobierno');
END $legacy$;
ROLLBACK;
