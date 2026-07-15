-- Reversion destructiva denegada por defecto. Retirar el esquema exige que el
-- operador aporte en esta sesion la confirmacion literal (por ejemplo mediante
-- PGOPTIONS del runner, nunca incrustada en este fichero), incluso si todas las
-- tablas parecen vacias:
--
--   -c vec.confirmar_destruccion_bolsa_baremacion=DESTRUIR_HISTORIA_BOLSA_BAREMACION_IRREVERSIBLE
--
-- La confirmacion no es un secreto ni sustituye inventario, copia, autorizacion
-- o doble control. Tampoco autoriza a borrar historia: con una sola fila el
-- desmontaje se rechaza, aun cuando el literal sea correcto.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_baremacion:migracion_down:v2',
        0
    )
);

DO $confirmacion_destruccion$
DECLARE
    confirmacion_constante constant text :=
        'DESTRUIR_HISTORIA_BOLSA_BAREMACION_IRREVERSIBLE';
BEGIN
    IF current_setting(
        'vec.confirmar_destruccion_bolsa_baremacion',
        true
    ) IS DISTINCT FROM confirmacion_constante THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down Bolsa rechazado: falta confirmacion destructiva explicita',
            DETAIL = 'la retirada del esquema esta denegada por defecto, incluso sin filas',
            HINT = 'verifique archivo, autorizacion y doble control; configure el opt-in solo para esta sesion';
    END IF;
END
$confirmacion_destruccion$;

-- Los bloqueos impiden que una operacion runtime escriba mientras se construye
-- el inventario destructivo. Se cubren tablas ordinarias y particionadas
-- actuales o futuras; un objeto de otra clase quedara para el DROP SCHEMA
-- RESTRICT final y hara abortar toda la transaccion.
DO $bloquear_relaciones$
DECLARE
    relacion record;
    contiene_filas boolean;
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_baremacion') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down Bolsa rechazado: falta el esquema';
    END IF;
    FOR relacion IN
        SELECT espacio.nspname, clase.relname
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind IN ('r', 'p')
         ORDER BY clase.oid
    LOOP
        EXECUTE format(
            'LOCK TABLE %I.%I IN ACCESS EXCLUSIVE MODE',
            relacion.nspname,
            relacion.relname
        );
    END LOOP;

    -- Solo se inspecciona despues de haber bloqueado el inventario completo.
    -- Los bloqueos permanecen hasta el final de la transaccion, por lo que una
    -- escritura concurrente no puede aparecer entre el preflight y los DROP.
    FOR relacion IN
        SELECT espacio.nspname, clase.relname
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind IN ('r', 'p')
         ORDER BY clase.oid
    LOOP
        EXECUTE format(
            'SELECT EXISTS (SELECT 1 FROM %I.%I LIMIT 1)',
            relacion.nspname,
            relacion.relname
        ) INTO contiene_filas;
        IF contiene_filas THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down Bolsa rechazado: existe historia durable',
                DETAIL = relacion.nspname || '.' || relacion.relname,
                HINT = 'archive mediante un procedimiento independiente; este down nunca elimina filas';
        END IF;
    END LOOP;
END
$bloquear_relaciones$;

-- Se desmontan conjuntamente solo las clases que esta version sabe gobernar y
-- siempre con RESTRICT. Una vista o funcion externa dependiente aborta el DROP
-- y PostgreSQL revierte toda la transaccion; nunca se amplia el radio mediante
-- DROP SCHEMA CASCADE.
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
     WHERE espacio.nspname = 'vec_bolsa_baremacion'
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
     WHERE espacio.nspname = 'vec_bolsa_baremacion'
       AND procedimiento.prokind = 'f';
    IF funciones IS NOT NULL THEN
        EXECUTE 'DROP FUNCTION ' || funciones || ' RESTRICT';
    END IF;
END
$retirar_objetos_conocidos$;

DROP SCHEMA vec_bolsa_baremacion RESTRICT;
COMMIT;
