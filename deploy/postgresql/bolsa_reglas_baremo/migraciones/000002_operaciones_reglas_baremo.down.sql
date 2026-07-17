BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000002', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'operaciones de reglas de baremo no instaladas';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        jsonb, jsonb, bytea, bytea, bytea
    ) FROM PUBLIC,
      vec_bolsa_reglas_baremo_ejecutor_gobierno,
      vec_bolsa_reglas_baremo_ejecutor_consulta,
      vec_bolsa_reglas_baremo_publicador_outbox;
REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
        jsonb, jsonb, bytea, bytea
    ) FROM PUBLIC,
      vec_bolsa_reglas_baremo_ejecutor_gobierno,
      vec_bolsa_reglas_baremo_ejecutor_consulta,
      vec_bolsa_reglas_baremo_publicador_outbox;
DROP FUNCTION vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
    jsonb, jsonb, bytea, bytea
);
DROP FUNCTION vec_bolsa_reglas_baremo.confirmar_cambio_v1(
    jsonb, jsonb, bytea, bytea, bytea
);
COMMIT;
