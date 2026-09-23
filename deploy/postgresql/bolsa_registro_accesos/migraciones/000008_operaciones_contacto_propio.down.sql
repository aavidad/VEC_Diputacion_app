\set ON_ERROR_STOP on
-- Sólo base desechable sin historia de operaciones F2. Nunca sobre recibos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR f IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_bolsa_accesos_propietario'::regrole AND prosecdef)
    OR to_regclass('vec_contacto_usuario_v1.operaciones') IS NOT NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
        WHERE action IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                         'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')) THEN
    RAISE EXCEPTION 'T13/8: DOWN requiere Contacto3 retirado y cero historia' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
REVOKE EXECUTE ON FUNCTION vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text) FROM vec_contacto_usuario_owner;
DROP FUNCTION vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text);
COMMIT;
