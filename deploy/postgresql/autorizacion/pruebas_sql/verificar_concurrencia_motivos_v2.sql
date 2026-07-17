\set ON_ERROR_STOP 1

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
BEGIN
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'motivos_autorizacion', 1, repeat('b', 64),
        'motivo_cccccccccccccccccccccccccccccccc'
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la barrera actual ignoro la retirada confirmada';
    END IF;
END
$prueba$;
COMMIT;
