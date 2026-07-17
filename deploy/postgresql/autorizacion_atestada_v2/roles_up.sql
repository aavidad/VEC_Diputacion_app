-- Bootstrap DBA de la puerta atestada VEC-AD-2. No crea cuentas LOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:roles_up:v1', 0
    )
);

DO $prevalidacion$
DECLARE
    encontrados text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap atestado V2 rechazado: requiere superusuario';
    END IF;
    IF pg_catalog.to_regprocedure('public.hmac(bytea,bytea,text)') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta pgcrypto.hmac(bytea,bytea,text)';
    END IF;
    IF pg_catalog.has_function_privilege(
           'public', 'public.hmac(bytea,bytea,text)', 'EXECUTE'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'pgcrypto.hmac conserva EXECUTE para PUBLIC',
            HINT = 'endurezca primero la extension en el bootstrap de la base dedicada';
    END IF;
    IF pg_catalog.to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_confianza_atestacion_v2.configuracion_raiz'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'faltan autorizacion V2 o catalogo de confianza VEC-AD-2';
    END IF;
    IF pg_catalog.to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL
       OR pg_catalog.to_regnamespace(
              'vec_autorizacion_atestada_v2_guardia'
          ) IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_event_trigger
            WHERE evtname =
                  'vec_autorizacion_atestada_v2_cerrar_acl_tipos'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ya existen objetos de autorizacion atestada V2';
    END IF;
    SELECT pg_catalog.array_agg(rolname::text ORDER BY rolname)
      INTO encontrados
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_autorizacion_atestada_v2_propietario',
         'vec_autorizacion_atestada_v2_migrador',
         'vec_autorizacion_atestada_v2_emisor_capacidad',
         'vec_autorizacion_atestada_v2_consumidor'
     ]);
    IF pg_catalog.cardinality(encontrados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'existen roles atestados V2; no se adoptan',
            DETAIL = pg_catalog.array_to_string(encontrados, ',');
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_autorizacion_atestada_v2_propietario NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v2_migrador NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v2_emisor_capacidad NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v2_consumidor NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT vec_autorizacion_atestada_v2_propietario
    TO vec_autorizacion_atestada_v2_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

CREATE SCHEMA vec_autorizacion_atestada_v2_guardia;
REVOKE ALL ON SCHEMA vec_autorizacion_atestada_v2_guardia FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v2_guardia.cerrar_acl_tipos()
RETURNS event_trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_autorizacion_atestada_v2'
           AND definicion.typelem = 0 AND definicion.typisdefined
    LOOP
        EXECUTE pg_catalog.format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I',
            tipo.nspname, tipo.typname,
            'vec_autorizacion_atestada_v2_emisor_capacidad',
            'vec_autorizacion_atestada_v2_consumidor'
        );
    END LOOP;
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v2_guardia.cerrar_acl_tipos()
    FROM PUBLIC;
CREATE EVENT TRIGGER vec_autorizacion_atestada_v2_cerrar_acl_tipos
    ON ddl_command_end
    WHEN TAG IN (
        'CREATE TABLE', 'CREATE TABLE AS', 'CREATE FOREIGN TABLE',
        'CREATE VIEW', 'CREATE MATERIALIZED VIEW', 'CREATE TYPE',
        'CREATE DOMAIN', 'ALTER TABLE', 'ALTER VIEW',
        'ALTER MATERIALIZED VIEW', 'ALTER TYPE', 'ALTER DOMAIN'
    )
    EXECUTE FUNCTION
        vec_autorizacion_atestada_v2_guardia.cerrar_acl_tipos();

DO $privilegios_base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT, CREATE ON DATABASE %I TO vec_autorizacion_atestada_v2_propietario',
        current_database()
    );
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v2_migrador, vec_autorizacion_atestada_v2_emisor_capacidad, vec_autorizacion_atestada_v2_consumidor',
        current_database()
    );
END
$privilegios_base$;

-- Dependencias estrechas: el propietario puede crear FK historicas y llamar
-- la revalidacion nominal. Ningun rol runtime recibe estas capacidades.
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_atestada_v2_propietario;
GRANT REFERENCES (decision_ref) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    TO vec_autorizacion_atestada_v2_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        bytea, bytea
    ) TO vec_autorizacion_atestada_v2_propietario;

GRANT USAGE ON SCHEMA vec_confianza_atestacion_v2
    TO vec_autorizacion_atestada_v2_propietario;
GRANT REFERENCES (configuracion_revision, clave_id, version) ON
    vec_confianza_atestacion_v2.configuracion_raiz
    TO vec_autorizacion_atestada_v2_propietario;

GRANT USAGE ON SCHEMA public
    TO vec_autorizacion_atestada_v2_propietario;
GRANT EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    TO vec_autorizacion_atestada_v2_propietario;

COMMIT;
