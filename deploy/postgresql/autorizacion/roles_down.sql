-- Ejecutar solo despues de 000001_autorizacion.down.sql y de retirar todas las
-- membresias LOGIN. DROP ROLE falla cerrado ante cualquier dependencia; no se
-- usa CASCADE ni se eliminan relaciones de membresia implicitamente.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:roles_down:v2', 0)
);

-- Los catalogos de roles son compartidos por todo el cluster. El orden es
-- deliberado: pg_authid impide que un GRANT concurrente conserve OID que van a
-- desaparecer y pg_auth_members serializa despues todas las aristas hasta el
-- COMMIT. Requiere superusuario y una ventana de mantenimiento coordinada.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
-- La ACL de la base tambien forma parte del inventario destructivo. Se congela
-- despues de los catalogos de roles para que no cambie entre validar y revocar.
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    esperado record;
    oid_dba_ejecutor oid;
    oid_otorgante_gobernado oid;
    oid_propietario oid;
    oid_migrador oid;
    oid_fuente oid;
    oid_registro oid;
    oid_dueno_base oid;
    roles_vec oid[];
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
            MESSAGE = 'retirada rechazada: el principal DBA no es un superusuario gobernado';
    END IF;

    -- PostgreSQL atribuye a su superusuario bootstrap las concesiones de rol
    -- emitidas por un superusuario, aunque la sesion use otro principal DBA.
    -- Su OID es una constante interna del cluster y sobrevive a renombrados.
    SELECT oid
      INTO oid_otorgante_gobernado
      FROM pg_catalog.pg_authid
     WHERE oid = 10
       AND rolsuper IS TRUE;
    IF oid_otorgante_gobernado IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: no existe el otorgante bootstrap gobernado';
    END IF;

    -- Se inventarian atributos que el bootstrap fijo. Un homonimo o un rol
    -- alterado no se adopta como propio durante una retirada destructiva.
    FOR esperado IN
        SELECT *
          FROM (VALUES
              ('vec_autorizacion_propietario'::text, false),
              ('vec_autorizacion_migrador'::text, false),
              ('vec_autorizacion_fuente'::text, true),
              ('vec_autorizacion_registro'::text, true)
          ) AS opciones(rol, hereda)
    LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_authid
             WHERE rolname = esperado.rol
               AND rolcanlogin IS FALSE
               AND rolsuper IS FALSE
               AND rolcreatedb IS FALSE
               AND rolcreaterole IS FALSE
               AND rolinherit IS NOT DISTINCT FROM esperado.hereda
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
                MESSAGE = 'retirada rechazada: falta un rol V1 o sus atributos cambiaron',
                DETAIL = esperado.rol;
        END IF;
    END LOOP;

    SELECT propietario.oid, migrador.oid, fuente.oid, registro.oid
      INTO STRICT oid_propietario, oid_migrador, oid_fuente, oid_registro
      FROM pg_catalog.pg_authid AS propietario
      CROSS JOIN pg_catalog.pg_authid AS migrador
      CROSS JOIN pg_catalog.pg_authid AS fuente
      CROSS JOIN pg_catalog.pg_authid AS registro
     WHERE propietario.rolname = 'vec_autorizacion_propietario'
       AND migrador.rolname = 'vec_autorizacion_migrador'
       AND fuente.rolname = 'vec_autorizacion_fuente'
       AND registro.rolname = 'vec_autorizacion_registro';
    roles_vec := ARRAY[
        oid_propietario, oid_migrador, oid_fuente, oid_registro
    ];

    -- Se inspeccionan los tres campos OID. Un rol VEC usado como otorgante de
    -- una relacion ajena tambien es inventario inesperado y debe bloquear la
    -- retirada. Solo se admite la arista estructural creada por roles_up, con
    -- sus opciones exactas y otorgada por el superusuario bootstrap gobernado.
    SELECT count(*)
      INTO cantidad_membresias
      FROM pg_catalog.pg_auth_members AS membresia
     WHERE membresia.roleid = ANY(roles_vec)
        OR membresia.member = ANY(roles_vec)
        OR membresia.grantor = ANY(roles_vec);

    IF cantidad_membresias <> 1 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
         WHERE membresia.roleid = oid_propietario
           AND membresia.member = oid_migrador
           AND membresia.grantor = oid_otorgante_gobernado
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: el inventario de membresias V1 no es el esperado',
            DETAIL = 'solo se admite propietario->migrador con opciones y otorgante gobernados';
    END IF;

    SELECT base.datdba
      INTO STRICT oid_dueno_base
      FROM pg_catalog.pg_database AS base
      JOIN pg_catalog.pg_authid AS dueno ON dueno.oid = base.datdba
     WHERE base.datname = current_database()
       AND base.datdba <> ALL(roles_vec);

    -- roles_up deja una ACL completa y determinista para una base dedicada. El
    -- otorgante de privilegios de objeto que ejecuta un superusuario es el
    -- propietario real de la base, no necesariamente current_user ni el OID 10.
    WITH actual AS (
        SELECT acl.grantor, acl.grantee, acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE base.datname = current_database()
    ), esperado_acl(grantor, grantee, privilege_type, is_grantable) AS (
        VALUES
            (oid_dueno_base, oid_dueno_base, 'CONNECT'::text, false),
            (oid_dueno_base, oid_dueno_base, 'CREATE'::text, false),
            (oid_dueno_base, oid_dueno_base, 'TEMPORARY'::text, false),
            (oid_dueno_base, oid_propietario, 'CONNECT'::text, false),
            (oid_dueno_base, oid_propietario, 'CREATE'::text, false),
            (oid_dueno_base, oid_migrador, 'CONNECT'::text, false),
            (oid_dueno_base, oid_fuente, 'CONNECT'::text, false),
            (oid_dueno_base, oid_registro, 'CONNECT'::text, false)
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
            MESSAGE = 'retirada rechazada: la ACL de la base no coincide con el inventario V1',
            DETAIL = format('%s diferencias entre privilegios esperados y observados', diferencias_acl);
    END IF;
END
$prevalidacion$;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_registro',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_fuente',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_migrador',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_propietario',
        current_database()
    );
END
$privilegios_base$;

REVOKE vec_autorizacion_propietario FROM vec_autorizacion_migrador;

-- Los bloqueos anteriores hacen estable esta comprobacion hasta el COMMIT. Si
-- REVOKE no retiro exactamente la arista prevalidada, no se ejecuta DROP ROLE.
DO $membresias_retiradas$
DECLARE
    roles_vec oid[] := ARRAY[
        'vec_autorizacion_propietario'::regrole::oid,
        'vec_autorizacion_migrador'::regrole::oid,
        'vec_autorizacion_fuente'::regrole::oid,
        'vec_autorizacion_registro'::regrole::oid
    ];
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
         WHERE membresia.roleid = ANY(roles_vec)
            OR membresia.member = ANY(roles_vec)
            OR membresia.grantor = ANY(roles_vec)
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: quedaron membresias V1 tras retirar la arista estructural';
    END IF;
END
$membresias_retiradas$;

DROP ROLE vec_autorizacion_registro;
DROP ROLE vec_autorizacion_fuente;
DROP ROLE vec_autorizacion_migrador;
DROP ROLE vec_autorizacion_propietario;

COMMIT;
