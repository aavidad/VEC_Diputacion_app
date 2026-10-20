\set ON_ERROR_STOP on
-- B142 (F1-40): la lista y la ficha del aspirante llevan la modalidad sellada
-- de la convocatoria. Reconstruye solo proyectar_abiertas_inscripcion_interna_v1
-- desde su cuerpo instalado (B96) y añade modalidad_inscripcion a cada carta;
-- un valor distinto de separada o conjunta responde 55000. Requiere BC12, que
-- proyecta el campo desde la version sellada. Mismo OID, propietario, ACL y
-- configuracion. Sin tablas, permisos ni datos nuevos. Sin DOWN ni reaplicacion.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000142',0));
DO $causal$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamp with time zone)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamp with time zone)')),'UTF8')),'hex')
    <>'d47623ea04fef1e6eed09aebb017771ddea8bd685613fb3e63a939455ee40ef8'
 THEN RAISE EXCEPTION 'B142: falta BC12 o la preimagen causal es incompatible' USING ERRCODE='55000'; END IF;
END $causal$;
DO $pre$
DECLARE f regprocedure; p record;
BEGIN
 f:=to_regprocedure('vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean,text)');
 IF f IS NULL THEN RAISE EXCEPTION 'B142: funcion causal ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>'v' OR p.proparallel::text<>'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'1cc75016f6015f8b79a049ee0ac3fdb311c3b436f55866647d1f0bb3d1906790'
 THEN RAISE EXCEPTION 'B142: definicion o metadatos divergentes' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec.b142.oid',f::oid::text,true);
