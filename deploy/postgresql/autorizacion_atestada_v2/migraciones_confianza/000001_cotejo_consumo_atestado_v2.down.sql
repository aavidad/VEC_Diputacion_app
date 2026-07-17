BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire primero el registro atestado V2';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
        vec_autorizacion_atestada_v2_propietario;
DROP FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    );
COMMIT;
