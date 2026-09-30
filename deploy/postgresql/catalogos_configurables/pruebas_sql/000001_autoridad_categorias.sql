-- Ejecutar solo en una base desechable con roles_up.sql y 000001 UP instalados.
-- Toda la prueba revierte sus datos sintéticos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $prueba$
DECLARE
    documento text := '{"id":"rpt-demo","version":1,"estado":"publicado","entradas":[{"clave":"cat-demo","etiqueta":"Categoria usada"},{"clave":"cat-empty","etiqueta":"Categoria sin usos"},{"clave":"cat-declared","etiqueta":"Categoria con total declarado"}]}';
    motivo text := 'motivos_rpt:1:motivo_11111111111111111111111111111111';
    motivo_nuevo text := 'motivos_rpt:2:motivo_22222222222222222222222222222222';
    huella text;
    revision bigint;
    documento_version text;
    huella_version text;
    documento_siguiente text;
    huella_siguiente text;
    preimagenes jsonb;
    huella_preimagenes text;
    huella_vacias text := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}', 'UTF8')), 'hex');
BEGIN
    IF pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario',
         'vec_catalogos_configurables.publicacion', 'SELECT') THEN
        RAISE EXCEPTION 'AD3 puede leer tablas directamente';
    END IF;
    huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento, 'UTF8')), 'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-demo', 1, huella, documento,
        '{}'::jsonb, huella_vacias,
        'aprobacion:a', 'aprobacion:b', 'actor:uno', 'decision:pub', 'recibo:pub-demo', motivo);
    IF vec_catalogos_configurables.publicar('rpt-demo', 1, huella, documento,
        '{}'::jsonb, huella_vacias,
        'aprobacion:a', 'aprobacion:b', 'actor:dos', 'decision:pub-replay', 'recibo:pub-demo', motivo_nuevo)
        <> 'recibo:pub-demo' THEN
        RAISE EXCEPTION 'replay de publicacion incorrecto';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:motivo-libre',
            'cat-demo', 'rpt-demo', 1, huella,
            'actor:uno', 'decision:sin-motivo', 'recibo:sin-motivo', 'motivo-libre');
        RAISE EXCEPTION 'reserva admitio motivo_ref mal formado';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res1', 'recibo:res1', motivo);
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:des', 'recibo:des', motivo);
    IF revision <> 2 THEN RAISE EXCEPTION 'revision de deshabilitacion incorrecta'; END IF;
    IF vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res1', 'recibo:res1', motivo_nuevo) <> 'recibo:res1' THEN
        RAISE EXCEPTION 'replay de reserva incorrecto';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:dos',
            'cat-demo', 'rpt-demo', 1, huella, 'actor:uno', 'decision:res2', 'recibo:res2', motivo);
        RAISE EXCEPTION 'deshabilitacion admitio una nueva reserva';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'tombstone', NULL, NULL, 'actor:uno', 'decision:borrar', 'recibo:borrar', motivo);
        RAISE EXCEPTION 'tombstone sin cobertura admitido';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'cobertura', 'evidencia:historica', 0, 'actor:uno', 'decision:cob', 'recibo:cob', motivo);
        RAISE EXCEPTION 'cobertura con reserva pendiente admitida';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.terminar_uso('consumidor-demo', 'uso:uno',
        'recibo:res1', 'confirmado', 'actor:uno', 'decision:conf', 'recibo:conf', motivo);
    IF vec_catalogos_configurables.terminar_uso('consumidor-demo', 'uso:uno',
        'recibo:res1', 'confirmado', 'actor:uno', 'decision:conf', 'recibo:conf', motivo_nuevo) <> 'recibo:conf' THEN
        RAISE EXCEPTION 'replay terminal incorrecto';
    END IF;
    IF vec_catalogos_configurables.reservar('consumidor-demo', 'uso:uno',
        'cat-demo', 'rpt-demo', 1, huella, 'actor:dos', 'decision:res-replay', 'recibo:res1', motivo_nuevo)
        <> 'recibo:res1' THEN
        RAISE EXCEPTION 'reserva terminal no recuperable';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
            'cobertura', 'evidencia:historica', 0, 'actor:uno', 'decision:cob', 'recibo:cob', motivo);
        RAISE EXCEPTION 'cobertura menor que los usos confirmados admitida';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
        'cobertura', 'evidencia:historica', 1, 'actor:uno', 'decision:cob', 'recibo:cob', motivo);
    IF revision <> 3 THEN RAISE EXCEPTION 'revision de cobertura incorrecta'; END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 3,
            'tombstone', NULL, NULL, 'actor:uno', 'decision:borrar', 'recibo:borrar', motivo);
        RAISE EXCEPTION 'tombstone admitio un uso confirmado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    IF vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 1,
        'deshabilitar', NULL, NULL, 'actor:dos', 'decision:des-replay', 'recibo:des', motivo_nuevo) <> 2
       OR vec_catalogos_configurables.cambiar_proyeccion('cat-demo', 2,
        'cobertura', 'evidencia:historica', 1, 'actor:dos', 'decision:cob-replay', 'recibo:cob', motivo_nuevo) <> 3 THEN
        RAISE EXCEPTION 'replay de proyeccion antigua incorrecto';
    END IF;

    -- Ni un total histórico declarado positivo sin usos locales puede
    -- autorizar tombstone. La cobertura cero de otra categoría sí puede.
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-declared', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:decl-des', 'recibo:decl-des', motivo);
    IF revision <> 2 THEN RAISE EXCEPTION 'deshabilitacion declarada incorrecta'; END IF;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-declared', 2,
        'cobertura', 'evidencia:declarada', 1, 'actor:uno', 'decision:decl-cob', 'recibo:decl-cob', motivo);
    IF revision <> 3 THEN RAISE EXCEPTION 'cobertura declarada incorrecta'; END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-declared', 3,
            'tombstone', NULL, NULL, 'actor:uno', 'decision:decl-tomb', 'recibo:decl-tomb', motivo);
        RAISE EXCEPTION 'tombstone admitio total historico positivo';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-empty', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:empty-des', 'recibo:empty-des', motivo);
    IF revision <> 2 THEN RAISE EXCEPTION 'deshabilitacion vacia incorrecta'; END IF;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-empty', 2,
        'cobertura', 'evidencia:vacia', 0, 'actor:uno', 'decision:empty-cob', 'recibo:empty-cob', motivo);
    IF revision <> 3 THEN RAISE EXCEPTION 'cobertura vacia incorrecta'; END IF;
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-empty', 3,
        'tombstone', NULL, NULL, 'actor:uno', 'decision:empty-tomb', 'recibo:empty-tomb', motivo);
    IF revision <> 4 THEN RAISE EXCEPTION 'tombstone sin usos rechazado'; END IF;
    IF vec_catalogos_configurables.cambiar_proyeccion('cat-empty', 3,
        'tombstone', NULL, NULL, 'actor:dos', 'decision:empty-replay', 'recibo:empty-tomb', motivo_nuevo) <> 4 THEN
        RAISE EXCEPTION 'replay de tombstone vacio incorrecto';
    END IF;

    -- Una publicación posterior no rehabilita una categoría deshabilitada.
    documento_version := pg_catalog.jsonb_build_object('id', 'rpt-version', 'version', 1,
        'estado', 'publicado', 'entradas', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object('clave', 'cat-version', 'etiqueta', 'Categoria versionada')))::text;
    huella_version := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(documento_version, 'UTF8')), 'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-version', 1, huella_version,
        documento_version, '{}'::jsonb, huella_vacias,
        'aprobacion:version-a', 'aprobacion:version-b',
        'actor:uno', 'decision:version-1', 'recibo:version-1', motivo);
    revision := vec_catalogos_configurables.cambiar_proyeccion('cat-version', 1,
        'deshabilitar', NULL, NULL, 'actor:uno', 'decision:version-des', 'recibo:version-des', motivo);
    IF revision <> 2 THEN RAISE EXCEPTION 'deshabilitacion versionada incorrecta'; END IF;
    documento_siguiente := pg_catalog.jsonb_build_object('id', 'rpt-version', 'version', 2,
        'estado', 'publicado', 'entradas', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object('clave', 'cat-version', 'etiqueta', 'Nueva definicion')))::text;
    huella_siguiente := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(documento_siguiente, 'UTF8')), 'hex');
    preimagenes := pg_catalog.jsonb_build_object('cat-version', pg_catalog.jsonb_build_object(
        'version', 1, 'huella_sha256', huella_version, 'revision', 1, 'estado', 'habilitada'));
    huella_preimagenes := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(preimagenes::text, 'UTF8')), 'hex');
    BEGIN
        PERFORM vec_catalogos_configurables.publicar('rpt-version', 2, huella_siguiente,
            documento_siguiente, preimagenes, huella_preimagenes,
            'aprobacion:version-a2', 'aprobacion:version-b2',
            'actor:uno', 'decision:version-2', 'recibo:version-2', motivo);
        RAISE EXCEPTION 'publicacion admitio preimagen de habilitada obsoleta';
    EXCEPTION WHEN SQLSTATE '40001' THEN NULL;
    END;
    preimagenes := pg_catalog.jsonb_build_object('cat-version', pg_catalog.jsonb_build_object(
        'version', 1, 'huella_sha256', huella_version, 'revision', 2, 'estado', 'deshabilitada'));
    huella_preimagenes := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(preimagenes::text, 'UTF8')), 'hex');
    BEGIN
        PERFORM vec_catalogos_configurables.publicar('rpt-version', 2, huella_siguiente,
            documento_siguiente, preimagenes, huella_vacias,
            'aprobacion:version-a2', 'aprobacion:version-b2',
            'actor:uno', 'decision:version-2', 'recibo:version-2', motivo);
        RAISE EXCEPTION 'publicacion admitio huella de preimagen distinta';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.publicar('rpt-version', 2, huella_siguiente,
        documento_siguiente, preimagenes, huella_preimagenes,
        'aprobacion:version-a2', 'aprobacion:version-b2',
        'actor:uno', 'decision:version-2', 'recibo:version-2', motivo);
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('consumidor-demo', 'uso:version',
            'cat-version', 'rpt-version', 2, huella_siguiente,
            'actor:uno', 'decision:version-res', 'recibo:version-res', motivo);
        RAISE EXCEPTION 'publicacion rehabilito categoria deshabilitada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END $prueba$;
