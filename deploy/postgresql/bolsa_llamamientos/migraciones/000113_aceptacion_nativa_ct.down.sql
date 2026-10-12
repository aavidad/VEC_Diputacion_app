\set ON_ERROR_STOP on
-- Reversión sólo antes de crear decisiones o consumos nominales B113.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000113',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
    WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'
      IN ('vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1','vec_bolsa_llamamientos.aceptacion_ct.persona.v1')
      AND convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'efecto_ref'
        LIKE 'recibo:respuesta-portal:%')
 THEN RAISE EXCEPTION 'B113 DOWN: no admitido con historia de consulta'
  USING ERRCODE='55000'; END IF;
END $historia$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B113 DOWN: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
