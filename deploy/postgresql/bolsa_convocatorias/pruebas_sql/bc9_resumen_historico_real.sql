\set ON_ERROR_STOP on
-- CC1 real + CC11 + BC9 en PostgreSQL 18 desechable. Sin solicitud ni PII:
-- B96 aporta únicamente refs de su página ya autorizada. Siempre ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='20s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE doc text; huella_cat text; vacia_sha text:=encode(sha256(convert_to('{}','UTF8')),'hex');
 id_a text:='proceso:bolsa:hist-a'; id_b text:='proceso:bolsa:hist-b';
 id_c text:='proceso:bolsa:hist-c'; r record; canonica jsonb; bytes bytea;
 refs text[]; resultado jsonb; resultado_en jsonb;
 f regprocedure:='vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(text[],text)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
  aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee=0)
 THEN RAISE EXCEPTION 'BC9: ACL de resumen histórico abierta'; END IF;
 doc:=jsonb_build_object('id','bolsa.categorias.historicas','version',1,
  'revision',1,'modulo_id','bolsa','nombre','Categorías históricas',
  'fuente_ref','fuente:sintetica','motivo_creacion','Prueba',
  'entradas',jsonb_build_array(
   jsonb_build_object('clave','cat.long','etiqueta',repeat('A',2048),
    'orden',1,'vigente_desde','2026-10-09T00:00:00Z',
    'atributos',jsonb_build_object('etiqueta_en',42)),
   jsonb_build_object('clave','cat.good','etiqueta','Categoría nueva ES',
    'orden',2,'vigente_desde','2026-10-09T00:00:00Z',
    'atributos',jsonb_build_object('etiqueta_en','New category EN'))),
  'estado','publicado','creado_por','persona:uno',
  'creado_en','2026-10-09T00:00:00Z','publicado_por','persona:dos',
  'publicado_en','2026-10-09T00:00:00Z',
  'aprobacion_ref','aprobacion:historia','motivo_publicacion','Prueba')::text;
 huella_cat:=encode(sha256(convert_to(doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.categorias.historicas',1,
  huella_cat,doc,'{}'::jsonb,vacia_sha,
  'aprobacion:hist:a','aprobacion:hist:b',
  'actor:bc9','decision:hist','recibo:bc9:hist',
  'motivos_bolsa:1:motivo_demo');
 FOR r IN SELECT * FROM (VALUES
  (id_a,1,'publicada','cat.long','Bolsa original',30),
  (id_a,2,'publicada','cat.good','Bolsa sucesora',10),
  (id_a,3,'retirada','cat.good','Bolsa retirada',5),
  (id_b,1,'publicada','cat.good','Bolsa actual',20),
  (id_b,2,'borrador','cat.good','Borrador posterior',2),
  (id_c,1,'publicada','cat.good','Bolsa retirada directa',25),
  (id_c,2,'retirada','cat.good','Retirada directa',4)
 ) AS v(id,secuencia,estado,categoria,titulo,dias) LOOP
  canonica:=jsonb_build_object('id',r.id,'secuencia',r.secuencia,
   'estado_gobierno',r.estado,
   'publicada_en',statement_timestamp()-interval '60 days',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',r.id||'#'||r.secuencia),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',r.id||'#'||r.secuencia),
   'contenido',jsonb_build_object('identificador_publico','bolsa-historica',
    'titulo',r.titulo,'resumen','Resumen publicado','tipo','bolsa',
    'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.historicas',
     'catalogo_version',1,'catalogo_huella_sha256',huella_cat),
    'categorias',jsonb_build_array(r.categoria),
    'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo:historico',
     'tipo','inscripcion','abre_en',statement_timestamp()-interval '90 days',
     'cierra_en',statement_timestamp()-make_interval(days=>r.dias))),
    'requisitos','[]'::jsonb));
  bytes:=convert_to(canonica::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria
   (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
  VALUES(r.id,r.secuencia,r.id||'#'||r.secuencia,r.estado,bytes,
   encode(sha256(bytes),'hex'),statement_timestamp());
 END LOOP;
 refs:=ARRAY[
  'cv1_'||encode(sha256(convert_to(id_a,'UTF8')),'hex')||'_v1',
  'cv1_'||encode(sha256(convert_to(id_a,'UTF8')),'hex')||'_v2',
  'cv1_'||encode(sha256(convert_to(id_b,'UTF8')),'hex')||'_v1',
  'cv1_'||encode(sha256(convert_to(id_c,'UTF8')),'hex')||'_v1'];
 resultado:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(refs,'es');
 IF jsonb_array_length(resultado)<>4
 OR resultado#>>'{0,estado_publicacion}'<>'sustituida'
 OR resultado#>>'{1,estado_publicacion}'<>'retirada'
 OR resultado#>>'{2,estado_publicacion}'<>'publicada'
 OR resultado#>>'{3,estado_publicacion}'<>'retirada'
 OR octet_length(resultado#>>'{0,categorias_resumen}')<>2048
 OR resultado#>>'{0,titulo}'<>'Bolsa original'
 OR resultado#>>'{1,categorias_resumen}'<>'Categoría nueva ES'
 OR resultado#>>'{0,numero_categorias}'<>'1'
 OR resultado#>>'{0,plazo_fin}' IS NULL
 OR resultado->0 ? 'convocatoria_id' OR resultado->0 ? 'unidad_ref'
 THEN RAISE EXCEPTION 'BC9: resumen histórico incorrecto'; END IF;
 resultado_en:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(refs,'en');
 IF octet_length(resultado_en#>>'{0,categorias_resumen}')<>2048
 OR resultado_en#>>'{1,categorias_resumen}'<>'New category EN'
 THEN RAISE EXCEPTION 'BC9: fallback EN histórico incorrecto'; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
   ARRAY[refs[1],refs[1]],'es');
  RAISE EXCEPTION 'BC9: refs duplicadas aceptadas';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
   ARRAY['cv1_'||repeat('f',64)||'_v1'],'es');
  RAISE EXCEPTION 'BC9: ref histórica ausente aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
END $test$;
ROLLBACK;
