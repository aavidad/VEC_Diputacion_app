-- Rol técnico nominal para la consulta exterior B11. Las identidades LOGIN se
-- administran fuera de Git; este fichero no crea cuentas ni concesiones a ellas.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:roles:b11:up', 0)
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap B11 requiere superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole
           AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta propietario de Bolsa para B11';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_consultor_participaciones_propias'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol exterior B11 ya existe';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_llamamientos_consultor_participaciones_propias
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
COMMIT;
