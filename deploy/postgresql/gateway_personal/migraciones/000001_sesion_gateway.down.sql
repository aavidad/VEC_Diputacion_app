BEGIN;
SET LOCAL ROLE vec_gateway_personal_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_gateway_personal:migracion:000001:v1', 0)
);

-- El cierre excluye aperturas/cierres en curso antes de decidir que no hay
-- historia. Si una transaccion runtime ya la estaba creando, el DOWN espera y
-- despues se rehusa; nunca elimina una historia que aun no habia observado.
LOCK TABLE vec_gateway_personal.sesion,
           vec_gateway_personal.evento_sesion IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_gateway_personal') IS NULL
       OR pg_catalog.to_regclass('vec_gateway_personal.sesion') IS NULL
       OR pg_catalog.to_regclass('vec_gateway_personal.evento_sesion') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de sesion gateway rechazado: instalacion incompleta';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_gateway_personal.sesion)
       OR EXISTS (SELECT 1 FROM vec_gateway_personal.evento_sesion) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de sesion gateway rechazado: existe historia';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION vec_gateway_personal.abrir_sesion_v1(
    text, text, text, timestamptz, timestamptz
) FROM vec_gateway_personal_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_gateway_personal.consultar_sesion_v1(text, timestamptz)
    FROM vec_gateway_personal_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_gateway_personal.cerrar_sesion_v1(text, timestamptz)
    FROM vec_gateway_personal_ejecutor;
REVOKE USAGE ON SCHEMA vec_gateway_personal
    FROM vec_gateway_personal_migrador, vec_gateway_personal_ejecutor;
DROP SCHEMA vec_gateway_personal CASCADE;
COMMIT;
