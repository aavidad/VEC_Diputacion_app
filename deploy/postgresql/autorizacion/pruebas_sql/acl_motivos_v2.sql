\set ON_ERROR_STOP 1

DO $prueba$
DECLARE
    propietario oid := 'vec_autorizacion_propietario'::regrole;
    proyector oid := 'vec_autorizacion_motivos_proyector'::regrole;
    evaluador oid := 'vec_autorizacion_motivos_evaluador'::regrole;
    tabla record;
    funciones_definidoras integer;
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_roles
         WHERE oid IN (proyector, evaluador)
           AND NOT rolcanlogin
           AND NOT rolsuper
           AND NOT rolcreaterole
           AND NOT rolcreatedb
           AND NOT rolinherit
           AND NOT rolreplication
           AND NOT rolbypassrls) <> 2 THEN
        RAISE EXCEPTION 'atributos de roles V2 inesperados';
    END IF;

    IF pg_catalog.pg_has_role(
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion_propietario',
        'MEMBER'
    ) OR pg_catalog.pg_has_role(
        'vec_autorizacion_motivos_evaluador',
        'vec_autorizacion_propietario',
        'MEMBER'
    ) THEN
        RAISE EXCEPTION 'un rol V2 hereda al propietario';
    END IF;

    IF EXISTS (
        SELECT 1
         FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tipo.typnamespace
         WHERE espacio.nspname = 'vec_autorizacion'
           AND tipo.typtype = 'c'
           AND tipo.typname IN (
               'version_rol',
               'control_vigencia_version_rol',
               'control_vigencia_version_rol_actual',
               'asignacion_perfil',
               'asignacion_perfil_actual',
               'politica_restrictiva',
               'politica_restrictiva_actual',
               'control_catalogo_politicas',
               'decision_autorizacion'
           )
           AND pg_catalog.has_type_privilege(
               'public', tipo.oid, 'USAGE'
           )
    ) THEN
        RAISE EXCEPTION
            'PUBLIC conserva USAGE en un tipo compuesto de autorizacion';
    END IF;

    IF NOT has_schema_privilege(
        'vec_autorizacion_motivos_proyector', 'vec_autorizacion', 'USAGE'
    ) OR NOT has_schema_privilege(
        'vec_autorizacion_motivos_evaluador', 'vec_autorizacion', 'USAGE'
    ) OR has_schema_privilege(
        'vec_autorizacion_motivos_proyector', 'vec_autorizacion', 'CREATE'
    ) OR has_schema_privilege(
        'vec_autorizacion_motivos_evaluador', 'vec_autorizacion', 'CREATE'
    ) THEN
        RAISE EXCEPTION 'ACL de esquema distinta del USAGE minimo';
    END IF;

    FOR tabla IN
        SELECT clase.oid, clase.relname, clase.relrowsecurity,
               clase.relforcerowsecurity
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_autorizacion'
           AND clase.relname IN (
               'motivo_v2_evento_origen',
               'motivo_v2_catalogo_publicado',
               'motivo_v2_entrada',
               'motivo_v2_retirada',
               'motivo_v2_checkpoint_origen'
           )
    LOOP
        IF NOT tabla.relrowsecurity OR NOT tabla.relforcerowsecurity THEN
            RAISE EXCEPTION 'RLS no forzada en %', tabla.relname;
        END IF;
        IF has_table_privilege(
            'vec_autorizacion_motivos_proyector', tabla.oid,
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) OR has_table_privilege(
            'vec_autorizacion_motivos_evaluador', tabla.oid,
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION 'ACL directa indebida en %', tabla.relname;
        END IF;
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_policy AS politica
             WHERE politica.polrelid = tabla.oid
               AND politica.polname = 'acceso_propietario_exacto'
               AND politica.polcmd = '*'
               AND politica.polroles = ARRAY[propietario]
        ) THEN
            RAISE EXCEPTION 'politica de propietario inesperada en %', tabla.relname;
        END IF;
        IF EXISTS (
            SELECT 1
              FROM aclexplode(COALESCE(
                    (SELECT relacl FROM pg_catalog.pg_class WHERE oid = tabla.oid),
                    acldefault('r', propietario)
              )) AS acl
             WHERE acl.grantee = 0
        ) THEN
            RAISE EXCEPTION 'PUBLIC conserva ACL de tabla en %', tabla.relname;
        END IF;
    END LOOP;

    IF (SELECT count(*) FROM pg_catalog.pg_class AS clase
         JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = clase.relnamespace
        WHERE espacio.nspname = 'vec_autorizacion'
          AND clase.relname LIKE 'motivo_v2_%'
          AND clase.relkind = 'r') <> 5 THEN
        RAISE EXCEPTION 'inventario de tablas V2 inesperado';
    END IF;

    IF (SELECT count(*)
          FROM pg_catalog.pg_constraint
         WHERE connamespace = 'vec_autorizacion'::regnamespace
           AND convalidated
           AND conname IN (
               'motivo_v2_evento_registro_finito',
               'motivo_v2_catalogo_publicacion_finita',
               'motivo_v2_entrada_instantes_finitos',
               'motivo_v2_retirada_instante_finito',
               'motivo_v2_checkpoint_actualizacion_finita'
           )) <> 5 THEN
        RAISE EXCEPTION 'faltan constraints de instantes finitos';
    END IF;

    IF NOT has_function_privilege(
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamptz,jsonb)',
        'EXECUTE'
    ) OR NOT has_function_privilege(
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion.retirar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,text,timestamptz)',
        'EXECUTE'
    ) OR has_function_privilege(
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion.resolver_motivo_autorizacion_v2_actual(text,integer,text,text)',
        'EXECUTE'
    ) OR has_function_privilege(
        'vec_autorizacion_motivos_proyector',
        'vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'superficie funcional inesperada para el proyector';
    END IF;

    IF NOT has_function_privilege(
        'vec_autorizacion_motivos_evaluador',
        'vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)',
        'EXECUTE'
    ) OR has_function_privilege(
        'vec_autorizacion_motivos_evaluador',
        'vec_autorizacion.resolver_motivo_autorizacion_v2_actual(text,integer,text,text)',
        'EXECUTE'
    ) OR has_function_privilege(
        'vec_autorizacion_motivos_evaluador',
        'vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamptz,jsonb)',
        'EXECUTE'
    ) OR has_function_privilege(
        'vec_autorizacion_motivos_evaluador',
        'vec_autorizacion.retirar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,text,timestamptz)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'superficie funcional inesperada para el evaluador';
    END IF;

    SELECT count(*)
      INTO funciones_definidoras
      FROM pg_catalog.pg_proc AS funcion
      JOIN pg_catalog.pg_namespace AS espacio
        ON espacio.oid = funcion.pronamespace
     WHERE espacio.nspname = 'vec_autorizacion'
       AND funcion.proname IN (
           'publicar_motivos_autorizacion_v2',
           'retirar_motivos_autorizacion_v2',
           'resolver_motivo_autorizacion_v2_historico',
           'resolver_motivo_autorizacion_v2_actual'
       )
       AND funcion.proowner = propietario
       AND funcion.prosecdef
       AND funcion.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
       AND NOT EXISTS (
            SELECT 1
              FROM aclexplode(COALESCE(
                    funcion.proacl,
                    acldefault('f', funcion.proowner)
              )) AS acl
             WHERE acl.grantee = 0
               AND acl.privilege_type = 'EXECUTE'
       );
    IF funciones_definidoras <> 4 THEN
        RAISE EXCEPTION 'propiedad, search_path o ACL de funciones inesperados: %',
            funciones_definidoras;
    END IF;
END
$prueba$;
