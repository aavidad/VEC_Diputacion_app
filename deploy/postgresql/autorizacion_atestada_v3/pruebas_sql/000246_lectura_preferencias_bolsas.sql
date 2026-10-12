\set ON_ERROR_STOP on
-- Ejecutar en PostgreSQL 18 desechable con AUT76 y AD246 instaladas.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)');
 previo bigint;posterior bigint;estado text;cuerpo text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD246: falta función'; END IF;
 SELECT pg_get_functiondef(f) INTO cuerpo;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR strpos(cuerpo,'acreditar_uso_registro_contexto_actor_v2')=0
 OR strpos(cuerpo,'revalidar_sesion_vinculo_externo_v2')=0
 OR strpos(cuerpo,'cotejar_lectura_preferencias_bolsas_actual_v1')=0
 THEN RAISE EXCEPTION 'AD246: frontera o ACL divergente'; END IF;
 SELECT count(*) INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),'{}'::jsonb);
  RAISE EXCEPTION 'AD246: aceptó emisor sin login externo';
 EXCEPTION WHEN insufficient_privilege THEN estado:=SQLSTATE; END;
 IF estado IS DISTINCT FROM '42501' THEN RAISE EXCEPTION 'AD246: rechazo inesperado %',estado; END IF;
 SELECT count(*) INTO posterior FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF posterior<>previo THEN RAISE EXCEPTION 'AD246: denegación creó auditoría positiva'; END IF;
END $test$;
ROLLBACK;
