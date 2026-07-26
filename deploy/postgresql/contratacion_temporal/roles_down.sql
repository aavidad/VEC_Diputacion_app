BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:roles_up:v1',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'retirada de roles rechazada: requiere superusuario';
    END IF;
    IF to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de roles rechazada: el esquema sigue instalado';
    END IF;
END
$prevalidacion$;

DO $privilegios$
BEGIN
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_migrador',
        current_database()
    );
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_ejecutor',
        current_database()
    );
    IF EXISTS(
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
    ) THEN
        EXECUTE format(
            'REVOKE CONNECT ON DATABASE %I FROM '
            || 'vec_contratacion_temporal_confirmador_cobertura',
            current_database()
        );
    END IF;
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_gobernador',
        current_database()
    );
END
$privilegios$;

REVOKE vec_contratacion_temporal_propietario
    FROM vec_contratacion_temporal_migrador;
DROP ROLE vec_contratacion_temporal_gobernador;
DROP ROLE IF EXISTS vec_contratacion_temporal_confirmador_cobertura;
DROP ROLE vec_contratacion_temporal_ejecutor;
DROP ROLE vec_contratacion_temporal_migrador;
DROP ROLE vec_contratacion_temporal_propietario;

COMMIT;
