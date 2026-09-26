-- Selección: bootstrap DBA de una sola ejecución. Crea los grupos técnicos
-- NOLOGIN del módulo. No crea identidades LOGIN: vec-server reutiliza la del
-- ejecutor de Bolsa con una sola membresía nueva, que concede el operador
-- después (deploy/principal/03_entorno.md):
--   GRANT vec_seleccion_ejecutor TO <LOGIN ejecutor de Bolsa>;
-- Requiere el núcleo de autorización atestada V3 instalado (sus roles).
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_seleccion:roles_up:v1', 0));

DO $prevalidacion$
DECLARE encontrados text[];
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper) THEN
  RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'el bootstrap de Selección requiere superusuario';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'vec_autorizacion_atestada_v3_propietario' AND NOT rolcanlogin) THEN
  RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'falta el núcleo de autorización atestada V3';
 END IF;
 SELECT array_agg(rolname::text ORDER BY rolname) INTO encontrados FROM pg_catalog.pg_roles
  WHERE rolname::text = ANY (ARRAY['vec_seleccion_propietario','vec_seleccion_migrador','vec_seleccion_ejecutor']);
 IF cardinality(encontrados) > 0 THEN
  RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'ya existen roles de Selección', DETAIL = array_to_string(encontrados, ',');
 END IF;
END $prevalidacion$;

CREATE ROLE vec_seleccion_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_seleccion_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_seleccion_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_seleccion_propietario TO vec_seleccion_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

DO $conexiones$
BEGIN
 EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_seleccion_propietario', current_database());
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_seleccion_migrador', current_database());
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_seleccion_ejecutor', current_database());
END $conexiones$;

-- Ninguna función nueva del propietario queda ejecutable por PUBLIC.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_seleccion_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
COMMIT;
