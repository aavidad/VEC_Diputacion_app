\set ON_ERROR_STOP on
-- Perfil SQL sintético, nunca en principal. V3/HTTP quedan fuera de alcance.
\if :{?B96_DISPOSABLE_CLONE}
\else
\echo 'B96: perfil restringido a clon desechable (-v B96_DISPOSABLE_CLONE=1)'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='60s';
SET LOCAL lock_timeout='2s';
INSERT INTO vec_catalogos_configurables.publicacion(
 catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
 preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
VALUES
 ('bolsa.categorias.inscripcion',1,encode(sha256(convert_to('{"id":"categorias"}','UTF8')),'hex'),
  '{"id":"categorias"}','{}',encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:cat:a','aprobacion:cat:b','actor:perf','decision:cat','recibo:cat'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  '{"id":"politica"}','{}',encode(sha256(convert_to('{}','UTF8')),'hex'),
  'aprobacion:pol:a','aprobacion:pol:b','actor:perf','decision:pol','recibo:pol');
INSERT INTO vec_catalogos_configurables.entrada_publicada(
 catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
VALUES
 ('bolsa.categorias.inscripcion',1,encode(sha256(convert_to('{"id":"categorias"}','UTF8')),'hex'),
  'cat.alpha','Alfa ES','{"etiquetas":{"es":"Alfa ES","en":"Alpha EN"}}'),
 ('bolsa.categorias.inscripcion',1,encode(sha256(convert_to('{"id":"categorias"}','UTF8')),'hex'),
  'cat.beta','Beta ES','{"etiquetas":{"es":"Beta ES","en":"Beta EN"}}'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  'presentacion.con.pendientes','Política','{"valor":true,"canal":"externa_personal"}'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  'requisito.pendiente','Pendiente ES','{"etiquetas":{"es":"Pendiente ES","en":"Pending EN"}}'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  'requisito.cumple','Cumple ES','{"etiquetas":{"es":"Cumple ES","en":"Meets EN"}}'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  'solicitud.existente','Ya solicitada ES','{"etiquetas":{"es":"Ya solicitada ES","en":"Already applied EN"}}'),
 ('bolsa.politica.inscripcion',1,encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
  'presentacion.no.disponible','No disponible ES','{"etiquetas":{"es":"No disponible ES","en":"Unavailable EN"}}');
CREATE TEMP TABLE b96_perf_filas AS
 SELECT i,
  CASE WHEN i<=1000 THEN 'per_'||repeat('a',22) ELSE 'per_'||repeat('b',22) END AS persona_ref,
  'cv1_'||encode(sha256(convert_to('proceso:perf:'||((i-1)%1000)::text,'UTF8')),'hex')||'_v1' AS convocatoria_ref,
  'solicitud_inscripcion_'||encode(sha256(convert_to('solicitud:perf:'||i::text,'UTF8')),'hex') AS solicitud_ref,
  CASE WHEN i%2=0 THEN 'cat.alpha' ELSE 'cat.beta' END AS categoria_ref,
  clock_timestamp()-interval '20 days'+i*interval '1 second' AS presentada_en,
  encode(sha256(convert_to('solicitud:perf:'||i::text,'UTF8')),'hex') AS huella
 FROM generate_series(1,2000) AS g(i);
INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion(
 solicitud_ref,persona_ref,contexto_actor_ref,contexto_actor_version,
 convocatoria_ref,convocatoria_id,secuencia,version_sha256,identificador_publico,
 categoria_ref,bases_ref,catalogo_ref,catalogo_version,catalogo_sha256,
 politica_catalogo_ref,politica_catalogo_version,politica_catalogo_sha256,
 formulario_ref,formulario_version,formulario_sha256,plazo_ref,plazo_abre_en,plazo_cierra_en,
 requisitos,requisitos_sha256,declaraciones,declaracion_ref,clave_sha256,material_sha256,canal,presentada_en)
SELECT f.solicitud_ref,f.persona_ref,'vca_'||repeat('v',22),1,
 f.convocatoria_ref,'proceso:perf:'||((f.i-1)%1000)::text,1,repeat('a',64),
 'publica:perf:'||((f.i-1)%1000)::text,f.categoria_ref,'bases.perf',
 'bolsa.categorias.inscripcion',1,
 encode(sha256(convert_to('{"id":"categorias"}','UTF8')),'hex'),
 'bolsa.politica.inscripcion',1,
 encode(sha256(convert_to('{"id":"politica"}','UTF8')),'hex'),
 'flujo.perf',1,repeat('b',64),'plazo.perf',clock_timestamp()-interval '21 days',
 CASE WHEN f.i%3=0 THEN clock_timestamp()-interval '15 days'
      WHEN f.i%3=1 THEN clock_timestamp()+interval '1 hour'
      ELSE clock_timestamp()+interval '30 days' END,
 $requisito$[{"referencia":"identidad_certificada","codigo":"identidad_certificada",
  "descripcion":"Identidad con certificado","obligatorio":true,"estado":"cumple",
  "motivo_codigo":""}]$requisito$::jsonb,repeat('c',64),'[]'::jsonb,
 'declaracion_inscripcion_'||f.huella,f.huella,f.huella,'externa_personal',f.presentada_en
FROM b96_perf_filas f;
INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
 solicitud_ref,version,estado,actor_ref,evaluacion,decision_ref,consumo_huella_sha256,auditoria_ref,aplicada_en)
SELECT f.solicitud_ref,1,'pendiente',f.persona_ref,
 $requisito$[{"referencia":"identidad_certificada","codigo":"identidad_certificada",
  "descripcion":"Identidad con certificado","obligatorio":true,"estado":"cumple",
  "motivo_codigo":""}]$requisito$::jsonb,
 'decision:perf:'||f.i::text,f.huella,'auditoria:perf:'||f.i::text,f.presentada_en
FROM b96_perf_filas f;
INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
 historia_ref,solicitud_ref,version,accion,actor_ref,estado,registrada_en)
