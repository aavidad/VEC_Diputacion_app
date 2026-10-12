\set ON_ERROR_STOP on
-- BC11 sobre BC9/CC11 en PostgreSQL 18 sintetico. Siempre ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='20s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE id text; ref text; material jsonb; bytes bytea; sha text;
 pol_doc text; pol_sha text; entradas jsonb; modalidad text;
 ver integer; i integer; respuesta jsonb; lista jsonb; detalle jsonb;
 ref_sep text; ref_joint text; ref_legacy text; ref_v2 text;
 f regprocedure; eleccion text[];
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(text,text[])'::regprocedure,
  'vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(text,text[],text)'::regprocedure,
  'vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(text,integer)'::regprocedure,
  'vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(text)'::regprocedure]
 LOOP
  IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
   OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee=0)
  THEN RAISE EXCEPTION 'BC11: ACL de consumidor abierta: %',f; END IF;
 END LOOP;
 IF has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])','EXECUTE')
 OR has_table_privilege('vec_bolsa_llamamientos_propietario',
  'vec_bolsa_convocatorias.version_convocatoria','SELECT')
 THEN RAISE EXCEPTION 'BC11: helper o tabla expuestos'; END IF;

 SELECT jsonb_agg(jsonb_build_object('clave',
  'ambito.gestion.'||encode(sha256(convert_to(q.id,'UTF8')),'hex')||'.v1',
  'etiqueta','Ambito sintético','orden',q.orden,
  'vigente_desde','2026-10-09T00:00:00Z',
  'atributos',jsonb_build_object('convocatoria_ref',
   'cv1_'||encode(sha256(convert_to(q.id,'UTF8')),'hex')||'_v1',
   'unidad_ref','unidad:rrhh-1','ambito_ref','ambito:gestion-1',
   'canal','interna_corporativa')) ORDER BY q.orden)
 INTO entradas FROM (
  SELECT 'proceso:bc11:'||g.valor AS id,g.valor AS orden
  FROM generate_series(1,3) AS g(valor)
 ) q;
 pol_doc:=jsonb_build_object('id','bolsa.politica.inscripcion','version',1,
  'revision',1,'modulo_id','bolsa','nombre','Politica sintetica',
  'fuente_ref','fuente:sintetica','motivo_creacion','Prueba BC11',
  'entradas',entradas,'estado','publicado','creado_por','persona:sintetica-1',
  'creado_en','2026-10-09T00:00:00Z','publicado_por','persona:sintetica-2',
  'publicado_en','2026-10-09T00:00:00Z',
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.politica.inscripcion',1,
  pol_sha,pol_doc,'{}'::jsonb,encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:bc11:a','aprobacion:bc11:b','actor:bc11','decision:bc11',
  'recibo:bc11','motivos_bolsa:1:motivo_demo');

 FOR i IN 1..3 LOOP
  id:='proceso:bc11:'||i;
  ref:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1';
  IF i=1 THEN modalidad:='separada'; ref_sep:=ref;
  ELSIF i=2 THEN modalidad:='conjunta'; ref_joint:=ref;
  ELSE modalidad:=NULL; ref_legacy:=ref; END IF;
  material:=jsonb_build_object('id',id,'secuencia',1,
   'estado_gobierno','publicada','publicada_en',statement_timestamp(),
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',id||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',id||'#1'),
   'contenido',jsonb_build_object('identificador_publico','bc11-'||i,
    'tipo','bolsa','titulo','Bolsa sintética '||i,'resumen','Resumen sintético',
    'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.inscripcion',
     'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64)),
    'categorias',jsonb_build_array('cat.alpha','cat.beta'),
    'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo:inscripcion',
     'tipo','inscripcion','abre_en',statement_timestamp()-interval '1 day',
     'cierra_en',statement_timestamp()+interval '1 day')),
    'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada',
     'orden',1,'descripcion','Identidad certificada','obligatorio',true))),
   'configuracion',jsonb_build_object('catalogos',jsonb_build_object(
    'id','bolsa.politica.inscripcion','version',1,'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo:inscripcion','version',1,
     'huella_contenido_sha256',repeat('c',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases',
     'publicacion_ref','bases:bc11:'||i,'documento_ref','documento:bc11:'||i,
     'version_documento',1,'huella_contenido_sha256',repeat('d',64)))));
  IF modalidad IS NOT NULL THEN
   material:=jsonb_set(material,'{contenido,modalidad_inscripcion}',to_jsonb(modalidad));
  END IF;
  bytes:=convert_to(material::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria
   (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
  VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());
 END LOOP;

 respuesta:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
  ref_sep,ARRAY['cat.beta']);
 IF respuesta->>'modalidad_inscripcion'<>'separada'
  OR respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.beta')
  OR respuesta->>'bases_documento_ref'<>'documento:bc11:1'
  OR respuesta->>'bases_version_documento'<>'1'
  OR respuesta->>'bases_sha256'<>repeat('d',64)
  OR respuesta->>'version_sha256' IS NULL
 THEN RAISE EXCEPTION 'BC11: separada incompleta: %',respuesta; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
   ref_sep,ARRAY['cat.alpha','cat.beta']);
  RAISE EXCEPTION 'BC11: separada acepto dos categorias';
 EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL; END;
 respuesta:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
  ref_joint,ARRAY['cat.beta','cat.alpha']);
 IF respuesta->>'modalidad_inscripcion'<>'conjunta'
  OR respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.alpha','cat.beta')
  OR respuesta->>'bases_ref'<>'bases:bc11:2'
 THEN RAISE EXCEPTION 'BC11: conjunta incompleta: %',respuesta; END IF;
 respuesta:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
  ref_joint,ARRAY['cat.alpha']);
 IF respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.alpha')
 THEN RAISE EXCEPTION 'BC11: conjunta no acepto subconjunto: %',respuesta; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
   ref_joint,ARRAY['cat.alpha','cat.alpha']);
  RAISE EXCEPTION 'BC11: categorias duplicadas aceptadas';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
   ref_joint,ARRAY['cat.alpha','cat.ajena']);
  RAISE EXCEPTION 'BC11: categoria ajena aceptada';
 EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
   ref_joint,ARRAY[]::text[]);
  RAISE EXCEPTION 'BC11: conjunto vacio aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
   ref_legacy,ARRAY['cat.alpha']);
  RAISE EXCEPTION 'BC11: modalidad ausente aceptada como nueva';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;

 lista:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(NULL,10);
 IF lista->>'total'<>'3' OR jsonb_array_length(lista->'items')<>3
  OR NOT (lista->'items' @> jsonb_build_array(jsonb_build_object(
   'convocatoria_ref',ref_joint,'modalidad_inscripcion','conjunta')))
  OR NOT (lista->'items' @> jsonb_build_array(jsonb_build_object(
   'convocatoria_ref',ref_legacy,'modalidad_inscripcion',NULL)))
 THEN RAISE EXCEPTION 'BC11: lista incompleta: %',lista; END IF;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(ref_joint);
 IF detalle->>'modalidad_inscripcion'<>'conjunta'
  OR detalle->'categorias_refs'<>jsonb_build_array('cat.alpha','cat.beta')
  OR detalle->>'version_sha256' IS NULL
  OR detalle->>'bases_documento_ref'<>'documento:bc11:2'
 THEN RAISE EXCEPTION 'BC11: detalle incorrecto: %',detalle; END IF;

 -- La version original sigue marcada publicada; una sucesora posterior la
 -- desplaza como vigente sin reetiquetar su fila. El replay conserva su huella.
 id:='proceso:bc11:1'; ref_v2:='cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v2';
 SELECT convert_from(version_canonica,'UTF8')::jsonb INTO material
 FROM vec_bolsa_convocatorias.version_convocatoria
 WHERE convocatoria_id=id AND secuencia=1;
 material:=jsonb_set(material,'{secuencia}','2'::jsonb);
 material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#2'));
 material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#2'));
 material:=jsonb_set(material,'{contenido,modalidad_inscripcion}','"conjunta"'::jsonb);
 bytes:=convert_to(material::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,2,id||'#2','publicada',bytes,encode(sha256(bytes),'hex'),statement_timestamp());
 SELECT huella_version_sha256 INTO sha FROM vec_bolsa_convocatorias.version_convocatoria
 WHERE convocatoria_id=id AND secuencia=1;
 IF sha=encode(sha256(bytes),'hex')
 THEN RAISE EXCEPTION 'BC11: la modalidad no cambio la huella'; END IF;
 respuesta:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
  ref_sep,ARRAY['cat.beta'],sha);
 IF respuesta->>'modalidad_inscripcion'<>'separada' OR respuesta->>'version_sha256'<>sha
 THEN RAISE EXCEPTION 'BC11: replay V1 alterado: %',respuesta; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
   ref_sep,ARRAY['cat.beta'],encode(sha256(bytes),'hex'));
  RAISE EXCEPTION 'BC11: huella de sucesora aceptada sobre V1';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 respuesta:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
  ref_v2,ARRAY['cat.beta','cat.alpha'],encode(sha256(bytes),'hex'));
 IF respuesta->>'modalidad_inscripcion'<>'conjunta'
  OR respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.alpha','cat.beta')
 THEN RAISE EXCEPTION 'BC11: replay V2 incorrecto: %',respuesta; END IF;
 respuesta:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
  ref_v2,ARRAY['cat.beta'],encode(sha256(bytes),'hex'));
 IF respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.beta')
 THEN RAISE EXCEPTION 'BC11: replay conjunto parcial incorrecto: %',respuesta; END IF;
 SELECT huella_version_sha256 INTO sha FROM vec_bolsa_convocatorias.version_convocatoria
 WHERE convocatoria_id='proceso:bc11:3' AND secuencia=1;
 respuesta:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
  ref_legacy,ARRAY['cat.alpha'],sha);
 IF respuesta->'modalidad_inscripcion'<>'null'::jsonb
  OR respuesta->'categorias_refs_elegidas'<>jsonb_build_array('cat.alpha')
 THEN RAISE EXCEPTION 'BC11: legado convertido a modalidad nueva: %',respuesta; END IF;
END $test$;
ROLLBACK;
