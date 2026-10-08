\set ON_ERROR_STOP on
-- Lista compacta B96: 100 convocatorias con 128 categorías, sólo clon desechable.
\if :{?B96_DISPOSABLE_CLONE}
\else
\echo 'B96: perfil restringido a clon desechable (-v B96_DISPOSABLE_CLONE=1)'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='60s';
SET LOCAL lock_timeout='2s';
DO $preparar$
DECLARE cat_doc text:='{"id":"bolsa.categorias.inscripcion","version":1}';
 pol_doc text:='{"id":"bolsa.politica.inscripcion","version":1}';
 cat_sha text;pol_sha text;id text;canon jsonb;bytes bytea;categorias jsonb;
 i integer;
BEGIN
 cat_sha:=encode(sha256(convert_to(cat_doc,'UTF8')),'hex');
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 INSERT INTO vec_catalogos_configurables.publicacion(
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES('bolsa.categorias.inscripcion',1,cat_sha,cat_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:cat:a','aprobacion:cat:b',
  'actor:b96','decision:cat','recibo:cat'),
 ('bolsa.politica.inscripcion',1,pol_sha,pol_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:pol:a','aprobacion:pol:b',
  'actor:b96','decision:pol','recibo:pol');
 INSERT INTO vec_catalogos_configurables.entrada_publicada(
  catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 SELECT 'bolsa.categorias.inscripcion',1,cat_sha,'cat.'||lpad(n::text,3,'0'),
  'Categoría '||lpad(n::text,3,'0'),jsonb_build_object('etiquetas',
    jsonb_build_object('es','Categoría '||lpad(n::text,3,'0')))
 FROM generate_series(1,128) AS g(n);
 INSERT INTO vec_catalogos_configurables.entrada_publicada(
  catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
 ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.con.pendientes','Política',
  '{"valor":true,"canal":"externa_personal"}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'requisito.pendiente','Pendiente ES','{}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'requisito.cumple','Cumple ES','{}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'solicitud.existente','Ya solicitada ES','{}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.no.disponible','No disponible ES','{}');
 SELECT jsonb_agg('cat.'||lpad(n::text,3,'0') ORDER BY n) INTO categorias
 FROM generate_series(1,128) AS g(n);
 FOR i IN 1..100 LOOP
  id:='proceso:bolsa:perf128:'||lpad(i::text,3,'0');
  canon:=jsonb_build_object('id',id,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en',clock_timestamp()-interval '1 day',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',id||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',id||'#1'),
   'contenido',jsonb_build_object('identificador_publico','perf128-'||i,
    'tipo','bolsa','titulo','Convocatoria sintética '||i,
    'resumen','Requisito de identidad certificado',
    'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.inscripcion',
      'catalogo_version',1,'catalogo_huella_sha256',cat_sha),
    'categorias',categorias,
    'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo.perf128',
      'tipo','inscripcion','abre_en',clock_timestamp()-interval '1 hour',
      'cierra_en',clock_timestamp()+interval '1 hour')),
    'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada',
      'orden',1,'descripcion','Identidad con certificado','obligatorio',true))),
   'configuracion',jsonb_build_object('catalogos',jsonb_build_object(
      'id','bolsa.politica.inscripcion','version',1,'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo.perf128','version',1,
      'huella_contenido_sha256',repeat('a',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases',
      'publicacion_ref','bases.perf128'))));
  bytes:=convert_to(canon::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
   convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
  VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),clock_timestamp());
 END LOOP;
END $preparar$;
DO $medir$
DECLARE limite integer;repeticion integer;antes timestamptz;tiempos numeric[];
 p95 numeric;respuesta jsonb;raw jsonb;bytes integer;
