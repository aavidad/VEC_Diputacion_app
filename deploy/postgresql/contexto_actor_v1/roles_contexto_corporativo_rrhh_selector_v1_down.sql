-- Retirada gobernada del rol selector corporativo RRHH. Solo revoca el
-- CONNECT creado por el alta y nunca elimina dependencias en cascada.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1',
        0
    )
);

LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    oid_selector oid;
    oid_base oid;
    oid_dueno_base oid;
    diferencias integer;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid
         WHERE rolname = current_user
           AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'retirada del selector corporativo RRHH rechazada';
    END IF;

    SELECT rol.oid
      INTO oid_selector
      FROM pg_catalog.pg_authid AS rol
     WHERE rol.rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
       AND rol.rolcanlogin IS FALSE
       AND rol.rolsuper IS FALSE
       AND rol.rolcreatedb IS FALSE
       AND rol.rolcreaterole IS FALSE
       AND rol.rolinherit IS FALSE
       AND rol.rolreplication IS FALSE
       AND rol.rolbypassrls IS FALSE
       AND rol.rolconnlimit = -1
       AND rol.rolpassword IS NULL
       AND rol.rolvaliduntil IS NULL
       AND pg_catalog.shobj_description(rol.oid, 'pg_authid') =
           'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_db_role_setting AS ajuste
            WHERE ajuste.setrole = rol.oid
       );
    IF oid_selector IS NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS relacion
         WHERE relacion.roleid = oid_selector
            OR relacion.member = oid_selector
            OR relacion.grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
         FROM pg_catalog.pg_shseclabel AS etiqueta
         WHERE etiqueta.classoid = 'pg_catalog.pg_authid'::regclass
           AND etiqueta.objoid = oid_selector
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE (
             espacio.nspname = 'vec_identidad_sesiones_v1'
             AND funcion.proname =
                 'revalidar_contexto_corporativo_rrhh_v1'
         ) OR (
             espacio.nspname = 'vec_contexto_actor_v1'
             AND funcion.proname = ANY (ARRAY[
                 'resolver_y_registrar_contexto_corporativo_rrhh_v1',
                 'reconciliar_contexto_corporativo_rrhh_v1',
                 'acreditar_uso_registro_contexto_corporativo_rrhh_v1'
             ]::name[])
         )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH rechazada';
    END IF;

    SELECT base.oid, base.datdba
      INTO oid_base, oid_dueno_base
      FROM pg_catalog.pg_database AS base
     WHERE base.datname = current_database();
    WITH actual AS (
        SELECT base.oid AS objeto, acl.grantor, acl.grantee,
               acl.privilege_type, acl.is_grantable
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ), esperado(
        objeto, grantor, grantee, privilege_type, is_grantable
    ) AS (
        VALUES (
            oid_base, oid_dueno_base, oid_selector, 'CONNECT'::text, false
        )
    ), diferencia AS (
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
    )
    SELECT count(*) INTO diferencias FROM diferencia;

    IF diferencias <> 0 OR NOT COALESCE((
        SELECT count(*) = 1
           AND bool_and(
               dependencia.dbid = 0
               AND dependencia.classid =
                   'pg_catalog.pg_database'::regclass
               AND dependencia.objid = oid_base
               AND dependencia.objsubid = 0
               AND dependencia.deptype = 'a'
           )
          FROM pg_catalog.pg_shdepend AS dependencia
         WHERE dependencia.refclassid =
                   'pg_catalog.pg_authid'::regclass
           AND dependencia.refobjid = oid_selector
    ), false) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH rechazada';
    END IF;

    -- Sin membresias y con un pg_shdepend exacto, cualquier autoridad efectiva
    -- adicional solo puede llegar de PUBLIC. Se cierra sobre toda superficie
    -- definida por usuarios antes de revocar el unico privilegio nominal.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS acl
         WHERE base.datistemplate IS FALSE
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.relacl) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute AS columna
          JOIN pg_catalog.pg_class AS objeto
            ON objeto.oid = columna.attrelid
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(columna.attacl) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tipo.typnamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  tipo.typacl,
                  pg_catalog.acldefault('T', tipo.typowner)
              )
          ) AS acl
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND (
               tipo.typtype IN ('c', 'd', 'e', 'm', 'r')
               OR (
                   tipo.typtype = 'b'
                   AND NOT EXISTS (
                       SELECT 1
                         FROM pg_catalog.pg_type AS elemento
                        WHERE elemento.oid = tipo.typelem
                          AND elemento.typarray = tipo.oid
                   )
               )
           )
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_largeobject_metadata AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.lomacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_data_wrapper AS envoltorio
          CROSS JOIN LATERAL pg_catalog.aclexplode(envoltorio.fdwacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_server AS servidor
          CROSS JOIN LATERAL pg_catalog.aclexplode(servidor.srvacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_language AS lenguaje
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  lenguaje.lanacl,
                  pg_catalog.acldefault('l', lenguaje.lanowner)
              )
          ) AS acl
         WHERE lenguaje.oid >= 16384
           AND acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_tablespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.spcacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl) AS acl
         WHERE acl.grantee = 0
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_policy AS politica
         WHERE 0::oid = ANY (politica.polroles)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_parameter_acl AS parametro
          CROSS JOIN LATERAL pg_catalog.aclexplode(parametro.paracl) AS acl
         WHERE acl.grantee = 0
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH rechazada';
    END IF;

    PERFORM pg_catalog.set_config(
        'vec.selector_corporativo_rrhh_oid',
        oid_selector::text,
        true
    );
END
$prevalidacion$;

DO $privilegio_base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_contexto_actor_corporativo_rrhh_selector',
        current_database()
    );
END
$privilegio_base$;

DO $postrevocacion$
DECLARE
    oid_selector oid := pg_catalog.current_setting(
        'vec.selector_corporativo_rrhh_oid'
    )::oid;
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_shdepend AS dependencia
         WHERE dependencia.refclassid =
                   'pg_catalog.pg_authid'::regclass
           AND dependencia.refobjid = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
         WHERE acl.grantee = oid_selector
            OR acl.grantor = oid_selector
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH incompleta';
    END IF;
END
$postrevocacion$;

DROP ROLE vec_contexto_actor_corporativo_rrhh_selector;

DO $postvalidacion$
DECLARE
    oid_selector oid := pg_catalog.current_setting(
        'vec.selector_corporativo_rrhh_oid'
    )::oid;
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_authid
         WHERE rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_db_role_setting
         WHERE setrole = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members
         WHERE roleid = oid_selector
            OR member = oid_selector
            OR grantor = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_shdepend
         WHERE refclassid = 'pg_catalog.pg_authid'::regclass
           AND refobjid = oid_selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_shseclabel
         WHERE classoid = 'pg_catalog.pg_authid'::regclass
           AND objoid = oid_selector
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del selector corporativo RRHH incompleta';
    END IF;
END
$postvalidacion$;

COMMIT;