END $pre$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(p_fuente jsonb, p_persona_ref text, p_idioma text, p_metodo text, p_garantia text, p_lista boolean, p_canal text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 v_items jsonb; v_item jsonb; v_categoria jsonb; v_requisito jsonb;
 v_selectores jsonb:='[]'::jsonb; v_etiquetas jsonb;
 v_categorias_pedidos jsonb; v_categorias_validas jsonb;
 v_gestion_pedidos jsonb; v_gestion_validos jsonb;
 v_propias jsonb; v_categorias jsonb; v_requisitos jsonb; v_bolsas jsonb:='[]'::jsonb;
 v_etiqueta jsonb; v_requisitos_resumen text; v_num_categorias integer; v_carta jsonb;
 v_estado text; v_puede boolean; v_catalogo_completo boolean; v_impedimento text;
 v_pos integer:=0; v_gestion_pos integer:=0;
 v_conv text; v_propia jsonb; v_salida jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en') OR p_lista IS NULL
 OR p_canal IS DISTINCT FROM 'externa_personal'
 OR jsonb_typeof(p_fuente) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B96: fuente abierta inválida' USING ERRCODE='55000'; END IF;
 v_items:=CASE WHEN p_lista THEN p_fuente->'items' ELSE jsonb_build_array(p_fuente) END;
 IF jsonb_typeof(v_items) IS DISTINCT FROM 'array' OR jsonb_array_length(v_items)>100
 THEN RAISE EXCEPTION 'B96: página abierta fuera de límite' USING ERRCODE='54000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(v_items) AS x(valor)
  WHERE (p_lista AND (
   coalesce(x.valor->>'numero_categorias','') !~ '^[1-9][0-9]{0,2}$'
   OR (x.valor->>'numero_categorias')::integer NOT BETWEEN 1 AND 128
   OR coalesce(x.valor->>'categoria_ref_comprobacion','')=''
   OR jsonb_typeof(x.valor->'categorias_refs_comprobacion') IS DISTINCT FROM 'array'
   OR jsonb_array_length(x.valor->'categorias_refs_comprobacion')
      IS DISTINCT FROM (x.valor->>'numero_categorias')::integer
   OR x.valor ? 'categorias'))
  OR (NOT p_lista AND (
   jsonb_typeof(x.valor->'categorias') IS DISTINCT FROM 'array'
   OR jsonb_array_length(x.valor->'categorias') NOT BETWEEN 1 AND 128)))
 THEN RAISE EXCEPTION 'B96: categorías publicadas incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',item.valor->>'catalogo_ref',
   'catalogo_version',(item.valor->>'catalogo_version')::integer,
   'catalogo_sha256',item.valor->>'catalogo_sha256',
   'categoria_ref',categoria.valor->>'categoria_ref',
   'politica_catalogo_ref',item.valor->>'politica_catalogo_ref',
   'politica_catalogo_version',(item.valor->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',item.valor->>'politica_catalogo_sha256')
   ORDER BY item.orden,categoria.orden),'[]'::jsonb)
 INTO v_selectores
 FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden)
 CROSS JOIN LATERAL jsonb_array_elements(
   CASE WHEN p_lista THEN jsonb_build_array(jsonb_build_object(
     'categoria_ref',item.valor->>'categoria_ref_comprobacion'))
   ELSE item.valor->'categorias' END
 ) WITH ORDINALITY AS categoria(valor,orden);
 v_etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  v_selectores,p_idioma,p_canal);
 IF jsonb_typeof(v_etiquetas) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_etiquetas)<>jsonb_array_length(v_selectores)
 THEN RAISE EXCEPTION 'B96: lote de etiquetas incompleto' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'politica_catalogo_ref',item.valor->>'politica_catalogo_ref',
  'politica_catalogo_version',(item.valor->>'politica_catalogo_version')::integer,
  'politica_catalogo_sha256',item.valor->>'politica_catalogo_sha256',
  'convocatoria_ref',item.valor->>'convocatoria_ref') ORDER BY item.orden),'[]'::jsonb)
 INTO v_gestion_pedidos
 FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden);
 v_gestion_validos:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  v_gestion_pedidos);
 IF jsonb_typeof(v_gestion_validos) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_gestion_validos)<>jsonb_array_length(v_items)
 THEN RAISE EXCEPTION 'B96: cotejo de gestión incompleto' USING ERRCODE='55000'; END IF;
 IF p_lista THEN
  SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',item.valor->>'catalogo_ref',
   'catalogo_version',(item.valor->>'catalogo_version')::integer,
   'catalogo_sha256',item.valor->>'catalogo_sha256',
   'categorias_refs',item.valor->'categorias_refs_comprobacion')
   ORDER BY item.orden),'[]'::jsonb)
  INTO v_categorias_pedidos
  FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden);
  v_categorias_validas:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
   v_categorias_pedidos,p_idioma);
  IF jsonb_typeof(v_categorias_validas) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_categorias_validas)<>jsonb_array_length(v_items)
  THEN RAISE EXCEPTION 'B96: cotejo de categorías incompleto' USING ERRCODE='55000'; END IF;
 END IF;
 SELECT coalesce(jsonb_object_agg(q.convocatoria_ref,
   jsonb_build_object('solicitud_ref',q.solicitud_ref,'estado',q.estado)),'{}'::jsonb)
 INTO v_propias FROM (
  SELECT s.convocatoria_ref,s.solicitud_ref,v.estado
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT estado FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY version DESC LIMIT 1) v ON true
  WHERE s.persona_ref=p_persona_ref
   AND s.convocatoria_ref IN (SELECT x.valor->>'convocatoria_ref' FROM jsonb_array_elements(v_items) AS x(valor))
 ) q;
 FOR v_item IN SELECT valor FROM jsonb_array_elements(v_items) AS x(valor) LOOP
  v_conv:=v_item->>'convocatoria_ref';
 v_categorias:='[]'::jsonb;
 v_requisitos:='[]'::jsonb;
  v_requisitos_resumen:=v_item->>'resumen';
  v_puede:=true;
  IF p_lista THEN
   v_num_categorias:=(v_item->>'numero_categorias')::integer;
   IF jsonb_typeof(v_categorias_validas->v_pos->'catalogo_completo') IS DISTINCT FROM 'boolean'
    OR v_categorias_validas->v_pos->>'numero_categorias'
       IS DISTINCT FROM v_num_categorias::text
   THEN RAISE EXCEPTION 'B96: cotejo de categorías divergente' USING ERRCODE='55000'; END IF;
   v_catalogo_completo:=(v_categorias_validas->v_pos->>'catalogo_completo')::boolean;
   v_etiqueta:=v_etiquetas->v_pos;
   v_pos:=v_pos+1;
   IF v_etiqueta->>'categoria_ref' IS DISTINCT FROM v_item->>'categoria_ref_comprobacion'
    OR coalesce(v_etiqueta->>'categoria','')=''
    OR jsonb_typeof(v_etiqueta->'politica_valida') IS DISTINCT FROM 'boolean'
    OR coalesce(v_etiqueta->>'motivo_etiqueta_pendiente','')=''
    OR coalesce(v_etiqueta->>'motivo_etiqueta_cumple','')=''
   THEN RAISE EXCEPTION 'B96: categoría del lote divergente' USING ERRCODE='55000'; END IF;
   v_puede:=(v_etiqueta->>'politica_valida')::boolean AND v_catalogo_completo;
  ELSE
   v_num_categorias:=jsonb_array_length(v_item->'categorias');
   IF v_item->>'numero_categorias' IS DISTINCT FROM v_num_categorias::text THEN
    RAISE EXCEPTION 'B96: total de categorías divergente' USING ERRCODE='55000';
   END IF;
   FOR v_categoria IN SELECT valor FROM jsonb_array_elements(v_item->'categorias') AS x(valor) LOOP
    v_etiqueta:=v_etiquetas->v_pos;
    v_pos:=v_pos+1;
    IF v_etiqueta->>'categoria_ref' IS DISTINCT FROM v_categoria->>'categoria_ref'
     OR coalesce(v_etiqueta->>'categoria','')=''
     OR jsonb_typeof(v_etiqueta->'politica_valida') IS DISTINCT FROM 'boolean'
     OR coalesce(v_etiqueta->>'motivo_etiqueta_pendiente','')=''
     OR coalesce(v_etiqueta->>'motivo_etiqueta_cumple','')=''
    THEN RAISE EXCEPTION 'B96: categoría del lote divergente' USING ERRCODE='55000'; END IF;
    v_categorias:=v_categorias||jsonb_build_array(jsonb_build_object(
     'categoria_ref',v_categoria->>'categoria_ref','categoria',v_etiqueta->>'categoria'));
    v_puede:=v_puede AND (v_etiqueta->>'politica_valida')::boolean;
   END LOOP;
   -- El catálogo de política es común a la versión, y el detalle sí presenta
   -- cada requisito con su estado y motivo desde la fuente publicada.
   v_etiqueta:=v_etiquetas->(v_pos-1);
   v_requisitos_resumen:=NULL;
   FOR v_requisito IN SELECT valor FROM jsonb_array_elements(v_item->'requisitos') AS x(valor) LOOP
   IF coalesce(v_requisito->>'descripcion','')='' THEN
    RAISE EXCEPTION 'B96: descripción de requisito ausente' USING ERRCODE='55000';
   END IF;
   v_requisitos_resumen:=concat_ws('; ',v_requisitos_resumen,v_requisito->>'descripcion');
   v_estado:='pendiente';
   IF v_requisito->>'referencia'='identidad_certificada'
    AND p_metodo IN('certificado','dnie') AND p_garantia='alto' THEN
    v_estado:='cumple';
   END IF;
   v_requisitos:=v_requisitos||jsonb_build_array(jsonb_build_object(
    'codigo',coalesce(v_requisito->>'codigo',v_requisito->>'referencia'),
    'descripcion',v_requisito->>'descripcion',
    'obligatorio',(v_requisito->>'obligatorio')::boolean,
    'estado',v_estado,
    'motivo_codigo',CASE WHEN v_estado='pendiente' THEN 'requisito.pendiente' ELSE '' END,
    'motivo_etiqueta',CASE WHEN v_estado='pendiente'
      THEN v_etiqueta->>'motivo_etiqueta_pendiente' ELSE v_etiqueta->>'motivo_etiqueta_cumple' END,
    'hito_cumplimiento',NULL,'hito_etiqueta',NULL));
  END LOOP;
  END IF;
  IF jsonb_typeof(v_gestion_validos->v_gestion_pos->'ambito_gestion_disponible')
     IS DISTINCT FROM 'boolean'
  THEN RAISE EXCEPTION 'B96: gestión de convocatoria divergente' USING ERRCODE='55000'; END IF;
  v_puede:=v_puede AND (v_gestion_validos->v_gestion_pos->>'ambito_gestion_disponible')::boolean;
  v_gestion_pos:=v_gestion_pos+1;
  v_propia:=v_propias->v_conv;
  v_puede:=v_puede AND v_propia IS NULL;
  v_impedimento:=CASE WHEN v_propia IS NOT NULL
    THEN v_etiqueta->>'impedimento_etiqueta_existente'
    WHEN NOT v_puede THEN v_etiqueta->>'impedimento_etiqueta_politica'
    ELSE NULL END;
  IF NOT v_puede AND coalesce(v_impedimento,'')='' THEN
   RAISE EXCEPTION 'B96: impedimento sin catálogo' USING ERRCODE='B9601';
  END IF;
  -- B142: la modalidad sellada viaja a lista y ficha; otro valor es incoherente.
  IF jsonb_typeof(v_item->'modalidad_inscripcion') IS DISTINCT FROM 'string'
   OR v_item->>'modalidad_inscripcion' NOT IN('separada','conjunta')
  THEN RAISE EXCEPTION 'B142: modalidad de inscripcion invalida' USING ERRCODE='55000'; END IF;
  v_carta:=jsonb_build_object(
   'convocatoria_ref',v_conv,'titulo',v_item->>'titulo',
   'modalidad_inscripcion',v_item->>'modalidad_inscripcion',
   'numero_categorias',v_num_categorias,
   'plazo_inicio',(v_item->>'plazo_abre_en')::timestamptz,
   'plazo_fin',(v_item->>'plazo_cierra_en')::timestamptz,
   'catalogo_version',(v_item->>'catalogo_version')::bigint,
   'requisitos_resumen',coalesce(v_requisitos_resumen,v_item->>'resumen'),
   'puede_iniciar',v_puede,
   'impedimento_etiqueta',v_impedimento,
   'estado_solicitud_propia',v_propia->>'estado',
   'solicitud_ref',v_propia->>'solicitud_ref');
  IF NOT p_lista THEN
   v_carta:=v_carta||jsonb_build_object('categorias',v_categorias,'requisitos',v_requisitos);
  END IF;
  v_bolsas:=v_bolsas||jsonb_build_array(v_carta);
 END LOOP;
 IF p_lista THEN
  v_salida:=jsonb_build_object('convocatorias',v_bolsas,'total',(p_fuente->>'total')::bigint,
   'cursor_siguiente',p_fuente->>'siguiente_cursor');
 ELSE v_salida:=v_bolsas->0; END IF;
 RETURN v_salida;
END $function$
;

DO $post$
DECLARE f regprocedure; p record;
BEGIN
 f:=to_regprocedure('vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean,text)');
 IF f IS NULL THEN RAISE EXCEPTION 'B142: funcion causal ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>'v' OR p.proparallel::text<>'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'4b0d847f7451f4097ffe8e82e07b4ac4144fb7c5d193b839f59ff87a6dcc63ce'
 THEN RAISE EXCEPTION 'B142: definicion o metadatos divergentes' USING ERRCODE='55000'; END IF;
 IF f::oid::text IS DISTINCT FROM current_setting('vec.b142.oid',true) THEN RAISE EXCEPTION 'B142: OID divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
