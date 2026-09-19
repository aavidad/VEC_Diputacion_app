\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000108', 0
    )
);

LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
    IN ACCESS EXCLUSIVE MODE;

DO $proteccion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia impide retirar auditoria de frontera';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)
    FROM vec_contratacion_temporal_registrador_frontera;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_registrador_frontera;
DROP FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text);
DROP TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta;
DROP FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_ruta_exacta_v1();
COMMIT;
