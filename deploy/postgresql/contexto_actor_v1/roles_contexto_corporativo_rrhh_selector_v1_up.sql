-- Alta de una sola ejecucion del rol nominal que seleccionara y registrara el
-- contexto corporativo RRHH. No concede acceso funcional.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1',
        0
    )
);

DO $prevalidacion$
DECLARE
    oid_dueno_base oid;
    oid_propietario oid;
    oid_migrador oid;
    oid_runtime oid;
    oid_esquema oid;
    funciones_runtime oid[];
    diferencias integer;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    IF current_setting('server_version_num')::integer < 180000
       OR pg_catalog.getdatabaseencoding() <> 'UTF8' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    SELECT base.datdba
      INTO oid_dueno_base
      FROM pg_catalog.pg_database AS base
     WHERE base.datname = current_database()
       AND base.datallowconn IS TRUE
       AND base.datistemplate IS FALSE
       AND NOT EXISTS (
           SELECT 1
             FROM LATERAL pg_catalog.aclexplode(
                 coalesce(
                     base.datacl,
                     pg_catalog.acldefault('d', base.datdba)
                 )
             ) AS acl
            WHERE acl.grantee = 0
       );
    IF oid_dueno_base IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    SELECT propietario.oid, migrador.oid, runtime.oid
      INTO oid_propietario, oid_migrador, oid_runtime
      FROM pg_catalog.pg_authid AS propietario
      CROSS JOIN pg_catalog.pg_authid AS migrador
      CROSS JOIN pg_catalog.pg_authid AS runtime
     WHERE propietario.rolname =
               'vec_contexto_actor_v1_propietario'
       AND migrador.rolname = 'vec_contexto_actor_v1_migrador'
       AND runtime.rolname = 'vec_contexto_actor_v1_runtime';
    IF oid_propietario IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    IF (
        SELECT count(*)
          FROM pg_catalog.pg_authid AS rol
         WHERE rol.oid = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
           AND rol.rolcanlogin IS FALSE
           AND rol.rolsuper IS FALSE
           AND rol.rolcreatedb IS FALSE
           AND rol.rolcreaterole IS FALSE
           AND rol.rolinherit IS FALSE
           AND rol.rolreplication IS FALSE
           AND rol.rolbypassrls IS FALSE
           AND rol.rolconnlimit = -1
           AND rol.rolpassword IS NULL
           AND rol.rolvaliduntil IS NULL
           AND pg_catalog.shobj_description(
                   rol.oid, 'pg_authid'
               ) IS NULL
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_db_role_setting AS ajuste
                WHERE ajuste.setrole = rol.oid
           )
    ) <> 3 OR oid_dueno_base = ANY (ARRAY[
        oid_propietario, oid_migrador, oid_runtime
    ]) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT relacion.roleid, relacion.member, relacion.grantor,
               relacion.admin_option, relacion.inherit_option,
               relacion.set_option
          FROM pg_catalog.pg_auth_members AS relacion
         WHERE relacion.roleid = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
            OR relacion.member = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
            OR relacion.grantor = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
    ), esperado(
        roleid, member, grantor, admin_option, inherit_option, set_option
    ) AS (
        VALUES (
            oid_propietario, oid_migrador, 10::oid, false, false, true
        )
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid
         WHERE oid = 10
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type,
               acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
           AND (
               acl.grantee = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
               OR acl.grantor = ANY (ARRAY[
                   oid_propietario, oid_migrador, oid_runtime
               ])
           )
    ), esperado(
        grantor, grantee, privilege_type, is_grantable
    ) AS (
        VALUES
            (oid_dueno_base, oid_propietario, 'CONNECT'::text, false),
            (oid_dueno_base, oid_migrador, 'CONNECT'::text, false),
            (oid_dueno_base, oid_runtime, 'CONNECT'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    SELECT espacio.oid
      INTO oid_esquema
      FROM pg_catalog.pg_namespace AS espacio
     WHERE espacio.nspname = 'vec_contexto_actor_v1'
       AND espacio.nspowner = oid_propietario;
    IF oid_esquema IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type,
               acl.is_grantable
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.nspacl) AS acl
         WHERE espacio.oid = oid_esquema
           AND (
               acl.grantee = oid_runtime
               OR acl.grantor = oid_runtime
           )
    ), esperado(
        grantor, grantee, privilege_type, is_grantable
    ) AS (
        VALUES (oid_propietario, oid_runtime, 'USAGE'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    funciones_runtime := ARRAY[
        pg_catalog.to_regprocedure(
            'vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'
        ),
        pg_catalog.to_regprocedure(
            'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'
        ),
        pg_catalog.to_regprocedure(
            'vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'
        )
    ];
    IF pg_catalog.array_position(funciones_runtime, NULL) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS funcion
            WHERE funcion.oid = ANY (funciones_runtime)
              AND (
                  funcion.proowner <> oid_propietario
                  OR funcion.prosecdef IS FALSE
                  OR funcion.proconfig IS DISTINCT FROM
                     ARRAY['search_path=pg_catalog']::text[]
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT funcion.oid, acl.grantor, acl.grantee,
               acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
         WHERE funcion.oid = ANY (funciones_runtime)
           AND (
               acl.grantee = oid_runtime
               OR acl.grantor = oid_runtime
           )
    ), esperado(
        oid, grantor, grantee, privilege_type, is_grantable
    ) AS (
        VALUES
            (funciones_runtime[1], oid_propietario, oid_runtime,
             'EXECUTE'::text, false),
            (funciones_runtime[2], oid_propietario, oid_runtime,
             'EXECUTE'::text, false),
            (funciones_runtime[3], oid_propietario, oid_runtime,
             'EXECUTE'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE acl.grantee = oid_runtime
            OR acl.grantor = oid_runtime
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute AS columna
          CROSS JOIN LATERAL pg_catalog.aclexplode(columna.attacl) AS acl
         WHERE acl.grantee = oid_runtime
            OR acl.grantor = oid_runtime
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) AS acl
         WHERE acl.grantee = oid_runtime
            OR acl.grantor = oid_runtime
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;

    -- Cualquier privilegio de PUBLIC sobre una superficie definida por el
    -- usuario alcanzaria tambien al nuevo rol y por tanto cierra el alta.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS acl
         WHERE base.datistemplate IS FALSE
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute AS columna
          JOIN pg_catalog.pg_class AS objeto
            ON objeto.oid = columna.attrelid
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(columna.attacl) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tipo.typnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  tipo.typacl,
                  pg_catalog.acldefault('T', tipo.typowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND (
               tipo.typtype IN ('c', 'd', 'e', 'm', 'r')
               OR (
                   tipo.typtype = 'b'
                   AND tipo.typelem = 0
                   AND tipo.typcategory <> 'A'
               )
           )
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_largeobject_metadata AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.lomacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_data_wrapper AS envoltorio
          CROSS JOIN LATERAL pg_catalog.aclexplode(envoltorio.fdwacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_server AS servidor
          CROSS JOIN LATERAL pg_catalog.aclexplode(servidor.srvacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_language AS lenguaje
          CROSS JOIN LATERAL pg_catalog.aclexplode(lenguaje.lanacl) AS acl
         WHERE lenguaje.oid >= 16384
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_tablespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.spcacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_policy AS politica
         WHERE 0::oid = ANY (politica.polroles)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_parameter_acl AS parametro
          CROSS JOIN LATERAL pg_catalog.aclexplode(parametro.paracl) AS acl
         WHERE acl.grantee = 0
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH rechazada';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_contexto_actor_corporativo_rrhh_selector
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS
    CONNECTION LIMIT -1 PASSWORD NULL;

COMMENT ON ROLE vec_contexto_actor_corporativo_rrhh_selector IS
    'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1';

DO $privilegio_base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_corporativo_rrhh_selector',
        current_database()
    );
END
$privilegio_base$;

DO $postvalidacion$
DECLARE
    oid_selector oid;
    oid_base oid;
    oid_dueno_base oid;
    diferencias integer;
BEGIN
    SELECT rol.oid
      INTO oid_selector
      FROM pg_catalog.pg_authid AS rol
     WHERE rol.rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
       AND rol.rolcanlogin IS FALSE
       AND rol.rolsuper IS FALSE
       AND rol.rolcreatedb IS FALSE
       AND rol.rolcreaterole IS FALSE
       AND rol.rolinherit IS FALSE
       AND rol.rolreplication IS FALSE
       AND rol.rolbypassrls IS FALSE
       AND rol.rolconnlimit = -1
       AND rol.rolpassword IS NULL
       AND rol.rolvaliduntil IS NULL
       AND pg_catalog.shobj_description(rol.oid, 'pg_authid') =
           'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_db_role_setting AS ajuste
            WHERE ajuste.setrole = rol.oid
       );
    SELECT base.oid, base.datdba
      INTO oid_base, oid_dueno_base
      FROM pg_catalog.pg_database AS base
     WHERE base.datname = current_database();

    IF oid_selector IS NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS relacion
         WHERE relacion.roleid = oid_selector
            OR relacion.member = oid_selector
            OR relacion.grantor = oid_selector
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH incompleta';
    END IF;

    WITH actual AS (
        SELECT base.oid AS objeto, acl.grantor, acl.grantee,
               acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ), esperado(
        objeto, grantor, grantee, privilege_type, is_grantable
    ) AS (
        VALUES (
            oid_base, oid_dueno_base, oid_selector, 'CONNECT'::text, false
        )
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;

    IF diferencias <> 0 OR NOT COALESCE((
        SELECT count(*) = 1
           AND bool_and(
               dependencia.dbid = 0
               AND dependencia.classid =
                   'pg_catalog.pg_database'::regclass
               AND dependencia.objid = oid_base
               AND dependencia.objsubid = 0
               AND dependencia.deptype = 'a'
           )
          FROM pg_catalog.pg_shdepend AS dependencia
         WHERE dependencia.refclassid =
                   'pg_catalog.pg_authid'::regclass
           AND dependencia.refobjid = oid_selector
    ), false) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.nspacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute AS columna
          CROSS JOIN LATERAL pg_catalog.aclexplode(columna.attacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          LEFT JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl) AS acl
            ON true
         WHERE defecto.defaclrole = oid_selector
            OR acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_policy AS politica
         WHERE oid_selector = ANY (politica.polroles)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_parameter_acl AS parametro
          CROSS JOIN LATERAL pg_catalog.aclexplode(parametro.paracl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
         FROM pg_catalog.pg_shseclabel AS etiqueta
         WHERE etiqueta.classoid = 'pg_catalog.pg_authid'::regclass
           AND etiqueta.objoid = oid_selector
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH incompleta';
    END IF;

    -- Sin membresias, todo privilegio efectivo adicional solo puede proceder
    -- de PUBLIC; se comprueba de nuevo despues de crear el rol.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
         WHERE (
             base.datistemplate IS FALSE
             AND
             base.oid <> oid_base
             AND pg_catalog.has_database_privilege(
                 oid_selector, base.oid, 'CONNECT'
             )
         ) OR pg_catalog.has_database_privilege(
             oid_selector, base.oid, 'CREATE'
         ) OR pg_catalog.has_database_privilege(
             oid_selector, base.oid, 'TEMPORARY'
         )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND (
               pg_catalog.has_schema_privilege(
                   oid_selector, espacio.oid, 'USAGE'
               )
               OR pg_catalog.has_schema_privilege(
                   oid_selector, espacio.oid, 'CREATE'
               )
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND (
               (
                   objeto.relkind IN ('r', 'p', 'v', 'm', 'f')
                   AND (
                       pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'SELECT'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'INSERT'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'UPDATE'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'DELETE'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'TRUNCATE'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'REFERENCES'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'TRIGGER'
                       )
                       OR pg_catalog.has_table_privilege(
                           oid_selector, objeto.oid, 'MAINTAIN'
                       )
                       OR pg_catalog.has_any_column_privilege(
                           oid_selector, objeto.oid,
                           'SELECT,INSERT,UPDATE,REFERENCES'
                       )
                   )
               )
               OR (
                   objeto.relkind = 'S'
                   AND (
                       pg_catalog.has_sequence_privilege(
                           oid_selector, objeto.oid, 'USAGE'
                       )
                       OR pg_catalog.has_sequence_privilege(
                           oid_selector, objeto.oid, 'SELECT'
                       )
                       OR pg_catalog.has_sequence_privilege(
                           oid_selector, objeto.oid, 'UPDATE'
                       )
                   )
               )
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND pg_catalog.has_function_privilege(
               oid_selector, funcion.oid, 'EXECUTE'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tipo.typnamespace
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND (
               tipo.typtype IN ('c', 'd', 'e', 'm', 'r')
               OR (
                   tipo.typtype = 'b'
                   AND tipo.typelem = 0
                   AND tipo.typcategory <> 'A'
               )
           )
           AND pg_catalog.has_type_privilege(
               oid_selector, tipo.oid, 'USAGE'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_largeobject_metadata AS objeto
         WHERE pg_catalog.has_largeobject_privilege(
                   oid_selector, objeto.oid, 'SELECT'
               )
            OR pg_catalog.has_largeobject_privilege(
                   oid_selector, objeto.oid, 'UPDATE'
               )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_data_wrapper AS envoltorio
         WHERE pg_catalog.has_foreign_data_wrapper_privilege(
             oid_selector, envoltorio.oid, 'USAGE'
         )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_server AS servidor
         WHERE pg_catalog.has_server_privilege(
             oid_selector, servidor.oid, 'USAGE'
         )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_language AS lenguaje
         WHERE lenguaje.oid >= 16384
           AND pg_catalog.has_language_privilege(
               oid_selector, lenguaje.oid, 'USAGE'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_tablespace AS espacio
         WHERE pg_catalog.has_tablespace_privilege(
             oid_selector, espacio.oid, 'CREATE'
         )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_parameter_acl AS parametro
         WHERE pg_catalog.has_parameter_privilege(
                   oid_selector, parametro.parname, 'SET'
               )
            OR pg_catalog.has_parameter_privilege(
                   oid_selector, parametro.parname, 'ALTER SYSTEM'
               )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del selector corporativo RRHH incompleta';
    END IF;
END
$postvalidacion$;

COMMIT;
