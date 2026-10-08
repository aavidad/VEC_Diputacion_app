\set ON_ERROR_STOP on
-- BC9: seam de la publicación gobernada de Bolsa para una solicitud propia.
-- La preparación BC8 no constituye publicación. El texto de requisitos se
-- conserva como pendiente: aquí no se deduce cumplimiento de una descripción.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000009',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.preparacion_bases_version_v3') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria_token_inscripcion_uq') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.huella_id_inscripcion_v1(text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamp with time zone)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(text,integer)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 THEN RAISE EXCEPTION 'BC9: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La referencia pública usa sólo la huella del identificador opaco. La
-- unicidad impide que una colisión de huellas en una misma secuencia resuelva
-- otra convocatoria y hace fallar cerrada la instalación o escritura futura.
-- PostgreSQL clasifica convert_to como STABLE por la codificación de base.
-- La preimagen exige UTF8 fijo; este envoltorio es inmutable dentro de ella.
CREATE FUNCTION vec_bolsa_convocatorias.huella_id_inscripcion_v1(p_id text)
RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path=pg_catalog,pg_temp
AS $f$ SELECT encode(sha256(convert_to(p_id,'UTF8')),'hex') $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.huella_id_inscripcion_v1(text)
 FROM PUBLIC,vec_bolsa_llamamientos_propietario;
CREATE UNIQUE INDEX version_convocatoria_token_inscripcion_uq
 ON vec_bolsa_convocatorias.version_convocatoria
 ((vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"),secuencia);

CREATE FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(
 p_convocatoria_ref text,p_categoria_ref text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE
 v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 c jsonb; plazo jsonb; candidato jsonb; catalogo jsonb; politica jsonb;
 formulario jsonb; bases jsonb;
 requisitos jsonb; pendientes jsonb; categorias jsonb; instante timestamptz(6);
 abre timestamptz; cierra timestamptz; v_secuencia bigint;
 partes text[]; v_id_sha256 text; publicada timestamptz; activos integer:=0;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 9 AND 200
 OR p_categoria_ref IS NULL OR p_categoria_ref !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 THEN RAISE EXCEPTION 'BC9: selector de publicación inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;
 v_id_sha256:=partes[1];
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;

 -- Bloqueo de fila y predicado SERIALIZABLE: la versión debe seguir siendo la
 -- publicada más reciente mientras se consume la autorización y se inserta.
 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"=v_id_sha256 COLLATE "C"
    AND secuencia=v_secuencia FOR SHARE;
 EXCEPTION WHEN no_data_found THEN
  RAISE EXCEPTION 'BC9: publicación no vigente' USING ERRCODE='B9601';
 WHEN too_many_rows THEN
  RAISE EXCEPTION 'BC9: referencia de publicación ambigua' USING ERRCODE='55000'; END;
 IF v.estado<>'publicada'
 OR v.referencia IS DISTINCT FROM v.convocatoria_id||'#'||v_secuencia::text
 OR encode(sha256(v.version_canonica),'hex') IS DISTINCT FROM v.huella_version_sha256
 OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
   WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v_secuencia
    AND posterior.estado IN('publicada','sustituida','retirada'))
 THEN RAISE EXCEPTION 'BC9: publicación no vigente' USING ERRCODE='B9601'; END IF;
 BEGIN c:=convert_from(v.version_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: versión gobernada ilegible' USING ERRCODE='55000';
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
 THEN RAISE EXCEPTION 'BC9: material publicado incompleto' USING ERRCODE='55000'; END IF;
 IF jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 THEN RAISE EXCEPTION 'BC9: categorías publicadas incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT ((c#>'{contenido,categorias}') @> jsonb_build_array(p_categoria_ref))
 THEN RAISE EXCEPTION 'BC9: categoría ajena a la publicación' USING ERRCODE='B9605'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END IF;

 instante:=date_trunc('microseconds',clock_timestamp());
 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
   WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción inválido' USING ERRCODE='55000';
  END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
     OR candidato->>'referencia' IS NULL THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción incompleto' USING ERRCODE='55000';
  END IF;
  IF abre<=instante AND instante<cierra THEN
   activos:=activos+1; plazo:=candidato;
  END IF;
 END LOOP;
 IF activos<>1 THEN RAISE EXCEPTION 'BC9: plazo de inscripción cerrado o ambiguo' USING ERRCODE='B9602'; END IF;
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
 THEN RAISE EXCEPTION 'BC9: referencias de bases incompletas' USING ERRCODE='55000'; END IF;
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
 THEN RAISE EXCEPTION 'BC9: requisitos incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',categoria.valor)
   ORDER BY categoria.valor),'[]'::jsonb)
 INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS categoria(valor);

 RETURN jsonb_build_object(
  'convocatoria_ref',p_convocatoria_ref,
  'convocatoria_id',v.convocatoria_id,'secuencia',v_secuencia,
  'categoria_ref',p_categoria_ref,'categorias',categorias,
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
  'comprobada_en',instante);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_convocatorias_ejecutor_preparacion_bases,
 vec_bolsa_convocatorias_lector_preparacion_bases;
