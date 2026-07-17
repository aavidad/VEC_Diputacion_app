BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta el publicador del panel';
    END IF;
    IF to_regprocedure(
           'vec_bolsa_panel.consultar_panel_interno_v1(jsonb,jsonb,bytea,bytea,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la consulta sigue instalada';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb)
    FROM vec_bolsa_panel_proyector;
REVOKE USAGE ON SCHEMA vec_bolsa_panel FROM vec_bolsa_panel_proyector;
DROP FUNCTION vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb);
COMMIT;
