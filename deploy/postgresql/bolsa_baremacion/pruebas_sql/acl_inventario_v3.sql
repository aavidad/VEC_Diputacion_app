-- Inventario cerrado de privilegios tras instalar 000005. Una funcion futura
-- no entra en runtime por accidente: cualquier superficie nueva rompe el test.
BEGIN;
SET LOCAL search_path = pg_catalog;

DO $inventario_acl_v3$
DECLARE
    esquema_oid oid;
    propietario_oid oid;
    rol text;
    actuales oid[];
    esperadas oid[];
    fachadas_v3 oid[];
    tablas_v3 constant text[] := ARRAY[
        'manifiesto_probatorio_v3',
        'manifiesto_autorizacion_v3',
        'manifiesto_evidencia_v3',
        'prevalidacion_archivo_probatorio_v3',
        'resultado_prevalidacion_archivo_v3'
    ];
BEGIN
    SELECT espacio.oid
      INTO STRICT esquema_oid
      FROM pg_namespace AS espacio
     WHERE espacio.nspname = 'vec_bolsa_baremacion';
    SELECT identidad.oid
      INTO STRICT propietario_oid
      FROM pg_roles AS identidad
     WHERE identidad.rolname = 'vec_bolsa_baremacion_propietario';

    IF EXISTS (
        SELECT 1
          FROM pg_proc AS funcion
          CROSS JOIN LATERAL aclexplode(COALESCE(
              funcion.proacl, acldefault('f', funcion.proowner)
          )) AS privilegio
         WHERE funcion.pronamespace = esquema_oid
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'PUBLIC conserva EXECUTE en el esquema Bolsa';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM aclexplode(COALESCE(
              (SELECT espacio.nspacl
                 FROM pg_namespace AS espacio
                WHERE espacio.oid = esquema_oid),
              acldefault('n', propietario_oid)
          )) AS privilegio
         WHERE privilegio.grantee = 0
    ) OR has_schema_privilege(
           'vec_bolsa_baremacion_ejecutor', esquema_oid, 'USAGE'
       ) IS NOT TRUE
       OR has_schema_privilege(
           'vec_bolsa_baremacion_ejecutor', esquema_oid, 'CREATE'
       )
       OR has_schema_privilege(
           'vec_bolsa_baremacion_lector_outbox', esquema_oid, 'USAGE'
       ) IS NOT TRUE
       OR has_schema_privilege(
           'vec_bolsa_baremacion_lector_outbox', esquema_oid, 'CREATE'
       )
       OR has_schema_privilege(
           'vec_bolsa_baremacion_registrador_atestacion',
           esquema_oid, 'USAGE'
       )
       OR has_schema_privilege(
           'vec_bolsa_baremacion_registrador_atestacion',
           esquema_oid, 'CREATE'
       ) THEN
        RAISE EXCEPTION 'ACL de esquema runtime inesperada';
    END IF;

    IF (SELECT count(*)
          FROM pg_class AS relacion
         WHERE relacion.relnamespace = esquema_oid
           AND relacion.relname = ANY (tablas_v3)
           AND relacion.relkind IN ('r', 'p')
           AND relacion.relowner = propietario_oid
           AND relacion.relrowsecurity
           AND relacion.relforcerowsecurity) <> 5
       OR (SELECT count(*)
             FROM pg_policy AS politica
             JOIN pg_class AS relacion ON relacion.oid = politica.polrelid
            WHERE relacion.relnamespace = esquema_oid
              AND relacion.relname = ANY (tablas_v3)
              AND politica.polname = 'acceso_propietario_exacto'
              AND politica.polpermissive
              AND politica.polcmd = '*'
              AND politica.polroles = ARRAY[propietario_oid]
              AND politica.polqual IS NOT NULL
              AND politica.polwithcheck IS NOT NULL
              AND pg_get_expr(politica.polqual, politica.polrelid) =
                  '(CURRENT_USER = ''vec_bolsa_baremacion_propietario''::name)'
              AND pg_get_expr(politica.polwithcheck, politica.polrelid)
                    IS NOT DISTINCT FROM
                  pg_get_expr(politica.polqual, politica.polrelid)) <> 5
       OR EXISTS (
            SELECT 1
              FROM pg_class AS relacion
             WHERE relacion.relnamespace = esquema_oid
               AND relacion.relname = ANY (tablas_v3)
               AND NOT EXISTS (
                   SELECT 1
                     FROM pg_policy AS politica
                    WHERE politica.polrelid = relacion.oid
               )
               OR relacion.relnamespace = esquema_oid
                  AND relacion.relname = ANY (tablas_v3)
                  AND (SELECT count(*) FROM pg_policy AS politica
                        WHERE politica.polrelid = relacion.oid) <> 1
       ) THEN
        RAISE EXCEPTION 'RLS/politica exacta incompleta en las cinco tablas V3';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion_lector_outbox',
        'vec_bolsa_baremacion_registrador_atestacion'
    ] LOOP
        IF EXISTS (
            SELECT 1
              FROM pg_class AS relacion
             WHERE relacion.relnamespace = esquema_oid
               AND relacion.relkind IN ('r', 'p', 'v', 'm', 'f')
               AND (
                   has_table_privilege(rol, relacion.oid, 'SELECT')
                   OR has_table_privilege(rol, relacion.oid, 'INSERT')
                   OR has_table_privilege(rol, relacion.oid, 'UPDATE')
                   OR has_table_privilege(rol, relacion.oid, 'DELETE')
                   OR has_table_privilege(rol, relacion.oid, 'TRUNCATE')
                   OR has_table_privilege(rol, relacion.oid, 'REFERENCES')
                   OR has_table_privilege(rol, relacion.oid, 'TRIGGER')
               )
        ) OR EXISTS (
            SELECT 1
              FROM pg_class AS relacion
              JOIN pg_attribute AS columna
                ON columna.attrelid = relacion.oid
               AND columna.attnum > 0
               AND NOT columna.attisdropped
             WHERE relacion.relnamespace = esquema_oid
               AND relacion.relkind IN ('r', 'p', 'v', 'm', 'f')
               AND (
                   has_column_privilege(
                       rol, relacion.oid, columna.attnum, 'SELECT'
                   ) OR has_column_privilege(
                       rol, relacion.oid, columna.attnum, 'INSERT'
                   ) OR has_column_privilege(
                       rol, relacion.oid, columna.attnum, 'UPDATE'
                   ) OR has_column_privilege(
                       rol, relacion.oid, columna.attnum, 'REFERENCES'
                   )
               )
        ) OR EXISTS (
            SELECT 1
              FROM pg_class AS secuencia
             WHERE secuencia.relnamespace = esquema_oid
               AND secuencia.relkind = 'S'
               AND (
                   has_sequence_privilege(rol, secuencia.oid, 'USAGE')
                   OR has_sequence_privilege(rol, secuencia.oid, 'SELECT')
                   OR has_sequence_privilege(rol, secuencia.oid, 'UPDATE')
               )
        ) OR EXISTS (
            SELECT 1
              FROM pg_type AS tipo
             WHERE tipo.typnamespace = esquema_oid
               AND has_type_privilege(rol, tipo.oid, 'USAGE')
        ) THEN
            RAISE EXCEPTION
                'el rol runtime % conserva ACL directa de objeto', rol;
        END IF;
    END LOOP;

    fachadas_v3 := ARRAY[
        to_regprocedure('vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea)')::oid,
        to_regprocedure('vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(jsonb,jsonb,bytea,bytea)')::oid,
        to_regprocedure('vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)')::oid,
        to_regprocedure('vec_bolsa_baremacion.obtener_version_vigente_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea)')::oid,
        to_regprocedure('vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea)')::oid,
        to_regprocedure('vec_bolsa_baremacion.obtener_evidencia_transaccion_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea)')::oid
    ];
    esperadas := array_prepend(
        to_regprocedure('vec_bolsa_baremacion.abandonar_reserva(jsonb,jsonb,bytea,bytea)')::oid,
        fachadas_v3
    );
    SELECT array_agg(funcion.oid ORDER BY funcion.oid)
      INTO actuales
      FROM pg_proc AS funcion
     WHERE funcion.pronamespace = esquema_oid
       AND has_function_privilege(
           'vec_bolsa_baremacion_ejecutor', funcion.oid, 'EXECUTE'
       );
    SELECT array_agg(oid ORDER BY oid) INTO esperadas
      FROM unnest(esperadas) AS inventario(oid);
    IF cardinality(esperadas) <> 7 OR array_position(esperadas, NULL) IS NOT NULL
       OR actuales IS DISTINCT FROM esperadas THEN
        RAISE EXCEPTION
            'superficie ejecutor inesperada: %, esperada %',
            actuales, esperadas;
    END IF;

    IF (SELECT count(*)
          FROM pg_proc AS funcion
         WHERE funcion.oid = ANY (fachadas_v3)
           AND funcion.proowner = propietario_oid
           AND funcion.prosecdef
           AND funcion.proconfig =
               ARRAY['search_path=pg_catalog, pg_temp']) <> 6 THEN
        RAISE EXCEPTION
            'owner/SECURITY DEFINER/search_path de fachadas V3 divergente';
    END IF;

    esperadas := ARRAY[
        to_regprocedure('vec_bolsa_baremacion.reclamar_evento_outbox(text,bytea,integer)')::oid,
        to_regprocedure('vec_bolsa_baremacion.finalizar_entrega_outbox(text,text,bytea,text,text)')::oid
    ];
    SELECT array_agg(funcion.oid ORDER BY funcion.oid)
      INTO actuales
      FROM pg_proc AS funcion
     WHERE funcion.pronamespace = esquema_oid
       AND has_function_privilege(
           'vec_bolsa_baremacion_lector_outbox', funcion.oid, 'EXECUTE'
       );
    SELECT array_agg(oid ORDER BY oid) INTO esperadas
      FROM unnest(esperadas) AS inventario(oid);
    IF cardinality(esperadas) <> 2 OR array_position(esperadas, NULL) IS NOT NULL
       OR actuales IS DISTINCT FROM esperadas THEN
        RAISE EXCEPTION
            'superficie lector outbox inesperada: %, esperada %',
            actuales, esperadas;
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_proc AS funcion
         WHERE funcion.pronamespace = esquema_oid
           AND has_function_privilege(
               'vec_bolsa_baremacion_registrador_atestacion',
               funcion.oid, 'EXECUTE'
           )
    ) THEN
        RAISE EXCEPTION 'el registrador reservado obtuvo una funcion';
    END IF;

    IF has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.reservar_cambio(jsonb,jsonb,bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.obtener_version_vigente(jsonb,jsonb,bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.obtener_version(jsonb,jsonb,bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.obtener_evidencia_transaccion(jsonb,jsonb,bytea,bytea)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'una de las cinco fachadas V1 sensibles sigue abierta';
    END IF;
END
$inventario_acl_v3$;

ROLLBACK;
