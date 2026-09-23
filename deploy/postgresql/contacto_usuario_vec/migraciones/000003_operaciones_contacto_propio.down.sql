\set ON_ERROR_STOP on
-- Reversión solamente de un ensayo vacío. La operación y sus recibos son
-- historia: con una sola fila confirmada, cancelada o preparada se deniega.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_contacto_usuario_v1.operaciones') IS NULL
    OR to_regprocedure('vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.operaciones)
    OR EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.operacion_eventos)
    OR EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.versiones)
    OR EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.actual)
    OR EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.outbox)
    OR EXISTS(SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
        WHERE action IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                         'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')) THEN
    RAISE EXCEPTION 'Contacto3: DOWN sólo permite base vacía' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_contacto_usuario_owner;
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.preparar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM vec_contacto_usuario_writer;
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.cancelar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM vec_contacto_usuario_writer;
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.listar_operaciones_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea) FROM vec_contacto_usuario_writer;
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.detalle_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM vec_contacto_usuario_writer;
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM vec_contacto_usuario_writer;
DROP FUNCTION vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.detalle_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.listar_operaciones_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.cancelar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.preparar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea);
DROP POLICY bloqueo_actual_preparacion ON vec_contacto_usuario_v1.actual;
DROP POLICY lectura_actual_preparacion ON vec_contacto_usuario_v1.actual;
DROP TABLE vec_contacto_usuario_v1.operacion_eventos;
DROP TABLE vec_contacto_usuario_v1.operaciones;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
COMMIT;
