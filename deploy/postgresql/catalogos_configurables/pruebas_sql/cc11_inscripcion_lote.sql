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
 f_asociacion regprocedure; f_gestion regprocedure; f_gestion_lote regprocedure;
 alcance text; clave_gestion text; otro_alcance text; otra_clave text; gestion jsonb;
BEGIN
 alcance:='cv1_'||encode(sha256(convert_to('proceso:bolsa:historica-2026','UTF8')),'hex')||'_v1';
 f:='vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text,text)'::regprocedure;
 f_categorias:='vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)'::regprocedure;
 f_asociacion:='vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)'::regprocedure;
 f_gestion:='vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)'::regprocedure;
 f_gestion_lote:='vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(jsonb)'::regprocedure;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_categorias,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_categorias,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_asociacion,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_asociacion,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_convocatorias_propietario',f_gestion,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',f_gestion,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_gestion,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_gestion_lote,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_propietario',f_gestion_lote,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_gestion_lote,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid IN(f,f_categorias,f_asociacion,f_gestion,f_gestion_lote) AND a.grantee=0)
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
   '{"clave":"cat.alpha","etiqueta":"Alfa ES","orden":1,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"etiqueta_en":"Alpha EN"}}'),
  ('bolsa.categorias.inscripcion',1,cat_sha,'cat.beta','Beta ES','{}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.con.pendientes','Política',
   '{"clave":"presentacion.con.pendientes","etiqueta":"Política","orden":1,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"valor":"true","canal":"externa_personal"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.con.pendientes.empleado','Política empleado',
   '{"clave":"presentacion.con.pendientes.empleado","etiqueta":"Política empleado","orden":2,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"valor":"true","canal":"interna_corporativa"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'requisito.pendiente','Pendiente ES',
   '{"clave":"requisito.pendiente","etiqueta":"Pendiente ES","orden":3,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"etiqueta_en":"Pending EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'requisito.cumple','Cumple ES',
   '{"clave":"requisito.cumple","etiqueta":"Cumple ES","orden":4,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"etiqueta_en":"Meets EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'solicitud.existente','Ya solicitada ES',
   '{"clave":"solicitud.existente","etiqueta":"Ya solicitada ES","orden":5,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"etiqueta_en":"Already applied EN"}}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.no.disponible','No disponible ES',
   '{"clave":"presentacion.no.disponible","etiqueta":"No disponible ES","orden":6,"vigente_desde":"2026-10-09T00:00:00Z","atributos":{"etiqueta_en":"Unavailable EN"}}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'requisito.pendiente','Pendiente ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'requisito.cumple','Cumple ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'solicitud.existente','Ya solicitada ES','{}'),
  ('bolsa.politica.sinpresentacion',1,pol_sha2,'presentacion.no.disponible','No disponible ES','{}'),
  ('bolsa.politica.inscripcion',1,pol_sha,'asociacion.manual.acta','Asociación manual',
   jsonb_build_object('clave','asociacion.manual.acta','etiqueta','Asociación manual',
    'orden',1,'vigente_desde','2026-10-09T00:00:00Z','atributos',
    jsonb_build_object('valor','true','canal','interna_corporativa',
     'motivo_ref','motivo:asociacion.manual','alcance_ref',alcance)));
 IF NOT vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
  'bolsa.politica.inscripcion',1,pol_sha,'externa_personal')
 THEN RAISE EXCEPTION 'CC11: política externa ausente'; END IF;
 -- Primera versión: sólo aspirantes externos. El canal empleado se rechaza
 -- aunque el catálogo publique una entrada «.empleado».
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,'interna_corporativa');
  RAISE EXCEPTION 'CC11: canal empleado admitido para presentar';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
   'bolsa.politica.sinpresentacion',1,pol_sha2,'externa_personal');
  RAISE EXCEPTION 'CC11: política externa prestada a otro catálogo';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 clave_gestion:='ambito.gestion.'||substring(alcance from '^cv1_([0-9a-f]{64})_v1$')||'.v1';
 otro_alcance:='cv1_'||repeat('b',64)||'_v1';
 otra_clave:='ambito.gestion.'||repeat('b',64)||'.v1';
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
  ('bolsa.politica.inscripcion',1,pol_sha,clave_gestion,'Ámbito RRHH',
   jsonb_build_object('clave',clave_gestion,'etiqueta','Ámbito RRHH',
    'orden',2,'vigente_desde','2026-10-09T00:00:00Z','atributos',
    jsonb_build_object('convocatoria_ref',alcance,'unidad_ref','unidad:rrhh-1',
     'ambito_ref','ambito:gestion-1','canal','interna_corporativa'))),
  ('bolsa.politica.inscripcion',1,pol_sha,otra_clave,'Ámbito ajeno',
   jsonb_build_object('clave',otra_clave,'etiqueta','Ámbito ajeno',
    'orden',3,'vigente_desde','2026-10-09T00:00:00Z','atributos',
    jsonb_build_object('convocatoria_ref',alcance,'unidad_ref','unidad:rrhh-1',
     'ambito_ref','ambito:gestion-1','canal','interna_corporativa'))),
  ('bolsa.politica.inscripcion',1,pol_sha,'ambito.gestion.'||repeat('c',64)||'.v1',
   'Ámbito deshabilitado',jsonb_build_object(
    'clave','ambito.gestion.'||repeat('c',64)||'.v1',
    'etiqueta','Ámbito deshabilitado','orden',4,'vigente_desde','2026-10-09T00:00:00Z',
    'atributos',jsonb_build_object('convocatoria_ref','cv1_'||repeat('c',64)||'_v1',
     'unidad_ref','unidad:rrhh-1','ambito_ref','ambito:gestion-1','canal','interna_corporativa'))),
  ('bolsa.politica.inscripcion',1,pol_sha,'ambito.gestion.'||repeat('e',64)||'.v1',
   'Ámbito con exceso',jsonb_build_object(
    'clave','ambito.gestion.'||repeat('e',64)||'.v1',
    'etiqueta','Ámbito con exceso','orden',5,'vigente_desde','2026-10-09T00:00:00Z',
    'atributos',jsonb_build_object('convocatoria_ref','cv1_'||repeat('e',64)||'_v1',
     'unidad_ref','unidad:rrhh-1','ambito_ref','ambito:gestion-1',
     'canal','interna_corporativa','otro','no admitido'))),
  ('bolsa.politica.inscripcion',1,pol_sha,'ambito.gestion.'||repeat('f',64)||'.v1',
   'Ámbito futuro',jsonb_build_object(
    'clave','ambito.gestion.'||repeat('f',64)||'.v1',
    'etiqueta','Ámbito futuro','orden',6,'vigente_desde','2099-01-01T00:00:00Z',
    'atributos',jsonb_build_object('convocatoria_ref','cv1_'||repeat('f',64)||'_v1',
     'unidad_ref','unidad:rrhh-1','ambito_ref','ambito:gestion-1',
     'canal','interna_corporativa')));
 INSERT INTO vec_catalogos_configurables.categoria_control
  (categoria_id,catalogo_id,version,huella_sha256,revision,estado)
 VALUES
  (clave_gestion,'bolsa.politica.inscripcion',1,pol_sha,1,'habilitada'),
  (otra_clave,'bolsa.politica.inscripcion',1,pol_sha,1,'habilitada'),
  ('ambito.gestion.'||repeat('c',64)||'.v1','bolsa.politica.inscripcion',1,pol_sha,1,'deshabilitada'),
  ('ambito.gestion.'||repeat('e',64)||'.v1','bolsa.politica.inscripcion',1,pol_sha,1,'habilitada'),
  ('ambito.gestion.'||repeat('f',64)||'.v1','bolsa.politica.inscripcion',1,pol_sha,1,'habilitada');
 gestion:=vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
  'bolsa.politica.inscripcion',1,pol_sha,alcance);
 IF gestion->>'unidad_ref'<>'unidad:rrhh-1' OR gestion->>'ambito_ref'<>'ambito:gestion-1'
 OR gestion->>'fuente_ref'<>clave_gestion OR gestion->>'fuente_version'<>'1'
 OR gestion->>'fuente_sha256'<>pol_sha
 THEN RAISE EXCEPTION 'CC11: ámbito de gestión incompleto: %',gestion; END IF;
 comprobacion:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  jsonb_build_array(
   jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
    'convocatoria_ref',alcance),
   jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
    'convocatoria_ref',otro_alcance),
   jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
    'convocatoria_ref','cv1_'||repeat('c',64)||'_v1'),
   jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
    'convocatoria_ref','cv1_'||repeat('d',64)||'_v1'),
   jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',repeat('0',64),
    'convocatoria_ref',alcance)));
 IF jsonb_array_length(comprobacion)<>5
 OR comprobacion#>>'{0,ambito_gestion_disponible}'<>'true'
 OR comprobacion#>>'{1,ambito_gestion_disponible}'<>'false'
 OR comprobacion#>>'{2,ambito_gestion_disponible}'<>'false'
 OR comprobacion#>>'{3,ambito_gestion_disponible}'<>'false'
 OR comprobacion#>>'{4,ambito_gestion_disponible}'<>'false'
 OR comprobacion->0 ? 'unidad_ref' OR comprobacion->0 ? 'ambito_ref'
 THEN RAISE EXCEPTION 'CC11: lote de ámbitos incorrecto o filtrado: %',comprobacion; END IF;
 -- CC1 tiene una sola fila de control actual por categoría; una revisión
 -- posterior deshabilitada sustituye r1 aunque la entrada siga publicada.
 UPDATE vec_catalogos_configurables.categoria_control
 SET estado='deshabilitada',revision=2
 WHERE categoria_id=clave_gestion AND revision=1 AND estado='habilitada';
 IF NOT FOUND THEN RAISE EXCEPTION 'CC11: control vigente no actualizado'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,alcance);
  RAISE EXCEPTION 'CC11: control r2 deshabilitado aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 comprobacion:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
   'convocatoria_ref',alcance)));
 IF comprobacion#>>'{0,ambito_gestion_disponible}'<>'false'
 THEN RAISE EXCEPTION 'CC11: lote admitió control r2 deshabilitado'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',2,pol_sha,alcance);
  RAISE EXCEPTION 'CC11: versión de ámbito ausente aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,repeat('0',64),alcance);
  RAISE EXCEPTION 'CC11: SHA de ámbito equivocado aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,otro_alcance);
  RAISE EXCEPTION 'CC11: ámbito de otra convocatoria aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,'cv1_'||repeat('c',64)||'_v1');
  RAISE EXCEPTION 'CC11: ámbito deshabilitado aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,'cv1_'||repeat('e',64)||'_v1');
  RAISE EXCEPTION 'CC11: definición abierta aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,'cv1_'||repeat('d',64)||'_v1');
  RAISE EXCEPTION 'CC11: ámbito ausente aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
   'bolsa.politica.inscripcion',1,pol_sha,'cv1_'||repeat('f',64)||'_v1');
  RAISE EXCEPTION 'CC11: ámbito futuro aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 comprobacion:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha,
   'convocatoria_ref','cv1_'||repeat('f',64)||'_v1')));
 IF comprobacion#>>'{0,ambito_gestion_disponible}'<>'false'
 THEN RAISE EXCEPTION 'CC11: lote admitió ámbito futuro'; END IF;
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
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1('[]','es','externa_personal');
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
  'en','externa_personal');
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
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.alpha',
   'politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),
  'es','interna_corporativa');
 -- Canal RRHH/empleado: lee etiquetas, pero nunca habilita presentar.
 IF r#>>'{0,politica_valida}'<>'false' OR r#>>'{0,categoria}'<>'Alfa ES'
 THEN RAISE EXCEPTION 'CC11: canal interno habilitó presentación: %',r; END IF;
 SELECT jsonb_agg(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.alpha',
  'politica_catalogo_ref','bolsa.politica.inscripcion',
  'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha))
 INTO lote_maximo FROM generate_series(1,12800);
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(lote_maximo,'es','externa_personal');
 IF jsonb_array_length(r)<>12800 OR r#>>'{12799,motivo_etiqueta_pendiente}'<>'Pendiente ES'
 OR r#>>'{12799,motivo_etiqueta_cumple}'<>'Cumple ES'
 THEN RAISE EXCEPTION 'CC11: página máxima truncada'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   lote_maximo||jsonb_build_array(lote_maximo->0),'es','externa_personal');
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
      encode(sha256(convert_to('{"id":"bolsa.politica.sinmotivo"}','UTF8')),'hex'))),'es','externa_personal');
  RAISE EXCEPTION 'CC11: motivo de requisito ausente aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL;
 END;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.ausente',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'es','externa_personal');
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
 -- CC1 admite hasta 2048 bytes; el resumen histórico lee la etiqueta,
 -- mientras la oferta de formulario <=200 se deniega sin ocultar la fila.
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
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 SELECT 'bolsa.categorias.inscripcion',1,cat_sha,'cat.max.'||g,repeat('L',2048),'{}'::jsonb
 FROM generate_series(1,128) AS g;
 SELECT jsonb_agg(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.max.'||g,
  'politica_catalogo_ref','bolsa.politica.inscripcion',
  'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha) ORDER BY g)
 INTO lote_maximo FROM generate_series(1,128) AS g;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  lote_maximo,'es','externa_personal');
 IF jsonb_array_length(r)<>128
 OR octet_length(r#>>'{0,categoria}')<>2048
 OR octet_length(r#>>'{127,categoria}')<>2048
 THEN RAISE EXCEPTION 'CC11: detalle 128×2048 truncado'; END IF;
 SELECT jsonb_agg('cat.max.'||g ORDER BY g) INTO refs_128 FROM generate_series(1,128) AS g;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',cat_sha,'categorias_refs',refs_128)),'es');
 IF comprobacion#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: 128 etiquetas 2048 bloquearon elegibilidad'; END IF;
 SELECT jsonb_agg('cat.'||g ORDER BY g) INTO refs_128 FROM generate_series(1,128) AS g;
 r:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  'bolsa.categorias.inscripcion',1,cat_sha,ARRAY['cat.128'],'es');
 IF r#>>'{0,categoria}'<>repeat('X',201)
 THEN RAISE EXCEPTION 'CC11: recibo histórico perdió categoría 201'; END IF;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.128',
   'politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'es','externa_personal');
 IF r#>>'{0,categoria}'<>repeat('X',201)
 OR r#>>'{0,politica_valida}'<>'true'
 THEN RAISE EXCEPTION 'CC11: lista/detalle ocultó categoría 201'; END IF;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.bad.en',
   'politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'en','externa_personal');
 IF r#>>'{0,categoria}'<>'Base ES'
 THEN RAISE EXCEPTION 'CC11: EN histórica inválida sin fallback ES'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.1',
    'politica_catalogo_ref','bolsa.politica.inscripcion',
    'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),'fr','externa_personal');
  RAISE EXCEPTION 'CC11: idioma del caller inválido aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,'categorias_refs',refs_128));
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es');
 IF comprobacion#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: etiqueta 201 bloqueó refs publicadas en ES'; END IF;
 comprobacion:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en');
 IF comprobacion#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: etiqueta 201 bloqueó refs publicadas en EN'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.bad.en')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es')#>>'{0,catalogo_completo}'<>'true'
 OR vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: EN mal tipada bloqueó ref publicada'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.en.larga')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'es')#>>'{0,catalogo_completo}'<>'true'
 OR vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: EN201 bloqueó ref publicada'; END IF;
 r:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
   'catalogo_version',1,'catalogo_sha256',cat_sha,'categoria_ref','cat.en.larga',
   'politica_catalogo_ref','bolsa.politica.inscripcion',
   'politica_catalogo_version',1,'politica_catalogo_sha256',pol_sha)),
  'en','externa_personal');
 IF octet_length(r#>>'{0,categoria}')<>201
 THEN RAISE EXCEPTION 'CC11: EN201 histórica truncada'; END IF;
 selector:=jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.inscripcion',
  'catalogo_version',1,'catalogo_sha256',cat_sha,
  'categorias_refs',jsonb_build_array('cat.fallback')));
 IF vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(selector,'en')#>>'{0,catalogo_completo}'<>'true'
 OR vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
   '{"etiquetas":{"es":"Bien ES"}}','Base ES','en')<>'Bien ES'
 THEN RAISE EXCEPTION 'CC11: fallback ES rechazado'; END IF;
 INSERT INTO vec_catalogos_configurables.publicacion
  (catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
   preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES('bolsa.inscripcion.motivos',1,
  encode(sha256(convert_to('{"id":"bolsa.inscripcion.motivos"}','UTF8')),'hex'),
  '{"id":"bolsa.inscripcion.motivos"}','{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:mot:a','aprobacion:mot:b','actor:cc11','decision:mot','recibo:cc11:mot');
 INSERT INTO vec_catalogos_configurables.entrada_publicada
  (catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES('bolsa.inscripcion.motivos',1,
  encode(sha256(convert_to('{"id":"bolsa.inscripcion.motivos"}','UTF8')),'hex'),
  'motivo.largo',repeat('M',201),'{}');
 BEGIN
  PERFORM vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   'bolsa.inscripcion.motivos',1,
   encode(sha256(convert_to('{"id":"bolsa.inscripcion.motivos"}','UTF8')),'hex'),
   ARRAY['motivo.largo'],'es');
  RAISE EXCEPTION 'CC11: motivo 201 aceptado como formulario';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
END $test$;
ROLLBACK;
