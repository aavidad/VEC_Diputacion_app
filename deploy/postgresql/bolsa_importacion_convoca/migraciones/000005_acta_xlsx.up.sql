\set ON_ERROR_STOP on
-- Admite el nombre .xlsx en el acta Convoca sin cambiar su contrato restante.
-- Requiere la función acta_valida de 000001 y conserva su identidad y permisos.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);

DO $migracion$
DECLARE
    funcion oid := pg_catalog.to_regprocedure(
        'vec_bolsa_importacion_convoca.acta_valida(jsonb)'
    );
    anterior text;
    siguiente text;
    actual text;
    metadatos jsonb;
    metadatos_actuales jsonb;
    dependencias jsonb;
    dependencias_actuales jsonb;
    propietario oid;
    permisos aclitem[];
    configuracion text[];
    definidora boolean;
    volatilidad "char";
    huella_actual text;
    huella_esperada text :=
        '40c26d1857a8221c261f4877c95fc1e98a4cb3bef38d5bc43ccd49185e9c9f89';
    ocurrencias integer;
    marca text := $marca$       OR pg_catalog.lower(pg_catalog.right(p_acta->>'nombre_fichero', 4))
          IS DISTINCT FROM '.xls'$marca$;
    reemplazo text := $nuevo$       OR (
           pg_catalog.lower(pg_catalog.right(p_acta->>'nombre_fichero', 4))
               IS DISTINCT FROM '.xls'
           AND (
               pg_catalog.lower(pg_catalog.right(p_acta->>'nombre_fichero', 5))
                   IS DISTINCT FROM '.xlsx'
               OR pg_catalog.octet_length(p_acta->>'nombre_fichero') <= 5
           )
       )$nuevo$;
BEGIN
    IF funcion IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: función ausente',
            DETAIL = 'clave=acta_valida(jsonb); actual=ausente; esperado=presente';
    END IF;

    SELECT pg_catalog.pg_get_functiondef(p.oid),
           pg_catalog.to_jsonb(p) - 'prosrc',
           p.proowner, p.proacl, p.proconfig, p.prosecdef, p.provolatile
      INTO STRICT anterior, metadatos, propietario, permisos, configuracion,
                  definidora, volatilidad
      FROM pg_catalog.pg_proc AS p
     WHERE p.oid = funcion;
    SELECT coalesce(
        pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
            ORDER BY d.classid, d.objid, d.objsubid,
                     d.refclassid, d.refobjid, d.refobjsubid, d.deptype),
        '[]'::jsonb
    ) INTO dependencias
      FROM pg_catalog.pg_depend AS d
     WHERE d.classid = 'pg_catalog.pg_proc'::pg_catalog.regclass
       AND d.objid = funcion;

    IF current_user <> 'vec_bolsa_importacion_convoca_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: rol incompatible',
            DETAIL = pg_catalog.format('clave=current_user; actual=%s; esperado=%s',
                current_user, 'vec_bolsa_importacion_convoca_propietario');
    END IF;
    IF propietario IS DISTINCT FROM
       'vec_bolsa_importacion_convoca_propietario'::pg_catalog.regrole THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: propietario incompatible',
            DETAIL = pg_catalog.format('clave=proowner; actual=%s; esperado=%s',
                propietario::pg_catalog.regrole,
                'vec_bolsa_importacion_convoca_propietario');
    END IF;
    IF permisos IS DISTINCT FROM ARRAY[
        'vec_bolsa_importacion_convoca_propietario=X/vec_bolsa_importacion_convoca_propietario'
    ]::aclitem[] THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: permisos incompatibles',
            DETAIL = pg_catalog.format('clave=proacl; actual=%s; esperado=%s',
                permisos, ARRAY[
                    'vec_bolsa_importacion_convoca_propietario=X/vec_bolsa_importacion_convoca_propietario'
                ]::aclitem[]);
    END IF;
    IF configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog'] THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: configuración incompatible',
            DETAIL = pg_catalog.format('clave=proconfig; actual=%s; esperado=%s',
                configuracion, ARRAY['search_path=pg_catalog']);
    END IF;
    IF definidora IS DISTINCT FROM false OR volatilidad IS DISTINCT FROM 'i' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: atributos incompatibles',
            DETAIL = pg_catalog.format(
                'clave=prosecdef/provolatile; actual=%s/%s; esperado=false/i',
                definidora, volatilidad);
    END IF;
    huella_actual := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(anterior, 'UTF8')), 'hex');
    IF huella_actual IS DISTINCT FROM huella_esperada THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: definición incompatible',
            DETAIL = pg_catalog.format('clave=pg_get_functiondef.sha256; actual=%s; esperado=%s',
                huella_actual, huella_esperada);
    END IF;
    ocurrencias := (pg_catalog.length(anterior) -
        pg_catalog.length(pg_catalog.replace(anterior, marca, '')))
        / pg_catalog.length(marca);
    IF ocurrencias IS DISTINCT FROM 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: marca incompatible',
            DETAIL = pg_catalog.format('clave=marca.ocurrencias; actual=%s; esperado=1',
                ocurrencias);
    END IF;

    siguiente := pg_catalog.replace(anterior, marca, reemplazo);
    EXECUTE siguiente;
    SELECT pg_catalog.pg_get_functiondef(funcion) INTO STRICT actual;
    SELECT pg_catalog.to_jsonb(p) - 'prosrc' INTO STRICT metadatos_actuales
      FROM pg_catalog.pg_proc AS p WHERE p.oid = funcion;
    SELECT coalesce(
            pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
                ORDER BY d.classid, d.objid, d.objsubid,
                         d.refclassid, d.refobjid, d.refobjsubid, d.deptype),
            '[]'::jsonb) INTO dependencias_actuales
      FROM pg_catalog.pg_depend AS d
     WHERE d.classid = 'pg_catalog.pg_proc'::pg_catalog.regclass
       AND d.objid = funcion;
    IF actual IS DISTINCT FROM siguiente
       OR pg_catalog.replace(actual, reemplazo, marca) IS DISTINCT FROM anterior THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: definición posterior incompatible',
            DETAIL = pg_catalog.format('clave=pg_get_functiondef.sha256; actual=%s; esperado=%s',
                pg_catalog.encode(pg_catalog.sha256(
                    pg_catalog.convert_to(actual, 'UTF8')), 'hex'),
                pg_catalog.encode(pg_catalog.sha256(
                    pg_catalog.convert_to(siguiente, 'UTF8')), 'hex'));
    END IF;
    IF metadatos_actuales IS DISTINCT FROM metadatos THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: metadatos posteriores incompatibles',
            DETAIL = pg_catalog.format('clave=pg_proc_sin_prosrc.sha256; actual=%s; esperado=%s',
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    metadatos_actuales::text, 'UTF8')), 'hex'),
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    metadatos::text, 'UTF8')), 'hex'));
    END IF;
    IF dependencias_actuales IS DISTINCT FROM dependencias THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Convoca 000005: dependencias posteriores incompatibles',
            DETAIL = pg_catalog.format('clave=pg_depend.sha256; actual=%s; esperado=%s',
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    dependencias_actuales::text, 'UTF8')), 'hex'),
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    dependencias::text, 'UTF8')), 'hex'));
    END IF;
END
$migracion$;
COMMIT;
