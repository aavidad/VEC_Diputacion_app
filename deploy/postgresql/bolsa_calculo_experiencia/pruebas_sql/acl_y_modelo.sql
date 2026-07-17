DO $inventario$
DECLARE
    rol_aplicacion text := 'vec_bolsa_calculo_experiencia_aplicacion';
    tabla text;
BEGIN
    IF has_schema_privilege(
           'public', 'vec_bolsa_calculo_experiencia', 'USAGE'
       ) THEN
        RAISE EXCEPTION 'PUBLIC conserva USAGE del esquema';
    END IF;
    IF has_table_privilege(
           rol_aplicacion,
           'vec_bolsa_calculo_experiencia.configuracion_tenant',
           'INSERT,UPDATE,DELETE,TRUNCATE'
       )
       OR NOT has_column_privilege(
           rol_aplicacion,
           'vec_bolsa_calculo_experiencia.configuracion_tenant',
           'tenant_id', 'SELECT'
       ) THEN
        RAISE EXCEPTION 'la aplicacion puede alterar el tenant instalado';
    END IF;
    FOREACH tabla IN ARRAY ARRAY[
        'resultado_oficial', 'intento', 'consumo_autorizaciones',
        'recibo', 'auditoria', 'outbox'
    ] LOOP
        IF NOT has_table_privilege(
                   rol_aplicacion,
                   'vec_bolsa_calculo_experiencia.' || tabla,
                   'SELECT,INSERT'
               )
           OR has_table_privilege(
                   rol_aplicacion,
                   'vec_bolsa_calculo_experiencia.' || tabla,
                   'UPDATE,DELETE,TRUNCATE'
               )
           OR has_table_privilege(
                   'public',
                   'vec_bolsa_calculo_experiencia.' || tabla,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
               ) THEN
            RAISE EXCEPTION 'ACL de tabla incorrecta: %', tabla;
        END IF;
    END LOOP;

    IF has_column_privilege(
           'vec_bolsa_calculo_experiencia_lector_operativo',
           'vec_bolsa_calculo_experiencia.resultado_oficial',
           'resultado_canonico', 'SELECT'
       )
       OR has_column_privilege(
           'vec_bolsa_calculo_experiencia_lector_operativo',
           'vec_bolsa_calculo_experiencia.recibo',
           'recibo_canonico', 'SELECT'
       )
       OR has_column_privilege(
           'vec_bolsa_calculo_experiencia_lector_operativo',
           'vec_bolsa_calculo_experiencia.resultado_oficial',
           'selector_fuente_canonico', 'SELECT'
       )
       OR NOT has_column_privilege(
           'vec_bolsa_calculo_experiencia_lector_operativo',
           'vec_bolsa_calculo_experiencia.resultado_oficial',
           'huella_resultado_sha256', 'SELECT'
       ) THEN
        RAISE EXCEPTION 'el lector operativo no esta minimizado';
    END IF;
    IF NOT has_table_privilege(
           'vec_bolsa_calculo_experiencia_publicador',
           'vec_bolsa_calculo_experiencia.outbox', 'SELECT'
       )
       OR has_table_privilege(
           'vec_bolsa_calculo_experiencia_publicador',
           'vec_bolsa_calculo_experiencia.outbox',
           'INSERT,UPDATE,DELETE,TRUNCATE'
       )
       OR has_table_privilege(
           'vec_bolsa_calculo_experiencia_publicador',
           'vec_bolsa_calculo_experiencia.resultado_oficial', 'SELECT'
       ) THEN
        RAISE EXCEPTION 'ACL del publicador incorrecta';
    END IF;
END
$inventario$;

