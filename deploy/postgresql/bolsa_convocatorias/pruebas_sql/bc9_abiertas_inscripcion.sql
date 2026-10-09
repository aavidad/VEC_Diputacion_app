\set ON_ERROR_STOP on
-- Fixture sintético de la tabla BC1/BC8 instalada. Siempre ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE id text; material jsonb; bytes bytea; referencia text; ref33 text; ref128 text; ref4 text; ref_largo text;
 pol_doc text; pol_sha text; pol_entradas jsonb;
 pagina jsonb; segunda jsonb; tercera jsonb; todos jsonb; item jsonb; detalle jsonb; posterior jsonb;
 f_lista regprocedure; f_detalle regprocedure; f_post regprocedure;
 i integer;
BEGIN
 SELECT jsonb_agg(jsonb_build_object(
  'clave','ambito.gestion.'||encode(sha256(convert_to(q.id,'UTF8')),'hex')||'.v1',
  'etiqueta','Ámbito RRHH','orden',q.orden,'vigente_desde','2026-10-09T00:00:00Z',
  'atributos',jsonb_build_object(
   'convocatoria_ref','cv1_'||encode(sha256(convert_to(q.id,'UTF8')),'hex')||'_v1',
   'unidad_ref','unidad:rrhh-1','ambito_ref','ambito:gestion-1',
   'canal','interna_corporativa')) ORDER BY q.orden)
 INTO pol_entradas FROM (
  SELECT 'proceso:bolsa:inscripcion-'||gs.valor AS id,gs.valor AS orden
  FROM generate_series(1,6) AS gs(valor)
  UNION ALL SELECT repeat('a',480),7
 ) AS q;
 pol_doc:=jsonb_build_object('id','bolsa.politica.inscripcion','version',1,
  'revision',1,'modulo_id','bolsa','nombre','Política sintética',
  'fuente_ref','fuente:sintetica','motivo_creacion','Prueba BC9',
  'entradas',pol_entradas,'estado','publicado','creado_por','persona:sintetica-1',
  'creado_en','2026-10-09T00:00:00Z','publicado_por','persona:sintetica-2',
  'publicado_en','2026-10-09T00:00:00Z',
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.politica.inscripcion',1,
  pol_sha,pol_doc,'{}'::jsonb,encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:bc9:a','aprobacion:bc9:b','actor:bc9','decision:bc9',
  'recibo:bc9','motivos_bolsa:1:motivo_demo');
 f_lista:='vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(text,integer)'::regprocedure;
 f_detalle:='vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(text)'::regprocedure;
 f_post:='vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)'::regprocedure;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_lista,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_detalle,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_post,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f_lista,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f_detalle,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid IN(f_lista,f_detalle,f_post) AND a.grantee=0)
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_bolsa_convocatorias.huella_id_inscripcion_v1(text)','EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_index i
  WHERE i.indexrelid='vec_bolsa_convocatorias.version_convocatoria_token_inscripcion_uq'::regclass
   AND i.indisunique AND i.indisvalid)
 THEN RAISE EXCEPTION 'BC9: ACL de lectores abierta'; END IF;
 FOR i IN 1..4 LOOP
  id:='proceso:bolsa:inscripcion-'||i;
  material:=jsonb_build_object('id',id,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en',statement_timestamp(),
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',id||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',id||'#1'),
   'contenido',jsonb_build_object(
    'identificador_publico','bolsa-test-'||i,'tipo','bolsa',
    'titulo','Bolsa sintética '||i,'resumen','Resumen sintético',
    'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.inscripcion',
      'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64)),
    'categorias',jsonb_build_array('cat.alpha','cat.beta'),
    'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo:inscripcion',
      'tipo','inscripcion','abre_en',statement_timestamp()-interval '1 day',
      'cierra_en',CASE WHEN i=3 THEN statement_timestamp()-interval '1 hour'
                       ELSE statement_timestamp()+interval '1 day' END)),
    'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada',
     'orden',1,'descripcion','Identidad con certificado','obligatorio',true))),
   'configuracion',jsonb_build_object(
    'catalogos',jsonb_build_object('id','bolsa.politica.inscripcion','version',1,
      'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo:inscripcion','version',1,
      'huella_contenido_sha256',repeat('c',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases',
      'publicacion_ref','bases:inscripcion-'||i))));
  bytes:=convert_to(material::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria
   (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
  VALUES (id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());
 END LOOP;
 -- Un borrador posterior no retira por sí mismo la publicación anterior.
 id:='proceso:bolsa:inscripcion-4';
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 SELECT id,2,id||'#2','borrador',version_canonica,huella_version_sha256,statement_timestamp()
 FROM vec_bolsa_convocatorias.version_convocatoria WHERE convocatoria_id=id AND secuencia=1;
 ref4:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';

 -- Una publicación de 33 categorías se ofrece y conserva todas.
 id:='proceso:bolsa:inscripcion-5';
 ref33:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';
 material:=jsonb_set(material,'{id}',to_jsonb(id));
 material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{contenido,identificador_publico}','"bolsa-test-5"'::jsonb);
 material:=jsonb_set(material,'{contenido,categorias}',
  (SELECT jsonb_agg('cat.'||g ORDER BY g) FROM generate_series(1,33) AS g));
 bytes:=convert_to(material::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());

 id:='proceso:bolsa:inscripcion-6';
 ref128:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';
 material:=jsonb_set(material,'{id}',to_jsonb(id));
 material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{contenido,identificador_publico}','"bolsa-test-6"'::jsonb);
 material:=jsonb_set(material,'{contenido,categorias}',
  (SELECT jsonb_agg('cat.'||g ORDER BY g) FROM generate_series(1,128) AS g));
 bytes:=convert_to(material::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());

 -- El token de hash mantiene visible un identificador válido de 480 bytes.
 id:=repeat('a',480);
 ref_largo:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';
 IF octet_length(ref_largo)>200 OR ref_largo LIKE '%'||id||'%'
 THEN RAISE EXCEPTION 'BC9: token largo o reversible'; END IF;
 material:=jsonb_set(material,'{id}',to_jsonb(id));
 material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{contenido,categorias}',jsonb_build_array('cat.alpha','cat.beta'));
 bytes:=convert_to(material::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());

 IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref4) IS NULL
 OR vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref4,'cat.alpha') IS NULL
 THEN RAISE EXCEPTION 'BC9: borrador posterior retiró la publicación original'; END IF;

 pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,2);
 IF pagina->>'total'<>'6' OR pagina->>'hay_mas'<>'true'
 OR jsonb_array_length(pagina->'items')<>2
 OR pagina#>>'{items,1,convocatoria_ref}' IS DISTINCT FROM pagina->>'siguiente_cursor'
 THEN RAISE EXCEPTION 'BC9: primer total/cursor: %',pagina; END IF;
 segunda:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(pagina->>'siguiente_cursor',2);
 IF segunda->>'total'<>'6' OR segunda->>'hay_mas'<>'true'
 OR jsonb_array_length(segunda->'items')<>2
 OR segunda#>>'{items,0,convocatoria_ref}'=pagina#>>'{items,0,convocatoria_ref}'
 THEN RAISE EXCEPTION 'BC9: segunda página: %',segunda; END IF;
 tercera:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(segunda->>'siguiente_cursor',2);
 IF tercera->>'total'<>'6' OR tercera->>'hay_mas'<>'false'
 OR jsonb_array_length(tercera->'items')<>2
 THEN RAISE EXCEPTION 'BC9: tercera página: %',tercera; END IF;
 todos:=pagina->'items'||segunda->'items'||tercera->'items';
 IF NOT (todos @> jsonb_build_array(jsonb_build_object('convocatoria_ref',ref_largo)))
 OR NOT (todos @> jsonb_build_array(jsonb_build_object('convocatoria_ref',ref33)))
 OR NOT (todos @> jsonb_build_array(jsonb_build_object('convocatoria_ref',ref128)))
 OR vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref_largo) IS NULL
 THEN RAISE EXCEPTION 'BC9: convocatoria abierta omitida'; END IF;
 FOR item IN SELECT valor FROM jsonb_array_elements(todos) AS x(valor) LOOP
  IF item ? 'categorias' OR item ? 'requisitos'
   OR item ? 'convocatoria_id' OR item ? 'identificador_publico'
   OR item ? 'bases_ref' OR item ? 'version_sha256'
   OR item ? 'unidad_ref' OR item ? 'ambito_ref' OR item ? 'fuente_ref'
   OR coalesce(item->>'categoria_ref_comprobacion','')=''
   OR (item->>'numero_categorias')::integer NOT BETWEEN 1 AND 128
   OR jsonb_typeof(item->'categorias_refs_comprobacion') IS DISTINCT FROM 'array'
   OR jsonb_array_length(item->'categorias_refs_comprobacion')<>(item->>'numero_categorias')::integer
   OR (item->>'numero_requisitos')::integer<>1
   OR coalesce(item->>'titulo','')=''
   OR item->>'plazo_abre_en' IS NULL OR item->>'plazo_cierra_en' IS NULL
  THEN RAISE EXCEPTION 'BC9: ítem de lista no compacto o incompleto: %',item; END IF;
 END LOOP;
 SELECT valor INTO item FROM jsonb_array_elements(todos) AS x(valor)
  WHERE valor->>'convocatoria_ref'=ref33;
 IF item->>'numero_categorias'<>'33' OR item->>'categoria_ref_comprobacion'<>'cat.1'
 OR item#>>'{categorias_refs_comprobacion,32}'<>'cat.33'
 THEN RAISE EXCEPTION 'BC9: lista 33 categorías resumida mal: %',item; END IF;
 SELECT valor INTO item FROM jsonb_array_elements(todos) AS x(valor)
  WHERE valor->>'convocatoria_ref'=ref128;
 IF item->>'numero_categorias'<>'128' OR item->>'categoria_ref_comprobacion'<>'cat.1'
 OR item#>>'{categorias_refs_comprobacion,127}'<>'cat.128'
 THEN RAISE EXCEPTION 'BC9: lista 128 categorías resumida mal: %',item; END IF;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref33);
 posterior:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref33,'cat.33');
 IF detalle->>'numero_categorias'<>'33' OR detalle ? 'categoria_ref_comprobacion'
 OR detalle ? 'categorias_refs_comprobacion'
 OR detalle->>'numero_requisitos'<>'1' OR jsonb_array_length(detalle->'requisitos')<>1
 OR jsonb_array_length(detalle->'categorias')<>33
 OR jsonb_array_length(posterior->'categorias')<>33
 THEN RAISE EXCEPTION 'BC9: 33 categorías truncadas'; END IF;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref128);
 posterior:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref128,'cat.128');
 IF detalle->>'numero_categorias'<>'128' OR detalle ? 'categoria_ref_comprobacion'
 OR detalle ? 'categorias_refs_comprobacion'
 OR detalle->>'numero_requisitos'<>'1' OR jsonb_array_length(detalle->'requisitos')<>1
 OR jsonb_array_length(detalle->'categorias')<>128
 OR jsonb_array_length(posterior->'categorias')<>128
 THEN RAISE EXCEPTION 'BC9: 128 categorías truncadas'; END IF;
 referencia:=ref4;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(referencia);
 IF detalle->>'convocatoria_ref'<>referencia
 OR detalle ? 'unidad_ref' OR detalle ? 'ambito_ref' OR detalle ? 'fuente_ref'
 OR jsonb_array_length(detalle->'categorias')<>2
 OR detalle#>>'{categorias,0,categoria_ref}'<>'cat.alpha'
 OR detalle#>>'{requisitos,0,estado}'<>'pendiente'
 OR detalle#>>'{requisitos,0,motivo_codigo}'<>'requisito.pendiente'
 OR detalle#>>'{requisitos,0,descripcion}'<>'Identidad con certificado'
 OR detalle->>'catalogo_ref'<>'bolsa.categorias.inscripcion'
 THEN RAISE EXCEPTION 'BC9: detalle sin categorías/requisitos: %',detalle; END IF;
 posterior:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(referencia,'cat.beta');
 IF posterior->>'categoria_ref'<>'cat.beta'
 OR posterior#>>'{requisitos,0,estado}'<>'pendiente'
 OR posterior->>'unidad_ref'<>'unidad:rrhh-1'
 OR posterior->>'ambito_ref'<>'ambito:gestion-1'
 OR posterior->>'fuente_ref' IS DISTINCT FROM
    'ambito.gestion.'||encode(sha256(convert_to('proceso:bolsa:inscripcion-4','UTF8')),'hex')||'.v1'
 OR posterior->>'fuente_version'<>'1'
 OR posterior->>'fuente_sha256'<>pol_sha
 THEN RAISE EXCEPTION 'BC9: POST usa otra versión: %',posterior; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(referencia,'cat.ajena');
  RAISE EXCEPTION 'BC9: POST aceptó categoría ajena';
 EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   referencia,'cat.ajena',posterior->>'version_sha256');
  RAISE EXCEPTION 'BC9: historia aceptó categoría ajena';
 EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL; END;
 IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(
   'cv1_'||encode(sha256(convert_to('proceso:bolsa:inscripcion-3','UTF8')),'hex')||'_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'BC9: detalle cerrado visible'; END IF;
 id:='proceso:bolsa:sin-ambito-2026';
 referencia:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';
 material:=jsonb_set(material,'{id}',to_jsonb(id));
 material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#1'));
 material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#1'));
 bytes:=convert_to(material::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(referencia,'cat.alpha');
  RAISE EXCEPTION 'BC9: POST sin ámbito gobernado aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
END $test$;
ROLLBACK;
