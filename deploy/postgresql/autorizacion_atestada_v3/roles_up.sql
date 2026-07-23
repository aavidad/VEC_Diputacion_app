-- Bootstrap DBA de la puerta consumidora VEC-AD-3. No crea LOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:roles:v1', 0
    )
);

DO $prevalidacion$
DECLARE
    v_roles text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'bootstrap VEC-AD-3 rechazado';
    END IF;
    IF pg_catalog.to_regprocedure(
           'public.hmac(bytea,bytea,text)'
       ) IS NULL
       OR pg_catalog.has_function_privilege(
           'public',
           'public.hmac(bytea,bytea,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'pgcrypto no está endurecido';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regnamespace(
           'vec_contratacion_temporal'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'faltan dependencias VEC-AD-3';
    END IF;
    SELECT pg_catalog.array_agg(rolname::text ORDER BY rolname)
      INTO v_roles
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_autorizacion_atestada_v3_propietario',
         'vec_autorizacion_atestada_v3_migrador',
         'vec_autorizacion_atestada_v3_emisor',
         'vec_autorizacion_atestada_v3_consumidor'
     ]);
    IF pg_catalog.cardinality(v_roles) > 0
       OR pg_catalog.to_regnamespace(
           'vec_autorizacion_atestada_v3'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'bootstrap VEC-AD-3 ya aplicado';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_autorizacion_atestada_v3_propietario
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v3_migrador
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v3_emisor
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v3_consumidor
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;

GRANT vec_autorizacion_atestada_v3_propietario
    TO vec_autorizacion_atestada_v3_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

CREATE SCHEMA vec_autorizacion_atestada_v3
    AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON SCHEMA vec_autorizacion_atestada_v3 FROM PUBLIC;

DO $base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v3_migrador, vec_autorizacion_atestada_v3_emisor, vec_autorizacion_atestada_v3_consumidor',
        pg_catalog.current_database()
    );
END
$base$;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    ),
    vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
        bytea, bytea, numeric, numeric
    ) TO vec_autorizacion_atestada_v3_propietario;
GRANT REFERENCES (decision_ref) ON
    vec_autorizacion.decision_concedida_contexto_actor_v3
    TO vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA public
    TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    TO vec_autorizacion_atestada_v3_propietario;

COMMIT;
