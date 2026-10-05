\set ON_ERROR_STOP on
-- Todas las expresiones regulares literales de
-- revalidar_decision_registro_accesos_bolsa_v2 deben compilar. Con {1,512}
-- (PostgreSQL solo admite repeticiones hasta 255) la guarda inicial fallaba
-- con 2201B, que el EXCEPTION WHEN data_exception final convertía en «false»:
-- la revalidación devolvía siempre false sin error visible. Como
-- superusuario, en ROLLBACK; no depende de datos.
BEGIN;
DO $prueba$
DECLARE patron text;n int:=0;fallos int:=0;
BEGIN
 FOR patron IN SELECT replace(m[1],'''''','''') FROM pg_proc p,
   regexp_matches(p.prosrc,'~\s*''((?:[^'']|'''')*)''','g') AS m
  WHERE p.oid='vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)'::regprocedure
 LOOP
  n:=n+1;
  BEGIN PERFORM 'x' ~ patron;
  EXCEPTION WHEN invalid_regular_expression THEN fallos:=fallos+1; RAISE NOTICE 'no compila: %',patron;
  END;
 END LOOP;
 IF n<5 OR fallos>0 THEN RAISE EXCEPTION 'FALLO expresiones_compilan (% de %)',fallos,n; END IF;
 RAISE NOTICE 'OK expresiones_compilan (%)',n;
 -- Misma intención que {1,512}: no vacía, sin espacios ni '*', hasta 512.
 IF NOT (repeat('d',512) ~ '^[^*[:space:][:cntrl:]]+$' AND pg_catalog.char_length(repeat('d',512))<=512)
  OR pg_catalog.char_length(repeat('d',513))<=512 OR 'a b' ~ '^[^*[:space:][:cntrl:]]+$' OR '' ~ '^[^*[:space:][:cntrl:]]+$'
 THEN RAISE EXCEPTION 'FALLO intencion_1_512'; END IF;
 RAISE NOTICE 'OK intencion_1_512';
END $prueba$;
ROLLBACK;
