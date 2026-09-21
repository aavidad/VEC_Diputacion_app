-- Delta DBA: grupo técnico exclusivo para registrar rechazos de frontera Bolsa.
-- Las identidades LOGIN nominales y su pertenencia se aprovisionan fuera de Git.
BEGIN;
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

COMMIT;
