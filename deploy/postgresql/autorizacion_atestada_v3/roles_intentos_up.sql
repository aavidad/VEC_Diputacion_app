-- DBA, una sola vez antes de AD169. El LOGIN de la aplicación se provisiona
-- fuera de Git; ninguna credencial ni membresía se crea aquí.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:roles:intentos:v1', 0));
DO $pre$
DECLARE
    v_rol record;
BEGIN
    SELECT rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO v_rol FROM pg_catalog.pg_roles WHERE rolname=current_user;
    IF v_rol.rolsuper IS NOT TRUE
       OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_registrador_intentos') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='precondiciones rol intentos no satisfechas';
    END IF;
END
$pre$;
CREATE ROLE vec_autorizacion_atestada_v3_registrador_intentos
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
DO $base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v3_registrador_intentos',
        pg_catalog.current_database());
END
$base$;
COMMIT;
