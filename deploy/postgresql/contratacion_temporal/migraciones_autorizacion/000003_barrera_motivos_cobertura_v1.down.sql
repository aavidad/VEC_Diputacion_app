-- Retirada de la superficie de lectura. DROP RESTRICT preserva consumidores
-- instalados; cualquier fallo revierte también las revocaciones de esta tx.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:barrera-motivo-cobertura-v1:000003',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_checkpoint_origen'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no existe la proyección requerida por la barrera';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_cobertura_actual_v1(
    text, integer, text, text, text
) FROM vec_contratacion_temporal_propietario;
REVOKE EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_cobertura_historico_v1(
    text, integer, text, text, text, timestamptz
) FROM vec_autorizacion_motivos_evaluador;

DROP FUNCTION vec_autorizacion.resolver_motivo_cobertura_actual_v1(
    text, integer, text, text, text
) RESTRICT;
DROP FUNCTION vec_autorizacion.resolver_motivo_cobertura_historico_v1(
    text, integer, text, text, text, timestamptz
) RESTRICT;

-- USAGE puede estar compartido con otras fronteras y se conserva.
COMMIT;
