BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(jsonb,bytea,bytea,text,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta autorizacion nominal de borradores';
    END IF;
    IF to_regclass('vec_bolsa_convocatorias.diario_borrador_version')
       IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el vertical de borradores sigue instalado';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
        jsonb, bytea, bytea, text, text, text, text, jsonb, timestamptz
    ) FROM vec_bolsa_convocatorias_propietario;
DROP FUNCTION
    vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
        jsonb, bytea, bytea, text, text, text, text, jsonb, timestamptz
    );
COMMIT;
