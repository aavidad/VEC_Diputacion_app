-- Reversion DBA. Falla cerrada si quedan objetos, opciones o membresias ajenas.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_baremacion:roles_down:v1', 0)
);

DO $prevalidacion$
DECLARE
    esperado record;
    enlace record;
    oid_dba oid;
    oid_esquema_guardia oid;
    oid_funcion_guardia oid;
    etiquetas_esperadas constant text[] := ARRAY[
        'CREATE TABLE',
        'CREATE TABLE AS',
        'CREATE FOREIGN TABLE',
        'CREATE VIEW',
        'CREATE MATERIALIZED VIEW',
        'CREATE TYPE',
        'CREATE DOMAIN',
        'ALTER TABLE',
        'ALTER VIEW',
        'ALTER MATERIALIZED VIEW',
        'ALTER TYPE',
        'ALTER DOMAIN'
    ];
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_baremacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el esquema Bolsa sigue instalado';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la frontera de autorizacion Bolsa sigue instalada';
    END IF;

    SELECT oid INTO oid_dba
      FROM pg_catalog.pg_roles
     WHERE rolname = current_user;
    SELECT oid INTO oid_esquema_guardia
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_bolsa_baremacion_guardia';
    SELECT oid INTO oid_funcion_guardia
      FROM pg_catalog.pg_proc
     WHERE oid = pg_catalog.to_regprocedure(
         'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
     );
    IF oid_dba IS NULL OR oid_esquema_guardia IS NULL
       OR oid_funcion_guardia IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la guarda DDL Bolsa';
    END IF;

    -- La retirada se ejecuta por el mismo principal DBA que creo la guarda.
    -- Se rechazan adopciones o cambios de propietario, aunque los dos objetos
    -- sigan teniendo nombres y firmas validos.
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = oid_funcion_guardia
         WHERE espacio.oid = oid_esquema_guardia
           AND espacio.nspowner = oid_dba
           AND funcion.pronamespace = espacio.oid
           AND funcion.proowner = oid_dba
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambio el propietario de la guarda DDL Bolsa';
    END IF;

    -- ACL exacta: el DBA conserva USAGE/CREATE sobre el esquema y EXECUTE
    -- sobre la funcion; ningun otro principal, incluido PUBLIC, recibe nada.
    IF (SELECT count(*)
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.oid = oid_esquema_guardia) <> 2
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.oid = oid_esquema_guardia
           AND NOT (
               privilegio.grantor = oid_dba
               AND privilegio.grantee = oid_dba
               AND privilegio.privilege_type IN ('USAGE', 'CREATE')
               AND privilegio.is_grantable IS FALSE
           )
       ) OR (SELECT count(*)
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.oid = oid_funcion_guardia) <> 1
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.oid = oid_funcion_guardia
           AND NOT (
               privilegio.grantor = oid_dba
               AND privilegio.grantee = oid_dba
               AND privilegio.privilege_type = 'EXECUTE'
               AND privilegio.is_grantable IS FALSE
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron las ACL de la guarda DDL Bolsa';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = disparador.evtfoid
          JOIN pg_catalog.pg_language AS lenguaje
            ON lenguaje.oid = funcion.prolang
         WHERE disparador.evtname =
                   'vec_bolsa_baremacion_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
           AND disparador.evttags @> etiquetas_esperadas
           AND disparador.evttags <@ etiquetas_esperadas
           AND cardinality(disparador.evttags) =
               cardinality(etiquetas_esperadas)
           AND funcion.oid = oid_funcion_guardia
           AND funcion.prosecdef
           AND funcion.prorettype = 'pg_catalog.event_trigger'::regtype
           AND funcion.pronargs = 0
           AND funcion.pronargdefaults = 0
           AND funcion.prokind = 'f'
           AND funcion.provolatile = 'v'
           AND funcion.proparallel = 'u'
           AND funcion.proleakproof IS FALSE
           AND funcion.proisstrict IS FALSE
           AND lenguaje.lanname = 'plpgsql'
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog, pg_temp'
           ]::text[]
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL Bolsa no es la esperada';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtfoid = oid_funcion_guardia
           AND evtname <> 'vec_bolsa_baremacion_cerrar_acl_tipos'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc
         WHERE pronamespace = oid_esquema_guardia
           AND oid <> oid_funcion_guardia
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class
         WHERE relnamespace = oid_esquema_guardia
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL Bolsa contiene objetos inesperados';
    END IF;

    FOR esperado IN
        SELECT *
          FROM (VALUES
              ('vec_bolsa_baremacion_propietario'::text, false),
              ('vec_bolsa_baremacion_migrador'::text, false),
              ('vec_bolsa_baremacion_ejecutor'::text, true),
              ('vec_bolsa_baremacion_lector_outbox'::text, true),
              ('vec_bolsa_baremacion_registrador_atestacion'::text, true)
          ) AS opciones(rol, hereda)
    LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_roles
             WHERE rolname = esperado.rol
               AND rolcanlogin IS FALSE
               AND rolsuper IS FALSE
               AND rolcreatedb IS FALSE
               AND rolcreaterole IS FALSE
               AND rolinherit IS NOT DISTINCT FROM esperado.hereda
               AND rolreplication IS FALSE
               AND rolbypassrls IS FALSE
               AND rolconnlimit = -1
               AND rolconfig IS NULL
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: falta un rol Bolsa o sus opciones cambiaron',
                DETAIL = esperado.rol;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_bolsa_baremacion_propietario'
           AND miembro.rolname = 'vec_bolsa_baremacion_migrador'
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la membresia estructural Bolsa exacta';
    END IF;

    -- Se inspeccionan ambos extremos: cuentas externas dentro de un grupo
    -- Bolsa y roles Bolsa incorporados a cualquier grupo ajeno. Solo se admite
    -- el enlace estructural creado por roles_up y con sus tres opciones exactas.
    FOR enlace IN
        SELECT miembro.rolname AS miembro, grupo.rolname AS grupo
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS miembro
            ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
         WHERE (
             grupo.rolname = ANY (ARRAY[
                 'vec_bolsa_baremacion_propietario',
                 'vec_bolsa_baremacion_migrador',
                 'vec_bolsa_baremacion_ejecutor',
                 'vec_bolsa_baremacion_lector_outbox',
                 'vec_bolsa_baremacion_registrador_atestacion'
             ]) OR miembro.rolname = ANY (ARRAY[
                 'vec_bolsa_baremacion_propietario',
                 'vec_bolsa_baremacion_migrador',
                 'vec_bolsa_baremacion_ejecutor',
                 'vec_bolsa_baremacion_lector_outbox',
                 'vec_bolsa_baremacion_registrador_atestacion'
             ])
         )
           AND NOT (
               grupo.rolname = 'vec_bolsa_baremacion_propietario'
               AND miembro.rolname = 'vec_bolsa_baremacion_migrador'
               AND membresia.admin_option IS FALSE
               AND membresia.inherit_option IS FALSE
               AND membresia.set_option IS TRUE
           )
    LOOP
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe una membresia Bolsa inesperada',
            DETAIL = enlace.miembro || ' -> ' || enlace.grupo;
    END LOOP;
END
$prevalidacion$;

DROP EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos;
DROP FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos() RESTRICT;
DROP SCHEMA vec_bolsa_baremacion_guardia RESTRICT;

DO $revocar_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_bolsa_baremacion_propietario',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_bolsa_baremacion_migrador, vec_bolsa_baremacion_ejecutor, vec_bolsa_baremacion_lector_outbox, vec_bolsa_baremacion_registrador_atestacion',
        current_database()
    );
END
$revocar_base$;

REVOKE vec_bolsa_baremacion_propietario
    FROM vec_bolsa_baremacion_migrador;

-- La migracion base cierra los defaults globales que PostgreSQL abre para
-- funciones y tipos. Al no poder crear ya objetos, se restauran solo esas dos
-- entradas para eliminar las dependencias pg_default_acl del rol. Esto no
-- concede privilegios sobre ningun objeto existente.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_baremacion_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_baremacion_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP ROLE vec_bolsa_baremacion_registrador_atestacion;
DROP ROLE vec_bolsa_baremacion_lector_outbox;
DROP ROLE vec_bolsa_baremacion_ejecutor;
DROP ROLE vec_bolsa_baremacion_migrador;
DROP ROLE vec_bolsa_baremacion_propietario;
COMMIT;
