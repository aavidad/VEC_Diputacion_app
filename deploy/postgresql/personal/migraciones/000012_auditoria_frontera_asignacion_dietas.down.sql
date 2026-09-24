\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SET LOCAL row_security = off;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_personal:migracion:000012:auditoria_frontera_asignacion_dietas', 0)
);

DO $precondicion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de auditoria Personal requiere DBA';
    END IF;
END
$precondicion$;

LOCK TABLE vec_personal.auditoria_frontera_asignacion_dietas IN ACCESS EXCLUSIVE MODE;

DO $proteccion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_personal.auditoria_frontera_asignacion_dietas) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia impide retirar auditoria de frontera Personal';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
         WHERE membresia.roleid =
             'vec_personal_registrador_frontera'::pg_catalog.regrole
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'miembros impiden retirar registrador de frontera Personal';
    END IF;
END
$proteccion$;

SET LOCAL ROLE vec_personal_propietario;
REVOKE EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)
    FROM vec_personal_registrador_frontera;
REVOKE USAGE ON SCHEMA vec_personal FROM vec_personal_registrador_frontera;
DROP FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint);
DROP TABLE vec_personal.auditoria_frontera_asignacion_dietas;
DROP FUNCTION vec_personal.rechazar_mutacion_auditoria_frontera_asignacion_dietas_v1();

RESET ROLE;
DO $retirar_rol$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_personal_registrador_frontera',
        pg_catalog.current_database()
    );
END
$retirar_rol$;
DROP ROLE vec_personal_registrador_frontera;
COMMIT;