DO $rls_y_funciones$
DECLARE
    frontera regprocedure :=
        'vec_autorizacion.revalidar_decision_calculo_experiencia_v1(text,text,text,text,text,text,text,text)'::regprocedure;
    registrador regprocedure :=
        'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)'::regprocedure;
    validador_sujeto regprocedure :=
        'vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido(text)'::regprocedure;
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_class AS tabla
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_calculo_experiencia'
           AND tabla.relkind = 'r'
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity) <> 6 THEN
        RAISE EXCEPTION 'inventario RLS incompleto';
    END IF;
    IF (SELECT count(*)
          FROM pg_catalog.pg_trigger AS disparador
          JOIN pg_catalog.pg_class AS tabla
            ON tabla.oid = disparador.tgrelid
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_calculo_experiencia'
           AND disparador.tgname = 'impedir_truncado'
           AND NOT disparador.tgisinternal) <> 7 THEN
        RAISE EXCEPTION 'barrera de TRUNCATE incompleta';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_bolsa_calculo_experiencia'
           AND funcion.prosecdef
    ) THEN
        RAISE EXCEPTION 'el esquema de resultados contiene SECURITY DEFINER';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = frontera
           AND prosecdef
           AND proconfig = ARRAY[
               'search_path=pg_catalog, pg_temp'
           ]::text[]
    )
       OR has_function_privilege('public', frontera, 'EXECUTE')
       OR NOT has_function_privilege(
           'vec_bolsa_calculo_experiencia_aplicacion', frontera, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'frontera V2 abierta o inutilizable';
    END IF;
    IF has_function_privilege(
           'vec_bolsa_calculo_experiencia_aplicacion', registrador, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el runtime recibio la puerta V2 no atestada';
    END IF;
    IF has_function_privilege('public', validador_sujeto, 'EXECUTE')
       OR NOT has_function_privilege(
           'vec_bolsa_calculo_experiencia_aplicacion',
           validador_sujeto,
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el validador de sujeto tiene una ACL incorrecta';
    END IF;
    IF NOT vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido(
               'hmac-sha256:personas:' || repeat('a', 64)
           )
       OR vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido('12345678Z')
       OR vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido(
           'persona@example.org'
       )
       OR vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido(
           '/personas/12345678Z'
       ) THEN
        RAISE EXCEPTION 'el formato HMAC cerrado de sujeto no se cumple';
    END IF;
    IF (SELECT count(*)
          FROM information_schema.columns
         WHERE table_schema = 'vec_bolsa_calculo_experiencia'
           AND table_name IN ('resultado_oficial', 'intento', 'recibo')
           AND column_name = 'generacion_clave_hmac'
           AND data_type = 'bigint') <> 3 THEN
        RAISE EXCEPTION 'la generacion HMAC no representa uint32 en SQL';
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'vec_bolsa_calculo_experiencia'
           AND data_type IN ('json', 'jsonb')
    ) THEN
        RAISE EXCEPTION 'el almacen contiene JSON libre';
    END IF;
    IF (SELECT count(*)
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_bolsa_calculo_experiencia.consumo_autorizaciones'::regclass
           AND confrelid =
             'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
           AND contype = 'f') <> 2 THEN
        RAISE EXCEPTION 'faltan vinculos a las dos decisiones V2';
    END IF;
    IF position(
           '67108864' IN pg_get_constraintdef((
               SELECT oid FROM pg_catalog.pg_constraint
                WHERE conrelid =
                    'vec_bolsa_calculo_experiencia.resultado_oficial'::regclass
                  AND conname = 'resultado_bytes_exactos'
           ))
    ) = 0 THEN
        RAISE EXCEPTION 'no se conserva el limite de 64 MiB';
    END IF;
    IF position(
           'sujeto_hmac_ref_valido' IN pg_get_constraintdef((
               SELECT oid FROM pg_catalog.pg_constraint
                WHERE conrelid =
                    'vec_bolsa_calculo_experiencia.resultado_oficial'::regclass
                  AND conname = 'resultado_referencias_validas'
           ))
       ) = 0
       OR position(
           'sujeto_hmac_ref_valido' IN pg_get_constraintdef((
               SELECT oid FROM pg_catalog.pg_constraint
                WHERE conrelid =
                    'vec_bolsa_calculo_experiencia.recibo'::regclass
                  AND conname = 'recibo_exacto'
           ))
       ) = 0 THEN
        RAISE EXCEPTION 'resultado o recibo no cierran la referencia HMAC';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_bolsa_calculo_experiencia.consumo_autorizaciones'::regclass
           AND confrelid =
             'vec_bolsa_calculo_experiencia.resultado_oficial'::regclass
           AND contype = 'f'
           AND position(
               'huella_selector_fuente_sha256' IN pg_get_constraintdef(oid)
           ) > 0
    ) OR position(
        'selector_fuente_canonico' IN pg_get_constraintdef((
            SELECT oid FROM pg_catalog.pg_constraint
             WHERE conrelid =
                 'vec_bolsa_calculo_experiencia.resultado_oficial'::regclass
               AND conname = 'resultado_bytes_exactos'
        ))
    ) = 0 THEN
        RAISE EXCEPTION 'el selector exacto no esta ligado de extremo a extremo';
    END IF;
END
$rls_y_funciones$;

-- Un GUC de sesion no forma parte de la frontera de aislamiento.
SET ROLE vec_bolsa_calculo_experiencia_aplicacion;
SET vec.tenant_id = 'tenant_controlado_por_atacante';
DO $tenant_fijo$
BEGIN
    IF (SELECT tenant_id
          FROM vec_bolsa_calculo_experiencia.configuracion_tenant) <>
       'diputacion_granada' THEN
        RAISE EXCEPTION 'el GUC altero el tenant instalado';
    END IF;
END
$tenant_fijo$;
RESET ROLE;
