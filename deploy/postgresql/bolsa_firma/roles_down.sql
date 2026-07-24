-- Desmontaje DBA explícito. Las migraciones down deben haberse aplicado antes.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_firma:roles_down:v1', 0)
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'desmontaje de firma de Bolsa requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_firma') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'aplique primero las migraciones down de firma';
    END IF;
END
$prevalidacion$;

DROP EVENT TRIGGER IF EXISTS vec_bolsa_firma_cerrar_acl_tipos;
DROP SCHEMA IF EXISTS vec_bolsa_firma_guardia CASCADE;

DO $desconectar$
DECLARE rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_firma_ejecutor',
        'vec_bolsa_firma_migrador',
        'vec_bolsa_firma_propietario'
    ] LOOP
        IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = rol) THEN
            EXECUTE format(
                'REVOKE CONNECT, CREATE ON DATABASE %I FROM %I',
                current_database(), rol
            );
        END IF;
    END LOOP;
END
$desconectar$;

REVOKE vec_bolsa_firma_propietario FROM vec_bolsa_firma_migrador;
DROP ROLE vec_bolsa_firma_ejecutor;
DROP ROLE vec_bolsa_firma_migrador;
DROP ROLE vec_bolsa_firma_propietario;
COMMIT;