SELECT 'historia_inscripcion_'||f.huella,f.solicitud_ref,1,'presentar',
 f.persona_ref,'pendiente',f.presentada_en FROM b96_perf_filas f;
INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
 evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
SELECT 'evento_inscripcion_'||encode(sha256(convert_to('evento:'||f.i::text,'UTF8')),'hex'),
 'historia_inscripcion_'||f.huella,f.solicitud_ref,1,
 'vec.bolsa.inscripcion.presentada.v1',jsonb_build_object('solicitud_ref',f.solicitud_ref),
 f.presentada_en FROM b96_perf_filas f;
INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
 recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
 historia_ref,evento_ref,auditoria_ref,emitida_en)
SELECT 'recibo_inscripcion_'||encode(sha256(convert_to('recibo:'||f.i::text,'UTF8')),'hex'),
 f.solicitud_ref,1,f.persona_ref,'externa_personal',f.huella,f.huella,
 'historia_inscripcion_'||f.huella,
 'evento_inscripcion_'||encode(sha256(convert_to('evento:'||f.i::text,'UTF8')),'hex'),
 'auditoria:perf:'||f.i::text,f.presentada_en FROM b96_perf_filas f;

DO $perfil$
DECLARE f jsonb; modo integer; repeticion integer; t timestamptz; muestras numeric[];
 p95 numeric; maximo numeric; total bigint; pagina jsonb; tam integer;
BEGIN
 FOR tam IN SELECT unnest(ARRAY[20,100]) LOOP
  FOR modo IN 1..2 LOOP
   muestras:=ARRAY[]::numeric[];
   f:=jsonb_build_object('estado','','convocatoria_ref','','limite',tam,'cursor','');
   FOR repeticion IN 1..40 LOOP
    t:=clock_timestamp();
    pagina:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
     modo=1,'per_'||repeat('a',22),f,'es');
    muestras:=array_append(muestras,extract(epoch FROM clock_timestamp()-t)*1000);
   END LOOP;
   SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY x.valor),max(x.valor)
    INTO p95,maximo FROM unnest(muestras) AS x(valor);
   total:=(pagina->>'total')::bigint;
   IF (modo=1 AND total<>1000) OR (modo=2 AND total<>2000)
    OR jsonb_array_length(pagina->'solicitudes')<>tam OR p95>=100
   THEN RAISE EXCEPTION 'B96 perf: propia=% limite=% total=% p95_ms=% max_ms=%',
    modo=1,tam,total,p95,maximo; END IF;
   RAISE NOTICE 'B96 perf SQL propia=% limite=% total=% p95_ms=% max_ms=%',
    modo=1,tam,total,round(p95,3),round(maximo,3);
  END LOOP;
 END LOOP;
END $perfil$;

EXPLAIN (ANALYZE,BUFFERS,TIMING OFF)
SELECT s.solicitud_ref,s.presentada_en
FROM vec_bolsa_llamamientos.solicitud_inscripcion s
WHERE s.persona_ref='per_'||repeat('a',22)
ORDER BY s.presentada_en DESC,s.solicitud_ref DESC LIMIT 100;
EXPLAIN (ANALYZE,BUFFERS,TIMING OFF)
SELECT s.solicitud_ref,s.presentada_en
FROM vec_bolsa_llamamientos.solicitud_inscripcion s
ORDER BY s.presentada_en DESC,s.solicitud_ref DESC LIMIT 100;
ROLLBACK;
