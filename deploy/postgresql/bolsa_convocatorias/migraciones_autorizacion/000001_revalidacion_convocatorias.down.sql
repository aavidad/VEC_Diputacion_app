BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la frontera esperada';
    END IF;
    IF to_regnamespace('vec_bolsa_convocatorias') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el consumidor sigue instalado';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) FROM vec_bolsa_convocatorias_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_bolsa_convocatorias_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(
    jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
);
COMMIT;
