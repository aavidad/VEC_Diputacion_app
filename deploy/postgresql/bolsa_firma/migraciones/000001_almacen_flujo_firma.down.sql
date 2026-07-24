BEGIN;
SET LOCAL ROLE vec_bolsa_firma_propietario;
SET LOCAL search_path = pg_catalog;

DO $confirmacion$
BEGIN
    IF current_setting(
        'vec.confirmar_reversion_bolsa_firma', true
    ) IS DISTINCT FROM 'REVERTIR_ALMACEN_FIRMA_BOLSA' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversión destructiva de firma no confirmada';
    END IF;
    IF to_regprocedure(
        'vec_bolsa_firma.crear_o_recuperar_flujo_v1(jsonb,bytea)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'revierta primero las operaciones de firma';
    END IF;
END
$confirmacion$;

-- Restaura el valor predeterminado nativo de PostgreSQL para no dejar una
-- ACL global propiedad del rol técnico después de eliminar el esquema.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_firma_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
DROP SCHEMA vec_bolsa_firma CASCADE;
COMMIT;
