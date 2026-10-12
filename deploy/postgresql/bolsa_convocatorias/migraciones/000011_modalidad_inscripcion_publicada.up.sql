\set ON_ERROR_STOP on
-- BC11: modalidad y conjunto elegido de categorias de la version sellada.
-- Separada exige una; conjunta admite cualquier subconjunto no vacio.
-- BC9 conserva su contrato V1 para solicitudes historicas. No hay DOWN:
-- las publicaciones y solicitudes anteriores no se reescriben.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000011',0));

DO $pre$
DECLARE f regprocedure;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria_token_inscripcion_uq') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 OR to_regrole('vec_bolsa_convocatorias_ejecutor_consulta') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(text,text[])') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(text,text[],text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(text,integer)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(text)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_index i
   WHERE i.indexrelid='vec_bolsa_convocatorias.version_convocatoria_token_inscripcion_uq'::regclass
    AND i.indisunique AND i.indisvalid)
 THEN RAISE EXCEPTION 'BC11: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[
  to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)'),
  to_regprocedure('vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)'),
  to_regprocedure('vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(text,integer)'),
  to_regprocedure('vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(text)')]
 LOOP
  IF f IS NULL OR NOT has_function_privilege('vec_bolsa_convocatorias_propietario',f,'EXECUTE')
   OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
  THEN RAISE EXCEPTION 'BC11: falta contrato BC9' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Una sola busqueda por lote mediante el indice del token BC9. El resultado
