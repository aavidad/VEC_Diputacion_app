\set ON_ERROR_STOP 1

DO $prueba$
BEGIN
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_dddddddddddddddddddddddddddddddd', 2, repeat('d', 64),
        'motivos_autorizacion', 1, repeat('b', 64), repeat('e', 64),
        '2026-02-01T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la retirada concurrente no se confirmo';
    END IF;
END
$prueba$;
