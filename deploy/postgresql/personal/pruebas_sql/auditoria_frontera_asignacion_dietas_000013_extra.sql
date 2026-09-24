\set ON_ERROR_STOP on
-- Ejecutar como LOGIN con membresía ajena; el permiso SQL heredado no basta.
DO $prueba$
BEGIN
    BEGIN
        PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
            'corr_no_disponible', 'acceso_denegado', 'api.personal.asignaciones_dietas',
            '/api/vec/personal/asignaciones-dietas/detalle', 'consultar', NULL,
            'rel_AbCdef0123456789_-QRST', 403::smallint);
        RAISE EXCEPTION 'se acepto login con membresia extra';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END
$prueba$;
