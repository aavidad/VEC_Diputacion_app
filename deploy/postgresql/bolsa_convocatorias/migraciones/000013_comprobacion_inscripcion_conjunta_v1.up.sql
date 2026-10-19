\set ON_ERROR_STOP on
-- BC13 (F1-41): comprobación de publicación para la solicitud conjunta.
-- Función nueva, copia de la comprobación instalada por BC12 con tres cambios:
-- exige la modalidad conjunta sellada (ausente o separada: B9608), recibe la
-- lista ordenada de categorías elegidas y la devuelve como seleccionadas.
-- BC9/BC12 quedan intactas. Sin tablas, datos ni DOWN; no se reaplica.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000013',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.huella_id_inscripcion_v1(text)') IS NULL
 -- La modalidad sellada la introdujo BC12: su comprobación debe ser la instalada.
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)')),'UTF8')),'hex')
    <>'510d5fb7fa0daa905cf8565f2e53de8d00b2ed5623bf0d5a1a028082cbc71547'
 THEN RAISE EXCEPTION 'BC13: falta BC12 o la preimagen es incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(p_convocatoria_ref text, p_categorias_refs jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE
 v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 c jsonb; plazo jsonb; candidato jsonb; catalogo jsonb; politica jsonb;
 formulario jsonb; bases jsonb; v_ambito jsonb;
 requisitos jsonb; pendientes jsonb; categorias jsonb; instante timestamptz(6);
 abre timestamptz; cierra timestamptz; v_secuencia bigint;
 partes text[]; v_id_sha256 text; publicada timestamptz; activos integer:=0;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 9 AND 200
 OR jsonb_typeof(p_categorias_refs) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_categorias_refs) NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_categorias_refs) AS x(valor)
   WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'string' OR x.valor#>>'{}' !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 -- Orden canónico C y sin repetidos: el mismo que sella el material del aspirante.
 OR p_categorias_refs IS DISTINCT FROM (SELECT jsonb_agg(DISTINCT (x.valor#>>'{}') COLLATE "C" ORDER BY (x.valor#>>'{}') COLLATE "C")
   FROM jsonb_array_elements(p_categorias_refs) AS x(valor))
 THEN RAISE EXCEPTION 'BC13: selector de publicación inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN RAISE EXCEPTION 'BC13: referencia no canónica' USING ERRCODE='22023'; END IF;
 v_id_sha256:=partes[1];
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'BC13: referencia no canónica' USING ERRCODE='22023'; END IF;

 -- Bloqueo de fila y predicado SERIALIZABLE: la versión debe seguir siendo la
 -- publicada más reciente mientras se consume la autorización y se inserta.
 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"=v_id_sha256 COLLATE "C"
    AND secuencia=v_secuencia FOR SHARE;
 EXCEPTION WHEN no_data_found THEN
  RAISE EXCEPTION 'BC13: publicación no vigente' USING ERRCODE='B9601';
 WHEN too_many_rows THEN
  RAISE EXCEPTION 'BC13: referencia de publicación ambigua' USING ERRCODE='55000'; END;
 IF v.estado<>'publicada'
 OR v.referencia IS DISTINCT FROM v.convocatoria_id||'#'||v_secuencia::text
 OR encode(sha256(v.version_canonica),'hex') IS DISTINCT FROM v.huella_version_sha256
 OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
   WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v_secuencia
    AND posterior.estado IN('publicada','sustituida','retirada'))
 THEN RAISE EXCEPTION 'BC13: publicación no vigente' USING ERRCODE='B9601'; END IF;
 BEGIN c:=convert_from(v.version_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC13: versión gobernada ilegible' USING ERRCODE='55000';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR c->>'id' IS DISTINCT FROM v.convocatoria_id
 OR c->>'secuencia' IS DISTINCT FROM v_secuencia::text
 OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(c#>'{contenido,plazos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,requisitos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,catalogo_categorias}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,catalogos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,flujo_solicitud}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
 OR c#>>'{contenido,identificador_publico}' IS NULL
 OR c#>>'{contenido,titulo}' IS NULL OR c#>>'{contenido,resumen}' IS NULL
 OR c#>>'{contenido,tipo}' IS NULL
 OR jsonb_typeof(c->'aprobacion_publicacion') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c->'comprobacion_dependencias') IS DISTINCT FROM 'object'
 OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR c->>'publicada_en' IS NULL
 THEN RAISE EXCEPTION 'BC13: material publicado incompleto' USING ERRCODE='55000'; END IF;
 -- BC13 (F1-41): solo la modalidad conjunta sellada admite una solicitud para
 -- varias categorías. Ausente (histórica) o separada: la solicitud es por
 -- categoría y se responde B9608 antes de consumir nada más.
 IF c#>'{contenido,modalidad_inscripcion}' IS NOT NULL
  AND (jsonb_typeof(c#>'{contenido,modalidad_inscripcion}') IS DISTINCT FROM 'string'
   OR c#>>'{contenido,modalidad_inscripcion}' NOT IN('separada','conjunta'))
 THEN RAISE EXCEPTION 'BC13: modalidad de inscripcion invalida' USING ERRCODE='55000'; END IF;
 IF c#>>'{contenido,modalidad_inscripcion}' IS DISTINCT FROM 'conjunta'
 THEN RAISE EXCEPTION 'BC13: la convocatoria no admite solicitud conjunta' USING ERRCODE='B9608'; END IF;
 IF jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 THEN RAISE EXCEPTION 'BC13: categorías publicadas incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT ((c#>'{contenido,categorias}') @> p_categorias_refs)
 THEN RAISE EXCEPTION 'BC13: categoría ajena a la publicación' USING ERRCODE='B9605'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC13: fecha de publicación inválida' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC13: fecha de publicación inválida' USING ERRCODE='55000'; END IF;

 instante:=date_trunc('microseconds',clock_timestamp());
 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
   WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC13: plazo de inscripción inválido' USING ERRCODE='55000';
  END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
     OR candidato->>'referencia' IS NULL THEN
   RAISE EXCEPTION 'BC13: plazo de inscripción incompleto' USING ERRCODE='55000';
  END IF;
  IF abre<=instante AND instante<cierra THEN
   activos:=activos+1; plazo:=candidato;
  END IF;
 END LOOP;
 IF activos<>1 THEN RAISE EXCEPTION 'BC13: plazo de inscripción cerrado o ambiguo' USING ERRCODE='B9602'; END IF;
 abre:=(plazo->>'abre_en')::timestamptz;
 cierra:=(plazo->>'cierra_en')::timestamptz;

 catalogo:=c#>'{contenido,catalogo_categorias}';
 politica:=c#>'{configuracion,catalogos}';
 formulario:=c#>'{configuracion,flujo_solicitud}';
 requisitos:=c#>'{contenido,requisitos}';
 SELECT valor INTO bases FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
  WHERE valor->>'rol'='bases';
 IF catalogo->>'catalogo_id' IS NULL OR coalesce(catalogo->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(catalogo->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 OR politica->>'id' IS NULL OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(politica->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR formulario->>'id' IS NULL OR coalesce(formulario->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(formulario->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
    WHERE valor->>'rol'='bases')<>1
 OR bases->>'publicacion_ref' IS NULL
 OR jsonb_array_length(requisitos)>256
 THEN RAISE EXCEPTION 'BC13: referencias de bases incompletas' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
    'referencia',r.valor->>'referencia','codigo',r.valor->>'referencia',
    'descripcion',r.valor->>'descripcion',
    'obligatorio',(r.valor->>'obligatorio')::boolean,
    'estado','pendiente','motivo_codigo','requisito.pendiente')
    ORDER BY (r.valor->>'orden')::integer,r.valor->>'referencia'),'[]'::jsonb)
 INTO pendientes FROM jsonb_array_elements(requisitos) AS r(valor);
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(requisitos) AS r(valor)
   WHERE coalesce(r.valor->>'referencia','')=''
      OR jsonb_typeof(r.valor->'obligatorio') IS DISTINCT FROM 'boolean'
      OR coalesce(r.valor->>'orden','') !~ '^[1-9][0-9]{0,5}$')
 THEN RAISE EXCEPTION 'BC13: requisitos incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',categoria.valor)
   ORDER BY categoria.valor),'[]'::jsonb)
 INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS categoria(valor);
 v_ambito:=vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
  politica->>'id',(politica->>'version')::integer,
  politica->>'huella_contenido_sha256',p_convocatoria_ref);

 RETURN jsonb_build_object(
  'convocatoria_ref',p_convocatoria_ref,
  'convocatoria_id',v.convocatoria_id,'secuencia',v_secuencia,
  'categorias_seleccionadas',p_categorias_refs,'categorias',categorias,
  'modalidad_inscripcion','conjunta',
  'numero_categorias',jsonb_array_length(categorias),
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'version_sha256',v.huella_version_sha256,
  'bases_ref',bases->>'publicacion_ref',
  'catalogo_ref',catalogo->>'catalogo_id',
  'catalogo_version',(catalogo->>'catalogo_version')::integer,
  'catalogo_sha256',catalogo->>'catalogo_huella_sha256',
  'politica_catalogo_ref',politica->>'id',
  'politica_catalogo_version',(politica->>'version')::integer,
  'politica_catalogo_sha256',politica->>'huella_contenido_sha256',
  'formulario_ref',formulario->>'id','formulario_version',(formulario->>'version')::integer,
  'formulario_sha256',formulario->>'huella_contenido_sha256',
  'plazo_ref',plazo->>'referencia','plazo_abre_en',abre,'plazo_cierra_en',cierra,
  'requisitos',pendientes,
  'requisitos_sha256',encode(sha256(convert_to(requisitos::text,'UTF8')),'hex'),
  'unidad_ref',v_ambito->>'unidad_ref','ambito_ref',v_ambito->>'ambito_ref',
  'fuente_ref',v_ambito->>'fuente_ref',
  'fuente_version',v_ambito->'fuente_version',
  'fuente_sha256',v_ambito->>'fuente_sha256',
  'comprobada_en',instante);
END $function$
;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb) FROM PUBLIC;
-- Solo la solicitud conjunta de Bolsa (B146) la consume, como BC9 con B96.
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)
 TO vec_bolsa_llamamientos_propietario;

DO $post$
DECLARE f regprocedure; p record;
BEGIN
 f:=to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)');
 IF f IS NULL THEN RAISE EXCEPTION 'BC13: funcion ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>'v' OR p.proparallel::text<>'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario=X/vec_bolsa_convocatorias_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'c293822620dd95dda2957dacaa5bf4644cfad6f08a4adb02b72a42c4ec065626'
 THEN RAISE EXCEPTION 'BC13: definicion o metadatos divergentes' USING ERRCODE='55000'; END IF;
 IF has_function_privilege('public',f,'EXECUTE')
 THEN RAISE EXCEPTION 'BC13: ejecucion publica' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
