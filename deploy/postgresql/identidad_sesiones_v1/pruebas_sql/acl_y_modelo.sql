DO $prueba$
DECLARE
    tabla text;
    politica record;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'cuenta', 'alias_hmac_cuenta', 'estado_cuenta',
        'estado_cuenta_actual', 'consumo_asercion'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_class AS definicion
            JOIN pg_catalog.pg_namespace AS espacio
              ON espacio.oid = definicion.relnamespace
            WHERE espacio.nspname = 'vec_identidad_sesiones_v1'
              AND definicion.relname = tabla
              AND definicion.relrowsecurity
              AND definicion.relforcerowsecurity
        ) THEN
            RAISE EXCEPTION 'RLS/FORCE ausente en %', tabla;
        END IF;
        IF has_table_privilege(
            'vec_identidad_sesiones_v1_registrador',
            'vec_identidad_sesiones_v1.' || tabla,
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) OR has_table_privilege(
            'vec_identidad_sesiones_v1_revalidador',
            'vec_identidad_sesiones_v1.' || tabla,
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION 'runtime con privilegio directo en %', tabla;
        END IF;
    END LOOP;

    IF has_schema_privilege('public', 'vec_identidad_sesiones_v1', 'USAGE')
       OR has_function_privilege(
           'public',
           'vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'public',
           'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'PUBLIC conserva capacidad de identidad';
    END IF;
    IF NOT has_function_privilege(
           'vec_identidad_sesiones_v1_registrador',
           'vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_identidad_sesiones_v1_revalidador',
           'vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'capacidades registrar/revalidar mezcladas';
    END IF;
    IF NOT has_function_privilege(
           'vec_identidad_sesiones_v1_revalidador',
           'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_identidad_sesiones_v1_registrador',
           'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_identidad_sesiones_v1_provisionador',
           'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_identidad_sesiones_v1_revocador',
           'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'ACL de revalidacion rica no es exclusiva';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
          JOIN pg_catalog.pg_roles AS propietario
            ON propietario.oid = funcion.proowner
         WHERE espacio.nspname = 'vec_identidad_sesiones_v1'
           AND funcion.proname = 'revalidar_autenticacion_actor_v1'
           AND pg_catalog.pg_get_function_identity_arguments(funcion.oid) =
               'p_autenticacion_ref text, p_sesion_ref text'
           AND funcion.prosecdef
           AND funcion.provolatile = 'v'
           AND propietario.rolname =
               'vec_identidad_sesiones_v1_propietario'
           AND funcion.proconfig @> ARRAY[
               'search_path=pg_catalog, pg_temp'
           ]::text[]
    ) THEN
        RAISE EXCEPTION 'revalidacion rica no esta endurecida';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'vec_identidad_sesiones_v1'
           AND table_name = 'consumo_asercion'
           AND column_name = 'control_sesion_revision'
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'vec_identidad_sesiones_v1'
           AND table_name = 'consumo_asercion'
           AND column_name IN ('cuenta_revision', 'cuenta_ordinaria_revision')
        GROUP BY table_schema, table_name HAVING count(*) = 2
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'vec_identidad_sesiones_v1'
           AND table_name = 'alias_hmac_cuenta'
           AND column_name = 'dominio_hmac_ref'
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'vec_identidad_sesiones_v1'
           AND table_name = 'consumo_asercion'
           AND column_name = 'autenticacion_huella_sha256'
    ) THEN
        RAISE EXCEPTION 'modelo de ligadura incompleto';
    END IF;

    IF vec_identidad_sesiones_v1.huella_hmac_valida(
           decode(repeat('00', 32), 'hex')
       ) IS NOT FALSE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(
           decode(repeat('01', 32), 'hex')
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'validador SQL de huellas HMAC incompleto';
    END IF;

    IF (SELECT count(*) FROM pg_catalog.pg_constraint AS restriccion
         JOIN pg_catalog.pg_class AS tabla_def
           ON tabla_def.oid = restriccion.conrelid
         JOIN pg_catalog.pg_namespace AS espacio
           ON espacio.oid = tabla_def.relnamespace
        WHERE espacio.nspname = 'vec_identidad_sesiones_v1'
          AND tabla_def.relname = 'consumo_asercion'
          AND restriccion.contype = 'f') < 7 THEN
        RAISE EXCEPTION 'faltan FKs compuestas de consumo';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS restriccion
          JOIN pg_catalog.pg_class AS tabla_def
            ON tabla_def.oid = restriccion.conrelid
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla_def.relnamespace
         WHERE espacio.nspname = 'vec_identidad_sesiones_v1'
           AND tabla_def.relname = 'consumo_asercion'
           AND restriccion.contype = 'u'
           AND pg_catalog.pg_get_constraintdef(restriccion.oid) =
               'UNIQUE (dominio_hmac_ref, autenticacion_huella_sha256)'
    ) THEN
        RAISE EXCEPTION 'replay exacto depende de la version HMAC';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_index AS indice
         WHERE indice.indexrelid = pg_catalog.to_regclass(
                   'vec_identidad_sesiones_v1.consumo_asercion_sesion_hmac_v1_idx'
               )
           AND indice.indrelid = pg_catalog.to_regclass(
                   'vec_identidad_sesiones_v1.consumo_asercion'
           )
           AND indice.indisvalid
           AND indice.indisready
           AND ARRAY(
               SELECT atributo.attname::text
                 FROM unnest(indice.indkey) WITH ORDINALITY AS clave(
                     numero_atributo, posicion
                 )
                 JOIN pg_catalog.pg_attribute AS atributo
                   ON atributo.attrelid = indice.indrelid
                  AND atributo.attnum = clave.numero_atributo
                ORDER BY clave.posicion
           ) = ARRAY[
               'esquema_hmac', 'dominio_hmac_ref', 'clave_hmac_id',
               'clave_hmac_version', 'sesion_id_hmac'
           ]::text[]
    ) THEN
        RAISE EXCEPTION 'falta indice para excluir sesiones HMAC activas';
    END IF;
END
$prueba$;
