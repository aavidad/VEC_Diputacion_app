\set ON_ERROR_STOP on
-- Fixture sintético de BC1/BC8 sobre PostgreSQL 18 desechable.
-- La publicación original está cerrada y tiene versión posterior; no se
-- reabre el plazo ni se modifica la fila histórica. Siempre ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE v_id text:='proceso:bolsa:historica-2026';
 v_ref text; material jsonb; bytes bytea; huella text; leida jsonb;
 v_secuencia integer;
 f regprocedure:='vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f,'EXECUTE')
 OR has_table_privilege('vec_bolsa_llamamientos_propietario',
  'vec_bolsa_convocatorias.version_convocatoria','SELECT')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
  aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee=0)
 THEN RAISE EXCEPTION 'BC9 histórico: ACL abierta'; END IF;
 v_ref:='cv1_'||encode(sha256(convert_to(v_id,'UTF8')),'hex')||'_v1';
 material:=jsonb_build_object('id',v_id,'secuencia',1,'estado_gobierno','publicada',
  'publicada_en',statement_timestamp()-interval '60 days',
  'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',v_id||'#1'),
  'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',v_id||'#1'),
  'contenido',jsonb_build_object('identificador_publico','bolsa-historica-2026',
   'titulo','Bolsa histórica sintética','resumen','Resumen histórico','tipo','bolsa',
   'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.inscripcion',
    'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64)),
   'categorias',jsonb_build_array('cat.alpha','cat.beta'),
   'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo:historico',
    'tipo','inscripcion','abre_en',statement_timestamp()-interval '59 days',
    'cierra_en',statement_timestamp()-interval '30 days')),
   'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada',
    'orden',1,'descripcion','Identidad certificada','obligatorio',true))),
  'configuracion',jsonb_build_object(
   'catalogos',jsonb_build_object('id','bolsa.politica.inscripcion','version',1,
    'huella_contenido_sha256',repeat('b',64)),
   'flujo_solicitud',jsonb_build_object('id','flujo:inscripcion','version',1,
    'huella_contenido_sha256',repeat('c',64)),
   'documentos',jsonb_build_array(jsonb_build_object('rol','bases',
    'publicacion_ref','bases:historica'))));
 bytes:=convert_to(material::text,'UTF8');
 huella:=encode(sha256(bytes),'hex');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(v_id,1,v_id||'#1','publicada',bytes,huella,statement_timestamp()-interval '60 days');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(v_id,2,v_id||'#2','borrador',bytes,huella,statement_timestamp()-interval '20 days');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria
  (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 SELECT v_id,e.secuencia,v_id||'#'||e.secuencia,e.estado,bytes,huella,statement_timestamp()-interval '10 days'
 FROM (VALUES (3,'sustituida'),(4,'retirada')) AS e(secuencia,estado);

 leida:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(v_ref,'cat.beta',huella);
 IF leida->>'convocatoria_ref'<>v_ref OR leida->>'categoria_ref'<>'cat.beta'
 OR leida->>'version_sha256'<>huella OR leida->>'bases_ref'<>'bases:historica'
 OR leida->>'catalogo_ref'<>'bolsa.categorias.inscripcion'
 OR leida->>'politica_catalogo_ref'<>'bolsa.politica.inscripcion'
 OR leida->>'formulario_ref'<>'flujo:inscripcion'
 OR jsonb_array_length(leida->'plazos_inscripcion')<>1
 OR leida#>>'{plazos_inscripcion,0,plazo_ref}'<>'plazo:historico'
 OR leida->>'requisitos_sha256' IS NULL
 THEN RAISE EXCEPTION 'BC9 histórico: historia original incorrecta: %',leida; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   v_ref,'cat.beta',repeat('0',64));
  RAISE EXCEPTION 'BC9 histórico: huella equivocada aceptada';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   v_ref,'cat.ajena',huella);
  RAISE EXCEPTION 'BC9 histórico: categoría ajena aceptada';
 EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   left(v_ref,length(v_ref)-1)||'2','cat.beta',huella);
  RAISE EXCEPTION 'BC9 histórico: borrador aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 FOR v_secuencia IN 3..4 LOOP
  BEGIN
   PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
    left(v_ref,length(v_ref)-1)||v_secuencia::text,'cat.beta',huella);
   RAISE EXCEPTION 'BC9 histórico: estado no publicado aceptado: %',v_secuencia;
  EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 END LOOP;
END $test$;
ROLLBACK;
