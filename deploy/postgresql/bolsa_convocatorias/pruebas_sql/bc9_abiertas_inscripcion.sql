\set ON_ERROR_STOP on
-- Fixture sintético de la tabla BC1/BC8 instalada. Siempre ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE id text; material jsonb; bytes bytea; referencia text; ref33 text; ref4 text; ref_largo text;
 pagina jsonb; segunda jsonb; detalle jsonb; posterior jsonb;
 f_lista regprocedure; f_detalle regprocedure; f_post regprocedure;
 i integer;
BEGIN
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
      'huella_contenido_sha256',repeat('b',64)),
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
 IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref4) IS NULL
 OR vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref4,'cat.alpha') IS NULL
 THEN RAISE EXCEPTION 'BC9: borrador posterior retiró la publicación original'; END IF;

 -- Un material abierto de 33 categorías nunca se ofrece parcialmente.
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

 pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,2);
 IF pagina->>'total'<>'4' OR pagina->>'hay_mas'<>'true'
 OR jsonb_array_length(pagina->'items')<>2
 OR pagina#>>'{items,1,convocatoria_ref}' IS DISTINCT FROM pagina->>'siguiente_cursor'
 THEN RAISE EXCEPTION 'BC9: primer total/cursor: %',pagina; END IF;
 segunda:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(pagina->>'siguiente_cursor',2);
 IF segunda->>'total'<>'4' OR segunda->>'hay_mas'<>'false'
 OR jsonb_array_length(segunda->'items')<>2
 OR segunda#>>'{items,0,convocatoria_ref}'=pagina#>>'{items,0,convocatoria_ref}'
 THEN RAISE EXCEPTION 'BC9: segunda página: %',segunda; END IF;
 IF NOT ((pagina->'items'||segunda->'items') @> jsonb_build_array(jsonb_build_object('convocatoria_ref',ref_largo)))
 OR vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref_largo) IS NULL
 THEN RAISE EXCEPTION 'BC9: convocatoria de 480 bytes oculta'; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref33,'cat.1');
  RAISE EXCEPTION 'BC9: POST de 33 categorías aceptado';
 EXCEPTION WHEN SQLSTATE 'B9607' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref33);
  RAISE EXCEPTION 'BC9: detalle de 33 categorías aceptado';
 EXCEPTION WHEN SQLSTATE 'B9607' THEN NULL; END;
 referencia:=pagina#>>'{items,0,convocatoria_ref}';
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(referencia);
 IF detalle->>'convocatoria_ref'<>referencia
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
END $test$;
ROLLBACK;
