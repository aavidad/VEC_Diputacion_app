BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) FROM PUBLIC, vec_bolsa_llamamientos_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_bolsa_llamamientos_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
    jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
);

COMMIT;
