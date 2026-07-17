BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_panel_bolsa_v2(jsonb,bytea,bytea,text,text,text,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la frontera V2 del panel';
    END IF;
    IF to_regnamespace('vec_bolsa_panel') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el consumidor del panel sigue instalado';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_panel_bolsa_v2(
        jsonb, bytea, bytea, text, text, text, timestamptz
    ) FROM vec_bolsa_panel_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion FROM vec_bolsa_panel_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_panel_bolsa_v2(
    jsonb, bytea, bytea, text, text, text, timestamptz
);
COMMIT;
