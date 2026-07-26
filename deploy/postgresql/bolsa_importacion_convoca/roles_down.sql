-- Reversión DBA. Exige retirar antes todos los LOGIN externos.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_importacion_convoca:roles:v1', 0)
);

DO $prevalidacion$
DECLARE
    miembros text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversion de roles Convoca requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_importacion_convoca') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion rechazada: queda el esquema de importacion Convoca';
    END IF;
    SELECT array_agg(miembro.rolname::text ORDER BY miembro.rolname)
      INTO miembros
      FROM pg_catalog.pg_auth_members AS membresia
      JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
      JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
     WHERE grupo.rolname IN (
         'vec_bolsa_importacion_convoca_ejecutor',
         'vec_bolsa_importacion_convoca_recuperador',
         'vec_bolsa_importacion_convoca_conciliador',
         'vec_bolsa_importacion_convoca_retencion',
         'vec_bolsa_importacion_convoca_gobernanza'
     );
    IF cardinality(miembros) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion rechazada: quedan identidades runtime',
            DETAIL = array_to_string(miembros, ',');
    END IF;
END
$prevalidacion$;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL ON DATABASE %I FROM vec_bolsa_importacion_convoca_gobernanza, vec_bolsa_importacion_convoca_retencion, vec_bolsa_importacion_convoca_conciliador, vec_bolsa_importacion_convoca_recuperador, vec_bolsa_importacion_convoca_ejecutor, vec_bolsa_importacion_convoca_migrador, vec_bolsa_importacion_convoca_propietario',
        current_database()
    );
END
$privilegios_base$;

REVOKE vec_bolsa_importacion_convoca_propietario
    FROM vec_bolsa_importacion_convoca_migrador;
DROP ROLE vec_bolsa_importacion_convoca_gobernanza;
DROP ROLE vec_bolsa_importacion_convoca_retencion;
DROP ROLE vec_bolsa_importacion_convoca_conciliador;
DROP ROLE vec_bolsa_importacion_convoca_recuperador;
DROP ROLE vec_bolsa_importacion_convoca_ejecutor;
DROP ROLE vec_bolsa_importacion_convoca_migrador;
DROP ROLE vec_bolsa_importacion_convoca_propietario;

COMMIT;
