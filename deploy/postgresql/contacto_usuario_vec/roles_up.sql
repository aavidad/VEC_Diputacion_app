\set ON_ERROR_STOP on
-- Provisión DBA única. No crea identidades LOGIN ni concede perfiles VEC.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:roles:v1', 0));

DO $precondicion$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='la provisión de contacto requiere DBA';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (
        'vec_contacto_usuario_owner', 'vec_contacto_usuario_writer',
        'vec_contacto_usuario_reader', 'vec_contacto_usuario_migrador')) THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='los roles de contacto ya existen';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles
        WHERE rolname='vec_autorizacion_atestada_v3_propietario' AND NOT rolcanlogin) THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='falta el propietario de autorización V3';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_database b
        CROSS JOIN LATERAL aclexplode(coalesce(b.datacl, acldefault('d', b.datdba))) a
        WHERE b.datname=current_database() AND a.grantee=0 AND a.privilege_type='CREATE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='la base no debe conceder CREATE a PUBLIC';
    END IF;
END
$precondicion$;

CREATE ROLE vec_contacto_usuario_owner NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contacto_usuario_writer NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contacto_usuario_reader NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contacto_usuario_migrador NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_contacto_usuario_owner TO vec_contacto_usuario_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $conexion$
DECLARE rol text;
BEGIN
    EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_contacto_usuario_owner', current_database());
    FOREACH rol IN ARRAY ARRAY['vec_contacto_usuario_writer', 'vec_contacto_usuario_reader', 'vec_contacto_usuario_migrador'] LOOP
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), rol);
    END LOOP;
END
$conexion$;
COMMIT;
