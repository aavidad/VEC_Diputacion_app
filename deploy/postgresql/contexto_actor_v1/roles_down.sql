\set ON_ERROR_STOP on
\if :{?confirmar_destruccion_contexto_actor_v1}
\else
DO $confirmacion_ausente$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'falta confirmacion explicita para retirar roles de ContextoActor V1';
END
$confirmacion_ausente$;
\endif
SELECT :'confirmar_destruccion_contexto_actor_v1' =
       'DESTRUIR_CONTEXTO_ACTOR_V1' AS confirmacion_valida \gset
\if :confirmacion_valida
\else
DO $confirmacion_incorrecta$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'confirmacion incorrecta para retirar roles de ContextoActor V1';
END
$confirmacion_incorrecta$;
\endif

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $superusuario$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de roles de ContextoActor V1 requiere superusuario';
    END IF;
END
$superusuario$;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:base:v1', 0
    )
);
LOCK TABLE
    pg_catalog.pg_authid,
    pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting,
    pg_catalog.pg_shdepend,
    pg_catalog.pg_shdescription,
    pg_catalog.pg_shseclabel,
    pg_catalog.pg_database
IN SHARE ROW EXCLUSIVE MODE;

DO $estado_inicial$
DECLARE
    roles oid[] := ARRAY[
        'vec_contexto_actor_v1_propietario'::regrole,
        'vec_contexto_actor_v1_migrador'::regrole,
        'vec_contexto_actor_v1_runtime'::regrole
    ];
BEGIN
    IF (
        SELECT count(*) <> 3
          FROM pg_catalog.pg_authid AS r
         WHERE r.oid = ANY (roles)
           AND NOT r.rolsuper AND NOT r.rolinherit
           AND NOT r.rolcreaterole AND NOT r.rolcreatedb
           AND NOT r.rolcanlogin AND NOT r.rolreplication
           AND NOT r.rolbypassrls AND r.rolconnlimit = -1
           AND r.rolvaliduntil IS NULL AND r.rolpassword IS NULL
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shdescription AS d
         WHERE d.classoid = 'pg_catalog.pg_authid'::regclass
           AND d.objoid = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shseclabel AS s
         WHERE s.classoid = 'pg_catalog.pg_authid'::regclass
           AND s.objoid = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_db_role_setting AS s
         WHERE s.setrole = 0 OR s.setrole = ANY (roles)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de roles rechazada: atributos o ajustes no acreditados';
    END IF;

    IF (SELECT count(*) FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = ANY (roles) OR m.member = ANY (roles)
            OR m.grantor = ANY (roles)) <> 1 OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = 'vec_contexto_actor_v1_propietario'::regrole
           AND m.member = 'vec_contexto_actor_v1_migrador'::regrole
           AND m.grantor = current_user::regrole
           AND NOT m.admin_option AND NOT m.inherit_option
           AND m.set_option
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de roles rechazada: membresias no acreditadas';
    END IF;

    IF (
        SELECT count(*) <> 3
          FROM pg_catalog.pg_database AS b
          CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) AS a
         WHERE a.grantee = ANY (roles) OR a.grantor = ANY (roles)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS b
          CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) AS a
         WHERE (a.grantee = ANY (roles) OR a.grantor = ANY (roles))
           AND (b.datname <> current_database()
                OR a.grantee <> ALL (roles)
                OR a.grantor <> current_user::regrole
                OR a.privilege_type <> 'CONNECT' OR a.is_grantable)
    ) OR (
        SELECT count(*) <> 3 FROM pg_catalog.pg_shdepend AS d
         WHERE d.refclassid = 'pg_catalog.pg_authid'::regclass
           AND d.refobjid = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shdepend AS d
         WHERE d.refclassid = 'pg_catalog.pg_authid'::regclass
           AND d.refobjid = ANY (roles)
           AND (d.dbid <> 0
                OR d.classid <> 'pg_catalog.pg_database'::regclass
                OR d.objid <> (
                    SELECT oid FROM pg_catalog.pg_database
                     WHERE datname = current_database()
                )
                OR d.objsubid <> 0 OR d.deptype <> 'a')
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de roles rechazada: propiedad, ACL o dependencia externa';
    END IF;
END
$estado_inicial$;

DO $base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_contexto_actor_v1_runtime, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_propietario',
        current_database()
    );
END
$base$;
REVOKE vec_contexto_actor_v1_propietario
    FROM vec_contexto_actor_v1_migrador;

DO $estado_final$
DECLARE
    roles oid[] := ARRAY[
        'vec_contexto_actor_v1_propietario'::regrole,
        'vec_contexto_actor_v1_migrador'::regrole,
        'vec_contexto_actor_v1_runtime'::regrole
    ];
BEGIN
    IF (
        SELECT count(*) <> 3 FROM pg_catalog.pg_authid AS r
         WHERE r.oid = ANY (roles)
           AND NOT r.rolsuper AND NOT r.rolinherit
           AND NOT r.rolcreaterole AND NOT r.rolcreatedb
           AND NOT r.rolcanlogin AND NOT r.rolreplication
           AND NOT r.rolbypassrls AND r.rolconnlimit = -1
           AND r.rolvaliduntil IS NULL AND r.rolpassword IS NULL
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_db_role_setting
         WHERE setrole = 0 OR setrole = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members
         WHERE roleid = ANY (roles) OR member = ANY (roles)
            OR grantor = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shdepend
         WHERE refclassid = 'pg_catalog.pg_authid'::regclass
           AND refobjid = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shdescription
         WHERE classoid = 'pg_catalog.pg_authid'::regclass
           AND objoid = ANY (roles)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_shseclabel
         WHERE classoid = 'pg_catalog.pg_authid'::regclass
           AND objoid = ANY (roles)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de roles rechazada: revalidacion final fallida';
    END IF;
END
$estado_final$;

DROP ROLE vec_contexto_actor_v1_runtime;
DROP ROLE vec_contexto_actor_v1_migrador;
DROP ROLE vec_contexto_actor_v1_propietario;
COMMIT;
