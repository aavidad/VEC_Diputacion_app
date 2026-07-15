-- Reversion destructiva denegada por defecto. Retirar el esquema siempre exige
-- que el operador aporte en esta sesion la confirmacion literal (por ejemplo
-- mediante PGOPTIONS del runner, nunca incrustada en este fichero), incluso
-- cuando la instalacion parezca vacia:
--
--   -c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE
--
-- La confirmacion no es un secreto ni sustituye inventario, copia, autorizacion
-- o doble control. Exigirla incondicionalmente evita que una tabla incorporada
-- por una migracion futura eluda una lista estatica de evidencia conocida.
BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_ejecucion_documental_v4:migracion_down:v2',
        0
    )
);

DO $confirmacion_destruccion$
DECLARE
    confirmacion_constante constant text :=
        'DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE';
BEGIN
    IF current_setting(
        'vec.confirmar_destruccion_ejecucion_documental_v4',
        true
    ) IS DISTINCT FROM confirmacion_constante THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down V4 rechazado: falta confirmacion destructiva explicita',
            DETAIL = 'la retirada del esquema esta denegada por defecto, incluso sin filas',
            HINT = 'verifique copia y autorizacion; configure el opt-in literal solo para esta sesion';
    END IF;
END
$confirmacion_destruccion$;

-- No se mantiene una lista estatica: se bloquea toda tabla ordinaria o
-- particionada que exista en el esquema en el momento de la retirada. El opt-in
-- anterior sigue siendo la barrera autoritativa ante objetos creados en el
-- futuro o DDL concurrente.
DO $bloquear_relaciones$
DECLARE
    relacion record;
BEGIN
    FOR relacion IN
        SELECT espacio.nspname, clase.relname
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND clase.relkind IN ('r', 'p')
         ORDER BY clase.oid
    LOOP
        EXECUTE format(
            'LOCK TABLE %I.%I IN ACCESS EXCLUSIVE MODE',
            relacion.nspname,
            relacion.relname
        );
    END LOOP;
END
$bloquear_relaciones$;

-- Se eliminan conjuntamente solo las clases de objeto que esta version sabe
-- desmontar y siempre con RESTRICT. Una vista o funcion externa que dependa de
-- V4 aborta la transaccion; un tipo de objeto futuro que este down no conozca
-- queda dentro del esquema y hace fallar el DROP SCHEMA final. Nunca se expande
-- silenciosamente el radio destructivo a otro modulo mediante CASCADE.
DO $retirar_objetos_conocidos$
DECLARE
    tablas text;
    funciones text;
BEGIN
    SELECT string_agg(
               format('%I.%I', espacio.nspname, clase.relname),
               ', ' ORDER BY clase.oid
           )
      INTO tablas
      FROM pg_catalog.pg_class AS clase
      JOIN pg_catalog.pg_namespace AS espacio
        ON espacio.oid = clase.relnamespace
     WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
       AND clase.relkind IN ('r', 'p');
    IF tablas IS NOT NULL THEN
        EXECUTE 'DROP TABLE ' || tablas || ' RESTRICT';
    END IF;

    SELECT string_agg(
               format(
                   '%I.%I(%s)',
                   espacio.nspname,
                   procedimiento.proname,
                   pg_catalog.pg_get_function_identity_arguments(
                       procedimiento.oid
                   )
               ),
               ', ' ORDER BY procedimiento.oid
           )
      INTO funciones
      FROM pg_catalog.pg_proc AS procedimiento
      JOIN pg_catalog.pg_namespace AS espacio
        ON espacio.oid = procedimiento.pronamespace
     WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
       AND procedimiento.prokind = 'f';
    IF funciones IS NOT NULL THEN
        EXECUTE 'DROP FUNCTION ' || funciones || ' RESTRICT';
    END IF;
END
$retirar_objetos_conocidos$;

DROP SCHEMA vec_ejecucion_documental_v4 RESTRICT;
COMMIT;
