\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000042',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.resultado_ejecucion_v1 IN ACCESS EXCLUSIVE MODE;
DO $conservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.resultado_ejecucion_v1) THEN
        RAISE EXCEPTION 'historia resultado conservada; DOWN rechazado' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid IN (
        'vec_resultado_rutas_dietas_registro'::regrole,
        'vec_resultado_borrador_dietas_registro'::regrole,
        'vec_resultado_marcaje_cronos_registro'::regrole)) THEN
        RAISE EXCEPTION 'runtime resultado dependiente; DOWN rechazado' USING ERRCODE='55000';
    END IF;
END
$conservar$;
-- RESTRICT implícito: nunca eliminar consumidores ni enlaces de historia.
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_rutas_dietas_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text);
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_borrador_dietas_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text);
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_marcaje_cronos_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text);
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_ejecucion_interno_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text);
DROP TABLE vec_autorizacion_atestada_v3.resultado_ejecucion_v1;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_resultado_rutas_dietas_registro,vec_resultado_borrador_dietas_registro,vec_resultado_marcaje_cronos_registro;
SET LOCAL ROLE vec_autorizacion_propietario;
DROP FUNCTION vec_autorizacion.concesion_resultado_ejecucion_v1(text,text,text,text,text,text,text,text,text,text,text);
RESET ROLE;
DO $conexion$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_resultado_rutas_dietas_registro,vec_resultado_borrador_dietas_registro,vec_resultado_marcaje_cronos_registro',current_database());
END
$conexion$;
DROP ROLE vec_resultado_rutas_dietas_registro;
DROP ROLE vec_resultado_borrador_dietas_registro;
DROP ROLE vec_resultado_marcaje_cronos_registro;
-- UP no modifica guardias ni audiencias AD3-041; DOWN tampoco las reconstruye.
COMMIT;
