\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL statement_timeout = '30s';

DO $prueba$
DECLARE
    acta jsonb;
    nombre text;
    funcion oid := to_regprocedure(
        'vec_bolsa_importacion_convoca.acta_valida(jsonb)'
    );
BEGIN
    IF funcion IS NULL THEN
        RAISE EXCEPTION 'Convoca 000005: falta acta_valida';
    END IF;
    SELECT acta_canonica INTO acta
      FROM vec_bolsa_importacion_convoca.lote
     ORDER BY importacion_ref LIMIT 1;
    IF acta IS NULL THEN
        RAISE EXCEPTION 'Convoca 000005: falta el lote sintético del clon';
    END IF;

    FOREACH nombre IN ARRAY ARRAY[
        'muestra.xls', 'muestra.XLS', 'muestra.xlsx', 'muestra.XLSX'
    ] LOOP
        IF vec_bolsa_importacion_convoca.acta_valida(
            jsonb_set(acta, '{nombre_fichero}', to_jsonb(nombre))
        ) IS NOT TRUE THEN
            RAISE EXCEPTION 'Convoca 000005: rechazó nombre válido %', nombre;
        END IF;
    END LOOP;

    FOREACH nombre IN ARRAY ARRAY[
        '.xlsx', '.XLSX', 'muestra.xlsm', 'muestra.xlsx/', 'ruta/muestra.xlsx',
        'ruta\muestra.xlsx', ' muestra.xlsx', 'muestra.xlsx ',
        'muestra' || chr(10) || '.xlsx'
    ] LOOP
        IF vec_bolsa_importacion_convoca.acta_valida(
            jsonb_set(acta, '{nombre_fichero}', to_jsonb(nombre))
        ) IS NOT FALSE THEN
            RAISE EXCEPTION 'Convoca 000005: admitió nombre inválido %', nombre;
        END IF;
    END LOOP;

    IF vec_bolsa_importacion_convoca.acta_valida(
        jsonb_set(acta, '{nombre_fichero}', 'null'::jsonb)
    ) IS NOT FALSE
       OR vec_bolsa_importacion_convoca.acta_valida(
           jsonb_set(acta, '{actor_ref}', '""'::jsonb)
       ) IS NOT FALSE
       OR EXISTS (
           SELECT 1 FROM vec_bolsa_importacion_convoca.lote
            WHERE vec_bolsa_importacion_convoca.acta_valida(acta_canonica)
                  IS NOT TRUE
       ) THEN
        RAISE EXCEPTION 'Convoca 000005: cambió otra guarda del acta';
    END IF;

    IF (SELECT p.proowner FROM pg_proc AS p WHERE p.oid = funcion)
       IS DISTINCT FROM
       'vec_bolsa_importacion_convoca_propietario'::regrole
       OR (SELECT p.proacl FROM pg_proc AS p WHERE p.oid = funcion)
          IS DISTINCT FROM ARRAY[
              'vec_bolsa_importacion_convoca_propietario=X/vec_bolsa_importacion_convoca_propietario'
          ]::aclitem[]
       OR (SELECT p.proconfig FROM pg_proc AS p WHERE p.oid = funcion)
          IS DISTINCT FROM ARRAY['search_path=pg_catalog']
       OR (SELECT p.prosecdef FROM pg_proc AS p WHERE p.oid = funcion)
          IS DISTINCT FROM false THEN
        RAISE EXCEPTION 'Convoca 000005: cambió la autoridad de la función';
    END IF;
END
$prueba$;
ROLLBACK;
