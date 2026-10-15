\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $t$
DECLARE p pg_proc%ROWTYPE;firma text;
BEGIN
 FOR firma IN SELECT unnest(ARRAY[
  'vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric)',
  'vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric)']) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(firma);
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
  OR p.prosecdef IS NOT TRUE OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR p.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
  THEN RAISE EXCEPTION 'AUT99: contrato de pareja divergente %',firma; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_proc pr WHERE
  pr.oid='vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(jsonb,jsonb)'::regprocedure
  AND pr.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
  AND pr.prosecdef) THEN RAISE EXCEPTION 'AUT99: predicado privado alterado'; END IF;
 IF vec_autorizacion.decision_documentos_bolsa_externa_exacta_v1(
  '{"modulo_id":"bolsa","accion":"bolsa.justificante_renuncia.descargar_rrhh"}'::jsonb,
  '{"referencia":{"catalogo_id":"motivos_autorizacion_justificante_renuncia_bolsa"}}'::jsonb) IS TRUE
 THEN RAISE EXCEPTION 'AUT99: RRHH admitido por pareja externa'; END IF;
END $t$;
ROLLBACK;
