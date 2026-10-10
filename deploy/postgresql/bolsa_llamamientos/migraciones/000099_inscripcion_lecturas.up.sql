-- B99: título histórico en cada solicitud y recuento pendiente por convocatoria.
-- Preimagen: funciones literales de B96 (aún no instalada en la principal).
-- Depende de B96 y BC9; B99 conserva firmas, dueño, ACL y auditoría de B96.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000099',0));
DO $guard$
DECLARE actual text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(text,text,text,integer,text)') IS NULL
 THEN RAISE EXCEPTION 'B99 requiere B96' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid IN (
  to_regprocedure('vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)'),
  to_regprocedure('vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)'),
  to_regprocedure('vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(text,text,text,integer,text)'))
  AND pg_catalog.pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario')
 THEN RAISE EXCEPTION 'B99 requiere dueño propietario B96' USING ERRCODE='55000'; END IF;
 SELECT md5(prosrc) INTO actual FROM pg_catalog.pg_proc
 WHERE oid=to_regprocedure('vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)');
 IF actual IS DISTINCT FROM '67c65e39fdd4c326b05a9c36db6eaea6'
 THEN RAISE EXCEPTION 'B99 preimagen listado: esperado %, actual %',
  '67c65e39fdd4c326b05a9c36db6eaea6',actual USING ERRCODE='55000'; END IF;
 SELECT md5(prosrc) INTO actual FROM pg_catalog.pg_proc
 WHERE oid=to_regprocedure('vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)');
 IF actual IS DISTINCT FROM 'e4747679291d6124bc0bfd9ea9e2f564'
 THEN RAISE EXCEPTION 'B99 preimagen detalle: esperado %, actual %',
  'e4747679291d6124bc0bfd9ea9e2f564',actual USING ERRCODE='55000'; END IF;
 SELECT md5(prosrc) INTO actual FROM pg_catalog.pg_proc
 WHERE oid=to_regprocedure('vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(text,text,text,integer,text)');
 IF actual IS DISTINCT FROM 'c8bf62701090b6761acfae13b253c8ec'
 THEN RAISE EXCEPTION 'B99 preimagen selector: esperado %, actual %',
  'c8bf62701090b6761acfae13b253c8ec',actual USING ERRCODE='55000'; END IF;
