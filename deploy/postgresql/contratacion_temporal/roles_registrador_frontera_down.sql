-- Ejecutar solo después del DOWN 000108 y de retirar los LOGIN miembros.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_registrador_frontera:v1', 0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down del registrador de frontera fuera de orden';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_frontera'
    ) THEN
        RETURN;
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
        JOIN pg_catalog.pg_roles rol ON rol.oid = membresia.roleid OR rol.oid = membresia.member
        WHERE rol.rolname = 'vec_contratacion_temporal_registrador_frontera'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de frontera conserva membresías';
    END IF;
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_registrador_frontera',
        pg_catalog.current_database()
    );
    EXECUTE 'DROP ROLE vec_contratacion_temporal_registrador_frontera';
END
$delta$;

COMMIT;
