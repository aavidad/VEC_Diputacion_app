\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid; firma text;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)',
  'vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(bytea,bytea,jsonb)',
  'vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
   OR NOT has_function_privilege('vec_meritos_propietario',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD250: fachada ausente o ACL abierta %',firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_lectura_cartera_meritos_formato_v1'
  AND c.convalidated)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check'
  AND c.convalidated
  AND pg_get_constraintdef(c.oid,false) LIKE '%vec_meritos.cartera.retirar_propio.v1%')
 THEN RAISE EXCEPTION 'AD250: audiencia o familia de auditoría ausente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
   NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'AD250: lectura sin contexto concedida' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(
   NULL,NULL,'{}'::jsonb);
  RAISE EXCEPTION 'AD250: intento sin identidad concedido' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $test$;
ROLLBACK;
