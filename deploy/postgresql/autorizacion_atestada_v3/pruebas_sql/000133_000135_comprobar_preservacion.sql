\set ON_ERROR_STOP on
-- Misma sesión que capturar_preservacion; después de una sola migración.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='60s';
DO $comprobar$
DECLARE r pg_catalog.record; p pg_catalog.pg_proc%ROWTYPE; fuente pg_catalog.text;
        etapa pg_catalog.text := pg_catalog.current_setting('vec.ad133135.etapa');
        n pg_catalog.int8; h pg_catalog.text; cambio_permitido pg_catalog.bool;
BEGIN
 FOR r IN SELECT * FROM pg_temp.ad133135_catalogo LOOP
  SELECT x.* INTO STRICT p FROM pg_catalog.pg_proc x WHERE x.oid=r.oid;
  cambio_permitido:=false;
  IF etapa='133' AND p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3') THEN
   IF pg_catalog.to_jsonb(p)-'proconfig' IS DISTINCT FROM r.ficha-'proconfig'
      OR p.proconfig IS DISTINCT FROM ARRAY(SELECT CASE WHEN x='search_path=pg_catalog'
         THEN 'search_path=pg_catalog, pg_temp' ELSE x END
         FROM pg_catalog.jsonb_array_elements_text(r.ficha->'proconfig') WITH ORDINALITY AS t(x,i) ORDER BY i) THEN
    RAISE EXCEPTION 'AD133: catálogo alterado fuera de search_path %',r.firma USING ERRCODE='55000';
   END IF;
   cambio_permitido:=true;
  ELSIF etapa='135' AND r.firma IN(
    'vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)',
    'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)') THEN
   fuente:=pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(p.prosrc,
    'vec_autorizacion_atestada_v3.puntero_configuracion_externa','vec_autorizacion_atestada_v3.puntero_configuracion_actual'),
    'vec_autorizacion_atestada_v3.checkpoint_gobierno_externo','vec_autorizacion_atestada_v3.checkpoint_gobierno'),
    'vec_autorizacion_atestada_v3.puntero_clave_emision_externa','vec_autorizacion_atestada_v3.puntero_clave_emision');
   IF fuente IS DISTINCT FROM r.ficha->>'prosrc'
      OR pg_catalog.to_jsonb(p)-'prosrc' IS DISTINCT FROM r.ficha-'prosrc' THEN
    RAISE EXCEPTION 'AD135: lector alterado fuera de sustituciones %',r.firma USING ERRCODE='55000';
   END IF;
   cambio_permitido:=true;
  END IF;
  IF NOT cambio_permitido AND pg_catalog.to_jsonb(p) IS DISTINCT FROM r.ficha THEN
   RAISE EXCEPTION 'Catálogo ajeno alterado %',r.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF EXISTS (WITH actuales AS (
   SELECT 'local'::pg_catalog.text AS clase,pg_catalog.to_jsonb(d) AS ficha FROM pg_catalog.pg_depend d
   WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid IN(SELECT oid FROM pg_temp.ad133135_catalogo)
   UNION ALL SELECT 'compartida',pg_catalog.to_jsonb(d) FROM pg_catalog.pg_shdepend d
   WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
   AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
   AND d.objid IN(SELECT oid FROM pg_temp.ad133135_catalogo))
  (SELECT clase,ficha FROM pg_temp.ad133135_dependencias EXCEPT SELECT clase,ficha FROM actuales)
  UNION ALL
  (SELECT clase,ficha FROM actuales EXCEPT SELECT clase,ficha FROM pg_temp.ad133135_dependencias)
 ) THEN RAISE EXCEPTION 'Dependencias previas alteradas' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM pg_temp.ad133135_tablas LOOP
  IF pg_catalog.to_regclass(r.tabla) IS NULL THEN
   RAISE EXCEPTION 'Tabla previa eliminada %',r.tabla USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.format('SELECT count(*),encode(sha256(convert_to(coalesce(string_agg(h,'''' ORDER BY h),''''),''UTF8'')),''hex'') FROM (SELECT encode(sha256(convert_to(to_jsonb(x)::text,''UTF8'')),''hex'') AS h FROM %s x) r',r.tabla) INTO n,h;
  IF n IS DISTINCT FROM r.filas OR h IS DISTINCT FROM r.sha THEN
   RAISE EXCEPTION 'Historia o datos previos alterados %',r.tabla USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF etapa='133' AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc proc_actual
     WHERE proc_actual.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
     AND NOT(proc_actual.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])) THEN
  RAISE EXCEPTION 'AD133: función pendiente' USING ERRCODE='55000';
 END IF;
END $comprobar$;
DROP TABLE pg_temp.ad133135_catalogo,pg_temp.ad133135_dependencias,pg_temp.ad133135_tablas;
COMMIT;
