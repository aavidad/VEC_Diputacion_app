\set ON_ERROR_STOP on
-- Solo ensayo PostgreSQL desechable SIN historia. Jamás ejecutar en principal.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000006:documento:v1',0));
DO $pre$
BEGIN
 IF session_user<>current_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regclass('vec_dietas.numero_documento_comision') IS NULL
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR to_regclass('vec_dietas.recibo_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.recibo_regla_creacion_comision') IS NULL
    OR to_regclass('vec_dietas.historia_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.outbox_comision') IS NULL
    OR to_regclass('vec_dietas.cola_circuito_comision') IS NOT NULL
 THEN RAISE EXCEPTION 'Dietas 000006 DOWN: preimagen o DBA inválidos' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_dietas.borrador_comision,vec_dietas.numero_documento_comision,
 vec_dietas.comision_revision,vec_dietas.recibo_operacion_comision,
 vec_dietas.recibo_regla_creacion_comision,
 vec_dietas.historia_operacion_comision,vec_dietas.outbox_comision,
 vec_dietas.regla_devengo_provisional IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_dietas.borrador_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.numero_documento_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.comision_revision)
    OR EXISTS(SELECT 1 FROM vec_dietas.recibo_operacion_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.recibo_regla_creacion_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.historia_operacion_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.outbox_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.auditoria_frontera_comision)
    OR (SELECT count(*) FROM vec_dietas.regla_devengo_provisional)<>1
    OR NOT EXISTS(SELECT 1 FROM vec_dietas.regla_devengo_provisional
       WHERE regla_ref='provisional:regla:nacional-ordinaria:20260923') THEN
  RAISE EXCEPTION 'Dietas 000006 DOWN: historia impide retirada' USING ERRCODE='55000'; END IF;
END $historia$;
SET LOCAL ROLE vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)
 FROM vec_dietas_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_dietas_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)
 FROM vec_dietas_registrador_frontera;
DROP FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text) RESTRICT;
DROP FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.validar_calculo_catalogado_v2(jsonb) RESTRICT;
DROP FUNCTION vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.autorizar_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.proyectar_revision_exacta_v2(text,bigint) RESTRICT;
DROP FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) RESTRICT;
DROP FUNCTION vec_dietas.validar_documento_v2(jsonb) RESTRICT;
DROP FUNCTION vec_dietas.validar_rutas_d4_v2(jsonb,numeric,text,text) RESTRICT;
DROP FUNCTION vec_dietas.intencion_documento_v2(jsonb) RESTRICT;
DROP FUNCTION vec_dietas.huella_semantica_mutacion_v2(text) RESTRICT;
DROP FUNCTION vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text) RESTRICT;
DROP TRIGGER numerar_comision ON vec_dietas.borrador_comision;
DROP FUNCTION vec_dietas.numerar_comision_v1() RESTRICT;
DROP TABLE vec_dietas.historia_operacion_comision RESTRICT;
DROP TABLE vec_dietas.outbox_comision RESTRICT;
DROP TABLE vec_dietas.recibo_operacion_comision RESTRICT;
DROP TABLE vec_dietas.recibo_regla_creacion_comision RESTRICT;
DROP TABLE vec_dietas.comision_revision RESTRICT;
DROP TABLE vec_dietas.regla_devengo_provisional RESTRICT;
DROP TABLE vec_dietas.numero_documento_comision RESTRICT;
DROP SEQUENCE vec_dietas.numero_documento_comision_seq RESTRICT;
COMMIT;