RESET ROLE;
DO $persistencia$
DECLARE
    motivo text := 'motivos_rpt:1:motivo_11111111111111111111111111111111';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion WHERE catalogo_id = 'rpt-demo')
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-demo' AND estado = 'deshabilitada'
                         AND revision = 3 AND total_historico_declarado = 1)
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-empty' AND estado = 'tombstone'
                         AND revision = 4 AND total_historico_declarado = 0)
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-declared' AND estado = 'deshabilitada'
                         AND revision = 3 AND total_historico_declarado = 1)
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso
                       WHERE consumidor = 'consumidor-demo' AND uso_ref = 'uso:uno' AND estado = 'confirmado')
       OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control
                       WHERE categoria_id = 'cat-version' AND version = 2
                         AND revision = 3 AND estado = 'deshabilitada')
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.entrada_publicada
                   WHERE categoria_id = 'cat-version' AND definicion ? 'preimagen_control')
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.historia
                   WHERE categoria_id IN ('cat-demo', 'cat-declared') AND accion = 'tombstone') THEN
        RAISE EXCEPTION 'proyección alteró publicación o uso';
    END IF;
    IF (SELECT count(*) FROM vec_catalogos_configurables.historia
         WHERE categoria_id IN ('cat-demo', 'cat-empty', 'cat-declared', 'cat-version')) <> 15
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.historia
                   WHERE categoria_id IN ('cat-demo', 'cat-empty', 'cat-declared', 'cat-version')
                     AND motivo_ref IS DISTINCT FROM motivo) THEN
        RAISE EXCEPTION 'replay añadió historia o sustituyó el motivo original';
    END IF;
    IF EXISTS (
        SELECT 1 FROM (VALUES
            ('cat-demo'::text, 'publicar'::text, NULL::text, 'habilitada'::text),
            ('cat-demo', 'reservar', NULL, 'reservado'),
            ('cat-demo', 'confirmar', 'reservado', 'confirmado'),
            ('cat-demo', 'deshabilitar', 'habilitada', 'deshabilitada'),
            ('cat-demo', 'cobertura', 'deshabilitada', 'deshabilitada'),
            ('cat-declared', 'cobertura', 'deshabilitada', 'deshabilitada'),
            ('cat-empty', 'tombstone', 'deshabilitada', 'tombstone'),
            ('cat-version', 'publicar', 'deshabilitada', 'deshabilitada')
        ) AS esperado(categoria_id, accion, estado_anterior, estado_posterior)
        WHERE NOT EXISTS (
            SELECT 1 FROM vec_catalogos_configurables.historia h
             WHERE h.categoria_id = esperado.categoria_id AND h.accion = esperado.accion
               AND h.estado_anterior IS NOT DISTINCT FROM esperado.estado_anterior
               AND h.estado_posterior = esperado.estado_posterior
               AND h.motivo_ref = motivo
        )
    ) THEN
        RAISE EXCEPTION 'historia omitió motivo o transición de estado';
    END IF;
