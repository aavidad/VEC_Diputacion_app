\set ON_ERROR_STOP on

BEGIN;

SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $compatibilidad_barreras$
DECLARE
    cambio record;
    definicion text;
    apariciones integer;
    oid_antes oid;
    oid_despues oid;
BEGIN
    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema BETWEEN 15 AND 25
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'barrera acumulativa de cobertura no disponible';
    END IF;

    FOR cambio IN
        SELECT *
          FROM (VALUES
            ('vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)',
             'version_esquema IN (2, 3)',
             'version_esquema BETWEEN 2 AND 25'),
            ('vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)',
             '''correlacion:vec:cobertura:''',
             '''correlacion_'''),
            ('vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)',
             'version_esquema = 3',
             'version_esquema BETWEEN 3 AND 25'),
            ('vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)',
             'version_esquema = 3',
             'version_esquema BETWEEN 3 AND 25'),
            ('vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)',
             'version_esquema=14',
             'version_esquema BETWEEN 14 AND 25'),
            ('vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)',
             'version_esquema=14',
             'version_esquema BETWEEN 14 AND 25'),
            ('vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)',
             'version_esquema=15',
             'version_esquema BETWEEN 15 AND 25')
          ) AS cambios(firma, anterior, nueva)
    LOOP
        oid_antes := pg_catalog.to_regprocedure(cambio.firma);
        IF oid_antes IS NULL THEN
            RAISE EXCEPTION 'función de cobertura ausente: %', cambio.firma;
        END IF;
        SELECT pg_catalog.pg_get_functiondef(oid_antes)
          INTO STRICT definicion;
        IF pg_catalog.strpos(definicion, cambio.nueva) = 0 THEN
            apariciones := (
                pg_catalog.length(definicion) -
                pg_catalog.length(
                    pg_catalog.replace(definicion, cambio.anterior, '')
                )
            ) / pg_catalog.length(cambio.anterior);
            IF apariciones <> 1 THEN
                RAISE EXCEPTION
                    'definición inesperada para %: apariciones=%',
                    cambio.firma, apariciones;
            END IF;
            EXECUTE pg_catalog.replace(
                definicion, cambio.anterior, cambio.nueva
            );
        END IF;
        oid_despues := pg_catalog.to_regprocedure(cambio.firma);
        IF oid_despues IS DISTINCT FROM oid_antes THEN
            RAISE EXCEPTION 'identidad de función alterada: %', cambio.firma;
        END IF;
        SELECT pg_catalog.pg_get_functiondef(oid_despues)
          INTO STRICT definicion;
        IF pg_catalog.strpos(definicion, cambio.nueva) = 0 THEN
            RAISE EXCEPTION 'barrera no actualizada: %', cambio.firma;
        END IF;
    END LOOP;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc p
          JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
          JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
          JOIN pg_catalog.pg_language l ON l.oid = p.prolang
         WHERE n.nspname = 'vec_contratacion_temporal'
           AND p.proname IN (
             'preparar_operacion_decision_cobertura_v1',
             'consultar_operacion_decision_cobertura_confirmada_v1',
             'leer_terminal_primario_decision_cobertura_o404c_v1',
             'confirmar_operacion_decision_cobertura_o404e_v1',
             'leer_terminal_primario_decision_cobertura_o404e_v1',
             'recuperar_resultado_propio_decision_cobertura_o405_v1'
           )
           AND (
             r.rolname <> 'vec_contratacion_temporal_propietario'
             OR NOT p.prosecdef
             OR l.lanname <> 'plpgsql'
             OR pg_catalog.has_function_privilege(
                  'public', p.oid, 'EXECUTE'
                )
           )
    ) THEN
        RAISE EXCEPTION 'seguridad de funciones de cobertura alterada';
    END IF;
END
$compatibilidad_barreras$;

COMMIT;