GRANT USAGE ON SCHEMA vec_bolsa_convocatorias TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)
 TO vec_bolsa_llamamientos_propietario;

-- Proyección de la versión gobernada; NULL significa que el plazo no está
-- abierto. No consulta la proyección pública ni deriva requisitos desde texto.
CREATE FUNCTION vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(
 p_id text,p_secuencia bigint,p_canonica bytea,p_huella_sha256 text,p_instante timestamptz
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE c jsonb; catalogo jsonb; politica jsonb; formulario jsonb;
 bases jsonb; requisitos jsonb; pendientes jsonb; categorias jsonb;
 plazo jsonb; candidato jsonb; abre timestamptz; cierra timestamptz;
 activos integer:=0; referencia text; publicada timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_id IS NULL OR p_id !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]*$'
 OR octet_length(p_id) NOT BETWEEN 1 AND 480
 OR p_secuencia NOT BETWEEN 1 AND 9007199254740991
 OR p_canonica IS NULL OR p_huella_sha256 IS NULL
 OR encode(sha256(p_canonica),'hex') IS DISTINCT FROM p_huella_sha256
 OR p_instante IS NULL
 THEN RAISE EXCEPTION 'BC9: versión de lectura inválida' USING ERRCODE='55000'; END IF;
 referencia:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(p_id)||'_v'||p_secuencia::text;
 BEGIN c:=convert_from(p_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: versión gobernada ilegible' USING ERRCODE='55000'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR c->>'id' IS DISTINCT FROM p_id OR c->>'secuencia' IS DISTINCT FROM p_secuencia::text
 OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR jsonb_typeof(c#>'{contenido,plazos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,requisitos}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,requisitos}')>256
 OR jsonb_typeof(c#>'{contenido,catalogo_categorias}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,catalogos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,flujo_solicitud}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
 OR c#>>'{contenido,identificador_publico}' IS NULL
 OR c#>>'{contenido,titulo}' IS NULL OR c#>>'{contenido,resumen}' IS NULL
 OR c#>>'{contenido,tipo}' IS NULL
 OR c->>'publicada_en' IS NULL
 OR jsonb_typeof(c->'aprobacion_publicacion') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c->'comprobacion_dependencias') IS DISTINCT FROM 'object'
 OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM p_id||'#'||p_secuencia::text
 OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM p_id||'#'||p_secuencia::text
 THEN RAISE EXCEPTION 'BC9: material publicado incompleto' USING ERRCODE='55000'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END IF;

 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
  WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción inválido' USING ERRCODE='55000'; END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
   OR abre>=cierra OR candidato->>'referencia' IS NULL
  THEN RAISE EXCEPTION 'BC9: plazo de inscripción incompleto' USING ERRCODE='55000'; END IF;
  IF abre<=p_instante AND p_instante<cierra THEN
   activos:=activos+1; plazo:=candidato;
  END IF;
 END LOOP;
 IF activos=0 THEN RETURN NULL; END IF;
 IF activos<>1 THEN RAISE EXCEPTION 'BC9: plazos abiertos ambiguos' USING ERRCODE='55000'; END IF;

 catalogo:=c#>'{contenido,catalogo_categorias}';
 politica:=c#>'{configuracion,catalogos}';
 formulario:=c#>'{configuracion,flujo_solicitud}';
 requisitos:=c#>'{contenido,requisitos}';
 SELECT valor INTO bases FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
  WHERE valor->>'rol'='bases';
 IF coalesce(catalogo->>'catalogo_id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(catalogo->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(catalogo->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 OR coalesce(politica->>'id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(politica->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR formulario->>'id' IS NULL
 OR coalesce(formulario->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(formulario->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
    WHERE valor->>'rol'='bases')<>1
 OR bases->>'publicacion_ref' IS NULL
 OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(requisitos) AS r(valor)
   WHERE coalesce(r.valor->>'referencia','')=''
    OR jsonb_typeof(r.valor->'obligatorio') IS DISTINCT FROM 'boolean'
    OR coalesce(r.valor->>'orden','') !~ '^[1-9][0-9]{0,5}$')
 THEN RAISE EXCEPTION 'BC9: referencias publicadas incompletas' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',x.valor)
  ORDER BY x.valor COLLATE "C"),'[]'::jsonb)
 INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'referencia',r.valor->>'referencia','codigo',r.valor->>'referencia',
  'descripcion',r.valor->>'descripcion',
  'obligatorio',(r.valor->>'obligatorio')::boolean,
  'estado','pendiente','motivo_codigo','requisito.pendiente')
  ORDER BY (r.valor->>'orden')::integer,r.valor->>'referencia'),'[]'::jsonb)
 INTO pendientes FROM jsonb_array_elements(requisitos) AS r(valor);
 abre:=(plazo->>'abre_en')::timestamptz;
 cierra:=(plazo->>'cierra_en')::timestamptz;
 RETURN jsonb_build_object(
  'convocatoria_ref',referencia,'convocatoria_id',p_id,'secuencia',p_secuencia,
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'titulo',c#>>'{contenido,titulo}','resumen',c#>>'{contenido,resumen}',
  'tipo',c#>>'{contenido,tipo}',
  'numero_categorias',jsonb_array_length(c#>'{contenido,categorias}'),
  'categoria_ref_comprobacion',c#>>'{contenido,categorias,0}',
  'categorias_refs_comprobacion',c#>'{contenido,categorias}',
  'categorias',categorias,'numero_requisitos',jsonb_array_length(requisitos),
  'version_sha256',p_huella_sha256,'bases_ref',bases->>'publicacion_ref',
  'catalogo_ref',catalogo->>'catalogo_id',
  'catalogo_version',(catalogo->>'catalogo_version')::integer,
  'catalogo_sha256',catalogo->>'catalogo_huella_sha256',
  'politica_catalogo_ref',politica->>'id',
  'politica_catalogo_version',(politica->>'version')::integer,
  'politica_catalogo_sha256',politica->>'huella_contenido_sha256',
  'formulario_ref',formulario->>'id',
  'formulario_version',(formulario->>'version')::integer,
  'formulario_sha256',formulario->>'huella_contenido_sha256',
  'plazo_ref',plazo->>'referencia','plazo_abre_en',abre,'plazo_cierra_en',cierra,
  'requisitos',pendientes,
  'requisitos_sha256',encode(sha256(convert_to(requisitos::text,'UTF8')),'hex'),
  'comprobada_en',p_instante);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamptz)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_convocatorias_ejecutor_preparacion_bases,
 vec_bolsa_convocatorias_lector_preparacion_bases,vec_bolsa_llamamientos_propietario;

