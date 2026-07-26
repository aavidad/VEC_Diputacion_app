BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:revalidacion_registro_accesos_bolsa_v2:000001', 0
));

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de frontera VEC T13 rechazado: funcion ausente';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'down de frontera VEC T13 rechazado: consulta instalada',
            HINT = 'retire antes la migracion principal T13';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
        jsonb,bytea,bytea,text,text,text,jsonb,text,text,text
    ) FROM vec_bolsa_accesos_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_bolsa_accesos_propietario;
DROP FUNCTION
    vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
        jsonb,bytea,bytea,text,text,text,jsonb,text,text,text
    );
COMMIT;
