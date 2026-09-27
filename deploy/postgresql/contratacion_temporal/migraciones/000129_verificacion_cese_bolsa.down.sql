\set ON_ERROR_STOP on
-- CT129 DOWN: solo después de retirar consumidores Bolsa de esta fachada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000129',0));
DO $pre$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL THEN
        RAISE EXCEPTION 'CT129 DOWN: estado incompatible' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
                WHERE tipo_evento='ct.cese.v1') THEN
        RAISE EXCEPTION 'CT129 DOWN: no admitido con historia de cese' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint) RESTRICT;
DROP POLICY verificacion_cese_bolsa_ct129 ON vec_contratacion_temporal.cese_nombramiento_v1;
COMMIT;
