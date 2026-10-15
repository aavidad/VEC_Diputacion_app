\set ON_ERROR_STOP on
-- CT223: una selección de oferta reservada impide confirmar la cancelación.
-- Reconstruye solo la función CT122 instalada en la copia S. CT220 debe
-- precederla; no altera reservas, recibos, efectos ni la selección singular.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path TO pg_catalog, pg_temp;
SET LOCAL timezone TO 'UTC';
SET LOCAL lock_timeout TO '5s';
SET LOCAL statement_timeout TO '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000223', 0));

DO $ct223$
DECLARE
  f oid := pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  anterior pg_catalog.pg_proc%ROWTYPE;
  posterior pg_catalog.pg_proc%ROWTYPE;
  definicion text;
  -- La marca incluye la adquisición del bloqueo de cabecera compartido con
  -- CT220. Su presencia una sola vez es una precondición de la sustitución.
  origen text := $origen$    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    v_dominio:=vec_contratacion_temporal.agregado_dominio_ct122(v_actual.agregado_json,v_actual.version);$origen$;
  destino text := $destino$    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF EXISTS (
      SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 seleccion
      WHERE seleccion.tipo_seleccion='oferta'
        AND seleccion.situacion IN ('propietaria','indeterminada')
        AND seleccion.solicitud_json->>'organizacion_ref'=m->>'organizacion_ref'
        AND seleccion.solicitud_json->>'expediente_ref'=m->>'expediente_ref'
    ) THEN
      RETURN jsonb_build_object('esquema',e,'resultado','seleccion_oferta_pendiente');
    END IF;
    v_dominio:=vec_contratacion_temporal.agregado_dominio_ct122(v_actual.agregado_json,v_actual.version);$destino$;
BEGIN
  IF pg_catalog.current_setting('server_version_num')::integer < 180000
     OR current_user <> 'vec_contratacion_temporal_propietario'
     OR f IS NULL
     OR pg_catalog.to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL
     OR pg_catalog.to_regprocedure('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_class t
       WHERE t.oid=pg_catalog.to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6')
         AND t.relowner='vec_contratacion_temporal_propietario'::regrole
         AND t.relrowsecurity AND t.relforcerowsecurity)
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_index x
       JOIN pg_catalog.pg_class idx ON idx.oid=x.indexrelid
       WHERE x.indrelid=pg_catalog.to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6')
         AND idx.relname='ejecucion_seleccion_oferta_unica_expediente_ct220'
         AND x.indisunique AND x.indisvalid)
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_attribute a
       WHERE a.attrelid=pg_catalog.to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6')
         AND a.attname IN ('tipo_seleccion','comando_oferta_json') AND NOT a.attisdropped
       GROUP BY a.attrelid HAVING pg_catalog.count(*)=2)
  THEN
    RAISE EXCEPTION 'CT223: PARO falta la preimagen CT122 o la dependencia CT220'
      USING ERRCODE='55000';
  END IF;
  SELECT * INTO STRICT anterior FROM pg_catalog.pg_proc WHERE oid=f;
  definicion:=pg_catalog.pg_get_functiondef(f);
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex')
       IS DISTINCT FROM '30af4df38af7ab6e004e5196657d67278cc8949b01b4521c8b30713309c09c35'
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM 'bc2145d14e5788b7b3315db797c407ba949837fe973d07da01ea411f9b6ab9a6'
     OR anterior.proowner <> 'vec_contratacion_temporal_propietario'::regrole
     OR anterior.pronamespace <> 'vec_contratacion_temporal'::regnamespace
     OR anterior.prorettype <> 'jsonb'::regtype
     OR anterior.pronargs <> 11
     OR anterior.prolang <> (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
     OR NOT anterior.prosecdef OR anterior.provolatile <> 'v'
     OR anterior.proconfig IS DISTINCT FROM ARRAY[
       'search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
     OR anterior.proacl::text IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
     OR (pg_catalog.length(anterior.prosrc)-pg_catalog.length(pg_catalog.replace(anterior.prosrc,origen,'')))
        <> pg_catalog.length(origen)
     OR (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,origen,'')))
        <> pg_catalog.length(origen)
  THEN
    RAISE EXCEPTION 'CT223: PARO cuerpo, propietario o ACL CT122 divergentes'
      USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.replace(definicion,origen,destino);
  EXECUTE 'ALTER FUNCTION vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET search_path TO pg_catalog, pg_temp';
  SELECT * INTO STRICT posterior FROM pg_catalog.pg_proc WHERE oid=f;
  IF posterior.prosrc IS DISTINCT FROM pg_catalog.replace(anterior.prosrc,origen,destino)
     OR posterior.proowner IS DISTINCT FROM anterior.proowner
     OR posterior.proacl IS DISTINCT FROM anterior.proacl
     OR posterior.prosecdef IS DISTINCT FROM anterior.prosecdef
     OR posterior.prorettype IS DISTINCT FROM anterior.prorettype
     OR posterior.provolatile IS DISTINCT FROM anterior.provolatile
     OR posterior.proconfig IS NULL
     OR pg_catalog.cardinality(posterior.proconfig) <> 4
     OR NOT (posterior.proconfig @> ARRAY['row_security=on','TimeZone=UTC','lock_timeout=2s']::text[])
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.unnest(posterior.proconfig) AS x(valor)
       WHERE pg_catalog.split_part(x.valor,'=',1)='search_path'
         AND pg_catalog.regexp_replace(pg_catalog.split_part(x.valor,'=',2),'[[:space:]]','','g')='pg_catalog,pg_temp')
  THEN
    RAISE EXCEPTION 'CT223: PARO postimagen, search_path o permisos divergentes'
      USING ERRCODE='55000';
  END IF;
END $ct223$;
COMMIT;
