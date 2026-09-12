BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:roles_up:v1', 0));

DO $prevalidacion$
DECLARE encontrados text[];
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN
    RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'bootstrap Personal rechazado: requiere superusuario';
  END IF;
  SELECT array_agg(rolname ORDER BY rolname) INTO encontrados FROM pg_roles
   WHERE rolname = ANY (ARRAY['vec_personal_propietario','vec_personal_migrador','vec_personal_ejecutor']);
  IF cardinality(encontrados) > 0 OR to_regnamespace('vec_personal') IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'bootstrap Personal rechazado: roles o esquema preexistentes incompatibles';
  END IF;
END $prevalidacion$;

CREATE ROLE vec_personal_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_personal_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_personal_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_propietario TO vec_personal_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

CREATE SCHEMA vec_personal AUTHORIZATION vec_personal_propietario;
REVOKE ALL ON SCHEMA vec_personal FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_migrador, vec_personal_ejecutor;
-- El propietario es dedicado: estas ACL globales solo gobiernan sus futuros
-- objetos y cierran también EXECUTE/USAGE que PostgreSQL concede por defecto.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario REVOKE USAGE ON TYPES FROM PUBLIC;
COMMIT;
