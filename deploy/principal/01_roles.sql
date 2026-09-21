\set ON_ERROR_STOP on
BEGIN;
-- Delta DBA: grupo técnico exclusivo para registrar rechazos de frontera Bolsa.
-- La contraseña del LOGIN nominal entra por variable psql y nunca se versiona.
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_llamamientos:rol_registrador_frontera:v1', 0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_bolsa_llamamientos') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'delta del registrador de frontera requiere DBA sobre Bolsa';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_registrador_frontera'
    ) THEN
        EXECUTE
            'CREATE ROLE vec_bolsa_llamamientos_registrador_frontera '
            || 'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
            || 'NOREPLICATION NOBYPASSRLS';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_registrador_frontera'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
        JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
        WHERE miembro.rolname = 'vec_bolsa_llamamientos_registrador_frontera'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles rol
         WHERE rol.oid <> 'vec_bolsa_llamamientos_registrador_frontera'::regrole
           AND pg_catalog.pg_has_role(
               'vec_bolsa_llamamientos_registrador_frontera', rol.oid, 'MEMBER'
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de frontera existente no es mínimo';
    END IF;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_registrador_frontera',
        pg_catalog.current_database()
    );
END
$delta$;


-- LOGIN nominal de desarrollo para la conexión segregada de auditoría B-BACK/B2.
SELECT pg_catalog.format(
    'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS',
    'vec_b2_auditoria_frontera_desarrollo',
    :'bolsa_auditoria_password'
)
WHERE NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_roles
    WHERE rolname = 'vec_b2_auditoria_frontera_desarrollo'
)
\gexec

ALTER ROLE vec_b2_auditoria_frontera_desarrollo
    WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS PASSWORD :'bolsa_auditoria_password';
GRANT vec_bolsa_llamamientos_registrador_frontera
    TO vec_b2_auditoria_frontera_desarrollo
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;

DO $comprobar_login$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_roles identidad
        JOIN pg_catalog.pg_auth_members membresia ON membresia.member = identidad.oid
        JOIN pg_catalog.pg_roles grupo ON grupo.oid = membresia.roleid
        WHERE identidad.rolname = 'vec_b2_auditoria_frontera_desarrollo'
          AND identidad.rolcanlogin AND identidad.rolinherit
          AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
          AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
          AND NOT identidad.rolbypassrls
          AND grupo.rolname = 'vec_bolsa_llamamientos_registrador_frontera'
          AND membresia.admin_option IS FALSE
          AND membresia.inherit_option IS TRUE
          AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'LOGIN de auditoría de frontera Bolsa no quedó cerrado';
    END IF;
END
$comprobar_login$;

:finalizar;
