\set ON_ERROR_STOP on
-- Reversión solo sin claves/atestaciones de esta audiencia; no usar sobre
-- historia conservada. El instalador necesita lectura AD3 para la guarda.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000068',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $historia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1')
 THEN RAISE EXCEPTION 'B68 DOWN: no admitido con historia de consulta' USING ERRCODE='55000'; END IF;
END $historia$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DROP FUNCTION vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
