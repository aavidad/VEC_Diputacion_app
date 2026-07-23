BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000004_integridad_agregado_alta', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.confirmacion_agregado_alta
    )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de integridad de altas protegida';
    END IF;
END
$proteccion$;

DROP FUNCTION vec_contratacion_temporal.reconciliar_agregado_alta_v1(
    bytea, text, text, text, text, text, text
);
DROP FUNCTION
    vec_contratacion_temporal.huella_prueba_agregado_alta_v1(text[]);

DROP TRIGGER outbox_alta_inmutable
    ON vec_contratacion_temporal.outbox_alta;
DROP TRIGGER confirmacion_agregado_inmutable
    ON vec_contratacion_temporal.confirmacion_agregado_alta;

ALTER TABLE vec_contratacion_temporal.reserva_alta_version
    DROP CONSTRAINT reserva_marcador_fk,
    DROP CONSTRAINT reserva_auditoria_marcador_fk,
    DROP CONSTRAINT reserva_outbox_marcador_fk;
ALTER TABLE vec_contratacion_temporal.expediente_alta
    DROP CONSTRAINT expediente_marcador_fk;
ALTER TABLE vec_contratacion_temporal.expediente_alta_version
    DROP CONSTRAINT version_marcador_fk;
ALTER TABLE vec_contratacion_temporal.actuacion_alta
    DROP CONSTRAINT actuacion_marcador_fk;
ALTER TABLE vec_contratacion_temporal.auditoria_alta
    DROP CONSTRAINT auditoria_marcador_fk;
ALTER TABLE vec_contratacion_temporal.outbox_alta
    DROP CONSTRAINT outbox_marcador_fk;

DROP TABLE vec_contratacion_temporal.confirmacion_agregado_alta;

DO $destruccion_reservas$
BEGIN
    IF pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal',
           true
       ) =
       'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        UPDATE vec_contratacion_temporal.reserva_alta_actual ra
           SET revision = (
               SELECT pg_catalog.max(r.revision)
                 FROM vec_contratacion_temporal.reserva_alta_version r
                WHERE r.ambito_hmac = ra.ambito_hmac
                  AND r.estado = 'reservada'
           )
         WHERE EXISTS (
             SELECT 1
               FROM vec_contratacion_temporal.reserva_alta_version vigente
              WHERE vigente.ambito_hmac = ra.ambito_hmac
                AND vigente.revision = ra.revision
                AND vigente.estado = 'confirmada'
         );
        ALTER TABLE vec_contratacion_temporal.reserva_alta_version
            DISABLE TRIGGER reserva_alta_version_inmutable;
        DELETE FROM vec_contratacion_temporal.reserva_alta_version
         WHERE estado = 'confirmada';
        ALTER TABLE vec_contratacion_temporal.reserva_alta_version
            ENABLE TRIGGER reserva_alta_version_inmutable;
    END IF;
END
$destruccion_reservas$;

ALTER TABLE vec_contratacion_temporal.reserva_alta_version
    DROP COLUMN confirmacion_ref;
ALTER TABLE vec_contratacion_temporal.expediente_alta
    DROP COLUMN confirmacion_ref;
ALTER TABLE vec_contratacion_temporal.expediente_alta_version
    DROP COLUMN confirmacion_ref;
ALTER TABLE vec_contratacion_temporal.actuacion_alta
    DROP COLUMN confirmacion_ref;
ALTER TABLE vec_contratacion_temporal.auditoria_alta
    DROP COLUMN confirmacion_ref,
    DROP COLUMN consumo_huella_sha256;
ALTER TABLE vec_contratacion_temporal.outbox_alta
    DROP COLUMN confirmacion_ref;
ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    DROP CONSTRAINT identidad_agregado_unica;

COMMIT;
