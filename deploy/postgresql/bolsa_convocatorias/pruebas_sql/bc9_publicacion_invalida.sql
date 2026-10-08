\set ON_ERROR_STOP on
-- Las cuatro lectoras rechazan la misma publicación mal acreditada. Cada
-- escenario se revierte en una subtransacción y la prueba entera en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='20s';
SET LOCAL lock_timeout='2s';
DO $test$
DECLARE numero integer; v_id text; v_ref text; material jsonb; bytes bytea; huella text;
BEGIN
 FOR numero IN 1..6 LOOP
  BEGIN
   v_id:='proceso:bolsa:publicacion-invalida-'||numero;
   v_ref:='cv1_'||encode(sha256(convert_to(v_id,'UTF8')),'hex')||'_v1';
   material:=jsonb_build_object('id',v_id,'secuencia',1,'estado_gobierno','publicada',
    'publicada_en',statement_timestamp(),
    'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',v_id||'#1'),
    'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',v_id||'#1'),
    'contenido',jsonb_build_object('identificador_publico','bolsa-invalida-'||numero,
     'titulo','Bolsa sintética','resumen','Resumen sintético','tipo','bolsa',
     'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.inscripcion',
      'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64)),
     'categorias',jsonb_build_array('cat.alpha','cat.beta'),
     'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo:inscripcion',
      'tipo','inscripcion','abre_en',statement_timestamp()-interval '1 day',
      'cierra_en',statement_timestamp()+interval '1 day')),
     'requisitos','[]'::jsonb),
    'configuracion',jsonb_build_object(
     'catalogos',jsonb_build_object('id','bolsa.politica.inscripcion','version',1,
      'huella_contenido_sha256',repeat('b',64)),
     'flujo_solicitud',jsonb_build_object('id','flujo:inscripcion','version',1,
      'huella_contenido_sha256',repeat('c',64)),
     'documentos',jsonb_build_array(jsonb_build_object('rol','bases',
      'publicacion_ref','bases:invalida'))));
   CASE numero
    WHEN 1 THEN material:=jsonb_set(material,'{aprobacion_publicacion}','null'::jsonb);
    WHEN 2 THEN material:=jsonb_set(material,'{aprobacion_publicacion,convocatoria_ref}',
      '"proceso:bolsa:ajena#1"'::jsonb);
    WHEN 3 THEN material:=jsonb_set(material,'{comprobacion_dependencias}','null'::jsonb);
    WHEN 4 THEN material:=jsonb_set(material,'{comprobacion_dependencias,convocatoria_ref}',
      '"proceso:bolsa:ajena#1"'::jsonb);
    WHEN 5 THEN material:=jsonb_set(material,'{publicada_en}','"infinity"'::jsonb);
    WHEN 6 THEN material:=jsonb_set(material,'{contenido,categorias}',
      jsonb_build_array('cat.alpha','cat.alpha'));
   END CASE;
   bytes:=convert_to(material::text,'UTF8');
   huella:=encode(sha256(bytes),'hex');
   INSERT INTO vec_bolsa_convocatorias.version_convocatoria
    (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
   VALUES(v_id,1,v_id||'#1','publicada',bytes,huella,statement_timestamp());
   BEGIN
    PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(v_ref,'cat.alpha');
    RAISE EXCEPTION 'BC9: POST aceptó publicación inválida %',numero;
   EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
   BEGIN
    PERFORM vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,10);
    RAISE EXCEPTION 'BC9: lista ofreció publicación inválida %',numero;
   EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
   BEGIN
    PERFORM vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(v_ref);
    RAISE EXCEPTION 'BC9: detalle aceptó publicación inválida %',numero;
   EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
   BEGIN
    PERFORM vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
     v_ref,'cat.alpha',huella);
    RAISE EXCEPTION 'BC9: historia aceptó publicación inválida %',numero;
   EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
   RAISE EXCEPTION 'escenario revertido' USING ERRCODE='ZV001';
  EXCEPTION WHEN SQLSTATE 'ZV001' THEN NULL; END;
 END LOOP;
END $test$;
ROLLBACK;
