-- Reversión únicamente antes de usar la capacidad. No destruye trabajo real.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo,
 vec_bolsa_llamamientos.llamamiento_integracion_desarrollo,
 vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
 vec_bolsa_llamamientos.outbox_integracion_desarrollo IN ACCESS EXCLUSIVE MODE;
DO $proteccion$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo) OR
  EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo) OR
  EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.auditoria_integracion_desarrollo) OR
  EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.outbox_integracion_desarrollo) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='se conserva la historia de llamamientos; reversión denegada';
 END IF;
END
$proteccion$;
DROP FUNCTION vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.buscar_integracion_desarrollo_v1(text);
DROP FUNCTION vec_bolsa_llamamientos.exigir_runtime_integracion_desarrollo();
DROP TABLE vec_bolsa_llamamientos.outbox_integracion_desarrollo;
DROP TABLE vec_bolsa_llamamientos.auditoria_integracion_desarrollo;
DROP TABLE vec_bolsa_llamamientos.llamamiento_integracion_desarrollo;
DROP TABLE vec_bolsa_llamamientos.integracion_desarrollo;
DROP FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_integracion_desarrollo();
COMMIT;
