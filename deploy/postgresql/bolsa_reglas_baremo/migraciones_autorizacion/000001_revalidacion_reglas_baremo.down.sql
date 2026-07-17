BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:revalidacion_reglas_baremo:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_reglas_baremo_v1(jsonb,bytea,bytea,text,text,text,text,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'frontera V2 de reglas de baremo no instalada';
    END IF;
    IF to_regnamespace('vec_bolsa_reglas_baremo') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire antes el almacen de reglas de baremo';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_reglas_baremo_v1(
        jsonb, bytea, bytea, text, text, text, text, timestamptz
    ) FROM vec_bolsa_reglas_baremo_propietario;
REVOKE REFERENCES (decision_ref) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM vec_bolsa_reglas_baremo_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_bolsa_reglas_baremo_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_reglas_baremo_v1(
    jsonb, bytea, bytea, text, text, text, text, timestamptz
);
COMMIT;
