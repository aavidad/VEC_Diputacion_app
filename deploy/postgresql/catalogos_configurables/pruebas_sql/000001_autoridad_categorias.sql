-- Ejecutar solo en una base desechable con roles_up.sql y 000001 UP instalados.
-- Toda la prueba revierte sus datos sintéticos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $prueba$
DECLARE
    documento text := '{"id":"rpt-demo","version":1,"estado":"publicado","entradas":[{"clave":"cat-demo","etiqueta":"Categoria de prueba"}]}';
    huella text;
    revision bigint;
    documento_version text;
    huella_version text;
    documento_siguiente text;
    huella_siguiente text;
BEGIN
    IF pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario',
         'vec_catalogos_configurables.publicacion', 'SELECT') THEN
        RAISE EXCEPTION 'AD3 puede leer tablas directamente';
    END IF;
    huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento, 'UTF8')), 'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-demo', 1, huella, documento,
        'aprobacion:a', 'aprobacion:b', 'actor:uno', 'decision:pub', 'recibo:pub-demo');
    IF vec_catalogos_configurables.publicar('rpt-demo', 1, huella, documento,
        'aprobacion:a', 'aprobacion:b', 'actor:dos', 'decision:pub-replay', 'recibo:pub-demo')
        <> 'recibo:pub-demo' THEN
        RAISE EXCEPTION 'replay de publicacion incorrecto';
    END IF;
    PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res1', 'recibo:res1');
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:des', 'recibo:des');
    IF revision <> 2 THEN RAISE EXCEPTION 'revision de deshabilitacion incorrecta'; END IF;
    IF vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res1', 'recibo:res1') <> 'recibo:res1' THEN
        RAISE EXCEPTION 'replay de reserva incorrecto';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:dos',
            'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res2', 'recibo:res2');
        RAISE EXCEPTION 'deshabilitacion admitio una nueva reserva';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'tombstone', NULL, NULL, 'actor:uno', 'decision:borrar', 'recibo:borrar');
        RAISE EXCEPTION 'tombstone sin cobertura admitido';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'cobertura', 'evidencia:historica', 0, 'actor:uno', 'decision:cob', 'recibo:cob');
        RAISE EXCEPTION 'cobertura con reserva pendiente admitida';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.terminar_uso('consumidor-demo', 'uso:uno',
        'recibo:res1', 'confirmado', 'actor:uno', 'decision:conf', 'recibo:conf');
    IF vec_catalogos_configurables.terminar_uso('consumidor-demo', 'uso:uno',
        'recibo:res1', 'confirmado', 'actor:uno', 'decision:conf', 'recibo:conf') <> 'recibo:conf' THEN
        RAISE EXCEPTION 'replay terminal incorrecto';
    END IF;
    IF vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:dos', 'decision:res-replay', 'recibo:res1')
        <> 'recibo:res1' THEN
        RAISE EXCEPTION 'reserva terminal no recuperable';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'cobertura', 'evidencia:historica', 0, 'actor:uno', 'decision:cob', 'recibo:cob');
        RAISE EXCEPTION 'cobertura menor que los usos confirmados admitida';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
        'cobertura', 'evidencia:historica', 1, 'actor:uno', 'decision:cob', 'recibo:cob');
    IF revision <> 3 THEN RAISE EXCEPTION 'revision de cobertura incorrecta'; END IF;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 3,
        'tombstone', NULL, NULL, 'actor:uno', 'decision:borrar', 'recibo:borrar');
    IF revision <> 4 THEN RAISE EXCEPTION 'revision de tombstone incorrecta'; END IF;
    IF vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 3,
        'tombstone', NULL, NULL, 'actor:dos', 'decision:borrar-replay', 'recibo:borrar') <> 4 THEN
        RAISE EXCEPTION 'replay de tombstone incorrecto';
    END IF;
    IF vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 1,
        'deshabilitar', NULL, NULL, 'actor:dos', 'decision:des-replay', 'recibo:des') <> 2
       OR vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
        'cobertura', 'evidencia:historica', 1, 'actor:dos', 'decision:cob-replay', 'recibo:cob') <> 3 THEN
        RAISE EXCEPTION 'replay de proyeccion antigua incorrecto';
    END IF;

    -- Una publicación posterior no rehabilita una categoría deshabilitada.
    documento_version := pg_catalog.jsonb_build_object('id', 'rpt-version', 'version', 1,
        'estado', 'publicado', 'entradas', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object('clave', 'cat-version', 'etiqueta', 'Categoria versionada')))::text;
    huella_version := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(documento_version, 'UTF8')), 'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-version', 1, huella_version,
        documento_version, 'aprobacion:version-a', 'aprobacion:version-b',
        'actor:uno', 'decision:version-1', 'recibo:version-1');
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-version', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:version-des', 'recibo:version-des');
    IF revision <> 2 THEN RAISE EXCEPTION 'deshabilitacion versionada incorrecta'; END IF;
    documento_siguiente := pg_catalog.jsonb_build_object('id', 'rpt-version', 'version', 2,
        'estado', 'publicado', 'entradas', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object('clave', 'cat-version', 'etiqueta', 'Nueva definicion',
                'preimagen_control', pg_catalog.jsonb_build_object('version', 1,
                    'huella_sha256', huella_version, 'revision', 1, 'estado', 'habilitada'))))::text;
    huella_siguiente := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(documento_siguiente, 'UTF8')), 'hex');
    BEGIN
        PERFORM vec_catalogos_configurables.publicar('rpt-version', 2, huella_siguiente,
            documento_siguiente, 'aprobacion:version-a2', 'aprobacion:version-b2',
            'actor:uno', 'decision:version-2', 'recibo:version-2');
        RAISE EXCEPTION 'publicacion admitio preimagen de habilitada obsoleta';
    EXCEPTION WHEN SQLSTATE '40001' THEN NULL;
    END;
    documento_siguiente := pg_catalog.jsonb_build_object('id', 'rpt-version', 'version', 2,
        'estado', 'publicado', 'entradas', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object('clave', 'cat-version', 'etiqueta', 'Nueva definicion',
                'preimagen_control', pg_catalog.jsonb_build_object('version', 1,
                    'huella_sha256', huella_version, 'revision', 2, 'estado', 'deshabilitada'))))::text;
    huella_siguiente := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(documento_siguiente, 'UTF8')), 'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-version', 2, huella_siguiente,
        documento_siguiente, 'aprobacion:version-a2', 'aprobacion:version-b2',
        'actor:uno', 'decision:version-2', 'recibo:version-2');
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:version',
            'cat-version', 'rpt-version', 2, huella_siguiente,
            'actor:uno', 'decision:version-res', 'recibo:version-res');
        RAISE EXCEPTION 'publicacion rehabilito categoria deshabilitada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END $prueba$;
RESET ROLE;
DO $persistencia$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion WHERE catalogo_id = 'rpt-demo')
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-demo' AND estado = 'tombstone')
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso
                       WHERE consumidor = 'consumidor-demo' AND uso_ref = 'uso:uno' AND estado = 'confirmado')
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-version' AND version = 2
                         AND revision = 3 AND estado = 'deshabilitada') THEN
        RAISE EXCEPTION 'tombstone perdio publicacion o uso';
    END IF;
END $persistencia$;
ROLLBACK;