-- preserva el orden solicitado. La modalidad ausente es NULL, nunca separada.
-- La funcion es interna del propietario; sus consumidores no leen la tabla.
CREATE FUNCTION vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(p_refs text[])
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE x record; c jsonb; docs jsonb; bases jsonb; categorias jsonb;
 modalidad text; salida jsonb:='[]'::jsonb; total integer:=0;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_refs IS NULL OR cardinality(p_refs)>100
 OR EXISTS(SELECT 1 FROM unnest(p_refs) AS r(ref)
  WHERE r.ref IS NULL OR r.ref !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
   OR CASE WHEN r.ref ~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
    THEN split_part(r.ref,'_v',2)::bigint>9007199254740991 ELSE true END)
 OR (SELECT count(DISTINCT r.ref COLLATE "C") FROM unnest(p_refs) AS r(ref))<>cardinality(p_refs)
 THEN RAISE EXCEPTION 'BC11: selectores invalidos' USING ERRCODE='22023'; END IF;
 IF cardinality(p_refs)=0 THEN RETURN salida; END IF;

 BEGIN
  FOR x IN
   WITH pedidos AS MATERIALIZED (
    SELECT r.ref,r.orden,
     substring(r.ref from '^cv1_([0-9a-f]{64})_v[1-9][0-9]{0,15}$') AS id_sha256,
     split_part(r.ref,'_v',2)::bigint AS secuencia
    FROM unnest(p_refs) WITH ORDINALITY AS r(ref,orden)
   )
   SELECT p.ref,p.orden,v.convocatoria_id,v.secuencia,v.referencia,v.estado,
    v.version_canonica,v.huella_version_sha256
   FROM pedidos p JOIN vec_bolsa_convocatorias.version_convocatoria v
    ON vec_bolsa_convocatorias.huella_id_inscripcion_v1(v.convocatoria_id)
      COLLATE "C"=p.id_sha256 COLLATE "C" AND v.secuencia=p.secuencia
   ORDER BY p.orden
  LOOP
   total:=total+1;
   IF x.estado<>'publicada'
    OR x.referencia IS DISTINCT FROM x.convocatoria_id||'#'||x.secuencia::text
    OR encode(sha256(x.version_canonica),'hex') IS DISTINCT FROM x.huella_version_sha256
   THEN RAISE EXCEPTION 'BC11: version no acreditada' USING ERRCODE='B9601'; END IF;
   c:=convert_from(x.version_canonica,'UTF8')::jsonb;
   IF jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR c->>'id' IS DISTINCT FROM x.convocatoria_id
    OR c->>'secuencia' IS DISTINCT FROM x.secuencia::text
    OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
    OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM x.referencia
    OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM x.referencia
    OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
    OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'BC11: version gobernada incompatible' USING ERRCODE='B9601'; END IF;
   IF jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
    OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS z(ref)
     WHERE z.ref !~ '^[a-z][a-z0-9_.:-]{2,127}$')
    OR (SELECT count(DISTINCT z.ref COLLATE "C")
     FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS z(ref))
      <>jsonb_array_length(c#>'{contenido,categorias}')
   THEN RAISE EXCEPTION 'BC11: categorias publicadas incompatibles' USING ERRCODE='B9601'; END IF;
   SELECT coalesce(jsonb_agg(z.ref ORDER BY z.ref COLLATE "C"),'[]'::jsonb)
    INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS z(ref);
   modalidad:=c#>>'{contenido,modalidad_inscripcion}';
   IF c#>'{contenido,modalidad_inscripcion}' IS NOT NULL
    AND (jsonb_typeof(c#>'{contenido,modalidad_inscripcion}') IS DISTINCT FROM 'string'
     OR modalidad NOT IN('separada','conjunta'))
   THEN RAISE EXCEPTION 'BC11: modalidad publicada invalida' USING ERRCODE='B9601'; END IF;
   docs:=c#>'{configuracion,documentos}';
   SELECT d.valor INTO bases FROM jsonb_array_elements(docs) AS d(valor)
    WHERE d.valor->>'rol'='bases';
   IF NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(docs) AS d(valor)
      WHERE d.valor->>'rol'='bases')<>1
    OR coalesce(bases->>'publicacion_ref','')=''
   THEN RAISE EXCEPTION 'BC11: bases publicadas incompatibles' USING ERRCODE='B9601'; END IF;
   -- Los documentos anteriores a BC11 pueden carecer de version y huella.
   -- Una nueva modalidad exige la identidad completa del documento oficial.
   IF modalidad IS NOT NULL AND (
    coalesce(bases->>'documento_ref','')=''
    OR coalesce(bases->>'version_documento','') !~ '^[1-9][0-9]{0,8}$'
    OR coalesce(bases->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$')
   THEN RAISE EXCEPTION 'BC11: documento de bases incompleto' USING ERRCODE='B9601'; END IF;
   salida:=salida||jsonb_build_array(jsonb_build_object(
    'convocatoria_ref',x.ref,'modalidad_inscripcion',modalidad,
    'categorias_refs',categorias,'numero_categorias',jsonb_array_length(categorias),
    'secuencia',x.secuencia,'version_sha256',x.huella_version_sha256,
    'bases_ref',bases->>'publicacion_ref',
    'bases_documento_ref',bases->>'documento_ref',
    'bases_version_documento',CASE WHEN coalesce(bases->>'version_documento','')
      ~ '^[1-9][0-9]{0,8}$' THEN (bases->>'version_documento')::integer ELSE NULL END,
    'bases_sha256',bases->>'huella_contenido_sha256'));
  END LOOP;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC11: version publicada ilegible' USING ERRCODE='B9601';
 END;
 IF total<>cardinality(p_refs)
 THEN RAISE EXCEPTION 'BC11: version publicada ausente' USING ERRCODE='B9601'; END IF;
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta,vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(
 p_convocatoria_ref text,p_categorias_refs text[]
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE base jsonb; meta jsonb; elegidas jsonb; n integer;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_categorias_refs IS NULL OR cardinality(p_categorias_refs) NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM unnest(p_categorias_refs) AS z(ref)
   WHERE z.ref IS NULL OR z.ref !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT z.ref COLLATE "C") FROM unnest(p_categorias_refs) AS z(ref))
    <>cardinality(p_categorias_refs)
 THEN RAISE EXCEPTION 'BC11: categorias elegidas invalidas' USING ERRCODE='22023'; END IF;
 base:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(
  p_convocatoria_ref,p_categorias_refs[1]);
 meta:=vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(
  ARRAY[p_convocatoria_ref])->0;
 IF meta->>'version_sha256' IS DISTINCT FROM base->>'version_sha256'
  OR meta->>'bases_ref' IS DISTINCT FROM base->>'bases_ref'
 THEN RAISE EXCEPTION 'BC11: version comprobada divergente' USING ERRCODE='B9601'; END IF;
 IF meta->>'modalidad_inscripcion' IS NULL
 THEN RAISE EXCEPTION 'BC11: publicacion sin modalidad nueva' USING ERRCODE='B9601'; END IF;
 SELECT jsonb_agg(z.ref ORDER BY z.ref COLLATE "C") INTO elegidas
  FROM unnest(p_categorias_refs) AS z(ref);
 n:=cardinality(p_categorias_refs);
 IF (meta->>'modalidad_inscripcion'='separada' AND n<>1)
  OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(elegidas) AS z(ref)
    WHERE NOT (meta->'categorias_refs' @> jsonb_build_array(z.ref)))
 THEN RAISE EXCEPTION 'BC11: conjunto de categorias ajeno a modalidad' USING ERRCODE='B9605'; END IF;
 RETURN base||meta||jsonb_build_object('categorias_refs_elegidas',elegidas);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(text,text[])
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(text,text[])
 TO vec_bolsa_llamamientos_propietario;

-- Replay: BC9 acredita la publicacion original y su huella sin pedir plazo
-- abierto. Una sucesora no altera la fila original publicada. El legado
-- conserva modalidad NULL y su seleccion V1 individual.
CREATE FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(
 p_convocatoria_ref text,p_categorias_refs text[],p_version_sha256 text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE base jsonb; meta jsonb; elegidas jsonb; n integer;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_categorias_refs IS NULL OR cardinality(p_categorias_refs) NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM unnest(p_categorias_refs) AS z(ref)
   WHERE z.ref IS NULL OR z.ref !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT z.ref COLLATE "C") FROM unnest(p_categorias_refs) AS z(ref))
    <>cardinality(p_categorias_refs)
 THEN RAISE EXCEPTION 'BC11: categorias historicas invalidas' USING ERRCODE='22023'; END IF;
 base:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
  p_convocatoria_ref,p_categorias_refs[1],p_version_sha256);
 meta:=vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(
  ARRAY[p_convocatoria_ref])->0;
 IF meta->>'version_sha256' IS DISTINCT FROM p_version_sha256
  OR meta->>'bases_ref' IS DISTINCT FROM base->>'bases_ref'
 THEN RAISE EXCEPTION 'BC11: version historica divergente' USING ERRCODE='B9601'; END IF;
 SELECT jsonb_agg(z.ref ORDER BY z.ref COLLATE "C") INTO elegidas
  FROM unnest(p_categorias_refs) AS z(ref);
 n:=cardinality(p_categorias_refs);
 IF ((meta->>'modalidad_inscripcion' IS NULL
    OR meta->>'modalidad_inscripcion'='separada') AND n<>1)
  OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(elegidas) AS z(ref)
    WHERE NOT (meta->'categorias_refs' @> jsonb_build_array(z.ref)))
 THEN RAISE EXCEPTION 'BC11: conjunto historico incompatible' USING ERRCODE='B9605'; END IF;
 RETURN base||meta||jsonb_build_object('categorias_refs_elegidas',elegidas);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(text,text[],text)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(text,text[],text)
 TO vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(
 p_cursor text,p_limite integer
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE pagina jsonb; metas jsonb; items jsonb; refs text[];
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 THEN RAISE EXCEPTION 'BC11: lector invalido' USING ERRCODE='42501'; END IF;
 pagina:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(p_cursor,p_limite);
 SELECT coalesce(array_agg(x.valor->>'convocatoria_ref' ORDER BY x.orden),ARRAY[]::text[])
  INTO refs FROM jsonb_array_elements(pagina->'items') WITH ORDINALITY AS x(valor,orden);
 metas:=vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(refs);
 SELECT coalesce(jsonb_agg(x.valor||y.valor ORDER BY x.orden),'[]'::jsonb)
  INTO items FROM jsonb_array_elements(pagina->'items') WITH ORDINALITY AS x(valor,orden)
  JOIN jsonb_array_elements(metas) WITH ORDINALITY AS y(valor,orden) ON x.orden=y.orden
  WHERE x.valor->>'convocatoria_ref'=y.valor->>'convocatoria_ref';
 IF jsonb_array_length(items)<>cardinality(refs)
 THEN RAISE EXCEPTION 'BC11: pagina divergente' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_set(pagina,'{items}',items);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(text,integer)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(text,integer)
 TO vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(
 p_convocatoria_ref text
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE detalle jsonb; meta jsonb;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 THEN RAISE EXCEPTION 'BC11: lector invalido' USING ERRCODE='42501'; END IF;
 detalle:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(p_convocatoria_ref);
 IF detalle IS NULL THEN RETURN NULL; END IF;
 meta:=vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(
  ARRAY[p_convocatoria_ref])->0;
 IF meta->>'version_sha256' IS DISTINCT FROM detalle->>'version_sha256'
  OR meta->>'bases_ref' IS DISTINCT FROM detalle->>'bases_ref'
 THEN RAISE EXCEPTION 'BC11: detalle divergente' USING ERRCODE='B9601'; END IF;
 RETURN detalle||meta;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(text)
 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(text)
 TO vec_bolsa_llamamientos_propietario;

-- Si existen privilegios por defecto ajenos, abortar toda la migracion.
DO $acl$
DECLARE f regprocedure; dueno oid; consumidor oid;
BEGIN
 SELECT oid INTO dueno FROM pg_roles WHERE rolname='vec_bolsa_convocatorias_propietario';
 SELECT oid INTO consumidor FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario';
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])'::regprocedure,
  'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v2(text,text[])'::regprocedure,
  'vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v2(text,text[],text)'::regprocedure,
  'vec_bolsa_convocatorias.listar_abiertas_inscripcion_v2(text,integer)'::regprocedure,
  'vec_bolsa_convocatorias.detalle_abierta_inscripcion_v2(text)'::regprocedure]
 LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.privilege_type='EXECUTE'
     AND (a.grantee<>dueno AND (f=
      'vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])'::regprocedure
      OR a.grantee<>consumidor)))
   OR (f<>'vec_bolsa_convocatorias.leer_modalidades_inscripcion_lote_v2(text[])'::regprocedure
    AND NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE'))
  THEN RAISE EXCEPTION 'BC11: ACL inesperada en %',f USING ERRCODE='42501'; END IF;
 END LOOP;
END $acl$;
COMMIT;
