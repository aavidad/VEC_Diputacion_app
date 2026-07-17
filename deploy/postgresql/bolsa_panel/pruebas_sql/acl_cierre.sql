-- Inventario ejecutado por una identidad DBA sobre una base efimera.
DO $inventario$
DECLARE
    consulta regprocedure :=
        'vec_bolsa_panel.consultar_panel_interno_v1(jsonb,jsonb,bytea,bytea,text)'::regprocedure;
    publicador regprocedure :=
        'vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb)'::regprocedure;
    rol text;
BEGIN
    IF has_schema_privilege(
           'public', 'vec_bolsa_panel', 'USAGE'
       ) OR has_function_privilege('public', consulta, 'EXECUTE')
       OR has_function_privilege('public', publicador, 'EXECUTE') THEN
        RAISE EXCEPTION 'PUBLIC conserva privilegios del panel';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_panel_ejecutor_consulta',
        'vec_bolsa_panel_registrador_atestacion'
    ] LOOP
        IF has_schema_privilege(rol, 'vec_bolsa_panel', 'USAGE')
           OR has_function_privilege(rol, consulta, 'EXECUTE')
           OR has_function_privilege(rol, publicador, 'EXECUTE')
           OR has_table_privilege(
               rol, 'vec_bolsa_panel.proyeccion_panel',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR has_table_privilege(
               rol, 'vec_bolsa_panel.atestacion_autorizacion_version',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR has_table_privilege(
               rol, 'vec_bolsa_panel.consulta_confirmada',
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           ) THEN
            RAISE EXCEPTION 'ACL abierta para %', rol;
        END IF;
    END LOOP;
    IF NOT has_schema_privilege(
           'vec_bolsa_panel_proyector', 'vec_bolsa_panel', 'USAGE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_panel_proyector', publicador, 'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_panel_proyector', consulta, 'EXECUTE'
       ) OR has_table_privilege(
           'vec_bolsa_panel_proyector', 'vec_bolsa_panel.proyeccion_panel',
           'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
       ) THEN
        RAISE EXCEPTION 'ACL del proyector no es minima';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = consulta AND prosecdef
           AND proconfig @> ARRAY[
               'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
           ]::text[]
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = publicador AND prosecdef
           AND proconfig @> ARRAY[
               'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
           ]::text[]
    ) THEN
        RAISE EXCEPTION 'funcion SECURITY DEFINER sin cierre completo';
    END IF;
END
$inventario$;

DO $rls$
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_class AS tabla
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_panel'
           AND tabla.relname IN (
               'proyeccion_panel', 'proyeccion_actual',
               'convocatoria_resumen', 'actuacion_pendiente',
               'atestacion_autorizacion_version',
               'atestacion_autorizacion_actual', 'auditoria',
               'auditoria_actual', 'consulta_confirmada'
           )
           AND tabla.relrowsecurity AND tabla.relforcerowsecurity) <> 9 THEN
        RAISE EXCEPTION 'inventario RLS incompleto';
    END IF;
END
$rls$;
