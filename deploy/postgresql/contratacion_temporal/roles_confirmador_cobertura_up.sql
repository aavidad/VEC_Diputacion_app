-- Delta DBA para instalaciones existentes 000001-000027.
-- Orden: este archivo, migraciones 000028+, aprovisionamiento del LOGIN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:rol_confirmador_cobertura:v1',0
  )
);
DO $delta$
DECLARE
  v_barrera_final boolean:=false;
  v_controles integer;
  v_version numeric;
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=current_user AND rolsuper
  ) THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='delta de rol confirmador requiere superusuario';
  END IF;
  IF pg_catalog.to_regclass(
       'vec_contratacion_temporal.control_migracion_cobertura_o4'
     ) IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='delta de rol fuera de una instalación CT O4';
  END IF;
  EXECUTE
    'SELECT count(*),min(version_esquema) FROM '
    ||'vec_contratacion_temporal.control_migracion_cobertura_o4 '
    ||'WHERE control'
    INTO v_controles,v_version;
  IF v_controles<>1 OR v_version NOT BETWEEN 7 AND 14 THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='barrera CT incompatible con delta de rol confirmador';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
  ) THEN
    EXECUTE 'CREATE ROLE vec_contratacion_temporal_confirmador_cobertura '
      ||'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
      ||'NOREPLICATION NOBYPASSRLS';
  END IF;
  v_barrera_final:=v_version=14;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
       AND NOT rolbypassrls
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_auth_members m
    JOIN pg_catalog.pg_roles r ON r.oid=m.member
     WHERE r.rolname='vec_contratacion_temporal_confirmador_cobertura'
  ) OR (
    EXISTS(
      SELECT 1 FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
       WHERE r.rolname='vec_contratacion_temporal_confirmador_cobertura'
    ) AND NOT v_barrera_final
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_auth_members m
    JOIN pg_catalog.pg_roles grupo ON grupo.oid=m.roleid
    JOIN pg_catalog.pg_roles miembro ON miembro.oid=m.member
     WHERE grupo.rolname=
       'vec_contratacion_temporal_confirmador_cobertura'
       AND (
         m.admin_option OR NOT m.inherit_option OR m.set_option
         OR NOT miembro.rolcanlogin OR miembro.rolsuper
         OR miembro.rolcreatedb OR miembro.rolcreaterole
         OR miembro.rolreplication OR miembro.rolbypassrls
       )
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='rol confirmador existente no es grupo mínimo';
  END IF;
  EXECUTE pg_catalog.format(
    'GRANT CONNECT ON DATABASE %I TO '
    ||'vec_contratacion_temporal_confirmador_cobertura',
    pg_catalog.current_database()
  );
END
$delta$;
COMMIT;
