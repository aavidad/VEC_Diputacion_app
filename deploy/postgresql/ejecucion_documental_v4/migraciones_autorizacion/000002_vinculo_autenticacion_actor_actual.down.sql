-- Reversion segura. Una base que ya contenga decisiones de 31 claves no puede
-- volver al CHECK de 30 sin mutilar evidencia inmutable; se rechaza el down.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.decision_autorizacion
         WHERE documento ? 'vinculo_autenticacion_actor'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen decisiones con contrato de 31 claves';
    END IF;
END
$prevalidacion$;

DROP TRIGGER decision_exige_vinculo_actual_v1
    ON vec_autorizacion.decision_autorizacion;
DROP FUNCTION vec_autorizacion.exigir_vinculo_actual_decision_v1();
DROP FUNCTION vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
    jsonb, text, text, text, timestamptz, timestamptz, timestamptz
);

DROP TABLE vec_autorizacion.contexto_actor_actual_v1;
DROP TABLE vec_autorizacion.contexto_actor_v1;
DROP TABLE vec_autorizacion.control_sesion_actual_v1;
DROP TABLE vec_autorizacion.control_sesion_v1;
DROP TABLE vec_autorizacion.sesion_autenticacion_v1;
DROP FUNCTION vec_autorizacion.validar_avance_contexto_actor_actual_v1();
DROP FUNCTION vec_autorizacion.validar_avance_control_sesion_actual_v1();

ALTER TABLE vec_autorizacion.decision_autorizacion
    DROP CONSTRAINT decision_documentos_tipo_v2;
ALTER TABLE vec_autorizacion.decision_autorizacion
    ADD CONSTRAINT decision_documentos_tipo CHECK (
        vec_autorizacion.documento_decision_estructura_valida_v1_legacy(
            documento
        ) IS TRUE
    );

DROP FUNCTION vec_autorizacion.documento_decision_estructura_valida(jsonb);
DROP FUNCTION
    vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(jsonb);
ALTER FUNCTION
    vec_autorizacion.documento_decision_estructura_valida_v1_legacy(jsonb)
    RENAME TO documento_decision_estructura_valida;

COMMIT;
