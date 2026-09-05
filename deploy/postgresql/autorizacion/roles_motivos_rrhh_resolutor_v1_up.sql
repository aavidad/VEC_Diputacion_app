-- Alta de una sola ejecucion del rol nominal que resolvera los motivos RRHH.
-- No concede acceso funcional: 000010 otorgara las dos fachadas exactas.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:rol-motivos-rrhh-resolutor:v1', 0
    )
);

DO $prevalidacion$
DECLARE
    oid_dueno_base oid;
    oid_propietario oid;
    oid_migrador oid;
    oid_proyector oid;
    oid_evaluador oid;
    diferencias integer;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    IF current_setting('server_version_num')::integer < 180000
       OR pg_catalog.getdatabaseencoding() <> 'UTF8' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_motivos_rrhh_resolutor'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    SELECT base.datdba
      INTO oid_dueno_base
      FROM pg_catalog.pg_database AS base
     WHERE base.datname = current_database()
       AND base.datallowconn IS TRUE
       AND base.datistemplate IS FALSE
       AND NOT EXISTS (
           SELECT 1
             FROM LATERAL pg_catalog.aclexplode(base.datacl) AS acl
            WHERE acl.grantee = 0
       );
    IF oid_dueno_base IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    SELECT propietario.oid, migrador.oid, proyector.oid, evaluador.oid
      INTO oid_propietario, oid_migrador, oid_proyector, oid_evaluador
      FROM pg_catalog.pg_authid AS propietario
      CROSS JOIN pg_catalog.pg_authid AS migrador
      CROSS JOIN pg_catalog.pg_authid AS proyector
      CROSS JOIN pg_catalog.pg_authid AS evaluador
     WHERE propietario.rolname = 'vec_autorizacion_propietario'
       AND migrador.rolname = 'vec_autorizacion_migrador'
       AND proyector.rolname = 'vec_autorizacion_motivos_proyector'
       AND evaluador.rolname = 'vec_autorizacion_motivos_evaluador';
    IF oid_propietario IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    IF (
        SELECT count(*)
          FROM pg_catalog.pg_authid
         WHERE oid = ANY (ARRAY[
                   oid_propietario, oid_migrador,
                   oid_proyector, oid_evaluador
               ])
           AND rolcanlogin IS FALSE
           AND rolsuper IS FALSE
           AND rolcreatedb IS FALSE
           AND rolcreaterole IS FALSE
           AND rolinherit IS FALSE
           AND rolreplication IS FALSE
           AND rolbypassrls IS FALSE
           AND rolconnlimit = -1
           AND rolpassword IS NULL
           AND rolvaliduntil IS NULL
           AND pg_catalog.shobj_description(oid, 'pg_authid') IS NULL
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_db_role_setting AS ajuste
                WHERE ajuste.setrole = pg_authid.oid
           )
    ) <> 4 OR oid_dueno_base = ANY (ARRAY[
        oid_propietario, oid_migrador, oid_proyector, oid_evaluador
    ]) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    -- La resolución puede instalarse después de que existan conexiones
    -- nominales de gobierno/proyección. No se retiran sus permisos ni se
    -- conceden otros: solo se admiten LOGIN limitados, sin delegación.
    -- Los grupos siguen sin pertenecer a otros roles ni otorgar membresías.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS relacion
          JOIN pg_catalog.pg_roles AS miembro
            ON miembro.oid = relacion.member
         WHERE (
             relacion.roleid = ANY (ARRAY[
                 oid_propietario, oid_proyector, oid_evaluador
             ])
             OR relacion.member = ANY (ARRAY[
                 oid_propietario, oid_proyector, oid_evaluador
             ])
             OR relacion.grantor = ANY (ARRAY[
                 oid_propietario, oid_proyector, oid_evaluador
             ])
         ) AND NOT (
             (
                 relacion.roleid = oid_propietario
                 AND relacion.member = oid_migrador
                 AND relacion.grantor = 10
                 AND NOT relacion.admin_option
                 AND NOT relacion.inherit_option
                 AND relacion.set_option
             ) OR (
                 relacion.roleid = ANY (ARRAY[
                     oid_propietario, oid_proyector, oid_evaluador
                 ])
                 AND relacion.grantor = 10
                 AND NOT relacion.admin_option
                 AND miembro.rolcanlogin
                 AND NOT miembro.rolsuper
                 AND NOT miembro.rolcreatedb
                 AND NOT miembro.rolcreaterole
                 AND miembro.rolinherit
                 AND NOT miembro.rolreplication
                 AND NOT miembro.rolbypassrls
                 AND NOT EXISTS (
                     SELECT 1 FROM pg_catalog.pg_auth_members AS subordinada
                      WHERE subordinada.roleid = miembro.oid
                 )
                 AND (
                     (relacion.roleid = oid_propietario
                      AND relacion.set_option)
                     OR (relacion.roleid IN (oid_proyector, oid_evaluador)
                         AND relacion.inherit_option)
                 )
                 AND NOT EXISTS (
                     SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
                      WHERE ajuste.setrole = miembro.oid
                 )
             )
         )
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid
         WHERE oid = 10
           AND rolsuper IS TRUE
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members
         WHERE roleid = oid_propietario
           AND member = oid_migrador
           AND grantor = 10
           AND admin_option IS FALSE
           AND inherit_option IS FALSE
           AND set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_autorizacion'
           AND nspowner = oid_propietario
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
           AND acl.grantee = ANY (ARRAY[
                   oid_propietario, oid_migrador,
                   oid_proyector, oid_evaluador
               ])
    ), esperado(grantor, grantee, privilege_type, is_grantable) AS (
        VALUES
            (oid_dueno_base, oid_propietario, 'CONNECT'::text, false),
            (oid_dueno_base, oid_propietario, 'CREATE'::text, false),
            (oid_dueno_base, oid_migrador, 'CONNECT'::text, false),
            (oid_dueno_base, oid_proyector, 'CONNECT'::text, false),
            (oid_dueno_base, oid_evaluador, 'CONNECT'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.nspacl) AS acl
         WHERE espacio.nspname = 'vec_autorizacion'
           AND (
               acl.grantee = ANY (ARRAY[oid_proyector, oid_evaluador])
               OR acl.grantor = ANY (ARRAY[oid_proyector, oid_evaluador])
           )
    ), esperado(grantor, grantee, privilege_type, is_grantable) AS (
        VALUES
            (oid_propietario, oid_proyector, 'USAGE'::text, false),
            (oid_propietario, oid_evaluador, 'USAGE'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;

    WITH actual AS (
        SELECT funcion.oid::regprocedure::text AS objeto,
               acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
         WHERE espacio.nspname = 'vec_autorizacion'
           AND (
               acl.grantee = ANY (ARRAY[oid_proyector, oid_evaluador])
               OR acl.grantor = ANY (ARRAY[oid_proyector, oid_evaluador])
           )
    ), esperado(objeto, grantor, grantee, privilege_type, is_grantable) AS (
        VALUES
          ('vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamp with time zone,jsonb)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false),
          ('vec_autorizacion.retirar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,text,timestamp with time zone)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false),
          ('vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamp with time zone)',
           oid_propietario, oid_evaluador, 'EXECUTE'::text, false),
          ('vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false),
          ('vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false),
          ('vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false),
          ('vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)',
           oid_propietario, oid_proyector, 'EXECUTE'::text, false)
        -- Cobertura puede estar instalada antes que la bandeja. Se preservan
        -- solo sus tres fachadas nominales, sin aceptar otras ACL adicionales.
        UNION ALL
        SELECT cobertura.objeto, oid_propietario, cobertura.destinatario,
               'EXECUTE'::text, false
          FROM (VALUES
            ('vec_autorizacion.publicar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,timestamp with time zone,jsonb)',
             oid_proyector),
            ('vec_autorizacion.retirar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,text,timestamp with time zone)',
             oid_proyector),
            ('vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)',
             oid_evaluador)
          ) AS cobertura(objeto, destinatario)
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = pg_catalog.to_regprocedure(cobertura.objeto)
           AND funcion.proowner = oid_propietario
           AND funcion.prosecdef
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;
    IF diferencias <> 0 OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = objeto.relnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE espacio.nspname = 'vec_autorizacion'
           AND (
               acl.grantee = ANY (ARRAY[oid_proyector, oid_evaluador])
               OR acl.grantor = ANY (ARRAY[oid_proyector, oid_evaluador])
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = tipo.typnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) AS acl
         WHERE espacio.nspname = 'vec_autorizacion'
           AND (
               acl.grantee = ANY (ARRAY[oid_proyector, oid_evaluador])
               OR acl.grantor = ANY (ARRAY[oid_proyector, oid_evaluador])
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH rechazada';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_autorizacion_motivos_rrhh_resolutor
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS
    CONNECTION LIMIT -1 PASSWORD NULL;

COMMENT ON ROLE vec_autorizacion_motivos_rrhh_resolutor IS
    'vec_autorizacion:rol-motivos-rrhh-resolutor:v1';

DO $privilegio_base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_motivos_rrhh_resolutor',
        current_database()
    );
END
$privilegio_base$;

DO $postvalidacion$
DECLARE
    oid_resolutor oid;
    oid_dueno_base oid;
    diferencias integer;
BEGIN
    SELECT rol.oid
      INTO oid_resolutor
      FROM pg_catalog.pg_authid AS rol
     WHERE rol.rolname = 'vec_autorizacion_motivos_rrhh_resolutor'
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
           'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_db_role_setting AS ajuste
            WHERE ajuste.setrole = rol.oid
       );
    SELECT datdba INTO oid_dueno_base
      FROM pg_catalog.pg_database
     WHERE datname = current_database();

    IF oid_resolutor IS NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS relacion
         WHERE relacion.roleid = oid_resolutor
            OR relacion.member = oid_resolutor
            OR relacion.grantor = oid_resolutor
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH incompleta';
    END IF;

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
           AND (acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor)
    ), esperado(grantor, grantee, privilege_type, is_grantable) AS (
        VALUES (oid_dueno_base, oid_resolutor, 'CONNECT'::text, false)
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;

    IF diferencias <> 0 OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.nspacl) AS acl
         WHERE acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
         WHERE acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) AS acl
         WHERE acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'alta del resolutor RRHH incompleta';
    END IF;
END
$postvalidacion$;

COMMIT;
