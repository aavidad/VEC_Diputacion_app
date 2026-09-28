-- Ejecutar solo después del DOWN 000136 y de retirar los LOGIN miembros.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_registrador_auditoria:v1', 0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down del registrador de auditoria fuera de orden';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_auditoria'
    ) THEN
        RETURN;
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles rol
            ON rol.oid = membresia.roleid OR rol.oid = membresia.member
         WHERE rol.rolname = 'vec_contratacion_temporal_registrador_auditoria'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de auditoria conserva membresias';
    END IF;
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_registrador_auditoria',
        pg_catalog.current_database()
    );
    EXECUTE 'DROP ROLE vec_contratacion_temporal_registrador_auditoria';
END
$delta$;

COMMIT;
