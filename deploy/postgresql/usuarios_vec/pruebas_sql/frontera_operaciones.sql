\set ON_ERROR_STOP on
DO $test$ BEGIN
 IF vec_usuarios.registrar_denegacion_preferencias_v1(
    'corr_no_disponible','autenticacion_requerida',
    'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',NULL) IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios 000003: 401 sin recibo'; END IF;
 IF vec_usuarios.registrar_denegacion_preferencias_v1(
    'corr_0123456789abcdef0123456789abcdef','acceso_denegado',
    'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',
    'per_abcdefghijklmnopqrstuv') IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios 000003: 403 sin recibo'; END IF;
 -- Cada rechazo revierte su subtransacción y preserva las dos filas anteriores.
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','autenticacion_requerida',
   'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',
   'per_abcdefghijklmnopqrstuv');
  RAISE EXCEPTION 'Usuarios 000003: actor en 401 aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','acceso_denegado',
   'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/otra',NULL);
  RAISE EXCEPTION 'Usuarios 000003: ruta ajena aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','acceso_denegado','api.contratacion.ruta_exacta',
   '/api/vec/usuarios/mis-preferencias',NULL);
  RAISE EXCEPTION 'Usuarios 000003: superficie ajena aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','dependencia',
   'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',NULL);
  RAISE EXCEPTION 'Usuarios 000003: motivo ajeno aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','acceso_denegado',
   'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',
   '12345678Z');
  RAISE EXCEPTION 'Usuarios 000003: identidad no opaca aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $test$;
BEGIN;
SELECT vec_usuarios.registrar_denegacion_preferencias_v1(
 'corr_no_disponible','acceso_denegado',
 'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',NULL);
ROLLBACK;
