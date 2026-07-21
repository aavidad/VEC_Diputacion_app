-- Bootstrap DBA para el registro durable de contexto de actor V1.
BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto actor V1 requiere bootstrap superusuario';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = ANY (ARRAY[
            'vec_contexto_actor_v1_propietario',
            'vec_contexto_actor_v1_migrador',
            'vec_contexto_actor_v1_runtime'
        ])
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'roles de contexto actor V1 ya existentes';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_database b
        CROSS JOIN LATERAL pg_catalog.aclexplode(
          coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
        ) a
         WHERE b.datname=current_database() AND a.grantee=0
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace n
        CROSS JOIN LATERAL pg_catalog.aclexplode(
          coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
        ) a
         WHERE n.nspname='public' AND a.grantee=0
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'la base dedicada debe llegar sin privilegios de PUBLIC';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_contexto_actor_v1_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contexto_actor_v1_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_contexto_actor_v1_runtime NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_contexto_actor_v1_propietario
    TO vec_contexto_actor_v1_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $base$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_propietario', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime', current_database());
END
$base$;
COMMIT;