BEGIN
 FOREACH limite IN ARRAY ARRAY[20,100] LOOP
  tiempos:=ARRAY[]::numeric[];
  FOR repeticion IN 1..20 LOOP
   antes:=clock_timestamp();
   raw:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(NULL,limite);
   respuesta:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
    raw,'per_'||repeat('a',22),'es','certificado','alto',true);
   tiempos:=array_append(tiempos,extract(epoch FROM clock_timestamp()-antes)*1000);
  END LOOP;
  bytes:=octet_length(respuesta::text);
  SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY x.valor)
   INTO p95 FROM unnest(tiempos) AS x(valor);
  IF respuesta->>'total'<>'100'
   OR jsonb_array_length(respuesta->'convocatorias')<>limite
   OR respuesta#>>'{convocatorias,0,numero_categorias}'<>'128'
   OR (respuesta#>'{convocatorias,0}') ? 'categorias'
   OR (respuesta#>'{convocatorias,0}') ? 'requisitos'
   OR bytes>262144 OR p95>=300
  THEN RAISE EXCEPTION 'B96 perf128: limite=% total=% bytes=% p95_ms=%',
    limite,respuesta->>'total',bytes,p95; END IF;
  RAISE NOTICE 'B96 perf128 SQL limite=% bytes=% p95_ms=%',limite,bytes,round(p95,3);
 END LOOP;
END $medir$;
-- Publicación nueva e inmutable: la categoría 128 existe, pero su etiqueta ES
-- (201 bytes, admitida por CC1) no cabe en el contrato visible de inscripción.
-- Nunca se altera una entrada ya publicada para construir el adversarial.
DO $catalogo_incompatible$
DECLARE id text:='proceso:bolsa:perf128:incompatible';
 catalogo text:='bolsa.categorias.inscripcion.incompatible';
 documento text:='{"id":"bolsa.categorias.inscripcion.incompatible","version":1}';
 huella text; canon jsonb; bytes bytea;
BEGIN
 huella:=encode(sha256(convert_to(documento,'UTF8')),'hex');
 INSERT INTO vec_catalogos_configurables.publicacion(
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES(catalogo,1,huella,documento,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:cat:incompatible:a',
  'aprobacion:cat:incompatible:b','actor:b96','decision:cat:incompatible','recibo:cat:incompatible');
 INSERT INTO vec_catalogos_configurables.entrada_publicada(
  catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 SELECT catalogo,1,huella,categoria_id,
  CASE WHEN categoria_id='cat.128' THEN repeat('x',201) ELSE etiqueta END,
  CASE WHEN categoria_id='cat.128' THEN '{}'::jsonb ELSE definicion END
 FROM vec_catalogos_configurables.entrada_publicada
 WHERE catalogo_id='bolsa.categorias.inscripcion' AND version=1;
 SELECT convert_from(version_canonica,'UTF8')::jsonb INTO STRICT canon
 FROM vec_bolsa_convocatorias.version_convocatoria
 WHERE convocatoria_id='proceso:bolsa:perf128:001' AND secuencia=1;
 canon:=jsonb_set(canon,'{id}',to_jsonb(id));
 canon:=jsonb_set(canon,'{contenido,identificador_publico}',to_jsonb('perf128-incompatible'::text));
 canon:=jsonb_set(canon,'{contenido,catalogo_categorias,catalogo_id}',to_jsonb(catalogo));
 canon:=jsonb_set(canon,'{contenido,catalogo_categorias,catalogo_huella_sha256}',to_jsonb(huella));
 canon:=jsonb_set(canon,'{aprobacion_publicacion,convocatoria_ref}',to_jsonb(id||'#1'));
 canon:=jsonb_set(canon,'{comprobacion_dependencias,convocatoria_ref}',to_jsonb(id||'#1'));
 bytes:=convert_to(canon::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
  convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES(id,1,id||'#1','publicada',bytes,encode(sha256(bytes),'hex'),clock_timestamp());
END $catalogo_incompatible$;
DO $categoria_incompatible$
DECLARE raw jsonb;pagina jsonb;detalle jsonb;ref text;cursor text;tarjeta jsonb;
 intento integer;
BEGIN
 ref:='cv1_'||encode(sha256(convert_to(
  'proceso:bolsa:perf128:incompatible','UTF8')),'hex')||'_v1';
 FOR intento IN 1..2 LOOP
  raw:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(cursor,100);
  pagina:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   raw,'per_'||repeat('a',22),'es','certificado','alto',true);
  IF pagina->>'total'<>'101' THEN
   RAISE EXCEPTION 'B96: total cambió al hallar categoría incompatible %',pagina->>'total';
  END IF;
  SELECT valor INTO tarjeta FROM jsonb_array_elements(pagina->'convocatorias') AS x(valor)
   WHERE valor->>'convocatoria_ref'=ref;
  EXIT WHEN FOUND;
  cursor:=raw->>'siguiente_cursor';
  EXIT WHEN cursor IS NULL;
 END LOOP;
 IF tarjeta IS NULL OR tarjeta->>'puede_iniciar' IS DISTINCT FROM 'false'
 OR tarjeta->>'impedimento_etiqueta' IS DISTINCT FROM 'No disponible ES'
 THEN RAISE EXCEPTION 'B96: categoría incompatible ofertada o fila omitida %',tarjeta; END IF;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(ref);
 BEGIN
  PERFORM vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   detalle,'per_'||repeat('a',22),'es','certificado','alto',false);
  RAISE EXCEPTION 'B96: detalle aceptó etiqueta incompatible';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL;
 END;
END $categoria_incompatible$;
ROLLBACK;
