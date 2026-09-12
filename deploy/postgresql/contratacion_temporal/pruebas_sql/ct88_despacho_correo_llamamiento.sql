\set ON_ERROR_STOP on
-- Contrato estático CT88. Se ejecuta sólo contra una base donde la candidata
-- ya se haya aplicado; no genera ni transmite correo.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $prueba$
DECLARE r record;
BEGIN
  IF to_regclass('vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1') IS NULL
     OR to_regclass('vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1') IS NULL
     OR to_regprocedure('vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
     OR to_regprocedure('vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,bytea,text,text)') IS NULL THEN RAISE EXCEPTION 'CT88: contrato ausente'; END IF;
  FOR r IN SELECT c.relname,c.relrowsecurity,c.relforcerowsecurity FROM pg_class c WHERE c.oid IN ('vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1'::regclass,'vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1'::regclass) LOOP
    IF NOT r.relrowsecurity OR NOT r.relforcerowsecurity THEN RAISE EXCEPTION 'CT88: RLS ausente en %',r.relname; END IF;
  END LOOP;
  IF EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1'::regclass AND NOT attisdropped AND attname IN ('destino','correo','asunto','cuerpo','capacidad_finalizacion','token_finalizacion')) THEN RAISE EXCEPTION 'CT88: datos de correo o capacidad en claro prohibidos'; END IF;
  IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
     OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,bytea,text,text)','EXECUTE') THEN RAISE EXCEPTION 'CT88: ACL de ejecutor ausente'; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1'::regclass AND attname='finalizacion_huella_sha256' AND NOT attisdropped) THEN RAISE EXCEPTION 'CT88: capacidad de finalización sin huella durable'; END IF;
  IF to_regprocedure('vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,text)') IS NOT NULL THEN RAISE EXCEPTION 'CT88: firma de cierre insegura conservada'; END IF;
END
$prueba$;
ROLLBACK;
