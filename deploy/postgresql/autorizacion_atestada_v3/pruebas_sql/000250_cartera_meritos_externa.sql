\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;def text;src text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT def
 FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF (length(def)-length(replace(def,'''lectura_cartera_meritos''::text','')))
    /length('''lectura_cartera_meritos''::text')<>1
 OR strpos(def,'''lectura_origen_datos_contacto''::text')=0
 THEN RAISE EXCEPTION 'AD250: familia lectora alterada' USING ERRCODE='55000';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_lectura_cartera_meritos_formato_v1'
   AND c.contype='c' AND c.convalidated)
 THEN RAISE EXCEPTION 'AD250: formato positivo ausente' USING ERRCODE='55000';END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT def
 FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check'
  AND c.contype='c' AND c.convalidated;
 IF (length(def)-length(replace(def,'''vec_meritos.cartera.retirar_propio.v1''::text','')))
    /length('''vec_meritos.cartera.retirar_propio.v1''::text')<>1
 OR strpos(def,'''vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1''::text')=0
 THEN RAISE EXCEPTION 'AD250: audiencia o audiencia histórica alterada' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 SELECT p.prosrc INTO STRICT src FROM pg_proc p WHERE p.oid=f;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
  AND p.proacl=ARRAY[
   'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario'::aclitem])
 OR src IS NULL OR strpos(src,'meritos_cartera_retirada')=0
 OR strpos(src,'registrar_y_revalidar_decision_meritos_externo_v3')=0
 OR strpos(src,'revalidar_decision_meritos_externo_v3_viva')=0
 OR strpos(src,'version_rol_bolsa_externa')=0
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD250: perfil M4 o perfiles previos ausentes' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD250: ACL consumidor divergente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD250: ACL lectura divergente' USING ERRCODE='55000';END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(bytea,bytea,jsonb,text)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD250: primitiva de intento expuesta' USING ERRCODE='55000';END IF;
 FOREACH f IN ARRAY ARRAY[
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(bytea,bytea,jsonb)')::oid,
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(bytea,bytea,jsonb)')::oid]
 LOOP
  IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
  OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD250: ACL de intento divergente' USING ERRCODE='55000';END IF;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'AD250: lectura aceptada sin emisor nominal' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'AD250: intento de lectura aceptado sin emisor nominal' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'AD250: intento aceptado sin emisor nominal' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD250: consumo aceptado sin emisor nominal' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $test$;
ROLLBACK;