END $persistencia$;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
DO $inmutabilidad$
BEGIN
    IF (SELECT count(*) FROM pg_catalog.pg_trigger t
         JOIN pg_catalog.pg_class c ON c.oid = t.tgrelid
         JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_catalogos_configurables'
           AND c.relname = ANY(ARRAY['publicacion','entrada_publicada','categoria_control','uso','historia'])
           AND NOT t.tgisinternal AND t.tgenabled <> 'D'
           AND (t.tgtype::integer & 32) = 32) <> 5 THEN
        RAISE EXCEPTION 'faltan barreras BEFORE TRUNCATE';
    END IF;
    BEGIN
        TRUNCATE vec_catalogos_configurables.publicacion,
                 vec_catalogos_configurables.entrada_publicada,
                 vec_catalogos_configurables.categoria_control,
                 vec_catalogos_configurables.uso,
                 vec_catalogos_configurables.historia;
        RAISE EXCEPTION 'TRUNCATE admitido';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_catalogos_configurables.uso
         WHERE consumidor = 'consumidor-demo' AND uso_ref = 'uso:uno';
        RAISE EXCEPTION 'DELETE de uso admitido';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_catalogos_configurables.categoria_control
         WHERE categoria_id = 'cat-empty';
        RAISE EXCEPTION 'DELETE de control admitido';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END $inmutabilidad$;
ROLLBACK;
