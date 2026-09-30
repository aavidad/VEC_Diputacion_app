\set ON_ERROR_STOP on
-- Documentos-9 DOWN: retira la custodia de documentos firmados. Se niega si
-- hay cualquier documento firmado custodiado o su auditoría/outbox: esa
-- historia no se borra. Después puede retirarse AD3-113.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000009',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regclass('vec_documentos.documento_firmado') IS NULL
    OR to_regprocedure('vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Documentos-9 DOWN: 000009 no instalada' USING ERRCODE='55000'; END IF;
END $pre$;
-- El propietario está sujeto a RLS forzada: se retira dentro de esta
-- transacción solo para contar sin filtro y se restaura antes de seguir.
ALTER TABLE vec_documentos.documento_firmado NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.auditoria_operacion NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.outbox NO FORCE ROW LEVEL SECURITY;
DO $vacia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_documentos.documento_firmado)
    OR EXISTS (SELECT 1 FROM vec_documentos.auditoria_operacion WHERE accion='documentos.firmado.custodiar')
    OR EXISTS (SELECT 1 FROM vec_documentos.outbox WHERE tipo='documento_firmado_custodiado')
 THEN RAISE EXCEPTION 'Documentos-9 DOWN: hay documentos firmados custodiados' USING ERRCODE='55000'; END IF;
END $vacia$;
ALTER TABLE vec_documentos.documento_firmado FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.auditoria_operacion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.outbox FORCE ROW LEVEL SECURITY;
DROP FUNCTION vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TRIGGER exigir_documento_firmado ON vec_documentos.documento;
DROP TRIGGER rechazar_tipo_reservado ON vec_documentos.referencia_externa;
DROP FUNCTION vec_documentos.rechazar_externa_reservada_v1();
DROP FUNCTION vec_documentos.exigir_documento_firmado_v1();
DROP TABLE vec_documentos.documento_firmado;
DROP TABLE vec_documentos.tipo_reservado_firmado;
ALTER TABLE vec_documentos.auditoria_operacion DROP CONSTRAINT auditoria_operacion_accion_check;
ALTER TABLE vec_documentos.auditoria_operacion ADD CONSTRAINT auditoria_operacion_accion_check
 CHECK(accion IN ('documentos.generado.alta','documentos.expediente.listar','documentos.original.descargar','documentos.notificacion.preparar','documentos.externo.registrar'));
ALTER TABLE vec_documentos.outbox DROP CONSTRAINT outbox_tipo_check;
ALTER TABLE vec_documentos.outbox ADD CONSTRAINT outbox_tipo_check
 CHECK(tipo IN ('documento_generado','notificacion_preparada','documento_externo_registrado'));
COMMIT;
