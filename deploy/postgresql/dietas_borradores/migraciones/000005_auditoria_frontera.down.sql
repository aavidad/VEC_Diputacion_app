\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_dietas:migracion:000005:auditoria_frontera', 0)
);

DO $precondicion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de auditoria Dietas requiere DBA';
    END IF;
END
$precondicion$;

SET LOCAL ROLE vec_dietas_propietario;
LOCK TABLE vec_dietas.auditoria_frontera_comision IN ACCESS EXCLUSIVE MODE;

DO $proteccion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_dietas.auditoria_frontera_comision) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia impide retirar auditoria de frontera Dietas';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
         WHERE membresia.roleid =
             'vec_dietas_registrador_frontera'::pg_catalog.regrole
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'miembros impiden retirar registrador de frontera Dietas';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)
    FROM vec_dietas_registrador_frontera;
REVOKE USAGE ON SCHEMA vec_dietas FROM vec_dietas_registrador_frontera;
DROP FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text);
DROP TABLE vec_dietas.auditoria_frontera_comision;
DROP FUNCTION vec_dietas.rechazar_mutacion_auditoria_frontera_comision_v1();

RESET ROLE;
DO $retirar_rol$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_dietas_registrador_frontera',
        pg_catalog.current_database()
    );
END
$retirar_rol$;
DROP ROLE vec_dietas_registrador_frontera;
COMMIT;
