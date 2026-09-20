BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(
    hashtextextended('vec_identidad_sesiones_v1:migracion:asercion-peticion:v1', 0)
);
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

LOCK TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
    IN ACCESS EXCLUSIVE MODE;

DO $guardas$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
    ) THEN
        RAISE EXCEPTION 'down de asercion por peticion rechazado: existe historia'
            USING ERRCODE = '55000';
    END IF;
END
$guardas$;

DROP FUNCTION vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
    text, text, text, bigint, bytea, text, text, text, text, text, text,
    timestamptz, timestamptz
);
DROP TABLE vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1;
COMMIT;
