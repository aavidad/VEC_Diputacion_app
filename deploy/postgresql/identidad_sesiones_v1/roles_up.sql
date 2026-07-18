-- Bootstrap DBA para el registro durable de identidad. No crea LOGIN ni
-- adopta roles preexistentes. Requiere una base VEC dedicada y endurecida.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_identidad_sesiones_v1:roles_up:v1', 0)
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap de identidad rechazado: requiere superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_migrador'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta el nucleo de autorizacion requerido';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = extension.extnamespace
         WHERE extension.extname = 'pgcrypto'
           AND espacio.nspname = 'public'
    ) OR pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'public.digest(bytea,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta pgcrypto instalado en el esquema public',
            HINT = 'el DBA debe instalar pgcrypto antes de este bootstrap';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.nspname = 'public'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'USAGE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'el esquema public conserva USAGE para PUBLIC';
    END IF;

    SELECT array_agg(rolname::text ORDER BY rolname::text COLLATE "C")
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_identidad_sesiones_v1_propietario',
         'vec_identidad_sesiones_v1_migrador',
         'vec_identidad_sesiones_v1_provisionador',
         'vec_identidad_sesiones_v1_registrador',
         'vec_identidad_sesiones_v1_revalidador',
         'vec_identidad_sesiones_v1_revocador'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles de identidad',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_identidad_sesiones_v1_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_provisionador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_registrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_revalidador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_revocador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_identidad_sesiones_v1_propietario
    TO vec_identidad_sesiones_v1_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- La extension no es una API de ejecucion. Solo el propietario NOLOGIN puede
-- usar los dos overloads exactos desde funciones SECURITY DEFINER cerradas.
REVOKE ALL ON FUNCTION public.gen_random_bytes(integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.digest(bytea, text) FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer)
    TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION public.digest(bytea, text)
    TO vec_identidad_sesiones_v1_propietario;

DO $privilegios_base$
DECLARE
    rol text;
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_identidad_sesiones_v1_propietario',
        current_database()
    );
    FOREACH rol IN ARRAY ARRAY[
        'vec_identidad_sesiones_v1_migrador',
        'vec_identidad_sesiones_v1_provisionador',
        'vec_identidad_sesiones_v1_registrador',
        'vec_identidad_sesiones_v1_revalidador',
        'vec_identidad_sesiones_v1_revocador'
    ] LOOP
        EXECUTE format(
            'GRANT CONNECT ON DATABASE %I TO %I', current_database(), rol
        );
    END LOOP;
END
$privilegios_base$;
COMMIT;
