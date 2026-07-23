BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000004_confirmar_alta_atestada', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_alta
    )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de confirmación protegida';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) FROM vec_contratacion_temporal_ejecutor;
DROP FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v1(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
);

-- La reversión recupera únicamente la preparación histórica. No abre v1.
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v2(jsonb)
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
