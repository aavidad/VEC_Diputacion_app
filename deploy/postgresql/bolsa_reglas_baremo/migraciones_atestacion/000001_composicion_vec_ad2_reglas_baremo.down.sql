BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:composicion-vec-ad2:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_reglas_baremo'
           AND clase.relname = 'recibo_cambio_atestado_v2'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'retire primero la migracion V2 de reglas de baremo';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) FROM vec_bolsa_reglas_baremo_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v2
    FROM vec_bolsa_reglas_baremo_propietario;
REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(
        text, text, text, text, text, text, text
    ) FROM vec_bolsa_reglas_baremo_propietario;
REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    ) FROM vec_bolsa_reglas_baremo_propietario;

-- Restaura exactamente el contrato central previo al retirar la composicion.
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) TO vec_autorizacion_atestada_v2_consumidor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    ) TO vec_autorizacion_atestada_v2_consumidor;

REVOKE REFERENCES (
    auditoria_ref, consumo_ref, registro_ref, decision_ref, efecto_ref,
    huella_registro_sha256
) ON vec_autorizacion_atestada_v2.auditoria_consumo_v2
  FROM vec_bolsa_reglas_baremo_propietario;

DROP FUNCTION
    vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(
        text, text, text, text, text, text, text
    );
REVOKE REFERENCES (
    consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
    efecto_ref, huella_efecto_sha256
) ON vec_autorizacion_atestada_v2.consumo_decision_v2
  FROM vec_bolsa_reglas_baremo_propietario;
REVOKE REFERENCES (
    registro_ref, decision_ref, huella_decision_sha256
) ON vec_autorizacion_atestada_v2.atestacion_decision_v2
  FROM vec_bolsa_reglas_baremo_propietario;

ALTER TABLE vec_autorizacion_atestada_v2.auditoria_consumo_v2
    DROP CONSTRAINT auditoria_consumo_v2_vinculo_reglas_unico;
ALTER TABLE vec_autorizacion_atestada_v2.consumo_decision_v2
    DROP CONSTRAINT consumo_decision_v2_vinculo_reglas_unico;

COMMIT;
