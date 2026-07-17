BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_panel') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta el esquema del panel';
    END IF;
    IF to_regprocedure(
           'vec_bolsa_panel.consultar_panel_interno_v1(jsonb,jsonb,bytea,bytea,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la consulta sigue instalada';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_bolsa_panel.consulta_confirmada)
       OR EXISTS (SELECT 1 FROM vec_bolsa_panel.auditoria)
       OR EXISTS (SELECT 1 FROM vec_bolsa_panel.proyeccion_panel)
       OR EXISTS (
           SELECT 1 FROM vec_bolsa_panel.atestacion_autorizacion_version
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia durable del panel';
    END IF;
END
$prevalidacion$;

-- Restablece los valores nativos para retirar las entradas de pg_default_acl;
-- el rol se elimina despues y no creara objetos nuevos entre ambas acciones.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_panel_propietario
    GRANT USAGE ON TYPES TO PUBLIC;
DROP SCHEMA vec_bolsa_panel CASCADE;
COMMIT;
