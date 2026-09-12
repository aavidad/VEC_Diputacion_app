\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_administracion_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_administracion:configuracion_correo:000001', 0));
-- No se permite revertir una configuración que pueda contener un secreto.
DO $bloquear$
BEGIN
    LOCK TABLE vec_administracion.configuracion_correo,
               vec_administracion.sobre_configuracion_correo,
               vec_administracion.auditoria_configuracion_correo,
               vec_administracion.outbox_configuracion_correo IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM vec_administracion.configuracion_correo)
       OR EXISTS (SELECT 1 FROM vec_administracion.sobre_configuracion_correo)
       OR EXISTS (SELECT 1 FROM vec_administracion.auditoria_configuracion_correo)
       OR EXISTS (SELECT 1 FROM vec_administracion.outbox_configuracion_correo) THEN
        RAISE EXCEPTION 'no ejecutar DOWN de configuracion de correo con historia' USING ERRCODE = '55000';
    END IF;
END
$bloquear$;
DROP FUNCTION vec_administracion.sobre_configuracion_correo_actual_v1();
DROP FUNCTION vec_administracion.leer_configuracion_correo_v1();
DROP FUNCTION vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TRIGGER outbox_configuracion_correo_inmutable ON vec_administracion.outbox_configuracion_correo;
DROP FUNCTION vec_administracion.rechazar_mutacion_outbox_configuracion_correo_v1();
DROP TABLE vec_administracion.outbox_configuracion_correo;
DROP TABLE vec_administracion.auditoria_configuracion_correo;
DROP TABLE vec_administracion.sobre_configuracion_correo;
DROP TABLE vec_administracion.configuracion_correo;
DROP SCHEMA vec_administracion RESTRICT;
COMMIT;
