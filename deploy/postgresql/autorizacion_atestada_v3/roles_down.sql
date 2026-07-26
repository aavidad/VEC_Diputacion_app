BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:roles:v1', 0
    )
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'retirada VEC-AD-3 rechazada';
    END IF;
    IF pg_catalog.to_regnamespace(
           'vec_autorizacion_atestada_v3'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retire primero las migraciones VEC-AD-3';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    ),
    vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
        bytea, bytea, numeric, numeric
    ) FROM vec_autorizacion_atestada_v3_propietario;
REVOKE REFERENCES (decision_ref) ON
    vec_autorizacion.decision_concedida_contexto_actor_v3
    FROM vec_autorizacion_atestada_v3_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_autorizacion_atestada_v3_propietario;
REVOKE EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    FROM vec_autorizacion_atestada_v3_propietario;
REVOKE USAGE ON SCHEMA public
    FROM vec_autorizacion_atestada_v3_propietario;

DO $base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_atestada_v3_migrador, vec_autorizacion_atestada_v3_emisor, vec_autorizacion_atestada_v3_consumidor',
        pg_catalog.current_database()
    );
END
$base$;

REVOKE vec_autorizacion_atestada_v3_propietario
    FROM vec_autorizacion_atestada_v3_migrador;
DROP ROLE vec_autorizacion_atestada_v3_consumidor;
DROP ROLE vec_autorizacion_atestada_v3_emisor;
DROP ROLE vec_autorizacion_atestada_v3_migrador;
DROP ROLE vec_autorizacion_atestada_v3_propietario;

COMMIT;
