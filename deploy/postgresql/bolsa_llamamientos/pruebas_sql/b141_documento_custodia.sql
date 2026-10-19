\set ON_ERROR_STOP on
-- En copia desechable después de B141. Solo ROLLBACK; no simula autorización.
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $test$ DECLARE r jsonb;ref text;
 -- Huella con ocho cifras seguidas de letra: casa por azar con el patrón de DNI.
 hex text:='12345678a'||repeat('b',55);
BEGIN
 FOREACH ref IN ARRAY ARRAY['registro/2026/0042','expediente:EXP-2026.17#acta','ref:'||repeat('c',64),
  'evento:ct:contrato-bolsa:'||hex] LOOP
  r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb_build_object('es_vacante',
   jsonb_build_object('valor',true,'documento_ref',ref,'huella',repeat('a',64))),'{}'::jsonb,'2026-10-19T00:00:00Z');
  IF r->>'resultado' IS DISTINCT FROM 'pendiente' THEN RAISE EXCEPTION 'rechazó referencia válida %: %',ref,r; END IF;
 END LOOP;
 FOREACH ref IN ARRAY ARRAY['dni:prueba','NIE-archivo','documento:12345678Z','X1234567L','ab',
  'con espacio','/registro','otro:contrato-bolsa:'||hex,'evento:ct:contrato-bolsa:'||upper(hex),
  'evento:ct:contrato-bolsa:'||left(hex,63),repeat('a',257)] LOOP
  BEGIN
   PERFORM vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb_build_object('es_vacante',
    jsonb_build_object('valor',true,'documento_ref',ref,'huella',repeat('a',64))),'{}'::jsonb,'2026-10-19T00:00:00Z');
   RAISE EXCEPTION 'admitió referencia inválida %',ref USING ERRCODE='P0001';
  EXCEPTION WHEN invalid_parameter_value THEN NULL;
  END;
 END LOOP;
 -- El resto del cálculo no cambia: huella y valor siguen siendo obligatorios.
 BEGIN
  PERFORM vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb_build_object('es_vacante',
   jsonb_build_object('valor',true,'documento_ref','registro/2026/0042','huella','x')),'{}'::jsonb,'2026-10-19T00:00:00Z');
  RAISE EXCEPTION 'admitió huella inválida' USING ERRCODE='P0001';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
END $test$;
RESET ROLE;
DO $acl$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('public',
   'vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)','EXECUTE')
 OR to_regprocedure('vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B141: permisos divergentes'; END IF;
END $acl$;
-- Sin material V3 la lectura nueva no devuelve filas: falla cerrada.
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $cerrada$ BEGIN
 PERFORM * FROM vec_bolsa_llamamientos.listar_contratos_participacion_v2('participacion:x','actor:x',
  NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 RAISE EXCEPTION 'lectura sin material' USING ERRCODE='P0001';
EXCEPTION WHEN raise_exception THEN RAISE;
 WHEN others THEN NULL;
END $cerrada$;
ROLLBACK;
