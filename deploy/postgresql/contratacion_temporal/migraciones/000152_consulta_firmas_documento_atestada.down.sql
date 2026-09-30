\set ON_ERROR_STOP on
-- CT152 DOWN: solo en clon desechable sin claves ni atestaciones de esta
-- audiencia. No ejecutar sobre historia conservada. La guarda requiere el
-- instalador con lectura AD3; carecer de ese permiso impide la reversión.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000152',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo='vec_contratacion_temporal.firmas_documento.consultar.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
               WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_contratacion_temporal.firmas_documento.consultar.v1')
 THEN RAISE EXCEPTION 'CT152 DOWN: no admitido con historia de consulta de firmas' USING ERRCODE='55000'; END IF;
END $historia$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v2(text,text) TO vec_contratacion_temporal_ejecutor;
COMMIT;
