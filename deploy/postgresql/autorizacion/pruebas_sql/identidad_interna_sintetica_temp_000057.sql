\set ON_ERROR_STOP on
-- Cotejo estructural de las fachadas instaladas; no crea roles ni objetos TEMP.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $cotejo$
DECLARE firma text; definicion record;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion.exigir_operador_identidad_interna_sintetica_v1()',
  'vec_autorizacion.preimagen_identidad_interna_sintetica_v1(jsonb,text)',
  'vec_autorizacion.aplicar_efecto_identidad_interna_sintetica_v1(text,text)',
  'vec_autorizacion.provisionar_identidad_interna_sintetica_v1(text,text)',
  'vec_autorizacion.recuperar_identidad_interna_sintetica_v1(text,text)',
  'vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb)',
  'vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb)'] LOOP
  SELECT p.prosecdef,p.proconfig INTO STRICT definicion
  FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(firma);
  IF NOT definicion.prosecdef OR NOT EXISTS(
   SELECT 1 FROM pg_catalog.unnest(definicion.proconfig) AS c(valor)
   WHERE pg_catalog.regexp_replace(c.valor,'[[:space:]]','','g')='search_path=pg_catalog,pg_temp')
  THEN RAISE EXCEPTION 'ruta_definer: clave=% esperado=pg_catalog,pg_temp observado=%',firma,definicion.proconfig; END IF;
 END LOOP;
END $cotejo$;
ROLLBACK;
