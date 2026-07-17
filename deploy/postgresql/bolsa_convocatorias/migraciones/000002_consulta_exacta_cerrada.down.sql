BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la consulta exacta';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_convocatorias.uso_decision_consulta
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_convocatorias.auditoria
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen consultas consumidas';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v1(
    jsonb, jsonb, bytea, bytea
);
COMMIT;
