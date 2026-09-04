BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:resolver-motivo-decision:000007',
        0
    )
);

REVOKE EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    text, text, timestamptz
) FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION
vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    text, text, timestamptz
);

REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_contratacion_temporal_ejecutor;

COMMIT;
