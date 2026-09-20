-- Retirada DBA: las migraciones deben haberse retirado y no puede haber historia.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_gateway_personal:roles_down:v1', 0)
);

DO $prevalidacion$
DECLARE
    propietario oid;
    migrador oid;
    ejecutor oid;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada gateway Personal rechazada: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_gateway_personal') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada gateway Personal rechazada: esquema aun instalado';
    END IF;
    SELECT oid INTO propietario FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_gateway_personal_propietario';
    SELECT oid INTO migrador FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_gateway_personal_migrador';
    SELECT oid INTO ejecutor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_gateway_personal_ejecutor';
    IF propietario IS NULL OR migrador IS NULL OR ejecutor IS NULL
       OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_auth_members
           WHERE (roleid IN (propietario, migrador, ejecutor)
                  OR member IN (propietario, migrador, ejecutor))
             AND NOT (roleid = propietario AND member = migrador
                      AND NOT admin_option AND NOT inherit_option AND set_option)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada gateway Personal rechazada: roles o membresias incompatibles';
    END IF;
END
$prevalidacion$;

REVOKE vec_gateway_personal_propietario FROM vec_gateway_personal_migrador;
DO $privilegios_base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_gateway_personal_propietario, vec_gateway_personal_migrador, vec_gateway_personal_ejecutor',
        current_database()
    );
END
$privilegios_base$;
DROP ROLE vec_gateway_personal_ejecutor;
DROP ROLE vec_gateway_personal_migrador;
DROP ROLE vec_gateway_personal_propietario;
COMMIT;
