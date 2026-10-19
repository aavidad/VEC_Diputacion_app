\set ON_ERROR_STOP on
-- Tras B137 en copia desechable; sin datos ni capacidad V3 sintética.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $acl$ DECLARE f regprocedure;p record;
BEGIN
 IF to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B137: siguió el lector antiguo'; END IF;
 f:='vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure;
 SELECT proowner,prosecdef,provolatile,proacl,proconfig,
  encode(sha256(convert_to(pg_get_functiondef(oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR p.huella IS DISTINCT FROM '401b1e9430d2d7c5be462ba733abb6378277b45d2224d1a554768fde93b9afac'
 OR p.proacl::text IS DISTINCT FROM
   '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR has_table_privilege('vec_autorizacion_atestada_v3_propietario',
   'vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1','SELECT')
 THEN RAISE EXCEPTION 'B137: definición o ACL divergente'; END IF;
END $acl$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $denegadas$ DECLARE c jsonb;
BEGIN
 FOREACH c IN ARRAY ARRAY[
  '[]'::jsonb,
  '{"expediente_ref":"expediente:otro"}'::jsonb,
  jsonb_build_object('organizacion_ref','organizacion:fixture','expediente_ref','expediente:fixture',
   'necesidad_ref','necesidad:fixture','unidad_ref','unidad:fixture','ambito_ref','ambito:otro',
   'categoria_ref','categoria:fixture','version_expediente',1,'modalidad_ofrecida','sustitucion',
   'causa_ofrecida','')
 ] LOOP
  BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
    'categoria:fixture','sustitucion','',c);
   RAISE EXCEPTION 'B137: se aceptó contexto inválido: %',c;
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
END $denegadas$;
ROLLBACK;
