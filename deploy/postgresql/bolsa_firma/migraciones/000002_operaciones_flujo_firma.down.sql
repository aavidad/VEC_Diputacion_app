BEGIN;
SET LOCAL ROLE vec_bolsa_firma_propietario;
SET LOCAL search_path = pg_catalog;

DO $confirmacion$
BEGIN
    IF current_setting(
        'vec.confirmar_reversion_bolsa_firma', true
    ) IS DISTINCT FROM 'REVERTIR_OPERACIONES_FIRMA_BOLSA' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversión de operaciones de firma no confirmada';
    END IF;
END
$confirmacion$;

REVOKE USAGE ON SCHEMA vec_bolsa_firma FROM vec_bolsa_firma_ejecutor;
DROP FUNCTION vec_bolsa_firma.liberar_arrendamiento_v1(jsonb, bytea);
DROP FUNCTION vec_bolsa_firma.guardar_flujo_v1(
    jsonb, jsonb, bytea, bytea
);
DROP FUNCTION vec_bolsa_firma.adquirir_arrendamiento_v1(jsonb, bytea);
DROP FUNCTION vec_bolsa_firma.obtener_flujo_v1(text, text, text);
DROP FUNCTION vec_bolsa_firma.crear_o_recuperar_flujo_v1(jsonb, bytea);
DROP FUNCTION vec_bolsa_firma.transicion_valida(jsonb, jsonb);
DROP FUNCTION vec_bolsa_firma.expediente_valido(jsonb, bytea);
COMMIT;
