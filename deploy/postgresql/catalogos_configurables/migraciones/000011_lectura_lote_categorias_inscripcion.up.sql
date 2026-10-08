\set ON_ERROR_STOP on
-- CC11: seam del catálogo gobernado para etiquetas de inscripción. Sólo
-- devuelve el idioma pedido; EN ausente utiliza la etiqueta ES publicada.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_catalogos_configurables:migracion:000011',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_catalogos_configurables.publicacion') IS NULL
 OR to_regclass('vec_catalogos_configurables.entrada_publicada') IS NULL
 OR to_regclass('vec_catalogos_configurables.categoria_control') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CC11: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
 p_definicion jsonb,p_etiqueta_es text,p_idioma text
) RETURNS text
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_es text; v_en text; resultado text;
BEGIN
 IF p_idioma IS NULL OR p_idioma NOT IN('es','en') OR p_etiqueta_es IS NULL
 OR octet_length(p_etiqueta_es) NOT BETWEEN 1 AND 200
 OR jsonb_typeof(p_definicion) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'CC11: etiqueta no disponible' USING ERRCODE='22023'; END IF;
 IF p_definicion ? 'etiquetas' THEN
  IF jsonb_typeof(p_definicion->'etiquetas') IS DISTINCT FROM 'object'
  THEN RAISE EXCEPTION 'CC11: idiomas publicados incompatibles' USING ERRCODE='55000'; END IF;
  IF p_definicion->'etiquetas' ? 'es' THEN
   IF jsonb_typeof(p_definicion#>'{etiquetas,es}') IS DISTINCT FROM 'string'
   THEN RAISE EXCEPTION 'CC11: etiqueta ES publicada incompatible' USING ERRCODE='55000'; END IF;
   v_es:=p_definicion#>>'{etiquetas,es}';
  END IF;
  IF p_definicion->'etiquetas' ? 'en' THEN
   IF jsonb_typeof(p_definicion#>'{etiquetas,en}') IS DISTINCT FROM 'string'
   THEN RAISE EXCEPTION 'CC11: etiqueta EN publicada incompatible' USING ERRCODE='55000'; END IF;
   v_en:=p_definicion#>>'{etiquetas,en}';
  END IF;
 END IF;
 resultado:=CASE WHEN p_idioma='en' AND v_en IS NOT NULL THEN v_en
  ELSE coalesce(v_es,p_etiqueta_es) END;
 IF octet_length(resultado) NOT BETWEEN 1 AND 200
 THEN RAISE EXCEPTION 'CC11: etiqueta fuera de límite' USING ERRCODE='54000'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(jsonb,text,text) FROM PUBLIC;

CREATE FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text,p_referencias text[],p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb; total integer;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR p_referencias IS NULL OR cardinality(p_referencias) NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM unnest(p_referencias) AS r(ref)
   WHERE r.ref IS NULL OR r.ref !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT r.ref) FROM unnest(p_referencias) AS r(ref))<>cardinality(p_referencias)
 THEN RAISE EXCEPTION 'CC11: selector de etiquetas inválido' USING ERRCODE='22023'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.publicacion p
   WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version AND p.huella_sha256=p_huella_sha256)
 THEN RAISE EXCEPTION 'CC11: publicación exacta ausente' USING ERRCODE='B9601'; END IF;
 SELECT count(*),jsonb_agg(jsonb_build_object(
  'categoria_ref',r.ref,
  'categoria',vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(e.definicion,e.etiqueta,p_idioma))
  ORDER BY r.orden)
 INTO total,resultado
 FROM unnest(p_referencias) WITH ORDINALITY AS r(ref,orden)
 JOIN vec_catalogos_configurables.entrada_publicada e
   ON e.catalogo_id=p_catalogo_id AND e.version=p_version
  AND e.huella_sha256=p_huella_sha256 AND e.categoria_id=r.ref;
 IF total<>cardinality(p_referencias) THEN
  RAISE EXCEPTION 'CC11: categoría no publicada en versión exacta' USING ERRCODE='B9601';
 END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text
) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE politica jsonb;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CC11: selector de política inválido' USING ERRCODE='22023'; END IF;
 SELECT e.definicion INTO politica
 FROM vec_catalogos_configurables.publicacion p
 JOIN vec_catalogos_configurables.entrada_publicada e
  ON e.catalogo_id=p.catalogo_id AND e.version=p.version
  AND e.huella_sha256=p.huella_sha256
 WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version
  AND p.huella_sha256=p_huella_sha256
  AND e.categoria_id='presentacion.con.pendientes';
 IF NOT FOUND OR jsonb_typeof(politica->'valor') IS DISTINCT FROM 'boolean'
 OR politica->'valor' IS DISTINCT FROM 'true'::jsonb
 OR politica->>'canal' IS DISTINCT FROM 'externa_personal'
 THEN RAISE EXCEPTION 'CC11: presentación pendiente sin política publicada' USING ERRCODE='B9601'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

