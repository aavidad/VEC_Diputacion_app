BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000003_confirmacion_atestada', 0
    )
);

DO $proteccion$
DECLARE
    v_historia bigint;
BEGIN
    SELECT
        (SELECT count(*) FROM
            vec_contratacion_temporal.expediente_alta)
      + (SELECT count(*) FROM
            vec_contratacion_temporal.expediente_alta_version)
      + (SELECT count(*) FROM
            vec_contratacion_temporal.actuacion_alta)
      + (SELECT count(*) FROM
            vec_contratacion_temporal.auditoria_alta)
      + (SELECT count(*) FROM
            vec_contratacion_temporal.outbox_alta)
      INTO v_historia;
    IF v_historia > 0
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de expedientes protegida';
    END IF;
END
$proteccion$;

DROP TABLE vec_contratacion_temporal.outbox_alta;
DROP TABLE vec_contratacion_temporal.auditoria_alta;
DROP TABLE vec_contratacion_temporal.control_cadenas_alta;
DROP TABLE vec_contratacion_temporal.actuacion_alta;
DROP TABLE vec_contratacion_temporal.expediente_alta_version;
DROP TABLE vec_contratacion_temporal.expediente_alta;
DROP FUNCTION vec_contratacion_temporal.reconstruir_sellos_hmac_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.reconstruir_alta_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.texto_json_go_v1(text);

COMMIT;
