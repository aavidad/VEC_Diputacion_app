\set ON_ERROR_STOP 1

BEGIN;
DO $prueba$
BEGIN
    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 1, repeat('a', 64),
        'motivos_autorizacion', 1, repeat('b', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_cccccccccccccccccccccccccccccccc","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'no se pudo preparar la carrera contra down';
    END IF;
END
$prueba$;
-- Conserva sin confirmar tanto la evidencia como los bloqueos de escritura.
SELECT pg_sleep(4);
COMMIT;
