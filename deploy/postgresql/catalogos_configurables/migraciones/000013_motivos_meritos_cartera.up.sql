\set ON_ERROR_STOP on
-- CC13: proyección privada de etiquetas gobernadas para motivos de retirada M4.
-- El documento CC declara la referencia AUT, pero CC no consulta tablas AUT.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_catalogos_configurables:migracion:000013',0));
DO $pre$ DECLARE cc11 pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO cc11 FROM pg_proc p WHERE p.oid=to_regprocedure(
  'vec_catalogos_configurables.listar_motivos_inscripcion_v1(text)');
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_catalogos_configurables.publicacion') IS NULL
 OR to_regclass('vec_catalogos_configurables.entrada_publicada') IS NULL
 OR to_regclass('vec_catalogos_configurables.categoria_control') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(text)') IS NOT NULL
 OR NOT FOUND OR cc11.proowner<>to_regrole('vec_catalogos_configurables_propietario')
 OR cc11.prosecdef IS NOT TRUE OR cc11.provolatile<>'s' OR cc11.proparallel<>'r'
 OR cc11.proconfig IS NULL
 OR has_function_privilege('public',cc11.oid,'EXECUTE')
 OR to_regrole('vec_autorizacion_propietario') IS NULL
 THEN RAISE EXCEPTION 'CC13: postimagen CC11 o autoridad incompatible' USING ERRCODE='55000';END IF;
END $pre$;

CREATE FUNCTION vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(p_idioma text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE pub record;doc jsonb;vinculo jsonb;filas jsonb;total integer;ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_idioma NOT IN ('es','en') OR p_idioma IS NULL
 OR session_user<>'vec_externo_meritos_cartera'
 OR NOT pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 THEN RAISE EXCEPTION 'CC13: lectura de motivos denegada' USING ERRCODE='42501';END IF;
 SELECT p.* INTO pub FROM vec_catalogos_configurables.publicacion p
 WHERE p.catalogo_id='meritos.cartera.motivos'
 ORDER BY p.version DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'CC13: catálogo sin publicación' USING ERRCODE='55000';END IF;
 BEGIN doc:=pub.documento_canonico::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CC13: documento incompatible' USING ERRCODE='55000';END;
 vinculo:=doc->'vinculo_autorizacion';
 IF doc->>'id' IS DISTINCT FROM pub.catalogo_id
 OR doc->>'version' IS DISTINCT FROM pub.version::text
 OR doc->>'estado' IS DISTINCT FROM 'publicado'
 OR jsonb_typeof(vinculo) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(vinculo))<>3
 OR (vinculo ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256']) IS NOT TRUE
 OR vinculo->>'catalogo_id' IS DISTINCT FROM 'motivos_meritos_cartera'
 OR jsonb_typeof(vinculo->'catalogo_version') IS DISTINCT FROM 'number'
 OR coalesce(vinculo->>'catalogo_version' ~ '^[1-9][0-9]{0,8}$',false) IS NOT TRUE
 OR coalesce(vinculo->>'catalogo_huella_sha256' ~ '^[0-9a-f]{64}$',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'CC13: vínculo de autorización incompatible' USING ERRCODE='55000';END IF;
 ahora:=clock_timestamp();
 WITH activos AS MATERIALIZED (
  SELECT e.categoria_id,e.etiqueta,e.definicion
  FROM vec_catalogos_configurables.entrada_publicada e
  JOIN vec_catalogos_configurables.categoria_control c ON c.categoria_id=e.categoria_id
  WHERE e.catalogo_id=pub.catalogo_id AND e.version=pub.version
   AND e.huella_sha256=pub.huella_sha256
   AND c.catalogo_id=e.catalogo_id AND c.version=e.version
   AND c.huella_sha256=e.huella_sha256 AND c.estado='habilitada'
 ),
 verificadas AS (
  SELECT categoria_id,etiqueta,definicion,
   definicion#>>'{etiquetas,es}' AS etiqueta_es,
   definicion#>>'{etiquetas,en}' AS etiqueta_en
  FROM activos
 )
 SELECT count(*)::integer,
  coalesce(jsonb_agg(jsonb_build_object(
   'entrada_clave',categoria_id,
   'etiqueta',CASE WHEN p_idioma='es' THEN etiqueta_es ELSE etiqueta_en END)
   ORDER BY categoria_id COLLATE "C"),'[]'::jsonb)
 INTO total,filas FROM verificadas
 WHERE categoria_id ~ '^motivo_[0-9a-f]{32}$'
  AND jsonb_typeof(definicion->'etiquetas')='object'
  AND (SELECT count(*) FROM jsonb_object_keys(definicion->'etiquetas'))=2
  AND definicion->'etiquetas' ?& ARRAY['es','en']
  AND etiqueta_es IS NOT NULL AND etiqueta_en IS NOT NULL
  AND etiqueta_es=etiqueta
  AND octet_length(etiqueta_es) BETWEEN 1 AND 200
  AND octet_length(etiqueta_en) BETWEEN 1 AND 200
  AND etiqueta_es=btrim(etiqueta_es) AND etiqueta_en=btrim(etiqueta_en)
  AND etiqueta_es !~ '[[:cntrl:]]' AND etiqueta_en !~ '[[:cntrl:]]';
 IF total NOT BETWEEN 1 AND 128
 OR total<>(SELECT count(*) FROM vec_catalogos_configurables.entrada_publicada e
   JOIN vec_catalogos_configurables.categoria_control c ON c.categoria_id=e.categoria_id
   WHERE e.catalogo_id=pub.catalogo_id AND e.version=pub.version
    AND e.huella_sha256=pub.huella_sha256
    AND c.catalogo_id=e.catalogo_id AND c.version=e.version
    AND c.huella_sha256=e.huella_sha256 AND c.estado='habilitada')
 THEN RAISE EXCEPTION 'CC13: etiquetas incompletas o fuera de límite' USING ERRCODE='55000';END IF;
 RETURN jsonb_build_object('catalogo_cc',jsonb_build_object(
  'referencia',pub.catalogo_id,'version',pub.version,'huella_sha256',pub.huella_sha256),
  'catalogo_aut',vinculo,'idioma',p_idioma,'motivos',filas,
  'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(text)
 TO vec_autorizacion_propietario;
COMMIT;
