\set ON_ERROR_STOP on
-- CT253: comprueba la reconstrucción exacta y las concesiones instaladas.
-- El positivo de portal y correo requiere decisiones V3 reales; se ensaya
-- por el recorrido HTTP con CT249 y B113, sin inventar firmas ni resoluciones.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $test$
DECLARE x record; f regprocedure; p record;
BEGIN
 IF to_regclass('vec_contratacion_temporal.resolucion_manual_respuesta_rrhh') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.origen_financiero_o6_ct249_v1(jsonb,jsonb,text)') IS NULL
 THEN RAISE EXCEPTION 'CT253: falta CT249'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'ebfd9584069ae2f52d224849715849e07c0cbe18bc0ab31f0cc74d4e8a49c552'),
  ('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '7740e3ec1afed7e99cc17b7134cb07f9566a5ab200278fd6bb2a3190160a1ccd')
 ) v(firma,huella) LOOP
  f:=to_regprocedure(x.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'CT253: falta función %',x.firma; END IF;
  SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p
  FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_contratacion_temporal_propietario'
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
  OR p.proparallel IS DISTINCT FROM 'u'
  OR p.proacl::text IS DISTINCT FROM
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp',
   'row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>x.huella
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE') IS NOT TRUE
  OR EXISTS(SELECT 1 FROM pg_proc q
      CROSS JOIN LATERAL aclexplode(coalesce(q.proacl,acldefault('f',q.proowner))) a
      WHERE q.oid=f AND a.grantee=0)
  THEN RAISE EXCEPTION 'CT253: definición o ACL divergente %',x.firma; END IF;
 END LOOP;
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',
  'vec_contratacion_temporal.origen_financiero_o6_ct249_v1(jsonb,jsonb,text)','EXECUTE')
 OR has_table_privilege('vec_contratacion_temporal_ejecutor',
  'vec_contratacion_temporal.resolucion_manual_respuesta_rrhh','SELECT')
 OR has_table_privilege('vec_contratacion_temporal_ejecutor',
  'vec_contratacion_temporal.propuesta_formalizacion','SELECT')
 THEN RAISE EXCEPTION 'CT253: lectura directa abierta'; END IF;
END $test$;
-- Sin el contexto transaccional y la autoridad requeridos, ninguna fachada
-- alcanza los antecedentes ni los efectos; no se construyen evidencias falsas.
DO $denegacion$
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(
   NULL::text,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
   NULL::numeric,NULL::numeric,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea);
  RAISE EXCEPTION 'CT253: lectura sin contexto admitida';
 EXCEPTION WHEN SQLSTATE 'P0613' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(
   NULL::text,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
   NULL::numeric,NULL::numeric,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea);
  RAISE EXCEPTION 'CT253: registro sin contexto admitido';
 EXCEPTION WHEN SQLSTATE 'P0613' THEN NULL;
 END;
END $denegacion$;
ROLLBACK;
