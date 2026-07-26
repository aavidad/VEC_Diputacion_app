-- Cierre DBA obligatorio inmediatamente después de la migración del esquema.
-- El privilegio CREATE solo existe durante el bootstrap y no persiste en
-- runtime. Separarlo es necesario porque su grantor es el DBA, no el migrador.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_registro_accesos:cerrar_acl:v1', 0)
);
DO $cerrar$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace(
        'vec_bolsa_registro_accesos'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'cierre ACL T13 requiere DBA y esquema instalado';
    END IF;
    EXECUTE pg_catalog.format(
        'REVOKE CREATE ON DATABASE %I FROM vec_bolsa_accesos_propietario',
        pg_catalog.current_database()
    );
END
$cerrar$;
COMMIT;
