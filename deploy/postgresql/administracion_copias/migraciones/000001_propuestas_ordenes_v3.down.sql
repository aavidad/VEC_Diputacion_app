\set ON_ERROR_STOP on
-- Sólo ensayo descartable y sin historia. Nunca ejecutarlo sobre instalada.
BEGIN;
SET LOCAL ROLE vec_administracion_copias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_administracion_copias:migracion:000001',0));
LOCK TABLE vec_administracion_copias.propuesta,vec_administracion_copias.revision,vec_administracion_copias.control_destino,
 vec_administracion_copias.orden,vec_administracion_copias.auditoria,vec_administracion_copias.outbox IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_administracion_copias.propuesta) OR EXISTS(SELECT 1 FROM vec_administracion_copias.revision)
    OR EXISTS(SELECT 1 FROM vec_administracion_copias.orden) OR EXISTS(SELECT 1 FROM vec_administracion_copias.auditoria)
    OR EXISTS(SELECT 1 FROM vec_administracion_copias.outbox) OR EXISTS(SELECT 1 FROM vec_administracion_copias.control_destino) THEN
  RAISE EXCEPTION 'copias_down_con_historia_denegado' USING ERRCODE='55000'; END IF;
END $historia$;
DROP FUNCTION vec_administracion_copias.leer_orden_comprometida_v1(text);
DROP FUNCTION vec_administracion_copias.comprometer_orden_v1(bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz);
DROP FUNCTION vec_administracion_copias.registrar_revision_v1(bytea,numeric,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_administracion_copias.registrar_propuesta_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_administracion_copias.registrar_control_v1(text,bytea,numeric,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_administracion_copias.plan_v1(bytea);
DROP TABLE vec_administracion_copias.auditoria,vec_administracion_copias.outbox,vec_administracion_copias.orden;
DROP TABLE vec_administracion_copias.revision;
DROP TABLE vec_administracion_copias.propuesta,vec_administracion_copias.control_destino;
DROP FUNCTION vec_administracion_copias.rechazar_mutacion();
DROP SCHEMA vec_administracion_copias;
COMMIT;
