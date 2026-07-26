-- Retirada protegida del enlace VEC-CT; exige bajar primero las migraciones CT.
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
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.o404d_registrar_decision_cobertura_sin_enlace_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.enlace_decision_cobertura_ct_o404e'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de enlace VEC-CT O4-04E fuera de orden';
    END IF;
END
$prevalidacion$;

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e
    ) AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_enlace_vec_ct_o404e', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_ENLACE_VEC_CT_O404E_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada O4-04E protegida por historia VEC-CT';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM vec_contratacion_temporal_propietario;
REVOKE REFERENCES (
    decision_ref, rama, codigo, accion, decision_huella_sha256,
    correlacion_ref, organizacion_ref, expediente_ref,
    version_expediente, reserva_ref,
    contexto_recurso_huella_sha256, huella_orden_sha256
) ON TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e
    FROM vec_contratacion_temporal_propietario;

DROP FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) RESTRICT;
DROP TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e RESTRICT;

ALTER TABLE vec_autorizacion.decision_denegada_contexto_actor_v3
    DROP CONSTRAINT decision_denegada_v3_identidad_compuesta_o404e,
    DROP COLUMN contexto_recurso_huella_o404e,
    DROP COLUMN recurso_ref_o404e,
    DROP COLUMN correlacion_ref_o404e,
    DROP COLUMN accion_o404e,
    DROP COLUMN codigo_probatorio_o404e;
ALTER TABLE vec_autorizacion.decision_concedida_contexto_actor_v3
    DROP CONSTRAINT decision_concedida_v3_identidad_compuesta_o404e,
    DROP COLUMN contexto_recurso_huella_o404e,
    DROP COLUMN recurso_ref_o404e,
    DROP COLUMN correlacion_ref_o404e,
    DROP COLUMN accion_o404e,
    DROP COLUMN codigo_probatorio_o404e;

ALTER FUNCTION
vec_autorizacion.o404d_registrar_decision_cobertura_sin_enlace_v1(
    bytea, bytea, numeric, numeric, jsonb
) RENAME TO registrar_decision_cobertura_contratacion_temporal_v1;

GRANT EXECUTE ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) TO vec_contratacion_temporal_propietario;

COMMENT ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) IS
    'Wrapper interno O4-04D: resuelve y coteja el canon del recurso autorizado; solo O4-04E podrá componerlo con la persistencia en la misma transacción.';

COMMIT;
