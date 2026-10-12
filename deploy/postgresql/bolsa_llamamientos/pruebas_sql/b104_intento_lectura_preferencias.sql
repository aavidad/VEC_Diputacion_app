\set ON_ERROR_STOP on
-- B104 instalada en una copia desechable.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(text,bytea,bytea,jsonb)');
 cuerpo text; previo bigint; estado text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B104: falta fachada de intento'; END IF;
 SELECT pg_get_functiondef(f) INTO cuerpo;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_portal_externo',f,'EXECUTE')
 OR strpos(cuerpo,'registrar_intento_lectura_preferencias_bolsas_v1')=0
 THEN RAISE EXCEPTION 'B104: fachada o ACL divergente'; END IF;
 SELECT count(*) INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(
   'can_AAAAAAAAAAAAAAAAAAAAAA',convert_to('{}','UTF8'),convert_to('{}','UTF8'),'{}'::jsonb);
  RAISE EXCEPTION 'B104: aceptó intento sin sesión';
 EXCEPTION WHEN insufficient_privilege THEN estado:=SQLSTATE; END;
 IF estado IS DISTINCT FROM '42501' OR
    (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>previo
 THEN RAISE EXCEPTION 'B104: intento sin sesión creó asiento'; END IF;
END $test$;
ROLLBACK;
