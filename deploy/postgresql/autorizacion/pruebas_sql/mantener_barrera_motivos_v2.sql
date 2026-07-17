\set ON_ERROR_STOP 1

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
BEGIN
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'motivos_autorizacion', 1, repeat('b', 64),
        'motivo_cccccccccccccccccccccccccccccccc'
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la barrera concurrente no encontro el motivo';
    END IF;
END
$prueba$;
SELECT pg_sleep(4);
COMMIT;
