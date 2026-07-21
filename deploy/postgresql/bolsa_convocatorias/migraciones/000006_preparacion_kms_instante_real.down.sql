BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir preparacion KMS v2';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) FROM vec_bolsa_convocatorias_proyector_gobierno;
DROP FUNCTION vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
    jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
    bytea
);
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) TO vec_bolsa_convocatorias_proyector_gobierno;

COMMIT;
