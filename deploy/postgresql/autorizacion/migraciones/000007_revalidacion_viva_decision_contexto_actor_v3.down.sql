BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:revalidacion-viva-v3:000007', 0
    )
);

DO $dependencias$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '2BP01',
            MESSAGE = 'retirada viva V3 bloqueada por consumidor O2-05';
    END IF;
END
$dependencias$;

DROP FUNCTION
    vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    );
DROP FUNCTION
    vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
        bytea, bytea, numeric, numeric
    );

COMMIT;
