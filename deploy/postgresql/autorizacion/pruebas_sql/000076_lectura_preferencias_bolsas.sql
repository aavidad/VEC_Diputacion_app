\set ON_ERROR_STOP on
-- Ejecutar en PostgreSQL 18 desechable con AUT76 instalada.
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1(jsonb,text,text)');
 g oid; cuerpo text; fuente_sha text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AUT76: falta función'; END IF;
 SELECT pg_get_functiondef(f) INTO cuerpo;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
  AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
 OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 OR strpos(cuerpo,'asignacion_perfil_actual_externa')=0
 OR strpos(cuerpo,'bolsa.preferencias_bolsas.consultar_propias')=0
 OR strpos(cuerpo,'gestion_preferencias_bolsas_propias')=0
 THEN RAISE EXCEPTION 'AUT76: autoridad o ACL divergentes'; END IF;
 IF vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1('{}'::jsonb,'per_inexistente','prf_inexistente')
  IS DISTINCT FROM false THEN RAISE EXCEPTION 'AUT76: falló denegación por defecto'; END IF;
 FOR g IN SELECT to_regprocedure(x)::oid FROM unnest(ARRAY[
  'vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)',
  'vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)']) x LOOP
  IF g IS NULL OR has_function_privilege('public',g,'EXECUTE')
  THEN RAISE EXCEPTION 'AUT76: publicador ausente o expuesto %',g; END IF;
  SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO fuente_sha
   FROM pg_proc p WHERE p.oid=g;
  IF (g=to_regprocedure('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)')
      AND fuente_sha IS DISTINCT FROM '4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da')
   OR (g=to_regprocedure('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)')
      AND fuente_sha NOT IN (
       'b58b39f2c35341a545640bfde366f07a89ee65831306950c7f3ea342ee846ff0',
       '958adcfd570ec730964e5bc39c170ac6a0013a1a49a1d12e437c47191bdadd8b'))
  THEN RAISE EXCEPTION 'AUT76: fuente externa divergente %',g; END IF;
 END LOOP;
END $test$;
ROLLBACK;
