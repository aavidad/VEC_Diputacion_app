\set ON_ERROR_STOP on
-- CANDIDATA: provisión DBA anterior a AD3-33 y ADMIN1; no acredita instalación.
-- No crea cuentas LOGIN ni credenciales. ADMIN1 asume primero el migrador y
-- después el propietario, antes de comprobar AD3-33 y crear el esquema.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_administracion:roles_up:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1',0));

DO $precondicion$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR current_setting('server_version_num')::integer < 160000 THEN
        RAISE EXCEPTION 'ADMIN roles: requiere provisión DBA y PostgreSQL compatible'
            USING ERRCODE='42501';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (
        'vec_administracion_propietario','vec_administracion_migrador',
        'vec_administracion_ejecutor'))
       OR to_regnamespace('vec_administracion') IS NOT NULL THEN
        RAISE EXCEPTION 'ADMIN roles: roles o esquema ya existentes; no modificar'
            USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_namespace n JOIN pg_roles r ON r.oid=n.nspowner
         WHERE n.nspname='vec_autorizacion_atestada_v3'
           AND r.rolname='vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin AND NOT r.rolinherit AND NOT r.rolsuper
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolreplication AND NOT r.rolbypassrls
    ) THEN
        RAISE EXCEPTION 'ADMIN roles: esquema o propietario AD3 incompatibles'
            USING ERRCODE='55000';
    END IF;
    -- No modificar ACL históricas para compensar una base que concede DDL
    -- a PUBLIC: esa preimagen no permite aislar el ejecutor administrativo.
    IF EXISTS (
        SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(
            coalesce(d.datacl,acldefault('d',d.datdba))) a
         WHERE d.datname=current_database() AND a.grantee=0
           AND a.privilege_type='CREATE'
    ) THEN
        RAISE EXCEPTION 'ADMIN roles: CREATE de base concedido a PUBLIC'
            USING ERRCODE='55000';
    END IF;
END $precondicion$;

CREATE ROLE vec_administracion_propietario
    NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_administracion_migrador
    NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_administracion_ejecutor
    NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS;

-- El migrador puede asumir el propietario de forma explícita, sin heredar
-- privilegios ni delegar membresías. El ejecutor no recibe ningún otro rol.
GRANT vec_administracion_propietario TO vec_administracion_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO '
        ||'vec_administracion_migrador,vec_administracion_ejecutor',current_database());
    -- CREATE aquí permite crear el esquema en esta base; no es CREATEDB.
    EXECUTE format('GRANT CREATE ON DATABASE %I TO vec_administracion_propietario',
        current_database());
END $privilegios$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_administracion_propietario;

DO $postcondicion$
DECLARE
    propietario oid := 'vec_administracion_propietario'::regrole;
    migrador oid := 'vec_administracion_migrador'::regrole;
    ejecutor oid := 'vec_administracion_ejecutor'::regrole;
BEGIN
    IF (SELECT count(*) FROM pg_roles
         WHERE oid IN (propietario,migrador,ejecutor)
           AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
           AND NOT rolcreatedb AND NOT rolcreaterole
           AND NOT rolreplication AND NOT rolbypassrls) <> 3
       OR (SELECT count(*) FROM pg_auth_members
            WHERE roleid IN (propietario,migrador,ejecutor)
               OR member IN (propietario,migrador,ejecutor)) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_auth_members
                       WHERE roleid=propietario AND member=migrador
                         AND grantor=current_user::regrole
                         AND NOT admin_option AND NOT inherit_option AND set_option)
       OR NOT has_database_privilege(migrador,current_database(),'CONNECT')
       OR NOT has_database_privilege(ejecutor,current_database(),'CONNECT')
       OR NOT has_database_privilege(propietario,current_database(),'CREATE')
       OR has_database_privilege(migrador,current_database(),'CREATE')
       OR has_database_privilege(ejecutor,current_database(),'CREATE')
       OR NOT has_schema_privilege(propietario,'vec_autorizacion_atestada_v3','USAGE')
       OR has_schema_privilege(propietario,'vec_autorizacion_atestada_v3','CREATE')
       OR has_schema_privilege(ejecutor,'vec_autorizacion_atestada_v3','CREATE')
       OR to_regnamespace('vec_administracion') IS NOT NULL THEN
        RAISE EXCEPTION 'ADMIN roles: postimagen de roles o privilegios divergente'
            USING ERRCODE='55000';
    END IF;
END $postcondicion$;
COMMIT;
