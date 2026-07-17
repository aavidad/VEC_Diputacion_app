-- Retirar solo despues de ejecutar el down de migracion y eliminar todas las
-- cuentas LOGIN miembro del lector. No usa CASCADE.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_confianza_atestacion_v2:roles_down:v1', 0)
);

DO $prevalidacion$
DECLARE
    propietario oid;
    migrador oid;
    lector oid;
    oid_dba oid;
    membresias bigint;
BEGIN
    IF pg_catalog.to_regnamespace('vec_confianza_atestacion_v2') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: el catalogo sigue instalado';
    END IF;
    SELECT oid INTO oid_dba
      FROM pg_catalog.pg_authid
     WHERE rolname = current_user AND rolsuper IS TRUE;
    IF oid_dba IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down de roles de confianza V2 requiere superusuario';
    END IF;

    SELECT oid INTO propietario FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_confianza_atestacion_v2_propietario'
       AND rolcanlogin IS FALSE AND rolsuper IS FALSE
       AND rolcreatedb IS FALSE AND rolcreaterole IS FALSE
       AND rolinherit IS FALSE AND rolreplication IS FALSE
       AND rolbypassrls IS FALSE AND rolpassword IS NULL
       AND rolvaliduntil IS NULL;
    SELECT oid INTO migrador FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_confianza_atestacion_v2_migrador'
       AND rolcanlogin IS FALSE AND rolsuper IS FALSE
       AND rolcreatedb IS FALSE AND rolcreaterole IS FALSE
       AND rolinherit IS FALSE AND rolreplication IS FALSE
       AND rolbypassrls IS FALSE AND rolpassword IS NULL
       AND rolvaliduntil IS NULL;
    SELECT oid INTO lector FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_confianza_atestacion_v2_lector_autoridad'
       AND rolcanlogin IS FALSE AND rolsuper IS FALSE
       AND rolcreatedb IS FALSE AND rolcreaterole IS FALSE
       AND rolinherit IS TRUE AND rolreplication IS FALSE
       AND rolbypassrls IS FALSE AND rolpassword IS NULL
       AND rolvaliduntil IS NULL;
    IF propietario IS NULL OR migrador IS NULL OR lector IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: faltan roles exactos de confianza V2';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_db_role_setting
         WHERE setrole IN (propietario, migrador, lector)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen ajustes de sesion en roles V2';
    END IF;

    SELECT count(*) INTO membresias
      FROM pg_catalog.pg_auth_members
     WHERE roleid IN (propietario, migrador, lector)
        OR member IN (propietario, migrador, lector)
        OR grantor IN (propietario, migrador, lector);
    IF membresias <> 1 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members
         WHERE roleid = propietario
           AND member = migrador
           AND admin_option IS FALSE
           AND inherit_option IS FALSE
           AND set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron las membresias de confianza V2';
    END IF;

    IF pg_catalog.to_regprocedure(
           'vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos()'
       ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.pronamespace = espacio.oid
          JOIN pg_catalog.pg_event_trigger AS disparador
            ON disparador.evtfoid = funcion.oid
         WHERE espacio.nspname = 'vec_confianza_atestacion_v2_guardia'
           AND espacio.nspowner = oid_dba
           AND funcion.proname = 'cerrar_acl_tipos'
           AND funcion.proowner = oid_dba
           AND funcion.prosecdef IS TRUE
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog'
           ]::text[]
           AND disparador.evtname =
               'vec_confianza_atestacion_v2_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL no es la esperada';
    END IF;
END
$prevalidacion$;

-- Devuelve los default ACL globales a los valores nativos antes de retirar el
-- rol dedicado. Los ACL ligados al esquema desaparecieron con DROP SCHEMA.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_confianza_atestacion_v2_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP EVENT TRIGGER vec_confianza_atestacion_v2_cerrar_acl_tipos;
DROP FUNCTION
    vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos()
    RESTRICT;
DROP SCHEMA vec_confianza_atestacion_v2_guardia RESTRICT;

DO $retirar_privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE CONNECT, CREATE ON DATABASE %I FROM vec_confianza_atestacion_v2_propietario',
        current_database()
    );
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_confianza_atestacion_v2_migrador',
        current_database()
    );
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_confianza_atestacion_v2_lector_autoridad',
        current_database()
    );
END
$retirar_privilegios_base$;

REVOKE vec_confianza_atestacion_v2_propietario
    FROM vec_confianza_atestacion_v2_migrador;
DROP ROLE vec_confianza_atestacion_v2_lector_autoridad;
DROP ROLE vec_confianza_atestacion_v2_migrador;
DROP ROLE vec_confianza_atestacion_v2_propietario;
COMMIT;
