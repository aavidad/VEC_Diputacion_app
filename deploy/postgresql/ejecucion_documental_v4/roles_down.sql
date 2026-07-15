-- Destructivo: ejecutar solo despues de las migraciones descendentes y de
-- retirar todas las cuentas LOGIN miembro de estos grupos.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);

-- Se valida antes de mutar ACL o membresias. Se admite exclusivamente el
-- enlace estructural creado por roles_up.sql; cualquier enlace entrante o
-- saliente adicional, incluso un LOGIN miembro del propietario, aborta todo.
DO $prevalidacion$
DECLARE
    rol text;
    enlace record;
BEGIN
    IF pg_catalog.to_regnamespace('vec_ejecucion_documental_v4') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el esquema V4 sigue instalado';
    END IF;

    IF pg_catalog.to_regnamespace(
           'vec_ejecucion_documental_v4_guardia'
       ) IS NULL OR pg_catalog.to_regprocedure(
           'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la guarda DDL V4';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = disparador.evtfoid
         WHERE disparador.evtname =
                   'vec_ejecucion_documental_v4_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
           AND funcion.oid = pg_catalog.to_regprocedure(
               'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
           )
           AND funcion.prosecdef
           AND funcion.prorettype = 'pg_catalog.event_trigger'::regtype
           AND funcion.pronargs = 0
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL V4 no es la esperada';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'vec_ejecucion_documental_v4_ejecutor_atestado',
        'vec_ejecucion_documental_v4_emisor_capacidad',
        'vec_ejecucion_documental_v4_migrador',
        'vec_ejecucion_documental_v4_propietario'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = rol AND NOT rolcanlogin
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: falta un rol V4 esperado',
                DETAIL = rol;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname =
                   'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la membresia estructural V4 exacta';
    END IF;

    FOR enlace IN
        SELECT miembro.rolname AS miembro,
               grupo.rolname AS grupo
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE (
             grupo.rolname = ANY (ARRAY[
                 'vec_ejecucion_documental_v4_ejecutor_atestado',
                 'vec_ejecucion_documental_v4_emisor_capacidad',
                 'vec_ejecucion_documental_v4_migrador',
                 'vec_ejecucion_documental_v4_propietario'
             ]) OR miembro.rolname = ANY (ARRAY[
                 'vec_ejecucion_documental_v4_ejecutor_atestado',
                 'vec_ejecucion_documental_v4_emisor_capacidad',
                 'vec_ejecucion_documental_v4_migrador',
                 'vec_ejecucion_documental_v4_propietario'
             ])
         )
           AND NOT (
               grupo.rolname =
                   'vec_ejecucion_documental_v4_propietario'
               AND miembro.rolname =
                   'vec_ejecucion_documental_v4_migrador'
               AND membresia.admin_option IS FALSE
               AND membresia.inherit_option IS FALSE
               AND membresia.set_option IS TRUE
           )
    LOOP
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe una membresia V4 inesperada',
            DETAIL = enlace.miembro || ' -> ' || enlace.grupo;
    END LOOP;
END
$prevalidacion$;

DROP EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos;
DROP FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();
DROP SCHEMA vec_ejecucion_documental_v4_guardia RESTRICT;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_emisor_capacidad',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_ejecutor_atestado',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_migrador',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_propietario',
        current_database()
    );
END
$privilegios_base$;

REVOKE EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    FROM vec_ejecucion_documental_v4_propietario;
REVOKE USAGE ON SCHEMA public
    FROM vec_ejecucion_documental_v4_propietario;

REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;

-- La migracion ascendente cierra globalmente los defaults que PostgreSQL abre
-- para funciones y tipos. Al desaparecer el rol ya no puede crear objetos;
-- se revierten solo esas dos entradas para poder eliminar su pg_default_acl.
-- Esto no concede nada sobre objetos existentes ni reabre pgcrypto.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_ejecucion_documental_v4_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_ejecucion_documental_v4_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP ROLE vec_ejecucion_documental_v4_ejecutor_atestado;
DROP ROLE vec_ejecucion_documental_v4_emisor_capacidad;
DROP ROLE vec_ejecucion_documental_v4_migrador;
DROP ROLE vec_ejecucion_documental_v4_propietario;
COMMIT;
