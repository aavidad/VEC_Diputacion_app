-- Retira primero la frontera exterior. Conserva toda evidencia durable.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
    )
);

LOCK TABLE vec_autorizacion.decision_concedida_contexto_actor_v3,
           vec_autorizacion.decision_denegada_contexto_actor_v3
    IN ACCESS EXCLUSIVE MODE;

DO $conservacion$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: existen decisiones V3 durables';
    END IF;
END
$conservacion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.registrar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    ) FROM vec_autorizacion_registro;
DROP FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3(
    bytea, bytea, numeric, numeric
);
ALTER TABLE vec_autorizacion.decision_concedida_contexto_actor_v3
    DROP CONSTRAINT decision_v3_bytes_canonicos;
ALTER TABLE vec_autorizacion.decision_denegada_contexto_actor_v3
    DROP CONSTRAINT decision_v3_bytes_canonicos;
DROP FUNCTION vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb);
DROP FUNCTION vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb);

COMMIT;
