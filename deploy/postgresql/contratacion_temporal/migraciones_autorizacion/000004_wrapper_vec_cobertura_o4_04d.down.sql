-- Retirada del wrapper privado; DROP RESTRICT protege la futura composición E.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de wrapper VEC O4-04D fuera de orden';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM vec_contratacion_temporal_propietario;

DROP FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) RESTRICT;
DROP FUNCTION vec_autorizacion.o404d_material_recurso_cobertura_v1(
    text, text, text, numeric, text, text, text
) RESTRICT;
DROP FUNCTION vec_autorizacion.o404d_registrar_decision_v3_viva_v1(
    bytea, bytea, numeric, numeric
) RESTRICT;
DROP FUNCTION vec_autorizacion.o404d_registrar_decision_v3_base_v1(
    bytea, bytea, numeric, numeric
) RESTRICT;
DROP FUNCTION
vec_autorizacion.o404d_decision_cobertura_v3_exacta_v1(
    jsonb, bytea
) RESTRICT;

COMMIT;
