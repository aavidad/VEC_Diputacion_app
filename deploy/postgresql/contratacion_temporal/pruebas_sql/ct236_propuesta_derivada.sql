\set ON_ERROR_STOP on
-- Ejecutar tras CT236 en copia local con CT234 instalado. No fabrica actos V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
DO $test$
DECLARE f regprocedure; p record; caso record; fuente text; mensaje text; estado text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_attribute a
    WHERE a.attrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
      AND a.attname='derivacion_ref' AND NOT a.attisdropped)
    OR EXISTS(SELECT 1 FROM pg_attribute a
      WHERE a.attrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
        AND a.attname IN ('llamamiento_ref','resolucion_ref','recibo_aceptacion_ref')
        AND a.attnotnull)
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
      WHERE c.conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
        AND c.conname='propuesta_origen_derivado_ct236' AND c.convalidated)
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
      WHERE c.conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
        AND c.conname='propuesta_formalizacion_check3' AND c.convalidated
        AND strpos(pg_get_constraintdef(c.oid),'AceptacionDerivada')>0)
 THEN RAISE EXCEPTION 'CT236: tabla o origen disjunto ausente'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.proteger_version_circuito_ct163()'::regprocedure,
  'vec_contratacion_temporal.proteger_version_circuito_ct164()'::regprocedure] LOOP
  SELECT proowner,prosecdef,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR p.prosecdef IS NOT TRUE
     OR NOT EXISTS(SELECT 1 FROM unnest(p.proconfig) cfg
          WHERE replace(cfg,' ','')='search_path=pg_catalog,pg_temp')
  THEN RAISE EXCEPTION 'CT236: función sin propietario o ruta fija: %',f; END IF;
 END LOOP;
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(jsonb,jsonb)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'CT236: helper privado expuesto'; END IF;
 SELECT prosrc INTO STRICT fuente FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.propuesta_derivada_ct236_v1(text,jsonb,jsonb,numeric)');
 IF strpos(fuente,'p_version=7')=0
    OR strpos(fuente,'''fase_actual''=''fiscalizacion''')=0
    OR strpos(fuente,'''fase_actual''=''nombramiento''')=0
    OR strpos(fuente,'''accion_clave''=''registrar_propuesta_formalizacion''')=0
    OR strpos(fuente,'acto6.actuacion_json=p_anterior')=0
    OR strpos(fuente,'p_nuevo-ARRAY[')=0
 THEN RAISE EXCEPTION 'CT236: paso de propuesta sin límite nominal'; END IF;

 -- Si el recorrido produjo una propuesta, cada referencia deriva de CT234
 -- y el hijo conserva sus dos actuaciones reales (v6 y v7).
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion propuesta
  LEFT JOIN vec_contratacion_temporal.nombramiento_derivado_v1 derivada
    ON derivada.derivacion_ref=propuesta.derivacion_ref
  LEFT JOIN vec_contratacion_temporal.expediente_version_integral v7
    ON v7.expediente_ref=propuesta.expediente_ref AND v7.version=7
  WHERE propuesta.derivacion_ref IS NOT NULL
    AND (derivada.hijo_ref IS DISTINCT FROM propuesta.expediente_ref
      OR derivada.organizacion_ref IS DISTINCT FROM propuesta.organizacion_ref
      OR propuesta.aceptacion_bolsa_json IS DISTINCT FROM jsonb_build_object(
        'DerivacionRef',derivada.derivacion_ref,'PadreRef',derivada.padre_ref,
        'HijoRef',derivada.hijo_ref,'OfertaRef',derivada.oferta_ref,
        'NumeroPlaza',derivada.numero_plaza,'AceptacionRef',derivada.aceptacion_ref,
        'ReciboDerivacionRef',derivada.recibo_ref)
      OR v7.origen_version IS DISTINCT FROM 'propuesta_formalizacion_o6'
      OR jsonb_array_length(v7.agregado_json->'actuaciones') IS DISTINCT FROM 2))
 THEN RAISE EXCEPTION 'CT236: propuesta derivada sin historia coherente'; END IF;
 FOR caso IN SELECT propuesta.expediente_ref,v6.agregado_json AS anterior,v7.agregado_json AS nuevo
  FROM vec_contratacion_temporal.propuesta_formalizacion propuesta
  JOIN vec_contratacion_temporal.expediente_version_integral v6
    ON v6.expediente_ref=propuesta.expediente_ref AND v6.version=6
  JOIN vec_contratacion_temporal.expediente_version_integral v7
    ON v7.expediente_ref=propuesta.expediente_ref AND v7.version=7
  WHERE propuesta.derivacion_ref IS NOT NULL LOOP
  IF vec_contratacion_temporal.propuesta_derivada_ct236_v1(
       caso.expediente_ref,caso.anterior,caso.nuevo,7) IS NOT TRUE
     OR vec_contratacion_temporal.propuesta_derivada_ct236_v1(
       caso.expediente_ref,caso.anterior,caso.nuevo,8) IS NOT FALSE
     OR vec_contratacion_temporal.propuesta_derivada_ct236_v1(
       caso.expediente_ref,caso.anterior,
       jsonb_set(caso.nuevo,'{actuaciones,1,accion_clave}',
         '"accion:ajena"'::jsonb),7) IS NOT FALSE
  THEN RAISE EXCEPTION 'CT236: transición derivada admitida fuera de propuesta'; END IF;
 END LOOP;

 BEGIN
  PERFORM vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1('{}'::jsonb,NULL);
  RAISE EXCEPTION 'CT236: llamada directa al helper aceptada';
 EXCEPTION WHEN SQLSTATE 'P0613' THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  IF estado<>'P0613' OR mensaje<>'CT236: antecedente denegado' THEN
   RAISE EXCEPTION 'CT236: denegación inesperada'; END IF;
 END;
END $test$;
ROLLBACK;
