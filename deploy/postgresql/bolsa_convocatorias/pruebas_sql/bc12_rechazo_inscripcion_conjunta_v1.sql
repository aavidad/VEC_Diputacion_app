\set ON_ERROR_STOP on
-- BC12: el lector que usa B96 rechaza una publicación conjunta antes de
-- que la transacción de inscripción escriba la solicitud.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 modalidad text;
 identificador text;
 referencia text;
 contenido jsonb;
 bytes bytea;
 rechazado boolean;
 b96 integer;
BEGIN
 SELECT count(*) INTO b96 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='solicitar_inscripcion_v1'
  AND position('comprobar_publicacion_inscripcion_v1(v_convocatoria,v_categoria)'
   in pg_get_functiondef(p.oid))>0;
 IF b96<>1 THEN RAISE EXCEPTION 'BC12: el consumidor B96 no usa BC9'; END IF;

 FOREACH modalidad IN ARRAY ARRAY['conjunta','desconocida','nula'] LOOP
  identificador:='convocatoria:bc12:'||modalidad;
  referencia:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(identificador)||'_v1';
  contenido:=jsonb_build_object(
   'id',identificador,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en','2026-10-16T00:00:00Z',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',identificador||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',identificador||'#1'),
   'contenido',jsonb_build_object(
    'identificador_publico',identificador,'titulo','Convocatoria BC12',
    'resumen','Bases de ensayo','tipo','bolsa',
    'modalidad_inscripcion',CASE WHEN modalidad='nula' THEN 'null'::jsonb
      ELSE to_jsonb(modalidad) END,
    'categorias',jsonb_build_array('cat.alpha'),
    'plazos','[]'::jsonb,'requisitos','[]'::jsonb,
    'catalogo_categorias','{}'::jsonb),
   'configuracion',jsonb_build_object(
    'catalogos','{}'::jsonb,'flujo_solicitud','{}'::jsonb,
    'documentos','[]'::jsonb));
  bytes:=convert_to(contenido::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
   convocatoria_id,secuencia,referencia,estado,version_canonica,
   huella_version_sha256,instancia_flujo_ref,registrada_en)
  VALUES(identificador,1,identificador||'#1','publicada',bytes,
   encode(sha256(bytes),'hex'),NULL,statement_timestamp());

  rechazado:=false;
  BEGIN
   PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(
    referencia,'cat.alpha');
  EXCEPTION
   WHEN SQLSTATE 'B9608' THEN
    IF modalidad<>'conjunta' THEN RAISE; END IF;
    rechazado:=true;
   WHEN SQLSTATE 'B9601' THEN
    IF modalidad NOT IN('desconocida','nula')
     OR SQLERRM<>'BC12: modalidad de inscripcion invalida'
    THEN RAISE; END IF;
    rechazado:=true;
  END;
  IF NOT rechazado THEN
   RAISE EXCEPTION 'BC12: modalidad % admitida por el lector V1',modalidad;
  END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
