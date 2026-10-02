\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    v_flujo jsonb := pg_catalog.jsonb_build_object(
        'definicion_ref', 'flujo:ct:rrhh:prueba', 'version', 2,
        'huella_sha256', pg_catalog.repeat('a', 64));
    v_actuacion_1 jsonb := pg_catalog.jsonb_build_object('accion_clave', 'registrar_solicitud');
    v_actuacion_2 jsonb := pg_catalog.jsonb_build_object(
        'accion_clave', 'confirmar_analisis', 'recibo_ref', 'recibo:prueba:163',
        'actor_ref', 'actor:prueba:163', 'unidad_ref', 'unidad:prueba:163',
        'realizada_en', '2026-10-02T12:00:00Z');
    v_anterior jsonb;
    v_siguiente jsonb;
    v_hito jsonb;
BEGIN
    v_anterior := pg_catalog.jsonb_build_object(
        'version', 1, 'referencia', 'expediente:prueba:163',
        'organizacion_ref', 'organizacion:prueba:163', 'flujo', v_flujo,
        'actuaciones', pg_catalog.jsonb_build_array(v_actuacion_1),
        'circuito', pg_catalog.jsonb_build_object(
            'definicion', v_flujo, 'estado_actual', 'solicitud',
            'hitos', '[]'::jsonb));
    v_hito := pg_catalog.jsonb_build_object(
        'secuencia', 1, 'version_expediente_entrada', 1,
        'actuacion_clave', 'confirmar_analisis',
        'recibo_ref', 'recibo:prueba:163', 'actor_ref', 'actor:prueba:163',
        'unidad_ref', 'unidad:prueba:163',
        'registrado_en', '2026-10-02T12:00:00Z',
        'origen', 'solicitud', 'destino', 'analisis');
    v_siguiente := pg_catalog.jsonb_set(v_anterior, '{version}', '2'::jsonb);
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{actuaciones}',
        pg_catalog.jsonb_build_array(v_actuacion_1, v_actuacion_2));
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_hito));
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,estado_actual}',
        '"analisis"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_siguiente) IS NOT TRUE THEN
        RAISE EXCEPTION 'CT163: un hito vinculado debe aceptarse';
    END IF;
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_hito, v_hito || '{"secuencia":2,"origen":"analisis","destino":"informe"}'::jsonb));
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,estado_actual}',
        '"informe"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_siguiente) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: dos hitos en una versión deben denegarse';
    END IF;
END
$prueba$;

ROLLBACK;
