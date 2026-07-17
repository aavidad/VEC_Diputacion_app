BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_panel.consultar_panel_interno_v1(jsonb,jsonb,bytea,bytea,text)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la consulta del panel';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_bolsa_panel.consulta_confirmada)
       OR EXISTS (SELECT 1 FROM vec_bolsa_panel.auditoria) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen lecturas confirmadas';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_bolsa_panel.consultar_panel_interno_v1(
    jsonb, jsonb, bytea, bytea, text
);
COMMIT;
