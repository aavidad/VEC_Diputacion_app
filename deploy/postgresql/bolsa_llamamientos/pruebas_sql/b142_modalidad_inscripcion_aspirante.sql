\set ON_ERROR_STOP on
-- B142: la proyección del aspirante copia la modalidad sellada en lista y ficha
-- y responde 55000 ante un valor incoherente. Todo ocurre en una transacción
-- revertida: los tres cotejos de catálogo se sustituyen por respuestas fijas
-- solo dentro de ella, para aislar la proyección de los datos de la copia.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';

DO $huella$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
   'vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean,text)')),'UTF8')),'hex')
   <>'4b0d847f7451f4097ffe8e82e07b4ac4144fb7c5d193b839f59ff87a6dcc63ce'
 THEN RAISE EXCEPTION 'B142: proyección instalada distinta de la declarada'; END IF;
END $huella$;

CREATE OR REPLACE FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(p_solicitudes jsonb,p_idioma text,p_canal text)
 RETURNS jsonb LANGUAGE sql AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',x->>'categoria_ref','categoria','Categoría B142',
  'politica_valida',true,'motivo_etiqueta_pendiente','Pendiente','motivo_etiqueta_cumple','Cumple',
  'impedimento_etiqueta_existente','Ya presentada','impedimento_etiqueta_politica','No disponible')),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) x $f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(p_solicitudes jsonb)
 RETURNS jsonb LANGUAGE sql AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('ambito_gestion_disponible',true)),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) x $f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(p_solicitudes jsonb,p_idioma text)
 RETURNS jsonb LANGUAGE sql AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('catalogo_completo',true,
  'numero_categorias',jsonb_array_length(x->'categorias_refs'))),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) x $f$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $prueba$
DECLARE
 base jsonb; item jsonb; salida jsonb; modalidad text; persona text:='per_b142aspirantesintetico0001';
 rechazado boolean;
BEGIN
 base:=jsonb_build_object('titulo','Convocatoria B142','resumen','Vector B142','numero_categorias',1,
  'catalogo_ref','catalogo:b142','catalogo_version',1,'catalogo_sha256',repeat('a',64),
  'politica_catalogo_ref','politica:b142','politica_catalogo_version',1,'politica_catalogo_sha256',repeat('b',64),
  'plazo_abre_en',statement_timestamp()-interval '1 day','plazo_cierra_en',statement_timestamp()+interval '60 days');
 FOREACH modalidad IN ARRAY ARRAY['separada','conjunta'] LOOP
  item:=base||jsonb_build_object('convocatoria_ref','cv1_'||repeat(CASE modalidad WHEN 'separada' THEN 'c' ELSE 'd' END,64)||'_v1',
   'modalidad_inscripcion',modalidad);
  -- Lista: categoría de comprobación y referencias de categorías.
  salida:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   jsonb_build_object('items',jsonb_build_array(item||jsonb_build_object('categoria_ref_comprobacion','cat.b142',
     'categorias_refs_comprobacion',jsonb_build_array('cat.b142'))),'total',1,'siguiente_cursor',NULL),
   persona,'es','certificado','alto',true,'externa_personal');
  IF salida#>>'{convocatorias,0,modalidad_inscripcion}' IS DISTINCT FROM modalidad
  THEN RAISE EXCEPTION 'B142: lista sin modalidad % (%)',modalidad,salida; END IF;
  -- Ficha: categorías y requisitos completos.
  salida:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   item||jsonb_build_object('categorias',jsonb_build_array(jsonb_build_object('categoria_ref','cat.b142')),
    'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada',
     'descripcion','Identidad certificada','obligatorio',true))),
   persona,'es','certificado','alto',false,'externa_personal');
  IF salida->>'modalidad_inscripcion' IS DISTINCT FROM modalidad
  THEN RAISE EXCEPTION 'B142: ficha sin modalidad % (%)',modalidad,salida; END IF;
 END LOOP;
 -- Ausente, desconocida o nula: material incoherente, 55000 sin proyectar.
 FOREACH modalidad IN ARRAY ARRAY['ausente','desconocida','nula'] LOOP
  item:=base||jsonb_build_object('convocatoria_ref','cv1_'||repeat('e',64)||'_v1',
   'categorias',jsonb_build_array(jsonb_build_object('categoria_ref','cat.b142')),'requisitos','[]'::jsonb);
  IF modalidad='desconocida' THEN item:=item||'{"modalidad_inscripcion":"mixta"}';
  ELSIF modalidad='nula' THEN item:=item||'{"modalidad_inscripcion":null}'; END IF;
  rechazado:=false;
  BEGIN
   PERFORM vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
    item,persona,'es','certificado','alto',false,'externa_personal');
  EXCEPTION WHEN SQLSTATE '55000' THEN
   IF SQLERRM<>'B142: modalidad de inscripcion invalida' THEN RAISE; END IF;
   rechazado:=true;
  END;
  IF NOT rechazado THEN RAISE EXCEPTION 'B142: modalidad % proyectada',modalidad; END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
