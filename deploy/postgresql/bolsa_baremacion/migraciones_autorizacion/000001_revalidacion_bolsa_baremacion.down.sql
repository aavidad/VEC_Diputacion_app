BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_baremacion:migracion_down:v2',
        0
    )
);

DO $confirmacion_reversion$
DECLARE
    confirmacion_constante constant text :=
        'REVERTIR_MIGRACION_BOLSA_BAREMACION_V1';
BEGIN
    IF current_setting(
        'vec.confirmar_reversion_bolsa_baremacion',
        true
    ) IS DISTINCT FROM confirmacion_constante THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down autorizacion Bolsa rechazado: falta confirmacion explicita',
            DETAIL = 'la reversion esta denegada por defecto, incluso sin consumidor',
            HINT = 'configure el opt-in literal solo en la sesion aprobada de migracion';
    END IF;
END
$confirmacion_reversion$;

-- Esta frontera solo se retira despues del esquema consumidor. Exigir su
-- ausencia es mas fuerte que intentar leer tablas ajenas desde el propietario
-- de autorizacion y evita desmontar parcialmente una Bolsa vacia o con historia.
DO $prevalidacion_consumidor$
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_baremacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down autorizacion Bolsa rechazado: el esquema consumidor sigue instalado';
    END IF;
END
$prevalidacion_consumidor$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) FROM vec_bolsa_baremacion_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_bolsa_baremacion_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
    jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
) RESTRICT;
COMMIT;
