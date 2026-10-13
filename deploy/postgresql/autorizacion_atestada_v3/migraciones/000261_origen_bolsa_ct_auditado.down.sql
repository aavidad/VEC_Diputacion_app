\set ON_ERROR_STOP on
-- Sólo puede revertirse en ensayo sin accesos positivos asentados.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000261',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
  WHERE accion='bolsa.origen_ct.consultar' AND resultado='exito')
 THEN RAISE EXCEPTION 'AD261 DOWN denegado con auditoría de origen CT' USING ERRCODE='55000'; END IF;
END $historia$;
DROP FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text);
DROP FUNCTION vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(bytea,bytea,text,text,text,text);
COMMIT;
