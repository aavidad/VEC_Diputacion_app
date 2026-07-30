-- Retirada gobernada del rol nominal. Solo revoca el CONNECT creado por el
-- alta y deja que DROP ROLE sin CASCADE detecte toda dependencia ajena.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:rol-motivos-rrhh-resolutor:v1', 0
    )
);

-- Orden comun de retirada de roles acreditado por roles_v2_down.sql.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    oid_resolutor oid;
    oid_dueno_base oid;
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
            MESSAGE = 'retirada del resolutor RRHH rechazada';
    END IF;

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
    IF oid_resolutor IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del resolutor RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS relacion
         WHERE relacion.roleid = oid_resolutor
            OR relacion.member = oid_resolutor
            OR relacion.grantor = oid_resolutor
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del resolutor RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_autorizacion'
           AND funcion.proname = ANY (ARRAY[
               'resolver_motivo_cuadro_rrhh_v1',
               'resolver_motivo_detalle_rrhh_v1'
           ]::name[])
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del resolutor RRHH rechazada';
    END IF;

    SELECT base.datdba
      INTO oid_dueno_base
      FROM pg_catalog.pg_database AS base
     WHERE base.datname = current_database();
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
    IF diferencias <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del resolutor RRHH rechazada';
    END IF;
END
$prevalidacion$;

DO $privilegio_base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_autorizacion_motivos_rrhh_resolutor',
        current_database()
    );
END
$privilegio_base$;

DO $postrevocacion$
DECLARE
    oid_resolutor oid :=
        'vec_autorizacion_motivos_rrhh_resolutor'::regrole::oid;
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
           AND (acl.grantee = oid_resolutor OR acl.grantor = oid_resolutor)
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del resolutor RRHH incompleta';
    END IF;
END
$postrevocacion$;

DROP ROLE vec_autorizacion_motivos_rrhh_resolutor;

COMMIT;
