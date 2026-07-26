-- Ejecutar después del down 000035 y de retirar todos sus LOGIN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:rol_lector_resultado_cobertura:v1',0));
DO $delta$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=current_user AND rolsuper
  ) OR NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .control_migracion_cobertura_o4
     WHERE control AND version_esquema=14
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='down de lector O4-05 fuera de orden';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
  ) THEN
    RETURN;
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
    JOIN pg_catalog.pg_roles r ON r.oid=m.roleid OR r.oid=m.member
     WHERE r.rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='rol lector O4-05 conserva membresías o atributos';
  END IF;
  IF NOT pg_catalog.has_database_privilege(
       'vec_contratacion_temporal_lector_resultado_cobertura',
       pg_catalog.current_database(),'CONNECT'
     ) OR pg_catalog.has_database_privilege(
       'vec_contratacion_temporal_lector_resultado_cobertura',
       pg_catalog.current_database(),'CREATE'
     ) OR pg_catalog.has_database_privilege(
       'vec_contratacion_temporal_lector_resultado_cobertura',
       pg_catalog.current_database(),'TEMP'
     ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='ACL de base del lector O4-05 no es mínima';
  END IF;
  EXECUTE pg_catalog.format(
    'REVOKE CONNECT ON DATABASE %I FROM '
    ||'vec_contratacion_temporal_lector_resultado_cobertura',
    pg_catalog.current_database()
  );
  EXECUTE
    'DROP ROLE vec_contratacion_temporal_lector_resultado_cobertura';
END
$delta$;
COMMIT;
