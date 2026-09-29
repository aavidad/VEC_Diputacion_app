\set ON_ERROR_STOP on
-- El segundo LOGIN conserva TEMP: sus vistas no sustituyen catálogos.
CREATE TEMP VIEW pg_roles AS SELECT * FROM pg_catalog.pg_roles WHERE false;
CREATE TEMP VIEW pg_auth_members AS SELECT * FROM pg_catalog.pg_auth_members WHERE false;
DO $test$ BEGIN
 IF vec_usuarios.registrar_denegacion_preferencias_v1(
  'corr_no_disponible','autenticacion_requerida',
  'api.usuarios.preferencias.ruta_exacta',
  '/api/vec/usuarios/area-personal/mis-preferencias',NULL) IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios 000003: 401 exterior sin registro'; END IF;
 IF vec_usuarios.registrar_denegacion_preferencias_v1(
  'corr_0123456789abcdef0123456789abcdef','acceso_denegado',
  'api.usuarios.preferencias.ruta_exacta',
  '/api/vec/usuarios/area-personal/mis-preferencias',
  'per_abcdefghijklmnopqrstuv') IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios 000003: 403 exterior sin registro'; END IF;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','autenticacion_requerida',
   'api.usuarios.preferencias.ruta_exacta',
   '/api/vec/usuarios/mis-preferencias',NULL);
  RAISE EXCEPTION 'Usuarios 000003: ingreso exterior cruzo ruta interna';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','acceso_denegado',
   'api.usuarios.preferencias.ruta_exacta',
   '/api/vec/usuarios/area-personal/mis-preferencias',NULL);
  RAISE EXCEPTION 'Usuarios 000003: 403 exterior sin actor aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $test$;
