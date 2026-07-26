-- Rol DBA nominativo y sin efectos para la recuperación propia O4-05.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:rol_lector_resultado_cobertura:v1',0));
DO $delta$
DECLARE
  v_version numeric;
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=current_user AND rolsuper
  ) OR pg_catalog.to_regclass(
    'vec_contratacion_temporal.control_migracion_cobertura_o4'
  ) IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='delta del lector O4-05 requiere DBA sobre CT';
  END IF;
  EXECUTE
    'SELECT min(version_esquema) FROM '
    ||'vec_contratacion_temporal.control_migracion_cobertura_o4 '
    ||'HAVING count(*)=1 AND bool_and(control)'
    INTO v_version;
  IF v_version IS NULL OR v_version NOT IN (14,15) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='barrera CT incompatible con lector O4-05';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
  ) THEN
    EXECUTE
      'CREATE ROLE vec_contratacion_temporal_lector_resultado_cobertura '
      ||'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
      ||'NOREPLICATION NOBYPASSRLS';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
       AND NOT rolbypassrls
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_auth_members m
    JOIN pg_catalog.pg_roles r ON r.oid=m.member
     WHERE r.rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_auth_members m
    JOIN pg_catalog.pg_roles grupo ON grupo.oid=m.roleid
    JOIN pg_catalog.pg_roles miembro ON miembro.oid=m.member
     WHERE grupo.rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
       AND (
         v_version=14 OR m.admin_option OR NOT m.inherit_option
         OR m.set_option OR NOT miembro.rolcanlogin OR miembro.rolsuper
         OR miembro.rolcreatedb OR miembro.rolcreaterole
         OR miembro.rolreplication OR miembro.rolbypassrls
       )
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='rol lector O4-05 existente no es mínimo';
  END IF;
END
$delta$;
COMMIT;