-- El total y la página salen del mismo conjunto materializado y del mismo
-- instante. El cursor es la referencia cv1 del último elemento, orden C.
CREATE FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(
 p_cursor text,p_limite integer
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb; instante timestamptz(6):=date_trunc('microseconds',statement_timestamp());
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
 OR (p_cursor IS NOT NULL AND (octet_length(p_cursor) NOT BETWEEN 9 AND 200
   OR p_cursor !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'))
 THEN RAISE EXCEPTION 'BC9: paginación inválida' USING ERRCODE='22023'; END IF;
 WITH candidatas AS MATERIALIZED (
  SELECT v.convocatoria_id,v.secuencia,v.version_canonica,v.huella_version_sha256
  FROM vec_bolsa_convocatorias.version_convocatoria AS v
  WHERE v.estado='publicada'
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria AS posterior
    WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
     AND posterior.estado IN('publicada','sustituida','retirada'))
 ), proyectadas AS MATERIALIZED (
  SELECT vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(
   v.convocatoria_id,v.secuencia,v.version_canonica,v.huella_version_sha256,instante) AS datos
  FROM candidatas v
 ), abiertas AS MATERIALIZED (
  SELECT jsonb_build_object(
   'convocatoria_ref',p.datos->>'convocatoria_ref',
   'titulo',p.datos->>'titulo','resumen',p.datos->>'resumen',
   'numero_categorias',p.datos->'numero_categorias',
   'numero_requisitos',p.datos->'numero_requisitos',
   'categoria_ref_comprobacion',p.datos->>'categoria_ref_comprobacion',
   'categorias_refs_comprobacion',p.datos->'categorias_refs_comprobacion',
   'catalogo_ref',p.datos->>'catalogo_ref',
   'catalogo_version',p.datos->'catalogo_version',
   'catalogo_sha256',p.datos->>'catalogo_sha256',
   'politica_catalogo_ref',p.datos->>'politica_catalogo_ref',
   'politica_catalogo_version',p.datos->'politica_catalogo_version',
   'politica_catalogo_sha256',p.datos->>'politica_catalogo_sha256',
   'plazo_abre_en',p.datos->'plazo_abre_en',
   'plazo_cierra_en',p.datos->'plazo_cierra_en') AS datos,
   p.datos->>'convocatoria_ref' AS cursor
  FROM proyectadas p WHERE p.datos IS NOT NULL
 ), pagina AS (
  SELECT datos,cursor,row_number() OVER (ORDER BY cursor COLLATE "C") AS orden
  FROM abiertas WHERE p_cursor IS NULL OR cursor COLLATE "C">p_cursor COLLATE "C"
  ORDER BY cursor COLLATE "C" LIMIT p_limite+1
 )
 SELECT jsonb_build_object(
  'total',(SELECT count(*) FROM abiertas),
  'items',coalesce(jsonb_agg(datos ORDER BY orden) FILTER (WHERE orden<=p_limite),'[]'::jsonb),
  'hay_mas',count(*)>p_limite,
  'siguiente_cursor',CASE WHEN count(*)>p_limite THEN
    (SELECT cursor FROM pagina WHERE orden=p_limite) ELSE NULL END,
  'comprobada_en',instante)
 INTO resultado FROM pagina;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(text,integer)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_convocatorias_ejecutor_preparacion_bases,
 vec_bolsa_convocatorias_lector_preparacion_bases;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(text,integer)
 TO vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(
 p_convocatoria_ref text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE partes text[]; v_id_sha256 text; v_secuencia bigint;
 v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 resultado jsonb; instante timestamptz(6):=date_trunc('microseconds',statement_timestamp());
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 9 AND 200
 THEN RAISE EXCEPTION 'BC9: selector de detalle inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;
 v_id_sha256:=partes[1];
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;
 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"=v_id_sha256 COLLATE "C"
    AND secuencia=v_secuencia;
 EXCEPTION WHEN no_data_found THEN RETURN NULL;
 WHEN too_many_rows THEN
  RAISE EXCEPTION 'BC9: referencia de detalle ambigua' USING ERRCODE='55000'; END;
 IF v.referencia IS DISTINCT FROM v.convocatoria_id||'#'||v.secuencia::text
 THEN RAISE EXCEPTION 'BC9: referencia de detalle incompatible' USING ERRCODE='55000'; END IF;
 IF v.estado<>'publicada' OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
  WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
   AND posterior.estado IN('publicada','sustituida','retirada'))
 THEN RETURN NULL; END IF;
 resultado:=vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(
  v.convocatoria_id,v.secuencia,v.version_canonica,v.huella_version_sha256,instante);
 IF resultado IS NULL THEN
  RETURN NULL;
 END IF;
 RETURN resultado - 'categoria_ref_comprobacion' - 'categorias_refs_comprobacion';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(text)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_convocatorias_ejecutor_preparacion_bases,
 vec_bolsa_convocatorias_lector_preparacion_bases;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(text)
 TO vec_bolsa_llamamientos_propietario;
-- Lectura histórica exacta para asociar una solicitud ya presentada; la
-- inscripción nueva sigue usando la comprobación de vigencia anterior.
CREATE FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
 p_convocatoria_ref text,p_categoria_ref text,p_version_sha256 text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE partes text[]; v_id_sha256 text; v_secuencia bigint;
 v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 c jsonb; catalogo jsonb; politica jsonb; formulario jsonb; bases jsonb;
 requisitos jsonb; plazos jsonb; candidato jsonb; publicada timestamptz;
 abre timestamptz; cierra timestamptz; total_plazos integer:=0;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 9 AND 200
 OR p_categoria_ref IS NULL OR p_categoria_ref !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version_sha256 IS NULL OR p_version_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'BC9: selector inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;
 v_id_sha256:=partes[1];
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;

 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"=v_id_sha256 COLLATE "C"
    AND secuencia=v_secuencia FOR SHARE;
 EXCEPTION WHEN no_data_found THEN
  RAISE EXCEPTION 'BC9: publicación original no acreditada' USING ERRCODE='B9601';
 WHEN too_many_rows THEN
  RAISE EXCEPTION 'BC9: referencia histórica ambigua' USING ERRCODE='55000'; END;
 IF v.estado<>'publicada'
 OR v.referencia IS DISTINCT FROM v.convocatoria_id||'#'||v_secuencia::text
 OR v.huella_version_sha256 IS DISTINCT FROM p_version_sha256
 OR encode(sha256(v.version_canonica),'hex') IS DISTINCT FROM p_version_sha256
 THEN RAISE EXCEPTION 'BC9: publicación original no acreditada' USING ERRCODE='B9601'; END IF;
 BEGIN c:=convert_from(v.version_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: versión original ilegible' USING ERRCODE='55000'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR c->>'id' IS DISTINCT FROM v.convocatoria_id
 OR c->>'secuencia' IS DISTINCT FROM v_secuencia::text
 OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(c->'aprobacion_publicacion') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c->'comprobacion_dependencias') IS DISTINCT FROM 'object'
 OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR jsonb_typeof(c#>'{contenido,plazos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,requisitos}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,requisitos}')>256
 OR jsonb_typeof(c#>'{contenido,catalogo_categorias}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,catalogos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,flujo_solicitud}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
 OR c#>>'{contenido,identificador_publico}' IS NULL
 OR c#>>'{contenido,titulo}' IS NULL OR c#>>'{contenido,resumen}' IS NULL
 OR c#>>'{contenido,tipo}' IS NULL
 OR c->>'publicada_en' IS NULL
 THEN RAISE EXCEPTION 'BC9: publicación original incompleta' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 THEN RAISE EXCEPTION 'BC9: categorías originales incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT ((c#>'{contenido,categorias}') @> jsonb_build_array(p_categoria_ref))
 OR (SELECT count(*) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
     WHERE x.valor=p_categoria_ref)<>1
 THEN RAISE EXCEPTION 'BC9: categoría no pertenece a publicación original' USING ERRCODE='B9605'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: instante de publicación inválido' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC9: instante de publicación inválido' USING ERRCODE='55000'; END IF;

 catalogo:=c#>'{contenido,catalogo_categorias}';
 politica:=c#>'{configuracion,catalogos}';
 formulario:=c#>'{configuracion,flujo_solicitud}';
 requisitos:=c#>'{contenido,requisitos}';
 SELECT valor INTO bases FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
  WHERE valor->>'rol'='bases';
 IF coalesce(catalogo->>'catalogo_id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(catalogo->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(catalogo->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 OR coalesce(politica->>'id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(politica->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR formulario->>'id' IS NULL
 OR coalesce(formulario->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(formulario->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
     WHERE valor->>'rol'='bases')<>1
 OR bases->>'publicacion_ref' IS NULL
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(requisitos) AS r(valor)
   WHERE coalesce(r.valor->>'referencia','')=''
    OR jsonb_typeof(r.valor->'obligatorio') IS DISTINCT FROM 'boolean'
    OR coalesce(r.valor->>'orden','') !~ '^[1-9][0-9]{0,5}$')
 THEN RAISE EXCEPTION 'BC9: referencias originales incompletas' USING ERRCODE='55000'; END IF;
 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
  WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC9: plazo original inválido' USING ERRCODE='55000'; END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
   OR abre>=cierra OR candidato->>'referencia' IS NULL
  THEN RAISE EXCEPTION 'BC9: plazo original incompleto' USING ERRCODE='55000'; END IF;
  total_plazos:=total_plazos+1;
 END LOOP;
 IF total_plazos NOT BETWEEN 1 AND 64
 THEN RAISE EXCEPTION 'BC9: sin plazo de inscripción original' USING ERRCODE='B9601'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('plazo_ref',p.valor->>'referencia',
  'plazo_abre_en',(p.valor->>'abre_en')::timestamptz,
  'plazo_cierra_en',(p.valor->>'cierra_en')::timestamptz)
  ORDER BY p.valor->>'referencia'),'[]'::jsonb)
 INTO plazos FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
 WHERE p.valor->>'tipo'='inscripcion';

 RETURN jsonb_build_object(
  'convocatoria_ref',p_convocatoria_ref,'convocatoria_id',v.convocatoria_id,
  'secuencia',v_secuencia,'version_sha256',p_version_sha256,
  'categoria_ref',p_categoria_ref,
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'bases_ref',bases->>'publicacion_ref','publicada_en',publicada,
  'catalogo_ref',catalogo->>'catalogo_id',
  'catalogo_version',(catalogo->>'catalogo_version')::integer,
  'catalogo_sha256',catalogo->>'catalogo_huella_sha256',
  'politica_catalogo_ref',politica->>'id',
  'politica_catalogo_version',(politica->>'version')::integer,
  'politica_catalogo_sha256',politica->>'huella_contenido_sha256',
  'formulario_ref',formulario->>'id',
  'formulario_version',(formulario->>'version')::integer,
  'formulario_sha256',formulario->>'huella_contenido_sha256',
  'requisitos_sha256',encode(sha256(convert_to(requisitos::text,'UTF8')),'hex'),
  'plazos_inscripcion',plazos);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_convocatorias_ejecutor_preparacion_bases,
 vec_bolsa_convocatorias_lector_preparacion_bases;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)
 TO vec_bolsa_llamamientos_propietario;
COMMIT;
