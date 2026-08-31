-- Retirada irreversible denegada salvo opt-in literal de esta sesion:
--   -c vec.confirmar_destruccion_registro_efectos_generacion_documental_v1=DESTRUIR_REGISTRO_EFECTOS_GENERACION_DOCUMENTAL_V1
BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_ejecucion_documental_v4:registro_efectos_generacion_documental_v1:down', 0
));

DO $confirmacion$
BEGIN
    IF current_setting(
        'vec.confirmar_destruccion_registro_efectos_generacion_documental_v1', true
    ) IS DISTINCT FROM 'DESTRUIR_REGISTRO_EFECTOS_GENERACION_DOCUMENTAL_V1' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down del registro de efectos documentales rechazado',
            DETAIL = 'la evidencia y los estados terminales se conservan por defecto',
            HINT = 'configure el opt-in literal solo tras copia y autorizacion';
    END IF;
END
$confirmacion$;

LOCK TABLE
    vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1
    IN ACCESS EXCLUSIVE MODE;

REVOKE EXECUTE ON FUNCTION
    vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb, jsonb)
    FROM vec_ejecucion_documental_v4_ejecutor_atestado;
REVOKE EXECUTE ON FUNCTION
    vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)
    FROM vec_ejecucion_documental_v4_ejecutor_atestado;
REVOKE EXECUTE ON FUNCTION
    vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)
    FROM vec_ejecucion_documental_v4_ejecutor_atestado;

DROP FUNCTION
    vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb, jsonb)
    RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)
    RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)
    RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
        text, text, text, text, timestamptz
    ) RESTRICT;

DROP TABLE
    vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1,
    vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1
    RESTRICT;
COMMIT;
