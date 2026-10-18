\set ON_ERROR_STOP on
-- AD246 instalada en una copia desechable. No requiere credenciales exteriores.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $login$ BEGIN
 IF to_regrole('vec_externo_bolsa_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_bolsa_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 END IF;
END $login$;
DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)');
 cuerpo text; previo bigint; estado text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD246: falta registrador de intento externo'; END IF;
 SELECT pg_get_functiondef(f) INTO cuerpo;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 OR has_function_privilege('public',f,'EXECUTE')
 OR has_function_privilege('vec_externo_bolsa_desarrollo',f,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR strpos(cuerpo,'revalidar_sesion_vinculo_externo_v2')=0
 OR strpos(cuerpo,'acreditar_uso_registro_contexto_actor_v2')=0
 OR strpos(cuerpo,'cotejar_lectura_preferencias_bolsas_actual_v1')<>0
 OR strpos(cuerpo,'''intento_nominal''')=0
 THEN RAISE EXCEPTION 'AD246: guardas o ACL de intento divergentes'; END IF;
 SELECT count(*) INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),'{}'::jsonb);
  RAISE EXCEPTION 'AD246: aceptó emisor sin login exterior';
 EXCEPTION WHEN insufficient_privilege THEN estado:=SQLSTATE; END;
 IF estado IS DISTINCT FROM '42501' OR
    (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>previo
 THEN RAISE EXCEPTION 'AD246: intento sin emisor creó asiento'; END IF;
END $test$;
ROLLBACK;