END $guard$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
 p_propia boolean,p_persona_ref text,p_filtro jsonb,p_idioma text,p_canal text,
 p_unidad_ref text,p_ambito_ref text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 v_estado text; v_convocatoria text; v_cursor text; v_limite integer;
 v_cursor_en timestamptz(6); v_total bigint; v_raw jsonb; v_labels jsonb;
 v_peticiones jsonb; v_solicitudes jsonb; v_mas boolean; v_siguiente text;
 v_meta_convocatoria jsonb; v_convocatoria_titulo text;
 v_refs text[]; v_titulos jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en') OR p_canal NOT IN('externa_personal','interna_corporativa')
 OR (p_propia AND p_canal<>'externa_personal')
 OR (NOT p_propia AND p_canal<>'interna_corporativa')
 OR (p_propia AND (p_unidad_ref IS NOT NULL OR p_ambito_ref IS NOT NULL))
 OR (NOT p_propia AND (
  coalesce(p_unidad_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR coalesce(p_ambito_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'))
 OR jsonb_typeof(p_filtro) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(p_filtro) ORDER BY 1) IS DISTINCT FROM
    ARRAY['convocatoria_ref','cursor','estado','limite']
 OR jsonb_typeof(p_filtro->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_filtro->>'limite','') !~ '^[1-9][0-9]{0,2}$'
 OR (p_filtro->>'limite')::integer NOT BETWEEN 1 AND 100
 OR jsonb_typeof(p_filtro->'cursor') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_filtro->'estado') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_filtro->'convocatoria_ref') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'B96: filtro de solicitudes inválido' USING ERRCODE='B9605'; END IF;
 v_estado:=p_filtro->>'estado';
 v_convocatoria:=p_filtro->>'convocatoria_ref';
 v_cursor:=p_filtro->>'cursor';
 v_limite:=(p_filtro->>'limite')::integer;
 IF v_estado NOT IN('','pendiente','admitida_a_convocatoria','rechazada','incorporada')
 OR (NOT p_propia AND v_convocatoria='')
 OR (v_convocatoria<>'' AND (v_convocatoria !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
   OR octet_length(v_convocatoria)>200))
 OR (v_cursor<>'' AND v_cursor !~ '^solicitud_inscripcion_[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'B96: filtro de solicitudes inválido' USING ERRCODE='B9605'; END IF;
 IF NOT p_propia THEN
  v_meta_convocatoria:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
   ARRAY[v_convocatoria],p_idioma);
  IF jsonb_typeof(v_meta_convocatoria) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_meta_convocatoria)<>1
   OR v_meta_convocatoria#>>'{0,convocatoria_ref}' IS DISTINCT FROM v_convocatoria
   OR coalesce(v_meta_convocatoria#>>'{0,titulo}','')=''
   OR char_length(v_meta_convocatoria#>>'{0,titulo}')>180
  THEN RAISE EXCEPTION 'B96: título de convocatoria no disponible' USING ERRCODE='B9601'; END IF;
  v_convocatoria_titulo:=v_meta_convocatoria#>>'{0,titulo}';
 END IF;
 IF v_cursor<>'' THEN
  SELECT presentada_en INTO v_cursor_en
  FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_cursor AND (NOT p_propia OR persona_ref=p_persona_ref)
   AND (v_convocatoria='' OR convocatoria_ref=v_convocatoria)
   AND (p_propia OR (unidad_ref=p_unidad_ref AND ambito_ref=p_ambito_ref));
  IF NOT FOUND THEN RAISE EXCEPTION 'B96: cursor no disponible' USING ERRCODE='B9605'; END IF;
 END IF;
 WITH filtradas AS MATERIALIZED (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
    ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  WHERE (NOT p_propia OR s.persona_ref=p_persona_ref)
   AND (p_propia OR (s.unidad_ref=p_unidad_ref AND s.ambito_ref=p_ambito_ref))
   AND (v_estado='' OR v.estado=v_estado)
   AND (v_convocatoria='' OR s.convocatoria_ref=v_convocatoria)
 )
 SELECT (SELECT count(*) FROM filtradas),
  (SELECT coalesce(jsonb_agg(to_jsonb(q) ORDER BY q.presentada_en DESC,q.solicitud_ref DESC),'[]'::jsonb)
   FROM (SELECT * FROM filtradas
    WHERE v_cursor='' OR (presentada_en,solicitud_ref)<(v_cursor_en,v_cursor)
    ORDER BY presentada_en DESC,solicitud_ref DESC LIMIT v_limite+1) q)
 INTO v_total,v_raw;
 v_mas:=jsonb_array_length(v_raw)>v_limite;
 IF v_mas THEN
  SELECT coalesce(jsonb_agg(r.valor ORDER BY r.orden),'[]'::jsonb)
  INTO v_raw FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden)
  WHERE r.orden<=v_limite;
  v_siguiente:=v_raw->(jsonb_array_length(v_raw)-1)->>'solicitud_ref';
 END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',r.valor->>'catalogo_ref',
   'catalogo_version',(r.valor->>'catalogo_version')::integer,
   'catalogo_sha256',r.valor->>'catalogo_sha256',
   'categoria_ref',r.valor->>'categoria_ref',
   'politica_catalogo_ref',r.valor->>'politica_catalogo_ref',
   'politica_catalogo_version',(r.valor->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',r.valor->>'politica_catalogo_sha256') ORDER BY r.orden),'[]'::jsonb)
 INTO v_peticiones FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden);
 v_labels:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  v_peticiones,p_idioma,p_canal);
 IF jsonb_array_length(v_labels)<>jsonb_array_length(v_raw) THEN
  RAISE EXCEPTION 'B96: proyección de catálogo incompleta' USING ERRCODE='55000'; END IF;
 IF p_propia THEN
  SELECT coalesce(array_agg(DISTINCT r.valor->>'convocatoria_ref'),ARRAY[]::text[])
  INTO v_refs FROM jsonb_array_elements(v_raw) AS r(valor);
  IF cardinality(v_refs)>0 THEN
   v_meta_convocatoria:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
    v_refs,p_idioma);
   IF jsonb_typeof(v_meta_convocatoria) IS DISTINCT FROM 'array'
    OR jsonb_array_length(v_meta_convocatoria)<>cardinality(v_refs)
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(v_meta_convocatoria) WITH ORDINALITY AS x(valor,orden)
      WHERE x.valor->>'convocatoria_ref' IS DISTINCT FROM v_refs[x.orden]
       OR coalesce(x.valor->>'titulo','')='' OR char_length(x.valor->>'titulo')>180)
   THEN RAISE EXCEPTION 'B99: títulos de convocatoria incompletos' USING ERRCODE='55000'; END IF;
   SELECT jsonb_object_agg(x.valor->>'convocatoria_ref',x.valor->>'titulo') INTO v_titulos
   FROM jsonb_array_elements(v_meta_convocatoria) AS x(valor);
  END IF;
 ELSE
  v_titulos:=jsonb_build_object(v_convocatoria,v_convocatoria_titulo);
 END IF;
  SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
   r.valor,l.valor->>'categoria',NULL,l.valor->>'motivo_etiqueta_pendiente',
   l.valor->>'motivo_etiqueta_cumple') ||
   jsonb_build_object('convocatoria_titulo',v_titulos->>(r.valor->>'convocatoria_ref'))
   ORDER BY r.orden),'[]'::jsonb)
 INTO v_solicitudes
 FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden)
 JOIN jsonb_array_elements(v_labels) WITH ORDINALITY AS l(valor,orden) USING(orden);
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(v_solicitudes) AS x(valor)
  WHERE coalesce(x.valor->>'convocatoria_titulo','')='')
 THEN RAISE EXCEPTION 'B99: título de solicitud incompleto' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('solicitudes',v_solicitudes,'total',v_total,
  'cursor_siguiente',v_siguiente)||CASE WHEN p_propia THEN '{}'::jsonb
  ELSE jsonb_build_object('convocatoria_titulo',v_convocatoria_titulo) END;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_lector_inscripciones;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
 p_propia boolean,p_persona_ref text,p_solicitud_ref text,p_idioma text,p_canal text,
 p_unidad_ref text,p_ambito_ref text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE fila jsonb; etiquetas jsonb; motivo_etiqueta text; v_estado text;
 v_meta_convocatoria jsonb; v_titulo text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR p_canal IS NULL OR p_canal NOT IN('externa_personal','interna_corporativa')
 OR (p_propia AND p_canal<>'externa_personal')
 OR (NOT p_propia AND p_canal<>'interna_corporativa')
 OR (p_propia AND (p_unidad_ref IS NOT NULL OR p_ambito_ref IS NOT NULL))
 OR (NOT p_propia AND (
  coalesce(p_unidad_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR coalesce(p_ambito_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'))
 THEN RAISE EXCEPTION 'B96: selector de solicitud inválido' USING ERRCODE='B9605'; END IF;
 SELECT to_jsonb(q) INTO fila FROM (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
   WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
   ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  WHERE s.solicitud_ref=p_solicitud_ref
   AND (NOT p_propia OR s.persona_ref=p_persona_ref)
   AND (p_propia OR (s.unidad_ref=p_unidad_ref AND s.ambito_ref=p_ambito_ref))
 ) q;
 IF NOT FOUND THEN RETURN NULL; END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object(
   'catalogo_ref',fila->>'catalogo_ref',
   'catalogo_version',(fila->>'catalogo_version')::integer,
   'catalogo_sha256',fila->>'catalogo_sha256',
   'categoria_ref',fila->>'categoria_ref',
   'politica_catalogo_ref',fila->>'politica_catalogo_ref',
   'politica_catalogo_version',(fila->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',fila->>'politica_catalogo_sha256')),p_idioma,p_canal);
 IF jsonb_array_length(etiquetas)<>1 OR etiquetas#>>'{0,categoria}' IS NULL
 THEN RAISE EXCEPTION 'B96: etiqueta de categoría incompleta' USING ERRCODE='55000'; END IF;
 v_estado:=fila->>'estado';
 IF v_estado='rechazada' THEN
  motivo_etiqueta:=(vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   fila->>'motivo_catalogo_ref',(fila->>'motivo_catalogo_version')::integer,
   fila->>'motivo_catalogo_sha256',ARRAY[fila->>'motivo_ref'],p_idioma))#>>'{0,categoria}';
 END IF;
 v_meta_convocatoria:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
  ARRAY[fila->>'convocatoria_ref'],p_idioma);
 IF jsonb_typeof(v_meta_convocatoria) IS DISTINCT FROM 'array'
  OR jsonb_array_length(v_meta_convocatoria)<>1
  OR v_meta_convocatoria#>>'{0,convocatoria_ref}' IS DISTINCT FROM fila->>'convocatoria_ref'
  OR coalesce(v_meta_convocatoria#>>'{0,titulo}','')=''
  OR char_length(v_meta_convocatoria#>>'{0,titulo}')>180
 THEN RAISE EXCEPTION 'B99: título de solicitud incompleto' USING ERRCODE='55000'; END IF;
 v_titulo:=v_meta_convocatoria#>>'{0,titulo}';
 RETURN vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
  fila,etiquetas#>>'{0,categoria}',motivo_etiqueta,
  etiquetas#>>'{0,motivo_etiqueta_pendiente}',
  etiquetas#>>'{0,motivo_etiqueta_cumple}') || jsonb_build_object('convocatoria_titulo',v_titulo);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_lector_inscripciones;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(
 p_unidad_ref text,p_ambito_ref text,p_cursor text,p_limite integer,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE v_total bigint;v_refs text[];v_mas boolean;v_cursor text;
 v_metadatos jsonb;v_cartas jsonb;v_resultado jsonb;v_salida jsonb;
 v_caben integer; v_pendientes jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR coalesce(p_unidad_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR coalesce(p_ambito_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR p_cursor IS NULL OR (p_cursor<>'' AND p_cursor !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$')
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 THEN RAISE EXCEPTION 'B96: selector RRHH inválido' USING ERRCODE='B9605'; END IF;
 SELECT count(DISTINCT convocatoria_ref) INTO v_total
 FROM vec_bolsa_llamamientos.solicitud_inscripcion
 WHERE unidad_ref=p_unidad_ref AND ambito_ref=p_ambito_ref;
 SELECT array_agg(q.convocatoria_ref ORDER BY q.convocatoria_ref COLLATE "C") INTO v_refs
 FROM (SELECT DISTINCT convocatoria_ref COLLATE "C" AS convocatoria_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE unidad_ref=p_unidad_ref AND ambito_ref=p_ambito_ref
   AND (p_cursor='' OR convocatoria_ref COLLATE "C">p_cursor COLLATE "C")
  ORDER BY convocatoria_ref LIMIT p_limite+1) AS q;
 v_refs:=coalesce(v_refs,ARRAY[]::text[]);
 v_mas:=cardinality(v_refs)>p_limite;
 IF v_mas THEN v_refs:=v_refs[1:p_limite];v_cursor:=v_refs[p_limite]; END IF;
 IF cardinality(v_refs)=0 THEN
  v_metadatos:='[]'::jsonb;
 ELSE
  v_metadatos:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
   v_refs,p_idioma);
 END IF;
 IF jsonb_typeof(v_metadatos) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_metadatos)<>cardinality(v_refs)
 THEN RAISE EXCEPTION 'B96: metadatos de convocatoria incompletos' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(v_metadatos) WITH ORDINALITY AS x(valor,orden)
   WHERE x.valor->>'convocatoria_ref' IS DISTINCT FROM v_refs[x.orden]
    OR coalesce(x.valor->>'titulo','')=''
    OR coalesce(x.valor->>'categorias_resumen','')=''
    OR coalesce(x.valor->>'estado_publicacion','') NOT IN('publicada','sustituida','retirada')
    OR x.valor->>'plazo_fin' IS NULL)
 THEN RAISE EXCEPTION 'B96: resumen histórico divergente' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_object_agg(q.convocatoria_ref,q.pendientes),'{}'::jsonb)
 INTO v_pendientes FROM (
  SELECT s.convocatoria_ref,count(*) AS pendientes
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT v.estado,v.version FROM vec_bolsa_llamamientos.solicitud_inscripcion_version v
    WHERE v.solicitud_ref=s.solicitud_ref ORDER BY v.version DESC LIMIT 1) ultima ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo recibo
    ON recibo.solicitud_ref=s.solicitud_ref AND recibo.version=ultima.version
  WHERE s.unidad_ref=p_unidad_ref AND s.ambito_ref=p_ambito_ref
   AND s.convocatoria_ref=ANY(v_refs) AND ultima.estado='pendiente'
  GROUP BY s.convocatoria_ref
 ) q;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'convocatoria_ref',x.valor->>'convocatoria_ref',
  'titulo',x.valor->>'titulo',
  'categorias_resumen',x.valor->>'categorias_resumen',
  'plazo_fin',x.valor->'plazo_fin',
  'estado_publicacion',x.valor->>'estado_publicacion',
  'pendientes',coalesce((v_pendientes->>(x.valor->>'convocatoria_ref'))::bigint,0)) ORDER BY x.orden),'[]'::jsonb)
 INTO v_cartas FROM jsonb_array_elements(v_metadatos) WITH ORDINALITY AS x(valor,orden);
 -- 240 KiB para la proyección, con 512 B reservados para total/cursor/claves.
 -- Una ventana sobre tamaños UTF-8 elige el mayor prefijo sin recomponer el
 -- JSON en cada fila; las restantes aparecen desde el nuevo cursor.
 SELECT count(*) INTO v_caben FROM (
  SELECT sum(octet_length(x.valor::text)+1) OVER (ORDER BY x.orden) AS acumulado
  FROM jsonb_array_elements(v_cartas) WITH ORDINALITY AS x(valor,orden)
 ) q WHERE q.acumulado<=245248;
 IF cardinality(v_refs)>0 AND v_caben=0 THEN
  RAISE EXCEPTION 'B96: resumen histórico excede página' USING ERRCODE='B9601'; END IF;
 SELECT coalesce(jsonb_agg(x.valor ORDER BY x.orden),'[]'::jsonb)
 INTO v_resultado FROM jsonb_array_elements(v_cartas) WITH ORDINALITY AS x(valor,orden)
 WHERE x.orden<=v_caben;
 v_mas:=v_mas OR v_caben<cardinality(v_refs);
 v_cursor:=CASE WHEN v_mas AND v_caben>0 THEN v_refs[v_caben] ELSE NULL END;
 v_salida:=jsonb_build_object('convocatorias',v_resultado,'total',v_total,
  'cursor_siguiente',v_cursor);
 IF octet_length(v_salida::text)>245760 THEN
  RAISE EXCEPTION 'B96: presupuesto de página incompatible' USING ERRCODE='55000'; END IF;
 RETURN v_salida;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(text,text,text,integer,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,
 vec_bolsa_llamamientos_lector_inscripciones,vec_bolsa_llamamientos_lector_inscripciones_rrhh;
COMMIT;
