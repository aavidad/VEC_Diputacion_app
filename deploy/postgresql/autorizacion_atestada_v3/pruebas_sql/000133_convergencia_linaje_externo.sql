\set ON_ERROR_STOP on
-- Sonda sobre un clon sintético post-AD118/AUT21/AD134/AD136. El guion inserta AD133.
-- No confirma efectos ni instala migraciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='60s';
SET LOCAL lock_timeout='5s';
CREATE TEMP TABLE ad133_funciones ON COMMIT DROP AS
 SELECT p.oid,pg_catalog.to_jsonb(p) AS imagen
 FROM pg_catalog.pg_proc p
 WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3');
CREATE TEMP TABLE ad133_dependencias ON COMMIT DROP AS
 SELECT pg_catalog.to_jsonb(d) AS imagen FROM pg_catalog.pg_depend d
 WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones)
 UNION ALL
 SELECT pg_catalog.to_jsonb(d) FROM pg_catalog.pg_shdepend d
 WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones);
CREATE TEMP TABLE ad133_historia(tabla pg_catalog.text PRIMARY KEY,filas pg_catalog.int8,huella pg_catalog.text) ON COMMIT DROP;
DO $historia$
DECLARE t pg_catalog.record; filas pg_catalog.int8; huella pg_catalog.text;
BEGIN
 FOR t IN SELECT n.nspname,c.relname FROM pg_catalog.pg_class c
 JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE c.relkind='r' AND n.nspname LIKE 'vec\_%' ESCAPE '\' ORDER BY n.nspname,c.relname LOOP
  EXECUTE pg_catalog.format('SELECT count(*),encode(sha256(convert_to(coalesce(string_agg(encode(sha256(convert_to(to_jsonb(r)::text,''UTF8'')),''hex''),'''' ORDER BY encode(sha256(convert_to(to_jsonb(r)::text,''UTF8'')),''hex'')),''''),''UTF8'')),''hex'') FROM %I.%I r',t.nspname,t.relname) INTO filas,huella;
  INSERT INTO pg_temp.ad133_historia VALUES(t.nspname||'.'||t.relname,filas,huella);
 END LOOP;
END $historia$;
-- AD133-CORRECTIVA-AQUI
DO $conservacion$
DECLARE cambiado pg_catalog.int4; t pg_catalog.record; filas pg_catalog.int8; huella pg_catalog.text;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_temp.ad133_funciones f LEFT JOIN pg_catalog.pg_proc p ON p.oid=f.oid
   WHERE p.oid IS NULL OR pg_catalog.to_jsonb(p)-'proconfig' IS DISTINCT FROM f.imagen-'proconfig')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
   AND NOT EXISTS(SELECT 1 FROM pg_temp.ad133_funciones f WHERE f.oid=p.oid)) THEN
  RAISE EXCEPTION 'AD133 sonda: cuerpos, OID, propietario o ACL alterados';
 END IF;
 SELECT count(*) INTO cambiado FROM pg_temp.ad133_funciones f JOIN pg_catalog.pg_proc p ON p.oid=f.oid
 WHERE pg_catalog.to_jsonb(p.proconfig) IS DISTINCT FROM f.imagen->'proconfig';
 IF cambiado<>17 OR EXISTS(SELECT 1 FROM pg_temp.ad133_funciones f JOIN pg_catalog.pg_proc p ON p.oid=f.oid
 WHERE pg_catalog.to_jsonb(p.proconfig) IS DISTINCT FROM f.imagen->'proconfig'
 AND pg_catalog.to_jsonb(p.proconfig) IS DISTINCT FROM
 (SELECT pg_catalog.jsonb_agg(CASE WHEN opcion='search_path=pg_catalog' THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
 FROM pg_catalog.jsonb_array_elements_text(f.imagen->'proconfig') WITH ORDINALITY x(opcion,indice))) THEN
  RAISE EXCEPTION 'AD133 sonda: cierre distinto del inventario de 17 firmas';
 END IF;
 IF EXISTS(WITH actual AS(
 SELECT pg_catalog.to_jsonb(d) imagen FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones)
 UNION ALL SELECT pg_catalog.to_jsonb(d) FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones))
 SELECT imagen FROM actual EXCEPT ALL SELECT imagen FROM pg_temp.ad133_dependencias)
 OR EXISTS(WITH actual AS(
 SELECT pg_catalog.to_jsonb(d) imagen FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones)
 UNION ALL SELECT pg_catalog.to_jsonb(d) FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid IN(SELECT oid FROM pg_temp.ad133_funciones))
 SELECT imagen FROM pg_temp.ad133_dependencias EXCEPT ALL SELECT imagen FROM actual) THEN
  RAISE EXCEPTION 'AD133 sonda: dependencias alteradas';
 END IF;
 FOR t IN SELECT * FROM pg_temp.ad133_historia ORDER BY tabla LOOP
  EXECUTE pg_catalog.format('SELECT count(*),encode(sha256(convert_to(coalesce(string_agg(encode(sha256(convert_to(to_jsonb(r)::text,''UTF8'')),''hex''),'''' ORDER BY encode(sha256(convert_to(to_jsonb(r)::text,''UTF8'')),''hex'')),''''),''UTF8'')),''hex'') FROM %s r',t.tabla) INTO filas,huella;
  IF filas IS DISTINCT FROM t.filas OR huella IS DISTINCT FROM t.huella THEN RAISE EXCEPTION 'AD133 sonda: historia alterada'; END IF;
 END LOOP;
END $conservacion$;
-- Tipos temporales incompatibles para demostrar la precedencia de pg_catalog.
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(false);
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(false);
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(false);
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK(false);
REVOKE ALL ON TYPE pg_temp.bytea,pg_temp.jsonb,pg_temp.text,pg_temp.numeric FROM PUBLIC;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $tipos$
BEGIN
 IF vec_autorizacion_atestada_v3.bytea_igual_constante(pg_catalog.decode('00','hex'),pg_catalog.decode('00','hex')) IS NOT TRUE
 OR vec_autorizacion_atestada_v3.bytea_igual_constante(pg_catalog.decode('00','hex'),pg_catalog.decode('01','hex')) IS NOT FALSE
 OR vec_autorizacion_atestada_v3.huella_sha256_valida(pg_catalog.repeat('a',64)) IS NOT TRUE THEN
  RAISE EXCEPTION 'AD133 sonda: tipos temporales cambiaron el contrato';
 END IF;
END $tipos$;
RESET ROLE;
ROLLBACK;
\echo AD133-CONSERVACION-17-TIPOS-HISTORIA-OK
