\set ON_ERROR_STOP on
-- Grupo exclusivo del LOGIN de ContextoActor del portal candidato. La clave y
-- el LOGIN se provisionan fuera de Git; nunca se concede el runtime general.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $preimagen$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
       OR pg_catalog.to_regrole('vec_contexto_actor_v1_candidato_externo') IS NOT NULL
       OR pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') IS NULL THEN
        RAISE EXCEPTION 'roles candidato externo ContextoActor: preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $preimagen$;
CREATE ROLE vec_contexto_actor_v1_candidato_externo NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $base$ BEGIN
    EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_candidato_externo', current_database());
END $base$;
COMMIT;
