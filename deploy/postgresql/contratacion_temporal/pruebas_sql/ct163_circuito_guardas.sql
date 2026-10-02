\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    v_flujo jsonb := pg_catalog.jsonb_build_object(
        'definicion_ref', 'flujo:ct:rrhh:20261002', 'version', 2,
        'huella_sha256', '1721c3a66576b21163b590602589f1627095bd6b6bfa37c79862af62775146e2');
    v_actuacion_1 jsonb := pg_catalog.jsonb_build_object('accion_clave', 'registrar_solicitud');
    v_actuacion_2 jsonb := pg_catalog.jsonb_build_object(
        'accion_clave', 'contratacion_temporal.analisis.registrar', 'recibo_ref', 'recibo:prueba:163',
        'actor_ref', 'actor:prueba:163', 'unidad_ref', 'unidad:prueba:163',
        'realizada_en', '2026-10-02T12:00:00Z');
    v_actuacion_3 jsonb := pg_catalog.jsonb_build_object(
        'accion_clave', 'contratacion_temporal.cobertura.decidir', 'recibo_ref', 'recibo:prueba:163:3',
        'actor_ref', 'actor:prueba:163', 'unidad_ref', 'unidad:prueba:163',
        'realizada_en', '2026-10-02T12:01:00Z');
    v_anterior jsonb;
    v_siguiente jsonb;
    v_hito jsonb;
    v_peticion jsonb;
    v_autorizacion jsonb;
    v_par jsonb;
    v_mal jsonb;
    v_tercero jsonb;
    v_otro jsonb;
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
        'actuacion_clave', 'contratacion_temporal.analisis.registrar',
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
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_siguiente) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: primer análisis incompleto aceptado';
    END IF;
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_hito, v_hito || '{"secuencia":2,"origen":"analisis","destino":"informe"}'::jsonb));
    v_siguiente := pg_catalog.jsonb_set(v_siguiente, '{circuito,estado_actual}',
        '"informe"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_siguiente) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: dos hitos en una versión deben denegarse';
    END IF;

    v_peticion := v_hito || pg_catalog.jsonb_build_object(
        'clave', 'contratacion_temporal.circuito.peticion_firmada',
        'tipo', 'peticion_firmada', 'destino', 'autorizacion_rrhh');
    v_autorizacion := v_hito || pg_catalog.jsonb_build_object(
        'secuencia', 2, 'clave', 'contratacion_temporal.circuito.autorizacion_rrhh',
        'tipo', 'autorizacion_rrhh', 'origen', 'autorizacion_rrhh', 'destino', 'credito');
    v_par := pg_catalog.jsonb_set(v_siguiente, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_peticion, v_autorizacion));
    v_par := pg_catalog.jsonb_set(v_par, '{circuito,estado_actual}', '"credito"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_par) IS NOT TRUE THEN
        RAISE EXCEPTION 'CT163: par inicial acreditado debe aceptarse';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{flujo,huella_sha256}',
        '"f9b83c1291fdf96f339233b9e7a2036b67803388cea8568b4d2b02b6bcd4e9fc"'::jsonb);
    v_mal := pg_catalog.jsonb_set(v_mal, '{circuito,definicion,huella_sha256}',
        '"f9b83c1291fdf96f339233b9e7a2036b67803388cea8568b4d2b02b6bcd4e9fc"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: sustitución de huella entre versiones aceptada';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{actuaciones,1,accion_clave}',
        '"contratacion_temporal.cobertura.decidir"'::jsonb);
    v_mal := pg_catalog.jsonb_set(v_mal, '{circuito,hitos,0,actuacion_clave}',
        '"contratacion_temporal.cobertura.decidir"'::jsonb);
    v_mal := pg_catalog.jsonb_set(v_mal, '{circuito,hitos,1,actuacion_clave}',
        '"contratacion_temporal.cobertura.decidir"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: par inicial fuera del análisis aceptado';
    END IF;

    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_autorizacion, v_peticion));
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: par invertido aceptado';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_peticion, v_peticion || '{"secuencia":2,"origen":"autorizacion_rrhh","destino":"credito"}'::jsonb));
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: petición duplicada aceptada';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_autorizacion || '{"secuencia":1,"origen":"solicitud"}'::jsonb, v_autorizacion));
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: autorización duplicada aceptada';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos,1,tipo}', '"credito_comprobado"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: otro tipo aceptado';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos,1,recibo_ref}', '"recibo:otro:163"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: recibos distintos aceptados';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos,1,actor_ref}', '"actor:otro:163"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: actores distintos aceptados';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_par, '{circuito,hitos,1,version_expediente_entrada}', '2'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: versiones de entrada distintas aceptadas';
    END IF;

    v_tercero := v_autorizacion || pg_catalog.jsonb_build_object(
        'secuencia', 3, 'version_expediente_entrada', 2,
        'clave', 'contratacion_temporal.circuito.credito_comprobado',
        'tipo', 'credito_comprobado', 'actuacion_clave', 'contratacion_temporal.cobertura.decidir',
        'recibo_ref', 'recibo:prueba:163:3', 'registrado_en', '2026-10-02T12:01:00Z',
        'origen', 'credito', 'destino', 'oferta');
    v_otro := pg_catalog.jsonb_set(v_par, '{version}', '3'::jsonb);
    v_otro := pg_catalog.jsonb_set(v_otro, '{actuaciones}',
        pg_catalog.jsonb_build_array(v_actuacion_1, v_actuacion_2, v_actuacion_3));
    v_otro := pg_catalog.jsonb_set(v_otro, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_peticion, v_autorizacion, v_tercero));
    v_otro := pg_catalog.jsonb_set(v_otro, '{circuito,estado_actual}', '"oferta"'::jsonb);
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_par, v_otro) IS NOT TRUE THEN
        RAISE EXCEPTION 'CT163: hito único posterior rechazado';
    END IF;
    v_mal := pg_catalog.jsonb_set(v_otro, '{circuito,hitos}',
        pg_catalog.jsonb_build_array(v_peticion, v_autorizacion, v_tercero,
            v_tercero || '{"secuencia":4,"origen":"oferta","destino":"adjudicacion"}'::jsonb));
    IF vec_contratacion_temporal.circuito_siguiente_ct163(v_par, v_mal) IS NOT FALSE THEN
        RAISE EXCEPTION 'CT163: dos hitos posteriores aceptados';
    END IF;
END
$prueba$;

ROLLBACK;
