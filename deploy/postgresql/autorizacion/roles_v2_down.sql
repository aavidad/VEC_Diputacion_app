-- Ejecutar solo despues de 000003_proyeccion_motivos_autorizacion_v2.down.sql
-- y de retirar todas las membresias LOGIN. DROP ROLE falla cerrado ante
-- cualquier dependencia; no se usa CASCADE.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0)
);

-- PostgreSQL puede retirar aristas de membresia al eliminar un rol. Esa
-- comodidad no es aceptable en una retirada gobernada: ninguna relacion se
-- adopta como propia ni se borra implicitamente. Primero se bloquea pg_authid
-- para impedir que un GRANT concurrente resuelva y conserve OID obsoletos;
-- despues pg_auth_members serializa la arista hasta que DROP ROLE confirma.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    esperado record;
    oid_dba_ejecutor oid;
    oid_proyector oid;
    oid_evaluador oid;
    oid_dueno_base oid;
    roles_v2 oid[];
    cantidad_membresias integer;
    diferencias_acl integer;
BEGIN
    SELECT oid
      INTO oid_dba_ejecutor
      FROM pg_catalog.pg_authid
     WHERE rolname = current_user
       AND rolsuper IS TRUE;
    IF oid_dba_ejecutor IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'retirada V2 rechazada: el principal DBA no es superusuario';
    END IF;

    FOR esperado IN
        SELECT *
          FROM (VALUES
              ('vec_autorizacion_motivos_proyector'::text),
              ('vec_autorizacion_motivos_evaluador'::text)
          ) AS opciones(rol)
    LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_authid
             WHERE rolname = esperado.rol
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
               AND NOT EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_db_role_setting AS ajuste
                    WHERE ajuste.setrole = pg_authid.oid
               )
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '55000',
                MESSAGE = 'retirada V2 rechazada: falta un rol o sus atributos cambiaron',
                DETAIL = esperado.rol;
        END IF;
    END LOOP;

    SELECT proyector.oid, evaluador.oid
      INTO STRICT oid_proyector, oid_evaluador
      FROM pg_catalog.pg_authid AS proyector
      CROSS JOIN pg_catalog.pg_authid AS evaluador
     WHERE proyector.rolname = 'vec_autorizacion_motivos_proyector'
       AND evaluador.rolname = 'vec_autorizacion_motivos_evaluador';
    roles_v2 := ARRAY[oid_proyector, oid_evaluador];

    -- No basta con mirar grupo y miembro: un rol retirado usado como otorgante
    -- tambien dejaria una relacion ajena que DROP ROLE no debe borrar.
    SELECT count(*)
      INTO cantidad_membresias
      FROM pg_catalog.pg_auth_members AS relacion
     WHERE relacion.roleid = ANY(roles_v2)
        OR relacion.member = ANY(roles_v2)
        OR relacion.grantor = ANY(roles_v2);

    IF cantidad_membresias > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada V2 rechazada: existen membresias; no se eliminan implicitamente',
            DETAIL = format('%s relaciones usan un rol V2 como grupo, miembro u otorgante', cantidad_membresias);
    END IF;

    SELECT base.datdba
      INTO STRICT oid_dueno_base
      FROM pg_catalog.pg_database AS base
      JOIN pg_catalog.pg_authid AS dueno ON dueno.oid = base.datdba
     WHERE base.datname = current_database()
       AND base.datdba <> ALL(roles_v2);

    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
         CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
           AND (
               acl.grantee = ANY(roles_v2)
               OR acl.grantor = ANY(roles_v2)
           )
    ), esperado_acl(grantor, grantee, privilege_type, is_grantable) AS (
        VALUES
            (oid_dueno_base, oid_proyector, 'CONNECT'::text, false),
            (oid_dueno_base, oid_evaluador, 'CONNECT'::text, false)
    ), sobrantes AS (
        SELECT * FROM actual
        EXCEPT ALL
        SELECT * FROM esperado_acl
    ), ausentes AS (
        SELECT * FROM esperado_acl
        EXCEPT ALL
        SELECT * FROM actual
    )
    SELECT (SELECT count(*) FROM sobrantes)
         + (SELECT count(*) FROM ausentes)
      INTO diferencias_acl;

    IF diferencias_acl <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada V2 rechazada: las ACL de los roles no son las esperadas',
            DETAIL = format('%s diferencias entre privilegios esperados y observados', diferencias_acl);
    END IF;
END
$prevalidacion$;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_motivos_evaluador',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_motivos_proyector',
        current_database()
    );
END
$privilegios_base$;

DROP ROLE vec_autorizacion_motivos_evaluador;
DROP ROLE vec_autorizacion_motivos_proyector;

COMMIT;
