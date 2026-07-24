BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion
    ) OR EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.cuenta
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de revalidacion rica rechazado: existe historia de identidad';
    END IF;
END
$prevalidacion$;

DROP FUNCTION
    vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text, text);

COMMIT;
