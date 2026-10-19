\set ON_ERROR_STOP on
-- Vectores BC14 dentro de una transacción revertida, con datos ficticios.
-- Una versión publicada en VEC no se abre a la inscripción hasta que consta
-- la publicación oficial de sus bases con fecha anterior al plazo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 id text:='convocatoria:bc14:vector'; ref text; contenido jsonb; bytes bytea; huella text;
 hoy date:=(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date;
 base jsonb; r1 jsonb; r2 jsonb; r3 jsonb; pagina jsonb; fallo text;
BEGIN
 ref:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(id)||'_v1';
 contenido:=jsonb_build_object(
  'id',id,'secuencia',1,'estado_gobierno','publicada',
  'publicada_en',statement_timestamp()-interval '3 days',
  'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',id||'#1'),
  'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',id||'#1'),
  'contenido',jsonb_build_object(
   'identificador_publico',id,'titulo','Convocatoria BC14','resumen','Vector de publicación',
   'tipo','bolsa','categorias',jsonb_build_array('cat.alpha'),
   'plazos',jsonb_build_array(jsonb_build_object('tipo','inscripcion','referencia','plazo:bc14',
     'abre_en',statement_timestamp()-interval '1 day','cierra_en',statement_timestamp()+interval '90 days')),
   'requisitos','[]'::jsonb,
   'catalogo_categorias',jsonb_build_object('catalogo_id','catalogo:bc14:categorias',
     'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64))),
  'configuracion',jsonb_build_object(
   'catalogos',jsonb_build_object('id','politica:bc14','version',1,'huella_contenido_sha256',repeat('b',64)),
   'flujo_solicitud',jsonb_build_object('id','flujo:bc14','version',1,'huella_contenido_sha256',repeat('c',64)),
   'documentos',jsonb_build_array(jsonb_build_object('rol','bases','publicacion_ref','publicacion:bc14'))));
 bytes:=convert_to(contenido::text,'UTF8'); huella:=encode(sha256(bytes),'hex');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria(convocatoria_id,secuencia,referencia,estado,
  version_canonica,huella_version_sha256,instancia_flujo_ref,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,huella,NULL,statement_timestamp());

 -- Sin declaración: ni detalle, ni lista, ni comprobación.
 IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref) IS NOT NULL
 THEN RAISE EXCEPTION 'BC14: detalle abierto sin publicación oficial'; END IF;
 pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,100);
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(pagina->'items') x WHERE x->>'convocatoria_ref'=ref)
 THEN RAISE EXCEPTION 'BC14: lista abierta sin publicación oficial'; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref,'cat.alpha');
  RAISE EXCEPTION 'BC14: comprobación abierta sin publicación oficial';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN
  IF SQLERRM<>'BC14: bases sin publicación oficial declarada' THEN RAISE; END IF;
 END;

 base:=jsonb_build_object('convocatoria_id',id,'secuencia',1,'huella_version_sha256',huella,
  'medio','bop_granada','boletin_numero',201,'anuncio_ref','ENSAYO-BOP-2026-000201',
  'fecha_publicacion',to_char(hoy,'YYYY-MM-DD'),'direccion','https://bop.example.test/anuncio/201',
  'sustituye_ref',NULL,'declarada_por','actor:prueba:bc14','motivo_ref','motivos_bolsa:1:publicacion_bases',
  'clave_idempotencia','bc14-prueba-0001');
 -- Publicada el mismo día que abre el plazo: consta, pero no abre.
 r1:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(base);
 IF (r1->>'reutilizada')::boolean OR r1->>'recibo_ref'<>'recibo:'||(r1->>'publicacion_ref')
 OR vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref) IS NOT NULL
 THEN RAISE EXCEPTION 'BC14: fecha no anterior al plazo abre la inscripción'; END IF;
 -- Reintento idéntico: mismo recibo, sin otra fila.
 r2:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(base);
 IF NOT (r2->>'reutilizada')::boolean OR r2->>'publicacion_ref'<>r1->>'publicacion_ref'
 OR (SELECT count(*) FROM vec_bolsa_convocatorias.publicacion_oficial_bases WHERE convocatoria_id=id)<>1
 THEN RAISE EXCEPTION 'BC14: reintento no idempotente'; END IF;
 -- Misma clave con otro contenido, segunda declaración inicial: 23505.
 FOREACH fallo IN ARRAY ARRAY['misma_clave','segunda_inicial'] LOOP
  BEGIN
   PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(
    CASE fallo WHEN 'misma_clave' THEN base||'{"anuncio_ref":"OTRO-1"}'
    ELSE base||'{"clave_idempotencia":"bc14-prueba-0002"}' END);
   RAISE EXCEPTION 'BC14: % admitida',fallo;
  EXCEPTION WHEN unique_violation THEN NULL; END;
 END LOOP;
 -- Entradas inválidas: 22023 sin escribir.
 FOREACH fallo IN ARRAY ARRAY['futura','tablon_con_boletin','boletin_sin_numero','clave_extra',
   'medio','direccion','fecha_mal'] LOOP
  BEGIN
   PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(
    (base||'{"clave_idempotencia":"bc14-prueba-0003"}')||CASE fallo
    WHEN 'futura' THEN jsonb_build_object('fecha_publicacion',to_char(hoy+1,'YYYY-MM-DD'))
    WHEN 'tablon_con_boletin' THEN '{"medio":"tablon_edictos_electronico"}'::jsonb
    WHEN 'boletin_sin_numero' THEN '{"boletin_numero":null}'::jsonb
    WHEN 'clave_extra' THEN '{"otra":1}'::jsonb
    WHEN 'medio' THEN '{"medio":"periodico"}'::jsonb
    WHEN 'direccion' THEN '{"direccion":"http://bop.example.test/a"}'::jsonb
    ELSE '{"fecha_publicacion":"2026-02-30"}'::jsonb END);
   RAISE EXCEPTION 'BC14: entrada % admitida',fallo;
  EXCEPTION WHEN invalid_parameter_value OR check_violation THEN NULL; END;
 END LOOP;
 -- Corrección de la vigente con fecha anterior al plazo: ahora sí abre.
 r3:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(base||jsonb_build_object(
  'clave_idempotencia','bc14-prueba-0004','fecha_publicacion',to_char(hoy-3,'YYYY-MM-DD'),
  'sustituye_ref',r1->>'publicacion_ref','direccion',NULL,'anuncio_ref','ENSAYO-BOP-2026-000199','boletin_numero',199));
 IF (SELECT huella_anterior_sha256 FROM vec_bolsa_convocatorias.publicacion_oficial_bases
     WHERE publicacion_ref=r3->>'publicacion_ref')<>r1->>'huella_registro_sha256'
 OR (SELECT tipo FROM vec_bolsa_convocatorias.publicacion_oficial_bases_evento
     WHERE publicacion_ref=r3->>'publicacion_ref')<>'bolsa.convocatoria.publicacion_oficial.corregida'
 THEN RAISE EXCEPTION 'BC14: cadena o evento de corrección incorrectos'; END IF;
 IF vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref) IS NULL
 THEN RAISE EXCEPTION 'BC14: declaración vigente sin abrir el detalle'; END IF;
 pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,100);
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(pagina->'items') x WHERE x->>'convocatoria_ref'=ref)
 THEN RAISE EXCEPTION 'BC14: declaración vigente sin abrir la lista'; END IF;
 BEGIN
  PERFORM vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(ref,'cat.alpha');
 EXCEPTION WHEN OTHERS THEN
  -- Sin ámbito publicado el comprobador se detiene después, en CC11.
  IF SQLERRM LIKE 'BC14:%' OR SQLERRM NOT LIKE 'CC11:%' THEN
   RAISE EXCEPTION 'BC14: comprobación no supera la publicación (% %)',SQLSTATE,SQLERRM; END IF;
 END;
 -- Solo se corrige la vigente: la sustituida ya no admite otra sucesora.
 BEGIN
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(base||jsonb_build_object(
   'clave_idempotencia','bc14-prueba-0005','sustituye_ref',r1->>'publicacion_ref'));
  RAISE EXCEPTION 'BC14: corrección de una declaración sustituida admitida';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 -- Solo adición.
 BEGIN
  UPDATE vec_bolsa_convocatorias.publicacion_oficial_bases SET anuncio_ref='X' WHERE convocatoria_id=id;
  RAISE EXCEPTION 'BC14: actualización admitida';
 EXCEPTION WHEN OTHERS THEN IF SQLERRM LIKE 'BC14:%' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_bolsa_convocatorias.publicacion_oficial_bases WHERE convocatoria_id=id;
  RAISE EXCEPTION 'BC14: borrado admitido';
 EXCEPTION WHEN OTHERS THEN IF SQLERRM LIKE 'BC14:%' THEN RAISE; END IF; END;
 -- Una versión posterior publicada deja de admitir declaraciones de la anterior.
 bytes:=convert_to(jsonb_set(jsonb_set(contenido,'{secuencia}','2'),'{aprobacion_publicacion,convocatoria_ref}',
  to_jsonb(id||'#2'))::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria(convocatoria_id,secuencia,referencia,estado,
  version_canonica,huella_version_sha256,instancia_flujo_ref,registrada_en)
 VALUES(id,2,id||'#2','publicada',bytes,encode(sha256(bytes),'hex'),NULL,statement_timestamp());
 BEGIN
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(base||jsonb_build_object(
   'clave_idempotencia','bc14-prueba-0006','sustituye_ref',r3->>'publicacion_ref'));
  RAISE EXCEPTION 'BC14: declaración sobre versión no vigente admitida';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
END $prueba$;
-- Ninguna cuenta de aplicación declara ni lee la tabla.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $sin_permiso$
BEGIN
 BEGIN
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1('{}'::jsonb);
  RAISE EXCEPTION 'BC14: declaración ajena admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases LIMIT 1;
  RAISE EXCEPTION 'BC14: lectura ajena admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $sin_permiso$;
ROLLBACK;
