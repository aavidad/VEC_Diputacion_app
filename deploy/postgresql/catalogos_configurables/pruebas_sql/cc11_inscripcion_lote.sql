\set ON_ERROR_STOP on
-- Fixture desechable, siempre ROLLBACK. Sólo prueba el contrato lector CC11.
BEGIN;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE cat_doc text:='{"id":"bolsa.categorias.inscripcion","version":1}';
 pol_doc text:='{"id":"bolsa.politica.inscripcion","version":1}';
 pol_doc2 text:='{"id":"bolsa.politica.sinpresentacion","version":1}';
 cat_sha text; pol_sha text; pol_sha2 text; r jsonb; lote_maximo jsonb; f regprocedure;
 f_categorias regprocedure; refs_128 jsonb; comprobacion jsonb; selector jsonb;
 lote_distinto jsonb; lote_repetido jsonb;
 f_asociacion regprocedure; alcance text;
BEGIN
 alcance:='cv1_'||translate(rtrim(encode(convert_to('proceso:bolsa:historica-2026','UTF8'),'base64'),'='),'+/','-_')||'_v1';
 f:='vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text)'::regprocedure;
 f_categorias:='vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)'::regprocedure;
 f_asociacion:='vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)'::regprocedure;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_categorias,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_categorias,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_asociacion,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_asociacion,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid IN(f,f_categorias,f_asociacion) AND a.grantee=0)
 THEN RAISE EXCEPTION 'CC11: ACL abierta'; END IF;
 cat_sha:=encode(sha256(convert_to(cat_doc,'UTF8')),'hex');
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 pol_sha2:=encode(sha256(convert_to(pol_doc2,'UTF8')),'hex');
 INSERT INTO vec_catalogos_configurables.publicacion (
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES
  ('bolsa.categorias.inscripcion',1,cat_sha,cat_doc,'{}',
   encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:cat:a','aprobacion:cat:b',
   'actor:cc11','decision:cat','recibo:cc11:cat'),
  ('bolsa.politica.inscripcion',1,pol_sha,pol_doc,'{}',
   encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:pol:a','aprobacion:pol:b',
   'actor:cc11','decision:pol','recibo:cc11:pol'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,pol_doc2,'{}',
   encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:pol2:a','aprobacion:pol2:b',
   'actor:cc11','decision:pol2','recibo:cc11:pol2');
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.alpha','Alfa ES',
   '{"etiquetas":{"es":"Alfa ES","en":"Alpha EN"}}'),
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.beta','Beta ES','{}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.con.pendientes','Política',
   '{"valor":true,"canal":"externa_personal"}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'requisito.pendiente','Pendiente ES',
   '{"etiquetas":{"es":"Pendiente ES","en":"Pending EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'requisito.cumple','Cumple ES',
   '{"etiquetas":{"es":"Cumple ES","en":"Meets EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'solicitud.existente','Ya solicitada ES',
   '{"etiquetas":{"es":"Ya solicitada ES","en":"Already applied EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.no.disponible','No disponible ES',
   '{"etiquetas":{"es":"No disponible ES","en":"Unavailable EN"}}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'requisito.pendiente','Pendiente ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'requisito.cumple','Cumple ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'solicitud.existente','Ya solicitada ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'presentacion.no.disponible','No disponible ES','{}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'asociacion.manual.acta','Asociación manual',
   jsonb_build_object('valor',true,'canal','interna_corporativa',
    'motivo_ref','motivo:asociacion.manual','alcance_ref',alcance));
 r:=vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
  'bolsa.politica.inscripcion',1,pol_sha);
 IF r->>'permitida'<>'true' OR r->>'politica_ref'<>'asociacion.manual.acta'
 OR r->>'version'<>'1' OR r->>'sha256'<>pol_sha
 OR r->>'motivo_ref'<>'motivo:asociacion.manual' OR r->>'alcance_ref'<>alcance
 THEN RAISE EXCEPTION 'CC11: política de asociación no exacta: %',r; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
   'bolsa.politica.sinpresentacion',1,pol_sha2);
  RAISE EXCEPTION 'CC11: asociación sin política aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES ('bolsa.politica.sinpresentacion',1,pol_sha2,'asociacion.manual.acta',
  'Asociación denegada',jsonb_build_object('valor',false,
   'canal','interna_corporativa','motivo_ref','motivo:asociacion.manual',
   'alcance_ref',alcance));
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
   'bolsa.politica.sinpresentacion',1,pol_sha2);
  RAISE EXCEPTION 'CC11: política false aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,repeat('0',64));
  RAISE EXCEPTION 'CC11: huella de asociación equivocada aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1('[]','es');
 IF r<>'[]'::jsonb THEN RAISE EXCEPTION 'CC11: lote vacío'; END IF;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(
   jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.alpha',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha),
   jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.beta',
    'politica_catalogo_ref','bolsa.politica.sinpresentacion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha2)),
  'en');
 IF jsonb_array_length(r)<>2 OR r#>>'{0,categoria}'<>'Alpha EN'
 OR r#>>'{0,politica_valida}'<>'true' OR r#>>'{1,categoria}'<>'Beta ES'
 OR r#>>'{1,politica_valida}'<>'false'
 OR r#>>'{0,motivo_etiqueta_pendiente}'<>'Pending EN'
 OR r#>>'{1,motivo_etiqueta_pendiente}'<>'Pendiente ES'
 OR r#>>'{0,motivo_etiqueta_cumple}'<>'Meets EN'
 OR r#>>'{1,motivo_etiqueta_cumple}'<>'Cumple ES'
 OR r#>>'{0,impedimento_etiqueta_existente}'<>'Already applied EN'
 OR r#>>'{1,impedimento_etiqueta_existente}'<>'Ya solicitada ES'
 OR r#>>'{0,impedimento_etiqueta_politica}'<>'Unavailable EN'
 OR r#>>'{1,impedimento_etiqueta_politica}'<>'No disponible ES'
 THEN RAISE EXCEPTION 'CC11: lote mixto, fallback o política: %',r; END IF;
 SELECT jsonb_agg(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.alpha',
  'politica_catalogo_ref','bolsa.politica.inscripcion',
  'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha))
 INTO lote_maximo FROM generate_series(1,12800);
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(lote_maximo,'es');
 IF jsonb_array_length(r)<>12800 OR r#>>'{12799,motivo_etiqueta_pendiente}'<>'Pendiente ES'
 OR r#>>'{12799,motivo_etiqueta_cumple}'<>'Cumple ES'
 THEN RAISE EXCEPTION 'CC11: página máxima truncada'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   lote_maximo||jsonb_build_array(lote_maximo->0),'es');
  RAISE EXCEPTION 'CC11: exceso de cardinalidad aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
 END;
 INSERT INTO vec_catalogos_configurables.publicacion (
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES ('bolsa.politica.sinmotivo',1,
  encode(sha256(convert_to('{"id":"bolsa.politica.sinmotivo"}','UTF8')),'hex'),
  '{"id":"bolsa.politica.sinmotivo"}','{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:sin:a','aprobacion:sin:b','actor:cc11','decision:sin','recibo:cc11:sin');
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.alpha',
    'politica_catalogo_ref','bolsa.politica.sinmotivo',
    'politica_catalogo_version',1,'politica_catalogo_sha256',
      encode(sha256(convert_to('{"id":"bolsa.politica.sinmotivo"}','UTF8')),'hex'))),'es');
  RAISE EXCEPTION 'CC11: motivo de requisito ausente aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL;
 END;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.ausente',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'es');
  RAISE EXCEPTION 'CC11: etiqueta ausente aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL;
 END;
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 SELECT 'bolsa.categorias.inscripcion',1,cat_sha,'cat.'||g,'Categoría '||g,'{}'::jsonb
 FROM generate_series(1,127) AS g;
 SELECT jsonb_agg('cat.'||g ORDER BY g) INTO refs_128 FROM generate_series(1,128) AS g;
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1('[]','es')<>'[]'::jsonb
 THEN RAISE EXCEPTION 'CC11: comprobación vacía incorrecta'; END IF;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
  jsonb_build_array(
   jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,
    'categorias_refs',jsonb_build_array('cat.1')),
   jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categorias_refs',refs_128)),'es');
 IF jsonb_array_length(comprobacion)<>2
 OR comprobacion#>>'{0,catalogo_completo}'<>'true'
 OR comprobacion#>>'{0,numero_categorias}'<>'1'
 OR comprobacion#>>'{1,catalogo_completo}'<>'false'
 OR comprobacion#>>'{1,numero_categorias}'<>'128'
 THEN RAISE EXCEPTION 'CC11: categoría 128 ausente no detectada: %',comprobacion; END IF;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',repeat('0',64),
   'categorias_refs',jsonb_build_array('cat.1'))),'es');
 IF comprobacion#>>'{0,catalogo_completo}'<>'false'
 THEN RAISE EXCEPTION 'CC11: huella equivocada completó catálogo'; END IF;
 SELECT jsonb_agg(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.'||g)) ORDER BY g)
 INTO lote_distinto FROM generate_series(1,100) AS g;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(lote_distinto,'es');
 IF jsonb_array_length(comprobacion)<>100
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(comprobacion) AS x(valor)
  WHERE x.valor->>'catalogo_completo'<>'true' OR x.valor->>'numero_categorias'<>'1')
 THEN RAISE EXCEPTION 'CC11: 100 combinaciones distintas desalineadas'; END IF;
 SELECT jsonb_agg(lote_distinto->0) INTO lote_repetido FROM generate_series(1,100);
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(lote_repetido,'es');
 IF jsonb_array_length(comprobacion)<>100
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(comprobacion) AS x(valor)
  WHERE x.valor->>'catalogo_completo'<>'true' OR x.valor->>'numero_categorias'<>'1')
 THEN RAISE EXCEPTION 'CC11: 100 combinaciones repetidas desalineadas'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,
    'categorias_refs',jsonb_build_array('cat.1','cat.1'))),'es');
  RAISE EXCEPTION 'CC11: refs duplicadas aceptadas';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 -- CC1 admite hasta 2048 bytes de etiqueta y objeto JSON arbitrario: estas
 -- entradas existen en la publicación, pero el traductor no puede ofrecerlas.
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.128',repeat('X',201),'{}'),
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.bad.en','Base ES',
   '{"etiquetas":{"es":"Bien ES","en":42}}'),
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.en.larga','Base ES',
   jsonb_build_object('etiquetas',jsonb_build_object('es','Bien ES','en',repeat('E',201)))),
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.fallback','Base ES',
   '{"etiquetas":{"es":"Bien ES"}}');
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   'bolsa.categorias.inscripcion',1,cat_sha,ARRAY['cat.128'],'es');
  RAISE EXCEPTION 'CC11: etiqueta 201 devolvió lectura unitaria';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.128',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'es');
  RAISE EXCEPTION 'CC11: detalle con etiqueta 201 no falló cerrado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.bad.en',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'en');
  RAISE EXCEPTION 'CC11: detalle con EN mal tipada no falló cerrado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.1',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'fr');
  RAISE EXCEPTION 'CC11: idioma del caller inválido aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,'categorias_refs',refs_128));
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es');
 IF comprobacion#>>'{0,catalogo_completo}'<>'false'
 THEN RAISE EXCEPTION 'CC11: etiqueta 201 ofertada en ES'; END IF;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en');
 IF comprobacion#>>'{0,catalogo_completo}'<>'false'
 THEN RAISE EXCEPTION 'CC11: etiqueta 201 ofertada en EN'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.bad.en')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es')#>>'{0,catalogo_completo}'<>'false'
 OR vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'false'
 THEN RAISE EXCEPTION 'CC11: EN mal tipada ofertada'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.en.larga')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es')#>>'{0,catalogo_completo}'<>'true'
 OR vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'false'
 THEN RAISE EXCEPTION 'CC11: límite por idioma divergente'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.fallback')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'true'
 OR vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
   '{"etiquetas":{"es":"Bien ES"}}','Base ES','en')<>'Bien ES'
 THEN RAISE EXCEPTION 'CC11: fallback ES rechazado'; END IF;
END $test$;
ROLLBACK;
