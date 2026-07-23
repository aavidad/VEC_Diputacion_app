\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;

DO $prueba$
DECLARE
    v_ambito text :=
        'hmac-sha256:vec.contratacion-temporal.'
        || 'ambito-idempotencia/v1:' || repeat('d', 64);
    v_filas bigint;
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM vec_contratacion_temporal.reserva_alta_actual AS actual
        JOIN vec_contratacion_temporal.reserva_alta_version AS version
          ON version.ambito_hmac = actual.ambito_hmac
         AND version.revision = actual.revision
        WHERE actual.ambito_hmac = v_ambito
          AND actual.revision = 1
          AND version.estado = 'reservada'
    ) THEN
        RAISE EXCEPTION 'falta la reserva v1 previa a la confirmación';
    END IF;

    INSERT INTO vec_contratacion_temporal.reserva_alta_version (
        ambito_hmac,
        revision,
        estado,
        version_expediente,
        auditoria_ref,
        evento_ref,
        confirmada_en,
        registrada_en
    )
    VALUES (
        v_ambito,
        2,
        'confirmada',
        7,
        'auditoria:alta-v1-confirmada',
        'evento:alta-v1-confirmada',
        '2026-07-23 12:34:56.123456+00'::timestamptz,
        '2026-07-23 12:34:56.123456+00'::timestamptz
    );
    UPDATE vec_contratacion_temporal.reserva_alta_actual
       SET revision = 2
     WHERE ambito_hmac = v_ambito
       AND revision = 1;
    GET DIAGNOSTICS v_filas = ROW_COUNT;
    IF v_filas <> 1 THEN
        RAISE EXCEPTION 'no se avanzó el puntero v1 confirmado';
    END IF;
END
$prueba$;

COMMIT;
