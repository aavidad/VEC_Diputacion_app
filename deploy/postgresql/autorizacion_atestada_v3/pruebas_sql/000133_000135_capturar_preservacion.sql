\set ON_ERROR_STOP on
-- Sólo laboratorio sintético. Ejecutar inmediatamente antes de AD133 o AD135.
-- Parámetro psql obligatorio: -v etapa=133 o -v etapa=135.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.set_config('vec.ad133135.etapa',:'etapa',false);
CREATE TEMP TABLE ad133135_catalogo ON COMMIT PRESERVE ROWS AS
 SELECT p.oid,pg_catalog.to_jsonb(p) AS ficha,
        p.oid::pg_catalog.regprocedure::pg_catalog.text AS firma
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE pg_catalog.left(n.nspname,4)='vec_';
CREATE TEMP TABLE ad133135_dependencias ON COMMIT PRESERVE ROWS AS
 SELECT 'local'::pg_catalog.text AS clase,pg_catalog.to_jsonb(d) AS ficha
 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT oid FROM pg_temp.ad133135_catalogo)
 UNION ALL
 SELECT 'compartida',pg_catalog.to_jsonb(d) FROM pg_catalog.pg_shdepend d
 WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.objid IN(SELECT oid FROM pg_temp.ad133135_catalogo);
CREATE TEMP TABLE ad133135_tablas(tabla pg_catalog.text PRIMARY KEY,
 filas pg_catalog.int8,sha pg_catalog.text) ON COMMIT PRESERVE ROWS;
DO $captura$
DECLARE t pg_catalog.record; n pg_catalog.int8; h pg_catalog.text;
BEGIN
 IF pg_catalog.current_setting('vec.ad133135.etapa') NOT IN('133','135') THEN
  RAISE EXCEPTION 'Etapa de preservación incompatible' USING ERRCODE='55000';
 END IF;
 FOR t IN SELECT c.oid,c.relname,s.nspname FROM pg_catalog.pg_class c
 JOIN pg_catalog.pg_namespace s ON s.oid=c.relnamespace
 WHERE pg_catalog.left(s.nspname,4)='vec_' AND c.relkind IN('r','p') LOOP
  EXECUTE pg_catalog.format('SELECT count(*),encode(sha256(convert_to(coalesce(string_agg(h,'''' ORDER BY h),''''),''UTF8'')),''hex'') FROM (SELECT encode(sha256(convert_to(to_jsonb(x)::text,''UTF8'')),''hex'') AS h FROM %I.%I x) r',t.nspname,t.relname) INTO n,h;
  INSERT INTO pg_temp.ad133135_tablas VALUES(pg_catalog.format('%I.%I',t.nspname,t.relname),n,h);
 END LOOP;
END $captura$;
COMMIT;
