-- Ejecutar una sola vez por base, con psql -X -v ON_ERROR_STOP=1 -1 -f.
-- No instala migraciones ni asigna permisos V3 a personas. Las contraseñas
-- se establecen por el gestor privado de secretos del DBA.
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(97013, 3);

DO $preparar$
DECLARE
  nombre text;
  grupo text;
  existente record;
BEGIN
  FOR nombre, grupo IN
    SELECT * FROM (VALUES
      ('vec_bolsa_inscripciones_lector', 'vec_bolsa_llamamientos_lector_inscripciones'),
      ('vec_bolsa_inscripciones_rrhh_lector', 'vec_bolsa_llamamientos_lector_inscripciones_rrhh'),
      ('vec_externo_bolsa_desarrollo', 'vec_bolsa_llamamientos_portal_externo')
    ) AS pares(nombre, grupo)
  LOOP
    IF pg_catalog.to_regrole(grupo) IS NULL THEN
      RAISE EXCEPTION 'Falta el grupo de una migración previa: %', grupo;
    END IF;
    IF pg_catalog.to_regrole(nombre) IS NULL THEN
      EXECUTE pg_catalog.format('CREATE ROLE %I LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS', nombre);
    END IF;
    SELECT rolcanlogin, rolinherit, rolsuper, rolcreatedb, rolcreaterole,
           rolreplication, rolbypassrls INTO existente
      FROM pg_catalog.pg_roles WHERE rolname = nombre;
    IF NOT existente.rolcanlogin OR NOT existente.rolinherit OR existente.rolsuper
       OR existente.rolcreatedb OR existente.rolcreaterole
       OR existente.rolreplication OR existente.rolbypassrls THEN
      RAISE EXCEPTION 'LOGIN con atributos incompatibles: %', nombre;
    END IF;
    IF EXISTS (
      SELECT 1 FROM pg_catalog.pg_auth_members m
      WHERE m.member = pg_catalog.to_regrole(nombre)
        AND m.roleid <> pg_catalog.to_regrole(grupo)
    ) THEN
      RAISE EXCEPTION 'LOGIN con membresía adicional: %', nombre;
    END IF;
    IF NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_auth_members m
      WHERE m.member = pg_catalog.to_regrole(nombre)
        AND m.roleid = pg_catalog.to_regrole(grupo)
    ) THEN
      EXECUTE pg_catalog.format('GRANT %I TO %I WITH INHERIT TRUE, SET FALSE, ADMIN FALSE', grupo, nombre);
    END IF;
    IF EXISTS (
      SELECT 1 FROM pg_catalog.pg_auth_members m
      WHERE m.member = pg_catalog.to_regrole(nombre)
        AND m.roleid = pg_catalog.to_regrole(grupo)
        AND (NOT m.inherit_option OR m.set_option OR m.admin_option)
    ) THEN
      RAISE EXCEPTION 'Membresía incompatible: %', nombre;
    END IF;
  END LOOP;
END
$preparar$;
