-- Retirar después de down 000028 y después de revocar sus LOGIN miembros.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:rol_confirmador_cobertura:v1',0
  )
);
DO $prevalidacion_control$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=current_user AND rolsuper
  ) THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='retirada del rol confirmador requiere superusuario';
  END IF;
  IF pg_catalog.to_regclass(
    'vec_contratacion_temporal.control_migracion_cobertura_o4'
  ) IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='control CT ausente al retirar el rol confirmador';
  END IF;
END
$prevalidacion_control$;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
 FOR UPDATE;
DO $delta$
BEGIN
  IF NOT EXISTS(
    SELECT 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=7
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='down 000028 debe preceder la retirada del rol confirmador';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
  ) THEN
    RETURN;
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
       AND NOT rolbypassrls
  ) OR EXISTS(
    SELECT 1
      FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
     WHERE r.rolname='vec_contratacion_temporal_confirmador_cobertura'
  ) OR EXISTS(
    SELECT 1
      FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles r ON r.oid=m.member
     WHERE r.rolname='vec_contratacion_temporal_confirmador_cobertura'
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='rol confirmador no mínimo o con membresías vigentes';
  END IF;
  EXECUTE pg_catalog.format(
    'REVOKE CONNECT ON DATABASE %I FROM '
    ||'vec_contratacion_temporal_confirmador_cobertura',
    pg_catalog.current_database()
  );
  EXECUTE 'DROP ROLE vec_contratacion_temporal_confirmador_cobertura';
END
$delta$;
COMMIT;
