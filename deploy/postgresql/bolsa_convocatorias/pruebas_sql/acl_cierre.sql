-- Se ejecuta como superusuario tras crear las identidades LOGIN de prueba.
DO $inventario$
DECLARE
    funcion regprocedure :=
        'vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'::regprocedure;
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion'
    ] LOOP
        IF has_schema_privilege(
               rol, 'vec_bolsa_convocatorias', 'USAGE'
           )
           OR has_function_privilege(rol, funcion, 'EXECUTE')
           OR has_table_privilege(
               rol,
               'vec_bolsa_convocatorias.version_convocatoria',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR has_table_privilege(
               rol,
               'vec_bolsa_convocatorias.atestacion_autorizacion_version',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR has_table_privilege(
               rol,
               'vec_bolsa_convocatorias.uso_decision_consulta',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           ) THEN
            RAISE EXCEPTION 'ACL abierta para %', rol;
        END IF;
    END LOOP;
    IF has_schema_privilege(
           'public', 'vec_bolsa_convocatorias', 'USAGE'
       ) OR has_function_privilege('public', funcion, 'EXECUTE') THEN
        RAISE EXCEPTION 'PUBLIC conserva privilegios';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc
         WHERE oid = funcion
           AND prosecdef
           AND proconfig @> ARRAY[
               'search_path=pg_catalog, pg_temp',
               'TimeZone=UTC'
           ]::text[]
    ) THEN
        RAISE EXCEPTION 'la funcion no conserva SECURITY DEFINER cerrado';
    END IF;
END
$inventario$;

-- Las siete tablas deben tener RLS forzada y no ACL runtime directa.
DO $rls$
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_class AS tabla
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_convocatorias'
           AND tabla.relname IN (
               'version_convocatoria', 'instancia_flujo_version',
               'atestacion_autorizacion_version',
               'atestacion_autorizacion_actual',
               'uso_decision_consulta', 'auditoria', 'auditoria_actual'
           )
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity) <> 7 THEN
        RAISE EXCEPTION 'inventario RLS incompleto';
    END IF;
END
$rls$;
