BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.guardar_propuesta_v1(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC, vec_bolsa_llamamientos_ejecutor,
    vec_bolsa_llamamientos_proyector_autoritativo,
    vec_bolsa_llamamientos_registrador_atestacion,
    vec_bolsa_llamamientos_despachador_outbox;
DROP FUNCTION vec_bolsa_llamamientos.guardar_propuesta_v1(
    jsonb, jsonb, bytea, bytea
);
COMMIT;
