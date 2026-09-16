\set ON_ERROR_STOP on
-- CT101: prueba de huella_analisis_derivado_v2 con observaciones.
-- Comprueba:
-- 1. Base sin observaciones produce una huella no nula.
-- 2. Con observaciones validas (1..4000 caracteres), la huella es exactamente la misma.
-- 3. Con observaciones excesivas (4001 caracteres), la funcion devuelve NULL.
-- 4. Con observaciones vacias (cadena vacia), la funcion devuelve NULL (no debe existir clave vacia).
-- 5. Con tipo incorrecto (ej. numero), la funcion devuelve NULL.
-- No ejecuta escrituras; bloque transaccional con rollback.

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '20s';

DO $ct101_prueba$
DECLARE
    v_base jsonb;
    v_con_obs jsonb;
    v_obs_4001 jsonb;
    v_obs_vacia jsonb;
    v_obs_invalida jsonb;
    h_base text;
    h_con_obs text;
BEGIN
    v_base := pg_catalog.jsonb_build_object(
        'modalidad_clave', 'contratacion_temporal.interinidad',
        'categoria_ref', 'categoria:auxiliar',
        'grupo_subgrupo', 'C2',
        'causa_clave', 'acumulacion_tareas',
        'periodo', pg_catalog.jsonb_build_object(
            'inicio', '2026-08-01T00:00:00Z',
            'fin', '2026-08-31T00:00:00Z'
        ),
        'porcentaje_jornada', 10000,
        'entrada_rc_esperada', pg_catalog.jsonb_build_object(
            'referencia', 'entrada:rc:o3:001',
            'huella_sha256', pg_catalog.repeat('4', 64)
        ),
        'actuacion_registro', pg_catalog.jsonb_build_object(
            'secuencia', 2,
            'version_expediente', 2,
            'accion_clave', 'contratacion_temporal.analisis.registrar',
            'fase_destino', 'solicitud',
            'recibo_ref', 'recibo:analisis:001'
        ),
        'validacion_rc', pg_catalog.jsonb_build_object(
            'resultado', 'no_requerida',
            'entrada_ref', 'entrada:rc:o3:001',
            'huella_entrada_sha256', pg_catalog.repeat('4', 64),
            'fuente_ref', 'fuente:presupuestaria:o3',
            'recibo_ref', 'recibo:fuente:rc:o3',
            'validada_en', '2026-08-01T00:00:00Z',
            'motivo', 'tramite_ordinario'
        )
    );

    h_base := vec_contratacion_temporal.huella_analisis_derivado_v2(v_base);
    IF h_base IS NULL THEN
        RAISE EXCEPTION 'CT101: huella_analisis_derivado_v2 devolvio NULL para analisis base';
    END IF;

    -- Caso 1: con observaciones validas (texto valido <= 4000 caracteres)
    v_con_obs := pg_catalog.jsonb_set(
        v_base,
        '{observaciones}',
        to_jsonb('Observaciones validas de prueba para el analisis RRHH'::text)
    );
    h_con_obs := vec_contratacion_temporal.huella_analisis_derivado_v2(v_con_obs);
    IF h_con_obs IS NULL THEN
        RAISE EXCEPTION 'CT101: huella_analisis_derivado_v2 devolvio NULL para analisis con observaciones';
    END IF;
    IF h_con_obs <> h_base THEN
        RAISE EXCEPTION 'CT101: huella con observaciones (%) difiere de huella base (%)',
            h_con_obs, h_base;
    END IF;

    -- Caso 2: con observaciones de 4001 caracteres -> debe devolver NULL
    v_obs_4001 := pg_catalog.jsonb_set(
        v_base,
        '{observaciones}',
        to_jsonb(pg_catalog.repeat('a', 4001))
    );
    IF vec_contratacion_temporal.huella_analisis_derivado_v2(v_obs_4001) IS NOT NULL THEN
        RAISE EXCEPTION 'CT101: huella_analisis_derivado_v2 no rechazo observaciones de 4001 caracteres';
    END IF;

    -- Caso 3: con observaciones vacias "" -> debe devolver NULL (la clave se omite si esta vacia)
    v_obs_vacia := pg_catalog.jsonb_set(
        v_base,
        '{observaciones}',
        to_jsonb(''::text)
    );
    IF vec_contratacion_temporal.huella_analisis_derivado_v2(v_obs_vacia) IS NOT NULL THEN
        RAISE EXCEPTION 'CT101: huella_analisis_derivado_v2 no rechazo observaciones de cadena vacia';
    END IF;

    -- Caso 4: con tipo no string -> debe devolver NULL
    v_obs_invalida := pg_catalog.jsonb_set(
        v_base,
        '{observaciones}',
        '123'::jsonb
    );
    IF vec_contratacion_temporal.huella_analisis_derivado_v2(v_obs_invalida) IS NOT NULL THEN
        RAISE EXCEPTION 'CT101: huella_analisis_derivado_v2 no rechazo observaciones no string';
    END IF;

    RAISE NOTICE 'CT101: pruebas de huella_analisis_derivado_v2 con observaciones superadas exitosamente';
END
$ct101_prueba$;

ROLLBACK;
