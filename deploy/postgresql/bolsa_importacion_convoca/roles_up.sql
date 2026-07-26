-- Bootstrap DBA. Las cuentas LOGIN se aprovisionan fuera del repositorio y
-- reciben un único grupo técnico mínimo.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_importacion_convoca:roles:v1', 0)
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
            MESSAGE = 'bootstrap de importacion Convoca requiere superusuario';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base,
               LATERAL pg_catalog.aclexplode(
                   COALESCE(base.datacl, pg_catalog.acldefault('d', base.datdba))
               ) AS permiso
         WHERE base.datname = current_database()
           AND permiso.grantee = 0
           AND permiso.privilege_type IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: la base conserva privilegios para PUBLIC';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio,
               LATERAL pg_catalog.aclexplode(
                   COALESCE(espacio.nspacl, pg_catalog.acldefault('n', espacio.nspowner))
               ) AS permiso
         WHERE espacio.nspname = 'public'
           AND permiso.grantee = 0
           AND permiso.privilege_type = 'CREATE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: el esquema public permite CREATE a PUBLIC';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_importacion_convoca') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existe el esquema de importacion Convoca';
    END IF;
    SELECT array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname::text = ANY (ARRAY[
         'vec_bolsa_importacion_convoca_propietario',
         'vec_bolsa_importacion_convoca_migrador',
         'vec_bolsa_importacion_convoca_ejecutor',
         'vec_bolsa_importacion_convoca_recuperador',
         'vec_bolsa_importacion_convoca_conciliador',
         'vec_bolsa_importacion_convoca_retencion',
         'vec_bolsa_importacion_convoca_gobernanza'
     ]);
    IF cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'bootstrap rechazado: existen roles de importacion Convoca',
            DETAIL = array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_importacion_convoca_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_ejecutor NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_recuperador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_conciliador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_retencion NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_importacion_convoca_gobernanza NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_bolsa_importacion_convoca_propietario
    TO vec_bolsa_importacion_convoca_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_bolsa_importacion_convoca_propietario',
        current_database()
    );
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_bolsa_importacion_convoca_migrador, vec_bolsa_importacion_convoca_ejecutor, vec_bolsa_importacion_convoca_recuperador, vec_bolsa_importacion_convoca_conciliador, vec_bolsa_importacion_convoca_retencion, vec_bolsa_importacion_convoca_gobernanza',
        current_database()
    );
END
$privilegios_base$;

CREATE SCHEMA vec_bolsa_importacion_convoca
    AUTHORIZATION vec_bolsa_importacion_convoca_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
DO $retirar_create_base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CREATE ON DATABASE %I FROM vec_bolsa_importacion_convoca_propietario',
        pg_catalog.current_database()
    );
END
$retirar_create_base$;

ALTER ROLE vec_bolsa_importacion_convoca_ejecutor
    SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_importacion_convoca_recuperador
    SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_importacion_convoca_conciliador
    SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_importacion_convoca_retencion
    SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_bolsa_importacion_convoca_gobernanza
    SET search_path = 'pg_catalog,pg_temp';

COMMIT;