-- Una sola llamada desde la bandeja, incluso con publicaciones y versiones
-- distintas. La posición de salida coincide con la de entrada. La categoría
-- exige publicación exacta; la política sólo informa si habilita presentar.
CREATE FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
 p_solicitudes jsonb,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb; total integer;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR jsonb_typeof(p_solicitudes) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_solicitudes)>12800
 OR octet_length(p_solicitudes::text)>16777216
 THEN RAISE EXCEPTION 'CC11: lote de etiquetas inválido' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_solicitudes)=0 THEN RETURN '[]'::jsonb; END IF;
 IF EXISTS(
  SELECT 1 FROM jsonb_array_elements(p_solicitudes) AS s(valor)
  WHERE jsonb_typeof(s.valor) IS DISTINCT FROM 'object'
   OR coalesce(s.valor->>'catalogo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
   OR coalesce(s.valor->>'catalogo_sha256','') !~ '^[0-9a-f]{64}$'
   OR coalesce(s.valor->>'categoria_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'politica_catalogo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'politica_catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
   OR coalesce(s.valor->>'politica_catalogo_sha256','') !~ '^[0-9a-f]{64}$'
 ) THEN RAISE EXCEPTION 'CC11: selector de lote inválido' USING ERRCODE='22023'; END IF;

 WITH pedidos AS MATERIALIZED (
  SELECT s.valor, s.orden,
   s.valor->>'catalogo_ref' AS catalogo_ref,
   (s.valor->>'catalogo_version')::integer AS catalogo_version,
   s.valor->>'catalogo_sha256' AS catalogo_sha256,
   s.valor->>'categoria_ref' AS categoria_ref,
   s.valor->>'politica_catalogo_ref' AS politica_catalogo_ref,
   (s.valor->>'politica_catalogo_version')::integer AS politica_catalogo_version,
   s.valor->>'politica_catalogo_sha256' AS politica_catalogo_sha256
  FROM jsonb_array_elements(p_solicitudes) WITH ORDINALITY AS s(valor,orden)
 ), resueltos AS (
  SELECT pedido.orden,pedido.categoria_ref,
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
    categoria.definicion,categoria.etiqueta,p_idioma) AS categoria,
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
    motivo.definicion,motivo.etiqueta,p_idioma) AS motivo_etiqueta_pendiente,
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
    motivo_cumple.definicion,motivo_cumple.etiqueta,p_idioma) AS motivo_etiqueta_cumple,
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
    impedimento_existente.definicion,impedimento_existente.etiqueta,p_idioma)
    AS impedimento_etiqueta_existente,
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(
    impedimento_politica.definicion,impedimento_politica.etiqueta,p_idioma)
    AS impedimento_etiqueta_politica,
   (politica_publicada.catalogo_id IS NOT NULL
    AND jsonb_typeof(politica.definicion->'valor')='boolean'
    AND politica.definicion->'valor'='true'::jsonb
    AND politica.definicion->>'canal'='externa_personal') AS politica_valida
  FROM pedidos pedido
  JOIN vec_catalogos_configurables.publicacion categoria_publicada
   ON categoria_publicada.catalogo_id=pedido.catalogo_ref
   AND categoria_publicada.version=pedido.catalogo_version
   AND categoria_publicada.huella_sha256=pedido.catalogo_sha256
  JOIN vec_catalogos_configurables.entrada_publicada categoria
   ON categoria.catalogo_id=categoria_publicada.catalogo_id
   AND categoria.version=categoria_publicada.version
   AND categoria.huella_sha256=categoria_publicada.huella_sha256
   AND categoria.categoria_id=pedido.categoria_ref
  LEFT JOIN vec_catalogos_configurables.publicacion politica_publicada
   ON politica_publicada.catalogo_id=pedido.politica_catalogo_ref
   AND politica_publicada.version=pedido.politica_catalogo_version
   AND politica_publicada.huella_sha256=pedido.politica_catalogo_sha256
  JOIN vec_catalogos_configurables.entrada_publicada motivo
   ON motivo.catalogo_id=politica_publicada.catalogo_id
   AND motivo.version=politica_publicada.version
   AND motivo.huella_sha256=politica_publicada.huella_sha256
   AND motivo.categoria_id='requisito.pendiente'
  JOIN vec_catalogos_configurables.entrada_publicada motivo_cumple
   ON motivo_cumple.catalogo_id=politica_publicada.catalogo_id
   AND motivo_cumple.version=politica_publicada.version
   AND motivo_cumple.huella_sha256=politica_publicada.huella_sha256
   AND motivo_cumple.categoria_id='requisito.cumple'
  JOIN vec_catalogos_configurables.entrada_publicada impedimento_existente
   ON impedimento_existente.catalogo_id=politica_publicada.catalogo_id
   AND impedimento_existente.version=politica_publicada.version
   AND impedimento_existente.huella_sha256=politica_publicada.huella_sha256
   AND impedimento_existente.categoria_id='solicitud.existente'
  JOIN vec_catalogos_configurables.entrada_publicada impedimento_politica
   ON impedimento_politica.catalogo_id=politica_publicada.catalogo_id
   AND impedimento_politica.version=politica_publicada.version
   AND impedimento_politica.huella_sha256=politica_publicada.huella_sha256
   AND impedimento_politica.categoria_id='presentacion.no.disponible'
  LEFT JOIN vec_catalogos_configurables.entrada_publicada politica
   ON politica.catalogo_id=politica_publicada.catalogo_id
   AND politica.version=politica_publicada.version
   AND politica.huella_sha256=politica_publicada.huella_sha256
   AND politica.categoria_id='presentacion.con.pendientes'
 )
 SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
  'categoria_ref',categoria_ref,'categoria',categoria,
  'motivo_etiqueta_pendiente',motivo_etiqueta_pendiente,
  'motivo_etiqueta_cumple',motivo_etiqueta_cumple,
  'impedimento_etiqueta_existente',impedimento_etiqueta_existente,
  'impedimento_etiqueta_politica',impedimento_etiqueta_politica,
  'politica_valida',coalesce(politica_valida,false)) ORDER BY orden),'[]'::jsonb)
 INTO total,resultado FROM resueltos;
 IF total<>jsonb_array_length(p_solicitudes) THEN
  RAISE EXCEPTION 'CC11: etiqueta o publicación exacta ausente' USING ERRCODE='B9601';
 END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_catalogos_configurables.listar_motivos_inscripcion_v1(p_idioma text)
RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE v_publicacion vec_catalogos_configurables.publicacion%ROWTYPE; motivos jsonb; total integer;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 THEN RAISE EXCEPTION 'CC11: idioma inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO v_publicacion FROM vec_catalogos_configurables.publicacion
  WHERE catalogo_id='bolsa.inscripcion.motivos' ORDER BY version DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CC11: motivos no publicados' USING ERRCODE='B9601'; END IF;
 SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
  'motivo_ref',e.categoria_id,
  'motivo_etiqueta',vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(e.definicion,e.etiqueta,p_idioma))
  ORDER BY e.categoria_id),'[]'::jsonb)
 INTO total,motivos
 FROM vec_catalogos_configurables.entrada_publicada e
 JOIN vec_catalogos_configurables.categoria_control c ON c.categoria_id=e.categoria_id
 WHERE e.catalogo_id=v_publicacion.catalogo_id AND e.version=v_publicacion.version
  AND e.huella_sha256=v_publicacion.huella_sha256
  AND c.catalogo_id=e.catalogo_id AND c.version=e.version
  AND c.huella_sha256=e.huella_sha256 AND c.estado='habilitada';
 IF total<1 OR total>128 THEN RAISE EXCEPTION 'CC11: motivos activos fuera de límite' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_build_object('catalogo_ref',v_publicacion.catalogo_id,
  'catalogo_version',v_publicacion.version,'catalogo_sha256',v_publicacion.huella_sha256,
  'motivos',motivos);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.listar_motivos_inscripcion_v1(text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(
 p_motivo_ref text,p_version integer,p_huella_sha256 text,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE catalogo jsonb; motivo jsonb;
BEGIN
 catalogo:=vec_catalogos_configurables.listar_motivos_inscripcion_v1(p_idioma);
 IF p_version IS DISTINCT FROM (catalogo->>'catalogo_version')::integer
 OR p_huella_sha256 IS DISTINCT FROM catalogo->>'catalogo_sha256'
 THEN RAISE EXCEPTION 'CC11: versión de motivos obsoleta' USING ERRCODE='B9601'; END IF;
 SELECT valor INTO motivo FROM jsonb_array_elements(catalogo->'motivos') AS m(valor)
  WHERE valor->>'motivo_ref'=p_motivo_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'CC11: motivo no publicado' USING ERRCODE='B9605'; END IF;
 RETURN motivo;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(text,integer,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

-- La asociación con un acta interna consume una política publicada exacta.
-- El consumidor compara alcance_ref con la convocatoria de la solicitud antes
-- de cualquier efecto; una referencia genérica no habilita otra convocatoria.
CREATE FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE politica jsonb;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CC11: selector de política de asociación inválido' USING ERRCODE='22023'; END IF;
 SELECT e.definicion INTO politica
 FROM vec_catalogos_configurables.publicacion p
 JOIN vec_catalogos_configurables.entrada_publicada e
  ON e.catalogo_id=p.catalogo_id AND e.version=p.version
  AND e.huella_sha256=p.huella_sha256
 WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version
  AND p.huella_sha256=p_huella_sha256
  AND e.categoria_id='asociacion.manual.acta';
 IF NOT FOUND OR jsonb_typeof(politica->'valor') IS DISTINCT FROM 'boolean'
 OR politica->'valor' IS DISTINCT FROM 'true'::jsonb
 OR politica->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR coalesce(politica->>'motivo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(politica->>'alcance_ref','') !~ '^cv1_[A-Za-z0-9_-]+_v[1-9][0-9]{0,15}$'
 OR octet_length(politica->>'alcance_ref')>200
 THEN RAISE EXCEPTION 'CC11: asociación sin política publicada exacta' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_build_object('permitida',true,
  'politica_ref','asociacion.manual.acta','version',p_version,
  'sha256',p_huella_sha256,'motivo_ref',politica->>'motivo_ref',
  'alcance_ref',politica->>'alcance_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.listar_motivos_inscripcion_v1(text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(text,integer,text,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)
 TO vec_bolsa_llamamientos_propietario;
COMMIT;
