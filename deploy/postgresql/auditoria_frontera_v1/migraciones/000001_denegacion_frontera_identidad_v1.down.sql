\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_auditoria_frontera_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_auditoria_frontera_v1:migracion:000001', 0)
);
LOCK TABLE vec_auditoria_frontera_v1.denegacion_identidad IN ACCESS EXCLUSIVE MODE;
DO $proteccion$
BEGIN
    IF current_setting('vec.confirmar_retirada_denegacion_frontera_identidad_v1', true)
           IS DISTINCT FROM 'RETIRAR_DENEGACION_FRONTERA_IDENTIDAD_V1'
       OR EXISTS (SELECT 1 FROM vec_auditoria_frontera_v1.denegacion_identidad) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'historia impide retirar denegacion de frontera';
    END IF;
END
$proteccion$;
REVOKE EXECUTE ON FUNCTION vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text)
    FROM vec_auditoria_frontera_identidad_v1_registrador;
REVOKE USAGE ON SCHEMA vec_auditoria_frontera_v1 FROM vec_auditoria_frontera_identidad_v1_registrador;
DROP FUNCTION vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text);
DROP TABLE vec_auditoria_frontera_v1.denegacion_identidad;
DROP FUNCTION vec_auditoria_frontera_v1.rechazar_mutacion_denegacion_identidad_v1();
DROP SCHEMA vec_auditoria_frontera_v1;
COMMIT;
