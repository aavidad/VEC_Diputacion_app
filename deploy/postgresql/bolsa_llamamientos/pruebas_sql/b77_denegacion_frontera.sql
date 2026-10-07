\set ON_ERROR_STOP on
-- El registrador B61 real escribe solo la ruta documental exacta. Identidad
-- LOGIN sintética y hechos quedan revertidos; no hay COSE ni datos personales.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $pre$
BEGIN
 IF pg_catalog.to_regrole('vec_b77_registro_prueba') IS NOT NULL THEN
  RAISE EXCEPTION 'PARO clave=rol_prueba_B77, actual=presente, esperado=ausente'; END IF;
END $pre$;
CREATE ROLE vec_b77_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_llamamientos_registrador_portal_externo TO vec_b77_registro_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_b77_registro_prueba;
DO $registro$
BEGIN
 IF NOT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_77777777777777777777777777777777','autenticacion_requerida',
   'api.bolsa.candidato.ruta_exacta','/api/vec/bolsa/mi-bolsa/solicitudes-documentales',NULL)
    OR NOT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_88888888888888888888888888888888','acceso_denegado',
   'api.bolsa.candidato.ruta_exacta','/api/vec/bolsa/mi-bolsa/solicitudes-documentales',
   'per_AAAAAAAAAAAAAAAAAAAAAA') THEN
  RAISE EXCEPTION 'B77 denegación 401/403 sin recibo'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_99999999999999999999999999999999','acceso_denegado',
   'api.bolsa.candidato.ruta_exacta','/api/vec/bolsa/mi-bolsa/solicitudes-documentales/ajena',
   'per_AAAAAAAAAAAAAAAAAAAAAA');
  RAISE EXCEPTION 'B77 ruta libre aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $registro$;
RESET SESSION AUTHORIZATION;
DO $resultado$
BEGIN
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo
     WHERE ruta='/api/vec/bolsa/mi-bolsa/solicitudes-documentales'
       AND correlacion_ref IN ('corr_77777777777777777777777777777777',
                              'corr_88888888888888888888888888888888'))<>2
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo
     WHERE correlacion_ref='corr_99999999999999999999999999999999') THEN
  RAISE EXCEPTION 'B77 historia frontera 401/403 distinta'; END IF;
END $resultado$;
ROLLBACK;
