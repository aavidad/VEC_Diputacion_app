-- Ejecutar como DBA tras crear los roles. Ninguna identidad runtime puede
-- entrar en el esquema ni invocar la funcion antes de la autoridad COSE real.
DO $acl$
DECLARE
    funcion regprocedure :=
      'vec_bolsa_llamamientos.guardar_propuesta_v1(jsonb,jsonb,bytea,bytea)'::regprocedure;
    rol text;
    tabla text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_llamamientos_ejecutor',
        'vec_bolsa_llamamientos_proyector_autoritativo',
        'vec_bolsa_llamamientos_registrador_atestacion',
        'vec_bolsa_llamamientos_despachador_outbox'
    ] LOOP
        IF has_schema_privilege(rol, 'vec_bolsa_llamamientos', 'USAGE') OR
           has_function_privilege(rol, funcion, 'EXECUTE') THEN
            RAISE EXCEPTION 'frontera abierta para %', rol;
        END IF;
        FOREACH tabla IN ARRAY ARRAY[
            'bolsa_autoritativa', 'necesidad_autoritativa',
            'necesidad_actual', 'politica_autoritativa',
            'instantanea_autoritativa', 'evaluacion_autoritativa',
            'atestacion_autorizacion_version',
            'atestacion_autorizacion_actual', 'propuesta',
            'referencia_consumida', 'uso_decision', 'auditoria',
            'auditoria_actual', 'outbox'
        ] LOOP
            IF has_table_privilege(
                rol, format('vec_bolsa_llamamientos.%I', tabla),
                'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
            ) THEN
                RAISE EXCEPTION 'tabla % accesible para %', tabla, rol;
            END IF;
        END LOOP;
    END LOOP;
    IF has_schema_privilege('public', 'vec_bolsa_llamamientos', 'USAGE') OR
       has_function_privilege('public', funcion, 'EXECUTE') THEN
        RAISE EXCEPTION 'PUBLIC conserva autoridad';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = funcion AND prosecdef AND proconfig @> ARRAY[
           'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
         ]::text[]
    ) THEN
        RAISE EXCEPTION 'funcion sin SECURITY DEFINER cerrado';
    END IF;
END
$acl$;

DO $rls$
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_llamamientos'
           AND clase.relkind = 'r'
           AND clase.relrowsecurity AND clase.relforcerowsecurity) <> 14 THEN
        RAISE EXCEPTION 'inventario RLS incompleto';
    END IF;
END
$rls$;
