\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE s text; f oid;
BEGIN
 f:='vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF strpos(s,'vec_autorizacion.acciones_meritos_cartera_v2()')=0 OR
  NOT has_function_privilege('vec_autorizacion_fuente_meritos_externa',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT95: fuente M4 cambió de autoridad'; END IF;
 f:='vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF strpos(s,'"evidencias"')=0 OR
  strpos(s,'vec_autorizacion.acciones_meritos_cartera_v2()')=0
 THEN RAISE EXCEPTION 'AUT95: lectura M4 no acepta versión 2'; END IF;
 f:='vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF strpos(s,'meritos.cartera.comprobante.subir_propio')=0 OR
  strpos(s,'meritos.cartera.comprobante.descargar_propio')=0 OR
  strpos(s,'motivos_autorizacion_meritos_cartera')=0 OR
  strpos(s,'["auditar"]')=0 OR
  NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT95: decisión M4 perdió historia o nuevas acciones'; END IF;
 f:='vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF strpos(s,'rol_meritos_cartera_cinco_v2')=0 OR
  strpos(s,'["accion":"meritos.cartera.consultar_propia"')=0
 THEN RAISE EXCEPTION 'AUT95: publicador perdió rol M4 legado'; END IF;
 IF vec_autorizacion.acciones_meritos_cartera_v2() IS DISTINCT FROM ARRAY[
  'meritos.cartera.comprobante.descargar_propio',
  'meritos.cartera.comprobante.subir_propio',
  'meritos.cartera.consultar_propia',
  'meritos.cartera.retirar_propio',
  'meritos.hecho.declarar']::text[]
 THEN RAISE EXCEPTION 'AUT95: acciones M4 v2 divergentes'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure,
  'vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)'::regprocedure,
  'vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure]
 LOOP
  s:=pg_get_functiondef(f);
  IF strpos(s,'vec_externo_documentos_datos')=0 OR strpos(s,'vec_documentos_subida_datos')=0 OR
   strpos(s,'vec_externo_meritos_operacion')=0
  THEN RAISE EXCEPTION 'AUT95: registro o sesión M4 no admiten cuenta nominal'; END IF;
 END LOOP;
END $test$;
ROLLBACK;
