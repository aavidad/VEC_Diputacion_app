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
 OR to_regrole('vec_bolsa_convocatorias_propietario') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.ambito_gestion_entrada_valida_v1(jsonb,text,text,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.ambito_gestion_vigente_v1(jsonb,timestamp with time zone)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(jsonb)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.proyectar_conjunto_gestion_rrhh_v1(jsonb,text,integer,text,timestamp with time zone)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(jsonb,text,text)') IS NOT NULL
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_resumen_inscripcion_lote_v1(jsonb,text)') IS NOT NULL
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
 IF p_definicion ? 'atributos' THEN
  IF jsonb_typeof(p_definicion->'atributos') IS DISTINCT FROM 'object'
  OR p_definicion ? 'etiquetas'
  OR jsonb_typeof(p_definicion->'clave') IS DISTINCT FROM 'string'
  OR jsonb_typeof(p_definicion->'etiqueta') IS DISTINCT FROM 'string'
  OR p_definicion->>'etiqueta' IS DISTINCT FROM p_etiqueta_es
  THEN RAISE EXCEPTION 'CC11: entrada de catálogo incompatible' USING ERRCODE='55000'; END IF;
  IF p_definicion->'atributos' ? 'etiqueta_es' THEN
   IF jsonb_typeof(p_definicion#>'{atributos,etiqueta_es}') IS DISTINCT FROM 'string'
   THEN RAISE EXCEPTION 'CC11: etiqueta ES publicada incompatible' USING ERRCODE='55000'; END IF;
   v_es:=p_definicion#>>'{atributos,etiqueta_es}';
  END IF;
  IF p_definicion->'atributos' ? 'etiqueta_en' THEN
   IF jsonb_typeof(p_definicion#>'{atributos,etiqueta_en}') IS DISTINCT FROM 'string'
   THEN RAISE EXCEPTION 'CC11: etiqueta EN publicada incompatible' USING ERRCODE='55000'; END IF;
   v_en:=p_definicion#>>'{atributos,etiqueta_en}';
  END IF;
 ELSIF p_definicion ? 'etiquetas' THEN
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

-- El rótulo de categoría histórica puede tener hasta 2048 bytes en CC1. La
-- traducción EN opcional nunca inventa texto: si falta o es inválida usa ES.
CREATE FUNCTION vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(
 p_definicion jsonb,p_etiqueta_es text,p_idioma text
) RETURNS text
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp
AS $f$
DECLARE v_en text;
BEGIN
 IF p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR p_etiqueta_es IS NULL OR octet_length(p_etiqueta_es) NOT BETWEEN 1 AND 2048
 OR jsonb_typeof(p_definicion) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'CC11: categoría histórica no disponible' USING ERRCODE='B9601'; END IF;
 IF p_definicion ? 'atributos' THEN
  IF jsonb_typeof(p_definicion->'atributos') IS DISTINCT FROM 'object'
  OR p_definicion ? 'etiquetas'
  OR jsonb_typeof(p_definicion->'clave') IS DISTINCT FROM 'string'
  OR p_definicion->>'etiqueta' IS DISTINCT FROM p_etiqueta_es
  THEN RAISE EXCEPTION 'CC11: categoría histórica publicada incompatible' USING ERRCODE='B9601'; END IF;
  IF jsonb_typeof(p_definicion#>'{atributos,etiqueta_en}')='string' THEN
   v_en:=p_definicion#>>'{atributos,etiqueta_en}'; END IF;
 ELSIF jsonb_typeof(p_definicion#>'{etiquetas,en}')='string' THEN
  v_en:=p_definicion#>>'{etiquetas,en}';
 END IF;
 IF p_idioma='en' AND octet_length(v_en) BETWEEN 1 AND 2048 THEN RETURN v_en; END IF;
 RETURN p_etiqueta_es;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(jsonb,text,text)
 FROM PUBLIC;

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
 BEGIN
 SELECT count(*),jsonb_agg(jsonb_build_object(
  'categoria_ref',r.ref,
  'categoria',CASE WHEN p_catalogo_id='bolsa.inscripcion.motivos' THEN
   vec_catalogos_configurables.etiqueta_inscripcion_idioma_v1(e.definicion,e.etiqueta,p_idioma)
   ELSE vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(
    e.definicion,e.etiqueta,p_idioma) END)
  ORDER BY r.orden)
 INTO total,resultado
 FROM unnest(p_referencias) WITH ORDINALITY AS r(ref,orden)
 JOIN vec_catalogos_configurables.entrada_publicada e
   ON e.catalogo_id=p_catalogo_id AND e.version=p_version
  AND e.huella_sha256=p_huella_sha256 AND e.categoria_id=r.ref;
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '55000' OR SQLSTATE '54000' THEN
  RAISE EXCEPTION 'CC11: etiqueta publicada incompatible' USING ERRCODE='B9601';
 END;
 IF total<>cardinality(p_referencias) THEN
  RAISE EXCEPTION 'CC11: categoría no publicada en versión exacta' USING ERRCODE='B9601';
 END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text,p_canal text
) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE politica jsonb; etiqueta text; v_clave text;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR p_canal IS NULL OR p_canal NOT IN('externa_personal','interna_corporativa')
 THEN RAISE EXCEPTION 'CC11: selector de política inválido' USING ERRCODE='22023'; END IF;
 v_clave:=CASE WHEN p_canal='externa_personal' THEN 'presentacion.con.pendientes'
  ELSE 'presentacion.con.pendientes.empleado' END;
 SELECT e.definicion,e.etiqueta INTO politica,etiqueta
 FROM vec_catalogos_configurables.publicacion p
 JOIN vec_catalogos_configurables.entrada_publicada e
  ON e.catalogo_id=p.catalogo_id AND e.version=p.version
  AND e.huella_sha256=p.huella_sha256
 WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version
  AND p.huella_sha256=p_huella_sha256
  AND e.categoria_id=v_clave;
 IF NOT FOUND OR NOT coalesce((
  (politica ? 'atributos'
   AND jsonb_typeof(politica->'atributos')='object'
   AND politica->>'clave'=v_clave
   AND politica->>'etiqueta'=etiqueta
   AND jsonb_typeof(politica#>'{atributos,valor}')='string'
   AND politica#>>'{atributos,valor}'='true'
   AND politica#>>'{atributos,canal}'=p_canal)
  OR (NOT (politica ? 'atributos')
   AND jsonb_typeof(politica->'valor')='boolean'
   AND politica->'valor'='true'::jsonb
   AND politica->>'canal'=p_canal)),false)
 THEN RAISE EXCEPTION 'CC11: presentación pendiente sin política publicada' USING ERRCODE='B9601'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

-- Una sola llamada desde la bandeja, incluso con publicaciones y versiones
-- distintas. La posición de salida coincide con la de entrada. La categoría
-- exige publicación exacta; la política sólo informa si habilita presentar.
CREATE FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
 p_solicitudes jsonb,p_idioma text,p_canal text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb; total integer;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR p_canal IS NULL OR p_canal NOT IN('externa_personal','interna_corporativa')
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

 BEGIN
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
   vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(
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
   (politica_publicada.catalogo_id IS NOT NULL AND (
    (politica.definicion ? 'atributos'
     AND jsonb_typeof(politica.definicion->'atributos')='object'
     AND politica.definicion->>'clave'=politica.categoria_id
     AND politica.definicion->>'etiqueta'=politica.etiqueta
     AND jsonb_typeof(politica.definicion#>'{atributos,valor}')='string'
     AND politica.definicion#>>'{atributos,valor}'='true'
     AND politica.definicion#>>'{atributos,canal}'=p_canal)
    OR (NOT (politica.definicion ? 'atributos')
     AND jsonb_typeof(politica.definicion->'valor')='boolean'
     AND politica.definicion->'valor'='true'::jsonb
     AND politica.definicion->>'canal'=p_canal))) AS politica_valida
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
   AND politica.categoria_id=CASE WHEN p_canal='externa_personal'
    THEN 'presentacion.con.pendientes' ELSE 'presentacion.con.pendientes.empleado' END
 )
 SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
  'categoria_ref',categoria_ref,'categoria',categoria,
  'motivo_etiqueta_pendiente',motivo_etiqueta_pendiente,
  'motivo_etiqueta_cumple',motivo_etiqueta_cumple,
  'impedimento_etiqueta_existente',impedimento_etiqueta_existente,
  'impedimento_etiqueta_politica',impedimento_etiqueta_politica,
  'politica_valida',coalesce(politica_valida,false)) ORDER BY orden),'[]'::jsonb)
 INTO total,resultado FROM resueltos;
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '55000' OR SQLSTATE '54000' THEN
  RAISE EXCEPTION 'CC11: etiqueta publicada incompatible' USING ERRCODE='B9601';
 END;
 IF total<>jsonb_array_length(p_solicitudes) THEN
  RAISE EXCEPTION 'CC11: etiqueta o publicación exacta ausente' USING ERRCODE='B9601';
 END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

-- Comprueba todos los códigos de cada convocatoria contra la publicación
-- exacta en una sola llamada. La falta de un código informa false; no oculta
-- la convocatoria ni sustituye la etiqueta publicada del código de muestra.
CREATE FUNCTION vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
 p_solicitudes jsonb,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR jsonb_typeof(p_solicitudes) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_solicitudes)>100
 OR octet_length(p_solicitudes::text)>16777216
 THEN RAISE EXCEPTION 'CC11: lote de categorías inválido' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_solicitudes)=0 THEN RETURN '[]'::jsonb; END IF;
 IF EXISTS(
  SELECT 1 FROM jsonb_array_elements(p_solicitudes) AS s(valor)
  WHERE jsonb_typeof(s.valor) IS DISTINCT FROM 'object'
   OR coalesce(s.valor->>'catalogo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
   OR coalesce(s.valor->>'catalogo_sha256','') !~ '^[0-9a-f]{64}$'
   OR jsonb_typeof(s.valor->'categorias_refs') IS DISTINCT FROM 'array'
   OR jsonb_array_length(s.valor->'categorias_refs') NOT BETWEEN 1 AND 128
   OR EXISTS(SELECT 1 FROM jsonb_array_elements(s.valor->'categorias_refs') AS r(valor)
     WHERE jsonb_typeof(r.valor) IS DISTINCT FROM 'string'
      OR r.valor#>>'{}' !~ '^[a-z][a-z0-9_.:-]{2,127}$')
   OR (SELECT count(DISTINCT r.valor COLLATE "C")
       FROM jsonb_array_elements_text(s.valor->'categorias_refs') AS r(valor))
      <>jsonb_array_length(s.valor->'categorias_refs')
 ) THEN RAISE EXCEPTION 'CC11: selector de categorías inválido' USING ERRCODE='22023'; END IF;

 WITH pedidos AS MATERIALIZED (
  SELECT s.orden,s.valor->>'catalogo_ref' AS catalogo_ref,
   (s.valor->>'catalogo_version')::integer AS catalogo_version,
   s.valor->>'catalogo_sha256' AS catalogo_sha256,
   s.valor->'categorias_refs' AS categorias_refs
  FROM jsonb_array_elements(p_solicitudes) WITH ORDINALITY AS s(valor,orden)
 ), pedidos_unicos AS MATERIALIZED (
  SELECT row_number() OVER () AS grupo,distintos.* FROM (
   SELECT DISTINCT catalogo_ref,catalogo_version,catalogo_sha256,categorias_refs
   FROM pedidos
  ) AS distintos
 ), refs AS MATERIALIZED (
  SELECT unico.grupo,ref.valor AS categoria_ref
  FROM pedidos_unicos unico CROSS JOIN LATERAL
   jsonb_array_elements_text(unico.categorias_refs) AS ref(valor)
 ), contadas AS (
  SELECT unico.grupo,jsonb_array_length(unico.categorias_refs) AS numero_categorias,
   count(entrada.categoria_id) AS publicadas
  FROM pedidos_unicos unico
  JOIN refs ON refs.grupo=unico.grupo
  LEFT JOIN vec_catalogos_configurables.publicacion publicacion
   ON publicacion.catalogo_id=unico.catalogo_ref
   AND publicacion.version=unico.catalogo_version
   AND publicacion.huella_sha256=unico.catalogo_sha256
  LEFT JOIN vec_catalogos_configurables.entrada_publicada entrada
   ON entrada.catalogo_id=publicacion.catalogo_id
   AND entrada.version=publicacion.version
   AND entrada.huella_sha256=publicacion.huella_sha256
   AND entrada.categoria_id=refs.categoria_ref
  GROUP BY unico.grupo,unico.categorias_refs
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'catalogo_completo',contadas.publicadas=contadas.numero_categorias,
  'numero_categorias',contadas.numero_categorias) ORDER BY pedido.orden),'[]'::jsonb)
 INTO resultado FROM pedidos pedido
 JOIN pedidos_unicos unico
  ON unico.catalogo_ref=pedido.catalogo_ref
  AND unico.catalogo_version=pedido.catalogo_version
  AND unico.catalogo_sha256=pedido.catalogo_sha256
  AND unico.categorias_refs=pedido.categorias_refs
 JOIN contadas ON contadas.grupo=unico.grupo;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)
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
DECLARE politica jsonb; etiqueta text; v_motivo text; v_alcance text;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CC11: selector de política de asociación inválido' USING ERRCODE='22023'; END IF;
 SELECT e.definicion,e.etiqueta INTO politica,etiqueta
 FROM vec_catalogos_configurables.publicacion p
 JOIN vec_catalogos_configurables.entrada_publicada e
  ON e.catalogo_id=p.catalogo_id AND e.version=p.version
  AND e.huella_sha256=p.huella_sha256
 WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version
  AND p.huella_sha256=p_huella_sha256
  AND e.categoria_id='asociacion.manual.acta';
 IF NOT FOUND THEN
  RAISE EXCEPTION 'CC11: asociación sin política publicada exacta' USING ERRCODE='B9601'; END IF;
 IF politica ? 'atributos' THEN
  IF jsonb_typeof(politica->'atributos') IS DISTINCT FROM 'object'
   OR politica->>'clave' IS DISTINCT FROM 'asociacion.manual.acta'
   OR politica->>'etiqueta' IS DISTINCT FROM etiqueta
   OR jsonb_typeof(politica#>'{atributos,valor}') IS DISTINCT FROM 'string'
   OR politica#>>'{atributos,valor}' IS DISTINCT FROM 'true'
   OR politica#>>'{atributos,canal}' IS DISTINCT FROM 'interna_corporativa'
   OR jsonb_typeof(politica#>'{atributos,motivo_ref}') IS DISTINCT FROM 'string'
   OR jsonb_typeof(politica#>'{atributos,alcance_ref}') IS DISTINCT FROM 'string'
  THEN RAISE EXCEPTION 'CC11: asociación sin política publicada exacta' USING ERRCODE='B9601'; END IF;
  v_motivo:=politica#>>'{atributos,motivo_ref}';
  v_alcance:=politica#>>'{atributos,alcance_ref}';
 ELSE
  IF jsonb_typeof(politica->'valor') IS DISTINCT FROM 'boolean'
   OR politica->'valor' IS DISTINCT FROM 'true'::jsonb
   OR politica->>'canal' IS DISTINCT FROM 'interna_corporativa'
  THEN RAISE EXCEPTION 'CC11: asociación sin política publicada exacta' USING ERRCODE='B9601'; END IF;
  v_motivo:=politica->>'motivo_ref';
  v_alcance:=politica->>'alcance_ref';
 END IF;
 IF coalesce(v_motivo,'') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(v_alcance,'') !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
 OR octet_length(v_alcance)>200
 THEN RAISE EXCEPTION 'CC11: asociación sin política publicada exacta' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_build_object('permitida',true,
  'politica_ref','asociacion.manual.acta','version',p_version,
  'sha256',p_huella_sha256,'motivo_ref',v_motivo,
  'alcance_ref',v_alcance);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;

-- La competencia de gestión procede de una entrada CC1 gobernada y activa,
-- vinculada a la publicación exacta de esta convocatoria. Nunca se deduce
-- de un nombre de cargo, una unidad enviada por cliente o un perfil genérico.
CREATE FUNCTION vec_catalogos_configurables.ambito_gestion_entrada_valida_v1(
 p_definicion jsonb,p_etiqueta text,p_clave text,p_convocatoria_ref text
) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog,pg_temp
AS $f$
DECLARE atributos jsonb;
BEGIN
 IF jsonb_typeof(p_definicion) IS DISTINCT FROM 'object'
 OR p_definicion->>'clave' IS DISTINCT FROM p_clave
 OR p_definicion->>'etiqueta' IS DISTINCT FROM p_etiqueta
 OR jsonb_typeof(p_definicion->'orden') IS DISTINCT FROM 'number'
 OR jsonb_typeof(p_definicion->'vigente_desde') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_definicion->'atributos') IS DISTINCT FROM 'object'
 THEN RETURN false; END IF;
 atributos:=p_definicion->'atributos';
 IF (SELECT count(*) FROM jsonb_object_keys(atributos))<>4
 OR jsonb_typeof(atributos->'convocatoria_ref') IS DISTINCT FROM 'string'
 OR atributos->>'convocatoria_ref' IS DISTINCT FROM p_convocatoria_ref
 OR jsonb_typeof(atributos->'unidad_ref') IS DISTINCT FROM 'string'
 OR coalesce(atributos->>'unidad_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR octet_length(atributos->>'unidad_ref')>128
 OR jsonb_typeof(atributos->'ambito_ref') IS DISTINCT FROM 'string'
 OR coalesce(atributos->>'ambito_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR octet_length(atributos->>'ambito_ref')>128
 OR jsonb_typeof(atributos->'canal') IS DISTINCT FROM 'string'
 OR atributos->>'canal' IS DISTINCT FROM 'interna_corporativa'
 THEN RETURN false; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.ambito_gestion_entrada_valida_v1(jsonb,text,text,text)
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_catalogos_configurables.ambito_gestion_vigente_v1(
 p_definicion jsonb,p_instante timestamptz
) RETURNS boolean
LANGUAGE plpgsql STABLE STRICT PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET "TimeZone"='UTC'
AS $f$
DECLARE v_desde timestamptz; v_hasta timestamptz;
BEGIN
 IF jsonb_typeof(p_definicion->'vigente_desde') IS DISTINCT FROM 'string'
 OR (p_definicion ? 'vigente_hasta'
  AND jsonb_typeof(p_definicion->'vigente_hasta') IS DISTINCT FROM 'string')
 THEN RETURN false; END IF;
 BEGIN
  v_desde:=(p_definicion->>'vigente_desde')::timestamptz;
  IF p_definicion ? 'vigente_hasta' THEN
   v_hasta:=(p_definicion->>'vigente_hasta')::timestamptz; END IF;
 EXCEPTION WHEN data_exception THEN RETURN false; END;
 RETURN v_desde IS NOT NULL AND isfinite(v_desde) AND v_desde<=p_instante
  AND (v_hasta IS NULL OR (isfinite(v_hasta) AND p_instante<v_hasta));
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.ambito_gestion_vigente_v1(jsonb,timestamptz)
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
 p_catalogo_id text,p_version integer,p_huella_sha256 text,p_convocatoria_ref text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE partes text[]; v_secuencia bigint; v_clave text; v_definicion jsonb; v_etiqueta text;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_version IS NULL OR p_version<1
 OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 71 AND 200
 THEN RAISE EXCEPTION 'CC11: selector de ámbito de gestión inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN
  RAISE EXCEPTION 'CC11: convocatoria de gestión no canónica' USING ERRCODE='22023'; END IF;
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991 THEN
  RAISE EXCEPTION 'CC11: secuencia de gestión inválida' USING ERRCODE='22023'; END IF;
 v_clave:='ambito.gestion.'||partes[1]||'.v'||v_secuencia::text;
 SELECT entrada.definicion,entrada.etiqueta INTO v_definicion,v_etiqueta
 FROM vec_catalogos_configurables.publicacion publicacion
 JOIN vec_catalogos_configurables.entrada_publicada entrada
  ON entrada.catalogo_id=publicacion.catalogo_id
  AND entrada.version=publicacion.version
  AND entrada.huella_sha256=publicacion.huella_sha256
 JOIN vec_catalogos_configurables.categoria_control control
  ON control.categoria_id=entrada.categoria_id
  AND control.catalogo_id=entrada.catalogo_id
  AND control.version=entrada.version
  AND control.huella_sha256=entrada.huella_sha256
  AND control.estado='habilitada'
 WHERE publicacion.catalogo_id=p_catalogo_id AND publicacion.version=p_version
  AND publicacion.huella_sha256=p_huella_sha256
  AND entrada.categoria_id=v_clave;
 IF NOT FOUND OR NOT coalesce(
  vec_catalogos_configurables.ambito_gestion_entrada_valida_v1(
   v_definicion,v_etiqueta,v_clave,p_convocatoria_ref),false)
 OR NOT coalesce(vec_catalogos_configurables.ambito_gestion_vigente_v1(
  v_definicion,statement_timestamp()),false)
 THEN RAISE EXCEPTION 'CC11: ámbito de gestión publicado no disponible' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_build_object('unidad_ref',v_definicion#>>'{atributos,unidad_ref}',
  'ambito_ref',v_definicion#>>'{atributos,ambito_ref}','fuente_ref',v_clave,
  'fuente_version',p_version,'fuente_sha256',p_huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor;

-- La lectura pública sólo recibe un booleano por convocatoria. Los ámbitos
-- y unidades permanecen en el contrato privado de presentación de BC9.
CREATE FUNCTION vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
 p_solicitudes jsonb
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR jsonb_typeof(p_solicitudes) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_solicitudes)>100
 OR octet_length(p_solicitudes::text)>1048576
 THEN RAISE EXCEPTION 'CC11: lote de ámbitos inválido' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_solicitudes)=0 THEN RETURN '[]'::jsonb; END IF;
 IF EXISTS(
  SELECT 1 FROM jsonb_array_elements(p_solicitudes) AS s(valor)
  WHERE jsonb_typeof(s.valor) IS DISTINCT FROM 'object'
   OR coalesce(s.valor->>'politica_catalogo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'politica_catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
   OR coalesce(s.valor->>'politica_catalogo_sha256','') !~ '^[0-9a-f]{64}$'
   OR coalesce(s.valor->>'convocatoria_ref','') !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
   OR CASE WHEN coalesce(s.valor->>'convocatoria_ref','') ~
      '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
     THEN (split_part(s.valor->>'convocatoria_ref','_v',2))::bigint>9007199254740991
     ELSE true END
 ) THEN RAISE EXCEPTION 'CC11: selector de ámbitos inválido' USING ERRCODE='22023'; END IF;
 WITH pedidos AS MATERIALIZED (
  SELECT s.orden,s.valor->>'politica_catalogo_ref' AS catalogo_ref,
   (s.valor->>'politica_catalogo_version')::integer AS catalogo_version,
   s.valor->>'politica_catalogo_sha256' AS catalogo_sha256,
   s.valor->>'convocatoria_ref' AS convocatoria_ref,
   'ambito.gestion.'||substring(s.valor->>'convocatoria_ref'
    from '^cv1_([0-9a-f]{64})_v[1-9][0-9]{0,15}$')||'.v'||
    split_part(s.valor->>'convocatoria_ref','_v',2) AS clave
  FROM jsonb_array_elements(p_solicitudes) WITH ORDINALITY AS s(valor,orden)
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'ambito_gestion_disponible',coalesce(control.categoria_id IS NOT NULL AND
   vec_catalogos_configurables.ambito_gestion_entrada_valida_v1(
    entrada.definicion,entrada.etiqueta,pedido.clave,pedido.convocatoria_ref)
   AND vec_catalogos_configurables.ambito_gestion_vigente_v1(
    entrada.definicion,statement_timestamp()),false))
   ORDER BY pedido.orden),'[]'::jsonb)
 INTO resultado
 FROM pedidos pedido
 LEFT JOIN vec_catalogos_configurables.publicacion publicacion
  ON publicacion.catalogo_id=pedido.catalogo_ref
  AND publicacion.version=pedido.catalogo_version
  AND publicacion.huella_sha256=pedido.catalogo_sha256
 LEFT JOIN vec_catalogos_configurables.entrada_publicada entrada
  ON entrada.catalogo_id=publicacion.catalogo_id
  AND entrada.version=publicacion.version
  AND entrada.huella_sha256=publicacion.huella_sha256
  AND entrada.categoria_id=pedido.clave
 LEFT JOIN vec_catalogos_configurables.categoria_control control
  ON control.categoria_id=entrada.categoria_id
  AND control.catalogo_id=entrada.catalogo_id
  AND control.version=entrada.version
  AND control.huella_sha256=entrada.huella_sha256
  AND control.estado='habilitada';
 IF jsonb_array_length(resultado)<>jsonb_array_length(p_solicitudes) THEN
  RAISE EXCEPTION 'CC11: lote de ámbitos desalineado' USING ERRCODE='55000'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(jsonb)
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor;

-- El puntero CC1 actual fija la única fuente del conjunto de gestión RRHH.
-- Este contrato privado aporta ámbito al PDP; no concede por sí solo acceso.
CREATE FUNCTION vec_catalogos_configurables.proyectar_conjunto_gestion_rrhh_v1(
 p_definicion jsonb,p_etiqueta text,p_version integer,p_sha256 text,p_instante timestamptz
) RETURNS jsonb
LANGUAGE plpgsql STABLE PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET "TimeZone"='UTC'
AS $f$
DECLARE atributos jsonb; v_desde timestamptz; v_hasta timestamptz;
BEGIN
 IF jsonb_typeof(p_definicion) IS DISTINCT FROM 'object'
 OR p_definicion->>'clave' IS DISTINCT FROM 'inscripcion.gestion.rrhh.conjunto'
 OR p_definicion->>'etiqueta' IS DISTINCT FROM p_etiqueta
 OR jsonb_typeof(p_definicion->'orden') IS DISTINCT FROM 'number'
 OR jsonb_typeof(p_definicion->'vigente_desde') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_definicion->'atributos') IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'CC11: conjunto de gestión RRHH no publicado' USING ERRCODE='B9601'; END IF;
 atributos:=p_definicion->'atributos';
 IF (SELECT count(*) FROM jsonb_object_keys(atributos))<>4
 OR jsonb_typeof(atributos->'conjunto_ref') IS DISTINCT FROM 'string'
 OR coalesce(atributos->>'conjunto_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR octet_length(atributos->>'conjunto_ref')>128
 OR jsonb_typeof(atributos->'unidad_ref') IS DISTINCT FROM 'string'
 OR coalesce(atributos->>'unidad_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR octet_length(atributos->>'unidad_ref')>128
 OR jsonb_typeof(atributos->'ambito_ref') IS DISTINCT FROM 'string'
 OR coalesce(atributos->>'ambito_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR octet_length(atributos->>'ambito_ref')>128
 OR jsonb_typeof(atributos->'canal') IS DISTINCT FROM 'string'
 OR atributos->>'canal' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'CC11: conjunto de gestión RRHH incompatible' USING ERRCODE='B9601'; END IF;
 BEGIN
  v_desde:=(p_definicion->>'vigente_desde')::timestamptz;
  IF p_definicion ? 'vigente_hasta' THEN
   IF jsonb_typeof(p_definicion->'vigente_hasta') IS DISTINCT FROM 'string'
   THEN RAISE EXCEPTION 'CC11: fin de vigencia incompatible' USING ERRCODE='B9601'; END IF;
   v_hasta:=(p_definicion->>'vigente_hasta')::timestamptz;
  END IF;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'CC11: vigencia de conjunto ilegible' USING ERRCODE='B9601'; END;
 IF p_instante IS NULL OR NOT isfinite(p_instante)
 OR v_desde IS NULL OR NOT isfinite(v_desde) OR v_desde>p_instante
 OR (v_hasta IS NOT NULL AND (NOT isfinite(v_hasta) OR p_instante>=v_hasta))
 THEN RAISE EXCEPTION 'CC11: conjunto de gestión RRHH no vigente' USING ERRCODE='B9601'; END IF;
 RETURN jsonb_build_object('conjunto_ref',atributos->>'conjunto_ref',
  'unidad_ref',atributos->>'unidad_ref','ambito_ref',atributos->>'ambito_ref',
  'fuente_ref','inscripcion.gestion.rrhh.conjunto',
  'fuente_version',p_version,'fuente_sha256',p_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.proyectar_conjunto_gestion_rrhh_v1(jsonb,text,integer,text,timestamptz)
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()
RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE v record;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 THEN RAISE EXCEPTION 'CC11: lector de conjunto no propietario' USING ERRCODE='42501'; END IF;
 SELECT control.catalogo_id,control.version,control.huella_sha256,
  entrada.etiqueta,entrada.definicion INTO v
 FROM vec_catalogos_configurables.categoria_control control
 JOIN vec_catalogos_configurables.publicacion publicacion
  ON publicacion.catalogo_id=control.catalogo_id
  AND publicacion.version=control.version
  AND publicacion.huella_sha256=control.huella_sha256
 JOIN vec_catalogos_configurables.entrada_publicada entrada
  ON entrada.catalogo_id=publicacion.catalogo_id
  AND entrada.version=publicacion.version
  AND entrada.huella_sha256=publicacion.huella_sha256
  AND entrada.categoria_id=control.categoria_id
 WHERE control.categoria_id='inscripcion.gestion.rrhh.conjunto'
  AND control.estado='habilitada';
 IF NOT FOUND THEN
  RAISE EXCEPTION 'CC11: conjunto de gestión RRHH no publicado' USING ERRCODE='B9601'; END IF;
 RETURN vec_catalogos_configurables.proyectar_conjunto_gestion_rrhh_v1(
  v.definicion,v.etiqueta,v.version,v.huella_sha256,statement_timestamp());
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor;

-- Recheck del efecto B96: el lock se conserva hasta el commit. El reloj se
-- toma después de adquirirlo, de modo que la caducidad durante la espera
-- deniega y una publicación concurrente del puntero no pasa inadvertida.
CREATE FUNCTION vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE v record; v_instante timestamptz(6);
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CC11: cotejo de conjunto requiere efecto serializable' USING ERRCODE='55000'; END IF;
 SELECT control.catalogo_id,control.version,control.huella_sha256,
  entrada.etiqueta,entrada.definicion INTO v
 FROM vec_catalogos_configurables.categoria_control control
 JOIN vec_catalogos_configurables.publicacion publicacion
  ON publicacion.catalogo_id=control.catalogo_id
  AND publicacion.version=control.version
  AND publicacion.huella_sha256=control.huella_sha256
 JOIN vec_catalogos_configurables.entrada_publicada entrada
  ON entrada.catalogo_id=publicacion.catalogo_id
  AND entrada.version=publicacion.version
  AND entrada.huella_sha256=publicacion.huella_sha256
  AND entrada.categoria_id=control.categoria_id
 WHERE control.categoria_id='inscripcion.gestion.rrhh.conjunto'
  AND control.estado='habilitada'
 FOR SHARE OF control;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'CC11: conjunto de gestión RRHH no publicado' USING ERRCODE='B9601'; END IF;
 v_instante:=date_trunc('microseconds',clock_timestamp());
 RETURN vec_catalogos_configurables.proyectar_conjunto_gestion_rrhh_v1(
  v.definicion,v.etiqueta,v.version,v.huella_sha256,v_instante);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()
 FROM PUBLIC,vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor;

-- Sólo para fichas históricas RRHH. Admite la etiqueta ES publicada de CC1
-- hasta 2048 bytes; EN mal tipada, larga o ausente vuelve al dato ES.
-- Nunca usa el límite de 200 bytes del formulario de inscripción activo.
CREATE FUNCTION vec_catalogos_configurables.leer_etiquetas_resumen_inscripcion_lote_v1(
 p_selectores jsonb,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE resultado jsonb; total integer; etiquetadas integer;
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR jsonb_typeof(p_selectores) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_selectores)>100
 OR octet_length(p_selectores::text)>1048576
 THEN RAISE EXCEPTION 'CC11: selectores de resumen inválidos' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_selectores)=0 THEN RETURN '[]'::jsonb; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_selectores) AS s(valor)
  WHERE jsonb_typeof(s.valor) IS DISTINCT FROM 'object'
   OR coalesce(s.valor->>'catalogo_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
   OR coalesce(s.valor->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
   OR coalesce(s.valor->>'catalogo_sha256','') !~ '^[0-9a-f]{64}$'
   OR coalesce(s.valor->>'categoria_ref','') !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 THEN RAISE EXCEPTION 'CC11: selector de resumen inválido' USING ERRCODE='22023'; END IF;
 WITH pedidos AS MATERIALIZED (
  SELECT s.orden,s.valor->>'catalogo_ref' AS catalogo_ref,
   (s.valor->>'catalogo_version')::integer AS catalogo_version,
   s.valor->>'catalogo_sha256' AS catalogo_sha256,
   s.valor->>'categoria_ref' AS categoria_ref
  FROM jsonb_array_elements(p_selectores) WITH ORDINALITY AS s(valor,orden)
 ), resueltos AS (
  SELECT pedido.orden,pedido.categoria_ref,
   vec_catalogos_configurables.etiqueta_categoria_historica_inscripcion_v1(
    entrada.definicion,entrada.etiqueta,p_idioma) AS categoria
  FROM pedidos pedido
  JOIN vec_catalogos_configurables.publicacion publicacion
   ON publicacion.catalogo_id=pedido.catalogo_ref
   AND publicacion.version=pedido.catalogo_version
   AND publicacion.huella_sha256=pedido.catalogo_sha256
  JOIN vec_catalogos_configurables.entrada_publicada entrada
   ON entrada.catalogo_id=publicacion.catalogo_id
   AND entrada.version=publicacion.version
   AND entrada.huella_sha256=publicacion.huella_sha256
   AND entrada.categoria_id=pedido.categoria_ref
  WHERE jsonb_typeof(entrada.definicion)='object'
   AND (NOT (entrada.definicion ? 'atributos') OR
    (jsonb_typeof(entrada.definicion->'atributos')='object'
     AND entrada.definicion->>'clave'=entrada.categoria_id
     AND entrada.definicion->>'etiqueta'=entrada.etiqueta))
 )
 SELECT count(*),count(categoria),coalesce(jsonb_agg(jsonb_build_object(
  'categoria_ref',categoria_ref,'categoria',categoria) ORDER BY orden),'[]'::jsonb)
 INTO total,etiquetadas,resultado FROM resueltos;
 IF total<>jsonb_array_length(p_selectores) OR etiquetadas<>total
 THEN RAISE EXCEPTION 'CC11: resumen histórico publicado no disponible' USING ERRCODE='B9601'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_etiquetas_resumen_inscripcion_lote_v1(jsonb,text)
 FROM PUBLIC,vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor;

GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(jsonb,text,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.listar_motivos_inscripcion_v1(text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(text,integer,text,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(jsonb)
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_etiquetas_resumen_inscripcion_lote_v1(jsonb,text)
 TO vec_bolsa_convocatorias_propietario;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_bolsa_convocatorias_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)
 TO vec_bolsa_convocatorias_propietario;
COMMIT;
