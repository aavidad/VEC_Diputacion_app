\set ON_ERROR_STOP on
-- Vectores de SQL dentro de una transaccion revertida, no bases aprobadas.
-- B96 debe rechazar conjunta antes de insertar una solicitud individual.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 modalidad text; efectiva text; identificador text; referencia text;
 contenido jsonb; bytes bytea; detalle jsonb; pagina jsonb;
 rechazado boolean; b96 integer;
BEGIN
 SELECT count(*) INTO b96 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='solicitar_inscripcion_v1'
  AND position('comprobar_publicacion_inscripcion_v1(v_convocatoria,v_categoria)'
   in pg_get_functiondef(p.oid))>0;
 IF b96<>1 THEN RAISE EXCEPTION 'BC12: el consumidor B96 no usa BC9'; END IF;

 FOREACH modalidad IN ARRAY ARRAY['ausente','separada','conjunta','desconocida','nula'] LOOP
  identificador:='convocatoria:bc12:'||modalidad;
  referencia:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(identificador)||'_v1';
  contenido:=jsonb_build_object(
   'id',identificador,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en',statement_timestamp()-interval '1 day',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',identificador||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',identificador||'#1'),
   'contenido',jsonb_build_object(
    'identificador_publico',identificador,'titulo','Convocatoria BC12',
    'resumen','Vector de inscripcion','tipo','bolsa',
    'categorias',jsonb_build_array('cat.alpha'),
    'plazos',jsonb_build_array(jsonb_build_object('tipo','inscripcion',
      'referencia','plazo:bc12','abre_en',statement_timestamp()-interval '1 day',
      'cierra_en',statement_timestamp()+interval '90 days')),
    'requisitos','[]'::jsonb,
    'catalogo_categorias',jsonb_build_object('catalogo_id','catalogo:bc12:categorias',
      'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64))),
   'configuracion',jsonb_build_object(
    'catalogos',jsonb_build_object('id','politica:bc12','version',1,'huella_contenido_sha256',repeat('b',64)),
    'flujo_solicitud',jsonb_build_object('id','flujo:bc12','version',1,'huella_contenido_sha256',repeat('c',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases','publicacion_ref','publicacion:bc12'))));
  IF modalidad<>'ausente' THEN
   contenido:=jsonb_set(contenido,'{contenido,modalidad_inscripcion}',
    CASE WHEN modalidad='nula' THEN 'null'::jsonb ELSE to_jsonb(modalidad) END);
  END IF;
  bytes:=convert_to(contenido::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
   convocatoria_id,secuencia,referencia,estado,version_canonica,
   huella_version_sha256,instancia_flujo_ref,registrada_en)
  VALUES(identificador,1,identificador||'#1','publicada',bytes,
   encode(sha256(bytes),'hex'),NULL,statement_timestamp());
  -- BC14: sin publicación oficial declarada ninguna versión se abre.
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb_build_object(
   'convocatoria_id',identificador,'secuencia',1,'huella_version_sha256',encode(sha256(bytes),'hex'),
   'medio','bop_granada','boletin_numero',12,'anuncio_ref','ENSAYO-BC12-'||modalidad,
   'fecha_publicacion',to_char((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date-2,'YYYY-MM-DD'),
   'direccion',NULL,'sustituye_ref',NULL,'declarada_por','actor:prueba:bc12',
   'motivo_ref','motivos_bolsa:1:publicacion_bases','clave_idempotencia','bc12-'||modalidad));

  IF modalidad IN('ausente','separada','conjunta') THEN
   efectiva:=CASE WHEN modalidad='ausente' THEN 'separada' ELSE modalidad END;
   detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(referencia);
   IF detalle IS NULL OR detalle->>'modalidad_inscripcion' IS DISTINCT FROM efectiva
   THEN RAISE EXCEPTION 'BC12: modalidad ausente o divergente en detalle'; END IF;
   pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,100);
   IF (SELECT count(*) FROM jsonb_array_elements(pagina->'items') AS x(dato)
     WHERE dato->>'convocatoria_ref'=referencia AND dato->>'modalidad_inscripcion'=efectiva)<>1
   THEN RAISE EXCEPTION 'BC12: modalidad ausente o divergente en lista'; END IF;
  ELSE
   -- Una version con modalidad invalida queda fuera sin tumbar la lista.
   IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(referencia) IS NOT NULL
   THEN RAISE EXCEPTION 'BC12: modalidad invalida proyectada'; END IF;
   pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,100);
   IF EXISTS(SELECT 1 FROM jsonb_array_elements(pagina->'items') AS x(dato)
     WHERE dato->>'convocatoria_ref'=referencia)
   OR (SELECT count(*) FROM jsonb_array_elements(pagina->'items') AS x(dato)
     WHERE dato->>'convocatoria_ref' IN(
      'cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1('convocatoria:bc12:ausente')||'_v1',
      'cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1('convocatoria:bc12:separada')||'_v1',
      'cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1('convocatoria:bc12:conjunta')||'_v1'))<>3
   THEN RAISE EXCEPTION 'BC12: lista alterada por modalidad invalida'; END IF;
  END IF;

  IF modalidad IN('ausente','separada') THEN
   -- El comprobador debe superar la modalidad: sin ambito publicado se detiene
   -- en CC11; nunca en B9608 ni en un rechazo de BC12.
   BEGIN
    PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(referencia,'cat.alpha');
   EXCEPTION WHEN OTHERS THEN
    IF SQLSTATE='B9608' OR SQLERRM LIKE 'BC12:%'
     OR SQLSTATE<>'B9601' OR SQLERRM NOT LIKE 'CC11:%' THEN
     RAISE EXCEPTION 'BC12: % no supera la modalidad (% %)',modalidad,SQLSTATE,SQLERRM;
    END IF;
   END;
  END IF;
  IF modalidad NOT IN('ausente','separada') THEN
   rechazado:=false;
   BEGIN
    PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(referencia,'cat.alpha');
   EXCEPTION
    WHEN SQLSTATE 'B9608' THEN
     IF modalidad<>'conjunta' THEN RAISE; END IF;
     rechazado:=true;
    WHEN SQLSTATE '55000' THEN
     IF modalidad NOT IN('desconocida','nula')
      OR SQLERRM<>'BC12: modalidad de inscripcion invalida' THEN RAISE; END IF;
     rechazado:=true;
   END;
   IF NOT rechazado THEN RAISE EXCEPTION 'BC12: modalidad % admitida por lector V1',modalidad; END IF;
  END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
