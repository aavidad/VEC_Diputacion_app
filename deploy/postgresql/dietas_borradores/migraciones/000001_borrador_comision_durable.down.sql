\set ON_ERROR_STOP on
BEGIN;
-- El canal de migración inspecciona la historia completa incluso con FORCE RLS.
-- Debe iniciarse desde una sesión nueva con autoridad de instalación; si no puede
-- desactivar RLS, falla antes de retirar ningún objeto.
SET LOCAL row_security=off;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000001:borrador:v1',0));
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
   RAISE EXCEPTION 'Dietas 000001 DOWN requiere autoridad de migración con inspección RLS completa' USING ERRCODE='42501';
 END IF;
END $$;
-- Las cinco tablas permanecen bloqueadas hasta COMMIT/ROLLBACK; así no existe
-- una ventana entre la preimagen y los DROP aunque FORCE RLS esté activo.
LOCK TABLE vec_dietas.borrador_comision,vec_dietas.recibo_borrador_comision,vec_dietas.historia_borrador_comision,vec_dietas.auditoria_borrador_comision,vec_dietas.outbox_borrador_comision IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_dietas.borrador_comision) OR EXISTS(SELECT 1 FROM vec_dietas.recibo_borrador_comision) OR EXISTS(SELECT 1 FROM vec_dietas.historia_borrador_comision) OR EXISTS(SELECT 1 FROM vec_dietas.auditoria_borrador_comision) OR EXISTS(SELECT 1 FROM vec_dietas.outbox_borrador_comision) THEN RAISE EXCEPTION 'Dietas 000001 DOWN protege historia/recibos/auditoría/outbox' USING ERRCODE='55000'; END IF;
END $$;
SET LOCAL ROLE vec_dietas_migrador;
SET LOCAL ROLE vec_dietas_propietario;
DROP FUNCTION vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.consumir_ad3_borrador_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.huella_semantica_crear_borrador_v1(text) RESTRICT;
DROP FUNCTION vec_dietas.cadena_json_go_v1(text) RESTRICT;
DROP FUNCTION vec_dietas.cotejar_recurso_dietas_borrador_v1(text,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.cotejar_contexto_dietas_borrador_v1(jsonb,bytea) RESTRICT;
DROP TABLE vec_dietas.outbox_borrador_comision,vec_dietas.auditoria_borrador_comision,vec_dietas.historia_borrador_comision,vec_dietas.recibo_borrador_comision,vec_dietas.borrador_comision RESTRICT;
DROP FUNCTION vec_dietas.rechazar_mutacion_borrador_v1() RESTRICT;
COMMIT;
